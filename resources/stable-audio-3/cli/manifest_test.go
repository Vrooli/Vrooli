package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	resourceManifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestResourceUsesCanonicalCLIAndExactOfflineDataClosure(t *testing.T) {
	if _, err := newApp(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../resource.json")
	if err != nil {
		t.Fatal(err)
	}
	var resource struct {
		Managed struct {
			Environment map[string]string `json:"environment"`
			Artifacts   []struct {
				Path        string `json:"path"`
				Acquisition struct {
					Targets []struct {
						URL  string `json:"url"`
						SHA  string `json:"sha256"`
						Auth struct {
							Scheme    string `json:"scheme"`
							LogicalID string `json:"logical_id"`
							Field     string `json:"field"`
						} `json:"auth"`
					} `json:"targets"`
				} `json:"acquisition"`
			} `json:"data_artifacts"`
		} `json:"managed_service"`
		Acceleration struct {
			Claim struct {
				Preferred int64 `json:"default_preferred_bytes"`
				Floor     int64 `json:"floor_bytes"`
			} `json:"claim"`
		} `json:"acceleration"`
	}
	if err := json.Unmarshal(data, &resource); err != nil {
		t.Fatal(err)
	}
	lockData, err := os.ReadFile("../server/model-lock.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(lockData)); got != "dfa4287e6957673fa8c7d21224f88cd7bcff83585e2f1af3535c3fe758f3991d" {
		t.Fatal("reviewed12-file model/support closure changed", got)
	}
	var lock struct {
		Files []struct{ Path, SHA256 string } `json:"files"`
	}
	if err := json.Unmarshal(lockData, &lock); err != nil {
		t.Fatal(err)
	}
	if len(resource.Managed.Artifacts) != 12 || len(lock.Files) != 12 {
		t.Fatal("complete model/support/notice closure required")
	}
	for _, item := range lock.Files {
		if !hex64.MatchString(item.SHA256) || item.SHA256 == strings.Repeat("0", 64) {
			t.Error("all reviewed model/support files must be pinned", item.Path)
		}
		found := 0
		for _, declared := range resource.Managed.Artifacts {
			if declared.Path == "models/medium/"+item.Path {
				found++
				if len(declared.Acquisition.Targets) != 1 {
					t.Error("exactly one pinned data target required", item.Path)
					continue
				}
				target := declared.Acquisition.Targets[0]
				if target.SHA != item.SHA256 || !strings.Contains(target.URL, "/resolve/"+modelRevision+"/"+item.Path) {
					t.Error("model closure identities differ", item.Path)
				}
				if target.Auth.Scheme != "bearer" || target.Auth.LogicalID != "vrooli/stable-audio-3" || target.Auth.Field != "huggingface-token" {
					t.Error("gated data must use the typed credential reference", item.Path)
				}
			}
		}
		if found != 1 {
			t.Error("model input must have exactly one declaration", item.Path)
		}
	}
	for _, key := range []string{"HF_HUB_OFFLINE", "TRANSFORMERS_OFFLINE", "PYTHONDONTWRITEBYTECODE", "PYTHONNOUSERSITE"} {
		if resource.Managed.Environment[key] != "1" {
			t.Error("missing offline/immutable runtime setting", key)
		}
	}
	if resource.Managed.Environment["ENABLE_TORCH_COMPILE"] != "0" {
		t.Error("compile profile differs")
	}
	if resource.Acceleration.Claim.Preferred != 9<<30 || resource.Acceleration.Claim.Floor != 9<<30 {
		t.Error("fixed estimated broker claim differs from resource contract")
	}
}

func TestPilotAccelerationProfileSatisfiesCanonicalOwnerContract(t *testing.T) {
	data, err := os.ReadFile("../resource.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Acceleration resourceManifest.AccelerationSpec `json:"acceleration"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if err := raw.Acceleration.Validate(); err != nil {
		t.Fatal(err)
	}
	claim := raw.Acceleration.Claim
	if claim == nil || claim.Profile == nil || len(claim.Profile.Steps) != 1 {
		t.Fatal("one fixed pilot rung required")
	}
	step := claim.Profile.Steps[0]
	if step.Label != "medium" || step.AmountBytes != 9<<30 || claim.FloorBytes != step.AmountBytes || claim.PreferredBytes != step.AmountBytes || claim.Confidence != "estimated" || !claim.YieldWhenIdle {
		t.Fatalf("wrong pilot profile: %+v", claim)
	}
	if claim.Profile.Apply.Verb != "capacity" || fmt.Sprint(claim.Profile.Apply.Argv) != "[degrade --to {label}]" || claim.Profile.Upshift {
		t.Fatal("fixed profile must expose canonical apply without invented higher/lower rungs")
	}
}

func TestPilotCapacityCommandUsesOnlyFixedProfileAndCanonicalLifecycle(t *testing.T) {
	calls := 0
	group := pilotCapacityCommands(func(_ context.Context, env []string, command string, args ...string) error {
		calls++
		if len(env) != 0 || command != "vrooli" || fmt.Sprint(args) != "[resource restart stable-audio-3]" {
			t.Fatalf("noncanonical lifecycle: env=%v %s %v", env, command, args)
		}
		return nil
	})
	for _, command := range group.Subcommands {
		if err := command.Run([]string{"--to", "medium"}); err != nil {
			t.Fatal(err)
		}
		before := calls
		if err := command.Run([]string{"--to", "cpu"}); err == nil || calls != before {
			t.Fatal("unsupported profile must not invoke lifecycle")
		}
	}
	if calls != 2 {
		t.Fatalf("canonical degrade/upshift calls=%d; want2", calls)
	}
}
