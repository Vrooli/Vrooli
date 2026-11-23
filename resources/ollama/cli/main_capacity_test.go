package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/vrooli/vrooli/packages/capacity/companion"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

func TestOllamaPrivateAdminCommandsJoinTheOwnerNamespace(t *testing.T) {
	t.Setenv("VROOLI_ACTIVITY_NETWORK_JOINED", "")
	for _, command := range []string{"ensure", "capacity"} {
		if !activityNetworkRequiredForOllamaCommand([]string{command}) {
			t.Errorf("%s must enter the managed daemon's private namespace", command)
		}
	}
	for _, command := range []string{"gateway", "models", "health-ready"} {
		if activityNetworkRequiredForOllamaCommand([]string{command}) {
			t.Errorf("%s must use the public activity edge", command)
		}
	}
	t.Setenv("VROOLI_ACTIVITY_NETWORK_JOINED", "1")
	if activityNetworkRequiredForOllamaCommand([]string{"capacity"}) {
		t.Fatal("already-joined namespace command would re-enter recursively")
	}
}

type fakeCapacityClient struct {
	running   []ensure.RunningModel
	unloaded  []string
	events    *[]string
	listErr   error
	unloadErr error
}

func (f *fakeCapacityClient) ListRunning(ctx context.Context) ([]ensure.RunningModel, error) {
	if f.events != nil {
		*f.events = append(*f.events, "list")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return f.running, f.listErr
}

func (f *fakeCapacityClient) Unload(_ context.Context, name string) error {
	f.unloaded = append(f.unloaded, name)
	if f.events != nil {
		*f.events = append(*f.events, "unload:"+name)
	}
	return f.unloadErr
}

func TestApplyOllamaCapacityRungUnloadsLargestUntilTarget(t *testing.T) {
	client := &fakeCapacityClient{running: []ensure.RunningModel{
		{Name: "small", SizeVRAM: 1 << 30},
		{Name: "large", SizeVRAM: 6 << 30},
		{Name: "medium", SizeVRAM: 3 << 30},
	}}
	target := func(string) (int64, error) { return 4 << 30, nil }

	if err := applyOllamaCapacityRungWith(context.Background(), "small-rung", target, client); err != nil {
		t.Fatalf("apply error = %v", err)
	}
	if want := []string{"large"}; !reflect.DeepEqual(client.unloaded, want) {
		t.Fatalf("unloaded = %v, want %v", client.unloaded, want)
	}
}

func TestApplyOllamaCapacityRungCPUReleasesAllVRAM(t *testing.T) {
	client := &fakeCapacityClient{running: []ensure.RunningModel{
		{Name: "small", SizeVRAM: 1 << 30},
		{Name: "large", SizeVRAM: 6 << 30},
	}}
	target := func(label string) (int64, error) {
		if label != "cpu" {
			t.Fatalf("label = %q, want cpu", label)
		}
		return 0, nil
	}

	if err := applyOllamaCapacityRungWith(context.Background(), "cpu", target, client); err != nil {
		t.Fatalf("apply error = %v", err)
	}
	if want := []string{"large", "small"}; !reflect.DeepEqual(client.unloaded, want) {
		t.Fatalf("unloaded = %v, want %v", client.unloaded, want)
	}
}

func TestOllamaCapacityStepDrainsBeforeUnloadAndRestartsAfter(t *testing.T) {
	var events []string
	client := &fakeCapacityClient{
		running: []ensure.RunningModel{{Name: "small", SizeVRAM: 1 << 30}, {Name: "large", SizeVRAM: 6 << 30}},
		events:  &events,
	}
	control := func(_ context.Context, action string) error {
		events = append(events, "control:"+action)
		return nil
	}
	group := companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{
		Resource: "ollama",
		Steps:    []companion.LifecycleStep{{Label: "cpu", Action: "restart", Env: []string{"VROOLI_GPU=off"}}},
		Apply: func(ctx context.Context, label string) error {
			return applyOllamaCapacityRungWithDrain(ctx, label, ollamaCapacityTargetForTest, client, control)
		},
		Exec: func(_ context.Context, env []string, name string, args ...string) error {
			events = append(events, "restart")
			if !reflect.DeepEqual(env, []string{"VROOLI_GPU=off"}) || name != "vrooli" || !reflect.DeepEqual(args, []string{"resource", "restart", "ollama"}) {
				t.Fatalf("managed restart = env %v, %q %v", env, name, args)
			}
			events = append(events, "health", "placement")
			return control(context.Background(), "resume")
		},
	})
	if err := group.Subcommands[0].Run([]string{"--to", "cpu"}); err != nil {
		t.Fatalf("capacity step: %v", err)
	}
	want := []string{"control:drain", "list", "unload:large", "unload:small", "restart", "health", "placement", "control:resume"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("capacity step order = %v, want %v", events, want)
	}
}

func TestOllamaCapacityStepResumesWhenUnloadFailsBeforeRestart(t *testing.T) {
	wantErr := errors.New("unload interrupted")
	var events []string
	client := &fakeCapacityClient{running: []ensure.RunningModel{{Name: "large", SizeVRAM: 6 << 30}}, events: &events, unloadErr: wantErr}
	control := func(_ context.Context, action string) error {
		events = append(events, "control:"+action)
		return nil
	}
	group := companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{
		Resource: "ollama",
		Steps:    []companion.LifecycleStep{{Label: "cpu", Action: "restart"}},
		Apply: func(ctx context.Context, label string) error {
			return applyOllamaCapacityRungWithDrain(ctx, label, ollamaCapacityTargetForTest, client, control)
		},
		Exec: func(context.Context, []string, string, ...string) error {
			events = append(events, "unexpected-restart")
			return nil
		},
	})
	err := group.Subcommands[0].Run([]string{"--to", "cpu"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("capacity step error = %v, want %v", err, wantErr)
	}
	want := []string{"control:drain", "list", "unload:large", "control:resume"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("failed-unload recovery order = %v, want %v", events, want)
	}
}

func TestOllamaCapacityRestartFailureStaysDrainedAndDemandUsesManagedWake(t *testing.T) {
	wantErr := errors.New("restart failed")
	var events []string
	client := &fakeCapacityClient{running: []ensure.RunningModel{{Name: "small", SizeVRAM: 1 << 30}}, events: &events}
	drained := false
	control := func(_ context.Context, action string) error {
		events = append(events, "control:"+action)
		if action == "drain" {
			drained = true
		}
		if action == "resume" {
			drained = false
		}
		return nil
	}
	group := companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{
		Resource: "ollama",
		Steps:    []companion.LifecycleStep{{Label: "cpu", Action: "restart", Env: []string{"VROOLI_GPU=off"}}},
		Apply: func(ctx context.Context, label string) error {
			return applyOllamaCapacityRungWithDrain(ctx, label, ollamaCapacityTargetForTest, client, control)
		},
		Exec: func(context.Context, []string, string, ...string) error {
			events = append(events, "restart")
			return wantErr
		},
	})
	err := group.Subcommands[0].Run([]string{"--to", "cpu"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("restart failure = %v, want %v", err, wantErr)
	}
	if !drained {
		t.Fatal("restart failure resumed requests against uncertain residency")
	}
	if want := []string{"control:drain", "list", "unload:small", "restart"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("restart-failure order = %v, want %v", events, want)
	}
	var wakeName string
	var wakeArgs []string
	if err := runOllamaWakeWith(context.Background(), func(_ context.Context, name string, args ...string) error {
		wakeName, wakeArgs = name, append([]string(nil), args...)
		return nil
	}); err != nil {
		t.Fatalf("managed demand wake: %v", err)
	}
	if wakeName != "vrooli" || !reflect.DeepEqual(wakeArgs, []string{"resource", "start", "ollama"}) {
		t.Fatalf("wake command = %q %v", wakeName, wakeArgs)
	}
}

func ollamaCapacityTargetForTest(label string) (int64, error) {
	if label != "cpu" {
		return 0, errors.New("unexpected capacity label")
	}
	return 0, nil
}

func TestOllamaCapacityCancellationLeavesActivityEdgeDrained(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var events []string
	control := func(_ context.Context, action string) error {
		events = append(events, "control:"+action)
		if action == "drain" {
			cancel()
		}
		return nil
	}
	client := &fakeCapacityClient{running: []ensure.RunningModel{{Name: "model", SizeVRAM: 1}}, events: &events}
	err := applyOllamaCapacityRungWithDrain(ctx, "cpu", func(string) (int64, error) { return 0, nil }, client, control)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled unload error = %v, want context.Canceled", err)
	}
	if want := []string{"control:drain", "list"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("canceled operation recovery = %v, want %v", events, want)
	}
}

func TestOllamaCapacityClientTargetsDeclaredPrivateUpstream(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "resource.json"))
	if err != nil {
		t.Fatalf("read Ollama resource manifest: %v", err)
	}
	var manifest struct {
		Ports []struct {
			Name      string `json:"name"`
			Host      int    `json:"host"`
			Container int    `json:"container"`
		} `json:"ports"`
		ManagedService struct {
			Environment map[string]string `json:"environment"`
		} `json:"managed_service"`
		Acceleration struct {
			Claim struct {
				IdleGraceSeconds     int `json:"idle_grace_seconds"`
				IdleUnloadTTLSeconds int `json:"idle_unload_ttl_seconds"`
			} `json:"claim"`
		} `json:"acceleration"`
		Requirements struct {
			RAMMB    int `json:"ram_mb"`
			CPUCores int `json:"cpu_cores"`
		} `json:"requirements"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode Ollama resource manifest: %v", err)
	}
	var upstreamPort int
	var upstreamHost int
	for _, port := range manifest.Ports {
		if port.Name == "upstream" {
			upstreamPort = port.Container
			upstreamHost = port.Host
		}
	}
	if want := "http://127.0.0.1:" + strconv.Itoa(upstreamPort); upstreamPort <= 0 || upstreamHost != 0 || ollamaCapacityUpstreamURL != want {
		t.Fatalf("capacity upstream = %q, want private namespace endpoint %q with no host publication (host=%d)", ollamaCapacityUpstreamURL, want, upstreamHost)
	}
	if manifest.ManagedService.Environment["OLLAMA_KEEP_ALIVE"] != "-1" {
		t.Fatalf("OLLAMA_KEEP_ALIVE = %q, want -1 so the shared capacity state machine owns model unload timing", manifest.ManagedService.Environment["OLLAMA_KEEP_ALIVE"])
	}
	if manifest.ManagedService.Environment["OLLAMA_NUM_PARALLEL"] != "1" {
		t.Fatalf("Ollama concurrency = %v, want one request", manifest.ManagedService.Environment)
	}
	// The loaded-model count is derived from the policy's residency, never tuned
	// by hand: residents plus one turnover slot.
	p, err := policy.LoadFile(filepath.Join("..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := manifest.ManagedService.Environment["OLLAMA_MAX_LOADED_MODELS"], strconv.Itoa(p.LoadedModelSlots()); got != want {
		t.Fatalf("OLLAMA_MAX_LOADED_MODELS = %q, want %s (resident models %v plus one turnover slot)", got, want, p.ResidentModels())
	}
	if manifest.Requirements.RAMMB != 8192 || manifest.Requirements.CPUCores != 4 {
		t.Fatalf("Ollama host budget = %d MiB/%d cores, want audited 8192 MiB/4 cores", manifest.Requirements.RAMMB, manifest.Requirements.CPUCores)
	}
	if got := manifest.Acceleration.Claim; got.IdleGraceSeconds != 900 || got.IdleUnloadTTLSeconds != 900 {
		t.Fatalf("resource claim idle policy = grace %ds/unload %ds, want both timers owned by shared capacity", got.IdleGraceSeconds, got.IdleUnloadTTLSeconds)
	}
}

// TestOllamaCapacityLadderIsTheDeclaredClaimProfile keeps resource.json the one
// rung list: every declared rung is accepted with its declared byte target,
// the zero-byte rung leaves the GPU, and nothing else is accepted.
func TestOllamaCapacityLadderIsTheDeclaredClaimProfile(t *testing.T) {
	resourceDir, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(resourceDir, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Acceleration struct {
			Claim struct {
				Profile struct {
					Steps []struct {
						Label  string `json:"label"`
						Amount int64  `json:"default_amount_bytes"`
					} `json:"steps"`
				} `json:"profile"`
			} `json:"claim"`
		} `json:"acceleration"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	declared := manifest.Acceleration.Claim.Profile.Steps
	if len(declared) < 2 {
		t.Fatalf("resource.json declares %d rungs", len(declared))
	}
	getenv := func(key string) string {
		if key == "RESOURCE_ROOT" {
			return resourceDir
		}
		return ""
	}
	steps, err := companion.ProfileSteps(getenv, appName, ollamaCapacityStep)()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := companion.DeclaredProfile(getenv, appName)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != len(declared) {
		t.Fatalf("ladder has %d rungs, resource.json declares %d", len(steps), len(declared))
	}
	for i, want := range declared {
		got := steps[i]
		wantEnv := "VROOLI_GPU=on"
		if want.Amount == 0 {
			wantEnv = "VROOLI_GPU=off"
		}
		if got.Label != want.Label || got.Action != "restart" || len(got.Env) != 1 || got.Env[0] != wantEnv {
			t.Errorf("rung %d = %+v, want %s restart %s", i, got, want.Label, wantEnv)
		}
		if bytes, err := ollamaRungBytes(profile.Steps, want.Label); err != nil || bytes != want.Amount {
			t.Errorf("rung %s target = %d (%v), want %d", want.Label, bytes, err, want.Amount)
		}
	}
	if _, err := ollamaRungBytes(profile.Steps, "llama3.2:1b"); err == nil {
		t.Fatal("an undeclared rung must be refused")
	}
}
