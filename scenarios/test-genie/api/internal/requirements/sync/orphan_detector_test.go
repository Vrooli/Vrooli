package sync

import (
	"context"
	"path/filepath"
	"testing"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

func TestOrphanDetectionChecksAllScenarioReferenceRoots(t *testing.T) {
	for _, base := range []string{"", "api", "ui", "test"} {
		name := base
		if name == "" {
			name = "scenario root"
		}
		t.Run(name, func(t *testing.T) {
			const scenarioRoot = "/scenario"
			const ref = "proof_test.go"
			reader := newMemReader()
			path := filepath.Join(scenarioRoot, base, ref)
			reader.files[path] = []byte("test file")

			index := parsing.NewModuleIndex()
			mustAddModule(t, index, &types.RequirementModule{
				FilePath: filepath.Join(scenarioRoot, "requirements", "module.json"),
				Requirements: []types.Requirement{{
					ID:          "REQ-ORPHAN-ROOTS",
					Validations: []types.Validation{{Type: types.ValTypeTest, Ref: ref}},
				}},
			})

			orphans, err := NewOrphanDetector(reader).DetectOrphans(context.Background(), index, scenarioRoot)
			if err != nil {
				t.Fatalf("detect orphans: %v", err)
			}
			if len(orphans) != 0 {
				t.Fatalf("existing reference %q under %q was marked orphan: %+v", ref, base, orphans)
			}
			if IsOrphanedRef(ref, scenarioRoot, reader) {
				t.Fatalf("IsOrphanedRef marked existing path %q under %q orphaned", ref, base)
			}

			preview, err := New(reader, newMemWriter()).Sync(context.Background(), index, types.NewEvidenceBundle(), Options{
				PruneOrphans: true,
				DryRun:       true,
				ScenarioRoot: scenarioRoot,
			})
			if err != nil {
				t.Fatalf("preview pruning: %v", err)
			}
			if preview.ValidationsRemoved != 0 || len(preview.Changes) != 0 {
				t.Fatalf("preview planned destructive removal for existing path: removed=%d changes=%+v", preview.ValidationsRemoved, preview.Changes)
			}
		})
	}
}

func TestOrphanDetectionKeepsMissingReferencesOrphaned(t *testing.T) {
	const scenarioRoot = "/scenario"
	const ref = "missing_test.go"
	reader := newMemReader()
	index := parsing.NewModuleIndex()
	mustAddModule(t, index, &types.RequirementModule{
		FilePath: filepath.Join(scenarioRoot, "requirements", "module.json"),
		Requirements: []types.Requirement{{
			ID:          "REQ-ORPHAN-MISSING",
			Validations: []types.Validation{{Type: types.ValTypeTest, Ref: ref}},
		}},
	})

	orphans, err := NewOrphanDetector(reader).DetectOrphans(context.Background(), index, scenarioRoot)
	if err != nil {
		t.Fatalf("detect orphans: %v", err)
	}
	if len(orphans) != 1 || orphans[0].RequirementID != "REQ-ORPHAN-MISSING" || orphans[0].ValidationRef != ref {
		t.Fatalf("missing reference classification = %+v, want the exact orphan", orphans)
	}
	if !IsOrphanedRef(ref, scenarioRoot, reader) {
		t.Fatal("IsOrphanedRef accepted a missing reference")
	}
}
