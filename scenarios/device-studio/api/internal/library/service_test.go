package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"device-studio/internal/routedblobstore"
	"github.com/vrooli/api-core/database"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func setupLibrary(t *testing.T) (context.Context, Service, *sql.DB, string) {
	t.Helper()
	db := databasetest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	leased := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: t.TempDir()})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: leased}, "library-test", time.Minute); err != nil {
		t.Fatal(err)
	}
	store, err := routedblobstore.New(roots, "device-studio-library")
	if err != nil {
		t.Fatal(err)
	}
	var next atomic.Int32
	service, err := NewService(NewSQLiteRepository(db), store, func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) }, func() string { return fmt.Sprintf("stage-%d", next.Add(1)) })
	if err != nil {
		t.Fatal(err)
	}
	return database.WithTestMode(context.Background()), service, db, filepath.Join(leased, "device-studio-library")
}

func fixtureInput(t *testing.T, bundle string, revision uint32, payload []byte) PublishInput {
	t.Helper()
	manifest := BundleManifest{Version: ManifestVersion, BundleID: bundle, Revision: revision, Artifacts: []Artifact{{
		ID: "geometry-original", Role: RoleGeometryGLB, MIME: "model/gltf-binary", SizeBytes: int64(len(payload)), SHA256: digest(payload),
		LicenseID: "CC-BY-4.0", Attribution: "Example author, source record E3-test",
		Provenance: ArtifactProvenance{Kind: ProvenanceAcquired, SourceLabel: "qualified fixture source", SourceSHA256: digest(payload)},
		Rights:     []Use{UseRender, UseEdit},
	}}}
	canonical, _, manifestSHA, err := canonicalManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	review := ReviewBinding{ID: "review-1", ReviewerID: "operator-1", ReviewedAt: time.Date(2026, 10, 20, 11, 0, 0, 0, time.UTC), Decision: "approved", ManifestSHA256: manifestSHA, AllowedUses: []Use{UseRender}}
	return PublishInput{Manifest: canonical, Review: review, Sources: map[string]io.Reader{"geometry-original": bytes.NewReader(payload)}}
}

func TestPublishIsImmutableReviewedAndExactReplayConverges(t *testing.T) {
	ctx, service, db, _ := setupLibrary(t)
	input := fixtureInput(t, "bundle-phone", 1, []byte("glTF binary fixture"))
	first, err := service.Publish(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ManifestSHA256 != first.Review.ManifestSHA256 || first.ManifestSHA256 == "" || len(first.artifactKeys) != 1 {
		t.Fatalf("published revision lost binding: %+v", first)
	}
	replayInput := input
	replayInput.Sources = nil
	replay, err := service.Publish(ctx, replayInput)
	if err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	if replay.ManifestSHA256 != first.ManifestSHA256 || replay.artifactKeys["geometry-original"] != first.artifactKeys["geometry-original"] {
		t.Fatalf("replay did not converge to stored revision")
	}
	changed := fixtureInput(t, "bundle-phone", 1, []byte("different bytes"))
	if _, err := service.Publish(ctx, changed); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("conflicting immutable revision error = %v", err)
	}
	var revisions, artifacts int
	if err := db.QueryRow(`SELECT count(*) FROM library_bundle_revisions`).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM library_bundle_artifacts`).Scan(&artifacts); err != nil {
		t.Fatal(err)
	}
	if revisions != 1 || artifacts != 1 {
		t.Fatalf("metadata rows after replay/conflict = %d/%d", revisions, artifacts)
	}
	if _, err := db.Exec(`UPDATE library_bundle_revisions SET manifest_sha256=? WHERE bundle_id=?`, strings.Repeat("0", 64), "bundle-phone"); err == nil {
		t.Fatal("immutable revision was mutable through SQL")
	}
}

func TestCatalogListUsesStableBundleRevisionCursorAndValidatesRows(t *testing.T) {
	ctx, service, _, _ := setupLibrary(t)
	for _, item := range []struct {
		id       string
		revision uint32
	}{{"bundle-a", 1}, {"bundle-a", 2}, {"bundle-b", 1}} {
		payload := []byte(fmt.Sprintf("%s-%d", item.id, item.revision))
		if _, err := service.Publish(ctx, fixtureInput(t, item.id, item.revision, payload)); err != nil {
			t.Fatal(err)
		}
	}
	first, err := service.ListRevisions(ctx, "", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].Manifest.BundleID != "bundle-a" || first[0].Manifest.Revision != 1 || first[1].Manifest.BundleID != "bundle-a" || first[1].Manifest.Revision != 2 {
		t.Fatalf("first catalog page is not stable bundle/revision order: %+v", first)
	}
	second, err := service.ListRevisions(ctx, first[1].Manifest.BundleID, first[1].Manifest.Revision, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].Manifest.BundleID != "bundle-b" || second[0].Manifest.Revision != 1 {
		t.Fatalf("second catalog page cursor skipped or duplicated entries: %+v", second)
	}
	if _, err := service.ListRevisions(ctx, "", 1, 2); err == nil {
		t.Fatal("revision without a bundle cursor was accepted")
	}
	if _, err := service.ListRevisions(ctx, "", 0, 101); err == nil {
		t.Fatal("oversized catalog page was accepted")
	}
}

func TestResolveChecksBothReviewAndArtifactRightsAndVerifiesStream(t *testing.T) {
	ctx, service, _, root := setupLibrary(t)
	payload := []byte("reviewed geometry bytes")
	revision, err := service.Publish(ctx, fixtureInput(t, "bundle-resolve", 2, payload))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenArtifact(ctx, "bundle-resolve", 2, "geometry-original", UseRedistribute); !errors.Is(err, ErrUseDenied) {
		t.Fatalf("review-denied use = %v", err)
	}
	resolved, err := service.OpenArtifact(ctx, "bundle-resolve", 2, "geometry-original", UseRender)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(resolved.Reader)
	closeErr := resolved.Reader.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(data, payload) {
		t.Fatalf("verified resolve = %q, read %v close %v", data, readErr, closeErr)
	}

	key := revision.artifactKeys["geometry-original"]
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(key)), []byte("reviewed geometry bytez"), 0o600); err != nil {
		t.Fatal(err)
	}
	corrupt, err := service.OpenArtifact(ctx, "bundle-resolve", 2, "geometry-original", UseRender)
	if err != nil {
		t.Fatal(err)
	}
	got, readErr := io.ReadAll(corrupt.Reader)
	if readErr == nil || bytes.HasSuffix(got, []byte("z")) {
		t.Fatalf("corrupt stored bytes were accepted/exposed: %q %v", got, readErr)
	}
	_ = corrupt.Reader.Close()
}

func TestRejectedBytesNeverPublishAndReviewMustBindCanonicalManifest(t *testing.T) {
	ctx, service, db, _ := setupLibrary(t)
	bad := fixtureInput(t, "bundle-bad", 1, []byte("truthful bytes"))
	bad.Sources["geometry-original"] = strings.NewReader("corrupt bytes!")
	if _, err := service.Publish(ctx, bad); err == nil {
		t.Fatal("corrupt candidate bytes were published")
	}
	if _, err := service.GetRevision(ctx, "bundle-bad", 1); !errors.Is(err, ErrRevisionNotFound) {
		t.Fatalf("bad upload became selectable: %v", err)
	}
	valid := fixtureInput(t, "bundle-review", 1, []byte("valid geometry"))
	valid.Review.ManifestSHA256 = strings.Repeat("a", 64)
	if _, err := service.Publish(ctx, valid); err == nil {
		t.Fatal("review for another manifest was accepted")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM library_bundle_revisions`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invalid candidate created %d metadata rows", count)
	}
	for _, invalid := range []Use{UseExternalAIUpload, Use("unknown")} {
		candidate := fixtureInput(t, "bundle-use-"+string(invalid), 1, []byte("g"))
		candidate.Review.AllowedUses = []Use{invalid}
		_, _, sha, err := canonicalManifest(candidate.Manifest)
		if err != nil {
			t.Fatal(err)
		}
		candidate.Review.ManifestSHA256 = sha
		if _, err := service.Publish(ctx, candidate); err == nil {
			t.Errorf("unsupported use %q accepted", invalid)
		}
	}
}

type commitThenError struct {
	Repository
	once sync.Once
}

func (r *commitThenError) CreateRevision(ctx context.Context, revision BundleRevision) error {
	var err error
	r.once.Do(func() { err = r.Repository.CreateRevision(ctx, revision) })
	if err != nil {
		return err
	}
	return errors.New("simulated lost commit acknowledgement")
}

func TestCommitAcknowledgementLossUsesExactReadbackWithoutDeletingObjects(t *testing.T) {
	ctx, service, db, root := setupLibrary(t)
	wrapped, err := NewService(&commitThenError{Repository: NewSQLiteRepository(db)}, service.store, nil, func() string { return "uncertain-stage" })
	if err != nil {
		t.Fatal(err)
	}
	input := fixtureInput(t, "bundle-uncertain", 1, []byte("durable bytes"))
	got, err := wrapped.Publish(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(got.artifactKeys["geometry-original"]))
	if data, err := os.ReadFile(path); err != nil || !bytes.Equal(data, []byte("durable bytes")) {
		t.Fatalf("referenced blob after lost acknowledgement = %q, %v", data, err)
	}
}

type committedReadbackFailure struct {
	Repository
	committed bool
}

func (r *committedReadbackFailure) CreateRevision(ctx context.Context, revision BundleRevision) error {
	if err := r.Repository.CreateRevision(ctx, revision); err != nil {
		return err
	}
	r.committed = true
	return errors.New("commit acknowledgement lost")
}

func (r *committedReadbackFailure) GetRevision(ctx context.Context, bundleID string, revision uint32) (BundleRevision, error) {
	if r.committed {
		return BundleRevision{}, errors.New("readback unavailable")
	}
	return r.Repository.GetRevision(ctx, bundleID, revision)
}

func TestAmbiguousReadbackFailureRetainsCommittedAndStagedObjects(t *testing.T) {
	ctx, service, db, root := setupLibrary(t)
	repo := &committedReadbackFailure{Repository: NewSQLiteRepository(db)}
	uncertain, err := NewService(repo, service.store, nil, func() string { return "uncertain-readback-stage" })
	if err != nil {
		t.Fatal(err)
	}
	input := fixtureInput(t, "bundle-readback-unknown", 1, []byte("keep referenced bytes"))
	if _, err := uncertain.Publish(ctx, input); err == nil || !strings.Contains(err.Error(), "readback unavailable") {
		t.Fatalf("ambiguous readback error = %v", err)
	}
	stored, err := NewSQLiteRepository(db).GetRevision(ctx, "bundle-readback-unknown", 1)
	if err != nil {
		t.Fatalf("committed row missing: %v", err)
	}
	for _, key := range stored.artifactKeys {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(key)))
		if err != nil || !bytes.Equal(data, []byte("keep referenced bytes")) {
			t.Fatalf("ambiguous outcome removed or damaged referenced object: %q %v", data, err)
		}
	}
}

func TestConcurrentExactPublicationConvergesToOneRevision(t *testing.T) {
	ctx, service, db, _ := setupLibrary(t)
	other, err := NewService(NewSQLiteRepository(db), service.store, nil, func() string { return "parallel-stage" })
	if err != nil {
		t.Fatal(err)
	}
	inputA := fixtureInput(t, "bundle-race", 1, []byte("same reviewed bytes"))
	inputB := fixtureInput(t, "bundle-race", 1, []byte("same reviewed bytes"))
	services := []Service{service, other}
	inputs := []PublishInput{inputA, inputB}
	results := make([]BundleRevision, 2)
	errs := make([]error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range services {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; results[i], errs[i] = services[i].Publish(ctx, inputs[i]) }(i)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("publisher %d failed: %v", i, err)
		}
	}
	if results[0].ManifestSHA256 != results[1].ManifestSHA256 {
		t.Fatal("concurrent exact publishers did not converge")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM library_bundle_revisions WHERE bundle_id='bundle-race'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("revision rows=%d err=%v", count, err)
	}
}

func TestConcurrentConflictingPublishersKeepOnlyWinnerReference(t *testing.T) {
	ctx, service, db, root := setupLibrary(t)
	var ids atomic.Int32
	other, err := NewService(NewSQLiteRepository(db), service.store, nil, func() string { return fmt.Sprintf("other-%d", ids.Add(1)) })
	if err != nil {
		t.Fatal(err)
	}
	inputs := []PublishInput{
		fixtureInput(t, "bundle-conflict-race", 1, []byte("candidate A bytes")),
		fixtureInput(t, "bundle-conflict-race", 1, []byte("candidate B bytes")),
	}
	services := []Service{service, other}
	results := make([]BundleRevision, 2)
	errs := make([]error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range services {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; results[i], errs[i] = services[i].Publish(ctx, inputs[i]) }(i)
	}
	close(start)
	wg.Wait()
	successes, conflicts := 0, 0
	for _, err := range errs {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrRevisionConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent publication error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successful publishers=%d conflicting publishers=%d errors=%v", successes, conflicts, errs)
	}
	winner, err := service.GetRevision(ctx, "bundle-conflict-race", 1)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM library_bundle_revisions WHERE bundle_id='bundle-conflict-race'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("revision rows=%d err=%v", count, err)
	}
	key := winner.artifactKeys["geometry-original"]
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(key)))
	if err != nil || digest(data) != winner.Manifest.Artifacts[0].SHA256 {
		t.Fatalf("winning immutable object invalid: %v", err)
	}
}

func TestVerifiedReaderWithholdsLastByteOnCorruptionAndRejectsEarlyClose(t *testing.T) {
	good := []byte("bounded stream")
	bad := []byte("bounded strean")
	reader := &verifiedReader{reader: io.NopCloser(bytes.NewReader(bad)), expectedSize: int64(len(good)), expectedSHA: digest(good), maxSize: 100, hash: sha256.New()}
	read, err := io.ReadAll(reader)
	if err == nil || bytes.HasSuffix(read, bad[len(bad)-1:]) {
		t.Fatalf("corrupt final byte was exposed: %q, %v", read, err)
	}
	_ = reader.Close()
	reader = &verifiedReader{reader: io.NopCloser(bytes.NewReader(good)), expectedSize: int64(len(good)), expectedSHA: digest(good), maxSize: 100, hash: sha256.New()}
	if _, err := reader.Read(make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); !errors.Is(err, ErrArtifactUnverified) {
		t.Fatalf("partial close = %v", err)
	}
}

func TestUploadVerifierRejectsTruncatedAndOversizedStreams(t *testing.T) {
	for _, tc := range []struct {
		name     string
		source   []byte
		expected []byte
	}{
		{name: "truncated", source: []byte("short"), expected: []byte("shorter")},
		{name: "oversized", source: []byte("oversized"), expected: []byte("small")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := &uploadVerifier{reader: bytes.NewReader(tc.source), expectedSize: int64(len(tc.expected)), maxSize: 64, expectedSHA: digest(tc.expected), hash: sha256.New()}
			_, err := io.Copy(io.Discard, v)
			if err == nil || v.verified() {
				t.Fatalf("invalid input accepted; err=%v", err)
			}
		})
	}
}

func TestCanonicalManifestSortingAndDigestAreStable(t *testing.T) {
	input := fixtureInput(t, "bundle-canonical", 1, []byte("geometry"))
	first, firstBytes, firstSHA, err := canonicalManifest(input.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	reversed := first
	reversed.Artifacts = append([]Artifact(nil), first.Artifacts...)
	_, secondBytes, secondSHA, err := canonicalManifest(reversed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) || firstSHA != secondSHA {
		t.Fatal("manifest canonicalization changed stable digest")
	}
	if _, err := hex.DecodeString(firstSHA); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalManifestExportAndProvenanceConsistency(t *testing.T) {
	payload := []byte("original bytes")
	manifest := fixtureInput(t, "bundle-provenance", 1, payload).Manifest
	canonical, data, sha, err := CanonicalizeManifest(manifest)
	if err != nil || len(data) == 0 || sha != digest(data) {
		t.Fatalf("exported canonical manifest = %q %s %v", data, sha, err)
	}
	if canonical.Artifacts[0].Provenance.SourceSHA256 != canonical.Artifacts[0].SHA256 {
		t.Fatal("acquired provenance must bind the uploaded artifact digest")
	}
	manifest.Artifacts[0].Provenance.SourceSHA256 = digest([]byte("different acquired bytes"))
	if _, _, _, err := CanonicalizeManifest(manifest); err == nil {
		t.Fatal("acquired provenance with a different source digest was accepted")
	}
	manifest.Artifacts[0].Provenance.SourceSHA256 = manifest.Artifacts[0].SHA256
	child := Artifact{ID: "derived", Role: RoleGeometryGLB, MIME: "model/gltf-binary", SizeBytes: 1, SHA256: digest([]byte("x")), LicenseID: "CC0", Attribution: "test", Provenance: ArtifactProvenance{Kind: ProvenanceAdapted, SourceLabel: "adaptation", SourceSHA256: digest([]byte("wrong")), DerivedFromArtifactID: "geometry-original", Adaptation: "converted"}, Rights: []Use{UseRender}}
	manifest.Artifacts = append(manifest.Artifacts, child)
	if _, _, _, err := CanonicalizeManifest(manifest); err == nil {
		t.Fatal("adapted source digest inconsistent with referenced artifact was accepted")
	}
	manifest.Artifacts[1].Provenance = ArtifactProvenance{Kind: ProvenanceAdapted, SourceLabel: "self", DerivedFromArtifactID: "derived", Adaptation: "converted"}
	if _, _, _, err := CanonicalizeManifest(manifest); err == nil {
		t.Fatal("self-referential artifact provenance was accepted")
	}
	manifest.Artifacts[0].Provenance = ArtifactProvenance{Kind: ProvenanceAdapted, SourceLabel: "cycle", DerivedFromArtifactID: "derived", Adaptation: "converted"}
	manifest.Artifacts[1].Provenance = ArtifactProvenance{Kind: ProvenanceAdapted, SourceLabel: "cycle", DerivedFromArtifactID: "geometry-original", Adaptation: "converted"}
	if _, _, _, err := CanonicalizeManifest(manifest); err == nil {
		t.Fatal("cyclic artifact provenance was accepted")
	}
}

func TestStoredMetadataBoundedBeforeDecode(t *testing.T) {
	if err := decodeCanonical(bytes.Repeat([]byte(" "), MaxStoredManifestBytes+1), &BundleManifest{}); err == nil {
		t.Fatal("oversized stored JSON was decoded")
	}
}

func TestVerificationReadersHonorCancellationBetweenReads(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	underlying := &countingReader{r: bytes.NewReader([]byte("content"))}
	reader := &verifiedReader{ctx: ctx, reader: io.NopCloser(underlying), expectedSize: 7, expectedSHA: digest([]byte("content")), maxSize: 10, hash: sha256.New()}
	buf := make([]byte, 2)
	if _, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := reader.Read(buf); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled read error = %v", err)
	}
	if underlying.reads != 1 {
		t.Fatalf("reader performed %d reads after cancellation", underlying.reads)
	}
	_ = reader.Close()
}

type countingReader struct {
	r     io.Reader
	reads int
}

func (r *countingReader) Read(p []byte) (int, error) { r.reads++; return r.r.Read(p) }
