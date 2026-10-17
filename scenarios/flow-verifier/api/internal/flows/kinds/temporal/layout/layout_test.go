package layout

import (
	"strings"
	"testing"
)

func TestDeriveTypeScript(t *testing.T) {
	got, err := Derive("ui/src/features/notes/flow/flow.json", LanguageTypeScript)
	if err != nil {
		t.Fatalf("Derive() error = %v", err)
	}
	want := Layout{
		Language:         LanguageTypeScript,
		BaseDir:          "ui/src/features/notes/flow",
		ModelPath:        "ui/src/features/notes/flow/generated/model.qnt",
		ArtifactPath:     "ui/src/features/notes/flow/generated/artifact.json",
		RuntimePath:      "ui/src/features/notes/flow/generated/runtime.ts",
		ReplayHelperPath: "ui/src/features/notes/flow/generated/replay.helper.ts",
		TransitionPath:   "ui/src/features/notes/flow/transition.ts",
		FixturesPath:     "ui/src/features/notes/flow/fixtures.ts",
		TestPath:         "ui/src/features/notes/flow/flow.test.ts",
	}
	if got != want {
		t.Fatalf("Derive() mismatch:\n got=%#v\nwant=%#v", got, want)
	}
}

func TestDeriveGo(t *testing.T) {
	got, err := Derive("api/internal/notes/flow/flow.json", LanguageGo)
	if err != nil {
		t.Fatalf("Derive() error = %v", err)
	}
	if got.BaseDir != "api/internal/notes/flow" {
		t.Fatalf("BaseDir = %s", got.BaseDir)
	}
	if got.TransitionPath != "api/internal/notes/flow/transition.go" {
		t.Fatalf("TransitionPath = %s", got.TransitionPath)
	}
	if got.TestPath != "api/internal/notes/flow/flow_test.go" {
		t.Fatalf("TestPath = %s", got.TestPath)
	}
	if got.FixturesPath != "" {
		t.Fatalf("Go layout should not declare FixturesPath, got %s", got.FixturesPath)
	}
}

func TestDeriveRejectsNonFlowDir(t *testing.T) {
	_, err := Derive("api/internal/notes/notes.flow.json", LanguageGo)
	if err == nil || !strings.Contains(err.Error(), `must live in a directory named "flow"`) {
		t.Fatalf("expected flow-dir rejection, got %v", err)
	}
}

func TestSubpackageImportPathStripsAPIPrefix(t *testing.T) {
	lay := Layout{BaseDir: "api/internal/notes/flow"}
	got := SubpackageImportPath(lay)
	want := "{{SCENARIO_ID}}/internal/notes/flow/generated"
	if got != want {
		t.Fatalf("SubpackageImportPath = %s, want %s", got, want)
	}
}
