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
