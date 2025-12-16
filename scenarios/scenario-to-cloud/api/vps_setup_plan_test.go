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
		Ports:  ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	plan, err := BuildVPSSetupPlan(manifest, bundlePath)
	if err != nil {
		t.Fatalf("BuildVPSSetupPlan: %v", err)
	}
	if len(plan) < 5 {
		t.Fatalf("expected plan steps, got: %+v", plan)
	}

	var hasUpload, hasSetup bool
	for _, step := range plan {
		if strings.Contains(step.Command, "scp") {
			hasUpload = true
		}
		if strings.Contains(step.Command, "./scripts/manage.sh setup") {
			hasSetup = true
		}
	}
	if !hasUpload || !hasSetup {
		t.Fatalf("expected upload and setup steps, got: %+v", plan)
	}
}

