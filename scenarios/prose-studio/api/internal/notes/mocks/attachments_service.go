package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"prose-studio/internal/notes"
)

// FakeAttachmentsService satisfies notes.AttachmentsService for handler
// tests that don't want validation/repository/blob-store plumbing in
// scope. Mirrors FakeService's surface: records each call's input,
// returns a canned Attachment when CreateOut is set, synthesises one
// from the input otherwise, and gates on CreateErr first so failure
// paths can be driven cleanly.
//
// Used wherever a future attachments-handler test wants to assert on
// routing, multipart wiring, or error translation without standing up
// the real notes-and-attachments service tree.
type FakeAttachmentsService struct {
	mu sync.Mutex

	// CreateInputs records each Create call's input in order.
	CreateInputs []notes.CreateAttachmentInput
	// CreateOut, when non-nil, is returned verbatim from Create on
	// success. When nil, FakeAttachmentsService synthesises an
	// Attachment from the input fields so the handler's success path
	// renders something sensible without test boilerplate.
	CreateOut *notes.Attachment
	CreateErr error

	CreateCalls atomic.Int64
}

func (f *FakeAttachmentsService) Create(ctx context.Context, in notes.CreateAttachmentInput) (notes.Attachment, error) {
	f.CreateCalls.Add(1)

	f.mu.Lock()
	f.CreateInputs = append(f.CreateInputs, in)
	f.mu.Unlock()

	if f.CreateErr != nil {
		return notes.Attachment{}, f.CreateErr
	}
	if f.CreateOut != nil {
		return *f.CreateOut, nil
	}
	return notes.Attachment{
		Key:       in.Key,
		MIMEType:  in.MIMEType,
		SizeBytes: in.SizeBytes,
		NoteID:    in.NoteID,
	}, nil
}

// Compile-time guarantee.
var _ notes.AttachmentsService = (*FakeAttachmentsService)(nil)
