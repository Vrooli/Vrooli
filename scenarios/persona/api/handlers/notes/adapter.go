package notes

import (
	"persona/internal/notes"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/notes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// domainToProto converts an internal notes.Note into the wire shape
// the notes proto declares. Timestamp lives in proto so every generated
// client sees a real time type rather than a string convention.
//
// Lives in the handler package by intent. The conversion is mechanical
// and only used at the transport edge; pulling it into a separate
// adapters package would create a one-import wrapper for no gain.
func domainToProto(n notes.Note) *notesv1.Note {
	return &notesv1.Note{
		Id:             n.ID,
		Title:          n.Title,
		Body:           n.Body,
		CreatedAt:      timestamppb.New(n.CreatedAt.UTC()),
		UpdatedAt:      timestamppb.New(n.UpdatedAt.UTC()),
		AttachmentKeys: append([]string(nil), n.AttachmentKeys...),
	}
}

func attachmentToProto(a notes.Attachment) *notesv1.Attachment {
	return &notesv1.Attachment{
		Key:        a.Key,
		MimeType:   a.MIMEType,
		SizeBytes:  a.SizeBytes,
		NoteId:     a.NoteID,
		UploadedAt: timestamppb.New(a.UploadedAt.UTC()),
	}
}
