package runtimehandlers

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/rootauthority"
)

func forceTestDecision() rootauthority.Decision {
	now := time.Now().UTC()
	return rootauthority.Decision{
		GrantID: "grant-force", ApprovedBy: "owner", RunID: "run-1",
		Operation: rootauthority.Operation, Target: rootauthority.Target("host", "/home/owner/.vrooli"),
		VerifiedAt: now, ExpiresAt: now.Add(time.Hour),
	}
}

func withForceSeams(t *testing.T) (candidate string, source func() (string, error)) {
	t.Helper()
	oldVerify, oldPrepare, oldRestore, oldRebaseline := verifyRootTransitionAuthorityFn, prepareForceControlsFn, restoreForceControlsFn, rebaselineRootFn
	oldRepoRoot := repoRootFromWorkingDirectory
	t.Cleanup(func() {
		verifyRootTransitionAuthorityFn, prepareForceControlsFn, restoreForceControlsFn, rebaselineRootFn = oldVerify, oldPrepare, oldRestore, oldRebaseline
		repoRootFromWorkingDirectory = oldRepoRoot
	})
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	checkout := t.TempDir()
	repoRootFromWorkingDirectory = func() (string, error) { return checkout, nil }
	return filepath.Join(checkout, "candidate"), func() (string, error) { return checkout, nil }
}

func forceArgs(candidate string) []string {
	return []string{"runtime", "update", "rebaseline", candidate, "--force", "--reason=additive schema"}
}

func TestForceRebaselineNeedsAStandingGrant(t *testing.T) {
	candidate, source := withForceSeams(t)
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	resolved, prepared, ran := false, false, false
	verifyRootTransitionAuthorityFn = func(_ context.Context, target string) (rootauthority.Decision, error) {
		return rootauthority.Decision{}, &rootauthority.Refusal{Target: target, Reason: "no approved grant was found"}
	}
	prepareForceControlsFn = func(context.Context, string, runtimeapp.ForceRequest) (runtimeapp.ForceControls, error) {
		prepared = true
		return runtimeapp.ForceControls{}, nil
	}
	rebaselineRootFn = func(context.Context, string, runtimeapp.RebaselineRequest) (runtimeapp.RebaselineResult, error) {
		ran = true
		return runtimeapp.RebaselineResult{}, nil
	}
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), forceArgs(candidate), os.Stdin, &stdout, &stderr, func() (string, error) {
		resolved = true
		return source()
	})
	if code != 1 || resolved || prepared || ran {
		t.Fatalf("an unauthorized force reached the transition: code=%d resolved=%t prepared=%t ran=%t", code, resolved, prepared, ran)
	}
	if !strings.Contains(stderr.String(), "vrooli agent authorize") || !strings.Contains(stderr.String(), rootauthority.Policy) {
		t.Fatalf("refusal does not name the request command: %q", stderr.String())
	}
}

func TestForceRebaselineRefusesWhenControlsCannotBePrepared(t *testing.T) {
	candidate, source := withForceSeams(t)
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	ran := false
	verifyRootTransitionAuthorityFn = func(context.Context, string) (rootauthority.Decision, error) {
		return forceTestDecision(), nil
	}
	prepareForceControlsFn = func(context.Context, string, runtimeapp.ForceRequest) (runtimeapp.ForceControls, error) {
		return runtimeapp.ForceControls{}, errors.New("snapshot unavailable")
	}
	rebaselineRootFn = func(context.Context, string, runtimeapp.RebaselineRequest) (runtimeapp.RebaselineResult, error) {
		ran = true
		return runtimeapp.RebaselineResult{}, nil
	}
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), forceArgs(candidate), os.Stdin, &stdout, &stderr, source)
	if code != 1 || ran || !strings.Contains(stderr.String(), "snapshot unavailable") {
		t.Fatalf("a force without controls ran: code=%d ran=%t stderr=%q", code, ran, stderr.String())
	}
}

func TestForceRebaselineRestoresStateOnFailure(t *testing.T) {
	candidate, source := withForceSeams(t)
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	restored := false
	var captured runtimeapp.ForceRequest
	verifyRootTransitionAuthorityFn = func(context.Context, string) (rootauthority.Decision, error) {
		return forceTestDecision(), nil
	}
	prepareForceControlsFn = func(_ context.Context, _ string, request runtimeapp.ForceRequest) (runtimeapp.ForceControls, error) {
		captured = request
		return runtimeapp.ForceControls{Dir: "/snapshot", ReceiptAt: "/receipt.json", Receipt: runtimeapp.ForceReceipt{Authority: request.Authority}}, nil
	}
	restoreForceControlsFn = func(context.Context, string, runtimeapp.ForceControls) error {
		restored = true
		return nil
	}
	rebaselineRootFn = func(_ context.Context, _ string, request runtimeapp.RebaselineRequest) (runtimeapp.RebaselineResult, error) {
		if request.RequireTransitionCaller == nil || request.RequireTransitionCaller() != nil {
			t.Fatal("a forced transition must not re-run the operator terminal gate")
		}
		return runtimeapp.RebaselineResult{}, errors.New("verification failed")
	}
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), forceArgs(candidate), os.Stdin, &stdout, &stderr, source)
	if code != 1 || !restored {
		t.Fatalf("failed forced transition did not restore state: code=%d restored=%t", code, restored)
	}
	if captured.Reason != "additive schema" || captured.Authority.GrantID != "grant-force" {
		t.Fatalf("force request lost its authority or reason: %+v", captured)
	}
	if !strings.Contains(stderr.String(), "verification failed") || !strings.Contains(stdout.String(), "/snapshot") {
		t.Fatalf("failure was not reported with its snapshot: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestForceRebaselineKeepsSnapshotOnSuccess(t *testing.T) {
	candidate, source := withForceSeams(t)
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	restored := false
	verifyRootTransitionAuthorityFn = func(context.Context, string) (rootauthority.Decision, error) {
		return forceTestDecision(), nil
	}
	prepareForceControlsFn = func(context.Context, string, runtimeapp.ForceRequest) (runtimeapp.ForceControls, error) {
		return runtimeapp.ForceControls{Dir: "/snapshot", ReceiptAt: "/receipt.json"}, nil
	}
	restoreForceControlsFn = func(context.Context, string, runtimeapp.ForceControls) error {
		restored = true
		return nil
	}
	rebaselineRootFn = func(context.Context, string, runtimeapp.RebaselineRequest) (runtimeapp.RebaselineResult, error) {
		return runtimeapp.RebaselineResult{Build: "build", Contract: "contract", RetainedDir: "/retained"}, nil
	}
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), forceArgs(candidate), os.Stdin, &stdout, &stderr, source)
	if code != 0 || restored {
		t.Fatalf("successful forced transition restored or failed: code=%d restored=%t stderr=%q", code, restored, stderr.String())
	}
	if !strings.Contains(stdout.String(), "/receipt.json") || !strings.Contains(stdout.String(), "/snapshot") {
		t.Fatalf("success did not report the receipt and snapshot: %q", stdout.String())
	}
}

func TestForceRebaselineRejectsAMissingReason(t *testing.T) {
	candidate, source := withForceSeams(t)
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	// --force without --reason is still accepted syntactically; the reason is
	// recorded as empty so the receipt shows an unexplained transition.
	var captured runtimeapp.ForceRequest
	verifyRootTransitionAuthorityFn = func(context.Context, string) (rootauthority.Decision, error) {
		return forceTestDecision(), nil
	}
	prepareForceControlsFn = func(_ context.Context, _ string, request runtimeapp.ForceRequest) (runtimeapp.ForceControls, error) {
		captured = request
		return runtimeapp.ForceControls{}, nil
	}
	rebaselineRootFn = func(context.Context, string, runtimeapp.RebaselineRequest) (runtimeapp.RebaselineResult, error) {
		return runtimeapp.RebaselineResult{Build: "b", Contract: "c"}, nil
	}
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), []string{"runtime", "update", "rebaseline", candidate, "--force"}, os.Stdin, &stdout, &stderr, source)
	if code != 0 || captured.Reason != "" {
		t.Fatalf("unexpected result: code=%d reason=%q stderr=%q", code, captured.Reason, stderr.String())
	}
}

// A verified standing grant is the consent. If a forced transition still waited
// for a TTY answer it would be unusable from an agent, and the verified grant
// would be advisory rather than authority.
func TestForceRebaselineDoesNotWaitForATerminalAnswer(t *testing.T) {
	candidate, source := withForceSeams(t)
	withCaller(t, cliutil.CallerKindExternalAgent, "")
	verifyRootTransitionAuthorityFn = func(context.Context, string) (rootauthority.Decision, error) {
		return forceTestDecision(), nil
	}
	prepareForceControlsFn = func(context.Context, string, runtimeapp.ForceRequest) (runtimeapp.ForceControls, error) {
		return runtimeapp.ForceControls{Dir: "/snapshot", ReceiptAt: "/receipt.json"}, nil
	}
	confirmed := false
	rebaselineRootFn = func(_ context.Context, _ string, request runtimeapp.RebaselineRequest) (runtimeapp.RebaselineResult, error) {
		if request.Confirm == nil {
			t.Fatal("the transition was not given a confirmation seam")
		}
		ok, err := request.Confirm(runtimeapp.RebaselinePlan{})
		if err != nil {
			t.Fatalf("confirm: %v", err)
		}
		confirmed = ok
		return runtimeapp.RebaselineResult{Build: "b", Contract: "c"}, nil
	}
	var stdout, stderr bytes.Buffer
	code := runRootRebaseline(context.Background(), forceArgs(candidate), os.Stdin, &stdout, &stderr, source)
	if code != 0 || !confirmed {
		t.Fatalf("a verified forced transition did not confirm itself: code=%d confirmed=%t stderr=%q", code, confirmed, stderr.String())
	}
}
