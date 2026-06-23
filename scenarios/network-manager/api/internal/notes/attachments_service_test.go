package notes_test

import (
	"context"
	"errors"
	"testing"

	"network-manager/internal/notes"

	"github.com/stretchr/testify/require"

	mocks "network-manager/internal/notes/mocks"
)

// newAttachmentsServiceWithNote returns an attachments service whose
// notes-side fake already contains a note with the given id, so happy-
// path tests can focus on the attachments validation surface without
// re-arranging the parent.
func newAttachmentsServiceWithNote(t *testing.T, noteID string) (notes.AttachmentsService, *mocks.FakeRepository, *mocks.FakeAttachmentsRepository) {
	t.Helper()
	notesRepo := mocks.NewFakeRepository()
	notesRepo.Items = []notes.Note{{ID: noteID, Title: "parent"}}
	attachmentsRepo := &mocks.FakeAttachmentsRepository{}
	svc := notes.NewAttachmentsService(notesRepo, attachmentsRepo)
	return svc, notesRepo, attachmentsRepo
}

func TestAttachmentsService_Create_RejectsEmptyNoteID(t *testing.T) {
	svc, _, attachmentsRepo := newAttachmentsServiceWithNote(t, "n")

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "",
		Key:       "k",
		SizeBytes: 1,
	})
	require.Error(t, err)
	var inv notes.ErrInvalidNote
	require.True(t, errors.As(err, &inv), "expected ErrInvalidNote, got %T: %v", err, err)
	require.Equal(t, "note_id", inv.Field)
	require.Equal(t, int64(0), attachmentsRepo.CreateCalls.Load(),
		"validation must reject before reaching the attachments repository")
}

func TestAttachmentsService_Create_TrimsNoteIDWhitespace(t *testing.T) {
	svc, _, _ := newAttachmentsServiceWithNote(t, "n")

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "   ",
		Key:       "k",
		SizeBytes: 1,
	})
	require.Error(t, err)
	var inv notes.ErrInvalidNote
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "note_id", inv.Field,
		"whitespace-only note id must be rejected the same as empty")
}

func TestAttachmentsService_Create_RejectsEmptyKey(t *testing.T) {
	svc, _, _ := newAttachmentsServiceWithNote(t, "n")

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "n",
		Key:       "",
		SizeBytes: 1,
	})
	require.Error(t, err)
	var inv notes.ErrInvalidNote
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "key", inv.Field)
}

func TestAttachmentsService_Create_TrimsKeyWhitespace(t *testing.T) {
	svc, _, _ := newAttachmentsServiceWithNote(t, "n")

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "n",
		Key:       "   ",
		SizeBytes: 1,
	})
	require.Error(t, err)
	var inv notes.ErrInvalidNote
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "key", inv.Field)
}

func TestAttachmentsService_Create_RejectsZeroSize(t *testing.T) {
	svc, _, _ := newAttachmentsServiceWithNote(t, "n")

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "n",
		Key:       "k",
		SizeBytes: 0,
	})
	require.Error(t, err)
	var inv notes.ErrInvalidNote
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "file", inv.Field)
}

func TestAttachmentsService_Create_RejectsNegativeSize(t *testing.T) {
	svc, _, _ := newAttachmentsServiceWithNote(t, "n")

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "n",
		Key:       "k",
		SizeBytes: -1,
	})
	require.Error(t, err)
	var inv notes.ErrInvalidNote
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "file", inv.Field,
		"negative SizeBytes is the same as 0 — both mean 'no bytes uploaded'")
}

func TestAttachmentsService_Create_PropagatesNotFoundFromNotesRepo(t *testing.T) {
	notesRepo := mocks.NewFakeRepository()
	svc := notes.NewAttachmentsService(notesRepo, &mocks.FakeAttachmentsRepository{})

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "ghost",
		Key:       "k",
		SizeBytes: 1,
	})
	require.Error(t, err)
	var nf notes.ErrNoteNotFound
	require.True(t, errors.As(err, &nf), "service must propagate the typed sentinel verbatim")
	require.Equal(t, "ghost", nf.ID)
}

func TestAttachmentsService_Create_PropagatesArbitraryNotesRepoError(t *testing.T) {
	want := errors.New("get boom")
	notesRepo := mocks.NewFakeRepository()
	notesRepo.GetErr = want
	attachmentsRepo := &mocks.FakeAttachmentsRepository{}
	svc := notes.NewAttachmentsService(notesRepo, attachmentsRepo)

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "n",
		Key:       "k",
		SizeBytes: 1,
	})
	require.ErrorIs(t, err, want)
	require.Equal(t, int64(0), attachmentsRepo.CreateCalls.Load(),
		"attachments repo must not be called when the notes-side lookup fails")
}

func TestAttachmentsService_Create_DelegatesTrimmedFieldsToRepo(t *testing.T) {
	svc, _, attachmentsRepo := newAttachmentsServiceWithNote(t, "n")

	got, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "  n  ",
		Key:       "  notes/n/file.txt  ",
		MIMEType:  "  text/plain  ",
		SizeBytes: 5,
	})
	require.NoError(t, err)
	require.Equal(t, "n", got.NoteID, "trimmed value, not raw input, is what reaches storage")
	require.Equal(t, "notes/n/file.txt", got.Key)
	require.Equal(t, "text/plain", got.MIMEType)
	require.Equal(t, int64(5), got.SizeBytes)
	require.Equal(t, int64(1), attachmentsRepo.CreateCalls.Load())
	require.Len(t, attachmentsRepo.Attachments, 1)
}

func TestAttachmentsService_Create_PropagatesAttachmentsRepoError(t *testing.T) {
	want := errors.New("repo boom")
	notesRepo := mocks.NewFakeRepository()
	notesRepo.Items = []notes.Note{{ID: "n"}}
	attachmentsRepo := &mocks.FakeAttachmentsRepository{CreateErr: want}
	svc := notes.NewAttachmentsService(notesRepo, attachmentsRepo)

	_, err := svc.Create(context.Background(), notes.CreateAttachmentInput{
		NoteID:    "n",
		Key:       "k",
		SizeBytes: 1,
	})
	require.ErrorIs(t, err, want)
	require.Empty(t, attachmentsRepo.Attachments, "repo errors must not produce a half-inserted row")
}
