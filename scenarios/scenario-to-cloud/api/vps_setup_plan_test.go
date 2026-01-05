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

func TestBuildVPSSetupPlanRequiresBundlePath(t *testing.T) {
	// [REQ:STC-P0-004] Install bundle and run setup - error case
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
		Scenario: ManifestScenario{ID: "test-scenario"},
	}

	_, err := BuildVPSSetupPlan(manifest, "")
	if err == nil {
		t.Fatal("expected error for empty bundle_path")
	}
	if !strings.Contains(err.Error(), "bundle_path") {
		t.Errorf("expected error about bundle_path, got: %v", err)
	}

	// Also test whitespace-only path
	_, err = BuildVPSSetupPlan(manifest, "   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only bundle_path")
	}
}

func TestBuildVPSSetupPlanStepOrder(t *testing.T) {
	// [REQ:STC-P0-004] Plan steps must execute in the correct order
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
		Scenario: ManifestScenario{ID: "test-scenario"},
	}

	plan, err := BuildVPSSetupPlan(manifest, bundlePath)
	if err != nil {
		t.Fatalf("BuildVPSSetupPlan: %v", err)
	}

	expectedOrder := []string{"mkdir", "bootstrap", "upload", "extract", "setup", "autoheal", "verify"}
	if len(plan) != len(expectedOrder) {
		t.Fatalf("expected %d steps, got %d", len(expectedOrder), len(plan))
	}

	for i, step := range plan {
		if step.ID != expectedOrder[i] {
			t.Errorf("step %d: expected ID %q, got %q", i, expectedOrder[i], step.ID)
		}
		if step.Title == "" {
			t.Errorf("step %d (%s): missing title", i, step.ID)
		}
		if step.Description == "" {
			t.Errorf("step %d (%s): missing description", i, step.ID)
		}
		if step.Command == "" {
			t.Errorf("step %d (%s): missing command", i, step.ID)
		}
	}
}

func TestBuildVPSSetupPlanSSHConfig(t *testing.T) {
	// [REQ:STC-P0-004] Plan should use correct SSH configuration from manifest
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.tar.gz")
	if err := os.WriteFile(bundlePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	tests := []struct {
		name        string
		host        string
		port        int
		user        string
		wantHostRef string
		wantUserRef string
	}{
		{
			name:        "standard config",
			host:        "192.168.1.100",
			port:        22,
			user:        "root",
			wantHostRef: "192.168.1.100",
			wantUserRef: "root@",
		},
		{
			name:        "custom port",
			host:        "example.com",
			port:        2222,
			user:        "deploy",
			wantHostRef: "example.com",
			wantUserRef: "deploy@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := CloudManifest{
				Version: "1.0.0",
				Target: ManifestTarget{
					Type: "vps",
					VPS: &ManifestVPS{
						Host:    tt.host,
						Port:    tt.port,
						User:    tt.user,
						Workdir: "/opt/vrooli",
					},
				},
				Scenario: ManifestScenario{ID: "test"},
			}

			plan, err := BuildVPSSetupPlan(manifest, bundlePath)
			if err != nil {
				t.Fatalf("BuildVPSSetupPlan: %v", err)
			}

			// Check that SSH commands reference the correct host
			for _, step := range plan {
				if strings.Contains(step.Command, "ssh") {
					if !strings.Contains(step.Command, tt.wantHostRef) {
						t.Errorf("step %s: expected host %q in command: %s", step.ID, tt.wantHostRef, step.Command)
					}
					if !strings.Contains(step.Command, tt.wantUserRef) {
						t.Errorf("step %s: expected user %q in command: %s", step.ID, tt.wantUserRef, step.Command)
					}
				}
			}
		})
	}
}

func TestBuildVPSSetupPlanAutohealConfig(t *testing.T) {
	// [REQ:STC-P0-010] Autoheal scope configured for mini-Vrooli
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle.tar.gz")
	if err := os.WriteFile(bundlePath, []byte("test"), 0o644); err != nil {
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
		Scenario: ManifestScenario{ID: "my-app"},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"my-app", "vrooli-autoheal"},
			Resources: []string{"postgres", "redis"},
		},
	}

	plan, err := BuildVPSSetupPlan(manifest, bundlePath)
	if err != nil {
		t.Fatalf("BuildVPSSetupPlan: %v", err)
	}

	// Find autoheal step
	var autohealStep *VPSPlanStep
	for i, step := range plan {
		if step.ID == "autoheal" {
			autohealStep = &plan[i]
			break
		}
	}

	if autohealStep == nil {
		t.Fatal("expected autoheal step in plan")
	}

	// Verify step writes to the correct path
	if !strings.Contains(autohealStep.Command, "autoheal-scope.json") {
		t.Errorf("autoheal step should write to autoheal-scope.json: %s", autohealStep.Command)
	}

	// Verify step creates parent directory
	if !strings.Contains(autohealStep.Command, "mkdir -p") {
		t.Errorf("autoheal step should create parent directory: %s", autohealStep.Command)
	}
}
