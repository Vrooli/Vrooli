package credentials

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// [REQ:STC-P0-032] P13-A01: an ordinary redeploy preserves generated values
// the target already holds and never re-sends them.
func TestMaterializePreservesGeneratedValuesOnRedeploy(t *testing.T) {
	h := newHarness(t)
	first := h.materialize(map[string]string{h.bindingFor("api-token").ID: "canary-mailer-token-1"})
	if len(first.Materialized) != 3 || len(first.Preserved) != 0 {
		t.Fatalf("first materialize = %+v", first)
	}
	store := h.bindingFor("password")
	if store.Version.Number != 1 || store.State != domain.CredentialBindingMaterialized {
		t.Fatalf("store binding after first deploy = %+v", store)
	}
	valueBefore := h.target.values[store.ID]
	ingestsBefore := strings.Count(h.argvJoined(), "credential ingest")

	second := h.materialize(nil)
	if len(second.Preserved) != 3 || len(second.Materialized) != 0 {
		t.Fatalf("second materialize = %+v", second)
	}
	if h.target.values[store.ID] != valueBefore {
		t.Fatal("redeploy replaced a generated value the target still held")
	}
	if strings.Count(h.argvJoined(), "credential ingest") != ingestsBefore {
		t.Fatal("redeploy re-sent a preserved credential")
	}
	if h.bindingFor("password").Version.Number != 1 {
		t.Fatal("redeploy minted a new version for a preserved binding")
	}
}

// [REQ:STC-P0-032] P13-A02 / SECRET-02: every intended consumer acknowledges
// the new version before the predecessor is revoked, in the documented order.
func TestRotationVerifiesConsumersBeforePredecessorRetirement(t *testing.T) {
	h := newHarness(t)
	h.materialize(nil)
	binding := h.bindingFor("password")
	rotation, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	h.mustState(rotation, domain.RotationComplete)
	wantSteps := []string{"new-version", "provider-prepare", "consumer-update", "verify", "revoke-predecessor", "complete"}
	for i, step := range wantSteps {
		if rotation.Receipts[i].Step != step {
			t.Fatalf("receipt %d = %s, want %s (%s)", i, rotation.Receipts[i].Step, step, receiptSummary(rotation))
		}
	}
	verifyIdx, revokeIdx := -1, -1
	for i, r := range rotation.Receipts {
		if r.Step == "verify" && r.Outcome == "verified" {
			verifyIdx = i
		}
		if r.Step == "revoke-predecessor" && r.Outcome == "revoked" {
			revokeIdx = i
		}
	}
	if verifyIdx < 0 || revokeIdx < verifyIdx {
		t.Fatalf("predecessor revoked before verification: %s", receiptSummary(rotation))
	}
	after := h.bindingFor("password")
	if after.Version.Number != 2 || after.PreviousVersion != nil {
		t.Fatalf("binding after rotation = %+v", after)
	}
	if len(h.target.acks[binding.ID]) != 2 {
		t.Fatalf("acks on target = %v, want both consumers", h.target.acks[binding.ID])
	}
	if len(h.restarter.restarted) != 1 {
		t.Fatalf("consumers restarted %d times, want 1", len(h.restarter.restarted))
	}
	if got := h.target.revoked[binding.ID]; len(got) != 1 || got[0] != 1 {
		t.Fatalf("revoked versions on target = %v, want [1]", got)
	}
	for _, c := range rotation.Consumers {
		if c.State != domain.ConsumerAcknowledged || c.Version != 2 {
			t.Fatalf("consumer %s = %+v", c.Consumer, c)
		}
	}
	// The database role was prepared with a SCRAM verifier on stdin, never
	// the password in argv.
	if !strings.Contains(h.argvJoined(), "psql") {
		t.Fatal("database owner argv was not run")
	}
	for _, in := range h.target.stdin {
		if strings.HasPrefix(in, "ALTER ROLE") && !strings.Contains(in, "SCRAM-SHA-256$") {
			t.Fatalf("role update carried a plaintext password: %q", in)
		}
	}
}

// [REQ:STC-P0-032] SECRET-03: a provider rejection leaves the predecessor as
// the active version and distributes nothing.
func TestProviderRejectionRetainsPredecessor(t *testing.T) {
	h := newHarness(t)
	h.svc.Providers[domain.CredentialClassExternalAPICredential] = &ExternalAPIProvider{Name: "mailer", Probe: func(context.Context, string) error { return errors.New("scope missing: mail.send") }}
	h.materialize(map[string]string{h.bindingFor("api-token").ID: "canary-mailer-token-1"})
	binding := h.bindingFor("api-token")
	ingestsBefore := strings.Count(h.argvJoined(), "credential ingest")
	rotation, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID, Value: "canary-mailer-token-2"})
	if !apierrors.Is(err, CodeDistributionFailed) {
		t.Fatalf("rotate error = %v, want %s", err, CodeDistributionFailed)
	}
	h.mustState(rotation, domain.RotationFailed)
	if rotation.Error == nil || !strings.Contains(rotation.Error.Message, "scope missing") {
		t.Fatalf("rotation error = %+v", rotation.Error)
	}
	if h.bindingFor("api-token").Version.Number != 1 {
		t.Fatal("binding version advanced after a provider rejection")
	}
	if h.target.values[binding.ID] != "canary-mailer-token-1" {
		t.Fatal("target value changed after a provider rejection")
	}
	if strings.Count(h.argvJoined(), "credential ingest") != ingestsBefore {
		t.Fatal("a rejected version was distributed")
	}
	if prepared := rotation.Receipts[len(rotation.Receipts)-1].Details["predecessor_retained"]; prepared != true {
		t.Fatalf("failure receipt does not state predecessor retention: %+v", rotation.Receipts[len(rotation.Receipts)-1])
	}
}

// [REQ:STC-P0-032] P13-A05: a consumer the target cannot confirm keeps the
// rotation explicitly incomplete; the predecessor stays active until every
// consumer acknowledges, and a later resume completes it.
func TestUnreachableConsumerKeepsRotationIncomplete(t *testing.T) {
	h := newHarness(t)
	h.materialize(nil)
	binding := h.bindingFor("password")
	h.target.unreachableConsumers["scenario:app"] = true
	rotation, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	h.mustState(rotation, domain.RotationConsumersUpdated)
	if !rotation.Incomplete() {
		t.Fatal("rotation reports complete with an unreachable consumer")
	}
	if len(rotation.Unreached) != 1 || rotation.Unreached[0] != "scenario:app" {
		t.Fatalf("unreached = %v", rotation.Unreached)
	}
	if h.bindingFor("password").Version.Number != 1 {
		t.Fatal("new version promoted before every consumer acknowledged")
	}
	if got := h.target.revoked[binding.ID]; len(got) != 0 {
		t.Fatalf("predecessor revoked while rotation incomplete: %v", got)
	}
	last := rotation.Receipts[len(rotation.Receipts)-1]
	if last.Outcome != "incomplete" || len(last.Limitations) == 0 {
		t.Fatalf("incomplete receipt = %+v", last)
	}

	delete(h.target.unreachableConsumers, "scenario:app")
	resumed, err := h.svc.Resume(context.Background(), h.targetRef(), rotation.ID, ResumeInput{})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	h.mustState(resumed, domain.RotationComplete)
	if h.bindingFor("password").Version.Number != 2 {
		t.Fatal("resume did not promote the new version")
	}
}

// [REQ:STC-P0-032] P13-A04: a revoked grant denies new distribution with a
// typed forbidden_revoked and no fallback; the predecessor is retained.
func TestRevokedGrantDeniesDistribution(t *testing.T) {
	h := newHarness(t)
	h.materialize(nil)
	binding := h.bindingFor("password")
	h.target.revokedNode = true
	rotation, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID})
	if !apierrors.Is(err, apierrors.CodeForbiddenRevoked) {
		t.Fatalf("rotate error = %v, want forbidden_revoked", err)
	}
	h.mustState(rotation, domain.RotationFailed)
	if rotation.Error == nil || rotation.Error.Code != apierrors.CodeForbiddenRevoked {
		t.Fatalf("rotation error = %+v", rotation.Error)
	}
	if h.bindingFor("password").Version.Number != 1 || h.target.versions[binding.ID] != 1 {
		t.Fatal("a revoked grant still changed the active version")
	}
	// Materialize under a revoked grant is refused the same way.
	h.target.values = map[string]string{}
	if _, err := h.svc.Materialize(context.Background(), MaterializeRequest{Target: h.targetRef(), Bindings: h.plan()}); !apierrors.Is(err, apierrors.CodeForbiddenRevoked) {
		t.Fatalf("materialize under revoked grant = %v", err)
	}
}

// [REQ:STC-P0-032] SECRET-04 / P13-A05: online revocation purges the target
// and states its limit; an unreachable target leaves revocation_incomplete
// with the node retained until a later resume confirms it.
func TestRevokeOnlineAndUnreachableAreClassifiedSeparately(t *testing.T) {
	h := newHarness(t)
	h.materialize(nil)
	online := h.bindingFor("password")
	rotation, err := h.svc.Revoke(context.Background(), RevokeBindingRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: online.ID})
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	h.mustState(rotation, domain.RotationComplete)
	if _, still := h.target.values[online.ID]; still {
		t.Fatal("online revocation left the value on the target")
	}
	if h.bindingFor("password").State != domain.CredentialBindingRevoked {
		t.Fatal("binding not marked revoked")
	}
	var stated bool
	for _, r := range rotation.Receipts {
		for _, l := range r.Limitations {
			if l == RevocationLimitation {
				stated = true
			}
		}
	}
	if !stated {
		t.Fatal("revocation receipt does not state what revocation cannot prove")
	}
	if _, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: online.ID}); !apierrors.Is(err, CodeRotationRefused) {
		t.Fatalf("rotation of a revoked binding = %v, want refusal", err)
	}

	unreachable := h.bindingFor("signing-key")
	h.target.unreachable = true
	incomplete, err := h.svc.Revoke(context.Background(), RevokeBindingRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: unreachable.ID, RequestKey: "revoke-signing"})
	if !apierrors.Is(err, CodeRevocationIncomplete) {
		t.Fatalf("unreachable revoke error = %v, want %s", err, CodeRevocationIncomplete)
	}
	h.mustState(incomplete, domain.RotationRevocationIncomplete)
	if len(incomplete.Unreached) != 1 || incomplete.Unreached[0] != h.targetRef().NodeLabel() {
		t.Fatalf("unreached = %v", incomplete.Unreached)
	}
	if h.bindingFor("signing-key").State == domain.CredentialBindingRevoked {
		t.Fatal("binding marked revoked although the target was never reached")
	}
	h.target.unreachable = false
	confirmed, err := h.svc.Resume(context.Background(), h.targetRef(), incomplete.ID, ResumeInput{})
	if err != nil {
		t.Fatalf("resume revoke: %v", err)
	}
	h.mustState(confirmed, domain.RotationComplete)
	if len(confirmed.Unreached) != 0 {
		t.Fatalf("unreached retained after confirmation: %v", confirmed.Unreached)
	}
}

// [REQ:STC-P0-032] Step 10: a provider without a revocation API hands off to
// the operator with a durable reference and a truthful pending state.
func TestExternalCredentialHandsOffWhenProviderHasNoRevokeAPI(t *testing.T) {
	h := newHarness(t)
	h.svc.Providers[domain.CredentialClassExternalAPICredential] = &ExternalAPIProvider{Name: "mailer", Probe: func(context.Context, string) error { return nil }, Instruction: "revoke the old token in the mailer console"}
	h.materialize(map[string]string{h.bindingFor("api-token").ID: "canary-mailer-token-1"})
	binding := h.bindingFor("api-token")
	rotation, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID, Value: "canary-mailer-token-2"})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	h.mustState(rotation, domain.RotationPendingOperatorInput)
	if rotation.PendingOperatorInput == nil || rotation.PendingOperatorInput.Reference == "" || rotation.PendingOperatorInput.Provider != "mailer" {
		t.Fatalf("handoff = %+v", rotation.PendingOperatorInput)
	}
	if h.bindingFor("api-token").Version.Number != 2 {
		t.Fatal("new version not active while waiting for the operator")
	}
	if _, err := h.svc.Resume(context.Background(), h.targetRef(), rotation.ID, ResumeInput{}); !apierrors.Is(err, CodePendingOperatorInput) {
		t.Fatalf("resume without confirmation = %v, want %s", err, CodePendingOperatorInput)
	}
	done, err := h.svc.Resume(context.Background(), h.targetRef(), rotation.ID, ResumeInput{OperatorConfirmed: true})
	if err != nil {
		t.Fatalf("resume with confirmation: %v", err)
	}
	h.mustState(done, domain.RotationComplete)
	if got := h.target.revoked[binding.ID]; len(got) != 1 || got[0] != 1 {
		t.Fatalf("predecessor purge on target = %v", got)
	}
}

// [REQ:STC-P0-032] A signing key keeps its predecessor verifiable for the
// overlap window and retires it only after the window elapses.
func TestSigningKeyRespectsOverlapWindow(t *testing.T) {
	h := newHarness(t)
	h.svc.Providers[domain.CredentialClassSigningKey] = &SigningKeyProvider{OverlapWindow: time.Hour, Now: func() time.Time { return h.clock }}
	h.materialize(nil)
	binding := h.bindingFor("signing-key")
	rotation, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	h.mustState(rotation, domain.RotationVerified)
	if rotation.ResumeAfter == nil {
		t.Fatal("no resume_after recorded for the overlap window")
	}
	if h.bindingFor("signing-key").PreviousVersion == nil {
		t.Fatal("predecessor discarded during the overlap window")
	}
	h.clock = h.clock.Add(2 * time.Hour)
	done, err := h.svc.Resume(context.Background(), h.targetRef(), rotation.ID, ResumeInput{})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	h.mustState(done, domain.RotationComplete)
}

// [REQ:STC-P0-032] Machine enrollment credentials are owned by Bridge; local
// rotation, revocation and break-glass are refused.
func TestMachineEnrollmentRotationIsRefused(t *testing.T) {
	h := newHarness(t)
	binding := domain.CredentialBinding{ID: BindingID("dep-1", domain.CredentialDescriptor{LogicalID: "vrooli/bridge", Field: "enrollment-token"}), DeploymentID: "dep-1", Descriptor: domain.CredentialDescriptor{LogicalID: "vrooli/bridge", Field: "enrollment-token"}, Class: domain.CredentialClassMachineEnrollment, Version: domain.CredentialVersion{Number: 1}, State: domain.CredentialBindingMaterialized}
	if err := h.store.UpsertBinding(context.Background(), &binding); err != nil {
		t.Fatal(err)
	}
	_, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID})
	if !apierrors.Is(err, CodeRotationRefused) {
		t.Fatalf("rotate = %v", err)
	}
	if typed := apierrors.As(err); typed == nil || typed.NextAction == nil || typed.NextAction.Owner != "vrooli-bridge" {
		t.Fatalf("refusal does not name Bridge as the owner: %v", err)
	}
	if _, err := h.svc.Revoke(context.Background(), RevokeBindingRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID}); !apierrors.Is(err, CodeRotationRefused) {
		t.Fatalf("revoke = %v", err)
	}
	if strings.Contains(h.argvJoined(), "enrollment") {
		t.Fatal("a refused enrollment rotation reached the target")
	}
}

// [REQ:STC-P0-032] P13-A06 / SECRET-07: host loss is recovered onto a
// replacement host from the encrypted bundle; a locked replacement store
// fails closed with credential_store_locked and a bundle that does not cover
// every binding is refused.
func TestRecoveryRestoresOntoReplacementHost(t *testing.T) {
	h := newHarness(t)
	h.materialize(map[string]string{h.bindingFor("api-token").ID: "canary-mailer-token-1"})
	bindings, _ := h.store.ListBindings(context.Background(), "dep-1")
	covers := make([]domain.CredentialDescriptor, 0, len(bindings))
	for _, b := range bindings {
		covers = append(covers, b.Descriptor)
	}
	replacement := newFakeTarget()
	h.svc.Distributor = &fakeDistributor{target: replacement}
	recovery := &fakeRecovery{covers: covers, restoreTo: replacement, bindings: bindings}
	h.svc.Recovery = recovery
	newTarget := h.targetRef()
	newTarget.Ref.Locator.Host = "203.0.113.99"

	if _, err := h.svc.Recover(context.Background(), RecoverRequest{DeploymentID: "dep-1", NewTarget: newTarget, BundleRef: "/root/recovery.bundle"}); !apierrors.Is(err, apierrors.CodeNeedsInput) {
		t.Fatalf("recover without passphrase = %v", err)
	}

	recovery.locked = true
	_, err := h.svc.Recover(context.Background(), RecoverRequest{DeploymentID: "dep-1", NewTarget: newTarget, BundleRef: "/root/recovery.bundle", Passphrase: "canary-passphrase"})
	if !apierrors.Is(err, CodeStoreLocked) {
		t.Fatalf("recover onto locked store = %v, want %s", err, CodeStoreLocked)
	}
	if len(replacement.values) != 0 {
		t.Fatal("a locked store still received values")
	}
	recovery.locked = false

	partial := &fakeRecovery{covers: covers[:1], restoreTo: replacement, bindings: bindings}
	h.svc.Recovery = partial
	failed, err := h.svc.Recover(context.Background(), RecoverRequest{DeploymentID: "dep-1", NewTarget: newTarget, BundleRef: "/root/partial.bundle", Passphrase: "canary-passphrase"})
	if !apierrors.Is(err, CodeRecoveryFailed) {
		t.Fatalf("partial bundle = %v", err)
	}
	h.mustState(failed, domain.RotationFailed)
	if missing, _ := failed.Receipts[len(failed.Receipts)-1].Details["missing"].([]string); len(missing) != 2 {
		t.Fatalf("missing descriptors = %v", failed.Receipts[len(failed.Receipts)-1].Details)
	}
	for _, s := range partial.seen {
		if strings.HasPrefix(s, "restore:") {
			t.Fatal("restore ran for a bundle that does not cover every binding")
		}
	}

	h.svc.Recovery = recovery
	done, err := h.svc.Recover(context.Background(), RecoverRequest{DeploymentID: "dep-1", NewTarget: newTarget, BundleRef: "/root/recovery.bundle", Passphrase: "canary-passphrase"})
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	h.mustState(done, domain.RotationComplete)
	if len(replacement.values) != 3 {
		t.Fatalf("replacement holds %d values, want 3", len(replacement.values))
	}
	for _, b := range bindings {
		after, _ := h.store.GetBinding(context.Background(), "dep-1", b.ID)
		if after.Version.Number != b.Version.Number {
			t.Fatalf("recovery changed the version of %s", b.Descriptor.Address())
		}
	}
	if strings.Contains(h.argvJoined(), "canary-passphrase") {
		t.Fatal("passphrase reached argv")
	}
}

// [REQ:STC-P0-032] SECRET-08: break-glass access needs the exact
// confirmation, a scope, an operator and a bounded window; it is audited, and
// the sweep rotates the emergency version away once the window closes.
func TestBreakGlassIsScopedTimeBoundedAuditedAndAutoRevoked(t *testing.T) {
	h := newHarness(t)
	h.materialize(nil)
	binding := h.bindingFor("password")
	base := BreakGlassRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID, Scope: "incident-42 restore store access", Window: 30 * time.Minute, Operator: "operator:alice"}

	wrong := base
	wrong.Confirmation = "yes"
	if _, err := h.svc.BreakGlass(context.Background(), wrong); !apierrors.Is(err, CodeConfirmationRequired) {
		t.Fatalf("wrong confirmation = %v", err)
	}
	tooLong := base
	tooLong.Confirmation = ExpectedBreakGlassConfirmation(binding.ID)
	tooLong.Window = MaxBreakGlassWindow + time.Minute
	if _, err := h.svc.BreakGlass(context.Background(), tooLong); !apierrors.Is(err, apierrors.CodeInvalidRequest) {
		t.Fatalf("window above the maximum = %v", err)
	}
	if h.bindingFor("password").Version.Number != 1 {
		t.Fatal("a refused break-glass changed the active version")
	}

	ok := base
	ok.Confirmation = ExpectedBreakGlassConfirmation(binding.ID)
	rotation, err := h.svc.BreakGlass(context.Background(), ok)
	if err != nil {
		t.Fatalf("break-glass: %v", err)
	}
	h.mustState(rotation, domain.RotationComplete)
	if rotation.Kind != domain.CredentialOperationBreakGlass || rotation.BreakGlass == nil {
		t.Fatalf("rotation = %+v", rotation)
	}
	if rotation.BreakGlass.Scope != ok.Scope || rotation.BreakGlass.Operator != ok.Operator || !rotation.BreakGlass.ExpiresAt.Equal(rotation.BreakGlass.IssuedAt.Add(30*time.Minute)) {
		t.Fatalf("window = %+v", rotation.BreakGlass)
	}
	if rotation.Receipts[0].Step != "break-glass-audit" || rotation.Receipts[0].Details["operator"] != "operator:alice" {
		t.Fatalf("audit receipt = %+v", rotation.Receipts[0])
	}
	emergency := h.bindingFor("password").Version.Number
	if emergency != 2 {
		t.Fatalf("emergency version = %d", emergency)
	}

	early, err := h.svc.SweepBreakGlass(context.Background(), "dep-1", h.targetRef())
	if err != nil || len(early) != 0 {
		t.Fatalf("sweep before expiry = %v, %v", early, err)
	}
	h.clock = h.clock.Add(time.Hour)
	followOns, err := h.svc.SweepBreakGlass(context.Background(), "dep-1", h.targetRef())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(followOns) != 1 {
		t.Fatalf("follow-on rotations = %d", len(followOns))
	}
	h.mustState(followOns[0], domain.RotationComplete)
	if h.bindingFor("password").Version.Number != 3 {
		t.Fatal("emergency version still active after the window")
	}
	swept, _ := h.store.GetRotation(context.Background(), rotation.ID)
	if swept.BreakGlass.AutoRevokedAt == nil {
		t.Fatal("break-glass window not marked auto-revoked")
	}
	found := false
	for _, v := range h.target.revoked[binding.ID] {
		if v == emergency {
			found = true
		}
	}
	if !found {
		t.Fatalf("emergency version %d not revoked on the target: %v", emergency, h.target.revoked[binding.ID])
	}
	again, err := h.svc.SweepBreakGlass(context.Background(), "dep-1", h.targetRef())
	if err != nil || len(again) != 0 {
		t.Fatalf("second sweep = %v, %v", again, err)
	}
}

// [REQ:STC-P0-032] SECRET-06: a second lifecycle operation on a binding with
// an open one is a typed conflict, and a deploy during an open rotation
// preserves rather than mints, so no unexplained mixed state can arise.
func TestConcurrentRotationAndDeploymentAreSerialisedByTheBinding(t *testing.T) {
	h := newHarness(t)
	h.materialize(nil)
	binding := h.bindingFor("password")
	h.target.unreachableConsumers["scenario:app"] = true
	open, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	h.mustState(open, domain.RotationConsumersUpdated)
	if _, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID}); !apierrors.Is(err, CodeRotationConflict) {
		t.Fatalf("second rotation = %v, want %s", err, CodeRotationConflict)
	}
	replay, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID, RequestKey: open.ID})
	if err != nil || replay.ID != open.ID {
		t.Fatalf("replay by request key = %v, %v", replay, err)
	}
	if _, err := h.svc.BreakGlass(context.Background(), BreakGlassRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: binding.ID, Scope: "x", Operator: "o", Window: time.Minute, Confirmation: ExpectedBreakGlassConfirmation(binding.ID)}); !apierrors.Is(err, CodeRotationConflict) {
		t.Fatalf("break-glass during open rotation = %v", err)
	}
	result := h.materialize(nil)
	if len(result.Materialized) != 0 {
		t.Fatalf("deploy during an open rotation minted versions: %+v", result)
	}
	if h.bindingFor("password").Version.Number != 1 {
		t.Fatal("deploy during an open rotation changed the active version")
	}
}

// [REQ:STC-P0-032] An encryption/recovery key retires its predecessor only
// after a rewrap and restore proved the new version; the binding carries the
// key reference the backup domain resolves.
func TestEncryptionRecoveryKeyRequiresRewrapProof(t *testing.T) {
	h := newHarness(t)
	plans := append(fixturePlans(), domain.BundleSecretPlan{ID: "backup-key", Class: domain.SecretClassPerInstallGenerated, Required: true, Target: domain.BundleSecretTarget{Type: "env", Name: "BACKUP_RECOVERY_KEY"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/backup", Field: "recovery-key"}})
	consumers := fixtureConsumers()
	consumers["fixture/backup:recovery-key"] = []string{"scenario:app"}
	bindings, err := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: plans, Consumers: consumers, Now: h.clock})
	if err != nil {
		t.Fatal(err)
	}
	var key domain.CredentialBinding
	for _, b := range bindings {
		if b.Descriptor.Field == "recovery-key" {
			key = b
		}
	}
	if key.Class != domain.CredentialClassEncryptionRecoveryKey || key.RecoveryKeyRef != "fixture/backup:recovery-key" {
		t.Fatalf("recovery key binding = %+v", key)
	}
	if _, err := h.svc.Materialize(context.Background(), MaterializeRequest{Target: h.targetRef(), Bindings: bindings}); err != nil {
		t.Fatal(err)
	}
	var seen []domain.CredentialVersionRef
	h.svc.Providers[domain.CredentialClassEncryptionRecoveryKey] = &EncryptionRecoveryKeyProvider{RewrapAndRestore: func(_ context.Context, ref domain.CredentialVersionRef) error {
		seen = append(seen, ref)
		if ref.Version == 2 {
			return errors.New("restore under the new key failed")
		}
		return nil
	}}
	failed, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: key.ID})
	if !apierrors.Is(err, CodeDistributionFailed) {
		t.Fatalf("rotate with failing rewrap = %v", err)
	}
	h.mustState(failed, domain.RotationFailed)
	stored, _ := h.store.GetBinding(context.Background(), "dep-1", key.ID)
	if stored.Version.Number != 1 {
		t.Fatal("predecessor key retired without a rewrap proof")
	}
	if len(seen) != 1 || seen[0].Version != 2 {
		t.Fatalf("rewrap calls = %v", seen)
	}
	done, err := h.svc.Rotate(context.Background(), RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: key.ID})
	if err != nil {
		t.Fatalf("second rotate: %v", err)
	}
	h.mustState(done, domain.RotationComplete)
	stored, _ = h.store.GetBinding(context.Background(), "dep-1", key.ID)
	if stored.Version.Number != 3 {
		t.Fatalf("active key version = %d", stored.Version.Number)
	}
}

// [REQ:STC-P0-032] SECRET-07: a locked or unreachable target store fails
// closed before any value is sent; nothing is written and no plaintext
// fallback exists.
func TestLockedStoreFailsClosedBeforeAnyWrite(t *testing.T) {
	h := newHarness(t)
	h.target.locked = true
	_, err := h.svc.Materialize(context.Background(), MaterializeRequest{Target: h.targetRef(), Bindings: h.plan()})
	if !apierrors.Is(err, CodeStoreLocked) {
		t.Fatalf("materialize onto locked store = %v", err)
	}
	if typed := apierrors.As(err); typed.NextAction == nil || typed.HTTPStatus != 503 {
		t.Fatalf("locked store error lacks next action or status: %+v", typed)
	}
	if len(h.target.stdin) != 0 {
		t.Fatal("a value was sent to a locked store")
	}
	bindings, _ := h.store.ListBindings(context.Background(), "dep-1")
	for _, b := range bindings {
		if b.Version.Number != 0 {
			t.Fatalf("binding %s versioned without a write: %+v", b.ID, b)
		}
	}
	h.target.locked = false
	h.target.unreachable = true
	if _, err := h.svc.Materialize(context.Background(), MaterializeRequest{Target: h.targetRef(), Bindings: h.plan()}); !IsUnreachable(err) {
		t.Fatalf("materialize onto unreachable target = %v", err)
	}
}
