package notes

import "testing"

func TestErrNoteNotFoundError(t *testing.T) {
	err := ErrNoteNotFound{ID: "note-17"}
	if got, want := err.Error(), `note "note-17" not found`; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestErrInvalidNoteError(t *testing.T) {
	err := ErrInvalidNote{Field: "title", Reason: "must not be empty"}
	if got, want := err.Error(), "title: must not be empty"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
