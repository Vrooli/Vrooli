package mocks

import (
	"context"
	"errors"
	"testing"
)

func TestFakeModelProberRecordsModels(t *testing.T) {
	prober := NewFakeModelProber()

	if err := prober.ProbeModel(context.Background(), "gpt-5"); err != nil {
		t.Fatalf("ProbeModel returned unexpected error: %v", err)
	}

	got := prober.Models()
	if len(got) != 1 || got[0] != "gpt-5" {
		t.Fatalf("expected recorded model gpt-5, got %v", got)
	}
}

func TestFailingModelProberReturnsError(t *testing.T) {
	wantErr := errors.New("unknown model")
	prober := NewFailingModelProber(wantErr)

	gotErr := prober.ProbeModel(context.Background(), "gpt-5")
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, gotErr)
	}
}
