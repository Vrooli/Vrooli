package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildVPSSetupPlanIncludesUploadAndSetup(t *testing.T) {
	// [REQ:STC-P0-004] Install bundle and run setup
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "mini-vrooli.tar.gz")
	if err := os.WriteFile(bundlePath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS: &ManifestVPS{
				Host:    "203.0.113.10",
				Port:    22,
				User:    "root",
				Workdir: "/root/Vrooli",
			},
		},
		Scenario: ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite", "vrooli-autoheal"},
			Resources: []string{"postgres"},
		},
		Bundle: ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	plan, err := BuildVPSSetupPlan(manifest, bundlePath)
	if err != nil {
		t.Fatalf("BuildVPSSetupPlan: %v", err)
	}
	if len(plan) < 7 {
		t.Fatalf("expected at least 7 plan steps (mkdir, bootstrap, upload, extract, setup, autoheal, verify), got: %d", len(plan))
	}

	var hasUpload, hasSetup, hasBootstrap bool
	for _, step := range plan {
		if strings.Contains(step.Command, "scp") {
			hasUpload = true
		}
		if strings.Contains(step.Command, "./scripts/manage.sh setup") {
			hasSetup = true
		}
		if step.ID == "bootstrap" && strings.Contains(step.Command, "apt-get") {
			hasBootstrap = true
		}
	}
	if !hasUpload || !hasSetup {
		t.Fatalf("expected upload and setup steps, got: %+v", plan)
	}
	if !hasBootstrap {
		t.Fatalf("expected bootstrap step with apt-get command, got: %+v", plan)
	}
}

