package administration

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/sessions"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"golang.org/x/crypto/bcrypt"
)

func TestSessionConnectLoginPropagatesCookieAndSessionID(t *testing.T) {
	service := &fakeAuth{hash: connectTestPasswordHash(t)}
	manager := &headerSessions{fakeSessions: &fakeSessions{session: connectTestSession()}}
	handler := NewSessionConnectHandler(testDependencies(service, manager))

	response, err := handler.Login(context.Background(), connect.NewRequest(&lpbsv1.LoginRequest{Email: "admin@example.test", Password: "correct-password"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetSessionId() != "session-1" || response.Header().Get("Set-Cookie") == "" {
		t.Fatalf("response=%#v headers=%v", response.Msg, response.Header())
	}
}

func TestSessionConnectRejectsUnauthenticatedSession(t *testing.T) {
	manager := &headerSessions{fakeSessions: &fakeSessions{session: connectTestSession()}}
	_, err := NewSessionConnectHandler(testDependencies(&fakeAuth{}, manager)).Session(context.Background(), connect.NewRequest(&lpbsv1.SessionRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code=%v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestSessionConnectReauthenticateRequiresPasswordAndSecondFactor(t *testing.T) {
	auth := &fakeAuth{hash: connectTestPasswordHash(t)}
	session := connectTestSession()
	session.Values["email"] = "admin@example.test"
	session.Values["session_id"] = "session-1"
	manager := &headerSessions{fakeSessions: &fakeSessions{session: session}}
	deps := testDependencies(auth, manager)
	deps.MFA = fakeSecondFactor{enabled: true, valid: "123456"}
	handler := NewSessionConnectHandler(deps)

	response, err := handler.Reauthenticate(context.Background(), connect.NewRequest(&lpbsv1.ReauthenticateRequest{Password: "correct-password", TotpCode: "123456"}))
	if err != nil || response == nil || !response.Msg.GetReauthenticated() || !auth.marked {
		t.Fatalf("response=%v err=%v marked=%v", response, err, auth.marked)
	}
}

func TestSessionConnectReauthenticateRejectsWrongFactor(t *testing.T) {
	auth := &fakeAuth{hash: connectTestPasswordHash(t)}
	session := connectTestSession()
	session.Values["email"] = "admin@example.test"
	session.Values["session_id"] = "session-1"
	deps := testDependencies(auth, &headerSessions{fakeSessions: &fakeSessions{session: session}})
	deps.MFA = fakeSecondFactor{enabled: true, valid: "123456"}
	_, err := NewSessionConnectHandler(deps).Reauthenticate(context.Background(), connect.NewRequest(&lpbsv1.ReauthenticateRequest{Password: "correct-password", TotpCode: "000000"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated || auth.marked {
		t.Fatalf("err=%v marked=%v", err, auth.marked)
	}
}

func TestResetConnectHandlerDoesNotLeakResetFailure(t *testing.T) {
	handler := NewResetConnectHandler(ResetDependencies{Reset: func(context.Context) error { return errors.New("database credentials") }, LogError: func(string, map[string]any) {}, Now: func() time.Time { return time.Time{} }})
	_, err := handler.ResetDemoData(context.Background(), connect.NewRequest(&lpbsv1.ResetDemoDataRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal || err == nil || strings.Contains(err.Error(), "credentials") {
		t.Fatalf("error=%v", err)
	}
}

type headerSessions struct{ *fakeSessions }

func (s *headerSessions) SaveSession(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	w.Header().Add("Set-Cookie", "admin_session=opaque; Path=/; HttpOnly")
	return s.fakeSessions.SaveSession(r, w, session)
}

func connectTestSession() *sessions.Session {
	session := sessions.NewSession(nil, sessionName)
	session.Values = map[any]any{}
	return session
}

func connectTestPasswordHash(t *testing.T) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
}
