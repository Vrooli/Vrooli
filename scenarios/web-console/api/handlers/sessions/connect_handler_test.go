package sessions

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
)

type fakeSessionsService struct{}

func (fakeSessionsService) Create(context.Context, CreateInput) (Session, error) {
	return Session{ID: "s1", Shell: "/bin/sh", CreatedAt: "now", Cols: 80, Rows: 24, Backend: "standard", Policy: Policy{Mode: "ttl", Duration: "1h"}, Origin: "ui"}, nil
}

func (fakeSessionsService) List(context.Context) ([]Session, error) {
	return []Session{{ID: "s1", Origin: "programmatic"}}, nil
}

func (fakeSessionsService) ListArchived(context.Context) ([]ArchivedSession, error) {
	return []ArchivedSession{{ID: "a1", RestoreState: RestoreStateReopenable}, {ID: "a2", RestoreState: RestoreStateReadOnly}, {ID: "a3", RestoreState: RestoreStateNothingToRestore}}, nil
}

func (fakeSessionsService) RecoveryStatus(context.Context) RecoveryStatus {
	return RecoveryStatus{InProgress: true, Total: 3, Recovered: 1, AwaitingRecovery: 1, Adopted: 1, StartedAtUnixMs: 10, CompletedAtUnixMs: 20}
}

func (fakeSessionsService) Get(context.Context, string) (Session, error) {
	return Session{ID: "s1", Origin: "remote", TrackingDegraded: true}, nil
}
func (fakeSessionsService) Archive(context.Context, string) error   { return nil }
func (fakeSessionsService) Unarchive(context.Context, string) error { return nil }
func (fakeSessionsService) Delete(context.Context, string) error    { return nil }
func (fakeSessionsService) GetArchiveRetention(context.Context) (ArchiveRetentionSnapshot, error) {
	return ArchiveRetentionSnapshot{Policy: ArchiveRetentionPolicy{MessageLessAge: 48 * time.Hour, AgentHomeAge: 72 * time.Hour, MaxBytes: 99}, Stats: ArchiveRetentionStats{EntryCount: 1, MessageCount: 2, TranscriptBytes: 3, AgentHomeBytes: 4, TotalBytes: 7}}, nil
}

func (fakeSessionsService) PruneArchive(context.Context, bool) (ArchivePruneResult, error) {
	stats := ArchiveRetentionStats{EntryCount: 1}
	return ArchivePruneResult{DryRun: true, Actions: []ArchivePruneAction{{SessionID: "s1", Kind: PruneTranscript, Bytes: 4}}, ReclaimedBytes: 4, Before: stats, After: stats}, nil
}

func (fakeSessionsService) ListRecoverable(context.Context) ([]RecoverableSession, error) {
	return []RecoverableSession{{ID: "r1", Backend: "standard", Shell: "/bin/sh", Cols: 80, Rows: 24, Recoverable: true, NotRecoverable: "", HeaderColor: "blue"}}, nil
}
func (fakeSessionsService) DismissRecoverable(context.Context, string) error { return nil }
func (fakeSessionsService) Recover(context.Context, RecoverInput) (RecoverResult, error) {
	return RecoverResult{OldSessionID: "old", NewSessionID: "new", AgentType: "codex", CommandSent: "resume", CodexHomeCopied: true}, nil
}

func (fakeSessionsService) GetPolicy(context.Context, string) (PolicyView, error) {
	return PolicyView{SessionID: "s1", Policy: Policy{Mode: "ttl", Duration: "1h"}, ExpiresAt: "later", TTLSeconds: 60, HasExpiry: true}, nil
}

func (fakeSessionsService) UpdatePolicy(context.Context, string, Policy) (PolicyView, error) {
	return PolicyView{SessionID: "s1", Policy: Policy{Mode: "never"}}, nil
}

func request[T any](msg *T) *connect.Request[T] { return connect.NewRequest(msg) }

func TestConnectHandlerOperations(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakeSessionsService{}})
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "fixture-human", Source: identity.SourcePersonalLocal, Verified: true, Scopes: []string{"web-console:write"}})

	createReq := request(&sessionsv1.CreateRequest{Shell: "/bin/bash", Cols: 100, Rows: 40, HasPolicy: true, Policy: &sessionsv1.ExpirationPolicy{Mode: "ttl", Duration: "1h"}, Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_UI})
	createReq.Header().Set(idempotencyHeader, "key")
	if resp, err := h.Create(ctx, createReq); err != nil || resp.Msg.Session.Id != "s1" || resp.Msg.Session.Origin != sessionsv1.SessionOrigin_SESSION_ORIGIN_UI {
		t.Fatalf("create: %#v %v", resp, err)
	}
	if resp, err := h.List(ctx, request(&sessionsv1.ListRequest{})); err != nil || len(resp.Msg.Sessions) != 1 || resp.Msg.Recovery.Total != 3 {
		t.Fatalf("list: %#v %v", resp, err)
	}
	if resp, err := h.ListArchived(ctx, request(&sessionsv1.ListArchivedRequest{})); err != nil || len(resp.Msg.Sessions) != 3 {
		t.Fatalf("archived: %#v %v", resp, err)
	}
	if resp, err := h.Get(ctx, request(&sessionsv1.GetRequest{Id: "s1"})); err != nil || resp.Msg.Session.Origin != sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE {
		t.Fatalf("get: %#v %v", resp, err)
	}
	if _, err := h.Delete(ctx, request(&sessionsv1.DeleteRequest{Id: "s1", Confirmation: "DELETE:s1"})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Archive(ctx, request(&sessionsv1.ArchiveRequest{Id: "s1"})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Delete(context.Background(), request(&sessionsv1.DeleteRequest{Id: "s1", Confirmation: "wrong"})); err == nil {
		t.Fatal("delete without the exact destructive confirmation succeeded")
	}
	if _, err := h.Unarchive(ctx, request(&sessionsv1.UnarchiveRequest{Id: "s1"})); err != nil {
		t.Fatal(err)
	}
	if resp, err := h.ListRecoverable(ctx, request(&sessionsv1.ListRecoverableRequest{})); err != nil || len(resp.Msg.Sessions) != 1 {
		t.Fatalf("recoverable: %#v %v", resp, err)
	}
	if _, err := h.DismissRecoverable(ctx, request(&sessionsv1.DismissRecoverableRequest{Id: "r1"})); err != nil {
		t.Fatal(err)
	}
	recoverReq := request(&sessionsv1.RecoverRequest{Id: "r1"})
	recoverReq.Header().Set(idempotencyHeader, "recover")
	if resp, err := h.Recover(ctx, recoverReq); err != nil || resp.Msg.NewSessionId != "new" || !resp.Msg.CodexHomeCopied {
		t.Fatalf("recover: %#v %v", resp, err)
	}
	if resp, err := h.Reopen(ctx, request(&sessionsv1.ReopenRequest{Id: "r1"})); err != nil || resp.Msg.OldSessionId != "old" {
		t.Fatalf("reopen: %#v %v", resp, err)
	}
	if resp, err := h.GetArchiveRetention(ctx, request(&sessionsv1.GetArchiveRetentionRequest{})); err != nil || resp.Msg.Policy.MessageLessAgeDays != 2 || resp.Msg.Stats.TotalBytes != 7 {
		t.Fatalf("retention: %#v %v", resp, err)
	}
	if resp, err := h.PruneArchive(ctx, request(&sessionsv1.PruneArchiveRequest{Apply: false})); err != nil || len(resp.Msg.Actions) != 1 || !resp.Msg.DryRun {
		t.Fatalf("prune: %#v %v", resp, err)
	}
	if resp, err := h.GetPolicy(ctx, request(&sessionsv1.GetPolicyRequest{Id: "s1"})); err != nil || resp.Msg.Policy.Policy.Mode != "ttl" {
		t.Fatalf("policy: %#v %v", resp, err)
	}
	if resp, err := h.UpdatePolicy(ctx, request(&sessionsv1.UpdatePolicyRequest{Id: "s1", Policy: &sessionsv1.ExpirationPolicy{Mode: "never"}})); err != nil || resp.Msg.Policy.Policy.Mode != "never" {
		t.Fatalf("update policy: %#v %v", resp, err)
	}
}

func TestConnectHandlerValidationAndClassification(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakeSessionsService{}})
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"get", func() error { _, e := h.Get(context.Background(), request(&sessionsv1.GetRequest{})); return e }},
		{"delete", func() error { _, e := h.Delete(context.Background(), request(&sessionsv1.DeleteRequest{})); return e }},
		{"archive", func() error { _, e := h.Archive(context.Background(), request(&sessionsv1.ArchiveRequest{})); return e }},
		{"unarchive", func() error {
			_, e := h.Unarchive(context.Background(), request(&sessionsv1.UnarchiveRequest{}))
			return e
		}},
		{"dismiss", func() error {
			_, e := h.DismissRecoverable(context.Background(), request(&sessionsv1.DismissRecoverableRequest{}))
			return e
		}},
		{"recover", func() error { _, e := h.Recover(context.Background(), request(&sessionsv1.RecoverRequest{})); return e }},
		{"reopen", func() error { _, e := h.Reopen(context.Background(), request(&sessionsv1.ReopenRequest{})); return e }},
		{"get policy", func() error {
			_, e := h.GetPolicy(context.Background(), request(&sessionsv1.GetPolicyRequest{}))
			return e
		}},
		{"update policy id", func() error {
			_, e := h.UpdatePolicy(context.Background(), request(&sessionsv1.UpdatePolicyRequest{}))
			return e
		}},
		{"update policy body", func() error {
			_, e := h.UpdatePolicy(context.Background(), request(&sessionsv1.UpdatePolicyRequest{Id: "s1"}))
			return e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ce *connect.Error
			if err := tc.call(); !errors.As(err, &ce) || ce.Code() != connect.CodeInvalidArgument {
				t.Fatalf("got %v", err)
			}
		})
	}
	for _, tc := range []struct {
		in   error
		want connect.Code
	}{
		{ErrNotFound, connect.CodeNotFound},
		{ErrInvalidArgument, connect.CodeInvalidArgument},
		{ErrFailedPrecondition, connect.CodeFailedPrecondition},
		{ErrTargetUnavailable, connect.CodeFailedPrecondition},
		{ErrTargetNotFound, connect.CodeNotFound},
		{ErrIdempotencyConflict, connect.CodeAlreadyExists},
		{ErrRemoteUnavailable, connect.CodeUnavailable},
		{ErrResourceExhausted, connect.CodeResourceExhausted},
		{ErrUnavailable, connect.CodeUnavailable},
		{ErrInternal, connect.CodeInternal},
		{errors.New("other"), connect.CodeInternal},
	} {
		var ce *connect.Error
		if err := h.classify(tc.in, "test"); !errors.As(err, &ce) || ce.Code() != tc.want {
			t.Errorf("%v: got %v want %v", tc.in, err, tc.want)
		}
	}
	for _, origin := range []string{"ui", "programmatic", "remote", "unknown"} {
		_ = originToEnum(origin)
	}
	for _, origin := range []sessionsv1.SessionOrigin{sessionsv1.SessionOrigin_SESSION_ORIGIN_UI, sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC, sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE, sessionsv1.SessionOrigin_SESSION_ORIGIN_UNSPECIFIED} {
		_ = originToString(origin)
	}
}

func TestReadNativeUsageRequiresVerifiedReadCapability(t *testing.T) {
	calls := 0
	h := NewConnectHandler(Deps{Service: &Adapter{NativeUsage: func(context.Context, *sessionsv1.ReadNativeUsageRequest) (*sessionsv1.ReadNativeUsageResponse, error) {
		calls++
		return &sessionsv1.ReadNativeUsageResponse{Status: "ok"}, nil
	}}})
	cases := []struct {
		name      string
		principal *identity.Principal
		code      connect.Code
	}{
		{"anonymous", nil, connect.CodeUnauthenticated},
		{"unverified", &identity.Principal{Scopes: []string{"web-console:read"}}, connect.CodeUnauthenticated},
		{"wrong scope", &identity.Principal{Verified: true, Kind: identity.ActorAgent, Subject: "test", Scopes: []string{"agent-manager:read"}}, connect.CodePermissionDenied},
		{"scoped read", &identity.Principal{Verified: true, Kind: identity.ActorAgent, Subject: "test", Scopes: []string{"web-console:read"}}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.principal != nil {
				ctx = identity.WithPrincipal(ctx, *tc.principal)
			}
			_, err := h.ReadNativeUsage(ctx, connect.NewRequest(&sessionsv1.ReadNativeUsageRequest{}))
			if (err == nil && tc.code != 0) || (err != nil && connect.CodeOf(err) != tc.code) {
				t.Fatalf("code %v, error %v", connect.CodeOf(err), err)
			}
		})
	}
	if calls != 1 {
		t.Fatalf("owner invoked %d times", calls)
	}
}

func TestReadNativeUsageVerifiedOwnSession(t *testing.T) {
	calls := 0
	var bound string
	h := NewConnectHandler(Deps{VerifyOwnSession: func(context.Context) (string, error) { return bound, nil }, Service: &Adapter{NativeUsage: func(context.Context, *sessionsv1.ReadNativeUsageRequest) (*sessionsv1.ReadNativeUsageResponse, error) {
		calls++
		return &sessionsv1.ReadNativeUsageResponse{}, nil
	}}})
	for _, tc := range []struct {
		name, bound string
		verified    bool
		want        bool
	}{{"own", "session-a", true, true}, {"other", "session-b", true, false}, {"missing", "", true, false}, {"unverified", "session-a", false, false}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := identity.WithPrincipal(context.Background(), identity.Principal{Verified: true, Kind: identity.ActorAgent, Subject: "agent"})
			status := provenance.VerificationInvalid
			if tc.verified {
				status = provenance.VerificationVerified
			}
			bound = tc.bound
			ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: status, RunID: "run-a", SessionID: "session-a"})
			_, err := h.ReadNativeUsage(ctx, connect.NewRequest(&sessionsv1.ReadNativeUsageRequest{SessionId: "session-a"}))
			if tc.want && err != nil {
				t.Fatal(err)
			}
			if !tc.want && connect.CodeOf(err) != connect.CodePermissionDenied {
				t.Fatalf("error %v", err)
			}
		})
	}
	if calls != 1 {
		t.Fatalf("owner invoked %d times", calls)
	}
}

func TestReadNativeThreadStateUsesSameSessionAuthorization(t *testing.T) {
	calls := 0
	h := NewConnectHandler(Deps{VerifyOwnSession: func(context.Context) (string, error) { return "own", nil }, Service: &Adapter{NativeThreadState: func(context.Context, *sessionsv1.ReadNativeThreadStateRequest) (*sessionsv1.ReadNativeThreadStateResponse, error) {
		calls++
		return &sessionsv1.ReadNativeThreadStateResponse{}, nil
	}}})
	verified := identity.WithPrincipal(context.Background(), identity.Principal{Verified: true, Kind: identity.ActorAgent, Subject: "agent"})
	verified = provenance.NewContext(verified, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", SessionID: "own"})
	for _, tc := range []struct {
		ctx     context.Context
		session string
		want    connect.Code
	}{
		{context.Background(), "own", connect.CodeUnauthenticated},
		{verified, "other", connect.CodePermissionDenied},
		{verified, "own", 0},
	} {
		_, err := h.ReadNativeThreadState(tc.ctx, connect.NewRequest(&sessionsv1.ReadNativeThreadStateRequest{SessionId: tc.session}))
		if (err == nil && tc.want != 0) || (err != nil && connect.CodeOf(err) != tc.want) {
			t.Fatalf("session=%s error=%v", tc.session, err)
		}
	}
	if calls != 1 {
		t.Fatalf("owner calls=%d", calls)
	}
	withoutVerifier := NewConnectHandler(Deps{Service: &Adapter{}})
	_, err := withoutVerifier.ReadNativeThreadState(verified, connect.NewRequest(&sessionsv1.ReadNativeThreadStateRequest{SessionId: "own"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("missing verifier must fail closed: %v", err)
	}
}

func TestNativeReadsRequireExplicitServiceReadCapability(t *testing.T) {
	for _, method := range []string{"thread state", "usage"} {
		t.Run(method, func(t *testing.T) {
			for _, tc := range []struct {
				name      string
				principal identity.Principal
				want      connect.Code
			}{
				{"configured manager", identity.Principal{Verified: true, Kind: identity.ActorService, Subject: "agent-manager", Scopes: []string{"web-console:read", "web-console:write"}}, 0},
				{"manager write alone", identity.Principal{Verified: true, Kind: identity.ActorService, Subject: "agent-manager", Scopes: []string{"web-console:write"}}, connect.CodePermissionDenied},
				{"other service write", identity.Principal{Verified: true, Kind: identity.ActorService, Subject: "other-service", Scopes: []string{"web-console:write"}}, connect.CodePermissionDenied},
				{"unverified manager claims", identity.Principal{Kind: identity.ActorService, Subject: "agent-manager", Scopes: []string{"web-console:read", "web-console:write"}}, connect.CodeUnauthenticated},
				{"wrong resource", identity.Principal{Verified: true, Kind: identity.ActorService, Subject: "agent-manager", Scopes: []string{"agent-manager:read"}}, connect.CodePermissionDenied},
			} {
				t.Run(tc.name, func(t *testing.T) {
					calls := 0
					h := NewConnectHandler(Deps{Service: &Adapter{
						NativeThreadState: func(context.Context, *sessionsv1.ReadNativeThreadStateRequest) (*sessionsv1.ReadNativeThreadStateResponse, error) {
							calls++
							return &sessionsv1.ReadNativeThreadStateResponse{}, nil
						},
						NativeUsage: func(context.Context, *sessionsv1.ReadNativeUsageRequest) (*sessionsv1.ReadNativeUsageResponse, error) {
							calls++
							return &sessionsv1.ReadNativeUsageResponse{}, nil
						},
					}})
					ctx := identity.WithPrincipal(context.Background(), tc.principal)
					var err error
					if method == "thread state" {
						_, err = h.ReadNativeThreadState(ctx, connect.NewRequest(&sessionsv1.ReadNativeThreadStateRequest{SessionId: "managed-session"}))
					} else {
						_, err = h.ReadNativeUsage(ctx, connect.NewRequest(&sessionsv1.ReadNativeUsageRequest{SessionId: "managed-session"}))
					}
					if tc.want == 0 && err != nil || tc.want != 0 && connect.CodeOf(err) != tc.want {
						t.Fatalf("unexpected authorization result: %v", err)
					}
					if (tc.want == 0 && calls != 1) || (tc.want != 0 && calls != 0) {
						t.Fatalf("owner calls=%d", calls)
					}
				})
			}
		})
	}
}

type nativeGoalServiceFixture struct {
	fakeSessionsService
	calls int
}

func (f *nativeGoalServiceFixture) SetNativeGoal(_ context.Context, req *sessionsv1.SetNativeGoalRequest) (*sessionsv1.SetNativeGoalResponse, error) {
	f.calls++
	return &sessionsv1.SetNativeGoalResponse{ThreadId: req.GetExpectedThreadId(), Status: req.GetStatus(), Objective: req.GetObjective()}, nil
}

func TestNativeGoalOriginCannotBeForgedByPublicAgentOrService(t *testing.T) {
	user := sessionsv1.NativeGoalMutationOrigin_NATIVE_GOAL_MUTATION_ORIGIN_USER_DIRECTED
	automatic := sessionsv1.NativeGoalMutationOrigin_NATIVE_GOAL_MUTATION_ORIGIN_AUTOMATIC
	for _, tc := range []struct {
		name    string
		kind    identity.ActorKind
		source  identity.AuthSource
		subject string
		origin  sessionsv1.NativeGoalMutationOrigin
		allowed bool
	}{
		{"human explicit", identity.ActorHuman, identity.SourceScenarioAuthenticator, "operator", user, true},
		{"admitted AM", identity.ActorService, identity.SourceScenarioAuthenticator, "agent-manager", user, true},
		{"agent assertion", identity.ActorAgent, identity.SourceScenarioAuthenticator, "agent", user, false},
		{"other service assertion", identity.ActorService, identity.SourceScenarioAuthenticator, "other-service", user, false},
		{"AM unverified source", identity.ActorService, identity.SourcePersonalLocal, "agent-manager", user, false},
		{"agent automatic", identity.ActorAgent, identity.SourceScenarioAuthenticator, "agent", automatic, true},
		{"missing origin", identity.ActorHuman, identity.SourceScenarioAuthenticator, "operator", 0, false},
		{"unknown origin", identity.ActorHuman, identity.SourceScenarioAuthenticator, "operator", 999, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &nativeGoalServiceFixture{}
			h := NewConnectHandler(Deps{Service: svc})
			ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: tc.kind, Source: tc.source, Subject: tc.subject, Verified: true, Scopes: []string{"web-console:write"}})
			_, err := h.SetNativeGoal(ctx, connect.NewRequest(&sessionsv1.SetNativeGoalRequest{SessionId: "session", ExpectedNativeOwner: "owner", ExpectedThreadId: "thread", Objective: "exact admitted task", Status: "active", Origin: tc.origin}))
			if (err == nil) != tc.allowed {
				t.Fatalf("err=%v allowed=%t", err, tc.allowed)
			}
			if (svc.calls > 0) != tc.allowed {
				t.Fatalf("owner calls=%d allowed=%t", svc.calls, tc.allowed)
			}
		})
	}
}

type retirementAuthorityFixture struct {
	fakeSessionsService
	calls int
}

func (f *retirementAuthorityFixture) Archive(context.Context, string) error { f.calls++; return nil }
func (f *retirementAuthorityFixture) Delete(context.Context, string) error  { f.calls++; return nil }
func TestSessionArchiveFailsOpenAndPermanentDeleteKeepsGuards(t *testing.T) {
	for _, method := range []string{"archive", "delete"} {
		t.Run(method, func(t *testing.T) {
			for _, tc := range []struct {
				name      string
				principal *identity.Principal
				want      connect.Code
				ownAgent  bool
			}{
				{name: "anonymous", want: connect.CodeUnauthenticated},
				{name: "unverified claims", principal: &identity.Principal{Kind: identity.ActorService, Subject: "agent-manager", Scopes: []string{"web-console:write"}}, want: connect.CodeUnauthenticated},
				{name: "read only", principal: &identity.Principal{Verified: true, Kind: identity.ActorService, Subject: "agent-manager", Scopes: []string{"web-console:read"}}, want: connect.CodePermissionDenied},
				{name: "wrong resource", principal: &identity.Principal{Verified: true, Kind: identity.ActorService, Subject: "agent-manager", Scopes: []string{"agent-manager:write"}}, want: connect.CodePermissionDenied},
				{name: "AM service write", principal: &identity.Principal{Verified: true, Kind: identity.ActorService, Subject: "agent-manager", Source: identity.SourceScenarioAuthenticator, Scopes: []string{"web-console:write"}}},
				{name: "human write", principal: &identity.Principal{Verified: true, Kind: identity.ActorHuman, Subject: "fixture-human", Source: identity.SourcePersonalLocal, Scopes: []string{"web-console:write"}}},
				{name: "human without scenario scope", principal: &identity.Principal{Verified: true, Kind: identity.ActorHuman, Subject: "fixture-human", Source: identity.SourcePersonalLocal}},
				{name: "verified agent own session", principal: &identity.Principal{Verified: true, Kind: identity.ActorAgent, Subject: "fixture-agent"}, want: connect.CodePermissionDenied, ownAgent: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					svc := &retirementAuthorityFixture{}
					h := NewConnectHandler(Deps{Service: svc, VerifyOwnSession: func(context.Context) (string, error) { return "exact-session", nil }})
					ctx := t.Context()
					if tc.principal != nil {
						ctx = identity.WithPrincipal(ctx, *tc.principal)
					}
					if tc.ownAgent {
						ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-a", SessionID: "exact-session"})
					}
					var err error
					if method == "archive" {
						_, err = h.Archive(ctx, request(&sessionsv1.ArchiveRequest{Id: "exact-session"}))
					} else {
						_, err = h.Delete(ctx, request(&sessionsv1.DeleteRequest{Id: "exact-session", Confirmation: "DELETE:exact-session"}))
					}
					wantCode := tc.want
					if method == "archive" {
						wantCode = 0
					}
					if wantCode == 0 && err != nil || wantCode != 0 && connect.CodeOf(err) != wantCode {
						t.Fatalf("error=%v wantCode=%v", err, wantCode)
					}
					wantCalls := 0
					if wantCode == 0 {
						wantCalls = 1
					}
					if svc.calls != wantCalls {
						t.Fatalf("retirement calls=%d want%d", svc.calls, wantCalls)
					}
				})
			}
		})
	}
}
