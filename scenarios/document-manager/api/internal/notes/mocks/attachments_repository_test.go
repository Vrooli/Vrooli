package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"document-manager/internal/notes"

	"github.com/stretchr/testify/require"
)

func TestFakeAttachmentsRepository_CreateBackfillsUploadedAt(t *testing.T) {
	var f FakeAttachmentsRepository
	got, err := f.CreateAttachment(context.Background(), notes.Attachment{
		Key:    "notes/a/file.txt",
		NoteID: "a",
	})
	require.NoError(t, err)
	require.False(t, got.UploadedAt.IsZero(),
		"CreateAttachment must backfill UploadedAt when caller leaves it zero, mirroring the sqlite repository")
	require.Equal(t, time.UTC, got.UploadedAt.Location(),
		"backfilled timestamp must be UTC for wire parity with the sqlite repository")
	require.Equal(t, int64(1), f.CreateCalls.Load())
	require.Len(t, f.Attachments, 1)
}

func TestFakeAttachmentsRepository_CreateRespectsCallerTimestamp(t *testing.T) {
	want := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	var f FakeAttachmentsRepository
	got, err := f.CreateAttachment(context.Background(), notes.Attachment{
		Key:        "notes/a/file.txt",
		NoteID:     "a",
		UploadedAt: want,
	})
	require.NoError(t, err)
	require.True(t, want.Equal(got.UploadedAt),
		"CreateAttachment must not overwrite a caller-supplied UploadedAt")
}

func TestFakeAttachmentsRepository_CreateErrSurfaces(t *testing.T) {
	want := errors.New("create boom")
	f := &FakeAttachmentsRepository{CreateErr: want}
	_, err := f.CreateAttachment(context.Background(), notes.Attachment{Key: "k", NoteID: "n"})
	require.ErrorIs(t, err, want)
	require.Empty(t, f.Attachments, "failed CreateAttachment must not mutate state")
}

func TestFakeAttachmentsRepository_ListAttachmentKeysFiltersByNoteID(t *testing.T) {
	f := &FakeAttachmentsRepository{Attachments: []notes.Attachment{
		{Key: "notes/a/one.txt", NoteID: "a"},
		{Key: "notes/b/two.txt", NoteID: "b"},
		{Key: "notes/a/three.txt", NoteID: "a"},
	}}
	got, err := f.ListAttachmentKeys(context.Background(), "a")
	require.NoError(t, err)
	require.Equal(t, []string{"notes/a/one.txt", "notes/a/three.txt"}, got,
		"only keys for the requested note id should be returned, in insertion order")
}

func TestFakeAttachmentsRepository_ListAttachmentKeysReturnsNilForMiss(t *testing.T) {
	f := &FakeAttachmentsRepository{Attachments: []notes.Attachment{
		{Key: "notes/a/one.txt", NoteID: "a"},
	}}
	got, err := f.ListAttachmentKeys(context.Background(), "b")
	require.NoError(t, err)
	require.Empty(t, got, "an unmatched note id is a clean empty result, not an error")
}

func TestFakeAttachmentsRepository_ListErrSurfaces(t *testing.T) {
	want := errors.New("list boom")
	f := &FakeAttachmentsRepository{
		ListErr:     want,
		Attachments: []notes.Attachment{{Key: "k", NoteID: "n"}},
	}
	// ListErr overrides even when the in-memory store has a match —
	// tests need to be able to drive the internal-error path
	// independently of the empty-result path.
	_, err := f.ListAttachmentKeys(context.Background(), "n")
	require.ErrorIs(t, err, want)
}

// TestFakeAttachmentsRepository_RaceCleanWhenSharedAcrossGoroutines
// pins the race-cleanliness of the mutex + atomic counters. Mirrors
// the FakeRepository race test so scenarios that copy these fakes
// inherit the same parallel-test guarantee.
func TestFakeAttachmentsRepository_RaceCleanWhenSharedAcrossGoroutines(t *testing.T) {
	t.Parallel()
	const goroutines = 50
	var f FakeAttachmentsRepository
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = f.CreateAttachment(context.Background(), notes.Attachment{Key: "k", NoteID: "n"})
		}()
	}
	wg.Wait()
	require.Equal(t, int64(goroutines), f.CreateCalls.Load())
	require.Len(t, f.Attachments, goroutines)
}
