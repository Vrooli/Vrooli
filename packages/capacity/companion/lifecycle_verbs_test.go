package companion

import (
	"context"
	"errors"
	"reflect"
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
