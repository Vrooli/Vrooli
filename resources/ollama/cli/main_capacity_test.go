package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
)

type fakeCapacityClient struct {
	running  []ensure.RunningModel
	unloaded []string
}

func (f *fakeCapacityClient) ListRunning(context.Context) ([]ensure.RunningModel, error) {
	return f.running, nil
}

func (f *fakeCapacityClient) Unload(_ context.Context, name string) error {
	f.unloaded = append(f.unloaded, name)
	return nil
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
