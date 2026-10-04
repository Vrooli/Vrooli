package orchestration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	_ "modernc.org/sqlite"
)

type effortOwnersFixture struct {
	enabled bool
	scopes  []string
}

func (f *effortOwnersFixture) CurrentEffortOwner(context.Context, string) (effortauthority.OwnerState, error) {
	return effortauthority.OwnerState{Subject: "finite-owner", Enabled: f.enabled, Scopes: f.scopes}, nil
}
func (f *effortOwnersFixture) ApproveEffortOwner(c context.Context, s string) (effortauthority.OwnerState, error) {
	v, e := f.CurrentEffortOwner(c, s)
	v.Ceiling = f.scopes
	return v, e
}

type effortScopeFixture struct{}

// fixtureUnitTerminal is synthetic owner evidence; no real unit is launched.
type fixtureUnitTerminal struct{}

func (fixtureUnitTerminal) Terminal(context.Context, string) error { return nil }

func (effortScopeFixture) CheckEffortScope(context.Context, effortauthority.Policy) error { return nil }
func nativeEffortFixture(t *testing.T) (*Orchestrator, *effortauthority.Authority, effortauthority.Policy, *domain.Task, *domain.AgentProfile, ed25519.PrivateKey, *time.Time) {
	t.Helper()
	ctx := context.Background()
	o := newDeclarationOrchestrator(t)
	newCurrentModelPolicyFixtureOption(t)(o)
	o.dispatcher.Close()
	repo := t.TempDir()
	o.config.DefaultProjectRoot = repo
	o.runStateRoot = t.TempDir()
	o.identitySecret = []byte("disposable-finite-native-secret")
	now := time.Now().UTC().Truncate(time.Second)
	o.clock = func() time.Time { return now }
	profile, e := o.CreateProfile(ctx, &domain.AgentProfile{Name: "Bounded fixture", ProfileKey: "finite-fixture", RoleRef: "code.default", MaxTurns: 5, Timeout: 30 * time.Minute, NetworkAccess: domain.NetworkAccessNone, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, DeclaredScopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"}})
	if e != nil {
		t.Fatal(e)
	}
	// Re-read the complete persisted owner profile, including database timestamps.
	profile, e = o.profiles.GetByKey(ctx, profile.ProfileKey)
	if e != nil {
		t.Fatal(e)
	}
	task, e := o.CreateTask(ctx, &domain.Task{Title: "Finite fixture", Description: "Approved finite work", ScopePath: repo, ProjectRoot: repo})
	if e != nil {
		t.Fatal(e)
	}
	db, e := sql.Open("sqlite", filepath.Join(t.TempDir(), "finite.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	store := &effortauthority.SQLStore{DB: db}
	if e = store.Ensure(ctx); e != nil {
		t.Fatal(e)
	}
	public, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	owners := &effortOwnersFixture{true, []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"}}
	a := &effortauthority.Authority{Store: store, Owners: owners, Approver: owners, Scope: effortScopeFixture{}, Now: func() time.Time { return now }}
	p := effortauthority.Policy{ID: "finite-fixture-policy", Owner: "finite-owner", Client: "fixture-client", ClientKey: public, Epoch: 1, Repository: repo, Effort: "finite-fixture-effort", Revision: "owner-approved-1", ContentDigest: effortauthority.Digest("approved"), Team: "fixture-team", TeamDigest: effortauthority.Digest("team"), BindingDigest: effortauthority.Digest("binding"), Members: []string{"lead"}, Profiles: map[string]string{profile.ProfileKey: EffortProfileDigest(profile)}, Scopes: owners.scopes, Effects: []string{"run.create", "run.child", "run.recover"}, NotBefore: now, Deadline: now.Add(8 * time.Hour), MaxStarts: 6, MaxConcurrent: 6, MaxTurns: 5, MaxToolCalls: 10, MaxRunSeconds: 1800, TotalTurns: 30, TotalToolCalls: 60, TotalRunSeconds: 10800, AllowChildren: true, AllowRecovery: true, SurviveLogout: true}
	if e = a.Approve(ctx, "fixture-human", p); e != nil {
		t.Fatal(e)
	}
	o.effortAuthority = a
	o.finiteNativeTerminal = fixtureUnitTerminal{}
	installFixtureNativeFactory(t, o, p, profile)
	return o, a, p, task, profile, key, &now
}
func installFixtureNativeFactory(t *testing.T, o *Orchestrator, p effortauthority.Policy, profile *domain.AgentProfile) {
	t.Helper()
	// Pure manifest composition qualifies admission only. Dispatcher remains closed;
	// this fixture never qualifies a host, starts a service, or launches an agent.
	h := effortauthority.Digest("fixture executable")
	factory, e := runner.NewFiniteNativeFactory(isolation.Manifest{Enabled: true, Backend: "linux-systemd-service-v1", SystemdRun: isolation.Executable{Path: "/usr/bin/systemd-run", SHA256: h}, Systemctl: isolation.Executable{Path: "/usr/bin/systemctl", SHA256: h}, RootFS: "/var/lib/finite-fixture/image", WorkspaceSource: "/var/lib/finite-fixture/worktree", RootFSDigest: h, NativeUID: 1234, NativeGID: 1234, Isolation: &effortauthority.NativeUIDIsolation{NativeUID: 1234, ProtectedPaths: []string{"/var/lib/finite-authority/key"}}, Bindings: map[string]isolation.Binding{p.ID: {PolicyDigest: effortauthority.Digest(p), ProfileDigest: p.Profiles[profile.ProfileKey], Repository: p.Repository, Deadline: p.Deadline}}, EnvKeys: []string{"PATH", "HOME", "VROOLI_AGENT_IDENTITY_TOKEN"}, NativeBinaries: map[string]string{"/usr/bin/codex": h}})
	if e != nil {
		t.Fatal(e)
	}
	o.finiteNativeFactory = factory

}
func effortRequest(t *testing.T, o *Orchestrator, p effortauthority.Policy, task *domain.Task, profile *domain.AgentProfile, key ed25519.PrivateKey, id string) CreateRunRequest {
	t.Helper()
	req := CreateRunRequest{TaskID: task.ID, ProfileRef: &ProfileRef{ProfileKey: profile.ProfileKey}, IdempotencyKey: id, Tag: id}
	intent := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: profile.ProfileKey, ProfileDigest: p.Profiles[profile.ProfileKey], Effect: "run.create", IdempotencyKey: id, InputDigest: effortauthority.Digest(effortNativeInput(req, task, profile.ProfileKey)), Turns: p.MaxTurns, ToolCalls: p.MaxToolCalls, RunSeconds: p.MaxRunSeconds}
	proof, e := effortauthority.Sign(effortauthority.Proof{Intent: intent, Nonce: "native-fixture-nonce-" + id, IssuedAt: o.now()}, key)
	if e != nil {
		t.Fatal(e)
	}
	b, e := json.Marshal(proof)
	if e != nil {
		t.Fatal(e)
	}
	req.EffortProof = base64.RawURLEncoding.EncodeToString(b)
	return req
}
func TestFinitePublicNativeAdmissionAfterHumanExpiryRetainsEffortDeadline(t *testing.T) {
	o, a, p, task, profile, key, now := nativeEffortFixture(t)
	*now = now.Add(5 * time.Hour)
	req := effortRequest(t, o, p, task, profile, key, "overnight")
	check := req
	if e := o.authenticateCreateRunCaller(context.Background(), &check); e != nil {
		t.Fatalf("initial caller proof failed: %v", e)
	}
	t.Logf("task scope=%q root=%q approved=%q", task.ScopePath, task.ProjectRoot, p.Repository)
	_, e := o.CreateRun(context.Background(), req)
	if !errors.Is(e, spawn.ErrDispatcherClosed) {
		t.Fatalf("expected isolated dispatcher boundary after admission, got %v", e)
	}
	run, e := o.runs.GetByIdempotencyKey(context.Background(), req.IdempotencyKey)
	if e != nil || run == nil {
		t.Fatal("missing native acceptance", e)
	}
	if run.OwnerSubject != p.Owner || run.OwnerExpiresAt == nil || !run.OwnerExpiresAt.Equal(p.Deadline) || run.ResolvedConfig.Admission.Effort == nil || run.ResolvedConfig.MaxToolCalls != p.MaxToolCalls {
		t.Fatal("native contract lost bounded authority")
	}
	record, e := a.Store.Get(context.Background(), p.ID)
	if e != nil || record.Reservations[req.IdempotencyKey].RunID != run.ID.String() {
		t.Fatal("missing exact native receipt", e)
	}
	replay, e := o.CreateRun(context.Background(), req)
	if e != nil || replay.ID != run.ID {
		t.Fatal("same-key replay did not read accepted run", e)
	}
	token := phases.GenerateIdentityToken(context.Background(), phases.GenerateIdentityTokenInput{Run: run, RequestedScopes: run.RequestedScopes, Secret: o.identitySecret, Deps: phases.Deps{Runs: o.runs, Clock: o.clock}})
	claims, e := identity.VerifyToken(token, o.identitySecret)
	if e != nil || claims.ExpiresAt != p.Deadline.Unix() {
		t.Fatal("native proof inherited login expiry or widened effort deadline", e)
	}
}
func TestFinitePublicOverridesAndInvalidProofRefuseBeforeNativeEffects(t *testing.T) {
	for _, name := range []string{"absent", "bad-proof", "human-mix", "run-mix", "in-place", "force", "prompt", "profile-update", "expired", "revoked", "changed-profile"} {
		t.Run(name, func(t *testing.T) {
			o, a, p, task, profile, key, now := nativeEffortFixture(t)
			req := effortRequest(t, o, p, task, profile, key, "negative")
			switch name {
			case "absent":
				req.EffortProof = ""
			case "bad-proof":
				req.EffortProof = "invalid"
			case "human-mix":
				req.OwnerToken = "offered-human"
			case "run-mix":
				req.RunIdentityToken = "offered-run"
			case "in-place":
				mode := domain.RunModeInPlace
				req.RunMode = &mode
			case "force":
				req.Force = true
			case "prompt":
				req.Prompt = "changed unbound instructions"
			case "profile-update":
				req.ProfileRef.UpdateExisting = true
				req.ProfileRef.Defaults = &domain.AgentProfile{Name: "changed"}
			case "expired":
				*now = p.Deadline
			case "revoked":
				if e := a.Revoke(context.Background(), "fixture", p.ID); e != nil {
					t.Fatal(e)
				}
			case "changed-profile":
				profile.Description = "changed policy"
				if e := o.profiles.Update(context.Background(), profile); e != nil {
					t.Fatal(e)
				}
			}
			before, _ := a.Store.Get(context.Background(), p.ID)
			if run, e := o.CreateRun(context.Background(), req); e == nil || run != nil {
				t.Fatal("invalid admission accepted")
			}
			after, _ := a.Store.Get(context.Background(), p.ID)
			if effortauthority.Digest(before) != effortauthority.Digest(after) {
				t.Fatal("rejection mutated finite admission ledger")
			}
			if run, e := o.runs.GetByIdempotencyKey(context.Background(), req.IdempotencyKey); e != nil || run != nil {
				t.Fatal("rejection persisted native run", e)
			}
		})
	}
}

func TestFiniteNativeExactParentLateChildAndRevocation(t *testing.T) {
	o, a, p, task, profile, key, now := nativeEffortFixture(t)
	root := effortRequest(t, o, p, task, profile, key, "original-parent")
	_, e := o.CreateRun(context.Background(), root)
	if !errors.Is(e, spawn.ErrDispatcherClosed) {
		t.Fatal(e)
	}
	parent, e := o.runs.GetByIdempotencyKey(context.Background(), root.IdempotencyKey)
	if e != nil {
		t.Fatal(e)
	}
	parent.Status = domain.RunStatusRunning
	parent.ResolvedConfig.Admission.RuntimeVersion = "disposable-runner/fixture"
	parent.ResolvedConfig.Admission.PassedControlArgs = []string{"--fixture"}
	token := phases.GenerateIdentityToken(context.Background(), phases.GenerateIdentityTokenInput{Run: parent, RequestedScopes: parent.RequestedScopes, Secret: o.identitySecret, Deps: phases.Deps{Runs: o.runs, Clock: o.clock}})
	parent.IdentityTokenHash = identity.HashToken(token)
	if e = o.runs.Update(context.Background(), parent); e != nil {
		t.Fatal(e)
	}
	*now = now.Add(5 * time.Hour)
	child := CreateRunRequest{TaskID: task.ID, ProfileRef: &ProfileRef{ProfileKey: profile.ProfileKey}, ParentRunID: &parent.ID, RunIdentityToken: token, IdempotencyKey: "late-child", Tag: "late-child"}
	beforeChild, _ := a.Store.Get(context.Background(), p.ID)
	_, e = o.CreateRun(context.Background(), child)
	if !errors.Is(e, effortauthority.ErrRefused) {
		t.Fatalf("unqualified finite child route accepted: %v", e)
	}
	accepted, readErr := o.runs.GetByIdempotencyKey(context.Background(), child.IdempotencyKey)
	afterChild, _ := a.Store.Get(context.Background(), p.ID)
	if readErr != nil || accepted != nil || effortauthority.Digest(beforeChild) != effortauthority.Digest(afterChild) {
		t.Fatal("unqualified finite child changed run or ledger", readErr)
	}
	if e = a.Revoke(context.Background(), "fixture-human", p.ID); e != nil {
		t.Fatal(e)
	}
	before, _ := a.Store.Get(context.Background(), p.ID)
	child.IdempotencyKey = "revoked-child"
	if _, e = o.CreateRun(context.Background(), child); e == nil {
		t.Fatal("revoked parent authority admitted child")
	}
	after, _ := a.Store.Get(context.Background(), p.ID)
	if effortauthority.Digest(before) != effortauthority.Digest(after) {
		t.Fatal("revoked child changed ledger")
	}
	if r, e := o.runs.GetByIdempotencyKey(context.Background(), child.IdempotencyKey); e != nil || r != nil {
		t.Fatal("revoked child persisted run", e)
	}
}

func TestFiniteNativeContinuationExpiryAndRevocationBeforeEffects(t *testing.T) {
	for _, operation := range []string{"resume", "wake"} {
		for _, refusal := range []string{"expired", "revoked"} {
			t.Run(operation+"/"+refusal, func(t *testing.T) {
				o, a, p, task, profile, key, now := nativeEffortFixture(t)
				req := effortRequest(t, o, p, task, profile, key, "original-continuation")
				_, e := o.CreateRun(context.Background(), req)
				if !errors.Is(e, spawn.ErrDispatcherClosed) {
					t.Fatal(e)
				}
				source, e := o.runs.GetByIdempotencyKey(context.Background(), req.IdempotencyKey)
				if e != nil {
					t.Fatal(e)
				}
				source.Status = domain.RunStatusPending
				if operation == "wake" {
					source.Status = domain.RunStatusParked
				}
				if e = o.runs.Update(context.Background(), source); e != nil {
					t.Fatal(e)
				}
				if refusal == "expired" {
					*now = p.Deadline
				} else {
					if e = a.Revoke(context.Background(), "fixture-human", p.ID); e != nil {
						t.Fatal(e)
					}
				}
				persistedBefore, readErr := o.runs.Get(context.Background(), source.ID)
				if readErr != nil {
					t.Fatal(readErr)
				}
				before := effortauthority.Digest(persistedBefore)
				record, _ := a.Store.Get(context.Background(), p.ID)
				if operation == "resume" {
					_, e = o.ResumeRun(context.Background(), source.ID)
				} else {
					_, e = o.WakeRun(context.Background(), WakeRunInput{RunID: source.ID})
				}
				if e == nil {
					t.Fatal("expired/revoked continuation admitted")
				}
				after, _ := o.runs.Get(context.Background(), source.ID)
				recordAfter, _ := a.Store.Get(context.Background(), p.ID)
				if before != effortauthority.Digest(after) || effortauthority.Digest(record) != effortauthority.Digest(recordAfter) {
					t.Fatal("refused continuation changed source/claim/budget")
				}
			})
		}
	}
}
func TestFiniteNativeUnqualifiedContinuationKeepsOriginalDeadlineAndCapacity(t *testing.T) {
	o, a, p, task, profile, key, now := nativeEffortFixture(t)
	req := effortRequest(t, o, p, task, profile, key, "resumable-finite")
	_, e := o.CreateRun(context.Background(), req)
	if !errors.Is(e, spawn.ErrDispatcherClosed) {
		t.Fatal(e)
	}
	source, e := o.runs.GetByIdempotencyKey(context.Background(), req.IdempotencyKey)
	if e != nil {
		t.Fatal(e)
	}
	source.Status = domain.RunStatusPending
	if e = o.runs.Update(context.Background(), source); e != nil {
		t.Fatal(e)
	}
	*now = p.Deadline.Add(-time.Minute)
	before, _ := a.Store.Get(context.Background(), p.ID)
	_, e = o.ResumeRun(context.Background(), source.ID)
	if e == nil {
		t.Fatal("unqualified finite continuation was accepted")
	}
	after, _ := o.runs.Get(context.Background(), source.ID)
	record, _ := a.Store.Get(context.Background(), p.ID)
	if len(record.Reservations) != len(before.Reservations) || !after.OwnerExpiresAt.Equal(p.Deadline) || effortauthority.Digest(record) != effortauthority.Digest(before) {
		t.Fatal("continuation renewed budget/deadline or exceeded remaining lifetime")
	}
}

func TestFiniteNativeFreshRecoveryRequiresExactNewProofBeforeClaim(t *testing.T) {
	for _, valid := range []bool{false, true} {
		t.Run(fmt.Sprint(valid), func(t *testing.T) {
			o, a, p, task, profile, key, now := nativeEffortFixture(t)
			root := effortRequest(t, o, p, task, profile, key, "failed-source")
			_, e := o.CreateRun(context.Background(), root)
			if !errors.Is(e, spawn.ErrDispatcherClosed) {
				t.Fatal(e)
			}
			source, e := o.runs.GetByIdempotencyKey(context.Background(), root.IdempotencyKey)
			if e != nil {
				t.Fatal(e)
			}
			source.Status = domain.RunStatusFailed
			if e = o.runs.Update(context.Background(), source); e != nil {
				t.Fatal(e)
			}
			*now = now.Add(5 * time.Hour)
			req := ResumeFromFailedRunRequest{RunID: source.ID, CustomContext: "Retain the same finite accepted work"}
			if valid {
				intent := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs/resume-from-failed", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: profile.ProfileKey, ProfileDigest: p.Profiles[profile.ProfileKey], Effect: "run.recover", IdempotencyKey: freshRecoveryKey(source.ID), SourceRunID: source.ID.String(), InputDigest: effortauthority.Digest(req), Turns: p.MaxTurns, ToolCalls: p.MaxToolCalls, RunSeconds: p.MaxRunSeconds}
				proof, e := effortauthority.Sign(effortauthority.Proof{Intent: intent, Nonce: "fresh-exact-recovery-nonce", IssuedAt: o.now()}, key)
				if e != nil {
					t.Fatal(e)
				}
				raw, _ := json.Marshal(proof)
				req.EffortProof = base64.RawURLEncoding.EncodeToString(raw)
			}
			before, _ := o.runs.Get(context.Background(), source.ID)
			ledger, _ := a.Store.Get(context.Background(), p.ID)
			_, e = o.ResumeFromFailedRun(context.Background(), req)
			if !valid {
				if e == nil {
					t.Fatal("recovery without client proof accepted")
				}
				after, _ := o.runs.Get(context.Background(), source.ID)
				record, _ := a.Store.Get(context.Background(), p.ID)
				if effortauthority.Digest(before) != effortauthority.Digest(after) || effortauthority.Digest(ledger) != effortauthority.Digest(record) {
					t.Fatal("unproven recovery changed claim or budget")
				}
				return
			}
			if !errors.Is(e, effortauthority.ErrRefused) {
				t.Fatalf("unqualified finite recovery accepted: %v", e)
			}
			replacement, readErr := o.runs.GetByIdempotencyKey(context.Background(), freshRecoveryKey(source.ID))
			after, _ := o.runs.Get(context.Background(), source.ID)
			record, _ := a.Store.Get(context.Background(), p.ID)
			if readErr != nil || replacement != nil || effortauthority.Digest(before) != effortauthority.Digest(after) || effortauthority.Digest(ledger) != effortauthority.Digest(record) {
				t.Fatal("unqualified finite recovery changed claim, run or budget", readErr)
			}

		})
	}
}

func TestFiniteNativeExactTerminalSettlementEnablesSingleSlotSuccessor(t *testing.T) {
	o, a, p, task, profile, key, _ := nativeEffortFixture(t)
	p.ID = "single-slot-native"
	p.MaxStarts = 2
	p.MaxConcurrent = 1
	p.TotalTurns = int64(p.MaxTurns * 2)
	p.TotalToolCalls = int64(p.MaxToolCalls * 2)
	p.TotalRunSeconds = p.MaxRunSeconds * 2
	if e := a.Approve(context.Background(), "fixture-human", p); e != nil {
		t.Fatal(e)
	}
	installFixtureNativeFactory(t, o, p, profile)
	root := effortRequest(t, o, p, task, profile, key, "single-slot-original")
	_, e := o.CreateRun(context.Background(), root)
	if !errors.Is(e, spawn.ErrDispatcherClosed) {
		t.Fatal(e)
	}
	source, e := o.runs.GetByIdempotencyKey(context.Background(), root.IdempotencyKey)
	if e != nil || source == nil {
		t.Fatal(e)
	}
	b := *source.ResolvedConfig.Admission.Effort
	record, _ := a.Store.Get(context.Background(), p.ID)
	if e = o.SyncEffortTerminal(context.Background(), b, source.IdempotencyKey, uuid.NewString()); e == nil {
		t.Fatal("unrelated terminal identity released capacity")
	}
	unchanged, _ := a.Store.Get(context.Background(), p.ID)
	if effortauthority.Digest(record) != effortauthority.Digest(unchanged) {
		t.Fatal("foreign terminal receipt changed ledger")
	}
	source.Status = domain.RunStatusComplete
	if e = o.runs.Update(context.Background(), source); e != nil {
		t.Fatal(e)
	}
	if e = o.SyncEffortTerminal(context.Background(), b, source.IdempotencyKey, source.ID.String()); e != nil {
		t.Fatal(e)
	}
	record, _ = a.Store.Get(context.Background(), p.ID)
	if !record.Reservations[source.IdempotencyKey].Terminal {
		t.Fatal("exact native terminal failed to settle")
	}
	successor := effortRequest(t, o, p, task, profile, key, "single-slot-successor")
	_, e = o.CreateRun(context.Background(), successor)
	if !errors.Is(e, spawn.ErrDispatcherClosed) {
		t.Fatalf("single-slot successor did not admit after exact settlement: %v", e)
	}
	record, _ = a.Store.Get(context.Background(), p.ID)
	if len(record.Reservations) != 2 {
		t.Fatal("successor renewed or forgot aggregate allowance")
	}
}
