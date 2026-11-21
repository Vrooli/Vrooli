package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/sessions"
	admin "landing-page-business-suite-api/internal/administration"
)

func TestRequireAdminStepUpRejectsStaleProof(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	session := sessions.NewSession(nil, "admin_session")
	session.Values["email"], session.Values["session_id"] = "admin@example.test", "session-1"
	manager := NewMockSessionManager()
	manager.Sessions["admin_session"] = session
	s := &Server{db: db, adminAuthService: admin.NewAdminAuthService(db), sessionManager: manager}
	query := regexp.QuoteMeta("SELECT expires_at, last_activity, assurance, reauthenticated_at")
	row := sqlmock.NewRows([]string{"expires_at", "last_activity", "assurance", "reauthenticated_at"}).AddRow(time.Now().Add(time.Hour), time.Now(), "full", sql.NullTime{})
	mock.ExpectQuery(query).WillReturnRows(row)
	row = sqlmock.NewRows([]string{"expires_at", "last_activity", "assurance", "reauthenticated_at"}).AddRow(time.Now().Add(time.Hour), time.Now(), "full", time.Now().Add(-11*time.Minute))
	mock.ExpectQuery(query).WillReturnRows(row)
	recorder := httptest.NewRecorder()
	s.requireAdminStepUp(func(http.ResponseWriter, *http.Request) {}).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/admin/stripe/settings", nil))
	if recorder.Code != http.StatusForbidden || recorder.Header().Get("X-Lpbs-Auth-Reason") != "admin_reauthentication_required" {
		t.Fatalf("status=%d headers=%v", recorder.Code, recorder.Header())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
