package runtimehandlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
)

// operatorPaneCgroup is a live web-console operator pane's membership
// (captured 2026-11-07); agentscope's caller tests cover the classification.
const operatorPaneCgroup = "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/tmux-spawn-a21c3737-0e07-4132-83a7-a22e5fff677b.scope\n"

func withCaller(t *testing.T, kind cliutil.CallerKind, cgroup string) {
	t.Helper()
	oldKind, oldCgroup := callerKindFn, callerCgroupFn
	t.Cleanup(func() { callerKindFn, callerCgroupFn = oldKind, oldCgroup })
	callerKindFn = func() cliutil.CallerKind { return kind }
	callerCgroupFn = func() string { return cgroup }
}

func TestRebaselineRefusesAgentSessionsBeforeAnythingElse(t *testing.T) {
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	resolved := false
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline", "/candidate"}, os.Stdin, &stdout, &stderr, func() (string, error) {
		resolved = true
		return "/source", nil
	})
	if code != 1 || resolved || !strings.Contains(stderr.String(), "operator decision") || !strings.Contains(stderr.String(), "Ask the operator") {
		t.Fatalf("an agent session reached the transition: code=%d resolved=%t %q", code, resolved, stderr.String())
	}
}

func TestRebaselineNeedsAnInteractiveTerminal(t *testing.T) {
	withCaller(t, cliutil.CallerKindUnknown, operatorPaneCgroup)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	_, _ = writer.WriteString("y\n")
	_ = writer.Close()
	resolved := false
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline", "/candidate"}, reader, &stdout, &stderr, func() (string, error) {
		resolved = true
		return "/source", nil
	})
	if code != 1 || resolved || stdout.Len() != 0 || !strings.Contains(stderr.String(), "interactive terminal") {
		t.Fatalf("piped confirmation was accepted: code=%d resolved=%t stdout=%q stderr=%q", code, resolved, stdout.String(), stderr.String())
	}
	stderr.Reset()
	if code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline"}, reader, &stdout, &stderr, nil); code != 2 || !strings.Contains(stderr.String(), "rebaseline CANDIDATE") {
		t.Fatalf("missing candidate was not a usage error: %d %q", code, stderr.String())
	}
}

func TestRebaselineIsHandledInProcessNotByTheInstalledCoordinator(t *testing.T) {
	var stdout, stderr bytes.Buffer
	handled, code := RunRootUpdateEffectCommand(context.Background(), []string{"runtime", "update", "rebaseline"}, &stdout, &stderr, nil)
	if !handled || code != 2 || !strings.Contains(stderr.String(), "rebaseline CANDIDATE") {
		t.Fatalf("rebaseline did not reach its own handler: %t %d %q", handled, code, stderr.String())
	}
}

func TestRebaselineSourceIsPinnedToTheCheckoutItRunsFrom(t *testing.T) {
	checkout, other := t.TempDir(), t.TempDir()
	old := repoRootFromWorkingDirectory
	t.Cleanup(func() { repoRootFromWorkingDirectory = old })
	repoRootFromWorkingDirectory = func() (string, error) { return checkout, nil }
	candidate := checkout + "/.vrooli/build/vrooli"
	if got, err := pinnedRebaselineSource(func() (string, error) { return checkout, nil }, candidate); err != nil || got != canonicalPath(checkout) {
		t.Fatalf("matching checkout = %q, %v", got, err)
	}
	if _, err := pinnedRebaselineSource(func() (string, error) { return other, nil }, candidate); err == nil || !strings.Contains(err.Error(), "override") {
		t.Fatalf("a disagreeing source root override was accepted: %v", err)
	}
	if _, err := pinnedRebaselineSource(func() (string, error) { return checkout, nil }, other+"/vrooli"); err == nil || !strings.Contains(err.Error(), "not built in the checkout") {
		t.Fatalf("a candidate from another checkout was accepted: %v", err)
	}
}

func TestRetainedRecoveryUsesProgressBoundSourceAfterCheckoutLoss(t *testing.T) {
	home := t.TempDir()
	manifest := strings.Repeat("a", 64)
	retained, err := cliinstall.RebaselineBundlePath(home, manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".vrooli", "state", "root-rebaseline", "progress.json")
	source := filepath.Join(t.TempDir(), "removed-checkout")
	p := rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("b", 64), Watchdog: strings.Repeat("c", 64), Bundle: manifest, SourceRoot: source}
	if err := rebaselineprogress.Write(path, p); err != nil {
		t.Fatal(err)
	}
	got, recovery, err := retainedRebaselineSource(home, filepath.Join(retained, "vrooli"))
	if err != nil || !recovery || got != source {
		t.Fatalf("retained source: %q %v %v", got, recovery, err)
	}
	if _, recovery, err := retainedRebaselineSource(home, filepath.Join(t.TempDir(), "vrooli")); err != nil || recovery {
		t.Fatalf("unbound candidate accepted: %v %v", recovery, err)
	}
	p.Complete = true
	if err := rebaselineprogress.Write(path, p); err != nil {
		t.Fatal(err)
	}
	if _, recovery, err := retainedRebaselineSource(home, filepath.Join(retained, "vrooli")); err != nil || recovery {
		t.Fatalf("completed record accepted as recovery: %v %v", recovery, err)
	}
}

func TestEnrollmentAndRetryRefuseAgentSessionsBeforeReachingTheWatchdog(t *testing.T) {
	withCaller(t, cliutil.CallerKindUnknown, "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/vrooli-agent-claude-code-946a880f99ff35ffbb4a0f3e77e3b25c.scope\n")
	for _, args := range [][]string{
		{"runtime", "update", "enroll"},
		{"runtime", "update", "disable", "--json"},
		{"runtime", "update", "retry", "/candidate"},
	} {
		resolved := false
		var stdout, stderr bytes.Buffer
		handled, code := RunRootUpdateEffectCommand(context.Background(), args, &stdout, &stderr, func() (string, error) {
			resolved = true
			return "/source", nil
		})
		if !handled || code != 1 || resolved || !strings.Contains(stderr.String(), "operator decision") || !strings.Contains(stderr.String(), "vrooli-agent-claude-code") {
			t.Fatalf("%v from an agent session was not refused first: handled=%t code=%d resolved=%t %q", args, handled, code, resolved, stderr.String())
		}
	}
}

func TestSameContractPreparedBundleUsesNormalInstaller(t *testing.T) {
	home := t.TempDir()
	var out, stderr bytes.Buffer
	called := 0
	install := func(source, bin, manifest string) error {
		called++
		if source != "/bundle" || bin != filepath.Join(home, ".vrooli", "bin") || manifest != "manifest" {
			t.Fatalf("wrong bundle: %q %q %q", source, bin, manifest)
		}
		return nil
	}
	handled, code := installSameContractBundle(fmt.Errorf("qualified: %w", runtimeapp.ErrRebaselineNotRequired), true, home, "/bundle/vrooli", "manifest", &out, &stderr, install)
	if !handled || code != 0 || called != 1 || !strings.Contains(out.String(), "no rebaseline was needed") {
		t.Fatalf("route: handled=%t code=%d calls=%d output=%q", handled, code, called, out.String())
	}
	for _, err := range []error{nil, errors.New("other refusal")} {
		if handled, _ := installSameContractBundle(err, true, home, "/bundle/vrooli", "manifest", &out, &stderr, install); handled {
			t.Fatal("unrelated result routed to installer")
		}
	}
	if handled, _ := installSameContractBundle(runtimeapp.ErrRebaselineNotRequired, false, home, "/bundle/vrooli", "manifest", &out, &stderr, install); handled {
		t.Fatal("unprepared candidate routed to bundle installer")
	}
	failure := errors.New("activation refused")
	handled, code = installSameContractBundle(runtimeapp.ErrRebaselineNotRequired, true, home, "/bundle/vrooli", "manifest", &out, &stderr, func(string, string, string) error { return failure })
	if !handled || code != 1 || !strings.Contains(stderr.String(), failure.Error()) {
		t.Fatalf("activation failure hidden: handled=%t code=%d stderr=%q", handled, code, stderr.String())
	}
}

func TestHubSameContractBundleDoesNotNeedNotificationHub(t *testing.T) {
	home, checkout := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	oldRepoRoot, oldDecisionFactory := repoRootFromWorkingDirectory, rootDecisionClientFactory
	oldRebaselineRoot, oldInstaller := rebaselineRootFn, installPreparedSameContractBundleFn
	t.Cleanup(func() {
		repoRootFromWorkingDirectory = oldRepoRoot
		rootDecisionClientFactory = oldDecisionFactory
		rebaselineRootFn = oldRebaselineRoot
		installPreparedSameContractBundleFn = oldInstaller
	})
	repoRootFromWorkingDirectory = func() (string, error) { return checkout, nil }
	var clientCalls, installCalls int
	var capturedRequest runtimeapp.RebaselineRequest
	rootDecisionClientFactory = func(context.Context) (*rootDecisionClient, error) {
		clientCalls++
		return nil, errors.New("Notification Hub unavailable")
	}
	rebaselineRootFn = func(_ context.Context, _ string, request runtimeapp.RebaselineRequest) (runtimeapp.RebaselineResult, error) {
		capturedRequest = request
		return runtimeapp.RebaselineResult{}, runtimeapp.ErrRebaselineNotRequired
	}
	bundle := filepath.Join(checkout, "bundle")
	source := filepath.Join(checkout, "source")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"vrooli", "vrooli-atomic-install", "vrooli-api", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog", "vrooli-watchdog"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("fixture "+name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := cliinstall.PrepareBundle(source, bundle); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	manifestDigest := sha256.Sum256(manifestBytes)
	manifest := hex.EncodeToString(manifestDigest[:])
	installPreparedSameContractBundleFn = func(gotBundle, bin, gotManifest, gotSource string, check func() error) error {
		installCalls++
		if gotBundle != bundle || gotManifest != manifest || gotSource != checkout || bin != filepath.Join(home, ".vrooli", "bin") || check != nil {
			t.Fatalf("wrong prepared installation: bundle=%q bin=%q manifest=%q source=%q check=%v", gotBundle, bin, gotManifest, gotSource, check != nil)
		}
		return nil
	}
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline", filepath.Join(bundle, "vrooli"), "--hub"}, stdin, &stdout, &stderr, func() (string, error) { return checkout, nil })
	if code != 0 || clientCalls != 0 || installCalls != 1 || !strings.Contains(stdout.String(), "no rebaseline was needed") {
		t.Fatalf("same-contract Hub route: code=%d client_calls=%d installs=%d stdout=%q stderr=%q", code, clientCalls, installCalls, stdout.String(), stderr.String())
	}
	prepared, err := readPreparedRootBundle(filepath.Join(bundle, "vrooli"))
	if err != nil {
		t.Fatal(err)
	}
	if capturedRequest.Candidate != filepath.Join(bundle, "vrooli") || capturedRequest.Source != checkout || !capturedRequest.HubDecision || capturedRequest.BundleSHA != manifest || capturedRequest.ExpectedCandidateSHA != prepared.rootSHA {
		t.Fatalf("prepared Hub request identity bindings: %+v", capturedRequest)
	}
	if capturedRequest.BeforeEffects == nil || capturedRequest.PrepareRecovery == nil || capturedRequest.AfterActivation == nil || capturedRequest.Confirm == nil {
		t.Fatalf("prepared Hub request omitted required authority/recovery callbacks: %+v", capturedRequest)
	}
}

// Uses actual prepared images and an isolated home. The caller seam models an
// operator pane; the character device is never read when the contract matches.
func TestSameContractRebaselineHandlerWithPreparedImages(t *testing.T) {
	image := os.Getenv("VROOLI_SAME_CONTRACT_TEST_IMAGE")
	if image == "" {
		t.Skip("set VROOLI_SAME_CONTRACT_TEST_IMAGE to a governed root image")
	}
	home, checkout := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	old := repoRootFromWorkingDirectory
	t.Cleanup(func() { repoRootFromWorkingDirectory = old })
	repoRootFromWorkingDirectory = func() (string, error) { return checkout, nil }
	bin := filepath.Join(home, ".vrooli", "bin")
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
	bundle := filepath.Join(checkout, "bundle")
	if err := cliinstall.PrepareBundle(filepath.Dir(image), bundle); err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	var out, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline", filepath.Join(bundle, "vrooli"), "--prepared"}, input, &out, &stderr, func() (string, error) { return checkout, nil })
	if code != 0 || !strings.Contains(out.String(), "no rebaseline was needed") {
		t.Fatalf("same-contract handler: code=%d stdout=%s stderr=%s", code, &out, &stderr)
	}
	for _, name := range []string{"vrooli-api", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog"} {
		if _, err := os.Stat(filepath.Join(bin, name)); err != nil {
			t.Fatalf("missing companion %s: %v", name, err)
		}
	}
}

func TestPreparedRuntimeSnapshotHandlerBindsCheckout(t *testing.T) {
	checkout, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	source := filepath.Join(checkout, "cmd", "vrooli", "main.go")
	if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fp, err := buildinfo.ComputeSourceFingerprintForPaths(checkout, buildinfo.FingerprintTargetsForExecutable("vrooli")...)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(home, ".vrooli", "state", "root-rebaseline", "snapshots", "rebaseline-bundle.fixture")
	if err := os.MkdirAll(snapshot, 0700); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(snapshot, "vrooli")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' '{\"cli_version\":\"1\",\"platform_version\":\"1\",\"build_identity\":\"%s\",\"runtime_home_contract_sha256\":\"%s\"}'\n", fp, strings.Repeat("a", 64))
	if err := os.WriteFile(candidate, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapshot, "SHA256SUMS"), []byte("invalid manifest"), 0600); err != nil {
		t.Fatal(err)
	}
	old := repoRootFromWorkingDirectory
	t.Cleanup(func() { repoRootFromWorkingDirectory = old })
	repoRootFromWorkingDirectory = func() (string, error) { return checkout, nil }
	withCaller(t, cliutil.CallerKindUnknown, operatorPaneCgroup)
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	var out, errout bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"vrooli", "runtime", "rebaseline", candidate, "--prepared"}, stdin, &out, &errout, func() (string, error) { return checkout, nil })
	if code == 0 || !strings.Contains(errout.String(), "manifest") {
		t.Fatalf("handler did not reach manifest verification: code=%d stderr=%s", code, errout.String())
	}
	_, check, err := pinnedRuntimeSnapshotSource(home, candidate, func() (string, error) { return checkout, nil })
	if err != nil || check == nil {
		t.Fatalf("source binding: %v", err)
	}
	if err := os.WriteFile(source, []byte("package main\nvar changed=true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := check(); err == nil {
		t.Fatal("changed checkout accepted")
	}
}
