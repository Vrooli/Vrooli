package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"web-console/backends/codex"
	"web-console/internal/sessionstore"
)

func TestNativeRecoveryPostReopenFailureDoesNotReplay(t *testing.T) {
	for _, stage := range []string{"mark live", "registration"} {
		t.Run(stage, func(t *testing.T) {
			db := setupTestDB(t)
			if _, err := db.Exec(codexRecoveryIntentSchema); err != nil {
				t.Fatal(err)
			}
			store := sessionstore.NewSQL(db)
			meta := sessionstore.Metadata{ID: "session", Status: sessionstore.StatusLive,
				LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeThreadID: "exact-thread",
				LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server"}`}
			if err := store.Save(t.Context(), meta); err != nil {
				t.Fatal(err)
			}
			calls := 0
			resume := func(ctx context.Context, _, ownerID, _, _, threadID, _ string, _ []string, _ []string) (*codex.ManagedOwner, error) {
				calls++
				var phase, thread, boot, owner string
				err := db.QueryRowContext(ctx, `SELECT phase,thread_id,boot_id,owner_id FROM codex_recovery_intents WHERE session_id=?`, meta.ID).
					Scan(&phase, &thread, &boot, &owner)
				if err != nil || phase != "claimed" || thread != meta.NativeThreadID || boot != bootA || owner != ownerID {
					t.Fatalf("provider called before exact durable claim: %s/%s/%s/%s %v", phase, thread, boot, owner, err)
				}
				return codex.NewManagedOwner(codex.NewClient(testWriteCloser{io.Discard}, strings.NewReader("")), ownerID, threadID)
			}
			var faultStore sessionstore.Store = store
			if stage == "mark live" {
				faultStore = nativeRecoveryFallibleStore{Store: store, liveErr: errors.New("fixture MarkLive lost after provider reopen")}
			}
			first := newManagedCodexRegistry(faultStore)
			first.intents, first.resume = &codexRecoveryIntents{db: db, boot: bootA}, resume
			if stage == "registration" {
				first.register = func(context.Context, string, *codex.ManagedOwner, string, string, string) error {
					return errors.New("fixture registration lost after provider reopen")
				}
			}
			if err := first.Recover(t.Context()); err != nil {
				t.Fatal(err)
			}
			got, err := store.Get(t.Context(), meta.ID)
			if err != nil || calls != 1 || got.ControlMode != sessionstore.ControlModeRecoveryOnly {
				t.Fatalf("unexpected first attempt: calls=%d metadata=%+v err=%v", calls, got, err)
			}
			for _, boot := range []string{bootA, bootB} {
				retry := newManagedCodexRegistry(store)
				retry.intents, retry.resume = &codexRecoveryIntents{db: db, boot: boot}, resume
				if err := retry.Recover(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 1 {
				t.Fatalf("ambiguous attempt repeated provider reopen: %d", calls)
			}
		})
	}
}

func TestNativeRecoveryIntentOnlyReplacesConfirmedOwnerAcrossVerifiedBoot(t *testing.T) {
	ctx := t.Context()
	db := setupTestDB(t)
	if _, err := db.Exec(codexRecoveryIntentSchema); err != nil {
		t.Fatal(err)
	}
	store := sessionstore.NewSQL(db)
	if err := store.Save(ctx, sessionstore.Metadata{ID: "session", Status: sessionstore.StatusLive,
		LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeThreadID: "thread"}); err != nil {
		t.Fatal(err)
	}
	first := &codexRecoveryIntents{db: db, boot: bootA}
	next := &codexRecoveryIntents{db: db, boot: bootB}
	if err := first.claim(ctx, "session", "stale-thread", "stale"); !errors.Is(err, errCodexReopenUnconfirmed) {
		t.Fatalf("wrong thread received claim: %v", err)
	}
	if err := first.claim(ctx, "session", "thread", "owner"); err != nil {
		t.Fatal(err)
	}
	if err := first.confirm(ctx, "session", "thread", "owner"); err != nil {
		t.Fatal(err)
	}
	if err := first.claim(ctx, "session", "thread", "duplicate"); !errors.Is(err, errCodexReopenUnconfirmed) {
		t.Fatalf("same-boot duplicate was admitted: %v", err)
	}
	if err := next.claim(ctx, "session", "thread", "new-boot-owner"); err != nil {
		t.Fatal(err)
	}
	if err := next.confirm(ctx, "session", "thread", "new-boot-owner"); err != nil {
		t.Fatal(err)
	}
	if err := next.release(ctx, "session", "new-boot-owner"); err != nil {
		t.Fatal(err)
	}
	if err := next.claim(ctx, "session", "thread", "after-clean-close"); err != nil {
		t.Fatal(err)
	}
}
