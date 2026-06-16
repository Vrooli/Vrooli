package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"

	"image-tools/internal/notes"

	"github.com/stretchr/testify/require"
)

func TestFakeAttachmentsService_CreateRecordsInputAndSynthesisesAttachment(t *testing.T) {
	var f FakeAttachmentsService
	got, err := f.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "n",
		Key:       "notes/n/file.txt",
		MIMEType:  "text/plain",
		SizeBytes: 5,
	})
	require.NoError(t, err)
	require.Equal(t, "n", got.NoteID)
	require.Equal(t, "notes/n/file.txt", got.Key)
	require.Equal(t, "text/plain", got.MIMEType)
	require.Equal(t, int64(5), got.SizeBytes)
	require.Equal(t, int64(1), f.CreateCalls.Load())
	require.Len(t, f.CreateInputs, 1)
	require.Equal(t, "notes/n/file.txt", f.CreateInputs[0].Key)
}

func TestFakeAttachmentsService_CreateOutOverridesSynthesis(t *testing.T) {
	canned := notes.Attachment{Key: "fixed", NoteID: "n"}
	f := &FakeAttachmentsService{CreateOut: &canned}
	got, err := f.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID: "n",
		Key:    "ignored",
	})
	require.NoError(t, err)
	require.Equal(t, "fixed", got.Key, "CreateOut must take precedence over synthesis")
}

func TestFakeAttachmentsService_CreateErrSurfaces(t *testing.T) {
	want := errors.New("create boom")
	f := &FakeAttachmentsService{CreateErr: want}
	_, err := f.Create(context.Background(), notes.CreateAttachmentInput{NoteID: "n", Key: "k"})
	require.ErrorIs(t, err, want)
	require.Len(t, f.CreateInputs, 1, "input still recorded — proves the call reached the fake")
}

// TestFakeAttachmentsService_RaceCleanWhenSharedAcrossGoroutines pins
// the race-cleanliness of the input-recording path. Mirrors the
// FakeService race test to keep the patterns symmetrical for scenarios
// that copy these mocks.
func TestFakeAttachmentsService_RaceCleanWhenSharedAcrossGoroutines(t *testing.T) {
	t.Parallel()
	const goroutines = 50
	var f FakeAttachmentsService
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = f.Create(context.Background(), notes.CreateAttachmentInput{NoteID: "n", Key: "k", SizeBytes: 1})
		}()
	}
	wg.Wait()
	require.Equal(t, int64(goroutines), f.CreateCalls.Load())
	require.Len(t, f.CreateInputs, goroutines)
}
