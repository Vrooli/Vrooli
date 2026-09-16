package administration

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	admin "landing-page-business-suite-api/internal/administration"
)

type memoryThrottle struct{ counts map[string]int }

func (m *memoryThrottle) Exceeded(_ context.Context, bucket string, rule admin.ThrottleRule) (bool, error) {
	return m.counts[bucket] >= rule.Limit, nil
}

func (m *memoryThrottle) Record(_ context.Context, bucket string) error {
	m.counts[bucket]++
	return nil
}

func (m *memoryThrottle) Reset(_ context.Context, bucket string) error {
	delete(m.counts, bucket)
	return nil
}

type fakeSecondFactor struct {
	enabled bool
	valid   string
}

func (f fakeSecondFactor) Enabled(context.Context, string) (bool, error) { return f.enabled, nil }

func (f fakeSecondFactor) Verify(_ context.Context, _ string, code string) error {
	if code != f.valid {
		return admin.ErrMFAInvalid
	}
	return nil
}

type missingAdmin struct{ *fakeAuth }

func (missingAdmin) PasswordHash(context.Context, string) (string, error) { return "", sql.ErrNoRows }

func securityDeps(t *testing.T, auth AuthService) (Dependencies, *memoryThrottle) {
	t.Helper()
	manager := &fakeSessions{session: sessions.NewSession(nil, sessionName)}
	manager.session.Values = map[any]any{}
	deps := testDependencies(auth, manager)
	throttle := &memoryThrottle{counts: map[string]int{}}
	deps.Throttle = throttle
	return deps, throttle
}

func login(deps Dependencies, request LoginRequest) *SessionError {
	_, err := LoginSession(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder(), request, deps)
	return err
}

func hashFor(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
}

func TestAdminLoginLocksAfterRepeatedFailuresEvenWithCorrectPassword(t *testing.T) {
	deps, _ := securityDeps(t, &fakeAuth{hash: hashFor(t, "correct-password")})
	for attempt := 0; attempt < admin.AdminFailuresPerUser.Limit; attempt++ {
		if err := login(deps, LoginRequest{Email: "admin@example.test", Password: "wrong"}); err == nil || err.Status != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %+v", attempt, err)
		}
	}
	err := login(deps, LoginRequest{Email: "admin@example.test", Password: "correct-password"})
	if err == nil || err.Status != http.StatusTooManyRequests || err.Kind != LoginKindLocked {
		t.Fatalf("locked login = %+v", err)
	}
}

func TestAdminLoginUnknownEmailCountsTowardLockout(t *testing.T) {
	deps, throttle := securityDeps(t, missingAdmin{&fakeAuth{}})
	if err := login(deps, LoginRequest{Email: "ghost@example.test", Password: "guess"}); err == nil || err.Status != http.StatusUnauthorized {
		t.Fatalf("unknown admin = %+v", err)
	}
	if throttle.counts[admin.ThrottleBucket("admin-login-email", "ghost@example.test")] != 1 {
		t.Fatalf("failure not recorded: %#v", throttle.counts)
	}
}

func TestAdminLoginSuccessClearsUserFailures(t *testing.T) {
	deps, throttle := securityDeps(t, &fakeAuth{hash: hashFor(t, "correct-password")})
	_ = login(deps, LoginRequest{Email: "admin@example.test", Password: "wrong"})
	if err := login(deps, LoginRequest{Email: "admin@example.test", Password: "correct-password"}); err != nil {
		t.Fatal(err)
	}
	if throttle.counts[admin.ThrottleBucket("admin-login-email", "admin@example.test")] != 0 {
		t.Fatalf("user bucket not reset: %#v", throttle.counts)
	}
}

func TestAdminLoginRequiresSecondFactorWhenEnabled(t *testing.T) {
	deps, throttle := securityDeps(t, &fakeAuth{hash: hashFor(t, "correct-password")})
	deps.MFA = fakeSecondFactor{enabled: true, valid: "123456"}

	err := login(deps, LoginRequest{Email: "admin@example.test", Password: "correct-password"})
	if err == nil || err.Kind != LoginKindMFARequired || err.Status != http.StatusPreconditionRequired {
		t.Fatalf("missing code = %+v", err)
	}
	if len(throttle.counts) != 0 {
		t.Fatalf("a correct password awaiting its code must not count as a failure: %#v", throttle.counts)
	}
	err = login(deps, LoginRequest{Email: "admin@example.test", Password: "correct-password", TOTPCode: "000000"})
	if err == nil || err.Kind != LoginKindMFAInvalid {
		t.Fatalf("wrong code = %+v", err)
	}
	if err := login(deps, LoginRequest{Email: "admin@example.test", Password: "correct-password", TOTPCode: "123456"}); err != nil {
		t.Fatalf("valid code = %+v", err)
	}
}

func TestAdminLoginWrongPasswordNeverRevealsSecondFactor(t *testing.T) {
	deps, _ := securityDeps(t, &fakeAuth{hash: hashFor(t, "correct-password")})
	deps.MFA = fakeSecondFactor{enabled: true, valid: "123456"}
	err := login(deps, LoginRequest{Email: "admin@example.test", Password: "wrong"})
	if err == nil || err.Kind == LoginKindMFARequired {
		t.Fatalf("wrong password leaked MFA state: %+v", err)
	}
}

func TestConnectLoginKeepsThePeerAddressForPerClientThrottling(t *testing.T) {
	request, _ := connectHTTP(context.Background(), http.Header{}, "198.51.100.7:52110")
	if request.RemoteAddr != "198.51.100.7:52110" {
		t.Fatalf("RemoteAddr = %q; Connect logins would share one throttle bucket", request.RemoteAddr)
	}
}

func TestAdminLoginWithoutClientIPDoesNotUseASharedBucket(t *testing.T) {
	deps, throttle := securityDeps(t, &fakeAuth{hash: hashFor(t, "correct-password")})
	deps.ClientIP = func(*http.Request) string { return "" }
	_ = login(deps, LoginRequest{Email: "admin@example.test", Password: "wrong"})
	for bucket := range throttle.counts {
		if bucket == admin.ThrottleBucket("admin-login-ip", "") {
			t.Fatalf("recorded a failure in the shared empty-IP bucket: %#v", throttle.counts)
		}
	}
}
