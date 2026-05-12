package notes

import (
	"context"
	"strings"
)

// AttachmentsService validates and records attachment metadata for notes.
type AttachmentsService interface {
	Create(ctx context.Context, in CreateAttachmentInput) (Attachment, error)
}

type CreateAttachmentInput struct {
	NoteID    string
	Key       string
	MIMEType  string
	SizeBytes int64
}

type attachmentsService struct {
	notesRepo       Repository
	attachmentsRepo AttachmentsRepository
}

func NewAttachmentsService(notesRepo Repository, attachmentsRepo AttachmentsRepository) AttachmentsService {
	return &attachmentsService{notesRepo: notesRepo, attachmentsRepo: attachmentsRepo}
}

var _ AttachmentsService = (*attachmentsService)(nil)

func (s *attachmentsService) Create(ctx context.Context, in CreateAttachmentInput) (Attachment, error) {
	noteID := strings.TrimSpace(in.NoteID)
	if noteID == "" {
		return Attachment{}, ErrInvalidNote{Field: "note_id", Reason: "required"}
	}
	key := strings.TrimSpace(in.Key)
	if key == "" {
		return Attachment{}, ErrInvalidNote{Field: "key", Reason: "required"}
	}
	if in.SizeBytes <= 0 {
		return Attachment{}, ErrInvalidNote{Field: "file", Reason: "empty"}
	}
	if _, err := s.notesRepo.Get(ctx, noteID); err != nil {
		return Attachment{}, err
	}
	return s.attachmentsRepo.CreateAttachment(ctx, Attachment{
		Key:       key,
		MIMEType:  strings.TrimSpace(in.MIMEType),
		SizeBytes: in.SizeBytes,
		NoteID:    noteID,
	})
}
