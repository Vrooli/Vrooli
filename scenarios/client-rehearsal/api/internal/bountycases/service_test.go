package bountycases

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLiteRegistryResolvesImmutableBundleByExactIDOrIssueURL(t *testing.T) {
	root := t.TempDir()
	bundle := testBundle(t, root)
	db, repo, svc := testService(t, root)
	defer db.Close()

	registered, err := svc.Register(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if !validSHA256(registered.BundleDigest) {
		t.Fatalf("bundle digest = %q", registered.BundleDigest)
	}
	byID, err := svc.Resolve(context.Background(), Lookup{CaseID: bundle.CaseID})
	if err != nil {
		t.Fatal(err)
	}
	byURL, err := svc.Resolve(context.Background(), Lookup{PublicURL: bundle.PublicURL})
	if err != nil {
		t.Fatal(err)
	}
	if byID.BundleDigest != registered.BundleDigest || byURL.BundleDigest != registered.BundleDigest || byURL.ReferenceCommit != bundle.ReferenceCommit {
		t.Fatalf("resolved bundle mismatch: byID=%+v byURL=%+v", byID, byURL)
	}
	repeated, err := svc.Register(context.Background(), bundle)
	if err != nil || repeated.BundleDigest != registered.BundleDigest {
		t.Fatalf("idempotent re-registration = %+v, err=%v", repeated, err)
	}
	conflict := bundle
	conflict.Task += " changed"
	if _, err := svc.Register(context.Background(), conflict); err == nil {
		t.Fatal("changed content at immutable version unexpectedly succeeded")
	} else {
		var conflict ErrConflict
		if !errors.As(err, &conflict) {
			t.Fatalf("changed registration error = %v, want ErrConflict", err)
		}
	}
	version2 := registered
	version2.BundleDigest = ""
	version2.BundleVersion = 2
	version2.Task += " Updated requirements retained in a new immutable version."
	version2, err = svc.Register(context.Background(), version2)
	if err != nil {
		t.Fatalf("append bundle version 2: %v", err)
	}
	latest, err := svc.Resolve(context.Background(), Lookup{PublicURL: bundle.PublicURL})
	if err != nil || latest.BundleVersion != 2 || latest.Task != version2.Task {
		t.Fatalf("latest version = %+v, err=%v", latest, err)
	}
	if _, err := svc.Resolve(context.Background(), Lookup{PublicURL: bundle.PublicURL + "/"}); err == nil {
		t.Fatal("non-exact public URL resolved")
	} else {
		var invalid ErrInvalid
		if !errors.As(err, &invalid) {
			t.Fatalf("trailing slash lookup error = %v, want exact URL rejection", err)
		}
	}
	_ = repo
}

func TestSolverProjectionContainsNoReferenceOrEvaluatorMaterial(t *testing.T) {
	root := t.TempDir()
	bundle := testBundle(t, root)
	_, _, svc := testService(t, root)
	projection := svc.ProjectForSolver(bundle)
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, hidden := range []string{bundle.ReferenceProjectRoot, bundle.FrozenEvaluatorRoot, bundle.ReferenceCommit, bundle.FrozenEvaluatorDigest, bundle.PullRequestURL, bundle.TestGenieProjectID} {
		if strings.Contains(text, hidden) {
			t.Fatalf("solver projection leaks hidden bundle value %q: %s", hidden, text)
		}
	}
	for _, forbiddenField := range []string{"referenceProjectRoot", "referenceCommit", "frozenEvaluator", "testGeniePlan", "pullRequestUrl", "bundleDigest"} {
		if strings.Contains(text, forbiddenField) {
			t.Fatalf("solver projection contains hidden field %q: %s", forbiddenField, text)
		}
	}
}

func TestResolveRejectsAmbiguousLookupAndCallerControlledPaths(t *testing.T) {
	root := t.TempDir()
	_, _, svc := testService(t, root)
	for _, lookup := range []Lookup{{}, {CaseID: "case-one", PublicURL: "https://github.com/acme/demo/issues/1"}} {
		if _, err := svc.Resolve(context.Background(), lookup); err == nil {
			t.Fatalf("lookup %+v unexpectedly succeeded", lookup)
		} else {
			var invalid ErrInvalid
			if !errors.As(err, &invalid) {
				t.Fatalf("lookup error = %v, want ErrInvalid", err)
			}
		}
	}
	// The public request shape has no path, intent, or evaluator fields. A
	// caller cannot choose a filesystem root through this service interface.
	if _, exists := reflectField(Lookup{}, "ReferenceProjectRoot"); exists {
		t.Fatal("lookup unexpectedly has a reference root field")
	}
}

func TestRegisterRejectsUnsafeWorkspaceRootsAndMismatchedIdentities(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	bundle := testBundle(t, root)
	_, _, svc := testService(t, root)

	bad := bundle
	bad.SourceProjectRoot = outside
	if _, err := svc.Register(context.Background(), bad); err == nil {
		t.Fatal("outside source root unexpectedly registered")
	}
	bad = bundle
	bad.PullRequestNumber++
	if _, err := svc.Register(context.Background(), bad); err == nil {
		t.Fatal("mismatched PR identity unexpectedly registered")
	}
	bad = bundle
	bad.FrozenEvaluatorRoot = filepath.Join(root, "missing")
	if _, err := svc.Register(context.Background(), bad); err == nil {
		t.Fatal("missing evaluator root unexpectedly registered")
	}
	bad = bundle
	bad.PublicURL = "https://github.com/acme/demo/issues/1?include=anything"
	if _, err := svc.Register(context.Background(), bad); err == nil {
		t.Fatal("query-bearing URL unexpectedly registered")
	}
}

func TestRegisterRejectsGenericTestGeniePhasesForFrozenReplay(t *testing.T) {
	root := t.TempDir()
	bundle := testBundle(t, root)
	bundle.TestGeniePlan.Phases = []string{"unit"}
	_, _, svc := testService(t, root)

	if _, err := svc.Register(context.Background(), bundle); err == nil || !strings.Contains(err.Error(), "cannot select generic Test Genie phases") {
		t.Fatalf("generic Test Genie phase selection error = %v, want explicit frozen-replay rejection", err)
	}
}

func TestResolveVerifiesBundleContentDigest(t *testing.T) {
	root := t.TempDir()
	bundle := testBundle(t, root)
	db, _, svc := testService(t, root)
	defer db.Close()
	if _, err := svc.Register(context.Background(), bundle); err != nil {
		t.Fatal(err)
	}
	modified := bundle
	modified.Task = "tampered hidden task"
	data, err := json.Marshal(modified)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TRIGGER bounty_replay_cases_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE bounty_replay_cases SET bundle_json = ? WHERE case_id = ?`, string(data), bundle.CaseID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve(context.Background(), Lookup{CaseID: bundle.CaseID}); err == nil {
		t.Fatal("tampered bundle unexpectedly resolved")
	} else {
		var corrupt ErrCorrupt
		if !errors.As(err, &corrupt) {
			t.Fatalf("tamper error = %v, want ErrCorrupt", err)
		}
	}
}

func TestResolveDetectsEvaluatorFileMutation(t *testing.T) {
	root := t.TempDir()
	bundle := testBundle(t, root)
	db, _, svc := testService(t, root)
	defer db.Close()
	if _, err := svc.Register(context.Background(), bundle); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle.FrozenEvaluatorRoot, bundle.FrozenEvaluatorEntrypoint), []byte("tampered evaluator"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve(context.Background(), Lookup{CaseID: bundle.CaseID}); err == nil {
		t.Fatal("changed evaluator bytes unexpectedly resolved")
	} else {
		var corrupt ErrCorrupt
		if !errors.As(err, &corrupt) {
			t.Fatalf("evaluator mutation error = %v, want ErrCorrupt", err)
		}
	}
}

func TestGetFrozenEvaluatorBundleUsesOpaqueVersionedIDAndBoundedFiles(t *testing.T) {
	root := t.TempDir()
	bundle := testBundle(t, root)
	if err := os.Mkdir(filepath.Join(bundle.FrozenEvaluatorRoot, "lib"), 0o700); err != nil {
		t.Fatal(err)
	}
	dependency := []byte("exports.run = () => true;\n")
	if err := os.WriteFile(filepath.Join(bundle.FrozenEvaluatorRoot, "lib", "runner.cjs"), dependency, 0o600); err != nil {
		t.Fatal(err)
	}
	db, _, svc := testService(t, root)
	defer db.Close()
	registered, err := svc.Register(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	id := evaluatorBundleID(registered)
	got, err := svc.ResolveEvaluatorBundle(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.EvaluatorBundleID != id || got.CaseID != bundle.CaseID || got.BundleVersion != 1 || got.Entrypoint != "acceptance.cjs" || len(got.Files) != 2 {
		t.Fatalf("bundle projection = %+v", got)
	}
	for _, file := range got.Files {
		if filepath.IsAbs(file.RelativePath) || strings.Contains(file.RelativePath, "..") || !validSHA256(file.SHA256) {
			t.Fatalf("unsafe evaluator file entry: %+v", file)
		}
	}
	if strings.Contains(string(mustJSON(t, got)), bundle.FrozenEvaluatorRoot) {
		t.Fatal("evaluator-byte response disclosed its host root")
	}
}

func TestBundleDigestUsesDeterministicTypedJSONExcludingDigestField(t *testing.T) {
	root := t.TempDir()
	bundle := testBundle(t, root)
	_, _, svc := testService(t, root)
	registered, err := svc.Register(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := canonicalBundleBytes(registered)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(bytes)
	if registered.BundleDigest != hex.EncodeToString(want[:]) {
		t.Fatalf("digest=%s want=%x", registered.BundleDigest, want)
	}
}

func TestLoadDirectoryImportsVersionedJSONIdempotently(t *testing.T) {
	root := t.TempDir()
	registrations := filepath.Join(root, "registrations")
	if err := os.MkdirAll(registrations, 0o700); err != nil {
		t.Fatal(err)
	}
	bundle := testBundle(t, root)
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(registrations, "example-v0001.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	db, _, svc := testService(t, root)
	defer db.Close()
	for attempt := 0; attempt < 2; attempt++ {
		loaded, err := LoadDirectory(context.Background(), svc, registrations)
		if err != nil || len(loaded) != 1 || loaded[0].BundleDigest == "" {
			t.Fatalf("load attempt %d: loaded=%+v err=%v", attempt+1, loaded, err)
		}
	}
}

func TestRegisteredVersionsRejectUpdateAndDelete(t *testing.T) {
	root := t.TempDir()
	bundle := testBundle(t, root)
	db, _, svc := testService(t, root)
	defer db.Close()
	registered, err := svc.Register(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE bounty_replay_cases SET bundle_json = '' WHERE case_id = ?`, bundle.CaseID); err == nil {
		t.Fatal("immutable case version unexpectedly accepted an UPDATE")
	}
	if _, err := db.Exec(`DELETE FROM bounty_replay_cases WHERE case_id = ?`, bundle.CaseID); err == nil {
		t.Fatal("immutable case version unexpectedly accepted a DELETE")
	}
	if _, err := svc.ResolveEvaluatorBundle(context.Background(), evaluatorBundleID(registered)); err != nil {
		t.Fatalf("rejected writes changed the registered bundle: %v", err)
	}
}

func testService(t *testing.T, root string) (*sql.DB, Repository, Service) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "cases.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(Schema()); err != nil {
		db.Close()
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	svc, err := NewService(repo, root)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db, repo, svc
}

func testBundle(t *testing.T, root string) CaseBundle {
	t.Helper()
	paths := make([]string, 3)
	for i, name := range []string{"source", "reference", "evaluator"} {
		paths[i] = filepath.Join(root, name)
		if err := os.MkdirAll(paths[i], 0o700); err != nil {
			t.Fatal(err)
		}
	}
	evaluator := []byte("module.exports = { run: () => true };\n")
	if err := os.WriteFile(filepath.Join(paths[2], "acceptance.cjs"), evaluator, 0o600); err != nil {
		t.Fatal(err)
	}
	evaluatorDigest := sha256.Sum256(evaluator)
	return CaseBundle{
		SchemaVersion: 1, BundleVersion: 1, CaseID: "orama-544", Project: "orama",
		Repository: "oramasearch/orama", Title: "Fix shared-prefix lookup", Task: "Make prefix search return correct records.",
		PublicURL: "https://github.com/oramasearch/orama/issues/544", IssueNumber: 544,
		PullRequestURL: "https://github.com/oramasearch/orama/pull/583", PullRequestNumber: 583,
		SourceCommit: "6d279527525533f1eb9f6a0714cae52bb9ddcf4f", ReferenceCommit: "701427dd2693c630595cf268a1ea0ae3b615357e",
		SourceProjectRoot: paths[0], ReferenceProjectRoot: paths[1], FrozenEvaluatorRoot: paths[2],
		FrozenEvaluatorDigest: hex.EncodeToString(evaluatorDigest[:]), FrozenEvaluatorEntrypoint: "acceptance.cjs", TestGenieProjectID: "f9-orama-544-v1",
		TestGeniePlan: TestGeniePlan{
			ValidationPurpose: "VALIDATION_PURPOSE_REGRESSION_CURRENT", RequiredStrength: "VALIDATION_STRENGTH_TARGETED", Phases: []string{},
			EvidencePolicy: ReplayEvidencePolicy{RequireSourceSnapshot: true, RequiredEvidenceKinds: []string{"test-genie-run"}},
			DeadlinePolicy: ReplayDeadlinePolicy{QueueBudgetMilliseconds: 5 * 60 * 1000, ExecutionBudgetMilliseconds: 20 * 60 * 1000, MaximumAttempts: 1},
		},
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func reflectField(value any, name string) (any, bool) {
	encoded, _ := json.Marshal(value)
	var fields map[string]any
	_ = json.Unmarshal(encoded, &fields)
	item, ok := fields[name]
	return item, ok
}
