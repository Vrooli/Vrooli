package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"device-studio/internal/revisionstore"
	"device-studio/internal/routedblobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func intakeFixtureGLB() []byte {
	jsonChunk := []byte(`{"asset":{"version":"2.0"}}`)
	for len(jsonChunk)%4 != 0 {
		jsonChunk = append(jsonChunk, ' ')
	}
	out := make([]byte, 12+8+len(jsonChunk))
	copy(out[:4], "glTF")
	binary.LittleEndian.PutUint32(out[4:8], 2)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(jsonChunk)))
	binary.LittleEndian.PutUint32(out[16:20], 0x4e4f534a)
	copy(out[20:], jsonChunk)
	return out
}

func intakeMaterialFixtureGLB() []byte {
	jsonChunk := []byte(`{"asset":{"version":"2.0"},"buffers":[{"byteLength":4}],"bufferViews":[{"buffer":0,"byteLength":4}],"accessors":[{"bufferView":0,"componentType":5121,"count":4,"type":"SCALAR"}],"meshes":[{"primitives":[{"attributes":{"POSITION":0}}]}],"nodes":[{"mesh":0}],"materials":[{"name":"case","pbrMetallicRoughness":{"baseColorFactor":[0.4,0.4,0.4,1],"metallicFactor":0,"roughnessFactor":0.8}},{ "name" : "untouched", "pbrMetallicRoughness" : {"roughnessFactor":0.5} }]}`)
	for len(jsonChunk)%4 != 0 {
		jsonChunk = append(jsonChunk, ' ')
	}
	bin := []byte{1, 2, 3, 4}
	out := make([]byte, 12+8+len(jsonChunk)+8+len(bin))
	copy(out, "glTF")
	binary.LittleEndian.PutUint32(out[4:8], 2)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(jsonChunk)))
	binary.LittleEndian.PutUint32(out[16:20], 0x4e4f534a)
	copy(out[20:], jsonChunk)
	position := 20 + len(jsonChunk)
	binary.LittleEndian.PutUint32(out[position:position+4], uint32(len(bin)))
	binary.LittleEndian.PutUint32(out[position+4:position+8], 0x004e4942)
	copy(out[position+8:], bin)
	return out
}

func TestIntakeMaterialCorrectionRequiresObservedParentAndPreservesGeometryBytes(t *testing.T) {
	ctx, service, _, _ := setupIntake(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "test", ProfileVersion: 1, Supported: true, NodeCount: 3, MeshCount: 2, TriangleCount: 14}, nil
	})
	original := intakeMaterialFixtureGLB()
	staged, err := service.BeginImport(ctx, intakeClaims("material-parent"), "source.glb", int64(len(original)), bytes.NewReader(original))
	if err != nil {
		t.Fatal(err)
	}
	parent, err := staged.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := service.MaterialInventory(ctx, parent.ID, parent.Revision, parent.RevisionSHA256)
	if err != nil || len(inventory.Materials) != 2 || inventory.Materials[0].Name != "case" || inventory.Materials[0].RoughnessFactor != 0.8 {
		t.Fatalf("inventory=%+v err=%v", inventory, err)
	}
	wrong := MaterialCorrectionRequest{ParentCandidateID: parent.ID, ParentRevision: parent.Revision, ParentRevisionSHA256: parent.RevisionSHA256, ChildCandidateID: "material-child", MaterialIndex: 0, MaterialName: "stale-name", RoughnessFactor: floatPointer(0.2)}
	if _, err := service.CorrectMaterial(ctx, wrong); !errors.Is(err, ErrInvalidIntake) {
		t.Fatalf("stale material selector error=%v", err)
	}
	wrong.MaterialName = "case"
	wrong.MetallicFactor = floatPointer(1.1)
	if _, err := service.CorrectMaterial(ctx, wrong); !errors.Is(err, ErrInvalidIntake) {
		t.Fatalf("out-of-range factor error=%v", err)
	}
	wrong.MetallicFactor = nil
	wrong.RoughnessFactor = floatPointer(0.2)
	child, err := service.CorrectMaterial(ctx, wrong)
	if err != nil {
		t.Fatal(err)
	}
	if child.ID != "material-child" || child.Revision != 1 || child.ParentID != parent.ID || child.ParentRevision != parent.Revision || child.ParentSHA256 != parent.RevisionSHA256 || child.ReviewStatus != "unreviewed" || child.RenderStatus != "not_reviewed" || child.Correction == nil || !child.Correction.GeometryUnchanged {
		t.Fatalf("invalid correction child: %+v", child)
	}
	if child.Correction.BeforeGLBSHA256 != parent.OriginalSHA256 || child.Correction.AfterGLBSHA256 != child.OriginalSHA256 || !reflect.DeepEqual(child.Correction.BeforeBINChunksSHA256, child.Correction.AfterBINChunksSHA256) {
		t.Fatalf("correction lineage lost hashes: %+v", child.Correction)
	}
	_, parentReader, err := service.Open(ctx, parent.ID, parent.Revision)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := io.ReadAll(parentReader)
	if err != nil {
		t.Fatal(err)
	}
	_ = parentReader.Close()
	if !bytes.Equal(retained, original) {
		t.Fatal("parent GLB bytes changed")
	}
	_, childReader, err := service.Open(ctx, child.ID, child.Revision)
	if err != nil {
		t.Fatal(err)
	}
	childBytes, err := io.ReadAll(childReader)
	if err != nil {
		t.Fatal(err)
	}
	_ = childReader.Close()
	parentDoc, _, _, err := decodeMaterialGLB(original)
	if err != nil {
		t.Fatal(err)
	}
	childDoc, _, _, err := decodeMaterialGLB(childBytes)
	if err != nil {
		t.Fatal(err)
	}
	var parentMaterials, childMaterials []json.RawMessage
	if err := json.Unmarshal(parentDoc["materials"], &parentMaterials); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(childDoc["materials"], &childMaterials); err != nil {
		t.Fatal(err)
	}
	var parentOther, childOther any
	if err := json.Unmarshal(parentMaterials[1], &parentOther); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(childMaterials[1], &childOther); err != nil {
		t.Fatal(err)
	}
	if len(parentMaterials) != 2 || len(childMaterials) != 2 || !reflect.DeepEqual(parentOther, childOther) {
		t.Fatal("unselected material values changed")
	}
	for _, key := range []string{"buffers", "bufferViews", "accessors", "meshes", "nodes"} {
		var parentValue, childValue any
		if err := json.Unmarshal(parentDoc[key], &parentValue); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(childDoc[key], &childValue); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(parentValue, childValue) {
			t.Fatalf("geometry document field %s changed", key)
		}
	}
	_, _, beforeBins, err := decodeMaterialGLB(original)
	if err != nil {
		t.Fatal(err)
	}
	_, _, afterBins, err := decodeMaterialGLB(childBytes)
	if err != nil || !reflect.DeepEqual(beforeBins, afterBins) {
		t.Fatalf("embedded BIN changed: before=%v after=%v err=%v", beforeBins, afterBins, err)
	}
	replayed, err := service.CorrectMaterial(ctx, wrong)
	if err != nil || replayed.RevisionSHA256 != child.RevisionSHA256 {
		t.Fatalf("exact operation replay=%+v err=%v", replayed, err)
	}
	wrong.RoughnessFactor = floatPointer(0.3)
	if _, err := service.CorrectMaterial(ctx, wrong); !errors.Is(err, ErrIntakeConflict) {
		t.Fatalf("child id reuse error=%v", err)
	}
}

func floatPointer(v float64) *float64 { return &v }

func setupIntake(t *testing.T, inspect IntakeInspector) (context.Context, *IntakeService, string, *sql.DB) {
	return setupIntakeWithDB(t, inspect)
}

func intakeClaims(id string) IntakeClaims {
	return IntakeClaims{CandidateID: id, DisplayName: "Reference phone", SourceLabel: "local source note", LicenseClaim: "CC-BY-4.0", Attribution: "Example author", DeclaredUses: []string{"render"}}
}

func TestIntakeStoresOriginalAsUnreviewedAndVerifiedContentReopens(t *testing.T) {
	ctx, service, storageRoot, _ := setupIntake(t, func(_ context.Context, b []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "physical-glb-test-v1", ProfileVersion: 1, Supported: true, NodeCount: 3, MeshCount: 2, TriangleCount: 14}, nil
	})
	data := intakeFixtureGLB()
	staged, err := service.BeginImport(ctx, intakeClaims("candidate-1"), "phone source.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	got, err := staged.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if got.ID != "candidate-1" || got.Revision != 1 || got.OriginalSHA256 != hex.EncodeToString(sum[:]) || got.SizeBytes != int64(len(data)) || got.ReviewStatus != "unreviewed" || got.RenderStatus != "not_reviewed" || !got.Inspection.Supported {
		t.Fatalf("unexpected intake projection: %+v", got)
	}
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(body, []byte("blob_key")) || bytes.Contains(body, []byte(storageRoot)) {
		t.Fatalf("public projection leaked storage details: %s", body)
	}
	reloaded, err := service.Get(ctx, got.ID, got.Revision)
	if err != nil || reloaded.RevisionSHA256 != got.RevisionSHA256 {
		t.Fatalf("get = %+v, %v", reloaded, err)
	}
	rows, err := service.List(ctx, "", 10)
	if err != nil || len(rows) != 1 || rows[0].ID != got.ID {
		t.Fatalf("list = %+v, %v", rows, err)
	}
	opened, reader, err := service.Open(ctx, got.ID, got.Revision)
	if err != nil {
		t.Fatal(err)
	}
	read, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if opened.OriginalSHA256 != got.OriginalSHA256 || !bytes.Equal(read, data) {
		t.Fatal("verified original bytes did not survive close/reopen")
	}
	files, err := filepath.Glob(filepath.Join(storageRoot, "intake", "candidates", "candidate-1", "staged", "*.glb"))
	if err != nil || len(files) != 1 {
		t.Fatalf("expected one referenced staged object, got %v, %v", files, err)
	}
}

func TestIntakeExactRetryConvergesAndClaimConflictDoesNotReplaceHead(t *testing.T) {
	ctx, service, storageRoot, _ := setupIntake(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "test", ProfileVersion: 1, Supported: false, Finding: "subset_not_supported"}, nil
	})
	data := intakeFixtureGLB()
	claims := intakeClaims("candidate-replay")
	first, err := service.BeginImport(ctx, claims, "source.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	winner, err := first.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := service.BeginImport(ctx, claims, "source.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := retry.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.RevisionSHA256 != winner.RevisionSHA256 || replayed.Inspection.Supported || replayed.ReviewStatus != "unreviewed" {
		t.Fatalf("retry did not converge to unsupported candidate: %+v", replayed)
	}
	conflict := claims
	conflict.SourceLabel = "different source claim"
	staged, err := service.BeginImport(ctx, conflict, "source.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := staged.Commit(ctx); !errors.Is(err, ErrIntakeConflict) {
		t.Fatalf("conflicting identity error = %v", err)
	}
	if _, err := service.Get(ctx, winner.ID, 1); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(storageRoot, "intake", "candidates", "candidate-replay", "staged", "*.glb"))
	if err != nil || len(files) != 1 {
		t.Fatalf("unreferenced retry/conflict objects were not safely cleaned: %v, %v", files, err)
	}
}

func TestIntakeRejectsMalformedOrOversizedContainerBeforeMetadata(t *testing.T) {
	ctx, service, _, _ := setupIntake(t, func(context.Context, []byte) (IntakeInspection, error) {
		t.Fatal("malformed bytes reached inspector")
		return IntakeInspection{}, nil
	})
	bad := intakeFixtureGLB()
	bad[8] = 0
	if _, err := service.BeginImport(ctx, intakeClaims("candidate-bad"), "bad.glb", int64(len(bad)), bytes.NewReader(bad)); !errors.Is(err, ErrInvalidIntake) {
		t.Fatalf("malformed container error = %v", err)
	}
	if _, err := service.BeginImport(ctx, intakeClaims("candidate-large"), "large.glb", MaxGeometryBytes+1, strings.NewReader("x")); !errors.Is(err, ErrIntakeTooLarge) {
		t.Fatalf("oversize declaration error = %v", err)
	}
	if _, err := service.Get(ctx, "candidate-bad", 1); !errors.Is(err, ErrIntakeNotFound) {
		t.Fatalf("malformed upload created metadata: %v", err)
	}
}

func TestIntakeRetainsStagedBytesWhenMetadataCommitOutcomeCannotBeReadBack(t *testing.T) {
	ctx, service, storageRoot, db := setupIntakeWithDB(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "test", ProfileVersion: 1, Supported: true}, nil
	})
	data := intakeFixtureGLB()
	staged, err := service.BeginImport(ctx, intakeClaims("candidate-uncertain"), "source.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_intake_write BEFORE INSERT ON library_intake_candidate_revisions BEGIN SELECT RAISE(ABORT, 'injected metadata failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := staged.Commit(ctx); err == nil {
		t.Fatal("injected metadata failure was not returned")
	}
	files, err := filepath.Glob(filepath.Join(storageRoot, "intake", "candidates", "candidate-uncertain", "staged", "*.glb"))
	if err != nil || len(files) != 1 {
		t.Fatalf("staged object was deleted despite uncertain metadata outcome: %v, %v", files, err)
	}
}

func TestIntakeAbortAndTamperedStoredContentNeverPublishUnverifiedBytes(t *testing.T) {
	ctx, service, storageRoot, _ := setupIntakeWithDB(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "test", ProfileVersion: 1, Supported: true}, nil
	})
	data := intakeFixtureGLB()
	aborted, err := service.BeginImport(ctx, intakeClaims("candidate-abort"), "source.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := aborted.Abort(ctx); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(storageRoot, "intake", "candidates", "candidate-abort", "staged", "*.glb"))
	if err != nil || len(files) != 0 {
		t.Fatalf("abort retained an unreferenced stage: %v, %v", files, err)
	}

	staged, err := service.BeginImport(ctx, intakeClaims("candidate-tamper"), "source.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := staged.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	files, err = filepath.Glob(filepath.Join(storageRoot, "intake", "candidates", "candidate-tamper", "staged", "*.glb"))
	if err != nil || len(files) != 1 {
		t.Fatalf("committed object not found: %v, %v", files, err)
	}
	if err := os.WriteFile(files[0], bytes.Repeat([]byte{'x'}, len(data)), 0o600); err != nil {
		t.Fatal(err)
	}
	_, reader, err := service.Open(ctx, candidate.ID, candidate.Revision)
	if err != nil {
		t.Fatal(err)
	}
	read, readErr := io.ReadAll(reader)
	if readErr == nil || len(read) == len(data) {
		t.Fatalf("tampered body was fully published: bytes=%d err=%v", len(read), readErr)
	}
	_ = reader.Close()
}

func TestIntakeConcurrentExactImportsConvergeToOneRevision(t *testing.T) {
	ctx, service, storageRoot, _ := setupIntakeWithDB(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "test", ProfileVersion: 1, Supported: true}, nil
	})
	data, claims := intakeFixtureGLB(), intakeClaims("candidate-concurrent")
	var wg sync.WaitGroup
	results := make(chan IntakeCandidate, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			staged, err := service.BeginImport(ctx, claims, "source.glb", int64(len(data)), bytes.NewReader(data))
			if err != nil {
				errs <- err
				return
			}
			candidate, err := staged.Commit(ctx)
			if err != nil {
				errs <- err
				return
			}
			results <- candidate
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var digest string
	for candidate := range results {
		if digest != "" && candidate.RevisionSHA256 != digest {
			t.Fatal("concurrent exact imports returned different revisions")
		}
		digest = candidate.RevisionSHA256
	}
	if digest == "" {
		t.Fatal("concurrent imports returned no candidate")
	}
	files, err := filepath.Glob(filepath.Join(storageRoot, "intake", "candidates", "candidate-concurrent", "staged", "*.glb"))
	if err != nil || len(files) != 1 {
		t.Fatalf("concurrent replay left %d staged objects: %v", len(files), err)
	}
}

type commitThenErrorRepository struct{ intakeRevisionRepository }

func (r commitThenErrorRepository) Save(ctx context.Context, proposed revisionstore.Record[intakeRecord], expected uint32) (revisionstore.Record[intakeRecord], error) {
	row, err := r.intakeRevisionRepository.Save(ctx, proposed, expected)
	if err != nil {
		return row, err
	}
	return revisionstore.Record[intakeRecord]{}, errors.New("injected lost commit acknowledgement")
}

func TestIntakeAmbiguousCommitReadbackRetainsTheReferencedBlob(t *testing.T) {
	ctx, service, storageRoot, _ := setupIntakeWithDB(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "test", ProfileVersion: 1, Supported: true}, nil
	})
	data := intakeFixtureGLB()
	staged, err := service.BeginImport(ctx, intakeClaims("candidate-ambiguous"), "source.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	service.repo = commitThenErrorRepository{service.repo}
	candidate, err := staged.Commit(ctx)
	if err != nil {
		t.Fatalf("readback should resolve the lost acknowledgement: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(storageRoot, "intake", "candidates", "candidate-ambiguous", "staged", "*.glb"))
	if err != nil || len(files) != 1 {
		t.Fatalf("referenced original was deleted after ambiguous commit: %v, %v", files, err)
	}
	readCandidate, body, err := service.Open(ctx, candidate.ID, candidate.Revision)
	if err != nil {
		t.Fatal(err)
	}
	read, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if readCandidate.OriginalSHA256 != candidate.OriginalSHA256 || !bytes.Equal(read, data) {
		t.Fatal("committed candidate no longer resolves its original bytes")
	}
}

func TestIntakeParentLineageMustResolveTheExactCandidateDigest(t *testing.T) {
	ctx, service, _, _ := setupIntakeWithDB(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "test", ProfileVersion: 1, Supported: true}, nil
	})
	data := intakeFixtureGLB()
	parentStaged, err := service.BeginImport(ctx, intakeClaims("candidate-parent"), "parent.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	parent, err := parentStaged.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	childClaims := intakeClaims("candidate-child")
	childClaims.ParentID, childClaims.ParentRevision, childClaims.ParentSHA256 = parent.ID, parent.Revision, strings.Repeat("0", 64)
	if _, err := service.BeginImport(ctx, childClaims, "child.glb", int64(len(data)), bytes.NewReader(data)); !errors.Is(err, ErrInvalidIntake) {
		t.Fatalf("mismatched parent digest accepted: %v", err)
	}
	childClaims.ParentSHA256 = parent.RevisionSHA256
	child, err := service.BeginImport(ctx, childClaims, "child.glb", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	created, err := child.Commit(ctx)
	if err != nil || created.ParentSHA256 != parent.RevisionSHA256 {
		t.Fatalf("valid parent lineage = %+v, %v", created, err)
	}
}

func setupIntakeWithDB(t *testing.T, inspect IntakeInspector) (context.Context, *IntakeService, string, *sql.DB) {
	t.Helper()
	// Keep the SQLite pool concrete for commit-boundary fault injection.
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: t.TempDir()})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: dataRoot}, "intake-test", time.Minute); err != nil {
		t.Fatal(err)
	}
	store, err := routedblobstore.New(roots, "library")
	if err != nil {
		t.Fatal(err)
	}
	var next atomic.Int32
	service, err := NewIntakeService(db, store, inspect, func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) }, func() string { return fmt.Sprintf("stage-%d", next.Add(1)) })
	if err != nil {
		t.Fatal(err)
	}
	return database.WithTestMode(context.Background()), service, filepath.Join(dataRoot, "library"), db
}
