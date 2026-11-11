package scenarioruntime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type attachedEditorLeaseLookupFunc func(context.Context, string) (EditorLease, error)

func (f attachedEditorLeaseLookupFunc) GetEditorLease(ctx context.Context, id string) (EditorLease, error) {
	return f(ctx, id)
}

func TestResolveAttachedEditorLeaseExactKeyPolicy(t *testing.T) {
	unavailable := errors.New("registry unavailable")
	canonical := EditorLease{SessionID: "host-session", PID: 123, Status: EditorLeaseActive}
	legacy := EditorLease{SessionID: "am-run", PID: 456, Status: EditorLeaseActive}
	for _, tc := range []struct {
		name       string
		harnessID  string
		canonical  EditorLease
		primaryErr error
		legacy     EditorLease
		legacyErr  error
		want       EditorLease
		wantErr    error
		wantKeys   []string
	}{
		{name: "canonical wins over different legacy process", harnessID: "host-session", canonical: canonical, legacy: legacy, want: canonical, wantKeys: []string{"host-session"}},
		{name: "legacy only when canonical missing", harnessID: "host-session", primaryErr: ErrNotFound, legacy: legacy, want: legacy, wantKeys: []string{"host-session", "am-run"}},
		{name: "wrapped missing permits legacy", harnessID: "host-session", primaryErr: fmt.Errorf("lookup: %w", ErrNotFound), legacy: legacy, want: legacy, wantKeys: []string{"host-session", "am-run"}},
		{name: "missing immutable anchor permits maintenance legacy", legacy: legacy, want: legacy, wantKeys: []string{"am-run"}},
		{name: "canonical unavailable never falls back", harnessID: "host-session", primaryErr: unavailable, legacy: legacy, wantErr: unavailable, wantKeys: []string{"host-session"}},
		{name: "canonical cancellation never falls back", harnessID: "host-session", primaryErr: context.Canceled, legacy: legacy, wantErr: context.Canceled, wantKeys: []string{"host-session"}},
		{name: "canonical deadline never falls back", harnessID: "host-session", primaryErr: context.DeadlineExceeded, legacy: legacy, wantErr: context.DeadlineExceeded, wantKeys: []string{"host-session"}},
		{name: "canonical mismatching row never falls back", harnessID: "host-session", canonical: legacy, legacy: legacy, wantErr: ErrEditorLeaseIdentityMismatch, wantKeys: []string{"host-session"}},
		{name: "empty canonical row never falls back", harnessID: "host-session", legacy: legacy, wantErr: ErrEditorLeaseIdentityMismatch, wantKeys: []string{"host-session"}},
		{name: "canonical conflict remains caller validation responsibility", harnessID: "host-session", canonical: EditorLease{SessionID: "host-session", Status: EditorLeaseStopped}, legacy: legacy, want: EditorLease{SessionID: "host-session", Status: EditorLeaseStopped}, wantKeys: []string{"host-session"}},
		{name: "missing legacy propagates", harnessID: "host-session", primaryErr: ErrNotFound, legacyErr: ErrNotFound, wantErr: ErrNotFound, wantKeys: []string{"host-session", "am-run"}},
		{name: "unavailable legacy propagates", harnessID: "host-session", primaryErr: ErrNotFound, legacyErr: unavailable, wantErr: unavailable, wantKeys: []string{"host-session", "am-run"}},
		{name: "mismatching legacy row rejected", harnessID: "host-session", primaryErr: ErrNotFound, legacy: canonical, wantErr: ErrEditorLeaseIdentityMismatch, wantKeys: []string{"host-session", "am-run"}},
		{name: "legacy-only mismatching row rejected", legacy: canonical, wantErr: ErrEditorLeaseIdentityMismatch, wantKeys: []string{"am-run"}},
		{name: "same canonical and legacy key read once", harnessID: "am-run", legacyErr: ErrNotFound, wantErr: ErrNotFound, wantKeys: []string{"am-run"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var keys []string
			reader := attachedEditorLeaseLookupFunc(func(_ context.Context, key string) (EditorLease, error) {
				keys = append(keys, key)
				switch key {
				case "host-session":
					return tc.canonical, tc.primaryErr
				case "am-run":
					return tc.legacy, tc.legacyErr
				default:
					t.Fatalf("unexpected lookup %q", key)
					return EditorLease{}, ErrNotFound
				}
			})
			got, err := ResolveAttachedEditorLease(context.Background(), reader, "am-run", tc.harnessID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("lease = %+v, want %+v", got, tc.want)
			}
			if !reflect.DeepEqual(keys, tc.wantKeys) {
				t.Fatalf("lookup keys = %q, want %q", keys, tc.wantKeys)
			}
		})
	}
}

func TestResolveAttachedEditorLeaseRequiresReaderAndExactNonemptySelectors(t *testing.T) {
	reader := attachedEditorLeaseLookupFunc(func(context.Context, string) (EditorLease, error) {
		t.Fatal("invalid input must not read the registry")
		return EditorLease{}, nil
	})
	for _, tc := range []struct {
		name             string
		reader           EditorLeaseLookup
		runID, harnessID string
	}{
		{name: "missing reader", runID: "am-run", harnessID: "host-session"},
		{name: "missing run id", reader: reader, harnessID: "host-session"},
		{name: "blank run id", reader: reader, runID: " \t", harnessID: "host-session"},
		{name: "no selector", reader: reader},
		{name: "blank harness id", reader: reader, runID: "am-run", harnessID: " \t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ResolveAttachedEditorLease(context.Background(), tc.reader, tc.runID, tc.harnessID); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestResolveAttachedEditorLeaseHonorsCancellationBeforeAndAfterReads(t *testing.T) {
	for _, tc := range []struct {
		name         string
		cancelBefore bool
		missing      bool
		wantCalls    int
	}{
		{name: "before lookup", cancelBefore: true},
		{name: "after canonical success", wantCalls: 1},
		{name: "after canonical missing prevents fallback", missing: true, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelBefore {
				cancel()
			}
			calls := 0
			reader := attachedEditorLeaseLookupFunc(func(gotCtx context.Context, key string) (EditorLease, error) {
				calls++
				if gotCtx != ctx {
					t.Fatal("lookup did not receive caller context")
				}
				cancel()
				if tc.missing {
					return EditorLease{}, ErrNotFound
				}
				return EditorLease{SessionID: key}, nil
			})
			got, err := ResolveAttachedEditorLease(ctx, reader, "am-run", "host-session")
			if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(got, EditorLease{}) || calls != tc.wantCalls {
				t.Fatalf("lease=%+v error=%v calls=%d, want cancellation and %d calls", got, err, calls, tc.wantCalls)
			}
		})
	}
}

func TestEditorLeaseBindingEqualIdentityFields(t *testing.T) {
	created := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	lease := EditorLease{
		SessionID: "session", Harness: "claude-code", Agent: "claude", PID: 123,
		HostBootID: "boot", Scope: "vrooli-agent-session.scope", Status: EditorLeaseActive,
		CreatedAt: created, Owner: EditorLeaseOwner{Kind: "terminal", Ref: "tty:/dev/pts/1", Source: "launch", EvidenceJSON: `{"pid":123}`},
	}
	for _, tc := range []struct {
		name   string
		change func(*EditorLease)
	}{
		{name: "attempt binding", change: func(l *EditorLease) { l.BindingID = "new-attempt" }},
		{name: "session", change: func(l *EditorLease) { l.SessionID = "other" }},
		{name: "harness", change: func(l *EditorLease) { l.Harness = "codex" }},
		{name: "agent", change: func(l *EditorLease) { l.Agent = "other" }},
		{name: "pid", change: func(l *EditorLease) { l.PID++ }},
		{name: "boot", change: func(l *EditorLease) { l.HostBootID = "other" }},
		{name: "scope", change: func(l *EditorLease) { l.Scope = "other" }},
		{name: "status", change: func(l *EditorLease) { l.Status = EditorLeaseStopped }},
		{name: "created", change: func(l *EditorLease) { l.CreatedAt = l.CreatedAt.Add(time.Second) }},
		{name: "owner kind", change: func(l *EditorLease) { l.Owner.Kind = "web-console-session" }},
		{name: "owner ref", change: func(l *EditorLease) { l.Owner.Ref = "other" }},
		{name: "owner source", change: func(l *EditorLease) { l.Owner.Source = "backfill" }},
		{name: "owner evidence", change: func(l *EditorLease) { l.Owner.EvidenceJSON = `{"pid":124}` }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := lease
			tc.change(&other)
			if EditorLeaseBindingEqual(lease, other) || EditorLeaseBindingEqual(other, lease) {
				t.Fatal("different identity binding compared equal")
			}
		})
	}
	other := lease
	other.CreatedAt = lease.CreatedAt.In(time.FixedZone("equivalent", 60*60))
	other.LastHeartbeatAt = created.Add(time.Minute)
	other.HeartbeatDeadlineAt = created.Add(time.Hour)
	other.WorkingDir = "/another/path"
	other.ContainmentMethod = "systemd"
	other.Claims = []string{"some/path"}
	other.StoppedAt = &created
	other.StopReason = "advisory metadata"
	other.Owner.RecordedAt = &created
	if !EditorLeaseBindingEqual(lease, lease) || !EditorLeaseBindingEqual(lease, other) || !EditorLeaseBindingEqual(other, lease) {
		t.Fatal("equal binding was changed by time representation or non-binding fields")
	}
}

func TestEditorLeaseBindingEqualRejectsSamePIDAndCreationTimeRebind(t *testing.T) {
	store, _ := newEditorLeaseStore(t)
	ctx := context.Background()
	lease := EditorLease{SessionID: "same-session", PID: 123, HostBootID: "same-boot", Scope: "vrooli-agent-same"}
	begin := func(id string) EditorLease {
		t.Helper()
		if _, err := store.CreateEditorLeaseBinding(ctx, lease, EditorLeaseProvenance{BindingID: id, PID: 123, UID: 1000, HostBootID: "same-boot", ProcessStart: "1234", IntendedScope: lease.Scope}, time.Minute); err != nil {
			t.Fatal(err)
		}
		got, err := store.GetEditorLease(ctx, lease.SessionID)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	before, after := begin("first-attempt"), begin("replacement-attempt")
	if before.PID != after.PID || !before.CreatedAt.Equal(after.CreatedAt) || before.Scope != after.Scope {
		t.Fatal("fixture did not preserve session identity")
	}
	if EditorLeaseBindingEqual(before, after) || EditorLeaseBindingEqual(after, before) {
		t.Fatal("same-process replacement compared equal")
	}
}
