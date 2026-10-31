package emaildelivery

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestOutcomeKindsKeepUnknownProviderBound(t *testing.T) {
	if OutcomeUnknown == OutcomePermanent {
		t.Fatal("unknown must remain distinct from permanent failure")
	}
}

func TestRetryDelayBacksOffAndCaps(t *testing.T) {
	if got := retryDelay(1); got != 5*time.Second {
		t.Fatalf("first retry delay = %s", got)
	}
	if got := retryDelay(4); got != 10*time.Minute {
		t.Fatalf("fourth retry delay = %s", got)
	}
	if got := retryDelay(99); got != 30*time.Minute {
		t.Fatalf("capped retry delay = %s", got)
	}
}

func TestWorkerDropsQueuedSuppressedRecipientBeforeRouting(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 9, 18, 3, 30, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE email_outbox SET status='expired'")).WithArgs(now).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE email_outbox o SET status='suppressed'")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, purpose, recipient, sender, subject, text_body, html_body, template_ref, template_data, idempotency_key, priority, attempts, provider_id, expires_at FROM email_outbox")).WillReturnRows(sqlmock.NewRows([]string{"id", "purpose", "recipient", "sender", "subject", "text_body", "html_body", "template_ref", "template_data", "idempotency_key", "priority", "attempts", "provider_id", "expires_at"}))
	mock.ExpectRollback()

	worker := Worker{DB: db, Now: func() time.Time { return now }}
	processed, err := worker.ProcessOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed {
		t.Fatal("suppressed queue cleanup should not route a message")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
