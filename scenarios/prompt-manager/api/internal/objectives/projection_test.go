package objectives

import (
	"context"
	"strings"
	"testing"
)

func TestBuildProjectionRendersWithoutMutation(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)
	if _, err := svc.Import(ctx, preview); err != nil {
		t.Fatalf("import: %v", err)
	}

	before, err := svc.ListObjectives(ctx)
	if err != nil {
		t.Fatalf("list before: %v", err)
	}

	projection, err := BuildProjection(ctx, svc)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	if len(projection.Objectives) != 2 {
		t.Fatalf("expected 2 objectives in the projection, got %d", len(projection.Objectives))
	}
	if projection.Objectives[0].ID != "T1" || projection.Objectives[1].ID != "I1" {
		t.Fatalf("projection must be in global order, got %+v", projection.Objectives)
	}
	if len(projection.Teams) != 3 {
		t.Fatalf("expected 3 teams in the projection, got %d", len(projection.Teams))
	}
	for _, team := range projection.Teams {
		if team.AttachmentRevision == "" {
			t.Fatalf("team %s projection is missing its attachment revision", team.TeamID)
		}
	}

	after, err := svc.ListObjectives(ctx)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	if len(before) != len(after) {
		t.Fatalf("projection must not mutate: before %d objectives, after %d", len(before), len(after))
	}
}

func TestRenderProjectionIsDeterministicAndReadOnly(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)
	if _, err := svc.Import(ctx, preview); err != nil {
		t.Fatalf("import: %v", err)
	}
	projection, err := BuildProjection(ctx, svc)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	first := RenderProjection(projection)
	second := RenderProjection(projection)
	if first != second {
		t.Fatal("projection rendering must be deterministic")
	}
	if !strings.Contains(first, "# Objective authority projection (read-only)") {
		t.Fatalf("projection must declare itself read-only, got:\n%s", first)
	}
	if !strings.Contains(first, "| T1 |") || !strings.Contains(first, "| I1 |") {
		t.Fatalf("projection must list both objectives, got:\n%s", first)
	}
}

func TestCompareProjectionReportsNoDriftAfterImport(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)
	if _, err := svc.Import(ctx, preview); err != nil {
		t.Fatalf("import: %v", err)
	}
	projection, err := BuildProjection(ctx, svc)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}

	drift := CompareProjection(preview, projection)
	if drift.HasDrift {
		t.Fatalf("expected no drift immediately after import, got %+v", drift.Items)
	}
}

func TestCompareProjectionDetectsAuthorityAndSourceDrift(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)
	if _, err := svc.Import(ctx, preview); err != nil {
		t.Fatalf("import: %v", err)
	}

	// A live edit of the objective meaning in the authority is exactly the
	// case the projection must surface: the statement changed after the team
	// acknowledged it.
	obj, ok, err := svc.GetObjective(ctx, "T1")
	if err != nil || !ok {
		t.Fatalf("get T1: ok=%v err=%v", ok, err)
	}
	obj.Title = "Income. Earn revenue the new way."
	if _, err := svc.UpsertObjective(ctx, obj, obj.MeaningRevision); err != nil {
		t.Fatalf("edit T1: %v", err)
	}
	// A source-only declaration (a team link the authority never received)
	// must be reported as missing-in-authority.
	preview.Attachments = append(preview.Attachments, ImportedAttachment{
		ObjectiveID: "I1",
		TeamID:      "ghost-team",
		Role:        "supporting",
	})

	projection, err := BuildProjection(ctx, svc)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	drift := CompareProjection(preview, projection)
	if !drift.HasDrift {
		t.Fatal("expected drift after an authority meaning edit and a new source declaration")
	}
	var sawMeaning, sawMissing, sawAttachment bool
	for _, item := range drift.Items {
		switch item.Kind {
		case DriftObjectiveMeaningChanged:
			if item.ObjectiveID == "T1" {
				sawMeaning = true
			}
		case DriftObjectiveUntrackedInSource:
			if item.ObjectiveID == "" {
				t.Fatalf("untracked-source item must name the objective: %+v", item)
			}
		case DriftAttachmentMissingInAuthority:
			if item.TeamID == "ghost-team" && item.ObjectiveID == "I1" {
				sawMissing = true
			}
		}
	}
	_ = sawAttachment
	if !sawMeaning {
		t.Fatalf("expected a meaning-changed drift for T1, got %+v", drift.Items)
	}
	if !sawMissing {
		t.Fatalf("expected a missing-in-authority drift for ghost-team/I1, got %+v", drift.Items)
	}
}

func TestCompareProjectionDetectsStaleAcknowledgement(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)
	if _, err := svc.Import(ctx, preview); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Drift the authority by editing T1's meaning without a team re-ack:
	// every serving team's acknowledgement becomes stale.
	obj, ok, err := svc.GetObjective(ctx, "T1")
	if err != nil || !ok {
		t.Fatalf("get T1: ok=%v err=%v", ok, err)
	}
	obj.Title = "Income. Rolling revenue target."
	if _, err := svc.UpsertObjective(ctx, obj, obj.MeaningRevision); err != nil {
		t.Fatalf("edit T1: %v", err)
	}
	projection, err := BuildProjection(ctx, svc)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	drift := CompareProjection(preview, projection)
	if !drift.HasDrift {
		t.Fatal("expected acknowledgement drift after a meaning edit")
	}
}
