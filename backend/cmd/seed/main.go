package main

import (
	"log"

	"office-file-sharing/backend/internal/shared/config"
	"office-file-sharing/backend/internal/shared/db"
	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()
	gormDB := db.Init(cfg.DatabaseURL)

	log.Println("Resetting and seeding database accounts...")

	// Clear role and school data only — preserve users and documents
	tables := []string{"roles", "organizations", "peer_connections"}
	for _, t := range tables {
		tx := gormDB.Exec("TRUNCATE TABLE " + t + " CASCADE;")
		if tx.Error != nil {
			log.Printf("Warning: failed to truncate table %s (might not exist yet): %v", t, tx.Error)
		}
	}

	// 1. Seed Schools/Tenants (idempotent)
	schoolDefs := []struct {
		Name string
		Slug string
	}{
		{"Directorate of Higher Education (DHE)", "dhe-hq"},
		{"Greenwood High School", "greenwood-high"},
		{"Delhi Public School", "dps"},
		{"Modern School", "modern-school"},
	}
	var dheSchool, school1, school2, school3 models.School
	schoolRefs := []*models.School{&dheSchool, &school1, &school2, &school3}
	for i, def := range schoolDefs {
		if err := gormDB.Where("slug = ?", def.Slug).First(schoolRefs[i]).Error; err != nil {
			schoolRefs[i].ID = uuid.New()
			schoolRefs[i].Name = def.Name
			schoolRefs[i].Slug = def.Slug
			if err := gormDB.Create(schoolRefs[i]).Error; err != nil {
				log.Fatalf("Failed to create tenant school %s: %v", def.Name, err)
			}
		}
	}
	log.Println("Seeded schools: DHE, Greenwood High, DPS, Modern School")

	// 2. Hash default password
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash default password:", err)
	}

	// 2a. Seed hierarchical roles
	rolesToSeed := []struct {
		Name          string
		IsAdminAccess bool
		ParentName    string
	}{
		{"SuperAdmin", true, ""},
		{"Admin", true, "SuperAdmin"},
		{"DHE", true, "Admin"},
		{"School Admin", true, "DHE"},
		{"Teaching staff", false, "School Admin"},
		{"non-teaching", false, "School Admin"},
		{"vocational", false, "School Admin"},
	}

	for _, r := range rolesToSeed {
		var parentID *uuid.UUID
		var parentPath string
		if r.ParentName != "" {
			var p models.Role
			if err := gormDB.Where("role_name = ?", r.ParentName).First(&p).Error; err == nil {
				parentID = &p.ID
				parentPath = p.Path
			}
		}
		newID := uuid.New()
		var path string
		if parentID == nil {
			path = "/" + newID.String() + "/"
		} else {
			path = parentPath + newID.String() + "/"
		}
		newRole := models.Role{
			ID:            newID,
			RoleName:      r.Name,
			IsAdminAccess: r.IsAdminAccess,
			ParentRoleID:  parentID,
			TenantID:      nil,
			CreatedBy:     "System",
			Path:          path,
		}
		if err := gormDB.Create(&newRole).Error; err != nil {
			log.Fatalf("Failed to seed role %s: %v", r.Name, err)
		}
	}
	log.Println("Seeded hierarchical roles.")

	// 3. Seed default users (idempotent — skip if email already exists)
	defaultUsers := []models.User{
		{Name: "Super Admin", Email: "superadmin@school.edu", PasswordHash: string(hash), Role: "SuperAdmin", SchoolID: nil},
		{Name: "System Administrator", Email: "admin@school.edu", PasswordHash: string(hash), Role: "DHE", SchoolID: &dheSchool.ID},

		// Greenwood High School
		{Name: "Rahul Gupta", Email: "rahul@school.edu", PasswordHash: string(hash), Role: "School Admin", SchoolID: &school1.ID},
		{Name: "Priya Patel", Email: "priya@school.edu", PasswordHash: string(hash), Role: "Teaching staff", SchoolID: &school1.ID, ClassSection: "Department A", Subject: "Science"},
		{Name: "Deepak Singh", Email: "deepak@school.edu", PasswordHash: string(hash), Role: "non-teaching", SchoolID: &school1.ID},

		// Delhi Public School
		{Name: "Gaurav Verma", Email: "gaurav@school.edu", PasswordHash: string(hash), Role: "School Admin", SchoolID: &school2.ID},
		{Name: "Neha Reddy", Email: "neha@school.edu", PasswordHash: string(hash), Role: "Teaching staff", SchoolID: &school2.ID, ClassSection: "Department B", Subject: "History"},

		// Modern School
		{Name: "Shalini Sen", Email: "shalini@school.edu", PasswordHash: string(hash), Role: "School Admin", SchoolID: &school3.ID},
		{Name: "Vikram Iyer", Email: "vikram@school.edu", PasswordHash: string(hash), Role: "Teaching staff", SchoolID: &school3.ID, ClassSection: "Department C", Subject: "Mathematics"},
		{Name: "Meera Menon", Email: "meera@school.edu", PasswordHash: string(hash), Role: "Teaching staff", SchoolID: &school3.ID, ClassSection: "Department D", Subject: "English"},
		{Name: "Aarav Sharma", Email: "aarav@school.edu", PasswordHash: string(hash), Role: "vocational", SchoolID: &school3.ID, ClassSection: "Department A"},
		{Name: "Ananya Iyer", Email: "ananya@school.edu", PasswordHash: string(hash), Role: "vocational", SchoolID: &school3.ID, ClassSection: "Department B"},
		{Name: "Rohan Das", Email: "rohan@school.edu", PasswordHash: string(hash), Role: "vocational", SchoolID: &school3.ID, ClassSection: "Department C"},
		{Name: "Kavya Menon", Email: "kavya@school.edu", PasswordHash: string(hash), Role: "vocational", SchoolID: &school3.ID, ClassSection: "Department D"},
	}

	for i := range defaultUsers {
		var existing models.User
		if err := gormDB.Where("email = ?", defaultUsers[i].Email).First(&existing).Error; err != nil {
			defaultUsers[i].ID = uuid.New()
			gormDB.Create(&defaultUsers[i])
		} else {
			existing.SchoolID = defaultUsers[i].SchoolID
			gormDB.Save(&existing)
			defaultUsers[i] = existing
		}
	}
	log.Println("Seeded users across multiple schools (existing users preserved).")

	// 3a. Seed organizations matching the schools
	orgDefs := []struct {
		SchoolID         uuid.UUID
		OrganizationName string
		Type             string
		AdminEmail       string
		IsDHE            bool
	}{
		{dheSchool.ID, "Directorate of Higher Education (DHE)", "dhe", "admin@school.edu", true},
		{school1.ID, "Greenwood High School", "school", "rahul@school.edu", false},
		{school2.ID, "Delhi Public School", "school", "gaurav@school.edu", false},
		{school3.ID, "Modern School", "school", "shalini@school.edu", false},
	}

	var dheOrgID *uuid.UUID
	for _, def := range orgDefs {
		var existingOrg models.Organization
		if err := gormDB.Where("tenant_id = ?", def.SchoolID).First(&existingOrg).Error; err != nil {
			var adminUser models.User
			var pocID *uuid.UUID
			if err := gormDB.Where("email = ?", def.AdminEmail).First(&adminUser).Error; err == nil {
				pocID = &adminUser.ID
			}

			newOrg := models.Organization{
				ID:               uuid.New(),
				OrganizationName: def.OrganizationName,
				Type:             def.Type,
				PointOfContactID: pocID,
				CreatedBy:        "SuperAdmin",
				TenantID:         &def.SchoolID,
			}
			if !def.IsDHE && dheOrgID != nil {
				newOrg.ParentOrgID = dheOrgID
			}
			gormDB.Create(&newOrg)
			if def.IsDHE {
				dheOrgID = &newOrg.ID
			}
		} else {
			if def.IsDHE {
				dheOrgID = &existingOrg.ID
			}
		}
	}
	log.Println("Seeded matching organizations.")

	// 4. Ensure document types are seeded for all schools
	var schools []models.School
	gormDB.Find(&schools)

	for _, s := range schools {
		docTypes := []models.DocumentType{
			{
				SchoolID:       s.ID,
				Name:           "Staff Grievance",
				Slug:           "staff-grievance",
				WorkflowStages: `[{"stage": 1, "role": "Teaching staff", "label": "Department Head", "optional": false}]`,
				RequiredFields: `[]`,
			},
			{
				SchoolID:       s.ID,
				Name:           "Infrastructure Issue",
				Slug:           "infrastructure-issue",
				WorkflowStages: `[{"stage": 1, "role": "School Admin", "label": "School Admin Final approval", "optional": false}]`,
				RequiredFields: `["reason", "urgency"]`,
			},
			{
				SchoolID:       s.ID,
				Name:           "Disciplinary Issue",
				Slug:           "disciplinary-issue",
				WorkflowStages: `[{"stage": 1, "role": "Teaching staff", "label": "Department Head", "optional": false}]`,
				RequiredFields: `["event_name", "event_date"]`,
			},
			{
				SchoolID:       s.ID,
				Name:           "Audit Report",
				Slug:           "audit-report",
				WorkflowStages: `[{"stage": 1, "role": "School Admin", "label": "School Admin Approval", "optional": false}]`,
				RequiredFields: `["audit_reason", "percentage"]`,
			},
		}
		for i := range docTypes {
			var existing models.DocumentType
			if err := gormDB.Where("school_id = ? AND slug = ?", s.ID, docTypes[i].Slug).First(&existing).Error; err != nil {
				docTypes[i].ID = uuid.New()
				gormDB.Create(&docTypes[i])
				log.Printf("Seeded missing document type for school %s: %s", s.Name, docTypes[i].Name)
			}
		}
	}

	log.Println("Database seeding completed successfully.")
}
