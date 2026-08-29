package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNormalizeInactiveInternalPins(t *testing.T) {
	pins := map[string]activeInternalPin{
		"Form": {
			latest:     "1.0.1",
			deprecated: map[string]bool{"1.0.0": true},
		},
		"Button": {
			latest:     "2.0.0",
			deprecated: map[string]bool{"1.0.0": true},
		},
	}
	input := `
import { Form } from "@vrooli/react-component-library/Form/1.0.0";
import { Button } from "@vrooli/react-component-library/Button/1.5.0";
import { External } from "@vrooli/react-component-library/External/1.0.0";
`
	want := `
import { Form } from "@vrooli/react-component-library/Form/1.0.1";
import { Button } from "@vrooli/react-component-library/Button/1.5.0";
import { External } from "@vrooli/react-component-library/External/1.0.0";
`
	if got := normalizeInactiveInternalPins(input, pins); got != want {
		t.Fatalf("normalizeInactiveInternalPins() = %q, want %q", got, want)
	}
}

func TestGeneratedArtifactMatchesRejectsThemeDriftAndIgnoresManagedReleaseHeader(t *testing.T) {
	if generatedArtifactMatches("tailwind.theme.json", "expected\nchanged", "expected\n") {
		t.Fatal("drifted theme unexpectedly matched")
	}
	expected := "/** @vrooliComponentSource foundations.tokens */\nexport const tokens = {};\n"
	current := "/**\n * @libraryId react-component-library:Tokens\n */\n" + expected
	if !generatedArtifactMatches(filepath.Join("versions", "1.0.1", "Tokens.tsx"), current, expected) {
		t.Fatal("publisher-managed release header should not make generated Tokens source stale")
	}
}

func TestLiveGeneratedTokenArtifactsAreCurrent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))
	cmd := exec.Command("go", "run", ".", "--root", root, "--check")
	cmd.Dir = wd
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated artifacts are stale: %v\n%s", err, output)
	}
}
