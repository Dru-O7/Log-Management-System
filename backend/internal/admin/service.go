package admin

import (
	"errors"
	"office-file-sharing/backend/internal/shared/models"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	GetStats(schoolID *string) (*SystemStats, error)
	GetAllUsers(schoolID *string) ([]UserResponse, error)
	CreateUser(req CreateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error)
	UpdateUser(id uuid.UUID, req UpdateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error)
	DeleteUser(id uuid.UUID) error
	GetAllDocumentTypes(schoolID *string) ([]DocumentTypeResponse, error)
	CreateDocumentType(req CreateDocTypeRequest) (*DocumentTypeResponse, error)
	UpdateDocumentType(id uuid.UUID, req UpdateDocTypeRequest) (*DocumentTypeResponse, error)
	DeleteDocumentType(id uuid.UUID) error
	GetAllSchools(schoolID *string) ([]SchoolResponse, error)
	UpdateSchool(id uuid.UUID, req UpdateSchoolRequest) (*SchoolResponse, error)
	GetAllRoles(actorRole string, actorSchoolID *uuid.UUID) ([]RoleResponse, error)
	CreateRole(req CreateRoleRequest, actorRole string, actorSchoolID *uuid.UUID) (*RoleResponse, error)
	UpdateRole(id uuid.UUID, req UpdateRoleRequest, actorRole string, actorSchoolID *uuid.UUID) (*RoleResponse, error)
	DeleteRole(id uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetStats(schoolID *string) (*SystemStats, error) {
	return s.repo.GetStats(schoolID)
}

func (s *service) GetAllUsers(schoolID *string) ([]UserResponse, error) {
	return s.repo.GetAllUsers(schoolID)
}

func (s *service) CreateUser(req CreateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		return nil, errors.New("name and email are required")
	}
	if req.Password == "" {
		req.Password = "password" // default password
	}

	var targetSchoolID *uuid.UUID
	if actorRole == "School Admin" {
		if req.Role == "DHE" || req.Role == "SuperAdmin" || req.Role == "Admin" {
			return nil, errors.New("school admin cannot assign administrative roles")
		}
		if actorSchoolID == nil {
			return nil, errors.New("school admin must belong to a school")
		}
		targetSchoolID = actorSchoolID
	} else {
		targetSchoolID = req.SchoolID
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &models.User{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		SchoolID:     targetSchoolID,
		ClassSection: req.ClassSection,
		Subject:      req.Subject,
		Phone:        req.Phone,
	}

	if err := s.repo.CreateUser(u); err != nil {
		return nil, err
	}

	resp := &UserResponse{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Role:         u.Role,
		SchoolID:     u.SchoolID,
		ClassSection: u.ClassSection,
		Subject:      u.Subject,
		Phone:        u.Phone,
		CreatedAt:    u.CreatedAt,
	}
	return resp, nil
}

func (s *service) UpdateUser(id uuid.UUID, req UpdateUserRequest, actorRole string, actorSchoolID *uuid.UUID) (*UserResponse, error) {
	u, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// If changing role or email, block if user is referenced in document workflows or logs
	roleChanged := req.Role != "" && req.Role != u.Role
	emailChanged := req.Email != "" && req.Email != u.Email
	if roleChanged || emailChanged {
		repoImpl, ok := s.repo.(*repository)
		if ok {
			var count int64
			repoImpl.db.Model(&models.Document{}).
				Where("uploader_id = ? OR current_owner_id = ? OR referral_owner_id = ?", id, id, id).
				Count(&count)
			if count > 0 {
				return nil, errors.New("cannot change role or email of a user with active or historical documents")
			}

			repoImpl.db.Model(&models.WorkflowHistory{}).
				Where("actor_id = ? OR target_id = ?", id, id).
				Count(&count)
			if count > 0 {
				return nil, errors.New("cannot change role or email of a user with workflow history logs")
			}
		}
	}

	if req.Name != "" {
		u.Name = req.Name
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Role != "" {
		if actorRole == "School Admin" {
			if req.Role == "DHE" || req.Role == "SuperAdmin" || req.Role == "Admin" {
				return nil, errors.New("school admin cannot assign administrative roles")
			}
		}
		u.Role = req.Role
	}

	// Enforce school restrictions for School Admin / DHE
	if actorRole == "School Admin" {
		if actorSchoolID == nil {
			return nil, errors.New("school admin must belong to a school")
		}
		if u.SchoolID == nil || *u.SchoolID != *actorSchoolID {
			return nil, errors.New("you are not authorized to update users outside your school")
		}
		u.SchoolID = actorSchoolID
	} else {
		// DHE/SuperAdmin can change the school of any user (including changing it to nil/None)
		u.SchoolID = req.SchoolID
	}
	u.ClassSection = req.ClassSection
	u.Subject = req.Subject
	u.Phone = req.Phone

	// Update password only if provided
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u.PasswordHash = string(hash)
	}

	if err := s.repo.UpdateUser(u); err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Role:         u.Role,
		SchoolID:     u.SchoolID,
		ClassSection: u.ClassSection,
		Subject:      u.Subject,
		Phone:        u.Phone,
		CreatedAt:    u.CreatedAt,
	}, nil
}

func (s *service) DeleteUser(id uuid.UUID) error {
	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return errors.New("invalid repository type")
	}

	var count int64

	// 1. Check Document table
	repoImpl.db.Model(&models.Document{}).
		Where("uploader_id = ? OR current_owner_id = ? OR referral_owner_id = ?", id, id, id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete user: they have active or historical documents")
	}

	// 2. Check WorkflowHistory table
	repoImpl.db.Model(&models.WorkflowHistory{}).
		Where("actor_id = ? OR target_id = ?", id, id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete user: they are referenced in workflow history logs")
	}

	// 3. Check DocumentPendingApprover table
	repoImpl.db.Model(&models.DocumentPendingApprover{}).
		Where("user_id = ?", id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete user: they are a pending workflow approver")
	}

	// 4. Check Attachment table
	repoImpl.db.Model(&models.Attachment{}).
		Where("uploaded_by = ?", id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete user: they uploaded files enclosed in documents")
	}

	return s.repo.DeleteUser(id)
}

func (s *service) GetAllDocumentTypes(schoolID *string) ([]DocumentTypeResponse, error) {
	return s.repo.GetAllDocumentTypes(schoolID)
}

func (s *service) CreateDocumentType(req CreateDocTypeRequest) (*DocumentTypeResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("document type name is required")
	}
	if req.WorkflowStages == "" {
		req.WorkflowStages = "[]"
	}
	if req.RequiredFields == "" {
		req.RequiredFields = "[]"
	}
	dt := &models.DocumentType{
		ID:             uuid.New(),
		SchoolID:       req.SchoolID,
		Name:           req.Name,
		Slug:           req.Slug,
		WorkflowStages: req.WorkflowStages,
		RequiredFields: req.RequiredFields,
		Active:         true,
	}

	if err := s.repo.CreateDocumentType(dt); err != nil {
		return nil, err
	}

	return &DocumentTypeResponse{
		ID:             dt.ID,
		SchoolID:       dt.SchoolID,
		Name:           dt.Name,
		Slug:           dt.Slug,
		WorkflowStages: dt.WorkflowStages,
		RequiredFields: dt.RequiredFields,
		SlaHours:       0,
		Active:         dt.Active,
	}, nil
}

func (s *service) UpdateDocumentType(id uuid.UUID, req UpdateDocTypeRequest) (*DocumentTypeResponse, error) {
	dt, err := s.repo.GetDocumentTypeByID(id)
	if err != nil {
		return nil, errors.New("document type not found")
	}

	// If changing stages/fields, block if any active documents use this type
	if req.WorkflowStages != "" || req.RequiredFields != "" {
		repoImpl, ok := s.repo.(*repository)
		if ok {
			var activeCount int64
			repoImpl.db.Model(&models.Document{}).
				Where("document_type_id = ? AND status NOT IN ?", id, []string{string(models.StatusClosed), string(models.StatusArchived), string(models.StatusRejected)}).
				Count(&activeCount)
			if activeCount > 0 {
				return nil, errors.New("cannot edit workflow stages or required fields: active documents exist of this type")
			}
		}
	}

	if req.Name != "" {
		dt.Name = req.Name
	}
	if req.Slug != "" {
		dt.Slug = req.Slug
	}
	if req.WorkflowStages != "" {
		dt.WorkflowStages = req.WorkflowStages
	}
	if req.RequiredFields != "" {
		dt.RequiredFields = req.RequiredFields
	}
	dt.Active = req.Active

	if err := s.repo.UpdateDocumentType(dt); err != nil {
		return nil, err
	}

	return &DocumentTypeResponse{
		ID:                dt.ID,
		SchoolID:          dt.SchoolID,
		Name:              dt.Name,
		Slug:              dt.Slug,
		WorkflowStages:    dt.WorkflowStages,
		RequiredFields:    dt.RequiredFields,
		SlaHours:          0,
		Active:            dt.Active,
	}, nil
}

func (s *service) DeleteDocumentType(id uuid.UUID) error {
	repoImpl, ok := s.repo.(*repository)
	if !ok {
		return errors.New("invalid repository type")
	}

	var count int64
	repoImpl.db.Model(&models.Document{}).
		Where("document_type_id = ?", id).
		Count(&count)
	if count > 0 {
		return errors.New("cannot delete document type: it is referenced by existing documents")
	}

	return s.repo.DeleteDocumentType(id)
}

func (s *service) GetAllSchools(schoolID *string) ([]SchoolResponse, error) {
	return s.repo.GetAllSchools(schoolID)
}

func (s *service) UpdateSchool(id uuid.UUID, req UpdateSchoolRequest) (*SchoolResponse, error) {
	school, err := s.repo.GetSchoolByID(id)
	if err != nil {
		return nil, errors.New("school not found")
	}

	if req.Name != "" {
		school.Name = req.Name
	}
	if req.Slug != "" {
		school.Slug = req.Slug
	}
	if req.Settings != "" {
		school.Settings = req.Settings
	}

	if err := s.repo.UpdateSchool(school); err != nil {
		return nil, err
	}

	return &SchoolResponse{
		ID:        school.ID,
		Name:      school.Name,
		Slug:      school.Slug,
		Settings:  school.Settings,
		CreatedAt: school.CreatedAt,
	}, nil
}

func (s *service) GetAllRoles(actorRole string, actorSchoolID *uuid.UUID) ([]RoleResponse, error) {
	var schoolIDStr *string
	// Global administrative roles can see all roles (no tenant scoping)
	globalAdminRoles := map[string]bool{
		"SuperAdmin": true,
		"Admin":      true,
		"DHE":        true,
	}
	if !globalAdminRoles[actorRole] {
		if actorSchoolID != nil {
			val := actorSchoolID.String()
			schoolIDStr = &val
		} else {
			// Non-global roles must have a school context to list roles
			return nil, errors.New("school context required to list roles")
		}
	}
	return s.repo.GetAllRoles(schoolIDStr)
}

func (s *service) CreateRole(req CreateRoleRequest, actorRole string, actorSchoolID *uuid.UUID) (*RoleResponse, error) {
	if strings.TrimSpace(req.RoleName) == "" {
		return nil, errors.New("role name is required")
	}

	// Enforce global SuperAdmin uniqueness
	if strings.EqualFold(req.RoleName, "SuperAdmin") {
		return nil, errors.New("creation of SuperAdmin role is prohibited")
	}

	// Find actor's role record to verify boundaries
	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, errors.New("unauthorized: actor role not found in system hierarchy")
	}

	// Determine tenant scope
	var targetTenantID *uuid.UUID
	if actorRole == "SuperAdmin" {
		targetTenantID = req.TenantID
	} else {
		targetTenantID = actorSchoolID
	}

	// Scoped uniqueness check
	var schoolIDStr *string
	if targetTenantID != nil {
		val := targetTenantID.String()
		schoolIDStr = &val
	}
	existingRoles, err := s.repo.GetAllRoles(schoolIDStr)
	if err == nil {
		for _, er := range existingRoles {
			if strings.EqualFold(er.RoleName, req.RoleName) {
				return nil, errors.New("role name must be unique within this tenant scope")
			}
		}
	}

	newID := uuid.New()
	var parentRoleID *uuid.UUID
	var path string

	if req.ParentRoleID != nil {
		parentRole, err := s.repo.GetRoleByID(*req.ParentRoleID)
		if err != nil {
			return nil, errors.New("parent role not found")
		}

		// Non-SuperAdmins can only parent to roles within their own subtree
		if actorRole != "SuperAdmin" {
			isDescendant := strings.HasPrefix(parentRole.Path, actorRoleRec.Path)
			isSelf := parentRole.ID == actorRoleRec.ID
			if !isDescendant && !isSelf {
				return nil, errors.New("cannot parent role outside your own subtree boundary")
			}
		}

		// Enforce tenant scoping
		if parentRole.TenantID != nil && (targetTenantID == nil || *parentRole.TenantID != *targetTenantID) {
			return nil, errors.New("cannot parent role to another tenant's role")
		}

		parentRoleID = req.ParentRoleID
		path = parentRole.Path + newID.String() + "/"
	} else {
		// Only SuperAdmins can create top-level/root roles
		if actorRole != "SuperAdmin" {
			return nil, errors.New("parent role is required for non-SuperAdmins")
		}
		path = "/" + newID.String() + "/"
	}

	// Max tree depth check (10 levels)
	depth := strings.Count(path, "/") - 1
	if depth > 10 {
		return nil, errors.New("maximum tree hierarchy depth of 10 exceeded")
	}

	// Elevation protection
	if req.IsAdminAccess && !actorRoleRec.IsAdminAccess {
		return nil, errors.New("cannot elevate administrative access above your own role level")
	}

	role := &models.Role{
		ID:            newID,
		RoleName:      req.RoleName,
		IsAdminAccess: req.IsAdminAccess,
		ParentRoleID:  parentRoleID,
		TenantID:      targetTenantID,
		CreatedBy:     actorRole,
		Path:          path,
	}

	if err := s.repo.CreateRole(role); err != nil {
		return nil, err
	}

	parentName := ""
	if role.ParentRoleID != nil {
		p, err := s.repo.GetRoleByID(*role.ParentRoleID)
		if err == nil {
			parentName = p.RoleName
		}
	}

	return &RoleResponse{
		ID:             role.ID,
		RoleName:       role.RoleName,
		IsAdminAccess:  role.IsAdminAccess,
		ParentRoleID:   role.ParentRoleID,
		ParentRoleName: parentName,
		TenantID:       role.TenantID,
		CreatedBy:      role.CreatedBy,
		Path:           role.Path,
		CreatedAt:      role.CreatedAt,
		UpdatedAt:      role.UpdatedAt,
	}, nil
}

func (s *service) UpdateRole(id uuid.UUID, req UpdateRoleRequest, actorRole string, actorSchoolID *uuid.UUID) (*RoleResponse, error) {
	role, err := s.repo.GetRoleByID(id)
	if err != nil {
		return nil, errors.New("role not found")
	}

	// SuperAdmin role protections
	if strings.EqualFold(role.RoleName, "SuperAdmin") {
		return nil, errors.New("the system SuperAdmin role cannot be modified")
	}
	if strings.EqualFold(req.RoleName, "SuperAdmin") && !strings.EqualFold(role.RoleName, "SuperAdmin") {
		return nil, errors.New("cannot rename role to SuperAdmin")
	}

	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return nil, errors.New("unauthorized: actor role not found in system hierarchy")
	}

	// Subtree boundary validation: must be a strict descendant of actor's role
	if actorRole != "SuperAdmin" {
		isDescendant := strings.HasPrefix(role.Path, actorRoleRec.Path)
		isSelf := role.ID == actorRoleRec.ID
		if !isDescendant || isSelf {
			return nil, errors.New("access denied: you can only update roles within your own subtree")
		}
	}

	// Scoped uniqueness check if name is changing
	if req.RoleName != "" && !strings.EqualFold(req.RoleName, role.RoleName) {
		var schoolIDStr *string
		if role.TenantID != nil {
			val := role.TenantID.String()
			schoolIDStr = &val
		}
		existingRoles, err := s.repo.GetAllRoles(schoolIDStr)
		if err == nil {
			for _, er := range existingRoles {
				if strings.EqualFold(er.RoleName, req.RoleName) {
					return nil, errors.New("role name must be unique within this tenant scope")
				}
			}
		}
		role.RoleName = req.RoleName
	}

	// Elevation protection
	if req.IsAdminAccess && !actorRoleRec.IsAdminAccess {
		return nil, errors.New("cannot elevate administrative access above your own role level")
	}
	role.IsAdminAccess = req.IsAdminAccess

	// Tenant reassignment validation
	if actorRole == "SuperAdmin" {
		role.TenantID = req.TenantID
	}

	// Handle parent reparenting and recursive materialized path updates
	parentChanged := false
	if req.ParentRoleID != nil {
		if role.ParentRoleID == nil || *req.ParentRoleID != *role.ParentRoleID {
			newParentRole, err := s.repo.GetRoleByID(*req.ParentRoleID)
			if err != nil {
				return nil, errors.New("proposed parent role not found")
			}

			// Verify actor has access to new parent role
			if actorRole != "SuperAdmin" {
				isParentDescendant := strings.HasPrefix(newParentRole.Path, actorRoleRec.Path)
				isParentSelf := newParentRole.ID == actorRoleRec.ID
				if !isParentDescendant && !isParentSelf {
					return nil, errors.New("cannot reparent role to a target outside your subtree")
				}
			}

			// Prevent circular cycles
			isCircular := newParentRole.ID == role.ID || strings.Contains(newParentRole.Path, "/"+role.ID.String()+"/")
			if isCircular {
				return nil, errors.New("circular hierarchy reference detected")
			}

			// Scoping check
			if newParentRole.TenantID != nil && (role.TenantID == nil || *newParentRole.TenantID != *role.TenantID) {
				return nil, errors.New("cannot parent role to another tenant's role")
			}

			oldPath := role.Path
			newPath := newParentRole.Path + role.ID.String() + "/"

			// Max tree depth check
			depth := strings.Count(newPath, "/") - 1
			if depth > 10 {
				return nil, errors.New("maximum tree hierarchy depth of 10 exceeded")
			}

			role.ParentRoleID = req.ParentRoleID
			role.Path = newPath
			parentChanged = true

			// Cascade path update to all descendants recursively
			repoImpl, ok := s.repo.(*repository)
			if ok {
				var descendants []models.Role
				if err := repoImpl.db.Where("path LIKE ?", oldPath+"%").Find(&descendants).Error; err == nil {
					for _, desc := range descendants {
						desc.Path = strings.Replace(desc.Path, oldPath, newPath, 1)
						repoImpl.db.Save(&desc)
					}
				}
			}
		}
	} else if role.ParentRoleID != nil {
		// Parent changed to nil
		if actorRole != "SuperAdmin" {
			return nil, errors.New("parent role is required for non-SuperAdmins")
		}

		oldPath := role.Path
		newPath := "/" + role.ID.String() + "/"

		role.ParentRoleID = nil
		role.Path = newPath
		parentChanged = true

		repoImpl, ok := s.repo.(*repository)
		if ok {
			var descendants []models.Role
			if err := repoImpl.db.Where("path LIKE ?", oldPath+"%").Find(&descendants).Error; err == nil {
				for _, desc := range descendants {
					desc.Path = strings.Replace(desc.Path, oldPath, newPath, 1)
					repoImpl.db.Save(&desc)
				}
			}
		}
	}
	_ = parentChanged

	if err := s.repo.UpdateRole(role); err != nil {
		return nil, err
	}

	parentName := ""
	if role.ParentRoleID != nil {
		p, err := s.repo.GetRoleByID(*role.ParentRoleID)
		if err == nil {
			parentName = p.RoleName
		}
	}

	return &RoleResponse{
		ID:             role.ID,
		RoleName:       role.RoleName,
		IsAdminAccess:  role.IsAdminAccess,
		ParentRoleID:   role.ParentRoleID,
		ParentRoleName: parentName,
		TenantID:       role.TenantID,
		CreatedBy:      role.CreatedBy,
		Path:           role.Path,
		CreatedAt:      role.CreatedAt,
		UpdatedAt:      role.UpdatedAt,
	}, nil
}

func (s *service) DeleteRole(id uuid.UUID, actorRole string, actorSchoolID *uuid.UUID) error {
	role, err := s.repo.GetRoleByID(id)
	if err != nil {
		return errors.New("role not found")
	}

	// SuperAdmin role protections
	if strings.EqualFold(role.RoleName, "SuperAdmin") {
		return errors.New("the system SuperAdmin role cannot be deleted")
	}

	actorRoleRec, err := s.repo.GetRoleByName(actorRole, actorSchoolID)
	if err != nil {
		return errors.New("unauthorized: actor role not found in system hierarchy")
	}

	// Subtree boundary validation: must be a strict descendant of actor's role
	if actorRole != "SuperAdmin" {
		isDescendant := strings.HasPrefix(role.Path, actorRoleRec.Path)
		isSelf := role.ID == actorRoleRec.ID
		if !isDescendant || isSelf {
			return errors.New("access denied: you can only delete roles within your own subtree")
		}
	}

	// Ensure no users are assigned to this role
	hasUsers, err := s.repo.CheckUsersWithRole(role.RoleName)
	if err != nil {
		return err
	}
	if hasUsers {
		return errors.New("cannot delete role: role is currently assigned to active users")
	}

	// Ensure no descendant child roles exist
	hasChildren, err := s.repo.CheckRoleHasChildren(role.ID)
	if err != nil {
		return err
	}
	if hasChildren {
		return errors.New("cannot delete role: delete descendant child roles first")
	}

	return s.repo.DeleteRole(id)
}
