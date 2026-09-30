package analysis

import (
	"context"
	"errors"
	"testing"
)

// TestAutoScan_FlagsGeneratedOutput is the IMG-P0-004 integration: the NSFW
// auto-scan adapter (used by the AI generation hook) returns the classifier
// verdict for generated output.
func TestAutoScan_FlagsGeneratedOutput(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		NSFWThreshold:  0.5,
		LookPath:       func(string) (string, error) { return "/usr/bin/python3", nil },
		Run: func(context.Context, string, []string) ([]byte, error) {
			return []byte(`{"score":0.88,"categories":[]}`), nil
		},
	})
	nsfw, score, err := svc.ScanNSFW(context.Background(), []byte("generated"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !nsfw {
		t.Error("expected flagged output")
	}
	if score < 0.5 {
		t.Errorf("score = %v, want >= 0.5", score)
	}
}

// TestAutoScan_ClearOutput passes SFW output through unflagged.
func TestAutoScan_ClearOutput(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		LookPath:       func(string) (string, error) { return "/usr/bin/python3", nil },
		Run: func(context.Context, string, []string) ([]byte, error) {
			return []byte(`{"score":0.02,"categories":[]}`), nil
		},
	})
	nsfw, _, err := svc.ScanNSFW(context.Background(), []byte("generated"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if nsfw {
		t.Error("expected clear output")
	}
}

// TestAutoScan_DegradesWhenUnavailable proves an unavailable classifier does NOT
// fail the generation job — it degrades to "not flagged" with no error.
func TestAutoScan_DegradesWhenUnavailable(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		LookPath:       func(string) (string, error) { return "", errors.New("no python") },
	})
	nsfw, score, err := svc.ScanNSFW(context.Background(), []byte("generated"))
	if err != nil {
		t.Fatalf("scan should degrade gracefully, got: %v", err)
	}
	if nsfw || score != 0 {
		t.Errorf("expected (false,0) on unavailable backend, got (%v,%v)", nsfw, score)
	}
}
