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
