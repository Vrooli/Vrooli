package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Whisper exposes a true unloaded rung only with a required stable activity
// edge, so a stopped model retains its request and wake owner.
func TestWhisperUnloadRungKeepsStableWakeOwner(t *testing.T) {
	root := findRepoRoot(t)
	path := filepath.Join(root, "resources", "whisper", "resource.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Acceleration struct {
			Claim struct {
				Profile struct {
					Model struct {
						Name   string `json:"name"`
						Digest string `json:"digest"`
					} `json:"model"`
					Steps []struct {
						Label       string `json:"label"`
						AmountBytes int64  `json:"default_amount_bytes"`
					} `json:"steps"`
				} `json:"profile"`
			} `json:"claim"`
		} `json:"acceleration"`
		Companions []struct {
			Name     string `json:"name"`
			Required bool   `json:"required"`
		} `json:"companions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	labels := make([]string, 0, len(doc.Acceleration.Claim.Profile.Steps))
	unloadedBytes := int64(-1)
	for _, step := range doc.Acceleration.Claim.Profile.Steps {
		labels = append(labels, step.Label)
		if step.Label == "unloaded" {
			unloadedBytes = step.AmountBytes
		}
	}
	if len(labels) != 3 || labels[0] != "vulkan" || labels[1] != "cpu" || labels[2] != "unloaded" || unloadedBytes != 0 {
		t.Fatalf("Whisper capacity steps = %v with unloaded amount %d, want [vulkan cpu unloaded@0]", labels, unloadedBytes)
	}
	wakeBoundary := false
	for _, companion := range doc.Companions {
		if companion.Name == "activity-edge" && companion.Required {
			wakeBoundary = true
		}
	}
	if !wakeBoundary {
		t.Fatal("Whisper must retain its required activity-edge request boundary")
	}
	if got := doc.Acceleration.Claim.Profile.Model; got.Name != "ggml-medium.bin" || got.Digest != "6c14d5adee5f86394037b4e4e8b59f1673b6cee10e3cf0b11bbdbee79c156208" {
		t.Fatalf("Whisper claim model = %+v, want its exact pinned medium artifact", got)
	}
	loaded, err := manifest.Load(path)
	if err != nil {
		t.Fatalf("load whisper manifest: %v", err)
	}
	cpu, ok := loaded.EffectiveAcceleration().Config(manifest.BackendCPU)
	if !ok || len(cpu.Arguments) != 1 || cpu.Arguments[0] != "--no-gpu" {
		t.Fatalf("Whisper CPU arguments = %v (present=%t), want [--no-gpu]", cpu.Arguments, ok)
	}
}
