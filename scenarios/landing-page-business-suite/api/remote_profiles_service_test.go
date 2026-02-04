package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newRemoteProfileServiceForTest(db *sql.DB, client HTTPDoer) *RemoteProfileService {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &RemoteProfileService{
		db:            db,
		encryptionKey: nil,
		httpClient:    client,
		dialects:      NewDialectHelper("postgres"),
	}
}

func TestRemoteProfileService_CreateAndList(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)
	defer db.Close()

	svc := newRemoteProfileServiceForTest(db, nil)
	ctx := context.Background()

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "https://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if profile.Tag != "prod" {
		t.Fatalf("expected tag prod, got %s", profile.Tag)
	}
	if profile.Status != remoteProfileStatusUnknown {
		t.Fatalf("expected status unknown, got %s", profile.Status)
	}
	if profile.HasSession {
		t.Fatalf("expected has_session false")
	}

	profiles, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
}

func TestRemoteProfileService_LoginAndProxy(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	var lastCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/admin/login":
			var req LoginRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:    remoteProfileCookieName,
				Value:   "session-123",
				Path:    "/",
				Expires: time.Now().Add(1 * time.Hour),
			})
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(LoginResponse{Email: req.Email, Authenticated: true, ResetEnabled: true})
		case "/api/v1/admin/session":
			cookie, _ := r.Cookie(remoteProfileCookieName)
			if cookie == nil || cookie.Value != "session-123" {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(LoginResponse{Authenticated: false, ResetEnabled: true})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(LoginResponse{Authenticated: true, ResetEnabled: true})
		case "/api/v1/admin/download-storage":
			cookie, _ := r.Cookie(remoteProfileCookieName)
			if cookie == nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			lastCookie = cookie.Value
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	svc := newRemoteProfileServiceForTest(db, srv.Client())
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: srv.URL + "/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	profile, err = svc.Login(ctx, profile.ID, defaultAdminEmail, "testpass")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if !profile.HasSession {
		t.Fatalf("expected profile to have session")
	}

	result, err := svc.Proxy(ctx, profile.ID, RemoteProfileProxyRequest{
		Method: "GET",
		Path:   "/admin/download-storage",
	})
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}
	if lastCookie != "session-123" {
		t.Fatalf("expected proxy to forward session cookie, got %s", lastCookie)
	}
}

func TestRemoteProfileService_ProxyDisallowedPath(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	svc := newRemoteProfileServiceForTest(db, nil)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "https://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err = svc.Proxy(ctx, profile.ID, RemoteProfileProxyRequest{
		Method: "GET",
		Path:   "/admin/users",
	})
	if !errors.Is(err, ErrRemoteProfileDisallowedPath) {
		t.Fatalf("expected disallowed path error, got %v", err)
	}
}

func TestNormalizeRemoteProfileTag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "trim-and-lowercase", input: " Prod_1 ", want: "prod_1"},
		{name: "max-length", input: "abc-123", want: "abc-123"},
		{name: "empty", input: " ", wantErr: true},
		{name: "invalid-char", input: "prod!", wantErr: true},
		{name: "invalid-start", input: "-bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeRemoteProfileTag(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestNormalizeRemoteProfileAPIBase(t *testing.T) {
	t.Run("development_allows_http", func(t *testing.T) {
		t.Setenv("LPBS_ENVIRONMENT", "development")
		got, err := normalizeRemoteProfileAPIBase("http://example.com/api/v1/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "http://example.com/api/v1" {
			t.Fatalf("expected trimmed api_base, got %q", got)
		}
	})

	t.Run("requires_api_v1", func(t *testing.T) {
		t.Setenv("LPBS_ENVIRONMENT", "development")
		if _, err := normalizeRemoteProfileAPIBase("https://example.com"); err == nil {
			t.Fatalf("expected error for missing /api/v1")
		}
	})

	t.Run("rejects_credentials", func(t *testing.T) {
		t.Setenv("LPBS_ENVIRONMENT", "development")
		if _, err := normalizeRemoteProfileAPIBase("https://user:pass@example.com/api/v1"); err == nil {
			t.Fatalf("expected error for credentials in api_base")
		}
	})

	t.Run("production_requires_https", func(t *testing.T) {
		t.Setenv("LPBS_ENVIRONMENT", "production")
		_, err := normalizeRemoteProfileAPIBase("http://example.com/api/v1")
		if err == nil || !strings.Contains(err.Error(), "https") {
			t.Fatalf("expected https requirement error, got %v", err)
		}
	})
}

func TestNormalizeRemoteProxyPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "allowlist-prefix", input: "/admin/download-storage", want: "/admin/download-storage"},
		{name: "trim-trailing-slash", input: "/admin/download-storage/", want: "/admin/download-storage"},
		{name: "admin-root", input: "/admin", want: "/admin"},
		{name: "missing-leading-slash", input: "admin/download-storage", wantErr: true},
		{name: "contains-query", input: "/admin/download-storage?x=1", wantErr: true},
		{name: "contains-parent", input: "/admin/../download-storage", wantErr: true},
		{name: "absolute-url", input: "https://example.com/admin/download-storage", wantErr: true},
		{name: "non-admin", input: "/public", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeRemoteProxyPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestRemoteProfileService_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	svc := &RemoteProfileService{encryptionKey: key}
	plaintext := "session-token-123"

	encrypted, err := svc.encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt returned error: %v", err)
	}
	if encrypted == plaintext {
		t.Fatalf("expected encrypted value to differ from plaintext")
	}

	decrypted, err := svc.decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt returned error: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestRemoteProfileService_EncryptDecrypt_NoKey(t *testing.T) {
	svc := &RemoteProfileService{encryptionKey: nil}
	plaintext := "session-token-plain"

	encrypted, err := svc.encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt returned error: %v", err)
	}
	if encrypted != plaintext {
		t.Fatalf("expected plaintext when encryption key is nil")
	}

	decrypted, err := svc.decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt returned error: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestRemoteProfileService_TestExpiredSessionClears(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/admin/session":
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(LoginResponse{Authenticated: false, ResetEnabled: true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	svc := newRemoteProfileServiceForTest(db, srv.Client())
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: srv.URL + "/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := svc.setSession(ctx, profile.ID, "session-123", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	_, err = svc.Test(ctx, profile.ID)
	if err == nil {
		t.Fatalf("expected expired session error")
	}
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) || remoteErr.Status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized RemoteProfileError, got %v", err)
	}

	updated, err := svc.GetByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if updated.Status != remoteProfileStatusExpired {
		t.Fatalf("expected status expired, got %s", updated.Status)
	}
	if updated.HasSession {
		t.Fatalf("expected session to be cleared")
	}
}
