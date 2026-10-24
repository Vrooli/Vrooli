package forensics

import (
	"context"
	"errors"
	"testing"
)

func TestMCENotInstalled(t *testing.T) {
	exec := &stubExec{err: errors.New("exec: \"ras-mc-ctl\": executable file not found in $PATH")}
	s := NewService(nil, exec, FileSystem{}, fixedNow)
	env := s.MCE(context.Background())
	if env.Available {
		t.Fatal("expected not available")
	}
	if env.Reason != "ras-mc-ctl not installed" {
		t.Errorf("reason: %q", env.Reason)
	}
}

func TestMCEParsesSummary(t *testing.T) {
	out := []byte(`Memory controller events summary:
2 Uncorrected error(s)
17 Corrected error(s)
Other stuff that doesn't matter
`)
	exec := &stubExec{out: out}
	s := NewService(nil, exec, FileSystem{}, fixedNow)
	env := s.MCE(context.Background())
	if !env.Available {
		t.Fatalf("expected available, reason=%q", env.Reason)
	}
	r := env.Data.(MCEReport)
	if r.Uncorrected != 2 {
		t.Errorf("Uncorrected=%d, want 2", r.Uncorrected)
	}
	if r.Corrected != 17 {
		t.Errorf("Corrected=%d, want 17", r.Corrected)
	}
}

func TestMCENoExecutor(t *testing.T) {
	s := NewService(nil, nil, FileSystem{}, fixedNow)
	env := s.MCE(context.Background())
	if env.Available {
		t.Fatal("expected not available")
	}
}

func TestMCEOtherError(t *testing.T) {
	exec := &stubExec{err: errors.New("permission denied")}
	s := NewService(nil, exec, FileSystem{}, fixedNow)
	env := s.MCE(context.Background())
	if env.Available {
		t.Fatal("expected not available")
	}
	if env.Reason == "ras-mc-ctl not installed" {
		t.Errorf("should be classified as failure, not missing: %q", env.Reason)
	}
}
