package cloudtarget

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/privilegebroker"
)

const (
	testDeployment = "dep-1"
	testScenario   = "landing-app"
)

type bundleEntry struct {
	name     string
	body     string
	typeflag byte
	linkname string
}

func defaultBundleEntries() []bundleEntry {
	return []bundleEntry{
		{name: "scenarios/" + testScenario + "/.vrooli/service.json", body: `{"service":{"name":"landing-app"}}`},
		{name: "scenarios/" + testScenario + "/api/main.go", body: "package main\n"},
	}
}

func writeBundle(t *testing.T, dir string, entries []bundleEntry) string {
	t.Helper()
	path := filepath.Join(dir, "bundle.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := tar.Header{Name: e.name, Mode: 0o644, Typeflag: e.typeflag, Linkname: e.linkname}
		if hdr.Typeflag == 0 {
			hdr.Typeflag = tar.TypeReg
		}
		if hdr.Typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(&hdr); err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, closer := range []interface{ Close() error }{tw, gz, f} {
		if err := closer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func sha256Of(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type fixture struct {
	store      *Store
	manifest   Manifest
	manifestPB string
	archive    string
	executable string
}

func (f fixture) verify() VerifyRequest {
	return VerifyRequest{ManifestPath: f.manifestPB, ArchivePath: f.archive, Executable: f.executable, GOOS: "linux", GOARCH: "amd64"}
}

func (f fixture) effect(op, step string, fence uint64) EffectRequest {
	return EffectRequest{DeploymentID: testDeployment, OperationID: op, Step: step, Fence: fence}
}

func newFixture(t *testing.T, entries []bundleEntry, mutate func(*Manifest)) fixture {
	t.Helper()
	dir := t.TempDir()
	archive := writeBundle(t, dir, entries)
	executable := filepath.Join(dir, "vrooli")
	if err := os.WriteFile(executable, []byte("fake-native-cli-"+t.Name()), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		SchemaVersion: SchemaVersion, BundleSHA256: sha256Of(t, archive),
		NativeCLI:     NativeCLI{SHA256: sha256Of(t, executable), GOOS: "linux", GOARCH: "amd64"},
		ClosureDigest: strings.Repeat("c", 64), ConfigurationDigest: strings.Repeat("d", 64),
		Provenance: Provenance{Builder: "scenario-to-cloud", Policy: "trusted-builder-v1"},
		Limits:     Limits{MaxEntries: 100, MaxExpandedBytes: 1 << 20, MaxEntryBytes: 1 << 16},
	}
	digest, err := ComputeReleaseDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ReleaseDigest = digest
	if mutate != nil {
		mutate(&manifest)
	}
	manifestPath := filepath.Join(dir, "release-manifest.json")
	raw, _ := json.Marshal(manifest)
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return fixture{store: NewStore(filepath.Join(dir, "runtime-home", "cloud", "deployments")), manifest: manifest, manifestPB: manifestPath, archive: archive, executable: executable}
}

func codeOf(t *testing.T, err error) (string, int) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a typed error, got nil")
	}
	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("error %v is not a cloudtarget.Error", err)
	}
	return typed.Code, typed.ExitCode()
}

type recordingActivator struct {
	activations []Activation
	err         error
}

func (r *recordingActivator) Activate(_ context.Context, activation Activation) error {
	r.activations = append(r.activations, activation)
	return r.err
}

// [REQ:STC-P0-007] A target refuses any fence below the highest accepted one
// and records each accepted fence durably.
func TestRunEffectRefusesStaleFenceAndRecordsAcceptedFence(t *testing.T) {
	f := newFixture(t, defaultBundleEntries(), nil)
	body := func(context.Context) (map[string]any, Outcome, error) {
		return map[string]any{"ran": true}, OutcomeSucceeded, nil
	}
	ctx := context.Background()
	first := f.effect("op-1", "step-a", 5)
	first.Verb = "test.verb"
	if _, err := f.store.RunEffect(ctx, first, body); err != nil {
		t.Fatalf("first effect: %v", err)
	}
	fence, err := f.store.ReadFence(testDeployment)
	if err != nil || fence.Current != 5 || fence.OperationID != "op-1" {
		t.Fatalf("fence = %+v err=%v", fence, err)
	}
	cases := []struct {
		name  string
		fence uint64
		code  string
		exit  int
	}{
		{"lower fence refused", 4, CodeFenceStale, ExitRefused},
		{"equal fence accepted", 5, "", 0},
		{"higher fence accepted", 9, "", 0},
		{"then the old equal fence is stale", 5, CodeFenceStale, ExitRefused},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := f.effect("op-"+string(rune('b'+i)), "step", tc.fence)
			req.Verb = "test.verb"
			_, err := f.store.RunEffect(ctx, req, body)
			if tc.code == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			code, exit := codeOf(t, err)
			if code != tc.code || exit != tc.exit {
				t.Fatalf("code/exit = %s/%d want %s/%d", code, exit, tc.code, tc.exit)
			}
		})
	}
}

// [REQ:STC-P0-007] Re-running a (operation, step) replays its receipt
// unchanged; the body never runs twice, and different inputs are refused.
func TestRunEffectReplaysReceiptPerOperationStep(t *testing.T) {
	f := newFixture(t, defaultBundleEntries(), nil)
	ctx := context.Background()
	runs := 0
	body := func(context.Context) (map[string]any, Outcome, error) {
		runs++
		return map[string]any{"run": runs}, OutcomeSucceeded, nil
	}
	req := f.effect("op-1", "stage", 1)
	req.Verb = "test.verb"
	req.Input = map[string]any{"release": "abc"}
	first, err := f.store.RunEffect(ctx, req, body)
	if err != nil || first.Replayed {
		t.Fatalf("first = %+v err=%v", first, err)
	}
	second, err := f.store.RunEffect(ctx, req, body)
	if err != nil || !second.Replayed || !second.Receipt.Replayed || runs != 1 {
		t.Fatalf("second = %+v runs=%d err=%v", second, runs, err)
	}
	first.Receipt.Replayed = true
	if a, b := mustJSON(t, first.Receipt), mustJSON(t, second.Receipt); a != b {
		t.Fatalf("replayed receipt differs:\n%s\n%s", a, b)
	}
	stored, err := f.store.ReadReceipt(testDeployment, "op-1", "stage")
	if err != nil || stored.Outcome != OutcomeSucceeded || stored.SchemaVersion != SchemaVersion || stored.Verb != "test.verb" || stored.Fence != 1 {
		t.Fatalf("stored = %+v err=%v", stored, err)
	}
	req.Input = map[string]any{"release": "different"}
	if code, exit := codeOf(t, must2(f.store.RunEffect(ctx, req, body))); code != CodeReceiptInputMismatch || exit != ExitRefused {
		t.Fatalf("code/exit = %s/%d", code, exit)
	}
	failing := f.effect("op-1", "activate", 1)
	failing.Verb = "test.verb"
	result, err := f.store.RunEffect(ctx, failing, func(context.Context) (map[string]any, Outcome, error) {
		return nil, OutcomeFailed, refuse(CodeReleaseNotStaged, "nope")
	})
	if err == nil || result.Receipt.Outcome != OutcomeFailed || result.Receipt.Error == nil || result.Receipt.Error.Code != CodeReleaseNotStaged {
		t.Fatalf("failed receipt = %+v err=%v", result.Receipt, err)
	}
	if _, err := f.store.ReadReceipt(testDeployment, "op-1", "missing"); mustCode(t, err) != CodeReceiptNotFound {
		t.Fatalf("missing receipt error = %v", err)
	}
}

func must2(result EffectResult, err error) error { return err }

func mustCode(t *testing.T, err error) string {
	t.Helper()
	code, _ := codeOf(t, err)
	return code
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The release digest is a documented canonical-JSON digest the cloud side
// can compute independently.
func TestComputeReleaseDigestIsCanonical(t *testing.T) {
	m := Manifest{BundleSHA256: "b", NativeCLI: NativeCLI{SHA256: "s", GOOS: "linux", GOARCH: "arm64"}, ClosureDigest: "c", ConfigurationDigest: "k"}
	canonical := `{"bundle_sha256":"b","closure_digest":"c","configuration_digest":"k","native_cli":{"goarch":"arm64","goos":"linux","sha256":"s"}}`
	sum := sha256.Sum256([]byte(canonical))
	got, err := ComputeReleaseDigest(m)
	if err != nil || got != hex.EncodeToString(sum[:]) {
		t.Fatalf("digest = %s err=%v", got, err)
	}
	m.Provenance = Provenance{Builder: "other"}
	m.Limits = Limits{MaxEntries: 1}
	if again, _ := ComputeReleaseDigest(m); again != got {
		t.Fatal("provenance or limits changed the release identity")
	}
}

// [REQ:STC-P0-010] Tampered content, a mismatched native CLI, and malicious
// archives are refused before any write.
func TestVerifyRefusesTamperedAndMaliciousReleases(t *testing.T) {
	traversal := append(defaultBundleEntries(), bundleEntry{name: "../escape", body: "x"})
	symlink := append(defaultBundleEntries(), bundleEntry{name: "scenarios/link", typeflag: tar.TypeSymlink, linkname: "../../../etc"})
	device := append(defaultBundleEntries(), bundleEntry{name: "dev/null", typeflag: tar.TypeChar})
	many := defaultBundleEntries()
	for i := 0; i < 150; i++ {
		many = append(many, bundleEntry{name: "scenarios/" + testScenario + "/f" + strings.Repeat("x", i%9) + ".txt", body: "1"})
	}
	large := append(defaultBundleEntries(), bundleEntry{name: "scenarios/big.bin", body: strings.Repeat("A", (1<<16)+1)})
	cases := []struct {
		name    string
		entries []bundleEntry
		mutate  func(*Manifest)
		tamper  func(t *testing.T, f *fixture)
		code    string
	}{
		{name: "release digest mismatch", entries: defaultBundleEntries(), mutate: func(m *Manifest) { m.ReleaseDigest = strings.Repeat("0", 64) }, code: CodeReleaseDigestMismatch},
		{name: "tampered archive byte", entries: defaultBundleEntries(), tamper: func(t *testing.T, f *fixture) {
			raw, _ := os.ReadFile(f.archive)
			raw[len(raw)-1] ^= 0xff
			if err := os.WriteFile(f.archive, raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}, code: CodeBundleSHA256Mismatch},
		{name: "native cli digest mismatch", entries: defaultBundleEntries(), tamper: func(t *testing.T, f *fixture) {
			if err := os.WriteFile(f.executable, []byte("other build"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, code: CodeNativeCLIMismatch},
		{name: "native cli arch mismatch", entries: defaultBundleEntries(), mutate: func(m *Manifest) {
			m.NativeCLI.GOARCH = "arm64"
			m.ReleaseDigest, _ = ComputeReleaseDigest(*m)
		}, code: CodeNativeCLIMismatch},
		{name: "traversal", entries: traversal, code: CodeArchiveTraversal},
		{name: "escaping symlink", entries: symlink, code: CodeArchiveSymlinkEscape},
		{name: "device entry", entries: device, code: CodeArchiveUnsupportedEntry},
		{name: "entry count bomb", entries: many, code: CodeArchiveEntryLimit},
		{name: "oversized entry", entries: large, code: CodeArchiveTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, tc.entries, tc.mutate)
			if tc.tamper != nil {
				tc.tamper(t, &f)
			}
			report, _, err := Verify(context.Background(), f.verify())
			code, exit := codeOf(t, err)
			if code != tc.code || exit != ExitRefused {
				t.Fatalf("code/exit = %s/%d want %s/2 (report %+v)", code, exit, tc.code, report)
			}
			if report.Verified {
				t.Fatal("report claims verified after a refusal")
			}
			if _, err := os.Stat(f.store.Root); err == nil {
				t.Fatal("verify wrote into the deployment store")
			}
			result, err := f.store.Stage(context.Background(), StageRequest{Effect: f.effect("op-x", "stage", 1), Verify: f.verify()})
			if mustCode(t, err) != tc.code || result.Receipt.Outcome != OutcomeFailed {
				t.Fatalf("stage code = %v receipt=%+v", err, result.Receipt)
			}
			releases, _ := f.store.releasesDir(testDeployment)
			if entries, _ := os.ReadDir(releases); len(entries) != 0 {
				t.Fatalf("refused stage left %d entries under releases/", len(entries))
			}
		})
	}
}

func TestVerifyAcceptsMatchingRelease(t *testing.T) {
	f := newFixture(t, defaultBundleEntries(), nil)
	report, manifest, err := Verify(context.Background(), f.verify())
	if err != nil || !report.Verified || manifest.ReleaseDigest != f.manifest.ReleaseDigest || report.Archive.Files != 2 {
		t.Fatalf("report=%+v manifest=%+v err=%v", report, manifest, err)
	}
	if len(report.Checks) != 4 {
		t.Fatalf("checks = %+v", report.Checks)
	}
}

// [REQ:STC-P0-011] Staging is restart-safe: success promotes a complete
// tree, failure leaves nothing activatable, and an existing release is
// unchanged.
func TestStagePromotesOnlyCompleteTrees(t *testing.T) {
	f := newFixture(t, defaultBundleEntries(), nil)
	ctx := context.Background()
	result, err := f.store.Stage(ctx, StageRequest{Effect: f.effect("op-1", "stage", 1), Verify: f.verify()})
	if err != nil || result.Receipt.Outcome != OutcomeSucceeded {
		t.Fatalf("stage = %+v err=%v", result.Receipt, err)
	}
	releaseDir, _ := f.store.ReleaseDir(testDeployment, f.manifest.ReleaseDigest)
	if releaseState(releaseDir) != ReleaseStateComplete {
		t.Fatalf("release state = %s", releaseState(releaseDir))
	}
	if _, err := os.Stat(filepath.Join(releaseDir, "scenarios", testScenario, ".vrooli", "service.json")); err != nil {
		t.Fatalf("extracted tree missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(releaseDir, incompleteMarker)); err == nil {
		t.Fatal("promoted release still carries the incomplete marker")
	}
	again, err := f.store.Stage(ctx, StageRequest{Effect: f.effect("op-2", "stage", 2), Verify: f.verify()})
	if err != nil || again.Receipt.Outcome != OutcomeUnchanged {
		t.Fatalf("restage = %+v err=%v", again.Receipt, err)
	}
	listing, err := f.store.ListReleases(testDeployment)
	if err != nil || len(listing.Releases) != 1 || listing.Releases[0].State != ReleaseStateComplete || listing.Releases[0].Role != "staged" {
		t.Fatalf("listing = %+v err=%v", listing, err)
	}

	crashed := newFixture(t, defaultBundleEntries(), nil)
	crashed.store.Hooks.Fault = func(point string) error {
		if point == "stage:after_extract" {
			return errors.New("simulated crash after extract")
		}
		return nil
	}
	result, err = crashed.store.Stage(ctx, StageRequest{Effect: crashed.effect("op-1", "stage", 1), Verify: crashed.verify()})
	if err == nil || result.Receipt.Outcome != OutcomeFailed {
		t.Fatalf("crashed stage = %+v err=%v", result.Receipt, err)
	}
	releases, _ := crashed.store.releasesDir(testDeployment)
	if entries, _ := os.ReadDir(releases); len(entries) != 0 {
		t.Fatalf("failed stage left %d entries under releases/", len(entries))
	}
}

// [REQ:STC-P0-011] An incomplete stage is never activatable and activation
// commits the pointer only after the runtime owner succeeds.
func TestActivateRequiresCompleteReleaseAndCommitsAfterRuntime(t *testing.T) {
	f := newFixture(t, defaultBundleEntries(), nil)
	ctx := context.Background()
	activator := &recordingActivator{}
	missing := f.effect("op-0", "activate", 1)
	if code, exit := codeOf(t, must2(f.store.Activate(ctx, ActivateRequest{Effect: missing, Release: f.manifest.ReleaseDigest, Activator: activator}))); code != CodeReleaseNotStaged || exit != ExitRefused {
		t.Fatalf("missing release code/exit = %s/%d", code, exit)
	}
	releaseDir, _ := f.store.ReleaseDir(testDeployment, f.manifest.ReleaseDigest)
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, incompleteMarker), []byte("op"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := mustCode(t, must2(f.store.Activate(ctx, ActivateRequest{Effect: f.effect("op-0", "activate-2", 1), Release: f.manifest.ReleaseDigest, Activator: activator}))); code != CodeReleaseIncomplete {
		t.Fatalf("incomplete release code = %s", code)
	}
	if len(activator.activations) != 0 {
		t.Fatal("runtime owner was invoked for an unactivatable release")
	}
	if err := os.RemoveAll(releaseDir); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Stage(ctx, StageRequest{Effect: f.effect("op-1", "stage", 2), Verify: f.verify()}); err != nil {
		t.Fatal(err)
	}
	result, err := f.store.Activate(ctx, ActivateRequest{Effect: f.effect("op-1", "activate", 2), Release: f.manifest.ReleaseDigest, Activator: activator})
	if err != nil || result.Receipt.Outcome != OutcomeSucceeded {
		t.Fatalf("activate = %+v err=%v", result.Receipt, err)
	}
	if len(activator.activations) != 1 || activator.activations[0].ReleaseDir != releaseDir || strings.Join(activator.activations[0].Scenarios, ",") != testScenario || activator.activations[0].Strategy != StrategySideBySide {
		t.Fatalf("activation = %+v", activator.activations)
	}
	active, err := f.store.ReadActive(testDeployment)
	if err != nil || active == nil || active.ActiveRelease != f.manifest.ReleaseDigest || active.PreviousRelease != "" || active.OperationID != "op-1" || active.Fence != 2 {
		t.Fatalf("active = %+v err=%v", active, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(releaseDir), "..", activationIntentFile)); err == nil {
		t.Fatal("activation intent remained after commit")
	}
	unchanged, err := f.store.Activate(ctx, ActivateRequest{Effect: f.effect("op-2", "activate", 3), Release: f.manifest.ReleaseDigest, Activator: activator})
	if err != nil || unchanged.Receipt.Outcome != OutcomeUnchanged || len(activator.activations) != 1 {
		t.Fatalf("re-activate = %+v err=%v activations=%d", unchanged.Receipt, err, len(activator.activations))
	}
}

// [REQ:STC-P0-011] A failed or interrupted activation leaves the prior
// release active and durable state names what actually happened.
func TestActivationFailureAndCrashLeavePriorActive(t *testing.T) {
	stageTwo := func(t *testing.T) (fixture, fixture) {
		t.Helper()
		first := newFixture(t, defaultBundleEntries(), nil)
		second := newFixture(t, append(defaultBundleEntries(), bundleEntry{name: "scenarios/" + testScenario + "/CHANGELOG", body: "v2"}), nil)
		second.store = first.store
		ctx := context.Background()
		if _, err := first.store.Stage(ctx, StageRequest{Effect: first.effect("op-1", "stage", 1), Verify: first.verify()}); err != nil {
			t.Fatal(err)
		}
		if _, err := first.store.Activate(ctx, ActivateRequest{Effect: first.effect("op-1", "activate", 1), Release: first.manifest.ReleaseDigest, Activator: &recordingActivator{}}); err != nil {
			t.Fatal(err)
		}
		if _, err := second.store.Stage(ctx, StageRequest{Effect: second.effect("op-2", "stage", 2), Verify: second.verify()}); err != nil {
			t.Fatal(err)
		}
		return first, second
	}
	t.Run("runtime owner fails", func(t *testing.T) {
		first, second := stageTwo(t)
		activator := &recordingActivator{err: errors.New("candidate never became healthy")}
		result, err := second.store.Activate(context.Background(), ActivateRequest{Effect: second.effect("op-2", "activate", 2), Release: second.manifest.ReleaseDigest, Activator: activator})
		if mustCode(t, err) != CodeActivationFailed || result.Receipt.Outcome != OutcomeFailed || result.Receipt.Details["prior_active_release"] != first.manifest.ReleaseDigest {
			t.Fatalf("result = %+v err=%v", result.Receipt, err)
		}
		active, _ := first.store.ReadActive(testDeployment)
		if active == nil || active.ActiveRelease != first.manifest.ReleaseDigest {
			t.Fatalf("active = %+v", active)
		}
		listing, _ := first.store.ListReleases(testDeployment)
		if listing.InterruptedActivation != nil {
			t.Fatal("a reported runtime failure must not read as an interrupted switch")
		}
	})
	for _, point := range []string{"activate:before_runtime", "activate:after_runtime"} {
		t.Run("crash at "+point, func(t *testing.T) {
			first, second := stageTwo(t)
			point := point
			first.store.Hooks.Fault = func(p string) error {
				if p == point {
					return errors.New("simulated crash")
				}
				return nil
			}
			activator := &recordingActivator{}
			result, err := second.store.Activate(context.Background(), ActivateRequest{Effect: second.effect("op-2", "activate", 2), Release: second.manifest.ReleaseDigest, Activator: activator})
			if err == nil || result.Receipt.Outcome != OutcomeFailed {
				t.Fatalf("result = %+v err=%v", result.Receipt, err)
			}
			active, _ := first.store.ReadActive(testDeployment)
			if active == nil || active.ActiveRelease != first.manifest.ReleaseDigest {
				t.Fatalf("active after crash = %+v", active)
			}
			listing, err := first.store.ListReleases(testDeployment)
			if err != nil || listing.InterruptedActivation == nil || listing.InterruptedActivation.Candidate != second.manifest.ReleaseDigest || listing.InterruptedActivation.Previous != first.manifest.ReleaseDigest {
				t.Fatalf("listing = %+v err=%v", listing, err)
			}
			wantRuntimeCalls := 0
			if point == "activate:after_runtime" {
				wantRuntimeCalls = 1
			}
			if len(activator.activations) != wantRuntimeCalls {
				t.Fatalf("runtime calls = %d want %d", len(activator.activations), wantRuntimeCalls)
			}
			first.store.Hooks.Fault = nil
			retry, err := second.store.Activate(context.Background(), ActivateRequest{Effect: second.effect("op-3", "activate", 3), Release: second.manifest.ReleaseDigest, Activator: activator})
			if err != nil || retry.Receipt.Outcome != OutcomeSucceeded {
				t.Fatalf("retry = %+v err=%v", retry.Receipt, err)
			}
			listing, _ = first.store.ListReleases(testDeployment)
			if listing.InterruptedActivation != nil || listing.Active.ActiveRelease != second.manifest.ReleaseDigest || listing.Active.PreviousRelease != first.manifest.ReleaseDigest {
				t.Fatalf("listing after retry = %+v", listing)
			}
		})
	}
}

// [REQ:STC-P0-011] Rollback is available only to the retained predecessor.
func TestRollbackOnlyToRetainedPredecessor(t *testing.T) {
	first := newFixture(t, defaultBundleEntries(), nil)
	second := newFixture(t, append(defaultBundleEntries(), bundleEntry{name: "scenarios/" + testScenario + "/CHANGELOG", body: "v2"}), nil)
	third := newFixture(t, append(defaultBundleEntries(), bundleEntry{name: "scenarios/" + testScenario + "/CHANGELOG", body: "v3"}), nil)
	second.store, third.store = first.store, first.store
	ctx := context.Background()
	activator := &recordingActivator{}
	for i, f := range []fixture{first, second} {
		op := "op-" + string(rune('1'+i))
		if _, err := f.store.Stage(ctx, StageRequest{Effect: f.effect(op, "stage", uint64(i+1)), Verify: f.verify()}); err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.Activate(ctx, ActivateRequest{Effect: f.effect(op, "activate", uint64(i+1)), Release: f.manifest.ReleaseDigest, Activator: activator}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := third.store.Stage(ctx, StageRequest{Effect: third.effect("op-3", "stage", 3), Verify: third.verify()}); err != nil {
		t.Fatal(err)
	}
	if code, exit := codeOf(t, must2(first.store.Rollback(ctx, RollbackRequest{Effect: first.effect("op-4", "rollback", 4), To: third.manifest.ReleaseDigest, Activator: activator}))); code != CodeRollbackNotEligible || exit != ExitRefused {
		t.Fatalf("rollback to a never-active release code/exit = %s/%d", code, exit)
	}
	if code := mustCode(t, must2(first.store.Rollback(ctx, RollbackRequest{Effect: first.effect("op-4", "rollback-2", 4), To: second.manifest.ReleaseDigest, Activator: activator}))); code != CodeRollbackNotEligible {
		t.Fatalf("rollback to the active release code = %s", code)
	}
	result, err := first.store.Rollback(ctx, RollbackRequest{Effect: first.effect("op-5", "rollback", 5), To: first.manifest.ReleaseDigest, Strategy: StrategyMaintenance, Activator: activator})
	if err != nil || result.Receipt.Outcome != OutcomeSucceeded {
		t.Fatalf("rollback = %+v err=%v", result.Receipt, err)
	}
	active, _ := first.store.ReadActive(testDeployment)
	if active.ActiveRelease != first.manifest.ReleaseDigest || active.PreviousRelease != second.manifest.ReleaseDigest || active.Strategy != StrategyMaintenance {
		t.Fatalf("active after rollback = %+v", active)
	}
	listing, _ := first.store.ListReleases(testDeployment)
	roles := map[string]string{}
	for _, row := range listing.Releases {
		roles[row.Digest] = row.Role
	}
	if roles[first.manifest.ReleaseDigest] != "active" || roles[second.manifest.ReleaseDigest] != "previous" || roles[third.manifest.ReleaseDigest] != "staged" {
		t.Fatalf("roles = %v", roles)
	}
}

type recordingRunner struct{ calls [][]string }

func (r *recordingRunner) LookPath(name string) (string, error) { return name, nil }

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return nil, nil
}

// The default activator delegates to the lifecycle owner through argv with
// the release tree as the scenario path.
func TestScenarioRestartActivatorUsesLifecycleArgv(t *testing.T) {
	runner := &recordingRunner{}
	activator := ScenarioRestartActivator{Runner: runner, Executable: "/opt/vrooli/bin/vrooli"}
	err := activator.Activate(context.Background(), Activation{ReleaseDir: "/home/deploy/.vrooli/cloud/deployments/d/releases/abc", Scenarios: []string{"landing-app"}, Strategy: StrategyMaintenance})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"/opt/vrooli/bin/vrooli", "scenario", "stop", "landing-app", "--json"},
		{"/opt/vrooli/bin/vrooli", "scenario", "restart", "landing-app", "--path", "/home/deploy/.vrooli/cloud/deployments/d/releases/abc/scenarios/landing-app", "--json"},
	}
	if mustJSON(t, runner.calls) != mustJSON(t, want) {
		t.Fatalf("calls = %q", runner.calls)
	}
}

func TestInventoryReportsHeuristicDirectoriesAndBindingCoverage(t *testing.T) {
	workdir := t.TempDir()
	scenarioDir := filepath.Join(workdir, "scenarios", testScenario)
	for _, rel := range []string{"data/uploads", "api/cache", "src/deep/one/two/three/logs", "api"} {
		if err := os.MkdirAll(filepath.Join(scenarioDir, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "data", "uploads", "a.bin"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	bindings, err := ParseBindings(`[{"id":"uploads","path":"data/uploads"}]`)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Inventory(InventoryRequest{DeploymentID: testDeployment, Workdir: workdir, Scenario: testScenario, Bindings: bindings})
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]InventoryEntry{}
	for _, entry := range report.Entries {
		byPath[entry.Path] = entry
	}
	if len(report.Entries) != 3 || report.UncoveredCount != 1 {
		t.Fatalf("report = %+v", report)
	}
	if e := byPath["data"]; !e.Covered || e.BindingID != "uploads" || e.Bytes != 5 || e.Files != 1 {
		t.Fatalf("data entry = %+v", e)
	}
	if e := byPath["data/uploads"]; !e.Covered || e.Bytes != 5 {
		t.Fatalf("uploads entry = %+v", e)
	}
	if e := byPath["api/cache"]; e.Covered {
		t.Fatalf("cache entry = %+v", e)
	}
	if _, ok := byPath["src/deep/one/two/three/logs"]; ok {
		t.Fatal("inventory descended past the heuristic depth")
	}
	if _, err := ParseBindings(`[{"path":"../outside"}]`); mustCode(t, err) != CodeInvalidArgument {
		t.Fatalf("escaping binding error = %v", err)
	}
}

type fakeBroker struct {
	requests  []privilegebroker.Request
	available bool
	result    privilegebroker.Result
}

func (b *fakeBroker) Available() bool { return b.available }

func (b *fakeBroker) Do(_ context.Context, req privilegebroker.Request) (privilegebroker.Result, error) {
	b.requests = append(b.requests, req)
	return b.result, nil
}

// [REQ:STC-P0-008] Host repair accepts only broker actions with typed
// subjects, and process stops go through the lifecycle owner.
func TestHostRepairDelegatesOnlyBrokerActions(t *testing.T) {
	f := newFixture(t, defaultBundleEntries(), nil)
	ctx := context.Background()
	for name, req := range map[string]HostRepairRequest{
		"generic shell":       {Action: "shell.exec", Subject: []byte(`{"command":"id"}`)},
		"unknown field":       {Action: privilegebroker.ActionAptPackagesEnsure, Subject: []byte(`{"apt":{"packages":["jq"]},"command":"id"}`)},
		"unlisted package":    {Action: privilegebroker.ActionAptPackagesEnsure, Subject: []byte(`{"apt":{"packages":["nmap"]}}`)},
		"management port":     {Action: privilegebroker.ActionEdgeUFWAllow, Subject: []byte(`{"edge":{"port":18767}}`)},
		"process pattern":     {Action: privilegebroker.ActionProcessStopScoped, Subject: []byte(`{"process":{"scenario":"-f node","workdir":"/opt"}}`)},
		"smuggled action":     {Action: privilegebroker.ActionAptPackagesEnsure, Subject: []byte(`{"action":"bridge.ufw.revoke","apt":{"packages":["jq"]}}`)},
		"bridge revoke shape": {Action: privilegebroker.ActionBridgeUFWRevoke, Subject: []byte(`{"subject":{"scenario":"vrooli-bridge","candidate_ip":"10.0.0.5","port":18767}}`)},
	} {
		t.Run(name, func(t *testing.T) {
			broker := &fakeBroker{available: true}
			runner := &recordingRunner{}
			_, _, err := f.store.HostRepair(ctx, req, broker, runner)
			if name == "bridge revoke shape" {
				// A well-formed existing broker action is admissible; the
				// broker remains the authority on its own policy.
				if err != nil {
					t.Fatalf("existing broker action refused: %v", err)
				}
				return
			}
			if code, exit := codeOf(t, err); code != CodeActionNotAllowed || exit != ExitRefused {
				t.Fatalf("code/exit = %s/%d", code, exit)
			}
			if len(broker.requests) != 0 || len(runner.calls) != 0 {
				t.Fatal("refused action reached an executor")
			}
		})
	}
	t.Run("apt through broker with receipt", func(t *testing.T) {
		broker := &fakeBroker{available: true, result: privilegebroker.Result{Version: privilegebroker.ProtocolVersion, Status: "completed", Changed: true}}
		req := HostRepairRequest{Action: privilegebroker.ActionAptPackagesEnsure, Subject: []byte(`{"apt":{"packages":["jq","caddy"]}}`), Effect: f.effect("op-h", "apt", 1)}
		result, effect, err := f.store.HostRepair(ctx, req, broker, &recordingRunner{})
		if err != nil || effect == nil || effect.Receipt.Outcome != OutcomeSucceeded || result.Executor != "privilege-broker" {
			t.Fatalf("result=%+v effect=%+v err=%v", result, effect, err)
		}
		if len(broker.requests) != 1 || broker.requests[0].Action != privilegebroker.ActionAptPackagesEnsure || broker.requests[0].Version != privilegebroker.ProtocolVersion {
			t.Fatalf("broker requests = %+v", broker.requests)
		}
		replay, effect, err := f.store.HostRepair(ctx, req, broker, &recordingRunner{})
		if err != nil || !effect.Replayed || len(broker.requests) != 1 || replay.Executor != "" {
			t.Fatalf("replay = %+v effect=%+v err=%v", replay, effect, err)
		}
	})
	t.Run("broker unavailable is a failure not a fallback", func(t *testing.T) {
		req := HostRepairRequest{Action: privilegebroker.ActionEdgeUFWAllow, Subject: []byte(`{"edge":{"port":80}}`)}
		_, _, err := f.store.HostRepair(ctx, req, &fakeBroker{}, &recordingRunner{})
		if code, exit := codeOf(t, err); code != CodeBrokerUnavailable || exit != ExitFailed {
			t.Fatalf("code/exit = %s/%d", code, exit)
		}
	})
	t.Run("scoped stop runs the lifecycle owner", func(t *testing.T) {
		broker := &fakeBroker{available: true}
		runner := &recordingRunner{}
		req := HostRepairRequest{Action: privilegebroker.ActionProcessStopScoped, Subject: []byte(`{"process":{"scenario":"landing-app","workdir":"/opt/vrooli"}}`)}
		result, _, err := f.store.HostRepair(ctx, req, broker, runner)
		if err != nil || result.Executor != "lifecycle-owner" || !result.Changed {
			t.Fatalf("result = %+v err=%v", result, err)
		}
		if len(broker.requests) != 0 || mustJSON(t, runner.calls) != mustJSON(t, [][]string{{"vrooli", "scenario", "stop", "landing-app", "--json"}}) {
			t.Fatalf("broker=%d runner=%q", len(broker.requests), runner.calls)
		}
	})
}
