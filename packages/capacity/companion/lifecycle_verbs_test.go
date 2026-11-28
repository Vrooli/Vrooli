package companion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLifecycleCapacityCommandsAppliesGuardBeforeLifecycleAction(t *testing.T) {
	var calls []string
	group := LifecycleCapacityCommands(LifecycleVerbsConfig{
		Resource: "speech",
		Steps:    DeviceSteps("gpu"),
		Apply: func(_ context.Context, label string) error {
			calls = append(calls, "apply:"+label)
			return nil
		},
		Exec: func(_ context.Context, env []string, name string, args ...string) error {
			calls = append(calls, "exec:"+env[0]+":"+name+":"+args[1]+":"+args[2])
			return nil
		},
	})

	if err := group.Subcommands[0].Run([]string{"--to", "cpu"}); err != nil {
		t.Fatalf("degrade error = %v", err)
	}
	want := []string{"apply:cpu", "exec:VROOLI_GPU=off:vrooli:restart:speech"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestLifecycleCapacityCommandsStopsWhenApplyFails(t *testing.T) {
	wantErr := errors.New("pin failed")
	executed := false
	group := LifecycleCapacityCommands(LifecycleVerbsConfig{
		Resource: "ollama",
		Steps:    DeviceSteps("gpu"),
		Apply:    func(context.Context, string) error { return wantErr },
		Exec: func(context.Context, []string, string, ...string) error {
			executed = true
			return nil
		},
	})

	if err := group.Subcommands[0].Run([]string{"--to", "cpu"}); !errors.Is(err, wantErr) {
		t.Fatalf("degrade error = %v, want %v", err, wantErr)
	}
	if executed {
		t.Fatal("lifecycle action ran after apply failure")
	}
}

func TestLifecycleCapacityCommandsDryRunShowsExactOwnerActionWithoutApplying(t *testing.T) {
	var output bytes.Buffer
	executed := false
	group := LifecycleCapacityCommands(LifecycleVerbsConfig{
		Resource: "whisper",
		Steps: []LifecycleStep{{
			Label:       "unloaded",
			Action:      "stop",
			Env:         []string{"VROOLI_CAPACITY_UNLOAD=1"},
			Description: "stop the managed Whisper model server and keep the activity edge",
		}},
		Stdout: &output,
		Exec: func(context.Context, []string, string, ...string) error {
			executed = true
			return nil
		},
	})
	if err := group.Subcommands[0].Run([]string{"--to", "unloaded", "--dry-run"}); err != nil {
		t.Fatalf("dry-run error = %v", err)
	}
	if executed {
		t.Fatal("dry-run invoked the lifecycle action")
	}
	for _, want := range []string{"target rung: unloaded", "vrooli resource stop whisper", "VROOLI_CAPACITY_UNLOAD=1", "keep the activity edge"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("dry-run output %q does not contain %q", output.String(), want)
		}
	}
}

func TestLifecycleCapacityCommandsDryRunRejectsUndeclaredStep(t *testing.T) {
	var output bytes.Buffer
	executed := false
	group := LifecycleCapacityCommands(LifecycleVerbsConfig{
		Resource: "whisper",
		Steps:    []LifecycleStep{{Label: "unloaded", Action: "stop"}},
		Stdout:   &output,
		Exec: func(context.Context, []string, string, ...string) error {
			executed = true
			return nil
		},
	})
	if err := group.Subcommands[0].Run([]string{"--to", "wildcard", "--dry-run"}); err == nil || !strings.Contains(err.Error(), ErrUnknownStep.Error()) {
		t.Fatalf("unknown dry-run step error = %v", err)
	}
	if executed || output.Len() != 0 {
		t.Fatalf("unknown dry-run changed state/output: executed=%t output=%q", executed, output.String())
	}
}

func TestProfileStepsResolvesTheDeclaredLadderWhenAVerbRuns(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"acceleration":{"claim":{"resource_kind":"vram","profile":{"steps":[{"label":"big:9b","default_amount_bytes":9},{"label":"small:4b","default_amount_bytes":4},{"label":"cpu","default_amount_bytes":0}]}}}}`
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == "RESOURCE_ROOT" {
			return dir
		}
		return ""
	}
	resolved := 0
	steps := ProfileSteps(getenv, "demo", func(step DeclaredStep) LifecycleStep {
		env := "VROOLI_GPU=on"
		if step.AmountBytes == 0 {
			env = "VROOLI_GPU=off"
		}
		return LifecycleStep{Action: "restart", Env: []string{env}}
	})
	var calls []string
	var output bytes.Buffer
	group := LifecycleCapacityCommands(LifecycleVerbsConfig{
		Resource: "demo",
		DeclaredSteps: func() ([]LifecycleStep, error) {
			resolved++
			return steps()
		},
		Stdout: &output,
		Exec: func(_ context.Context, env []string, _ string, args ...string) error {
			calls = append(calls, env[0]+" "+strings.Join(args, " "))
			return nil
		},
	})
	if resolved != 0 {
		t.Fatal("building the command group must not read resource.json")
	}
	if err := group.Subcommands[0].Run([]string{"--to", "small:4b"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[0].Run([]string{"--to", "cpu"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"VROOLI_GPU=on resource restart demo", "VROOLI_GPU=off resource restart demo"}
	if !reflect.DeepEqual(calls, want) || resolved != 2 {
		t.Fatalf("calls = %v (resolved %d), want %v", calls, resolved, want)
	}
	if err := group.Subcommands[0].Run([]string{"--to", "undeclared:1b"}); !errors.Is(err, ErrUnknownStep) {
		t.Fatalf("undeclared rung error = %v, want ErrUnknownStep", err)
	}
}

func TestProfileStepsFailsClosedWithoutADeclaredProfile(t *testing.T) {
	group := LifecycleCapacityCommands(LifecycleVerbsConfig{
		Resource:      "demo",
		DeclaredSteps: ProfileSteps(func(string) string { return t.TempDir() + "/resources/demo" }, "absent-resource-for-test", func(DeclaredStep) LifecycleStep { return LifecycleStep{} }),
		Exec: func(context.Context, []string, string, ...string) error {
			t.Fatal("no action without a ladder")
			return nil
		},
	})
	if err := group.Subcommands[0].Run([]string{"--to", "cpu"}); err == nil || !strings.Contains(err.Error(), "declares no acceleration.claim.profile.steps") {
		t.Fatalf("error = %v, want a missing-profile refusal", err)
	}
}
