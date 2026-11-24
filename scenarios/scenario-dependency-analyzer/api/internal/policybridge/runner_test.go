package policybridge

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecyclePublicationIsOptIn(t *testing.T) {
	t.Setenv(PublishEnabledEnv, "false")
	var output bytes.Buffer
	stop := Start(context.Background(), log.New(&output, "", 0))
	stop()
	if !strings.Contains(output.String(), "publication withheld") {
		t.Fatalf("disabled publication log = %q", output.String())
	}
	t.Setenv(PublishEnabledEnv, "true")
	if !PublicationEnabled() {
		t.Fatal("explicit publication enablement was ignored")
	}
}

func TestRunnerPreflightRequiresEvaluatorIdentityAndReviewedActionChange(t *testing.T) {
	t.Setenv(ReviewedEffectsDigestEnv, "")
	data := []byte(`{"provider_id":"scenario-dependency-analyzer","rules":[{"evaluator_id":"scenario-dependency-analyzer:dependency-addition","risk":"dependency_addition","action":"route"}]}`)
	for _, tc := range []struct {
		name   string
		staged string
		live   string
		want   string
	}{
		{name: "runner without identity", staged: `{"action":"route"}`, live: `{"action":"allow"}`, want: "did not preserve"},
		{name: "unreviewed action change", staged: `{"evaluator_id":"scenario-dependency-analyzer:dependency-addition","action":"route"}`, live: `{"action":"allow"}`, want: "review this exact policy effect digest"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary := fakeRunner(t, tc.staged, tc.live)
			err := (RunnerSink{Binary: binary}).preflight(context.Background(), binary, data)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("preflight error = %v, want substring %q", err, tc.want)
			}
		})
	}

	t.Run("reviewed action change", func(t *testing.T) {
		digest, err := EffectsDigest(data)
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv(ReviewedEffectsDigestEnv, digest)
		binary := fakeRunner(t, `{"evaluator_id":"scenario-dependency-analyzer:dependency-addition","action":"route"}`, `{"action":"allow"}`)
		if err := (RunnerSink{Binary: binary}).preflight(context.Background(), binary, data); err != nil {
			t.Fatalf("preflight with reviewed action change: %v", err)
		}
	})
}

func TestEffectsDigestIgnoresRefreshTimesAndBindsPolicyChanges(t *testing.T) {
	first := []byte(`{"provider_id":"sda","captured_at":"2026-01-01T00:00:00Z","rules":[{"evaluator_id":"sda:add","risk":"dependency_addition","action":"route","reason":"route"}]}`)
	refreshed := []byte(`{"provider_id":"sda","captured_at":"2026-02-01T00:00:00Z","rules":[{"evaluator_id":"sda:add","risk":"dependency_addition","action":"route","reason":"route"}]}`)
	changed := []byte(`{"provider_id":"sda","captured_at":"2026-02-01T00:00:00Z","rules":[{"evaluator_id":"sda:add","risk":"dependency_addition","action":"deny","reason":"route"}]}`)
	one, err := EffectsDigest(first)
	if err != nil {
		t.Fatal(err)
	}
	two, err := EffectsDigest(refreshed)
	if err != nil {
		t.Fatal(err)
	}
	three, err := EffectsDigest(changed)
	if err != nil {
		t.Fatal(err)
	}
	if one != two || one == three {
		t.Fatalf("effect digests first=%s refreshed=%s changed=%s", one, two, three)
	}
}

func TestEventForRiskUsesRecognizedNonMutatingRepresentatives(t *testing.T) {
	for risk := range map[string]bool{
		"dependency_addition": true, "dependency_upgrade": true, "dependency_removal": true,
		"lifecycle_execution": true, "unknown": true,
	} {
		if _, err := eventForRisk(risk); err != nil {
			t.Errorf("eventForRisk(%q): %v", risk, err)
		}
	}
	if _, err := eventForRisk("publish"); err == nil {
		t.Fatal("unsupported policy risk was accepted")
	}
}

func fakeRunner(t *testing.T, staged, live string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runner")
	script := "#!/bin/sh\nif [ \"$1\" = \"canary\" ]; then\n  case \" $* \" in\n    *--snapshot-dir*) printf '%s\\n' '" + staged + "' ;;\n    *) printf '%s\\n' '" + live + "' ;;\n  esac\nfi\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
