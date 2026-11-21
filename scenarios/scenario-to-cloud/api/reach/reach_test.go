package reach

import (
	"context"
	"errors"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/identity"
)

type recording struct {
	calls int
	caps  Capabilities
}

func (r *recording) Exec(context.Context, identity.TargetRef, Command) (Result, error) {
	r.calls++
	return Result{ExitCode: 0, Stdout: "ok"}, nil
}

func (r *recording) Deliver(context.Context, identity.TargetRef, Delivery) (DeliveryReceipt, error) {
	r.calls++
	return DeliveryReceipt{}, nil
}

func (r *recording) Negotiate(context.Context, identity.TargetRef) (Capabilities, error) {
	r.calls++
	return r.caps, nil
}

type revocation struct {
	revoked bool
	reason  string
}

func (p revocation) Revoked(context.Context, identity.TargetRef) (bool, string, error) {
	return p.revoked, p.reason, nil
}

func bridgeTarget() identity.TargetRef {
	return identity.TargetRef{MachineID: "m-1", NodeID: "n-1", EnrollmentGeneration: 3, Transport: identity.TransportBridge}
}

func sshTarget() identity.TargetRef {
	return identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.10", User: "root", Workdir: "/root/Vrooli"}}
}

// [REQ:STC-P0-024] A validated verb never carries shell syntax; anything
// that could be read as shell by a joining transport is refused before any
// adapter is touched.
func TestValidateCommandRefusesShellSyntax(t *testing.T) {
	bad := []Command{
		{Verb: "cloud-target receipt get", Args: []string{"--deployment", "dep; rm -rf /"}},
		{Verb: "cloud-target receipt get", Args: []string{"$(whoami)"}},
		{Verb: "cloud-target receipt get", Args: []string{"a|b"}},
		{Verb: "cloud-target receipt get", Args: []string{"it's"}},
		{Verb: "cloud-target receipt get", Args: []string{""}},
		{Verb: "Cloud-Target", Args: nil},
		{Verb: "", Args: nil},
		{Verb: "receipt get && true", Args: nil},
	}
	for _, cmd := range bad {
		if err := ValidateCommand(cmd); !IsKind(err, KindInvalidArgument) {
			t.Fatalf("command %+v must be refused as invalid_request, got %v", cmd, err)
		}
	}
	good := Command{Verb: "cloud-target release stage", Args: []string{"--deployment", "dep-1", "--release", "sha256:abc", "--fence", "7", "--archive", "/root/Vrooli/.vrooli/cloud/releases/x.tar.gz"}}
	if err := ValidateCommand(good); err != nil {
		t.Fatalf("valid command refused: %v", err)
	}
	if got := good.Argv()[0]; got != "vrooli" {
		t.Fatalf("argv[0] = %q", got)
	}
}

// [REQ:STC-P0-024] The router dispatches on the explicit binding and never
// tries the other transport.
func TestRouterHonoursExplicitTransportWithoutFallback(t *testing.T) {
	bridge, sshr := &recording{}, &recording{}
	r := &Router{Bridge: bridge, SSH: sshr}
	cmd := Command{Verb: "cloud-target receipt get", Args: []string{"--deployment", "dep-1"}}
	if _, err := r.Exec(context.Background(), bridgeTarget(), cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Exec(context.Background(), sshTarget(), cmd); err != nil {
		t.Fatal(err)
	}
	if bridge.calls != 1 || sshr.calls != 1 {
		t.Fatalf("calls bridge=%d ssh=%d", bridge.calls, sshr.calls)
	}
	// Bridge selected but not configured: no SSH attempt.
	r2 := &Router{SSH: sshr}
	if _, err := r2.Exec(context.Background(), bridgeTarget(), cmd); !IsKind(err, KindUnavailable) {
		t.Fatalf("expected reach_unavailable, got %v", err)
	}
	if sshr.calls != 1 {
		t.Fatalf("ssh must not be used as a fallback, calls=%d", sshr.calls)
	}
	// No transport bound at all.
	if _, err := r.Exec(context.Background(), identity.TargetRef{Locator: identity.TargetLocator{Host: "h"}}, cmd); !IsKind(err, KindUnavailable) {
		t.Fatalf("expected reach_unavailable for unbound transport, got %v", err)
	}
	// Bridge binding without an enrolled node is a revoked/absent enrollment.
	if _, err := r.Exec(context.Background(), identity.TargetRef{MachineID: "m", Transport: identity.TransportBridge}, cmd); !IsKind(err, KindEnrollmentRevoked) {
		t.Fatalf("expected enrollment_revoked, got %v", err)
	}
	if bridge.calls != 1 {
		t.Fatalf("bridge must not be called for an unenrolled binding, calls=%d", bridge.calls)
	}
}

// [REQ:STC-P0-024] A revoked enrollment is refused before any transport call
// on either path (REACH-02, REACH-06).
func TestRouterRefusesRevokedTargetsWithZeroTransportCalls(t *testing.T) {
	bridge, sshr := &recording{}, &recording{}
	r := &Router{Bridge: bridge, SSH: sshr, Revocation: revocation{revoked: true, reason: "grant revoked by operator"}}
	cmd := Command{Verb: "cloud-target receipt get", Args: []string{"--deployment", "dep-1"}}
	for _, target := range []identity.TargetRef{bridgeTarget(), sshTarget()} {
		_, err := r.Exec(context.Background(), target, cmd)
		if !IsKind(err, KindEnrollmentRevoked) {
			t.Fatalf("target %s: expected enrollment_revoked, got %v", target.Key(), err)
		}
		if _, err := r.Negotiate(context.Background(), target); !IsKind(err, KindEnrollmentRevoked) {
			t.Fatalf("negotiate %s: expected enrollment_revoked, got %v", target.Key(), err)
		}
	}
	if bridge.calls != 0 || sshr.calls != 0 {
		t.Fatalf("no adapter may be called after revocation: bridge=%d ssh=%d", bridge.calls, sshr.calls)
	}
	api := APIError(errors.New("wrapped: " + (&Error{Kind: KindEnrollmentRevoked}).Error()))
	if api.Code != apierrors.CodeInternal {
		t.Fatalf("a non-reach error must not be classified as reach: %s", api.Code)
	}
	typed := APIError(&Error{Kind: KindEnrollmentRevoked, Transport: identity.TransportBridge, Target: "machine:m-1"})
	if typed.Code != apierrors.CodeEnrollmentRevoked || typed.Details["transport"] != identity.TransportBridge {
		t.Fatalf("typed error = %+v", typed)
	}
}

func TestRequireScopeFollowsCatalogWildcards(t *testing.T) {
	if err := RequireScope([]string{"vrooli:read"}, "vrooli:write"); !IsKind(err, KindScopeMissing) {
		t.Fatalf("expected scope_missing, got %v", err)
	}
	for _, held := range [][]string{{"vrooli:write"}, {"*"}, {"vrooli:*"}, {"*:write"}} {
		if err := RequireScope(held, "vrooli:write"); err != nil {
			t.Fatalf("held %v must satisfy vrooli:write: %v", held, err)
		}
	}
	if err := RequireScope(nil, ""); err != nil {
		t.Fatal(err)
	}
}

// Negotiation surfaces node offline, protocol mismatch and missing scope as
// distinct states through the router (REACH-04 groundwork).
func TestNegotiateDistinctStatesPassThrough(t *testing.T) {
	bridge := &recording{caps: Capabilities{Transport: identity.TransportBridge, Online: true, Scopes: []string{"vrooli:read"}}}
	r := &Router{Bridge: bridge}
	caps, err := r.Negotiate(context.Background(), bridgeTarget())
	if err != nil || !caps.Online {
		t.Fatalf("caps=%+v err=%v", caps, err)
	}
	if err := RequireScope(caps.Scopes, "vrooli:write"); !IsKind(err, KindScopeMissing) {
		t.Fatalf("expected scope_missing, got %v", err)
	}
}
