package policygate

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	_ "modernc.org/sqlite"
)

func TestMemoryIntentServiceBindsExactSubjectAndConsumesOnce(t *testing.T) {
	now := time.Date(2026, 9, 6, 20, 0, 0, 0, time.UTC)
	service := NewIntentService(NewMemoryIntentStore()).WithClock(func() time.Time { return now })
	principal := Principal{Kind: cliutil.CallerKindHuman, Subject: "operator-1", Verified: true}
	intent, err := service.Issue(context.Background(), principal, IntentRequest{RepositoryID: "repo-1", Operation: "repo.commit", ExpectedRevision: "head-1", SubjectDigest: "subject-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Consume(context.Background(), principal, intent.ID, "repo-1", "repo.commit", "head-1", "changed-subject"); !errors.Is(err, ErrIntentMismatch) {
		t.Fatalf("changed subject error=%v, want mismatch", err)
	}
	if _, err := service.Consume(context.Background(), principal, intent.ID, "repo-1", "repo.commit", "head-1", "subject-1"); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if _, err := service.Consume(context.Background(), principal, intent.ID, "repo-1", "repo.commit", "head-1", "subject-1"); !errors.Is(err, ErrIntentReplay) {
		t.Fatalf("replay error=%v, want replay", err)
	}
}

func TestMemoryIntentServiceConcurrentConsumeHasOneWinner(t *testing.T) {
	service := NewIntentService(NewMemoryIntentStore())
	principal := Principal{Kind: cliutil.CallerKindHuman, Subject: "operator-1", Verified: true}
	intent, err := service.Issue(context.Background(), principal, IntentRequest{RepositoryID: "repo-1", Operation: "repo.commit", ExpectedRevision: "head-1", SubjectDigest: "subject-1"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners, replays := 0, 0
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.Consume(context.Background(), principal, intent.ID, "repo-1", "repo.commit", "head-1", "subject-1")
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				winners++
			} else if errors.Is(err, ErrIntentReplay) {
				replays++
			}
		}()
	}
	wg.Wait()
	if winners != 1 || replays != 11 {
		t.Fatalf("winners=%d replays=%d, want 1/11", winners, replays)
	}
}

func TestSQLIntentStorePersistsSafeReceiptAndAtomicConsume(t *testing.T) {
	db, err := sql.Open("sqlite", "file:gct-intents?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE git_mutation_intents (
		intent_hash TEXT PRIMARY KEY, principal_id TEXT NOT NULL, repository_id TEXT NOT NULL,
		operation TEXT NOT NULL, expected_revision TEXT NOT NULL, subject_digest TEXT NOT NULL,
		policy_version TEXT NOT NULL, issued_at TEXT NOT NULL, expires_at TEXT NOT NULL,
		consumed_at TEXT, step_up_required INTEGER NOT NULL DEFAULT 0)`)
	if err != nil {
		t.Fatal(err)
	}
	service := NewIntentService(NewSQLIntentStore(db))
	principal := Principal{Kind: cliutil.CallerKindHuman, Subject: "operator-1", Verified: true}
	intent, err := service.Issue(context.Background(), principal, IntentRequest{RepositoryID: "repo-1", Operation: "repo.commit", ExpectedRevision: "head-1", SubjectDigest: "subject-1"})
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := db.QueryRow(`SELECT intent_hash FROM git_mutation_intents`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == intent.ID || stored == "" {
		t.Fatalf("raw intent was persisted: %q", stored)
	}
	consumed, err := service.Consume(context.Background(), principal, intent.ID, "repo-1", "repo.commit", "head-1", "subject-1")
	if err != nil {
		t.Fatal(err)
	}
	if consumed.ID != intent.ID {
		t.Fatalf("consumed intent ID=%q, want %q", consumed.ID, intent.ID)
	}
	if _, ok := ConsumedIntentFromContext(WithIntent(WithPrincipal(context.Background(), principal), consumed.AsHumanIntent())); !ok {
		t.Fatal("SQL-consumed intent should remain visible to the domain writer guard")
	}
	if _, err := service.Consume(context.Background(), principal, intent.ID, "repo-1", "repo.commit", "head-1", "subject-1"); !errors.Is(err, ErrIntentReplay) {
		t.Fatalf("replay error=%v", err)
	}
}

func TestIntentServiceRejectsAgentIssuance(t *testing.T) {
	service := NewIntentService(NewMemoryIntentStore())
	_, err := service.Issue(context.Background(), Principal{Kind: cliutil.CallerKindVrooliAgent, Subject: "run-1", Verified: true}, IntentRequest{RepositoryID: "repo-1", Operation: "repo.commit", ExpectedRevision: "head-1", SubjectDigest: "subject-1"})
	if !errors.Is(err, ErrMutationPermission) {
		t.Fatalf("error=%v, want permission refusal", err)
	}
}
