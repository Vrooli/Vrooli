package capabilities

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestScenarioChecker_Healthy(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"scenarios":[{"name":"audio-tools","status":"healthy"}]}`), nil
		},
	}
	status, msg := c.Check(context.Background())
	if status != StatusAvailable {
		t.Fatalf("status = %q, want %q (msg=%q)", status, StatusAvailable, msg)
	}
}

func TestScenarioChecker_Stopped(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"scenarios":[{"name":"audio-tools","status":"stopped"}]}`), nil
		},
	}
	status, msg := c.Check(context.Background())
	if status != StatusUnavailable {
		t.Fatalf("status = %q, want %q (msg=%q)", status, StatusUnavailable, msg)
	}
}

func TestScenarioChecker_CLIMissing(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("exec: vrooli: not found")
		},
	}
	status, msg := c.Check(context.Background())
	if status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg == "" {
		t.Errorf("expected non-empty hint message when CLI is missing")
	}
}

func TestScenarioChecker_NoSlug(t *testing.T) {
	c := &ScenarioChecker{}
	status, _ := c.Check(context.Background())
	if status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", status, StatusUnavailable)
	}
}

func TestScenarioChecker_UnknownStatus(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"scenarios":[{"name":"audio-tools","status":"weird"}]}`), nil
		},
		Timeout: 100 * time.Millisecond,
	}
	status, _ := c.Check(context.Background())
	if status != StatusUnknown {
		t.Fatalf("status = %q, want %q", status, StatusUnknown)
	}
}
