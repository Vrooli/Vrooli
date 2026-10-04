package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
)

// These spies count effects explicitly. Unimplemented embedded dependencies
// panic if refusal accidentally reaches another service or dispatch path.
type auth01Effects struct{ taskWrites, runWrites, reservations, cacheWrites, cacheReads int }
type auth01Runs struct {
	repository.RunRepository
	effects          *auth01Effects
	parent, accepted *domain.Run
	tokenHash        string
}

func (r *auth01Runs) GetByTokenHash(_ context.Context, hash string) (*domain.Run, error) {
	if hash != r.tokenHash {
		return nil, nil
	}
	return r.parent, nil
}
func (r *auth01Runs) Get(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	if r.parent != nil && r.parent.ID == id {
		return r.parent, nil
	}
	if r.accepted != nil && r.accepted.ID == id {
		return r.accepted, nil
	}
	return nil, nil
}
func (r *auth01Runs) GetByIdempotencyKey(context.Context, string) (*domain.Run, error) {
	r.effects.cacheReads++
	return r.accepted, nil
}
func (r *auth01Runs) Create(context.Context, *domain.Run) error {
	r.effects.runWrites++
	return errors.New("unexpected run write")
}
func (r *auth01Runs) Update(context.Context, *domain.Run) error {
	r.effects.runWrites++
	return errors.New("unexpected run write")
}

type auth01Tasks struct {
	repository.TaskRepository
	effects *auth01Effects
}

func (r *auth01Tasks) Create(context.Context, *domain.Task) error {
	r.effects.taskWrites++
	return errors.New("unexpected task write")
}
func (r *auth01Tasks) Update(context.Context, *domain.Task) error {
	r.effects.taskWrites++
	return errors.New("unexpected task write")
}

type auth01Idempotency struct {
	repository.IdempotencyRepository
	effects *auth01Effects
}

func (r *auth01Idempotency) Check(context.Context, string) (*domain.IdempotencyRecord, error) {
	return nil, nil
}
func (r *auth01Idempotency) Reserve(context.Context, string, time.Duration) (*domain.IdempotencyRecord, error) {
	r.effects.reservations++
	return nil, errors.New("unexpected reservation")
}
func (r *auth01Idempotency) Fail(context.Context, string) error { r.effects.cacheWrites++; return nil }
func (r *auth01Idempotency) Complete(context.Context, string, uuid.UUID, string, []byte) error {
	r.effects.cacheWrites++
	return nil
}

func auth01CallerFixture(t *testing.T) (*Orchestrator, *auth01Runs, *auth01Effects, CreateRunRequest, string, coreidentity.Principal) {
	t.Helper()
	now := time.Now().Truncate(time.Second)
	effects := &auth01Effects{}
	parent := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), OwnerSubject: "owner-a", Status: domain.RunStatusRunning}
	secret := []byte("disposable-auth01-fixture-key")
	token, err := identity.GenerateToken(&identity.Claims{RunID: parent.ID, TaskID: parent.TaskID, Subject: parent.OwnerSubject, Scopes: []string{"agent-manager:read", "agent-manager:orchestrate"}, ExpiresAt: now.Add(time.Hour).Unix()}, secret)
	if err != nil {
		t.Fatal(err)
	}
	runs := &auth01Runs{effects: effects, parent: parent, tokenHash: identity.HashToken(token)}
	owner := coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "owner-a", Scopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"}, ExpiresAt: now.Add(30 * time.Minute)}
	o := &Orchestrator{clock: func() time.Time { return now }, runs: runs, tasks: &auth01Tasks{effects: effects}, idempotency: &auth01Idempotency{effects: effects}, identitySecret: secret, ownerIdentity: createOwnerVerifier(func(_ context.Context, offered string) (coreidentity.Principal, error) {
		if offered != "fixture-owner" {
			return coreidentity.Principal{}, errors.New("invalid fixture credential")
		}
		return owner, nil
	})}
	req := CreateRunRequest{TaskID: uuid.New(), OwnerToken: "fixture-owner", IdempotencyKey: "fixture-admission"}
	return o, runs, effects, req, token, owner
}
func assertAuth01NoEffects(t *testing.T, e *auth01Effects) {
	t.Helper()
	if e.taskWrites != 0 || e.runWrites != 0 || e.reservations != 0 || e.cacheWrites != 0 {
		t.Fatalf("refusal effects: %+v", e)
	}
}

func TestAuth01PublicCreateRunRejectsCallerBeforeEffects(t *testing.T) {
	for _, name := range []string{"absent", "body-parent-only", "forged-projection", "blank-run", "wrong-channel", "invalid-run", "expired-run", "revoked-run", "run-verifier-unavailable", "wrong-parent", "retained-task-mismatch", "retained-subject-mismatch", "wrong-purpose", "run-without-parent", "owner-unverified", "owner-service", "owner-expired", "owner-unavailable", "owner-empty-grant", "owner-read-only", "foreign-owner", "scope-widening", "invalid-second-run", "invalid-second-owner"} {
		t.Run(name, func(t *testing.T) {
			o, runs, effects, req, token, owner := auth01CallerFixture(t)
			switch name {
			case "absent":
				req.OwnerToken = ""
			case "body-parent-only":
				req.OwnerToken = ""
				req.ParentRunID = &runs.parent.ID
			case "forged-projection":
				req.OwnerToken = ""
				req.OwnerSubject = "owner-a"
				req.OwnerScopes = owner.Scopes
				req.OwnerExpiresAt = &owner.ExpiresAt
				req.caller = &domain.CreateRunCaller{Kind: "human", Subject: "owner-a"}
			case "blank-run":
				req.RunIdentityToken = " "
			case "wrong-channel":
				req.OwnerToken = token
			case "invalid-run":
				req.OwnerToken = ""
				req.RunIdentityToken = "invalid-fixture-run"
			case "expired-run":
				req.OwnerToken = ""
				claims := &identity.Claims{RunID: runs.parent.ID, ExpiresAt: time.Now().Add(-time.Hour).Unix()}
				var err error
				req.RunIdentityToken, err = identity.GenerateToken(claims, o.identitySecret)
				if err != nil {
					t.Fatal(err)
				}
			case "revoked-run":
				req.OwnerToken = ""
				req.RunIdentityToken = token
				req.ParentRunID = &runs.parent.ID
				now := o.now()
				runs.parent.IdentityTokenRevokedAt = &now
			case "run-verifier-unavailable":
				req.OwnerToken = ""
				req.RunIdentityToken = token
				o.identitySecret = nil
			case "wrong-parent":
				req.OwnerToken = ""
				req.RunIdentityToken = token
				id := uuid.New()
				req.ParentRunID = &id
			case "retained-task-mismatch":
				req.OwnerToken, req.RunIdentityToken, req.ParentRunID = "", token, &runs.parent.ID
				runs.parent.TaskID = uuid.New()
			case "retained-subject-mismatch":
				req.OwnerToken, req.RunIdentityToken, req.ParentRunID = "", token, &runs.parent.ID
				runs.parent.OwnerSubject = "foreign-owner"
			case "wrong-purpose":
				req.OwnerToken, req.ParentRunID = "", &runs.parent.ID
				var err error
				req.RunIdentityToken, err = identity.GenerateToken(&identity.Claims{RunID: runs.parent.ID, TaskID: runs.parent.TaskID, Subject: runs.parent.OwnerSubject, Purpose: "delegated", ExpiresAt: o.now().Add(time.Hour).Unix()}, o.identitySecret)
				if err != nil {
					t.Fatal(err)
				}
				runs.tokenHash = identity.HashToken(req.RunIdentityToken)
			case "run-without-parent":
				req.OwnerToken = ""
				req.RunIdentityToken = token
			case "owner-unverified":
				owner.Verified = false
			case "owner-service":
				owner.Kind = coreidentity.ActorService
			case "owner-expired":
				owner.ExpiresAt = o.now()
			case "owner-unavailable":
				o.ownerIdentity = nil
			case "owner-empty-grant":
				owner.Scopes = []string{}
			case "owner-read-only":
				owner.Scopes = []string{"agent-manager:read"}
			case "foreign-owner":
				req.RunIdentityToken = token
				req.ParentRunID = &runs.parent.ID
				owner.Subject = "owner-b"
			case "scope-widening":
				req.RequestedScopes = []string{"agent-manager:destructive"}
			case "invalid-second-run":
				req.RunIdentityToken = "invalid-fixture-run"
				req.ParentRunID = &runs.parent.ID
			case "invalid-second-owner":
				req.OwnerToken = "invalid-fixture-owner"
				req.RunIdentityToken = token
				req.ParentRunID = &runs.parent.ID
			}
			if strings.HasPrefix(name, "owner-") && name != "owner-unavailable" || name == "foreign-owner" {
				o.ownerIdentity = createOwnerVerifier(func(context.Context, string) (coreidentity.Principal, error) { return owner, nil })
			}
			got, err := o.CreateRun(context.Background(), req)
			if got != nil || err == nil || strings.Contains(err.Error(), token) {
				t.Fatalf("invalid caller admitted or credential leaked: run=%v err=%v", got, err)
			}
			assertAuth01NoEffects(t, effects)
			if effects.cacheReads != 0 {
				t.Fatal("invalid caller reached idempotency recovery")
			}
		})
	}
}

func TestAuth01VerifiedCallerIntersectsBothChannels(t *testing.T) {
	for _, kind := range []string{"human", "exact-parent", "both", "empty-parent-ceiling"} {
		t.Run(kind, func(t *testing.T) {
			o, runs, effects, req, token, owner := auth01CallerFixture(t)
			if kind != "human" {
				req.RunIdentityToken = token
				req.ParentRunID = &runs.parent.ID
			}
			if kind == "exact-parent" || kind == "empty-parent-ceiling" {
				req.OwnerToken = ""
			}
			if kind == "empty-parent-ceiling" {
				token, err := identity.GenerateToken(&identity.Claims{RunID: runs.parent.ID, TaskID: runs.parent.TaskID, Subject: "owner-a", Scopes: []string{}, ExpiresAt: o.now().Add(time.Hour).Unix()}, o.identitySecret)
				if err != nil {
					t.Fatal(err)
				}
				req.RunIdentityToken = token
				runs.tokenHash = identity.HashToken(token)
			}
			if err := o.authenticateCreateRunCaller(context.Background(), &req); err != nil {
				t.Fatal(err)
			}
			if req.caller == nil || req.OwnerSubject != "owner-a" {
				t.Fatal("verified caller attribution lost")
			}
			if kind == "human" && req.caller.Kind != "human" {
				t.Fatal("human mislabeled")
			}
			if kind != "human" && (req.caller.Kind != "run" || req.caller.RunID != runs.parent.ID) {
				t.Fatal("exact parent mislabeled")
			}
			if kind == "both" && (!req.OwnerExpiresAt.Equal(owner.ExpiresAt) || len(req.OwnerScopes) != 2) {
				t.Fatal("dual channel widened scope or expiry")
			}
			if kind == "empty-parent-ceiling" && len(req.OwnerScopes) != 0 {
				t.Fatal("empty parent grant became authority")
			}
			cfg := &domain.RunConfig{Admission: buildRunAdmission(req, domain.DefaultRunConfig())}
			body, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(body), token) || strings.Contains(string(body), "fixture-owner") {
				t.Fatal("credential persisted")
			}
			var restored domain.RunConfig
			if err := json.Unmarshal(body, &restored); err != nil {
				t.Fatal(err)
			}
			if restored.Admission.CreateCaller == nil || *restored.Admission.CreateCaller != *req.caller {
				t.Fatal("durable JSON caller provenance lost")
			}
			assertAuth01NoEffects(t, effects)
		})
	}
}

func TestAuth01PublicReplayRequiresOriginalCallerAndGrant(t *testing.T) {
	for _, variant := range []string{"same", "missing", "different-owner", "historical-anonymous", "different-task", "reduced-grant", "expanded-grant"} {
		t.Run(variant, func(t *testing.T) {
			o, runs, effects, req, _, owner := auth01CallerFixture(t)
			caller := &domain.CreateRunCaller{Kind: "human", Subject: "owner-a"}
			runs.accepted = &domain.Run{ID: uuid.New(), TaskID: req.TaskID, OwnerSubject: owner.Subject, OwnerScopes: owner.Scopes, ResolvedConfig: &domain.RunConfig{Admission: &domain.RunAdmission{CreateCaller: caller}}}
			switch variant {
			case "missing":
				req.OwnerToken = ""
			case "different-owner":
				owner.Subject = "owner-b"
			case "historical-anonymous":
				runs.accepted.ResolvedConfig.Admission.CreateCaller = nil
				runs.accepted.OwnerSubject = ""
				runs.accepted.OwnerScopes = nil
			case "different-task":
				req.TaskID = uuid.New()
			case "reduced-grant":
				owner.Scopes = []string{"agent-manager:write"}
			case "expanded-grant":
				owner.Scopes = append(append([]string{}, owner.Scopes...), "agent-manager:destructive")
			}
			o.ownerIdentity = createOwnerVerifier(func(context.Context, string) (coreidentity.Principal, error) { return owner, nil })
			got, err := o.CreateRun(context.Background(), req)
			if variant == "same" {
				if err != nil || got == nil || got.ID != runs.accepted.ID {
					t.Fatalf("legitimate replay not recovered: %v", err)
				}
				if effects.taskWrites != 0 || effects.runWrites != 0 || effects.reservations != 0 {
					t.Fatal("replay redispatched")
				}
			} else {
				if err == nil || got != nil {
					t.Fatal("foreign or unproved replay admitted")
				}
				assertAuth01NoEffects(t, effects)
			}
		})
	}
}

func TestAuth01RevalidationRejectsRevokedOrChangedCallerBeforeEffects(t *testing.T) {
	for _, variant := range []string{"same", "revoked", "changed-grant", "expired"} {
		t.Run(variant, func(t *testing.T) {
			o, runs, effects, req, token, owner := auth01CallerFixture(t)
			req.RunIdentityToken, req.ParentRunID = token, &runs.parent.ID
			if err := o.authenticateCreateRunCaller(context.Background(), &req); err != nil {
				t.Fatal(err)
			}
			switch variant {
			case "revoked":
				now := o.now()
				runs.parent.IdentityTokenRevokedAt = &now
			case "changed-grant":
				owner.Scopes = []string{"agent-manager:write"}
			case "expired":
				owner.ExpiresAt = o.now()
			}
			o.ownerIdentity = createOwnerVerifier(func(context.Context, string) (coreidentity.Principal, error) { return owner, nil })
			err := o.revalidateCreateRunCaller(context.Background(), req)
			if (variant == "same") != (err == nil) {
				t.Fatalf("%s: %v", variant, err)
			}
			assertAuth01NoEffects(t, effects)
		})
	}
}

type auth01ProfileReads struct {
	*auth01DelegationProfiles
	existing *domain.AgentProfile
}

func (r *auth01ProfileReads) GetByKey(context.Context, string) (*domain.AgentProfile, error) {
	return r.existing, nil
}
func TestAuth01ProfileReconciliationIsReadOnlyBeforeAdmission(t *testing.T) {
	for _, variant := range []string{"existing", "identical-noop", "new", "changed"} {
		t.Run(variant, func(t *testing.T) {
			o, _, effects, req, _, _ := auth01CallerFixture(t)
			existing := &domain.AgentProfile{ID: uuid.New(), Name: "fixture-profile", ProfileKey: "fixture-profile", RoleRef: "code.default"}
			profiles := &auth01ProfileReads{auth01DelegationProfiles: &auth01DelegationProfiles{}, existing: existing}
			o.profiles = profiles
			ref := &ProfileRef{ProfileKey: existing.ProfileKey}
			if variant == "identical-noop" || variant == "changed" {
				copy := *existing
				ref.Defaults = &copy
				ref.UpdateExisting = true
			}
			if variant == "changed" {
				ref.Defaults.Name = "changed"
			}
			if variant == "new" {
				profiles.existing = nil
				ref.Defaults = existing
			}
			if variant == "new" || variant == "changed" {
				// Actual public entry must stop before resolver, reservation or task write.
				o.tasks = &auth01DelegationTasks{auth01Tasks: &auth01Tasks{effects: effects}, task: &domain.Task{ID: req.TaskID, ScopePath: ".", ProjectRoot: t.TempDir()}}
				req.ProfileRef = ref
				if run, err := o.CreateRun(context.Background(), req); run != nil || err == nil || !strings.Contains(err.Error(), "cannot mutate") {
					t.Fatalf("profile mutation admitted: %v", err)
				}
			} else {
				plan, err := o.planProfileAdmission(context.Background(), ref)
				if err != nil || plan == nil || plan.profile != existing {
					t.Fatalf("existing profile refused: %v", err)
				}
			}
			assertAuth01NoEffects(t, effects)
			if profiles.writes != 0 {
				t.Fatal("profile mutated before admission")
			}
		})
	}
}
