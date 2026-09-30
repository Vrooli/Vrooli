package agentsessions

import (
	"context"
	"mime"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/idgen"
)

var allowedSessionImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func (s *Service) UploadAttachments(ctx context.Context, sessionID string, uploads []AttachmentUpload) ([]Attachment, error) {
	if len(uploads) == 0 {
		return []Attachment{}, nil
	}
	if len(uploads) > 6 {
		return nil, apierr.BadRequest("no more than 6 image attachments are allowed per message")
	}
	store, err := s.storeFor(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := store.LoadSession(strings.TrimSpace(sessionID)); err != nil {
		return nil, mapStoreError(err)
	}
	attachments := make([]Attachment, 0, len(uploads))
	for _, upload := range uploads {
		mediaType, _, _ := mime.ParseMediaType(upload.ContentType)
		if !allowedSessionImageTypes[mediaType] {
			return nil, apierr.BadRequest("unsupported file type: %s", mediaType)
		}
		if upload.Reader == nil {
			return nil, apierr.BadRequest("attachment file is required")
		}
		attachment := Attachment{
			ID:          "att_" + idgen.Generate(),
			Filename:    strings.TrimSpace(upload.Filename),
			ContentType: mediaType,
			SizeBytes:   upload.SizeBytes,
			CreatedAt:   nowRFC3339(),
		}
		if attachment.Filename == "" {
			attachment.Filename = "unnamed"
		}
		if err := store.SaveAttachment(sessionID, attachment, upload.Reader); err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}
	return attachments, nil
}

func (s *Service) AttachmentPath(ctx context.Context, sessionID string, attachmentID string) (string, Attachment, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return "", Attachment{}, err
	}
	path, attachment, err := store.AttachmentPath(strings.TrimSpace(sessionID), strings.TrimSpace(attachmentID))
	if err != nil {
		return "", Attachment{}, mapStoreError(err)
	}
	return path, attachment, nil
}

func sessionAttachmentsByID(session Session, attachmentIDs []string) []Attachment {
	if len(attachmentIDs) == 0 || len(session.Attachments) == 0 {
		return nil
	}
	wanted := make(map[string]struct{}, len(attachmentIDs))
	for _, id := range attachmentIDs {
		wanted[strings.TrimSpace(id)] = struct{}{}
	}
	var attachments []Attachment
	for _, attachment := range session.Attachments {
		if _, ok := wanted[attachment.ID]; ok {
			attachments = append(attachments, attachment)
		}
	}
	return attachments
}
