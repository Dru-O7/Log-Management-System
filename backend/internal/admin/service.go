package admin

import (
	"errors"
	"office-file-sharing/backend/internal/shared/models"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	GetStats() (*SystemStats, error)
	GetAllUsers() ([]UserResponse, error)
	CreateUser(req CreateUserRequest) (*UserResponse, error)
	UpdateUser(id uuid.UUID, req UpdateUserRequest) (*UserResponse, error)
	DeleteUser(id uuid.UUID) error
	GetAllDocumentTypes() ([]DocumentTypeResponse, error)
	CreateDocumentType(req CreateDocTypeRequest) (*DocumentTypeResponse, error)
	UpdateDocumentType(id uuid.UUID, req UpdateDocTypeRequest) (*DocumentTypeResponse, error)
	DeleteDocumentType(id uuid.UUID) error
	GetAllSchools() ([]SchoolResponse, error)
	UpdateSchool(id uuid.UUID, req UpdateSchoolRequest) (*SchoolResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetStats() (*SystemStats, error) {
	return s.repo.GetStats()
}

func (s *service) GetAllUsers() ([]UserResponse, error) {
	return s.repo.GetAllUsers()
}

func (s *service) CreateUser(req CreateUserRequest) (*UserResponse, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		return nil, errors.New("name and email are required")
	}
	if req.Password == "" {
		req.Password = "password" // default password
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
		SchoolID:     req.SchoolID,
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

func (s *service) UpdateUser(id uuid.UUID, req UpdateUserRequest) (*UserResponse, error) {
	u, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if req.Name != "" {
		u.Name = req.Name
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Role != "" {
		u.Role = req.Role
	}
	if req.SchoolID != nil {
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
	return s.repo.DeleteUser(id)
}

func (s *service) GetAllDocumentTypes() ([]DocumentTypeResponse, error) {
	return s.repo.GetAllDocumentTypes()
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
	if req.SlaHours == 0 {
		req.SlaHours = 72
	}

	dt := &models.DocumentType{
		ID:                uuid.New(),
		SchoolID:          req.SchoolID,
		Name:              req.Name,
		Slug:              req.Slug,
		WorkflowStages:    req.WorkflowStages,
		RequiredFields:    req.RequiredFields,
		SlaHours:          req.SlaHours,
		NeedsParentCosign: req.NeedsParentCosign,
		Active:            true,
	}

	if err := s.repo.CreateDocumentType(dt); err != nil {
		return nil, err
	}

	return &DocumentTypeResponse{
		ID:                dt.ID,
		SchoolID:          dt.SchoolID,
		Name:              dt.Name,
		Slug:              dt.Slug,
		WorkflowStages:    dt.WorkflowStages,
		RequiredFields:    dt.RequiredFields,
		SlaHours:          dt.SlaHours,
		NeedsParentCosign: dt.NeedsParentCosign,
		Active:            dt.Active,
	}, nil
}

func (s *service) UpdateDocumentType(id uuid.UUID, req UpdateDocTypeRequest) (*DocumentTypeResponse, error) {
	dt, err := s.repo.GetDocumentTypeByID(id)
	if err != nil {
		return nil, errors.New("document type not found")
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
	if req.SlaHours > 0 {
		dt.SlaHours = req.SlaHours
	}
	dt.NeedsParentCosign = req.NeedsParentCosign
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
		SlaHours:          dt.SlaHours,
		NeedsParentCosign: dt.NeedsParentCosign,
		Active:            dt.Active,
	}, nil
}

func (s *service) DeleteDocumentType(id uuid.UUID) error {
	return s.repo.DeleteDocumentType(id)
}

func (s *service) GetAllSchools() ([]SchoolResponse, error) {
	return s.repo.GetAllSchools()
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
