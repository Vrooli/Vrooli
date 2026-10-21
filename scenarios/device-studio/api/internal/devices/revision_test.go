package devices

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"

	"device-studio/internal/library"
	"device-studio/internal/physical"
)

func setupDeviceRevisions(t *testing.T) (context.Context, *sql.DB, *DeviceRevisionService) {
	t.Helper()
	db := databasetest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteDeviceRevisionRepository(db)
	service, err := NewDeviceRevisionService(repo, func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	return context.Background(), db, service
}

func TestDeviceCandidateConcurrentDifferentWritersHaveOneWinner(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	other, err := NewDeviceRevisionService(service.repo, service.now)
	if err != nil {
		t.Fatal(err)
	}
	left, right := candidateFixture(), candidateFixture()
	left.Definition.Description = "left winner candidate"
	right.Definition.Description = "right winner candidate"
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, item := range []struct {
		service *DeviceRevisionService
		input   DeviceCandidate
	}{{service, left}, {other, right}} {
		wg.Add(1)
		go func(service *DeviceRevisionService, input DeviceCandidate) {
			defer wg.Done()
			<-start
			_, err := service.SaveCandidate(ctx, input, 0)
			results <- err
		}(item.service, item.input)
	}
	close(start)
	wg.Wait()
	close(results)
	winners, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent write error: %v", err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("concurrent different candidates: winners=%d conflicts=%d", winners, conflicts)
	}
	head, err := service.GetHead(ctx, "fixture")
	if err != nil || head.Revision != 1 {
		t.Fatalf("expected one immutable winner, head=%d err=%v", head.Revision, err)
	}
}

func candidateFixture() DeviceCandidate {
	return DeviceCandidate{
		Version:              DeviceCandidateVersion,
		Definition:           validFixture(),
		Geometry:             physicalRefFixture(),
		GeometryUnit:         CanonicalUnit,
		CoordinateConvention: physical.CoordinateConvention,
		SourceToCanonical:    identityTRS(),
		Parts:                []SemanticPart{{ID: "body", Role: "assembly", NodeIndex: 0, NodeName: "unverified-glb-root", LocalFrame: identityTRS()}},
		DisplayBindings:      []DisplayPartBinding{{DisplayID: "front", PartID: "body", LocalFrame: identityTRS()}},
	}
}

func physicalRefFixture() physical.ArtifactRef {
	return physical.ArtifactRef{
		BundleID: "reviewed-phone", BundleRevision: 1,
		ManifestSHA256: strings.Repeat("a", 64), ReviewSHA256: strings.Repeat("b", 64),
		ArtifactID: "phone-glb", Role: library.RoleGeometryGLB, SizeBytes: 128, SHA256: strings.Repeat("c", 64),
	}
}

func identityTRS() physical.TRS {
	return physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
}

func TestDeviceCandidateRevisionReplayConflictAndImmutableHistory(t *testing.T) {
	ctx, db, service := setupDeviceRevisions(t)
	first, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != DeviceRevisionCandidate {
		t.Fatalf("new device revision state=%q, want candidate", first.State)
	}
	replay, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil || replay.Revision != 1 || replay.SHA256 != first.SHA256 {
		t.Fatalf("exact replay did not converge: revision=%d err=%v", replay.Revision, err)
	}
	changed := candidateFixture()
	changed.Definition.Description = "candidate revision two"
	second, err := service.SaveCandidate(ctx, changed, 1)
	if err != nil || second.Revision != 2 {
		t.Fatalf("save next revision: revision=%d err=%v", second.Revision, err)
	}
	if _, err := service.SaveCandidate(ctx, candidateFixture(), 1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("reverting content at expected target revision 2 must conflict; save it at the current head instead: %v", err)
	}
	stale := changed
	stale.Definition.Description = "different stale edit"
	if _, err := service.SaveCandidate(ctx, stale, 1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected stale head conflict, got %v", err)
	}
	reverted, err := service.SaveCandidate(ctx, candidateFixture(), 2)
	if err != nil || reverted.Revision != 3 || reverted.SHA256 != first.SHA256 {
		t.Fatalf("A→B→A must create revision 3: revision=%d err=%v", reverted.Revision, err)
	}
	firstReplay, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil || firstReplay.Revision != 1 {
		t.Fatalf("retry targeting revision 1 should return revision 1: revision=%d err=%v", firstReplay.Revision, err)
	}
	if _, err := service.SaveCandidate(ctx, changed, 2); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("different content at occupied target revision should conflict, got %v", err)
	}
	if _, err := db.Exec(`UPDATE device_revisions SET candidate_json=candidate_json WHERE device_id='fixture' AND revision=1`); err == nil {
		t.Fatal("immutable revision trigger allowed update")
	}
	items, err := service.ListRevisions(ctx, "fixture", 0, 10)
	if err != nil || len(items) != 3 || items[0].Revision != 1 || items[1].Revision != 2 || items[2].Revision != 3 {
		t.Fatalf("revision listing mismatch: %+v err=%v", items, err)
	}
}

func TestDeviceCandidateUsesLibraryStableIdentifierContract(t *testing.T) {
	input := candidateFixture()
	input.Geometry.BundleID = strings.Repeat("B", 70)
	input.Geometry.ArtifactID = "Geometry-Original_01"
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err != nil {
		t.Fatalf("library-valid uppercase/long reference ID was rejected: %v", err)
	}
}

func TestDeviceCandidateCanonicalizationDoesNotMutateInputAndIsIdempotent(t *testing.T) {
	input := candidateFixture()
	input.Parts[0].LocalFrame.Rotation = physical.QuatXYZW{X: 0.1, Y: 0.2, Z: 0.3, W: 0.4}
	original := input.Parts[0].LocalFrame.Rotation
	first, _, firstDigest, err := CanonicalizeDeviceCandidate(input)
	if err != nil {
		t.Fatal(err)
	}
	if input.Parts[0].LocalFrame.Rotation != original {
		t.Fatal("canonicalization mutated caller part transform")
	}
	second, _, secondDigest, err := CanonicalizeDeviceCandidate(first)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != secondDigest || first.Parts[0].LocalFrame.Rotation != second.Parts[0].LocalFrame.Rotation {
		t.Fatalf("canonicalization is not idempotent: %s != %s", firstDigest, secondDigest)
	}
}

func TestDeviceCandidateRejectsRevisionOverflow(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	if _, err := service.SaveCandidate(ctx, candidateFixture(), ^uint32(0)); err == nil {
		t.Fatal("expected exhausted revision number to be rejected")
	}
}

func TestDeviceCandidateConcurrentIdenticalWritersConverge(t *testing.T) {
	ctx, _, service := setupDeviceRevisions(t)
	other, err := NewDeviceRevisionService(service.repo, service.now)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan DeviceRevision, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, writer := range []*DeviceRevisionService{service, other} {
		wg.Add(1)
		go func(writer *DeviceRevisionService) {
			defer wg.Done()
			<-start
			result, err := writer.SaveCandidate(ctx, candidateFixture(), 0)
			results <- result
			errs <- err
		}(writer)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("identical concurrent writer failed to converge: %v", err)
		}
	}
	for result := range results {
		if result.Revision != 1 {
			t.Fatalf("concurrent replay created unexpected revision %d", result.Revision)
		}
	}
}

func TestDeviceCandidateStoredDigestAndCanonicalJSONAreVerified(t *testing.T) {
	ctx, db, service := setupDeviceRevisions(t)
	created, err := service.SaveCandidate(ctx, candidateFixture(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TRIGGER device_revisions_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE device_revisions SET sha256=? WHERE device_id=? AND revision=1`, strings.Repeat("d", 64), created.DeviceID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRevision(ctx, created.DeviceID, 1); err == nil {
		t.Fatal("mismatched stored digest passed integrity verification")
	}
	if _, err := db.Exec(`UPDATE device_revisions SET sha256=? WHERE device_id=? AND revision=1`, created.SHA256, created.DeviceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE device_revisions SET candidate_json=candidate_json || ' ' WHERE device_id=? AND revision=1`, created.DeviceID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRevision(ctx, created.DeviceID, 1); err == nil {
		t.Fatal("noncanonical stored bytes passed integrity verification")
	}
}

func TestDeviceCandidateWrapperPreservesLegacyDefinitionJSON(t *testing.T) {
	definition := validFixture()
	before, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	input := candidateFixture()
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(input.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("candidate wrapper changed the legacy Definition JSON representation")
	}
}

func TestDisplayBindingScreenSelectorPairIsOptionalButAtomic(t *testing.T) {
	input := candidateFixture()
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err != nil {
		t.Fatalf("legacy display binding without a selector must remain valid: %v", err)
	}
	primitive, material := uint32(0), uint32(0)
	input.DisplayBindings[0].ScreenPrimitiveIndex = &primitive
	input.DisplayBindings[0].ScreenMaterialIndex = &material
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err != nil {
		t.Fatalf("explicit primitive/material zeroes must remain distinguishable from omitted selectors: %v", err)
	}
	input.DisplayBindings[0].ScreenMaterialIndex = nil
	if _, _, _, err := CanonicalizeDeviceCandidate(input); err == nil || !strings.Contains(err.Error(), "must pin both") {
		t.Fatalf("partial screen selector pair was accepted: %v", err)
	}
}

func TestDeviceCandidateRejectsInvalidReferenceGraphsAndTransforms(t *testing.T) {
	tests := map[string]func(*DeviceCandidate){
		"duplicate part": func(c *DeviceCandidate) { c.Parts = append(c.Parts, c.Parts[0]) },
		"parent cycle": func(c *DeviceCandidate) {
			c.Parts = append(c.Parts, SemanticPart{ID: "screen", Role: "display", ParentID: "body", NodeIndex: 1, NodeName: "screen", LocalFrame: identityTRS()})
			c.Parts[0].ParentID = "screen"
		},
		"unknown display": func(c *DeviceCandidate) { c.DisplayBindings[0].DisplayID = "missing" },
		"unknown joint part": func(c *DeviceCandidate) {
			c.Definition.Joints = []Joint{{ID: "hinge", ParentPart: "body", ChildPart: "missing"}}
		},
		"joint cycle": func(c *DeviceCandidate) {
			c.Parts = append(c.Parts, SemanticPart{ID: "lid", Role: "display", ParentID: "body", NodeIndex: 1, NodeName: "lid", LocalFrame: identityTRS()})
			c.Definition.Joints = []Joint{{ID: "hinge-a", ParentPart: "body", ChildPart: "lid"}, {ID: "hinge-b", ParentPart: "lid", ChildPart: "body"}}
		},
		"joint contradicts part hierarchy": func(c *DeviceCandidate) {
			c.Parts = append(c.Parts,
				SemanticPart{ID: "base", Role: "base", NodeIndex: 1, NodeName: "base", LocalFrame: identityTRS()},
				SemanticPart{ID: "lid", Role: "lid", ParentID: "base", NodeIndex: 2, NodeName: "lid", LocalFrame: identityTRS()},
			)
			c.Definition.Joints = []Joint{{ID: "hinge", ParentPart: "body", ChildPart: "lid"}}
		},
		"multiple joints for child": func(c *DeviceCandidate) {
			c.Parts = append(c.Parts, SemanticPart{ID: "lid", Role: "lid", ParentID: "body", NodeIndex: 1, NodeName: "lid", LocalFrame: identityTRS()})
			c.Definition.Joints = []Joint{{ID: "hinge-x", ParentPart: "body", ChildPart: "lid"}, {ID: "hinge-y", ParentPart: "body", ChildPart: "lid"}}
		},
		"nonfinite transform":   func(c *DeviceCandidate) { c.Parts[0].LocalFrame.Translation.X = math.Inf(1) },
		"claimed qualification": func(c *DeviceCandidate) { c.Definition.Qualification = "qualified" },
		"claimed qualification review": func(c *DeviceCandidate) {
			c.Definition.QualificationReviews = []QualificationReview{{Scope: "physical_geometry", Result: "passed", Reviewer: "caller"}}
		},
		"wrong geometry role": func(c *DeviceCandidate) { c.Geometry.Role = library.RoleTextureImage },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			input := candidateFixture()
			mutate(&input)
			if _, _, _, err := CanonicalizeDeviceCandidate(input); err == nil {
				t.Fatal("invalid candidate was accepted")
			}
		})
	}
}
