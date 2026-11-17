package companion

import (
	"bytes"
	"context"
	"errors"
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
