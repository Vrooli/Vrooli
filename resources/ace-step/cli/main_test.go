package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapacityRungLabelsStayAlignedWithResourceContract(t *testing.T) {
	steps := capacitySteps()
	if len(steps) != 2 || steps[0].Label != "full" || steps[1].Label != "offload-dit" {
		t.Fatalf("capacity steps = %+v", steps)
	}
}

func TestPinnedModelInputsHaveOneExecutableCodeAuthority(t *testing.T) {
	data, err := os.ReadFile("../resource.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ManagedService struct {
			DataArtifacts []struct {
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"data_artifacts"`
			Environment map[string]string `json:"environment"`
		} `json:"managed_service"`
		Lifecycle struct {
			StartTimeoutSeconds int `json:"start_timeout_seconds"`
		} `json:"lifecycle"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range manifest.ManagedService.DataArtifacts {
		if strings.EqualFold(filepath.Ext(artifact.Path), ".py") {
			t.Errorf("%s declares executable Python as model data; pinned source must own checkpoint code", artifact.Name)
		}
	}
	if got := len(manifest.ManagedService.DataArtifacts); got != 32 {
		t.Errorf("model data artifacts = %d, want 32 complete pinned inputs", got)
	}
	if manifest.ManagedService.Environment["HF_HUB_OFFLINE"] != "1" {
		t.Error("model loading must remain offline")
	}
	if got := manifest.ManagedService.Environment["PYTHONPATH"]; !strings.Contains(got, "${RESOURCE_ARTIFACT_DIR}/source:") {
		t.Errorf("source package is missing from PYTHONPATH: %q", got)
	}
	if got := manifest.Lifecycle.StartTimeoutSeconds; got != 600 {
		t.Errorf("actual lifecycle startup budget = %d, want 600 seconds", got)
	}

	// These identities protect the complete reviewed closure, not just its size.
	// An intentional pin change requires updating this contract and its review.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	service, ok := raw["managed_service"].(map[string]any)
	if !ok {
		t.Fatal("managed_service must be an object")
	}
	for key, want := range map[string]string{
		"data_artifacts": "2d18a2ab98babb4b19f6de49cbca38f08526dbe184e89c3b3d1f69dec083a7e0",
		"acquisition":    "cb39e4f59c2f1e66e3227f4725841e58afbe673a653a8d2f400208c1f8e61391",
		"artifact":       "2a57cb66dfaf71b80cdc2eacffebdabce2d1ec033275d663f70ec6c83cace758",
	} {
		canonical, err := json.Marshal(service[key])
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(canonical)); got != want {
			t.Errorf("%s pinned closure changed: got %s, want %s", key, got, want)
		}
	}
	lock, err := os.ReadFile("../requirements.lock")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(lock)); got != "0346acdd966b77c08f450ac1f1d84a2bc5b9c135d21d44f8beb71efd29222497" {
		t.Errorf("reviewed wheel lock changed: %s", got)
	}
}

// A weight alone does not make an advertised profile loadable offline.
func TestAdvertisedProfilesDeclareCompleteCheckpointFiles(t *testing.T) {
	data, err := os.ReadFile("../resource.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ManagedService struct {
			DataArtifacts []struct {
				Path        string `json:"path"`
				Acquisition struct {
					Targets []struct {
						URL    string `json:"url"`
						SHA256 string `json:"sha256"`
					} `json:"targets"`
				} `json:"acquisition"`
			} `json:"data_artifacts"`
		} `json:"managed_service"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	declared := map[string]int{}
	for index, artifact := range manifest.ManagedService.DataArtifacts {
		if _, exists := declared[artifact.Path]; exists {
			t.Errorf("duplicate artifact path %s", artifact.Path)
		}
		declared[artifact.Path] = index
	}
	for _, profile := range modelProfiles {
		t.Run(profile.ID, func(t *testing.T) {
			required := map[string][]string{
				profile.Variant: {"model.safetensors", "config.json", "silence_latent.pt"},
				profile.Planner: {"model.safetensors", "config.json", "tokenizer.json", "tokenizer_config.json", "chat_template.jinja", "added_tokens.json", "merges.txt", "special_tokens_map.json", "vocab.json"},
			}
			for model, files := range required {
				for _, file := range files {
					path := "checkpoints/" + model + "/" + file
					index, exists := declared[path]
					if !exists {
						t.Errorf("advertised %s profile lacks declared offline input %s", profile.ID, path)
						continue
					}
					artifact := manifest.ManagedService.DataArtifacts[index]
					if len(artifact.Acquisition.Targets) != 1 {
						t.Errorf("%s requires one exact reviewed target", path)
						continue
					}
					target := artifact.Acquisition.Targets[0]
					if !strings.Contains(target.URL, "/"+model+"/") || !strings.HasSuffix(target.URL, "/"+file) || !strings.Contains(target.URL, "/resolve/") || len(target.SHA256) != 64 {
						t.Errorf("%s target does not bind its own pinned model file: %+v", path, target)
					}
				}
			}
		})
	}
}
