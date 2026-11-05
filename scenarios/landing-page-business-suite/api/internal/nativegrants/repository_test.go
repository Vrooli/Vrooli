package nativegrants

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestIssueHashesCodeAndConsumeBurnsIt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO native_auth_grants")).
		WithArgs(hashCode("native-code"), "user-1", "challenge", "http://127.0.0.1:43111/callback", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE native_auth_grants")).
		WithArgs(hashCode("native-code")).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "code_challenge", "redirect_uri", "authenticated_at", "sign_in_ip", "sign_in_user_agent"}).AddRow("user-1", "challenge", "http://127.0.0.1:43111/callback", time.Now(), "127.0.0.1", "UA"))

	repo := NewRepository(db)
	if err := repo.Issue(t.Context(), "native-code", "user-1", "challenge", "http://127.0.0.1:43111/callback", "binding", "127.0.0.1", "UA", time.Now(), time.Minute); err != nil {
		t.Fatal(err)
	}
	grant, err := repo.Consume(t.Context(), "native-code")
	if err != nil || grant.UserID != "user-1" {
		t.Fatalf("grant=%+v err=%v", grant, err)
	}
	if hashCode("native-code") == "native-code" {
		t.Fatal("stored code was not hashed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConsumeReplayRevokesAttachedSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE native_auth_grants")).WithArgs(hashCode("used-code")).WillReturnRows(sqlmock.NewRows([]string{"x"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT session_id FROM native_auth_grants")).WithArgs(hashCode("used-code")).WillReturnRows(sqlmock.NewRows([]string{"session_id"}).AddRow("session-1"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_sessions SET revoked = TRUE")).WithArgs("session-1").WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := NewRepository(db).Consume(t.Context(), "used-code"); err != ErrRejected {
		t.Fatalf("err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
