package cliinstall

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	"github.com/vrooli/vrooli/internal/rootmetrics"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVerifiedCompanionInstallRefusesReplacement(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "candidate"), filepath.Join(dir, "companion")
	good := []byte("qualified bytes")
	sum := fmt.Sprintf("%x", sha256.Sum256(good))
	if err := os.WriteFile(src, good, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("serving"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("replaced"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := AtomicInstallVerifiedSource(src, dst, sum); err == nil {
		t.Fatal("replacement accepted")
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "serving" {
		t.Fatalf("serving bytes changed: %q", got)
	}
	if err := os.WriteFile(src, good, 0700); err != nil {
		t.Fatal(err)
	}
	if err := AtomicInstallVerifiedSource(src, dst, sum); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(dst)
	if string(got) != string(good) {
		t.Fatalf("installed %q", got)
	}
	if err := AtomicInstallVerifiedSource(src, filepath.Join(dir, "vrooli"), sum); err == nil {
		t.Fatal("root activation bypassed")
	}
}

func TestCompanionsRefuseRootChangedWhileWaitingForPublicationLock(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "build")
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(src, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range companionNames {
		if err := os.WriteFile(filepath.Join(src, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join(bin, "vrooli")
	old := []byte("old root")
	if err := os.WriteFile(root, old, 0700); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(old))
	release, err := buildinfo.AcquireBinaryInstallLock(root)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- InstallCompanions(src, bin, digest) }()
	if err := os.WriteFile(root, []byte("new root"), 0700); err != nil {
		t.Fatal(err)
	}
	release()
	if err := <-done; err == nil {
		t.Fatal("old companions published under a new root")
	}
	for _, name := range companionNames {
		if _, err := os.Stat(filepath.Join(bin, name)); !os.IsNotExist(err) {
			t.Fatalf("companion %s was published", name)
		}
	}
}

func TestLegacyCompanionsHonorRootWorkflowAndRecoveryState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	rootData := []byte("serving root")
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), rootData, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range companionNames {
		if err := os.WriteFile(filepath.Join(src, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(rootData))
	release, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- InstallCompanions(src, bin, digest) }()
	select {
	case err := <-done:
		release()
		t.Fatalf("crossed bundle workflow: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		release()
		t.Fatal(err)
	}
	path := filepath.Join(state, "root-rebaseline", "progress.json")
	if err := rebaselineprogress.Write(path, rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64)}); err != nil {
		release()
		t.Fatal(err)
	}
	release()
	if err := <-done; err == nil || !strings.Contains(err.Error(), "rebaseline is incomplete") {
		t.Fatalf("interrupted rebaseline ignored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(bin, companionNames[0])); !os.IsNotExist(err) {
		t.Fatalf("companion published: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	transition, err := buildinfo.AcquireRebaselineTransition(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	err = InstallCompanions(src, bin, digest)
	transition()
	if err == nil || !strings.Contains(err.Error(), "transition is busy") {
		t.Fatalf("active transition ignored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(bin, companionNames[0])); !os.IsNotExist(err) {
		t.Fatalf("companion published: %v", err)
	}
	// Persist a valid prepared operation, then prove the companion path invokes
	// the operation guard rather than merely the transition lock.
	id := strings.Repeat("1", 32)
	spec := platform.OperationSpec{Path: filepath.Join(home, "watchdog"), Dir: home, Args: []string{"--core-update-execute", id}, Env: []string{}, Limits: platform.OperationLimits{MemoryMaxBytes: 128 << 20, TasksMax: 16, CPUWeight: 10, WallTime: time.Minute}}
	intent, err := platform.NewOperationIntent("core-activation", id, spec)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	hash := strings.Repeat("a", 64)
	r := coreactivation.Record{SchemaVersion: 1, ID: id, Home: home, SourceRoot: filepath.Join(home, "source"), Revision: 1, Phase: coreactivation.Prepared, Fence: coreactivation.Ready, CandidateSHA: hash, CandidateBuild: hash, CandidateStateContract: hash, CandidateUnitSHA: hash, LayoutSHA: hash, WorkerSHA: hash, PriorSHA: hash, PriorBuild: hash, PriorStateContract: hash, PriorUnitSHA: hash, PriorPID: 456, PriorIncarnation: "01234567-89ab-cdef-0123-456789abcdef:12345", PriorNativeInvocation: strings.Repeat("5", 32), NativeOverridesDigest: hash, CreatedUnixNano: now.UnixNano(), DeadlineUnixNano: now.Add(time.Minute).UnixNano(), WorkerIntent: intent, FailedReleases: []string{}, FailedBuilds: []string{}}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.Directory(), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Directory(), "initialized"), []byte("vrooli-core-activation-store-v1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Directory(), "current.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, exists, err := store.Current(); err != nil || !exists {
		t.Fatalf("invalid active operation fixture: exists=%v err=%v", exists, err)
	}
	if err := InstallCompanions(src, bin, digest); err == nil || !strings.Contains(err.Error(), "recorded root activation") {
		t.Fatalf("active operation ignored: %v", err)
	}
}

func TestStandalonePreparedCompanionsWaitForBundleWorkflow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(src, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), []byte("vrooli"), 0700); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(src, bundle); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	manifestSHA := fmt.Sprintf("%x", sha256.Sum256(manifest))
	rootSHA := fmt.Sprintf("%x", sha256.Sum256([]byte("vrooli")))
	release, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- InstallPreparedCompanions(bundle, bin, rootSHA, manifestSHA) }()
	select {
	case err := <-done:
		release()
		t.Fatalf("crossed bundle workflow: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	release()
	if err := <-done; err != nil {
		t.Fatalf("publication after workflow release: %v", err)
	}
}

func TestSourcePointerFollowsWinningBundleAcrossCheckouts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	prepare := func(label string) (string, string, string) {
		t.Helper()
		checkout := t.TempDir()
		source := filepath.Join(checkout, "build")
		if err := os.Mkdir(source, 0700); err != nil {
			t.Fatal(err)
		}
		for _, name := range bundleNames {
			if err := os.WriteFile(filepath.Join(source, name), []byte(label+name), 0700); err != nil {
				t.Fatal(err)
			}
		}
		bundle := filepath.Join(checkout, "bundle")
		if err := PrepareBundle(source, bundle); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
		if err != nil {
			t.Fatal(err)
		}
		return checkout, bundle, fmt.Sprintf("%x", sha256.Sum256(data))
	}
	a, bundleA, digestA := prepare("A")
	b, bundleB, digestB := prepare("B")
	for _, name := range append([]string{"vrooli"}, companionNames...) {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("B"+name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	pointer := repocontract.SourceRootPointerPath(home)
	release, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	aDone, bDone := make(chan error, 1), make(chan error, 1)
	go func() { aDone <- PublishPreparedBundleSource(bundleA, bin, digestA, a, pointer) }()
	go func() { bDone <- PublishPreparedBundleSource(bundleB, bin, digestB, b, pointer) }()
	release()
	if err := <-aDone; err == nil || !strings.Contains(err.Error(), "installed bundle changed") {
		t.Fatalf("losing checkout published pointer: %v", err)
	}
	if err := <-bDone; err != nil {
		t.Fatalf("winning checkout failed: %v", err)
	}
	data, err := os.ReadFile(pointer)
	if err != nil || strings.TrimSpace(string(data)) != b {
		t.Fatalf("source pointer = %q, %v", data, err)
	}
}

func TestPreparedCompanionsPreflightWholeBundleBeforePublishing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "build")
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(src, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	var manifest strings.Builder
	for _, name := range bundleNames {
		data := []byte(name)
		if err := os.WriteFile(filepath.Join(src, name), data, 0700); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&manifest, "%x  %s\n", sha256.Sum256(data), name)
	}
	contents := []byte(manifest.String())
	if err := os.WriteFile(filepath.Join(src, "SHA256SUMS"), contents, 0600); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(contents))
	if err := os.WriteFile(filepath.Join(src, companionNames[len(companionNames)-1]), []byte("changed"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedCompanionsLocked(src, bin, digest); err == nil {
		t.Fatal("changed bundle accepted")
	}
	for _, name := range companionNames {
		if _, err := os.Stat(filepath.Join(bin, name)); !os.IsNotExist(err) {
			t.Fatalf("published %s before complete preflight", name)
		}
	}
}

func TestPreparedCompanionsRecoverAfterPartialPublication(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "build")
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(src, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	var manifest strings.Builder
	for _, name := range bundleNames {
		data := []byte("prepared " + name)
		if err := os.WriteFile(filepath.Join(src, name), data, 0700); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&manifest, "%x  %s\n", sha256.Sum256(data), name)
	}
	contents := []byte(manifest.String())
	if err := os.WriteFile(filepath.Join(src, "SHA256SUMS"), contents, 0600); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(contents))
	// Model a prior interrupted publication: one companion was already replaced.
	if err := os.WriteFile(filepath.Join(bin, companionNames[0]), []byte("prepared "+companionNames[0]), 0700); err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedCompanionsLocked(src, bin, digest); err != nil {
		t.Fatal(err)
	}
	for _, name := range companionNames {
		data, err := os.ReadFile(filepath.Join(bin, name))
		if err != nil || string(data) != "prepared "+name {
			t.Fatalf("%s not recovered: %q %v", name, data, err)
		}
	}
}

func TestPrepareBundlePreflightsAllImagesAndPreservesSnapshot(t *testing.T) {
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(t.TempDir(), "snapshot")
	if err := PrepareBundle(source, output); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(output, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	sha := fmt.Sprintf("%x", sha256.Sum256(manifest))
	if err := VerifyPreparedBundle(output, sha); err != nil {
		t.Fatal(err)
	}
	if err := PrepareBundle(source, output); err == nil {
		t.Fatal("existing snapshot overwritten")
	}
	if err := os.WriteFile(filepath.Join(source, "vrooli-api"), []byte("new build"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := VerifyPreparedBundle(output, sha); err != nil {
		t.Fatalf("source change affected snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(output, "vrooli-gocacheprog"), []byte("tampered"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := VerifyPreparedBundle(output, sha); err == nil {
		t.Fatal("tampered image accepted")
	}
}

func TestPrepareBundleRemovesIncompleteOutput(t *testing.T) {
	source := t.TempDir()
	output := filepath.Join(t.TempDir(), "snapshot")
	if err := PrepareBundle(source, output); err == nil {
		t.Fatal("missing images accepted")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("incomplete snapshot remains: %v", err)
	}
}

func TestPreparedPublicationRejectsBundleForDifferentRoot(t *testing.T) {
	source, bin := t.TempDir(), t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(t.TempDir(), "snapshot")
	if err := PrepareBundle(source, output); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(output, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	root := []byte("another root")
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedCompanions(output, bin, fmt.Sprintf("%x", sha256.Sum256(root)), fmt.Sprintf("%x", sha256.Sum256(manifest))); err == nil {
		t.Fatal("different root accepted")
	}
	for _, name := range companionNames {
		if _, err := os.Stat(filepath.Join(bin, name)); !os.IsNotExist(err) {
			t.Fatalf("published %s: %v", name, err)
		}
	}
}

func TestPreparedPublicationRetainsPartialEvidenceOnlyForCanonicalBin(t *testing.T) {
	previous := beginBundleObservation
	var observed *rootmetrics.Deferred
	beginBundleObservation = func(home, kind string) *rootmetrics.Deferred {
		observed = rootmetrics.BeginDeferred(home, kind)
		return observed
	}
	t.Cleanup(func() { beginBundleObservation = previous })
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "bundle")
	if err := os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	var manifest strings.Builder
	for _, name := range bundleNames {
		data := []byte(name)
		if err := os.WriteFile(filepath.Join(src, name), data, 0700); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&manifest, "%x  %s\n", sha256.Sum256(data), name)
	}
	contents := []byte(manifest.String())
	if err := os.WriteFile(filepath.Join(src, "SHA256SUMS"), contents, 0600); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(contents))
	// A directory at the second destination forces a failure after the first write.
	if err := os.Mkdir(filepath.Join(bin, companionNames[1]), 0700); err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedCompanionsLocked(src, bin, digest); err == nil {
		t.Fatal("expected publication failure")
	}
	if observed == nil {
		t.Fatal("missing observation")
	}
	<-observed.Done()
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(state, "root-rebaseline", "attempts")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("observation: %v %v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Kind     string                           `json:"kind"`
		Manifest string                           `json:"bundle_manifest_sha256"`
		Images   map[string]string                `json:"images"`
		Events   []struct{ Stage, Reason string } `json:"events"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Kind != "bundle_publication" || receipt.Manifest != digest || len(receipt.Images) != len(bundleNames) || len(receipt.Events) != 4 || receipt.Events[2].Stage != "vrooli_api" || receipt.Events[3].Reason != "failed" {
		t.Fatalf("incorrect partial evidence: %s", data)
	}
	if strings.Contains(string(data), home) || strings.Contains(string(data), src) {
		t.Fatalf("path leaked: %s", data)
	}
	other := t.TempDir()
	if err := InstallPreparedCompanionsLocked(src, other, digest); err != nil {
		t.Fatal(err)
	}
	entries, err = os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("noncanonical destination wrote receipt: %v %v", entries, err)
	}
}

func TestPreparedCompanionsRefuseActiveRebaseline(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "bundle")
	if err := os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	var manifest strings.Builder
	for _, name := range bundleNames {
		data := []byte(name)
		if err := os.WriteFile(filepath.Join(src, name), data, 0700); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&manifest, "%x  %s\n", sha256.Sum256(data), name)
	}
	data := []byte(manifest.String())
	if err := os.WriteFile(filepath.Join(src, "SHA256SUMS"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), []byte("vrooli"), 0700); err != nil {
		t.Fatal(err)
	}
	release, err := buildinfo.AcquireRebaselineTransition(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := InstallPreparedCompanions(src, bin, fmt.Sprintf("%x", sha256.Sum256([]byte("vrooli"))), fmt.Sprintf("%x", sha256.Sum256(data))); err == nil {
		t.Fatal("published during active rebaseline")
	}
	if _, err := os.Stat(filepath.Join(bin, companionNames[0])); !os.IsNotExist(err) {
		t.Fatalf("companion published: %v", err)
	}
}

func TestPreparedBundleRefusesInterruptedRebaseline(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(bin, "vrooli")
	if err := os.WriteFile(root, []byte("serving"), 0700); err != nil {
		t.Fatal(err)
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(state, "root-rebaseline", "progress.json")
	// The install starts while a transition owns the workflow. Its check must
	// happen after acquiring that lock and seeing the newly durable intent.
	release, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	started := make(chan struct{})
	source := t.TempDir()
	go func() {
		close(started)
		done <- InstallPreparedBundle(source, bin, "")
	}()
	<-started
	if err := rebaselineprogress.Write(path, rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64)}); err != nil {
		release()
		t.Fatal(err)
	}
	release()
	if err := <-done; err == nil || !strings.Contains(err.Error(), "rebaseline is incomplete") {
		t.Fatalf("ordinary install did not fence interrupted transition: %v", err)
	}
	data, err := os.ReadFile(root)
	if err != nil || string(data) != "serving" {
		t.Fatalf("serving root changed: %q %v", data, err)
	}
}

// Uses a frozen governed image and an isolated home; no native service runs.
func TestSameContractPreparedBundleWithFrozenRoot(t *testing.T) {
	image := os.Getenv("VROOLI_SAME_CONTRACT_TEST_IMAGE")
	if image == "" {
		t.Skip("set VROOLI_SAME_CONTRACT_TEST_IMAGE to a governed root CLI for isolated integration")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	root, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "vrooli"), root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), root, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range bundleNames[1:] {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "prepared")
	if err := PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	manifestSHA := fmt.Sprintf("%x", sha256.Sum256(manifest))
	// Simulate another transition starting after the caller's qualification
	// but before this installer reacquires the workflow lock.
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	progressPath := filepath.Join(state, "root-rebaseline", "progress.json")
	if err := rebaselineprogress.Write(progressPath, rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64)}); err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedBundleSameContract(bundle, bin, manifestSHA); err == nil || !strings.Contains(err.Error(), "rebaseline is incomplete") {
		t.Fatalf("incomplete transition not fenced: %v", err)
	}
	if err := rebaselineprogress.Write(progressPath, rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64), Complete: true}); err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedBundleSameContract(bundle, bin, manifestSHA); err != nil {
		t.Fatal(err)
	}
	for _, name := range companionNames {
		data, err := os.ReadFile(filepath.Join(bin, name))
		if err != nil || string(data) != name {
			t.Fatalf("companion %s: %q %v", name, data, err)
		}
	}
}

func TestInterruptedOrdinaryBundleFencesAndResumesRetainedBytes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	retained, err := beginOrdinaryBundle(home, bundle, bin, digest, false, "")
	if err != nil {
		t.Fatal(err)
	}
	// Simulate interruption after root and the first companion have published.
	for _, name := range []string{"vrooli", companionNames[0]} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(bundle); err != nil {
		t.Fatal(err)
	}
	if err := buildinfo.GuardRootActivation(context.Background(), filepath.Join(bin, "vrooli")); err == nil {
		t.Fatal("unfinished bundle did not fence root")
	}
	if _, err := beginOrdinaryBundle(home, retained, bin, strings.Repeat("a", 64), false, ""); err == nil {
		t.Fatal("different bundle replaced recovery")
	}
	if err := ResumePreparedBundle(bin); err != nil {
		t.Fatal(err)
	}
	for _, name := range companionNames {
		got, err := os.ReadFile(filepath.Join(bin, name))
		if err != nil || string(got) != name {
			t.Fatalf("%s = %q, %v", name, got, err)
		}
	}
	if intent, err := buildinfo.ReadPreparedBundleIntent(home); err != nil || intent != nil {
		t.Fatalf("intent = %+v, %v", intent, err)
	}
}

func TestAbortOrdinaryBundleRequiresUnchangedPrior(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	if _, err := beginOrdinaryBundle(home, bundle, bin, digest, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, companionNames[0]), []byte("published"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := AbortPreparedBundle(bin); err == nil {
		t.Fatal("aborted after publication")
	}
	if err := os.Remove(filepath.Join(bin, companionNames[0])); err != nil {
		t.Fatal(err)
	}
	if err := AbortPreparedBundle(bin); err != nil {
		t.Fatal(err)
	}
	if intent, err := buildinfo.ReadPreparedBundleIntent(home); err != nil || intent != nil {
		t.Fatalf("intent = %+v, %v", intent, err)
	}
}

func TestSameContractRecoveryRetainsQualification(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), []byte("unqualified prior"), 0700); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	if _, err := beginOrdinaryBundle(home, bundle, bin, digest, true, ""); err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedBundle(bundle, bin, digest); err == nil {
		t.Fatal("generic publication accepted same-contract recovery")
	}
	if err := ResumePreparedBundle(bin); err == nil {
		t.Fatal("recovery bypassed qualification")
	}
	got, err := os.ReadFile(filepath.Join(bin, "vrooli"))
	if err != nil || string(got) != "unqualified prior" {
		t.Fatalf("root changed: %q, %v", got, err)
	}
}

func TestEnrolledBundleAndResumeUseBoundSourceRoot(t *testing.T) {
	image := os.Getenv("VROOLI_SAME_CONTRACT_TEST_IMAGE")
	if image == "" {
		t.Skip("set VROOLI_SAME_CONTRACT_TEST_IMAGE to a governed root image")
	}
	home, sourceRoot, stale := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	t.Setenv(buildinfo.SourceRootEnvVar, stale)
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), root, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConfigureEnrollment(context.Background(), true, func(context.Context) (func(), error) { return func() {}, nil }); err != nil {
		t.Fatal(err)
	}
	worker, err := coreactivation.InstalledWatchdogPath(home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(worker), 0700); err != nil {
		t.Fatal(err)
	}
	trace := filepath.Join(home, "handoff")
	t.Setenv("TEST_ROOT_HANDOFF", trace)
	script := "#!/bin/sh\nprintf '%s\\n' \"$3\" >> \"$TEST_ROOT_HANDOFF\"\nprintf '%s\\n' '{\"id\":\"\",\"phase\":\"\",\"failure_class\":\"admission_refused\",\"reason\":\"fixture refusal\"}'\nexit 1\n"
	if err := os.WriteFile(worker, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for _, name := range bundleNames {
		data := []byte(name)
		if name == "vrooli" {
			data = append(append([]byte{}, root...), '\n')
		}
		if err := os.WriteFile(filepath.Join(source, name), data, 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(manifest))
	if err := InstallPreparedBundleWithSource(bundle, bin, digest, sourceRoot); err == nil {
		t.Fatal("fixture watchdog should refuse")
	}
	if err := ResumePreparedBundle(bin); err == nil {
		t.Fatal("fixture watchdog should refuse resume")
	}
	data, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Fields(string(data)); len(got) != 2 || got[0] != sourceRoot || got[1] != sourceRoot {
		t.Fatalf("handoff did not preserve transaction source: %q", data)
	}
}

func TestSameContractRefusalDoesNotCreateRecoveryIntent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), []byte("unqualified serving root"), 0700); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedBundleSameContract(bundle, bin, fmt.Sprintf("%x", sha256.Sum256(data))); err == nil {
		t.Fatal("unqualified root accepted")
	}
	if intent, err := buildinfo.ReadPreparedBundleIntent(home); err != nil || intent != nil {
		t.Fatalf("refusal left recovery intent: %+v %v", intent, err)
	}
	// Equal bytes are not proof that the executable has a qualified identity.
	if err := os.WriteFile(filepath.Join(bin, "vrooli"), []byte("vrooli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := InstallPreparedBundleSameContract(bundle, bin, fmt.Sprintf("%x", sha256.Sum256(data))); err == nil {
		t.Fatal("unqualified identical root accepted")
	}
	if intent, err := buildinfo.ReadPreparedBundleIntent(home); err != nil || intent != nil {
		t.Fatalf("identical-root refusal left recovery intent: %+v %v", intent, err)
	}
}

func TestSameContractQualificationFencesRootReplacement(t *testing.T) {
	image := os.Getenv("VROOLI_SAME_CONTRACT_TEST_IMAGE")
	if image == "" {
		t.Skip("set VROOLI_SAME_CONTRACT_TEST_IMAGE to a governed root image")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.ReadFile(image)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(bin, "vrooli")
	if err := os.WriteFile(path, root, 0700); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(filepath.Dir(image), bundle); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	prior, err := qualifyOrdinaryBundle(home, bundle, bin, digest, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("concurrent root"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := beginOrdinaryBundle(home, bundle, bin, digest, true, prior); err == nil || !strings.Contains(err.Error(), "serving root changed") {
		t.Fatalf("replacement accepted: %v", err)
	}
	if intent, err := buildinfo.ReadPreparedBundleIntent(home); err != nil || intent != nil {
		t.Fatalf("race left recovery intent: %+v %v", intent, err)
	}
}

func TestInterruptedBundleRecoversSourceBinding(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	checkout := t.TempDir()
	if _, err := beginOrdinaryBundleWithSource(home, bundle, bin, digest, false, "", checkout); err != nil {
		t.Fatal(err)
	}
	for _, name := range append([]string{"vrooli"}, companionNames...) {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	// A crash after the final image publication still leaves an exact source intent.
	if _, err := beginOrdinaryBundleWithSource(home, bundle, bin, digest, false, "", t.TempDir()); err == nil {
		t.Fatal("different source accepted for unfinished bundle")
	}
	if err := ResumePreparedBundle(bin); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(repocontract.SourceRootPointerPath(home))
	if err != nil || string(got) != checkout+"\n" {
		t.Fatalf("pointer = %q, %v", got, err)
	}
	if intent, err := buildinfo.ReadPreparedBundleIntent(home); err != nil || intent != nil {
		t.Fatalf("intent = %+v, %v", intent, err)
	}
}

func TestOrdinaryBundlePointerFailureKeepsIntentForRetry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	manifestSHA := fmt.Sprintf("%x", sha256.Sum256(manifestBytes))
	checkout := t.TempDir()
	retained, err := beginOrdinaryBundleWithSource(home, bundle, bin, manifestSHA, false, "", checkout)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := readPreparedManifest(retained, manifestSHA)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range append([]string{"vrooli"}, companionNames...) {
		data, err := os.ReadFile(filepath.Join(retained, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bin, name), data, 0700); err != nil {
			t.Fatal(err)
		}
	}

	pointer := repocontract.SourceRootPointerPath(home)
	if err := os.MkdirAll(pointer, 0700); err != nil {
		t.Fatal(err)
	}
	if err := finishOrdinaryBundle(home, bin, manifest); err == nil {
		t.Fatal("source pointer write unexpectedly succeeded")
	}
	intent, err := buildinfo.ReadPreparedBundleIntent(home)
	if err != nil || intent == nil || intent.SourceRoot != checkout {
		t.Fatalf("failed pointer publication lost recoverable intent: %+v, %v", intent, err)
	}
	if err := os.RemoveAll(pointer); err != nil {
		t.Fatal(err)
	}
	if err := ResumePreparedBundle(bin); err != nil {
		t.Fatalf("retry recovery: %v", err)
	}
	got, err := os.ReadFile(pointer)
	if err != nil || string(got) != checkout+"\n" {
		t.Fatalf("source pointer = %q, %v", got, err)
	}
	intent, err = buildinfo.ReadPreparedBundleIntent(home)
	if err != nil || intent != nil {
		t.Fatalf("completed retry left intent: %+v, %v", intent, err)
	}
}

func TestSameContractSourceCheckRunsUnderFenceBeforeIntent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	checked := false
	err = InstallPreparedBundleSameContractWithSourceChecked(t.TempDir(), bin, strings.Repeat("a", 64), t.TempDir(), func() error {
		checked = true
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		release, err := buildinfo.AcquireRootBundleWorkflow(ctx, home)
		if err == nil {
			release()
			t.Fatal("source check ran outside workflow fence")
		}
		return errors.New("source changed")
	})
	if !checked || err == nil || !strings.Contains(err.Error(), "source changed") {
		t.Fatalf("source check: checked=%v err=%v", checked, err)
	}
	intent, _, err := buildinfo.PreparedBundlePaths(home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(intent); !os.IsNotExist(err) {
		t.Fatalf("source refusal wrote intent: %v", err)
	}
}
