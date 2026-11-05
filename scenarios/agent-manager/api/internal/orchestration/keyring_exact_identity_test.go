package orchestration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modules"
	"agent-manager/internal/supervision"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	coreidentity "github.com/vrooli/api-core/identity"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

// Actual native lifecycle through a synthetic in-process runner: no model,
// subprocess, services or fabricated run status/admission receipt. Keyring grants
// exact metadata authority only; the existing verified native owner independently
// supplies run authority. One Keyring approval covers parent and admitted child.
func TestKeyringExactOneApprovalActualParentChildLifecycle(t *testing.T) {
	ctx := context.Background()
	o := newDeclarationOrchestrator(t)
	newCurrentModelPolicyFixtureOption(t)(o)
	o.identitySecret = []byte("disposable-keyring-lifecycle-secret")
	o.runStateRoot = t.TempDir()
	root := t.TempDir()
	o.config.DefaultProjectRoot = root
	o.config.RequireSandboxByDefault = false
	defer o.dispatcher.Close()
	type launched struct {
		request runner.ExecuteRequest
		token   string
	}
	starts := make(chan launched, 8)
	release := make(chan struct{})
	var released sync.Once
	finish := func() { released.Do(func() { close(release) }) }
	defer finish()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		token := req.Environment["VROOLI_AGENT_IDENTITY_TOKEN"]
		select {
		case starts <- launched{req, token}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "synthetic only"}}, nil
	}
	o.runners = runner.NewRegistry()
	if err := o.runners.Register(&launchInfoStub{MockRunner: mock, args: []string{"--synthetic-in-process"}, runtimeVersion: "synthetic-in-process/1"}); err != nil {
		t.Fatal(err)
	}
	expiry := time.Now().Add(time.Hour)
	owner := coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "fixture-owner", Scopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"}, ExpiresAt: expiry}
	ownerCtx := coreidentity.WithPrincipal(ctx, owner)
	profile, err := o.CreateProfile(ctx, &domain.AgentProfile{Name: "synthetic parent child", ProfileKey: "keyring-inprocess", RoleRef: "code.default", DeclaredScopes: owner.Scopes, MaxTurns: 5, Timeout: time.Minute, NetworkAccess: domain.NetworkAccessNone, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}})
	if err != nil {
		t.Fatal(err)
	}
	newTask := func(title string) *domain.Task {
		t.Helper()
		created, err := o.CreateTask(ctx, &domain.Task{Title: title, Description: "synthetic in-process only", ScopePath: root, ProjectRoot: root})
		if err != nil {
			t.Fatal(err)
		}
		return created
	}
	parentTask := newTask("isolated native parent")
	childTask := newTask("isolated native child")
	wait := func() launched {
		select {
		case got := <-starts:
			return got
		case <-time.After(10 * time.Second):
			for _, key := range []string{"keyring-real-parent", "keyring-real-child"} {
				if saved, err := o.runs.GetByIdempotencyKey(ctx, key); err == nil && saved != nil {
					t.Logf("synthetic run %s status=%s failure=%s", key, saved.Status, saved.ErrorMsg)
				}
			}
			t.Fatal("synthetic native lifecycle did not launch")
			return launched{}
		}
	}
	parentEffort := &eventpb.WorkReference{
		Kind: "effort", Id: "effort:actual-parent", Revision: "goal-rev-7", Relationship: "orchestrator",
		SourceEventId: "source-event-7", EvidenceDigest: "sha256:fixture-evidence",
		Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
		State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
	}
	parentReferenceControls := []*eventpb.WorkReference{
		{Kind: "effort", Id: "effort:private", Relationship: "orchestrator", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PRIVATE, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE},
		{Kind: "effort", Id: "effort:unverified", Relationship: "orchestrator", Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE},
		{Kind: "effort", Id: "effort:expired", Relationship: "orchestrator", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_EXPIRED},
		{Kind: "effort", Id: "effort:unavailable", Relationship: "orchestrator", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_UNAVAILABLE},
		{Kind: "effort", Id: "effort:worker", Relationship: "worker", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE},
	}
	parent, err := o.CreateRun(ownerCtx, CreateRunRequest{
		TaskID: parentTask.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", OwnerToken: "disposable-native-owner-fixture",
		IdempotencyKey: "keyring-real-parent", WorkReferences: append([]*eventpb.WorkReference{parentEffort}, parentReferenceControls...),
	})
	if err != nil {
		t.Fatal("parent admission", err)
	}
	if len(parent.WorkReferences) != 6 || parent.WorkReferences[0].GetRelationship() != "orchestrator" ||
		parent.WorkReferences[0].GetRevision() != parentEffort.Revision || !parent.WorkReferences[0].GetVerified() {
		t.Fatalf("root orchestrator attribution was changed during admission: %+v", parent.WorkReferences)
	}
	parentLaunch := wait()
	parentVerified, err := o.VerifyIdentityToken(ctx, parentLaunch.token)
	if err != nil || !parentVerified.Valid || parentVerified.Claims.RunID != parent.ID || parentVerified.RunStatus != domain.RunStatusRunning {
		t.Fatal("actual parent custody/lifecycle", err)
	}
	persistedParent, err := o.runs.Get(ctx, parent.ID)
	if err != nil || persistedParent == nil || persistedParent.OwnerSubject != owner.Subject || persistedParent.ResolvedConfig == nil || persistedParent.ResolvedConfig.Admission == nil || persistedParent.ResolvedConfig.Admission.CreateCaller == nil {
		t.Fatalf("parent admission caller was not durably retained: run=%+v err=%v", persistedParent, err)
	}
	parentCaller := persistedParent.ResolvedConfig.Admission.CreateCaller
	if parentCaller.Kind != "human" || parentCaller.Subject != owner.Subject || parentCaller.RunID != uuid.Nil {
		t.Fatalf("parent admission caller attribution mismatch: %+v", parentCaller)
	}
	child, err := o.CreateRun(ctx, CreateRunRequest{
		TaskID: childTask.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", ParentRunID: &parent.ID,
		RunIdentityToken: parentLaunch.token, RequestedScopes: []string{"agent-manager:read"}, IdempotencyKey: "keyring-real-child",
		WorkReferences: []*eventpb.WorkReference{
			{Kind: "effort", Id: parentEffort.Id, Relationship: "worker"},
			{Kind: "issue", Id: "ISSUE-7", Relationship: "tracks"},
		},
	})
	if err != nil {
		t.Fatal("genuine child admission", err)
	}
	persistedChild, err := o.runs.Get(ctx, child.ID)
	if err != nil || persistedChild == nil || persistedChild.ParentRunID == nil || *persistedChild.ParentRunID != parent.ID || persistedChild.OwnerSubject != owner.Subject || persistedChild.ResolvedConfig == nil || persistedChild.ResolvedConfig.Admission == nil || persistedChild.ResolvedConfig.Admission.CreateCaller == nil {
		t.Fatalf("exact parent attribution was not durably retained: run=%+v err=%v", persistedChild, err)
	}
	childCaller := persistedChild.ResolvedConfig.Admission.CreateCaller
	if childCaller.Kind != "run" || childCaller.RunID != parent.ID {
		t.Fatalf("child caller is not bound to the exact parent: %+v", childCaller)
	}
	var inherited *eventpb.WorkReference
	var retainedIssue *eventpb.WorkReference
	inheritedEffortCount := 0
	for _, ref := range child.WorkReferences {
		if ref.GetKind() == "effort" {
			inheritedEffortCount++
		}
		if ref.GetKind() == "effort" && ref.GetId() == parentEffort.Id {
			if inherited != nil {
				t.Fatal("inherited effort reference was duplicated")
			}
			inherited = ref
		}
		if ref.GetKind() == "issue" && ref.GetId() == "ISSUE-7" {
			retainedIssue = ref
		}
	}
	if inheritedEffortCount != 1 || inherited == nil || inherited.GetRelationship() != "worker" || !inherited.GetVerified() ||
		inherited.GetVisibility() != eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC ||
		inherited.GetState() != eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE ||
		inherited.GetRevision() != parentEffort.Revision || inherited.GetSourceEventId() != parentEffort.SourceEventId ||
		inherited.GetEvidenceDigest() != parentEffort.EvidenceDigest {
		t.Fatalf("child did not inherit the parent's canonical worker reference: %+v", child.WorkReferences)
	}
	if retainedIssue == nil || retainedIssue.GetRelationship() != "tracks" {
		t.Fatalf("explicit generic child reference was not preserved: %+v", child.WorkReferences)
	}
	childLaunch := wait()
	childVerified, err := o.VerifyIdentityToken(ctx, childLaunch.token)
	if err != nil || !childVerified.Valid || childVerified.Claims.RunID != child.ID || child.ParentRunID == nil || *child.ParentRunID != parent.ID || childVerified.RunStatus != domain.RunStatusRunning {
		t.Fatal("actual child custody/lifecycle", err)
	}

	// Validate the merged persisted contract, not only the caller's input bound:
	// one inherited parent effort plus 100 explicit declarations exceeds the
	// existing maximum and must fail before a run or idempotency record is kept.
	overflowTask := newTask("overflow child")
	tooManyRefs := make([]*eventpb.WorkReference, maxWorkReferenceDeclarations)
	for i := range tooManyRefs {
		tooManyRefs[i] = &eventpb.WorkReference{Kind: "issue", Id: fmt.Sprintf("ISSUE-OVERFLOW-%d", i), Relationship: "tracks"}
	}
	overflowKey := "keyring-merged-reference-overflow"
	if _, err := o.CreateRun(ctx, CreateRunRequest{
		TaskID: overflowTask.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", ParentRunID: &parent.ID,
		RunIdentityToken: parentLaunch.token, RequestedScopes: []string{"agent-manager:read"}, IdempotencyKey: overflowKey,
		WorkReferences: tooManyRefs,
	}); err == nil || !strings.Contains(err.Error(), "at most 100 declarations") {
		t.Fatalf("merged work-reference limit was not enforced: err=%v", err)
	}
	if saved, getErr := o.runs.GetByIdempotencyKey(ctx, overflowKey); getErr != nil || saved != nil {
		t.Fatalf("overflow child was persisted: run=%+v err=%v", saved, getErr)
	}

	delegated, err := o.MintDelegatedIdentity(ctx, MintDelegatedIdentityRequest{ParentToken: parentLaunch.token, ChildRunID: child.ID, RequestedScopes: []string{"agent-manager:read"}})
	if err != nil {
		t.Fatal("genuine admitted child attenuation", err)
	}
	childVerified, err = o.VerifyIdentityToken(ctx, delegated.Token)
	if err != nil || !childVerified.Valid || len(childVerified.Claims.Scopes) != 1 || childVerified.Claims.Scopes[0] != "agent-manager:read" || delegated.Token == parentLaunch.token || delegated.ExpiresAt.After(expiry) {
		t.Fatal("distinct narrower child authority", err)
	}
	if old, err := o.VerifyIdentityToken(ctx, childLaunch.token); err != nil || old.Valid {
		t.Fatal("old child generation retained", err)
	}

	// Redundant same-ID declarations may omit canonical fields, but cannot
	// contradict the parent-derived worker role or retained source terms.
	for _, ref := range []*eventpb.WorkReference{
		{Kind: "effort", Id: parentEffort.Id, Relationship: "orchestrator"},
		{Kind: "effort", Id: parentEffort.Id, Relationship: "worker", Revision: "stale-revision"},
	} {
		key := "keyring-conflicting-child-" + uuid.NewString()
		conflictTask := newTask("conflicting child")
		_, err := o.CreateRun(ctx, CreateRunRequest{
			TaskID: conflictTask.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", ParentRunID: &parent.ID,
			RunIdentityToken: parentLaunch.token, RequestedScopes: []string{"agent-manager:read"}, IdempotencyKey: key,
			WorkReferences: []*eventpb.WorkReference{ref},
		})
		if err == nil || !strings.Contains(err.Error(), "parent's worker attribution") {
			t.Fatalf("contradictory child effort reference accepted: ref=%+v err=%v", ref, err)
		}
		if saved, getErr := o.runs.GetByIdempotencyKey(ctx, key); getErr != nil || saved != nil {
			t.Fatalf("conflicting child was persisted: run=%+v err=%v", saved, getErr)
		}
	}

	// Parentless effort attribution is not invented for a parent with no
	// eligible effort declaration; explicit generic references remain intact.
	noEffortParentTask := newTask("no-effort parent")
	noEffortParent, err := o.CreateRun(ownerCtx, CreateRunRequest{
		TaskID: noEffortParentTask.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", OwnerToken: "disposable-native-owner-fixture",
		IdempotencyKey: "keyring-no-effort-parent",
	})
	if err != nil {
		t.Fatal("no-effort parent admission", err)
	}
	noEffortParentLaunch := wait()
	noEffortChildTask := newTask("no-effort child")
	noEffortChild, err := o.CreateRun(ctx, CreateRunRequest{
		TaskID: noEffortChildTask.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", ParentRunID: &noEffortParent.ID,
		RunIdentityToken: noEffortParentLaunch.token, RequestedScopes: []string{"agent-manager:read"},
		IdempotencyKey: "keyring-no-effort-child", WorkReferences: []*eventpb.WorkReference{{Kind: "issue", Id: "ISSUE-8", Relationship: "tracks"}},
	})
	if err != nil {
		t.Fatal("no-effort child admission", err)
	}
	wait()
	if len(noEffortChild.WorkReferences) != 1 || noEffortChild.WorkReferences[0].GetKind() != "issue" || noEffortChild.WorkReferences[0].GetId() != "ISSUE-8" {
		t.Fatalf("child without parent effort gained or lost references: %+v", noEffortChild.WorkReferences)
	}

	foreignParentID := uuid.New()
	foreignKey := "keyring-foreign-parent-" + uuid.NewString()
	foreignTask := newTask("foreign parent child")
	if _, err = o.CreateRun(ctx, CreateRunRequest{
		TaskID: foreignTask.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", ParentRunID: &foreignParentID,
		RunIdentityToken: parentLaunch.token, RequestedScopes: []string{"agent-manager:read"}, IdempotencyKey: foreignKey,
	}); err == nil {
		t.Fatal("child request with a foreign parent identity was accepted")
	}
	if saved, getErr := o.runs.GetByIdempotencyKey(ctx, foreignKey); getErr != nil || saved != nil {
		t.Fatalf("foreign-parent child was persisted: run=%+v err=%v", saved, getErr)
	}
	db, err := sqlx.Connect("sqlite", filepath.Join(t.TempDir(), "keyring-exact.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = coredb.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		t.Fatal(err)
	}
	repo := supervision.NewRepository(db)
	service := supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{})
	v := service.InstalledExactVerifier()
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	p := supervision.ExactPolicy{ID: "one-standing-grant", Owner: owner.Subject, Instance: "synthetic-native", Issuer: "synthetic-keyring", KeyVersion: 1, Epoch: 1, Deadline: expiry.Unix(), EffortRef: "effort:synthetic", AuthorizedBy: owner.Subject, AuthorityRef: "one-keyring-approval", Profile: supervision.ExactProfile, PublicKey: public, Operations: []string{"effort.reconcile", "receipt.read"}, ReadEligibility: supervision.ExactStandingReadEligibility}
	seed := &pb.EffortEnrollment{EffortRef: p.EffortRef, DisplayName: "before", Revision: 1, AuthorizedBy: p.AuthorizedBy, AuthorityRef: p.AuthorityRef}
	if err = repo.SaveEffort(ctx, seed, &pb.EffortBoardRow{}, 0, "seed", "seed"); err != nil {
		t.Fatal(err)
	}
	staged, err := v.Setup(ownerCtx, supervision.ExactSetupRequest{Action: "stage", Policy: &p})
	if err != nil {
		t.Fatal(err)
	}
	activation, _ := json.Marshal(staged.ActivationChallenge)
	if _, err = v.Setup(ownerCtx, supervision.ExactSetupRequest{Action: "activate", Grant: p.ID, ExpectedEpoch: 1, PolicyDigest: staged.PolicyDigest, KeyProof: base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, activation))}); err != nil {
		t.Fatal(err)
	}
	nativeCtx := func(token string) context.Context {
		verified, err := o.VerifyIdentityToken(ctx, token)
		if err != nil || !verified.Valid {
			t.Fatal("real native leaf refused", err)
		}
		return coreidentity.WithPrincipal(ctx, coreidentity.Principal{Kind: coreidentity.ActorAgent, Verified: true, Subject: verified.Claims.RunID.String(), Owner: verified.Claims.Subject, Scopes: verified.Claims.Scopes, ExpiresAt: time.Unix(verified.Claims.ExpiresAt, 0)})
	}
	req := &pb.ReconcileEffortMetadataRequest{Enrollment: proto.Clone(seed).(*pb.EffortEnrollment), ExpectedRevision: 1, IdempotencyKey: "parent-original-action", Authority: pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}
	req.Enrollment.DisplayName = "after"
	action, _ := protojson.Marshal(req)
	parentCtx := nativeCtx(parentLaunch.token)
	challenge, err := v.Challenge(parentCtx, p.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	sign := func(c *supervision.ExactChallenge, action []byte) (string, string) {
		e := supervision.ExactEnvelope{Version: 1, Purpose: c.Purpose, Instance: c.Instance, Audience: c.Audience, Issuer: p.Issuer, KeyVersion: 1, Action: action, Grant: p.ID, Epoch: 1, Initiator: p.Owner, Executor: c.Executor, Operation: c.Operation, Target: c.Target, Payload: c.Payload, Request: c.Request, Expires: time.Now().Add(30 * time.Second).Unix()}
		raw, _ := json.Marshal(e)
		return base64.RawURLEncoding.EncodeToString(raw), base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, raw))
	}
	raw, sig := sign(challenge, action)
	sealed, err := v.Verify(parentCtx, req, raw, sig)
	if err != nil {
		t.Fatal(err)
	}
	actor, ok, err := supervision.ExactEffortActor(sealed, p.EffortRef)
	if !ok || err != nil {
		t.Fatal(err)
	}
	if _, err = service.ReconcileMetadata(sealed, req, actor); err != nil {
		t.Fatal(err)
	}
	read := supervision.ExactReceiptRequest{OperationKey: "reconcile-metadata:" + req.IdempotencyKey, EffortRef: p.EffortRef, OriginalExecutor: parent.ID.String(), OriginalPayload: challenge.Payload}
	childCtx := nativeCtx(delegated.Token)
	readChallenge, err := v.ReceiptChallenge(childCtx, p.ID, read)
	if err != nil {
		t.Fatal("same approval child eligibility", err)
	}
	readAction, _ := json.Marshal(read)
	raw, sig = sign(readChallenge, readAction)
	sealed, err = v.VerifyReceipt(childCtx, read, raw, sig)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := service.ReadExactReceiptProof(sealed, read)
	if err != nil || receipt.OriginalExecutor != parent.ID.String() || receipt.OriginalPayload != challenge.Payload {
		t.Fatal("child changed original attribution", err)
	}
	var count int
	if db.Get(&count, `SELECT COUNT(*) FROM keyring_exact_policy_setup`) != nil || count != 1 {
		t.Fatal("per-actor approval", count)
	}
	if err = repo.RevokeExactAuthority(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ReadExactReceiptProof(sealed, read); err == nil {
		t.Fatal("revoked original authority allowed child read")
	}
	finish()
	deadline := time.Now().Add(10 * time.Second)
	for {
		parentNow, parentErr := o.runs.Get(ctx, parent.ID)
		childNow, childErr := o.runs.Get(ctx, child.ID)
		if parentErr != nil || childErr != nil {
			t.Fatal("synthetic terminal cleanup read")
		}
		if parentNow.Status.IsTerminal() && childNow.Status.IsTerminal() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("synthetic runner did not settle")
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, token := range []string{parentLaunch.token, delegated.Token} {
		if verified, err := o.VerifyIdentityToken(ctx, token); err != nil || verified.Valid {
			t.Fatal("terminal fixture identity not revoked", err)
		}
	}

}
