package projectid

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestParse(t *testing.T) {
	want := uuid.New()
	got, err := Parse("  " + want.String() + "  ")
	if err != nil || got != want {
		t.Fatalf("Parse valid ID = %v, %v; want %v, nil", got, err, want)
	}
	if _, err := Parse(" \t "); !errors.Is(err, ErrRequired) {
		t.Fatalf("Parse blank error = %v; want ErrRequired", err)
	}
	if _, err := Parse("not-a-uuid"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Parse malformed error = %v; want ErrInvalid", err)
	}
}
