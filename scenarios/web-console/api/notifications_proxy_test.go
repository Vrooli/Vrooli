package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotificationsProxyForwardsCatalogGET(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/notifications/catalog" {
			t.Fatalf("method=%q path=%q", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"sources":[],"preferences":[]}`)
	}))
	defer upstream.Close()
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil })
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/catalog", nil)
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), "preferences") {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
}

func TestNotificationsProxyForwardsPUTBodyAndAuthorization(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/notifications/preferences" {
			t.Fatalf("method=%q path=%q", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer owner-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"source_scenario":"nutrition-planner"`) {
			t.Fatalf("body=%q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil })
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences",
		strings.NewReader(`{"source_scenario":"nutrition-planner","category":"meals","enabled":true}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer owner-token")
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), "ok") {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
}

func TestNotificationsProxyForwardsCloudflareAccessAssertion(t *testing.T) {
	var sawAccess, sawAccessWhenAbsent string
	present := true
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if present {
			sawAccess = r.Header.Get("Cf-Access-Jwt-Assertion")
		} else {
			sawAccessWhenAbsent = r.Header.Get("Cf-Access-Jwt-Assertion")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil })

	// With the assertion present, it is forwarded so the hub's human-proof path
	// attributes the answer to the actual signed-in human.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/answer", strings.NewReader(`{}`))
	req.Header.Set("Cf-Access-Jwt-Assertion", "eyJhbGciOiJSUzI1NiJ9.payload.sig")
	proxy.ServeHTTP(httptest.NewRecorder(), req)
	if sawAccess != "eyJhbGciOiJSUzI1NiJ9.payload.sig" {
		t.Fatalf("Cf-Access-Jwt-Assertion was not forwarded: %q", sawAccess)
	}

	// Without it (bearer/cookie path), the header is simply omitted.
	present = false
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/answer", strings.NewReader(`{}`))
	proxy.ServeHTTP(httptest.NewRecorder(), req2)
	if sawAccessWhenAbsent != "" {
		t.Fatalf("Cf-Access-Jwt-Assertion must be omitted when absent, got %q", sawAccessWhenAbsent)
	}
}

func TestNotificationsProxyUpstreamDownReturns503(t *testing.T) {
	proxy := newNotificationsProxy(func(context.Context) (string, error) {
		return "", errors.New("notification-hub not discovered")
	})
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/catalog", nil)
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
	if !strings.Contains(record.Body.String(), "unavailable") {
		t.Fatalf("expected JSON error, got %q", record.Body.String())
	}
}
