package message

import (
	"strings"

	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	CreateMessage(msg *models.Message) error
	UpdateMessage(msg *models.Message) error
	GetMessageByID(id uuid.UUID) (*models.Message, error)
	GetInboxPaginated(recipientID uuid.UUID, page, limit int, search string) ([]models.Message, int64, error)
	GetSentPaginated(senderID uuid.UUID, page, limit int, search string) ([]models.Message, int64, error)
	GetDrafts(senderID uuid.UUID) ([]models.Message, error)
	GetTrash(userID uuid.UUID) ([]models.Message, error)
	SaveDraft(senderID uuid.UUID, req SaveDraftRequest) (*models.Message, error)
	DeleteDraft(senderID, draftID uuid.UUID) error
	ToggleReadStatus(userID, id uuid.UUID, isRead bool) error
	SoftDeleteMessage(userID, id uuid.UUID) error
	RestoreMessage(userID, id uuid.UUID) error
	GetUnreadCount(userID uuid.UUID) (int64, error)
	SearchUsersByQuery(currentUserID uuid.UUID, query string) ([]models.User, error)
	GetUserByEmail(email string) (*models.User, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateMessage(msg *models.Message) error {
	return r.db.Create(msg).Error
}

func (r *repository) UpdateMessage(msg *models.Message) error {
	return r.db.Save(msg).Error
}

func (r *repository) GetMessageByID(id uuid.UUID) (*models.Message, error) {
	var msg models.Message
	err := r.db.Preload("Sender").Preload("Recipient").First(&msg, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *repository) GetInboxPaginated(recipientID uuid.UUID, page, limit int, search string) ([]models.Message, int64, error) {
	var msgs []models.Message
	var total int64

	query := r.db.Model(&models.Message{}).
		Where("recipient_id = ? AND is_draft = ? AND deleted_by_recipient = ?", recipientID, false, false)

	search = strings.TrimSpace(search)
	if search != "" {
		pattern := "%" + search + "%"
		query = query.Joins("LEFT JOIN users AS senders ON senders.id = messages.sender_id").
			Where("LOWER(messages.subject) LIKE LOWER(?) OR LOWER(messages.body) LIKE LOWER(?) OR LOWER(senders.name) LIKE LOWER(?) OR LOWER(senders.email) LIKE LOWER(?)",
				pattern, pattern, pattern, pattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	err := query.Preload("Sender").Preload("Recipient").
		Order("created_at desc").
		Offset(offset).Limit(limit).
		Find(&msgs).Error

	return msgs, total, err
}

func (r *repository) GetSentPaginated(senderID uuid.UUID, page, limit int, search string) ([]models.Message, int64, error) {
	var msgs []models.Message
	var total int64

	query := r.db.Model(&models.Message{}).
		Where("sender_id = ? AND is_draft = ? AND deleted_by_sender = ?", senderID, false, false)

	search = strings.TrimSpace(search)
	if search != "" {
		pattern := "%" + search + "%"
		query = query.Joins("LEFT JOIN users AS recipients ON recipients.id = messages.recipient_id").
			Where("LOWER(messages.subject) LIKE LOWER(?) OR LOWER(messages.body) LIKE LOWER(?) OR LOWER(recipients.name) LIKE LOWER(?) OR LOWER(recipients.email) LIKE LOWER(?)",
				pattern, pattern, pattern, pattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	err := query.Preload("Sender").Preload("Recipient").
		Order("created_at desc").
		Offset(offset).Limit(limit).
		Find(&msgs).Error

	return msgs, total, err
}

func (r *repository) GetDrafts(senderID uuid.UUID) ([]models.Message, error) {
	var msgs []models.Message
	err := r.db.Preload("Sender").Preload("Recipient").
		Where("sender_id = ? AND is_draft = ? AND deleted_by_sender = ?", senderID, true, false).
		Order("updated_at desc").
		Find(&msgs).Error
	return msgs, err
}

func (r *repository) GetTrash(userID uuid.UUID) ([]models.Message, error) {
	var msgs []models.Message
	err := r.db.Preload("Sender").Preload("Recipient").
		Where("(sender_id = ? AND deleted_by_sender = ?) OR (recipient_id = ? AND deleted_by_recipient = ? AND is_draft = ?)",
			userID, true, userID, true, false).
		Order("updated_at desc").
		Find(&msgs).Error
	return msgs, err
}

func (r *repository) SaveDraft(senderID uuid.UUID, req SaveDraftRequest) (*models.Message, error) {
	var draft models.Message

	if req.ID != nil && *req.ID != uuid.Nil {
		err := r.db.First(&draft, "id = ? AND sender_id = ? AND is_draft = ?", *req.ID, senderID, true).Error
		if err != nil {
			// Create new draft if not found
			draft = models.Message{
				ID:          *req.ID,
				SenderID:    senderID,
				RecipientID: req.RecipientID,
				Subject:     strings.TrimSpace(req.Subject),
				Body:        strings.TrimSpace(req.Body),
				IsDraft:     true,
			}
			if err := r.db.Create(&draft).Error; err != nil {
				return nil, err
			}
		} else {
			draft.RecipientID = req.RecipientID
			draft.Subject = strings.TrimSpace(req.Subject)
			draft.Body = strings.TrimSpace(req.Body)
			if err := r.db.Save(&draft).Error; err != nil {
				return nil, err
			}
		}
	} else {
		draft = models.Message{
			ID:          uuid.New(),
			SenderID:    senderID,
			RecipientID: req.RecipientID,
			Subject:     strings.TrimSpace(req.Subject),
			Body:        strings.TrimSpace(req.Body),
			IsDraft:     true,
		}
		if err := r.db.Create(&draft).Error; err != nil {
			return nil, err
		}
	}

	return r.GetMessageByID(draft.ID)
}

func (r *repository) DeleteDraft(senderID, draftID uuid.UUID) error {
	return r.db.Where("id = ? AND sender_id = ? AND is_draft = ?", draftID, senderID, true).Delete(&models.Message{}).Error
}

func (r *repository) ToggleReadStatus(userID, id uuid.UUID, isRead bool) error {
	return r.db.Model(&models.Message{}).
		Where("id = ? AND (recipient_id = ? OR sender_id = ?)", id, userID, userID).
		Update("is_read", isRead).Error
}

func (r *repository) SoftDeleteMessage(userID, id uuid.UUID) error {
	var msg models.Message
	if err := r.db.First(&msg, "id = ?", id).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{}
	if msg.SenderID == userID {
		updates["deleted_by_sender"] = true
	}
	if msg.RecipientID != nil && *msg.RecipientID == userID {
		updates["deleted_by_recipient"] = true
	}

	if len(updates) == 0 {
		return gorm.ErrRecordNotFound
	}

	return r.db.Model(&msg).Updates(updates).Error
}

func (r *repository) RestoreMessage(userID, id uuid.UUID) error {
	var msg models.Message
	if err := r.db.First(&msg, "id = ?", id).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{}
	if msg.SenderID == userID {
		updates["deleted_by_sender"] = false
	}
	if msg.RecipientID != nil && *msg.RecipientID == userID {
		updates["deleted_by_recipient"] = false
	}

	if len(updates) == 0 {
		return gorm.ErrRecordNotFound
	}

	return r.db.Model(&msg).Updates(updates).Error
}

func (r *repository) GetUnreadCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Message{}).
		Where("recipient_id = ? AND is_draft = ? AND deleted_by_recipient = ? AND is_read = ?", userID, false, false, false).
		Count(&count).Error
	return count, err
}

func (r *repository) SearchUsersByQuery(currentUserID uuid.UUID, query string) ([]models.User, error) {
	var users []models.User
	pattern := "%" + query + "%"
	err := r.db.Where("id != ? AND (LOWER(name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?))", currentUserID, pattern, pattern).
		Limit(10).
		Find(&users).Error
	return users, err
}

func (r *repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("LOWER(email) = LOWER(?)", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
