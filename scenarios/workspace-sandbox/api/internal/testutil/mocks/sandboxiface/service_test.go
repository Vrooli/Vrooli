package sandboxiface

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

func TestFakeService_NotImplementedByDefault(t *testing.T) {
	s := NewFakeService()
	if _, err := s.Get(context.Background(), uuid.New()); err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("default Get should return NotImplemented, got %v", err)
	}
}

func TestFakeService_DispatchesToFn(t *testing.T) {
	called := false
	want := uuid.New()
	s := NewFakeService()
	s.GetFn = func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
		called = true
		return &types.Sandbox{ID: id}, nil
	}
	got, err := s.Get(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("GetFn should have been invoked")
	}
	if got.ID != want {
		t.Errorf("got %s, want %s", got.ID, want)
	}
}
