package message

import (
	"time"

	"github.com/google/uuid"
)

type SendMessageRequest struct {
	DraftID     *uuid.UUID `json:"draft_id,omitempty"`
	RecipientID uuid.UUID  `json:"recipient_id"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
}

type SaveDraftRequest struct {
	ID          *uuid.UUID `json:"id,omitempty"`
	RecipientID *uuid.UUID `json:"recipient_id,omitempty"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
}

type ToggleReadRequest struct {
	IsRead bool `json:"is_read"`
}

type MessageResponse struct {
	ID                 uuid.UUID  `json:"id"`
	SenderID           uuid.UUID  `json:"sender_id"`
	SenderName         string     `json:"sender_name"`
	SenderEmail        string     `json:"sender_email"`
	SenderRole         string     `json:"sender_role"`
	RecipientID        *uuid.UUID `json:"recipient_id"`
	RecipientName      string     `json:"recipient_name"`
	RecipientEmail     string     `json:"recipient_email"`
	RecipientRole      string     `json:"recipient_role"`
	Subject            string     `json:"subject"`
	Body               string     `json:"body"`
	IsRead             bool       `json:"is_read"`
	IsDraft            bool       `json:"is_draft"`
	DeletedBySender    bool       `json:"deleted_by_sender"`
	DeletedByRecipient bool       `json:"deleted_by_recipient"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type PaginatedMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	Limit    int               `json:"limit"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type UserSearchResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
}
