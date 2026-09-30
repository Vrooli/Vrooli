package handoff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/planworkshop"
)

func TestBuildIdeaPackage_UsesPlanWorkshopResponses(t *testing.T) {
	itemDir := filepath.Join(t.TempDir(), "ideas", "alpha")
	if err := os.MkdirAll(filepath.Join(itemDir, "archive"), 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}

	writeFile(t, filepath.Join(itemDir, "spec.json"), `{"kind":"idea","name":"alpha"}`)
	writeFile(t, filepath.Join(itemDir, "notes.md"), "processing notes")
	dataRoot := filepath.Dir(filepath.Dir(itemDir))
	subject := planworkshop.Subject{Kind: planworkshop.SubjectBacklog, Ref: "idea/alpha"}
	if err := planworkshop.NewStore(dataRoot).Save(planworkshop.Session{
		ID:             planworkshop.WorkshopID(subject),
		Subject:        subject,
		SubjectVersion: "v1",
		Packet: planworkshop.ReviewPacket{Questions: []planworkshop.DecisionQuestion{
			{ID: "d1", Question: "Database", Options: []string{"SQLite", "Postgres"}},
			{ID: "d2", Question: "Deferred"},
		}},
		Responses: []planworkshop.Response{{ID: "response-1", Answers: map[string]string{"d1": "SQLite"}}},
	}); err != nil {
		t.Fatalf("save plan workshop session: %v", err)
	}

	pkg, err := BuildIdeaPackage(BuildRequest{
		BacklogKind:             "idea",
		BacklogName:             "alpha",
		BacklogTitle:            "Alpha",
		BacklogDescription:      "Create alpha.",
		ItemFolder:              itemDir,
		DeliverablePath:         "plan-manager:alpha",
		TargetScenario:          "alpha",
		Operation:               "generator",
		SuggestedSteerProfileID: "rapid-mvp",
		AcceptanceAllow:         []string{"scenarios/alpha/**"},
		AcceptanceDeny:          []string{"scenarios/alpha/secrets/**"},
	})
	if err != nil {
		t.Fatalf("BuildIdeaPackage() error = %v", err)
	}

	if pkg.Dir == "" || pkg.BriefPath == "" || pkg.ManifestPath == "" || pkg.SourceIndexPath == "" {
		t.Fatalf("expected package paths to be populated: %#v", pkg)
	}
	for _, path := range []string{pkg.BriefPath, pkg.ManifestPath, pkg.SourceIndexPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	if !strings.Contains(pkg.BriefMarkdown, "Use this `brief.md` file as supporting execution context") {
		t.Fatalf("brief missing execution-context instruction:\n%s", pkg.BriefMarkdown)
	}
	if !strings.Contains(pkg.BriefMarkdown, "plan-manager:alpha") {
		t.Fatalf("brief missing canonical plan path:\n%s", pkg.BriefMarkdown)
	}
	if !strings.Contains(pkg.BriefMarkdown, "Round 001 `d1`: Database -> SQLite") {
		t.Fatalf("brief missing locked decision summary:\n%s", pkg.BriefMarkdown)
	}
	if !strings.Contains(pkg.BriefMarkdown, "Round 001 `d2`: Deferred") {
		t.Fatalf("brief missing open decision summary:\n%s", pkg.BriefMarkdown)
	}

	var manifest Manifest
	manifestData, err := os.ReadFile(pkg.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if manifest.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %q, want %q", manifest.SchemaVersion, SchemaVersion)
	}
	if manifest.TargetScenario != "alpha" {
		t.Fatalf("target scenario = %q", manifest.TargetScenario)
	}
	if manifest.Operation != "generator" {
		t.Fatalf("operation = %q", manifest.Operation)
	}
	if len(manifest.LockedDecisions) != 1 || manifest.LockedDecisions[0].SelectedLabel != "SQLite" {
		t.Fatalf("locked decisions = %#v", manifest.LockedDecisions)
	}
	if len(manifest.OpenDecisions) != 1 || manifest.OpenDecisions[0].Topic != "Deferred" {
		t.Fatalf("open decisions = %#v", manifest.OpenDecisions)
	}
	if len(manifest.ValidationCommands) != 3 {
		t.Fatalf("validation commands = %#v", manifest.ValidationCommands)
	}
}

func TestBuildIdeaPackage_DoesNotReferenceLegacyRoundPaths(t *testing.T) {
	itemDir := filepath.Join(t.TempDir(), "ideas", "alpha")
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		t.Fatalf("mkdir item: %v", err)
	}
	writeFile(t, filepath.Join(itemDir, "spec.json"), `{"kind":"idea","name":"alpha"}`)

	pkg, err := BuildIdeaPackage(BuildRequest{
		BacklogKind:     "idea",
		BacklogName:     "alpha",
		ItemFolder:      itemDir,
		DeliverablePath: "plan-manager:alpha",
	})
	if err != nil {
		t.Fatalf("BuildIdeaPackage() error = %v", err)
	}
	if strings.Contains(pkg.BriefMarkdown, "round-") || strings.Contains(pkg.BriefMarkdown, "research/summary.md") {
		t.Fatalf("brief references deleted legacy paths:\n%s", pkg.BriefMarkdown)
	}
	var source SourceIndex
	data, err := os.ReadFile(pkg.SourceIndexPath)
	if err != nil {
		t.Fatalf("read source index: %v", err)
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatalf("decode source index: %v", err)
	}
}

func TestBuildIdeaPackage_RejectsNonIdeaItems(t *testing.T) {
	if _, err := BuildIdeaPackage(BuildRequest{BacklogKind: "fix", ItemFolder: t.TempDir()}); err == nil {
		t.Fatal("expected non-idea build to fail")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
