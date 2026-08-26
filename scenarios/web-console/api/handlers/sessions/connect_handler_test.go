package sessions

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
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
	ctx := context.Background()

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
	if _, err := h.Delete(ctx, request(&sessionsv1.DeleteRequest{Id: "s1"})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Archive(ctx, request(&sessionsv1.ArchiveRequest{Id: "s1"})); err != nil {
		t.Fatal(err)
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
