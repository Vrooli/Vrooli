package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"scenario-to-desktop-api/state"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

// desktopEvidenceFixtureHandler creates a small, deterministic artifact set
// exclusively for a BAS request that already carries the routed test-mode
// context. It gives the browser journey a real mutating success path without
// ever allowing a test fixture to write to an operator's desktop state.
func (s *Server) desktopEvidenceFixtureHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !database.IsTestMode(r.Context()) || s.fileRoots == nil || !s.fileRoots.HasTestRoots() || s.stateService == nil {
		http.NotFound(w, r)
		return
	}

	dataRoot, err := s.fileRoots.Pick(r.Context(), storage.ClassData)
	if err != nil {
		http.Error(w, "resolve leased desktop-evidence root: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fixtureRoot := filepath.Join(dataRoot, "desktop-evidence-fixture")
	if err := os.MkdirAll(fixtureRoot, 0o755); err != nil {
		http.Error(w, "create leased desktop-evidence root: "+err.Error(), http.StatusInternalServerError)
		return
	}
	artifactPath := filepath.Join(fixtureRoot, "fixture-desktop.AppImage")
	if err := os.WriteFile(artifactPath, []byte("Vrooli BAS leased desktop artifact\n"), 0o644); err != nil {
		http.Error(w, "write leased desktop artifact: "+err.Error(), http.StatusInternalServerError)
		return
	}
	reportPath := filepath.Join(fixtureRoot, "fixture-smoke-report.json")
	report := []byte(`{"status":"passed","process_state":"stopped_cleanly","health":"healthy","metrics":{"fixture":true}}` + "\n")
	if err := os.WriteFile(reportPath, report, 0o644); err != nil {
		http.Error(w, "write leased smoke report: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.fileRoots.RecordWrite(r.Context())
	s.fileRoots.RecordWrite(r.Context())

	now := time.Now().UTC()
	_, err = s.stateService.SaveState(r.Context(), "scenario-to-desktop", state.SaveStateRequest{
		FormState: state.FormState{
			SelectedTemplate: "bas-evidence-fixture",
			Framework:        "electron",
			AppDisplayName:   "BAS Desktop Evidence Fixture",
			DeploymentMode:   "bundled",
			LocationMode:     "staging",
			Platforms:        state.PlatformSelection{Linux: true},
		},
		BuildArtifacts: []state.BuildArtifact{{
			Platform: "linux", Status: "ready", FilePath: artifactPath,
			FileName: filepath.Base(artifactPath), FileSize: int64(len("Vrooli BAS leased desktop artifact\n")),
			BuildID: "bas-leased-fixture", BuiltAt: &now,
		}},
		StageResults: map[string]json.RawMessage{
			state.StageGenerate:  json.RawMessage(`{"status":"passed","artifact":"fixture-desktop.AppImage"}`),
			state.StageBuild:     json.RawMessage(`{"status":"passed","artifact":"fixture-desktop.AppImage"}`),
			state.StageSmokeTest: json.RawMessage(`{"status":"passed","report":"fixture-smoke-report.json","process_state":"stopped_cleanly"}`),
		},
	})
	if err != nil {
		http.Error(w, "persist leased desktop evidence: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "passed", "artifact": filepath.Base(artifactPath), "smoke_report": filepath.Base(reportPath),
		"test_root_writes": s.fileRoots.LeaseStats().TestRootWrites,
	})
}
