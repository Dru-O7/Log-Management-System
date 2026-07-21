package main

import (
	"fmt"
	"log"

	"office-file-sharing/backend/internal/shared/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=postgres password=110085 dbname=office_files port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var orgs []models.Organization
	err = db.Preload("ParentOrg").Preload("PointOfContact").Find(&orgs).Error
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total organizations found: %d\n", len(orgs))
	for _, org := range orgs {
		fmt.Printf("Org ID: %s, Name: %s, POC ID: %v\n", org.ID, org.OrganizationName, org.PointOfContactID)
		if org.PointOfContact != nil {
			fmt.Printf("  -> POC User Name: %s, Email: %s\n", org.PointOfContact.Name, org.PointOfContact.Email)
		} else {
			fmt.Println("  -> POC User is NIL!")
		}
	}
}
