package emaildelivery

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMailerEnqueueCommitsOneDurableRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO email_outbox")).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = (Mailer{DB: db}).Enqueue(context.Background(), Message{
		DedupeKey: "signin:event-1", Purpose: PurposeSignIn, Recipient: "person@example.net", Sender: "noreply@example.com",
		Subject: "Sign in", TextBody: "code 123456", TemplateRef: "auth.sign-in",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMailerEnqueueRollsBackWhenOutboxWriteFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO email_outbox")).WillReturnError(errors.New("outbox unavailable"))
	mock.ExpectRollback()

	err = (Mailer{DB: db}).Enqueue(context.Background(), Message{
		DedupeKey: "security:event-1", Purpose: PurposeSecurity, Recipient: "person@example.net", Sender: "noreply@example.com",
		Subject: "Security", TextBody: "alert", TemplateRef: "security.alert",
	})
	if err == nil {
		t.Fatal("expected enqueue failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
