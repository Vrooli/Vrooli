package modelregistry

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"agent-manager/internal/testutil/mocks"
)

func TestHealthStore_MarkAndSnapshot(t *testing.T) {
	hs := NewHealthStore()
	hs.RegisterRunners([]string{"codex", "claude-code"})

	hs.Mark("codex", "gpt-5.2-codex", ModelHealthFailed, "unknown model")
	hs.Mark("claude-code", "opus", ModelHealthOK, "")

	snap := hs.Snapshot()
	if _, ok := snap.Runners["codex"]; !ok {
		t.Fatal("expected codex in snapshot")
	}
	entry, ok := snap.Runners["codex"]["gpt-5.2-codex"]
	if !ok {
		t.Fatal("expected codex/gpt-5.2-codex entry")
	}
	if entry.Status != ModelHealthFailed || entry.Message != "unknown model" {
		t.Fatalf("unexpected codex entry: %+v", entry)
	}
	if opus := snap.Runners["claude-code"]["opus"]; opus.Status != ModelHealthOK {
		t.Fatalf("expected claude-code/opus to be ok, got %+v", opus)
	}
}

func TestHealthProbe_RunOnce_MarksPerModel(t *testing.T) {
	reg := &Registry{
		Version: 1,
		Runners: map[string]RunnerModelRegistry{
			"codex": {
				Models: []ModelOption{{ID: "gpt-5"}, {ID: "gpt-4"}},
				Presets: map[string]PresetChain{
					"SMART": {"gpt-5", "gpt-4"},
					"FAST":  {"gpt-4"},
					"CHEAP": {"gpt-4"},
				},
			},
			"claude-code": {
				Models: []ModelOption{{ID: "opus"}},
				Presets: map[string]PresetChain{
					"SMART": {"opus"},
					"FAST":  {"opus"},
					"CHEAP": {"opus"},
				},
			},
		},
	}
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := Save(path, reg); err != nil {
		t.Fatalf("save registry: %v", err)
	}
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	health := NewHealthStore()
	health.RegisterRunners([]string{"codex", "claude-code"})

	resolve := func(runnerType string) ModelProber {
		if runnerType == "codex" {
			return mocks.NewFailingModelProber(errors.New("unknown model"))
		}
		return mocks.NewFakeModelProber()
	}

	probe := NewHealthProbe(store, health, resolve, ProbeConfig{Interval: 0})
	probe.RunOnce(context.Background())

	snap := health.Snapshot()
	if snap.Runners["codex"]["gpt-5"].Status != ModelHealthFailed {
		t.Fatalf("expected gpt-5 failed, got %+v", snap.Runners["codex"]["gpt-5"])
	}
	if snap.Runners["codex"]["gpt-4"].Status != ModelHealthFailed {
		t.Fatalf("expected gpt-4 failed, got %+v", snap.Runners["codex"]["gpt-4"])
	}
	if snap.Runners["claude-code"]["opus"].Status != ModelHealthOK {
		t.Fatalf("expected opus ok, got %+v", snap.Runners["claude-code"]["opus"])
	}
}

func TestHealthProbe_UnregisteredRunnerIsSkipped(t *testing.T) {
	reg := &Registry{
		Version: 1,
		Runners: map[string]RunnerModelRegistry{
			"codex": {
				Models: []ModelOption{{ID: "gpt-5"}},
				Presets: map[string]PresetChain{
					"SMART": {"gpt-5"},
					"FAST":  {"gpt-5"},
					"CHEAP": {"gpt-5"},
				},
			},
		},
	}
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := Save(path, reg); err != nil {
		t.Fatalf("save: %v", err)
	}
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	health := NewHealthStore()
	health.RegisterRunners([]string{"codex"})

	resolve := func(_ string) ModelProber { return nil }
	NewHealthProbe(store, health, resolve, ProbeConfig{Interval: 0}).RunOnce(context.Background())

	snap := health.Snapshot()
	if len(snap.Runners["codex"]) != 0 {
		t.Fatalf("expected no entries for unregistered prober, got %+v", snap.Runners["codex"])
	}
}
