package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceManifestDeclaresAuditedResidencyAndThreadBudgets(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "resource.json"))
	if err != nil {
		t.Fatalf("read resource manifest: %v", err)
	}
	var manifest struct {
		ManagedService struct {
			Arguments   []string          `json:"arguments"`
			Environment map[string]string `json:"environment"`
		} `json:"managed_service"`
		Requirements struct {
			RAMMB    int `json:"ram_mb"`
			CPUCores int `json:"cpu_cores"`
		} `json:"requirements"`
		Acceleration struct {
			Claim struct {
				IdleGraceSeconds     int `json:"idle_grace_seconds"`
				IdleUnloadTTLSeconds int `json:"idle_unload_ttl_seconds"`
			} `json:"claim"`
		} `json:"acceleration"`
		Ports []struct {
			Name string `json:"name"`
			Host int    `json:"host"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode resource manifest: %v", err)
	}
	if manifest.Requirements.RAMMB != 3072 || manifest.Requirements.CPUCores != 2 {
		t.Fatalf("host budget = %d MiB/%d cores, want audited 3072 MiB/2 cores", manifest.Requirements.RAMMB, manifest.Requirements.CPUCores)
	}
	if manifest.ManagedService.Environment["RERANKER_TOKENIZATION_WORKERS"] != "4" || manifest.ManagedService.Environment["RAYON_NUM_THREADS"] != "4" || manifest.ManagedService.Environment["RERANKER_MAX_CONCURRENT_REQUESTS"] != "2" {
		t.Fatalf("reranker worker limits = %v, want tokenizer/Rayon 4 and concurrency 2", manifest.ManagedService.Environment)
	}
	joinedArgs := strings.Join(manifest.ManagedService.Arguments, " ")
	if !strings.Contains(joinedArgs, "--max-concurrent-requests=${RERANKER_MAX_CONCURRENT_REQUESTS}") {
		t.Fatalf("TEI arguments do not apply the declared concurrency budget: %v", manifest.ManagedService.Arguments)
	}
	if got := manifest.Acceleration.Claim; got.IdleGraceSeconds != 900 || got.IdleUnloadTTLSeconds != 900 {
		t.Fatalf("resource claim idle policy = grace %ds/unload %ds, want shared capacity policy", got.IdleGraceSeconds, got.IdleUnloadTTLSeconds)
	}
	for _, port := range manifest.Ports {
		if port.Name == "upstream" {
			if port.Host != 0 {
				t.Fatalf("backend host port = %d, want unpublished backend behind the activity edge", port.Host)
			}
			return
		}
	}
	t.Fatal("resource manifest has no private upstream port")
}

func TestNewAppConfiguresResourceApp(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	if app == nil {
		t.Fatal("newApp() returned nil app")
	}
	if app.CLI == nil {
		t.Fatal("newApp() returned nil CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("newApp() returned nil stale checker")
	}
	if app.StaleChecker.SourceContextPath != ".." {
		t.Fatalf("SourceContextPath = %q, want %q", app.StaleChecker.SourceContextPath, "..")
	}
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
	if len(app.StaleChecker.FreshnessInputs) < 3 {
		t.Fatalf("FreshnessInputs len = %d, want at least the 3 required inputs", len(app.StaleChecker.FreshnessInputs))
	}
	if got, want := app.StaleChecker.FreshnessInputs[0], "cli/**"; got != want {
		t.Fatalf("FreshnessInputs[0] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[1], "resource.json"; got != want {
		t.Fatalf("FreshnessInputs[1] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[2], "../../packages/cli-core"; got != want {
		t.Fatalf("FreshnessInputs[2] = %q, want %q", got, want)
	}
}
