package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
	"hello-python/internal/notes"
)

// FakeAttachmentsRepository satisfies notes.AttachmentsRepository for
// service tests that don't want the sqlite round-trip. Mirrors
// FakeRepository's shape: in-memory slice of Attachments, per-method
// error knobs (CreateErr, ListErr), and atomic call counters so
// `go test -race` stays quiet under fan-out.
//
// Production attachment rows are inserted through the sqlite repository
// in attachments_sqlite.go; this fake exists so `attachmentsService`
// tests can drive the validation + delegation surface without standing
// up the on-disk schema.
type FakeAttachmentsRepository struct {
	mu sync.Mutex

	// Attachments records each successful insert in arrival order.
	// ListAttachmentKeys filters by NoteID so a single fake serves
	// multiple notes within one test.
	Attachments []notes.Attachment

	CreateErr error
	ListErr   error

	CreateCalls atomic.Int64
	ListCalls   atomic.Int64
}

// CreateAttachment appends a to the in-memory slice. Mirrors the
// sqlite repository's UploadedAt backfill (zero time → now in UTC) so
// service tests don't have to pre-populate the timestamp. Returns
// CreateErr if set, before mutating state — keeps the failure path
// observable as "didn't insert" via len(f.Attachments).
func (f *FakeAttachmentsRepository) CreateAttachment(ctx context.Context, a notes.Attachment) (notes.Attachment, error) {
	f.CreateCalls.Add(1)
	if f.CreateErr != nil {
		return notes.Attachment{}, f.CreateErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if a.UploadedAt.IsZero() {
		a.UploadedAt = time.Now().UTC()
	}
	f.Attachments = append(f.Attachments, a)
	return a, nil
}

// ListAttachmentKeys returns the keys of every recorded attachment
// whose NoteID matches. Returns ListErr if set (overrides any
// in-memory matches) so tests can drive the internal-error path
// independently of the empty-result path. Order is insertion order;
// tests that care about sqlite's `ORDER BY uploaded_at DESC` should
// use the real repository.
func (f *FakeAttachmentsRepository) ListAttachmentKeys(ctx context.Context, noteID string) ([]string, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return nil, f.ListErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	var keys []string
	for _, a := range f.Attachments {
		if a.NoteID == noteID {
			keys = append(keys, a.Key)
		}
	}
	return keys, nil
}

// Compile-time guarantee that *FakeAttachmentsRepository satisfies
// notes.AttachmentsRepository.
var _ notes.AttachmentsRepository = (*FakeAttachmentsRepository)(nil)
