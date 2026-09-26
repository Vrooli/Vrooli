package accountsecurity

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListSessionsScopesToUserAndMasksDeviceData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	created := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lastUsed := created.Add(time.Hour)
	expires := created.Add(24 * time.Hour)
	absolute := created.Add(90 * 24 * time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, created_at, last_used_at, expires_at, absolute_expires_at, auth_method, ip_address::text, user_agent")).
		WithArgs("user-a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "last_used_at", "expires_at", "absolute_expires_at", "auth_method", "ip_address", "user_agent"}).
			AddRow("session-a", created, lastUsed, expires, absolute, "email_code", "203.0.113.42", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/140.0"))

	service := NewService(Options{Store: db, Now: func() time.Time { return created }})
	sessions, err := service.ListSessions(t.Context(), "user-a", "session-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || !sessions[0].GetCurrent() || sessions[0].GetIpHint() != "203.0.113.x" || sessions[0].GetDeviceLabel() != "Chrome on macOS" {
		t.Fatalf("unexpected session projection: %+v", sessions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRevokeSessionCannotCrossUserBoundary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_sessions SET revoked = TRUE, revoked_at = NOW(), revoked_reason = 'user'")).
		WithArgs("session-b", "user-a").
		WillReturnResult(sqlmock.NewResult(0, 0))

	service := NewService(Options{Store: db})
	revoked, err := service.RevokeSession(t.Context(), "user-a", "session-b")
	if err != nil {
		t.Fatal(err)
	}
	if revoked {
		t.Fatal("cross-user session was revoked")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMaskIP(t *testing.T) {
	if got := maskIP("2001:db8:abcd:1234::1"); got != "2001:db8:abcd:*" {
		t.Fatalf("masked IPv6 = %q", got)
	}
	if got := maskIP("not-an-ip"); got != "unknown" {
		t.Fatalf("invalid IP = %q", got)
	}
}
