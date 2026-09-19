package mocks

import (
	"context"
	"errors"
	"testing"
)

func TestFakeGitOps_DefaultsAndCalls(t *testing.T) {
	g := NewFakeGitOps()
	if !g.IsGitRepo(context.Background(), "/tmp") {
		t.Error("default IsGitRepo should be true")
	}
	hash, err := g.GetCommitHash(context.Background(), "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if hash != "abc123" {
		t.Errorf("default CommitHash = %q, want abc123", hash)
	}
	if !g.WasCalled("GetCommitHash") {
		t.Error("WasCalled should report GetCommitHash")
	}
	g.Reset()
	if g.WasCalled("GetCommitHash") {
		t.Error("Reset should clear call history")
	}
}

func TestFakeGitOps_ErrInjection(t *testing.T) {
	g := NewFakeGitOps()
	g.GetCommitHashErr = errors.New("nope")

	if _, err := g.GetCommitHash(context.Background(), "/tmp"); err == nil {
		t.Fatal("expected GetCommitHashErr to surface")
	}
}
