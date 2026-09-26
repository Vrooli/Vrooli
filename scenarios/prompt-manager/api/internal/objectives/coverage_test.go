package objectives

import (
	"context"
	"testing"
)

func TestBuildCoverageReadsFromAuthority(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	// A team that declares no objective at all must still be named as
	// unattached, which the authority alone cannot answer.
	writeTeam(t, configDir, "empty-team", `{"id":"empty-team","objectivesServed":[]}`)

	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)
	if _, err := svc.Import(ctx, preview); err != nil {
		t.Fatalf("import: %v", err)
	}

	cov, err := BuildCoverage(ctx, svc, repoRoot, configDir)
	if err != nil {
		t.Fatalf("build coverage: %v", err)
	}
	if cov.SourcePath == "" {
		t.Fatal("coverage must name the operator statement source path")
	}
	if len(cov.Rows) != 2 || cov.Rows[0].ID != "T1" || cov.Rows[1].ID != "I1" {
		t.Fatalf("expected T1 then I1 in global order, got %+v", cov.Rows)
	}
	for _, row := range cov.Rows {
		if !row.Served || len(row.ServedBy) == 0 {
			t.Fatalf("objective %s should be served from the authority, got %+v", row.ID, row)
		}
		if len(row.DeclaredBy) == 0 {
			t.Fatalf("objective %s should retain its declaration context, got %+v", row.ID, row)
		}
		if row.MeaningRevision == "" {
			t.Fatalf("objective %s should expose the authority meaning revision, got %+v", row.ID, row)
		}
		for _, ref := range row.ServedBy {
			if ref.AcknowledgedRevision == "" {
				t.Fatalf("served ref for %s should expose the authority acknowledgement, got %+v", row.ID, ref)
			}
		}
	}
	if cov.Rows[0].GlobalOrder >= cov.Rows[1].GlobalOrder {
		t.Fatalf("rows must follow authority global order, got %d then %d", cov.Rows[0].GlobalOrder, cov.Rows[1].GlobalOrder)
	}
	// The team-major runtime read must carry the same ordered objectives and
	// meaning revisions as the objective-major rows.
	if len(cov.Teams) != 3 {
		t.Fatalf("expected the three attached teams in the runtime read, got %+v", cov.Teams)
	}
	for _, tc := range cov.Teams {
		if tc.AttachmentRevision == "" {
			t.Fatalf("team %s must carry its attachment revision", tc.TeamID)
		}
		if len(tc.Objectives) == 0 {
			t.Fatalf("team %s must carry its ordered objective context", tc.TeamID)
		}
		for _, ref := range tc.Objectives {
			if ref.MeaningRevision == "" {
				t.Fatalf("team %s objective %s must carry the authority meaning revision", tc.TeamID, ref.ObjectiveID)
			}
			if ref.RestatementPending {
				t.Fatalf("team %s objective %s should not be restatement-pending after matching acknowledgements", tc.TeamID, ref.ObjectiveID)
			}
		}
	}
	if cov.Unserved != 0 || cov.Undeclared != 0 {
		t.Fatalf("expected no unserved objectives, got unserved=%d undeclared=%d", cov.Unserved, cov.Undeclared)
	}
	if len(cov.UnattachedTeams) != 1 || cov.UnattachedTeams[0] != "empty-team" {
		t.Fatalf("expected empty-team as the only unattached team, got %v", cov.UnattachedTeams)
	}
	if cov.Drift.HasDrift {
		t.Fatalf("expected no drift immediately after import, got %+v", cov.Drift.Items)
	}
	if cov.Validation.Errors != 0 {
		t.Fatalf("expected no validation errors, got %+v", cov.Validation)
	}
}

func TestBuildCoverageSurfacesAuthorityDrift(t *testing.T) {
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

	// A live meaning edit in the authority is exactly what the retained
	// declarations must not silently mask.
	obj, ok, err := svc.GetObjective(ctx, "T1")
	if err != nil || !ok {
		t.Fatalf("get T1: ok=%v err=%v", ok, err)
	}
	obj.Title = "Income, restated"
	if _, err := svc.UpsertObjective(ctx, obj, obj.MeaningRevision); err != nil {
		t.Fatalf("restate T1: %v", err)
	}

	cov, err := BuildCoverage(ctx, svc, repoRoot, configDir)
	if err != nil {
		t.Fatalf("build coverage: %v", err)
	}
	if !cov.Drift.HasDrift {
		t.Fatal("expected drift after a live authority meaning edit")
	}
	found := false
	for _, item := range cov.Drift.Items {
		if item.Kind == DriftObjectiveMeaningChanged && item.ObjectiveID == "T1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a T1 meaning-change drift item, got %+v", cov.Drift.Items)
	}
}
