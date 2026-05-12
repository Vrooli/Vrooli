package operatingmode

import (
	"errors"
	"testing"
)

func TestStore_WriteReadArtifactWithinModeRoot(t *testing.T) {
	store := testStore(t)

	rel, err := store.WriteArtifact("sandboxing", ModeHolisticLoop, "modes/holistic-loop/findings.md", []byte("# Findings\n"))
	if err != nil {
		t.Fatalf("WriteArtifact: %v", err)
	}
	if rel != "modes/holistic-loop/findings.md" {
		t.Fatalf("relative path = %q", rel)
	}

	snapshot, err := store.ReadArtifact("sandboxing", ModeHolisticLoop, rel)
	if err != nil {
		t.Fatalf("ReadArtifact: %v", err)
	}
	if snapshot.Content != "# Findings\n" {
		t.Fatalf("content = %q", snapshot.Content)
	}
	if !snapshot.Required {
		t.Fatal("declared artifact should be marked required")
	}
	if snapshot.ContentType != "text/markdown" {
		t.Fatalf("content type = %q", snapshot.ContentType)
	}
}

func TestStore_ArtifactPathRejectsTraversalAndWrongModeRoot(t *testing.T) {
	store := testStore(t)

	if _, err := store.ArtifactPath("sandboxing", ModeHolisticLoop, "../outside.md"); err == nil {
		t.Fatal("ArtifactPath accepted traversal")
	}
	if _, err := store.ArtifactPath("sandboxing", ModeHolisticLoop, "modes/phased-plan-drain/phased-plan.md"); err == nil {
		t.Fatal("ArtifactPath accepted artifact outside holistic-loop root")
	}
}

func TestStore_ListDeclaredArtifactsIncludesMissingAndExisting(t *testing.T) {
	store := testStore(t)
	if _, err := store.WriteArtifact("sandboxing", ModePhasedPlanDrain, "modes/phased-plan-drain/phased-plan.md", []byte("plan")); err != nil {
		t.Fatalf("WriteArtifact: %v", err)
	}

	artifacts, err := store.ListDeclaredArtifacts("sandboxing", ModePhasedPlanDrain)
	if err != nil {
		t.Fatalf("ListDeclaredArtifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("artifact count = %d, want 2", len(artifacts))
	}
	if artifacts[0].Path != "modes/phased-plan-drain/phased-plan.md" || artifacts[0].Content != "plan" {
		t.Fatalf("first artifact = %+v", artifacts[0])
	}
	if artifacts[1].Path != "modes/phased-plan-drain/progress.json" || artifacts[1].Content != "" {
		t.Fatalf("missing artifact = %+v", artifacts[1])
	}
}

func TestStore_ReadArtifactNotFound(t *testing.T) {
	store := testStore(t)
	_, err := store.ReadArtifact("sandboxing", ModeHolisticLoop, "modes/holistic-loop/findings.md")
	if !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("ReadArtifact err = %v, want ErrArtifactNotFound", err)
	}
}
