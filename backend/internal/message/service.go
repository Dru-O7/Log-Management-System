package message

import (
	"errors"
	"office-file-sharing/backend/internal/shared/models"
	"strings"

	"github.com/google/uuid"
)

type Service interface {
	SendMessage(senderID uuid.UUID, req SendMessageRequest) (*MessageResponse, error)
	GetInbox(recipientID uuid.UUID, page, limit int, search string) (*PaginatedMessagesResponse, error)
	GetSent(senderID uuid.UUID, page, limit int, search string) (*PaginatedMessagesResponse, error)
	GetDrafts(senderID uuid.UUID) ([]MessageResponse, error)
	SaveDraft(senderID uuid.UUID, req SaveDraftRequest) (*MessageResponse, error)
	DeleteDraft(senderID, draftID uuid.UUID) error
	GetTrash(userID uuid.UUID) ([]MessageResponse, error)
	GetMessageDetails(id, currentUserID uuid.UUID) (*MessageResponse, error)
	ToggleReadStatus(userID, id uuid.UUID, isRead bool) error
	SoftDeleteMessage(userID, id uuid.UUID) error
	RestoreMessage(userID, id uuid.UUID) error
	GetUnreadCount(userID uuid.UUID) (*UnreadCountResponse, error)
	SearchUsers(currentUserID uuid.UUID, query string) ([]UserSearchResponse, error)
	GetUserByEmail(email string) (*UserSearchResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) SendMessage(senderID uuid.UUID, req SendMessageRequest) (*MessageResponse, error) {
	if req.RecipientID == uuid.Nil {
		return nil, errors.New("recipient is required")
	}
	if req.RecipientID == senderID {
		return nil, errors.New("you cannot message yourself")
	}
	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		subject = "Chat Message"
	}
	if strings.TrimSpace(req.Body) == "" {
		return nil, errors.New("message body is required")
	}

	var msg *models.Message

	if req.DraftID != nil && *req.DraftID != uuid.Nil {
		existingDraft, err := s.repo.GetMessageByID(*req.DraftID)
		if err == nil && existingDraft != nil && existingDraft.SenderID == senderID && existingDraft.IsDraft {
			existingDraft.RecipientID = &req.RecipientID
			existingDraft.Subject = subject
			existingDraft.Body = strings.TrimSpace(req.Body)
			existingDraft.IsDraft = false
			if err := s.repo.UpdateMessage(existingDraft); err != nil {
				return nil, err
			}
			msg = existingDraft
		}
	}

	if msg == nil {
		recipientID := req.RecipientID
		msg = &models.Message{
			ID:          uuid.New(),
			SenderID:    senderID,
			RecipientID: &recipientID,
			Subject:     subject,
			Body:        strings.TrimSpace(req.Body),
			IsRead:      false,
			IsDraft:     false,
		}

		if err := s.repo.CreateMessage(msg); err != nil {
			return nil, err
		}
	}

	fetchedMsg, err := s.repo.GetMessageByID(msg.ID)
	if err != nil {
		return nil, err
	}

	return s.toMessageResponse(fetchedMsg), nil
}

func (s *service) GetInbox(recipientID uuid.UUID, page, limit int, search string) (*PaginatedMessagesResponse, error) {
	msgs, total, err := s.repo.GetInboxPaginated(recipientID, page, limit, search)
	if err != nil {
		return nil, err
	}

	responses := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		responses[i] = *s.toMessageResponse(&m)
	}

	return &PaginatedMessagesResponse{
		Messages: responses,
		Total:    total,
		Page:     page,
		Limit:    limit,
	}, nil
}

func (s *service) GetSent(senderID uuid.UUID, page, limit int, search string) (*PaginatedMessagesResponse, error) {
	msgs, total, err := s.repo.GetSentPaginated(senderID, page, limit, search)
	if err != nil {
		return nil, err
	}

	responses := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		responses[i] = *s.toMessageResponse(&m)
	}

	return &PaginatedMessagesResponse{
		Messages: responses,
		Total:    total,
		Page:     page,
		Limit:    limit,
	}, nil
}

func (s *service) GetDrafts(senderID uuid.UUID) ([]MessageResponse, error) {
	msgs, err := s.repo.GetDrafts(senderID)
	if err != nil {
		return nil, err
	}

	responses := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		responses[i] = *s.toMessageResponse(&m)
	}
	return responses, nil
}

func (s *service) SaveDraft(senderID uuid.UUID, req SaveDraftRequest) (*MessageResponse, error) {
	draft, err := s.repo.SaveDraft(senderID, req)
	if err != nil {
		return nil, err
	}
	return s.toMessageResponse(draft), nil
}

func (s *service) DeleteDraft(senderID, draftID uuid.UUID) error {
	return s.repo.DeleteDraft(senderID, draftID)
}

func (s *service) GetTrash(userID uuid.UUID) ([]MessageResponse, error) {
	msgs, err := s.repo.GetTrash(userID)
	if err != nil {
		return nil, err
	}

	responses := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		responses[i] = *s.toMessageResponse(&m)
	}
	return responses, nil
}

func (s *service) GetMessageDetails(id, currentUserID uuid.UUID) (*MessageResponse, error) {
	msg, err := s.repo.GetMessageByID(id)
	if err != nil {
		return nil, errors.New("message not found")
	}

	isRecipient := msg.RecipientID != nil && *msg.RecipientID == currentUserID

	if msg.SenderID != currentUserID && !isRecipient {
		return nil, errors.New("unauthorized to view this message")
	}

	// Mark as read if current user is the recipient
	if isRecipient && !msg.IsRead {
		_ = s.repo.ToggleReadStatus(currentUserID, id, true)
		msg.IsRead = true
	}

	return s.toMessageResponse(msg), nil
}

func (s *service) ToggleReadStatus(userID, id uuid.UUID, isRead bool) error {
	return s.repo.ToggleReadStatus(userID, id, isRead)
}

func (s *service) SoftDeleteMessage(userID, id uuid.UUID) error {
	return s.repo.SoftDeleteMessage(userID, id)
}

func (s *service) RestoreMessage(userID, id uuid.UUID) error {
	return s.repo.RestoreMessage(userID, id)
}

func (s *service) GetUnreadCount(userID uuid.UUID) (*UnreadCountResponse, error) {
	count, err := s.repo.GetUnreadCount(userID)
	if err != nil {
		return nil, err
	}
	return &UnreadCountResponse{Count: count}, nil
}

func (s *service) SearchUsers(currentUserID uuid.UUID, query string) ([]UserSearchResponse, error) {
	query = strings.TrimSpace(query)
	if len(query) < 1 {
		return []UserSearchResponse{}, nil
	}

	users, err := s.repo.SearchUsersByQuery(currentUserID, query)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 && strings.Contains(query, "@") {
		exactUser, err := s.repo.GetUserByEmail(query)
		if err == nil && exactUser != nil && exactUser.ID != currentUserID {
			users = append(users, *exactUser)
		}
	}

	res := make([]UserSearchResponse, len(users))
	for i, u := range users {
		res[i] = UserSearchResponse{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
			Role:  u.Role,
		}
	}
	return res, nil
}

func (s *service) GetUserByEmail(email string) (*UserSearchResponse, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, errors.New("email is required")
	}

	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return nil, errors.New("user not found with this email")
	}

	return &UserSearchResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

func (s *service) toMessageResponse(m *models.Message) *MessageResponse {
	resp := &MessageResponse{
		ID:                 m.ID,
		SenderID:           m.SenderID,
		SenderName:         m.Sender.Name,
		SenderEmail:        m.Sender.Email,
		SenderRole:         m.Sender.Role,
		Subject:            m.Subject,
		Body:               m.Body,
		IsRead:             m.IsRead,
		IsDraft:            m.IsDraft,
		DeletedBySender:    m.DeletedBySender,
		DeletedByRecipient: m.DeletedByRecipient,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}

	if m.RecipientID != nil {
		resp.RecipientID = m.RecipientID
		if m.Recipient != nil {
			resp.RecipientName = m.Recipient.Name
			resp.RecipientEmail = m.Recipient.Email
			resp.RecipientRole = m.Recipient.Role
		}
	}

	return resp
}
