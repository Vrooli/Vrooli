package main

import (
	"os"
	"path/filepath"
	"testing"

	"git-control-tower/cli/domains"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	if got := app.core.APIPath("health"); got != "/api/v1/health" {
		t.Fatalf("expected /api/v1/health, got %q", got)
	}
	if got := app.core.APIPath(" "); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestPrimitiveEvidence(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	cliapptest.RequirePrimitiveEvidence(t, filepath.Join("..", cliapp.EvidenceArtifactRelPath), cliapp.EvidenceExportInput{Scenario: "git-control-tower", ManifestRaw: manifestBytes, Groups: groups}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
