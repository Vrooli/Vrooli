package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func profileFixture(t *testing.T) *profileHandlers {
	t.Helper()
	root := t.TempDir()
	return &profileHandlers{GetEnv: func(k string) string {
		if k == "RESOURCE_DATA_DIR" {
			return root
		}
		return ""
	}, Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Info: func(context.Context) (map[string]any, error) {
		return map[string]any{"model_loaded": true, "error": "", "loaded_variant": "acestep-v15-turbo", "loaded_planner": "acestep-5Hz-lm-0.6B"}, nil
	}}
}

func TestFixedProfilesPersistDesiredWithoutRelabelingLoadedOrRestart(t *testing.T) {
	h := profileFixture(t)
	before, err := h.desired()
	if err != nil || before.ID != "fast" {
		t.Fatalf("default: %+v %v", before, err)
	}
	restarts := 0
	h.Restart = func(context.Context) error { restarts++; return nil }
	if err := h.Activate([]string{"--profile", "balanced", "--no-restart", "--json"}); err != nil {
		t.Fatal(err)
	}
	if restarts != 0 {
		t.Fatal("desired-only selection restarted service")
	}
	data, err := os.ReadFile(h.statePath())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ACE_STEP_VARIANT=acestep-v15-turbo\nACE_STEP_PLANNER=acestep-5Hz-lm-1.7B\n" {
		t.Fatalf("unexpected state %s", data)
	}
	st, _ := os.Stat(h.statePath())
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("state mode %v", st.Mode())
	}
	h.Out.(*bytes.Buffer).Reset()
	if err := h.Status([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	var status struct {
		Desired         modelProfile
		Loaded          modelProfile
		RestartRequired bool `json:"restart_required"`
	}
	if err := json.Unmarshal(h.Out.(*bytes.Buffer).Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Desired.ID != "balanced" || status.Loaded.ID != "fast" || !status.RestartRequired {
		t.Fatalf("relabelled loaded profile: %+v", status)
	}
	if err := h.Activate([]string{"--profile", "unknown", "--no-restart"}); err == nil {
		t.Fatal("unknown profile accepted")
	}
	after, _ := os.ReadFile(h.statePath())
	if !bytes.Equal(data, after) {
		t.Fatal("invalid selection mutated state")
	}
}

func TestProfileActivationRequiresActualLoadedAcknowledgment(t *testing.T) {
	h := profileFixture(t)
	calls := 0
	h.Restart = func(context.Context) error { calls++; return nil }
	if err := h.Activate([]string{"--profile", "detail"}); err == nil || !strings.Contains(err.Error(), "did not acknowledge") {
		t.Fatalf("stale loaded model accepted: %v", err)
	}
	if calls != 1 {
		t.Fatalf("restart calls %d", calls)
	}
	h.Info = func(context.Context) (map[string]any, error) {
		return map[string]any{"model_loaded": true, "error": "", "loaded_variant": "acestep-v15-sft", "loaded_planner": "acestep-5Hz-lm-1.7B"}, nil
	}
	if err := h.Activate([]string{"--profile", "detail", "--json"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("restart calls %d", calls)
	}
}

func TestProfileStatusDoesNotInventLoadedIdentityFromOldOrUnavailableAPI(t *testing.T) {
	for _, mode := range []string{"old", "failure", "initializing"} {
		t.Run(mode, func(t *testing.T) {
			h := profileFixture(t)
			h.Info = func(context.Context) (map[string]any, error) {
				if mode == "failure" {
					return nil, errors.New("offline")
				}
				return map[string]any{"model_loaded": mode != "initializing", "variant": "acestep-v15-turbo", "planner": "acestep-5Hz-lm-0.6B"}, nil
			}
			if err := h.Status([]string{"--json"}); err != nil {
				t.Fatal(err)
			}
			var out map[string]any
			if err := json.Unmarshal(h.Out.(*bytes.Buffer).Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			if out["loaded"] != nil || out["restart_required"] != nil || out["observation_error"] == nil {
				t.Fatalf("unknown observation became a fact: %+v", out)
			}
		})
	}
}

func TestProfileStateRejectsIncompleteAndInjectedSettings(t *testing.T) {
	for _, body := range []string{"ACE_STEP_VARIANT=acestep-v15-sft\n", "ACE_STEP_VARIANT=acestep-v15-turbo\nACE_STEP_PLANNER=acestep-5Hz-lm-0.6B\nOTHER=value\n", "ACE_STEP_VARIANT=acestep-v15-turbo\nACE_STEP_VARIANT=acestep-v15-sft\nACE_STEP_PLANNER=acestep-5Hz-lm-1.7B\n"} {
		h := profileFixture(t)
		if err := os.WriteFile(h.statePath(), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := h.desired(); err == nil {
			t.Fatal("invalid persisted profile accepted")
		}
	}
	h := profileFixture(t)
	if err := h.List([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	var out struct{ Profiles []modelProfile }
	if err := json.Unmarshal(h.Out.(*bytes.Buffer).Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Profiles) != 3 || out.Profiles[2].Variant != "acestep-v15-sft" || out.Profiles[2].Planner != "acestep-5Hz-lm-1.7B" {
		t.Fatalf("catalog drift: %+v", out)
	}
	if filepath.Base(h.statePath()) != "active.env" {
		t.Fatal("selection did not use existing environment-file contract")
	}
}
