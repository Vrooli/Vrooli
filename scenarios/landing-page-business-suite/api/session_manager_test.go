package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCookieSessionManagerAcceptsPreviousSecretDuringRotation(t *testing.T) {
	oldManager := NewCookieSessionManager("01234567890123456789012345678901")
	request := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	writer := httptest.NewRecorder()
	session, err := oldManager.GetSession(request, "lpbs-session")
	if err != nil {
		t.Fatalf("get old session: %v", err)
	}
	session.Values["user_id"] = int64(42)
	if err := oldManager.SaveSession(request, writer, session); err != nil {
		t.Fatalf("save old session: %v", err)
	}

	rotatedManager := NewCookieSessionManager("12345678901234567890123456789012", "01234567890123456789012345678901")
	rotatedRequest := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	rotatedRequest.Header.Set("Cookie", strings.Split(writer.Header().Get("Set-Cookie"), ";")[0])
	rotatedSession, err := rotatedManager.GetSession(rotatedRequest, "lpbs-session")
	if err != nil {
		t.Fatalf("read session with previous secret: %v", err)
	}
	if got := rotatedSession.Values["user_id"]; got != int64(42) {
		t.Fatalf("previous session value = %#v, want int64(42)", got)
	}
}

func TestCookieSessionManagerDoesNotAcceptPreviousSecretWithoutOverlap(t *testing.T) {
	oldManager := NewCookieSessionManager("01234567890123456789012345678901")
	request := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	writer := httptest.NewRecorder()
	session, err := oldManager.GetSession(request, "lpbs-session")
	if err != nil {
		t.Fatalf("get old session: %v", err)
	}
	session.Values["user_id"] = int64(42)
	if err := oldManager.SaveSession(request, writer, session); err != nil {
		t.Fatalf("save old session: %v", err)
	}

	rotatedManager := NewCookieSessionManager("12345678901234567890123456789012")
	rotatedRequest := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	rotatedRequest.Header.Set("Cookie", strings.Split(writer.Header().Get("Set-Cookie"), ";")[0])
	if _, err := rotatedManager.GetSession(rotatedRequest, "lpbs-session"); err == nil {
		t.Fatal("old session was accepted after previous-key overlap was removed")
	}
}
