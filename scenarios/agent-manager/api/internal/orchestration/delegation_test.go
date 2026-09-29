package orchestration

import (
	"context"
	"slices"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
)

func TestDependentSandboxPermissionsSurviveWireRoundTrip(t *testing.T) {
	parent := &domain.SandboxConfig{
		Mode: domain.SandboxModeProtected, NetworkMode: domain.NetworkAccessNone, ManualReview: true,
		Acceptance:  domain.SandboxAcceptanceConfig{Mode: "allowlist", Allow: domain.SandboxFileCriteria{PathGlobs: []string{}}, Deny: domain.SandboxFileCriteria{Extensions: []string{".secret"}}},
		Lifecycle:   domain.SandboxLifecycleConfig{CheckpointOn: []domain.SandboxLifecycleEvent{}, TTL: time.Hour},
		WritePolicy: &domain.WorkspaceWritePolicy{Paths: []string{"src"}},
	}
	for _, variant := range []string{"same", "mode", "allow-globs", "allow-extensions", "deny-globs", "deny-extensions", "ignore-binary", "checkpoint", "stop", "delete", "ttl", "idle"} {
		t.Run(variant, func(t *testing.T) {
			child := protoconv.SandboxConfigFromProto(protoconv.SandboxConfigToProto(parent))
			switch variant {
			case "mode":
				child.Acceptance.Mode = ""
			case "allow-globs":
				child.Acceptance.Allow.PathGlobs = []string{"**"}
			case "allow-extensions":
				child.Acceptance.Allow.Extensions = []string{".go"}
			case "deny-globs":
				child.Acceptance.Deny.PathGlobs = []string{"private/**"}
			case "deny-extensions":
				child.Acceptance.Deny.Extensions = nil
			case "ignore-binary":
				child.Acceptance.IgnoreBinary = true
			case "checkpoint":
				child.Lifecycle.CheckpointOn = []domain.SandboxLifecycleEvent{"run_completed"}
			case "stop":
				child.Lifecycle.StopOn = []domain.SandboxLifecycleEvent{"run_completed"}
			case "delete":
				child.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{"run_completed"}
			case "ttl":
				child.Lifecycle.TTL = 2 * time.Hour
			case "idle":
				child.Lifecycle.IdleTimeout = time.Minute
			}
			if got := delegatedSandboxWithin(parent, child); got != (variant == "same") {
				t.Fatalf("policy round-trip admission=%v; nil/empty representations must be equivalent, changed controls must refuse", got)
			}
		})
	}
}

func TestMintDelegatedIdentityRespectsChildAuthority(t *testing.T) {
	for _, variant := range []string{"narrowed", "omitted-profile-scopes", "empty-request", "empty-owner", "no-profile", "revoked", "terminal", "wrong-owner", "expired-owner", "excess-expiry", "supervisor-binding"} {
		t.Run(variant, func(t *testing.T) {
			ctx := context.Background()
			repos, _, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			now := time.Now().Truncate(time.Second)
			secret := []byte("child-authority-regression")
			parent := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), Status: domain.RunStatusRunning}
			task := &domain.Task{ID: parent.TaskID, Title: "delegation fixture", ScopePath: "src"}
			if err := repos.Tasks.Create(ctx, task); err != nil {
				t.Fatal(err)
			}
			parentToken, err := identity.GenerateToken(&identity.Claims{
				RunID: parent.ID, TaskID: task.ID, Subject: "owner-a", ProfileKey: "parent",
				ScopePath: task.ScopePath, Scopes: []string{"agent-manager:read", "agent-manager:supervise"},
				IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
				Meta: map[string]string{"workflowNodeId": "parent-review"},
			}, secret)
			if err != nil {
				t.Fatal(err)
			}
			parent.IdentityTokenHash = identity.HashToken(parentToken)
			if err := repos.Runs.Create(ctx, parent); err != nil {
				t.Fatal(err)
			}
			profile := &domain.AgentProfile{ID: uuid.New(), Name: "child", ProfileKey: "child", DeclaredScopes: []string{"agent-manager:read"}}
			expires := now.Add(20 * time.Minute)
			child := &domain.Run{
				ID: uuid.New(), TaskID: task.ID, ParentRunID: &parent.ID, AgentProfileID: &profile.ID,
				Status: domain.RunStatusRunning, OwnerSubject: "owner-a", OwnerExpiresAt: &expires,
				OwnerScopes: []string{"agent-manager:read", "agent-manager:supervise"},
				CustomEnv:   map[string]string{workflowNodeEnv: "child-implementation"},
			}
			req := MintDelegatedIdentityRequest{ParentToken: parentToken, ChildRunID: child.ID}
			want := []string{"agent-manager:read"}
			wantErr := false
			switch variant {
			case "omitted-profile-scopes":
				profile.DeclaredScopes, want = nil, nil
			case "empty-request":
				child.RequestedScopes, want = []string{}, nil
			case "empty-owner":
				child.OwnerScopes, want = nil, nil
			case "no-profile":
				child.AgentProfileID = nil
				child.RequestedScopes = []string{"agent-manager:read"}
			case "revoked":
				child.IdentityTokenRevokedAt, wantErr = &now, true
			case "terminal":
				child.Status, wantErr = domain.RunStatusComplete, true
			case "wrong-owner":
				child.OwnerSubject, wantErr = "owner-b", true
			case "expired-owner":
				child.OwnerExpiresAt, wantErr = &now, true
			case "excess-expiry":
				req.ExpiresAt, wantErr = now.Add(40*time.Minute), true
			case "supervisor-binding":
				child.DispatchBinding = &domain.DispatchBinding{EffortRef: "effort:other", AuthorizationID: "other-grant"}
				wantErr = true
			}
			if err := repos.Profiles.Create(ctx, profile); err != nil {
				t.Fatal(err)
			}
			if err := repos.Runs.Create(ctx, child); err != nil {
				t.Fatal(err)
			}
			o := &Orchestrator{runs: repos.Runs, profiles: repos.Profiles, identitySecret: secret, clock: func() time.Time { return now }}
			result, err := o.MintDelegatedIdentity(ctx, req)
			if wantErr {
				if err == nil || result != nil {
					t.Fatal("child authority restriction did not prevent credential issuance")
				}
				saved, readErr := repos.Runs.Get(ctx, child.ID)
				if readErr != nil || saved.IdentityTokenHash != child.IdentityTokenHash {
					t.Fatal("refused issuance changed the child credential", readErr)
				}
				return
			}
			if err != nil || result == nil {
				t.Fatalf("allowed exchange failed: %v", err)
			}
			verified, err := o.VerifyIdentityToken(ctx, result.Token)
			if err != nil || verified == nil || !verified.Valid {
				t.Fatalf("issued credential is not active: %v", err)
			}
			claims := verified.Claims
			if !slices.Equal(claims.Scopes, want) || claims.ExpiresAt != expires.Unix() {
				t.Fatalf("child ceiling lost: scopes=%v expiry=%v", claims.Scopes, claims.ExpiresAt)
			}
			wantProfile := profile.ProfileKey
			if child.AgentProfileID == nil {
				wantProfile = ""
			}
			if claims.ProfileKey != wantProfile {
				t.Fatalf("child credential retained parent profile %q", claims.ProfileKey)
			}
			if claims.Meta["workflowNodeId"] != "child-implementation" {
				t.Fatal("child credential impersonated the parent workflow node")
			}
			second, err := o.MintDelegatedIdentity(ctx, req)
			if err != nil || second.Token == result.Token {
				t.Fatal("replacement reused the earlier credential generation", err)
			}
			old, err := o.VerifyIdentityToken(ctx, result.Token)
			if err != nil || old.Valid {
				t.Fatal("replaced credential remained active", err)
			}
		})
	}
}
