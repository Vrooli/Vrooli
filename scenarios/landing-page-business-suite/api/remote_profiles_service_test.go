package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/administration"
)

func newRemoteProfileServiceForTest(db *sql.DB, client administration.HTTPDoer) *RemoteProfileService {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &RemoteProfileService{
		db:            db,
		encryptionKey: nil,
		httpClient:    client,
		now:           time.Now,
	}
}

func TestRemoteProfileService_CreateAndList(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

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

func TestRemoteProfileService_ListHandlesNullConnectorID(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	svc := newRemoteProfileServiceForTest(db, nil)
	ctx := context.Background()

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO remote_profiles (tag, label, api_base, connector_id, status, created_at, updated_at)
		VALUES ('prod', 'Production', 'https://example.com/api/v1', NULL, $1, NOW(), NOW())
	`, remoteProfileStatusUnknown); err != nil {
		t.Fatalf("failed to seed legacy remote profile: %v", err)
	}

	profiles, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if strings.TrimSpace(profiles[0].ConnectorID) == "" {
		t.Fatalf("expected connector_id to be backfilled")
	}

	var stored sql.NullString
	if err := db.QueryRow(`SELECT connector_id FROM remote_profiles WHERE tag = 'prod'`).Scan(&stored); err != nil {
		t.Fatalf("query connector id: %v", err)
	}
	if !stored.Valid || strings.TrimSpace(stored.String) == "" {
		t.Fatalf("expected persisted connector_id, got %v", stored)
	}
}

func TestRemoteProfileService_LoginAndProxy(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

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

	if err := svc.setSession(ctx, profile.ID, "session-123", "", nil); err != nil {
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

func TestRemoteProfileService_CreateTagConflict(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	svc := newRemoteProfileServiceForTest(db, nil)
	ctx := context.Background()

	_, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err = svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Duplicate",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if !errors.Is(err, ErrRemoteProfileTagExists) {
		t.Fatalf("expected ErrRemoteProfileTagExists, got %v", err)
	}
}

func TestRemoteProfileService_LoginInvalidInput(t *testing.T) {
	svc := &RemoteProfileService{}
	_, err := svc.Login(context.Background(), 1, " ", "")
	if !errors.Is(err, ErrRemoteProfileInvalid) {
		t.Fatalf("expected ErrRemoteProfileInvalid, got %v", err)
	}
}

func TestRemoteProfileService_UpdateClearsSessionOnAPIBaseChange(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	svc := newRemoteProfileServiceForTest(db, nil)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := svc.setSession(ctx, profile.ID, "session-123", "", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	newBase := "http://example.net/api/v1"
	newLabel := "Updated"
	updated, err := svc.Update(ctx, profile.ID, RemoteProfileUpdateRequest{
		APIBase: &newBase,
		Label:   &newLabel,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.APIBase != newBase {
		t.Fatalf("expected api_base %q, got %q", newBase, updated.APIBase)
	}
	if updated.Status != remoteProfileStatusUnknown {
		t.Fatalf("expected status unknown, got %s", updated.Status)
	}
	if updated.HasSession {
		t.Fatalf("expected session to be cleared after api_base change")
	}
}

func TestRemoteProfileService_UpdateTagConflict(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	svc := newRemoteProfileServiceForTest(db, nil)
	ctx := context.Background()

	_, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	other, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "staging",
		Label:   "Staging",
		APIBase: "http://staging.example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	conflict := "prod"
	_, err = svc.Update(ctx, other.ID, RemoteProfileUpdateRequest{Tag: &conflict})
	if !errors.Is(err, ErrRemoteProfileTagExists) {
		t.Fatalf("expected tag exists error, got %v", err)
	}
}

func TestRemoteProfileService_DeleteNotFound(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	svc := newRemoteProfileServiceForTest(db, nil)
	err := svc.Delete(context.Background(), 99999)
	if !errors.Is(err, ErrRemoteProfileNotFound) {
		t.Fatalf("expected ErrRemoteProfileNotFound, got %v", err)
	}
}

func TestRemoteProfileService_LogoutClearsSession(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/logout" {
				t.Fatalf("unexpected logout path: %s", req.URL.Path)
			}
			return newHTTPResponse(http.StatusOK, `{}`, nil, "application/json"), nil
		},
	}
	svc := newRemoteProfileServiceForTest(db, client)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := svc.setSession(ctx, profile.ID, "session-123", "", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	updated, err := svc.Logout(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if updated.HasSession {
		t.Fatalf("expected session cleared after logout")
	}
	if updated.Status != remoteProfileStatusExpired {
		t.Fatalf("expected status expired, got %s", updated.Status)
	}
}

func TestRemoteProfileService_RemoteLogoutUnauthorizedNoError(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/logout" {
				t.Fatalf("unexpected logout path: %s", req.URL.Path)
			}
			return newHTTPResponse(http.StatusUnauthorized, "unauthorized", nil, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{httpClient: client}

	if err := svc.remoteLogout(context.Background(), "http://example.com/api/v1", "session-abc"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRemoteProfileService_RemoteLogoutServerError(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/logout" {
				t.Fatalf("unexpected logout path: %s", req.URL.Path)
			}
			return newHTTPResponse(http.StatusInternalServerError, "boom", nil, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{httpClient: client}

	err := svc.remoteLogout(context.Background(), "http://example.com/api/v1", "session-abc")
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", remoteErr.Status)
	}
}

func TestRemoteProfileService_ProxyUnauthorizedClearsSession(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	client := stubHTTPClient{
		do: func(_ *http.Request) (*http.Response, error) {
			return newHTTPResponse(http.StatusUnauthorized, "", nil, ""), nil
		},
	}
	svc := newRemoteProfileServiceForTest(db, client)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := svc.setSession(ctx, profile.ID, "session-123", "", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	_, err = svc.Proxy(ctx, profile.ID, RemoteProfileProxyRequest{
		Method: "GET",
		Path:   "/admin/download-storage",
	})
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}

	updated, err := svc.GetByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if updated.Status != remoteProfileStatusExpired {
		t.Fatalf("expected status expired, got %s", updated.Status)
	}
	if updated.HasSession {
		t.Fatalf("expected session cleared after unauthorized proxy")
	}
}

func TestRemoteProfileService_ProxyServerErrorUpdatesStatus(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	client := stubHTTPClient{
		do: func(_ *http.Request) (*http.Response, error) {
			return newHTTPResponse(http.StatusInternalServerError, "boom", nil, "text/plain"), nil
		},
	}
	svc := newRemoteProfileServiceForTest(db, client)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := svc.setSession(ctx, profile.ID, "session-123", "", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	_, err = svc.Proxy(ctx, profile.ID, RemoteProfileProxyRequest{
		Method: "GET",
		Path:   "/admin/download-storage",
	})
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}

	updated, err := svc.GetByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if updated.Status != remoteProfileStatusError {
		t.Fatalf("expected status error, got %s", updated.Status)
	}
}

func TestRemoteProfileService_ProxyMissingSession(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	svc := newRemoteProfileServiceForTest(db, nil)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err = svc.Proxy(ctx, profile.ID, RemoteProfileProxyRequest{
		Method: "GET",
		Path:   "/admin/download-storage",
	})
	if !errors.Is(err, ErrRemoteProfileSessionMissing) {
		t.Fatalf("expected ErrRemoteProfileSessionMissing, got %v", err)
	}
}

func TestRemoteProfileService_SessionLinks(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/remote-profile-sessions" {
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
			body := `{"sessions":[{"session_id":"remote-session-1","admin_email":"admin@localhost","connector_id":"connector-abc","profile_tag":"prod","origin":"local","created_at":"2025-01-01T00:00:00Z","last_activity":"2025-01-01T01:00:00Z","expires_at":"2025-01-01T02:00:00Z"}]}`
			return newHTTPResponse(http.StatusOK, body, nil, "application/json"), nil
		},
	}
	svc := newRemoteProfileServiceForTest(db, client)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := svc.setSession(ctx, profile.ID, "session-123", "remote-session-1", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}
	created, err := svc.GetByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if created.ConnectorID == "" {
		t.Fatalf("expected connector id to be set")
	}

	links, err := svc.SessionLinks(ctx, profile.ID)
	if err != nil {
		t.Fatalf("SessionLinks returned error: %v", err)
	}
	if len(links.RemoteSessions) != 1 {
		t.Fatalf("expected 1 remote session, got %d", len(links.RemoteSessions))
	}
}

func TestRemoteProfileService_SessionLinksUnauthorizedClearsSession(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/admin/remote-profile-sessions" {
				return newHTTPResponse(http.StatusUnauthorized, `{"error":"expired"}`, nil, "application/json"), nil
			}
			return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
		},
	}
	svc := newRemoteProfileServiceForTest(db, client)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := svc.setSession(ctx, profile.ID, "session-123", "remote-session-1", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	_, err = svc.SessionLinks(ctx, profile.ID)
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", remoteErr.Status)
	}

	current, err := svc.GetByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if current.HasSession {
		t.Fatalf("expected local session to be cleared after unauthorized session-links check")
	}
	if current.Status != remoteProfileStatusExpired {
		t.Fatalf("expected expired status, got %q", current.Status)
	}
}

func TestRemoteProfileService_RevokeRemoteSessions(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && req.URL.Path == "/api/v1/admin/remote-profile-sessions":
				body := `{"sessions":[{"session_id":"remote-session-2","admin_email":"admin@localhost","connector_id":"connector-abc","profile_tag":"prod","origin":"local","created_at":"2025-01-01T00:00:00Z","last_activity":"2025-01-01T01:00:00Z","expires_at":"2025-01-01T02:00:00Z"}]}`
				return newHTTPResponse(http.StatusOK, body, nil, "application/json"), nil
			case req.Method == http.MethodDelete && req.URL.Path == "/api/v1/admin/remote-profile-sessions/remote-session-2":
				return newHTTPResponse(http.StatusOK, `{"success":true}`, nil, "application/json"), nil
			default:
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
		},
	}
	svc := newRemoteProfileServiceForTest(db, client)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := svc.setSession(ctx, profile.ID, "session-123", "remote-session-2", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	links, err := svc.RevokeRemoteSessions(ctx, profile.ID)
	if err != nil {
		t.Fatalf("RevokeRemoteSessions returned error: %v", err)
	}
	if links.LocalHasSession {
		t.Fatalf("expected local session to be cleared")
	}
}

func TestRemoteProfileService_RevokeRemoteSessions_MissingLocalSession(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	svc := newRemoteProfileServiceForTest(db, nil)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err = svc.RevokeRemoteSessions(ctx, profile.ID)
	if !errors.Is(err, ErrRemoteProfileSessionMissing) {
		t.Fatalf("expected ErrRemoteProfileSessionMissing, got %v", err)
	}
}

func TestRemoteProfileService_RevokeRemoteSessions_ListRemoteSessionsError(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet && req.URL.Path == "/api/v1/admin/remote-profile-sessions" {
				return newHTTPResponse(http.StatusInternalServerError, `{"error":"remote failed"}`, nil, "application/json"), nil
			}
			return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
		},
	}
	svc := newRemoteProfileServiceForTest(db, client)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := svc.setSession(ctx, profile.ID, "session-123", "remote-session-2", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	_, err = svc.RevokeRemoteSessions(ctx, profile.ID)
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", remoteErr.Status)
	}

	current, err := svc.GetByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if !current.HasSession {
		t.Fatalf("expected local session to remain when remote list fails")
	}
}

func TestRemoteProfileService_RevokeRemoteSessions_RevokeOneRemoteSessionFails(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && req.URL.Path == "/api/v1/admin/remote-profile-sessions":
				body := `{"sessions":[{"session_id":"remote-session-1","admin_email":"admin@localhost","connector_id":"connector-abc","profile_tag":"prod","origin":"local","created_at":"2025-01-01T00:00:00Z","last_activity":"2025-01-01T01:00:00Z","expires_at":"2025-01-01T02:00:00Z"},{"session_id":"remote-session-2","admin_email":"admin@localhost","connector_id":"connector-abc","profile_tag":"prod","origin":"local","created_at":"2025-01-01T00:00:00Z","last_activity":"2025-01-01T01:00:00Z","expires_at":"2025-01-01T02:00:00Z"}]}`
				return newHTTPResponse(http.StatusOK, body, nil, "application/json"), nil
			case req.Method == http.MethodDelete && req.URL.Path == "/api/v1/admin/remote-profile-sessions/remote-session-1":
				return newHTTPResponse(http.StatusOK, `{"success":true}`, nil, "application/json"), nil
			case req.Method == http.MethodDelete && req.URL.Path == "/api/v1/admin/remote-profile-sessions/remote-session-2":
				return newHTTPResponse(http.StatusInternalServerError, `{"error":"revoke failed"}`, nil, "application/json"), nil
			default:
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
		},
	}
	svc := newRemoteProfileServiceForTest(db, client)
	ctx := context.Background()

	profile, err := svc.Create(ctx, RemoteProfileCreateRequest{
		Tag:     "prod",
		Label:   "Production",
		APIBase: "http://example.com/api/v1",
	}, defaultAdminEmail)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := svc.setSession(ctx, profile.ID, "session-123", "remote-session-2", nil); err != nil {
		t.Fatalf("setSession returned error: %v", err)
	}

	_, err = svc.RevokeRemoteSessions(ctx, profile.ID)
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", remoteErr.Status)
	}

	current, err := svc.GetByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if !current.HasSession {
		t.Fatalf("expected local session to remain when one remote revoke fails")
	}
}

func TestRemoteProfileService_EnsureConnectorID_ReturnsTrimmedCurrent(t *testing.T) {
	svc := &RemoteProfileService{}
	connectorID, err := svc.ensureConnectorID(context.Background(), 1, "  connector-abc  ")
	if err != nil {
		t.Fatalf("ensureConnectorID returned error: %v", err)
	}
	if connectorID != "connector-abc" {
		t.Fatalf("expected trimmed connector id, got %q", connectorID)
	}
}

func TestRemoteProfileService_EnsureConnectorID_GeneratesAndPersistsWhenMissing(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	db := setupTestDB(t)

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

	if _, err := db.Exec(`UPDATE remote_profiles SET connector_id = '' WHERE id = $1`, profile.ID); err != nil {
		t.Fatalf("failed to clear connector id: %v", err)
	}

	connectorID, err := svc.ensureConnectorID(ctx, profile.ID, "")
	if err != nil {
		t.Fatalf("ensureConnectorID returned error: %v", err)
	}
	if connectorID == "" {
		t.Fatalf("expected generated connector id")
	}

	var stored string
	if err := db.QueryRow(`SELECT connector_id FROM remote_profiles WHERE id = $1`, profile.ID).Scan(&stored); err != nil {
		t.Fatalf("query connector id: %v", err)
	}
	if strings.TrimSpace(stored) == "" {
		t.Fatalf("expected persisted connector id, got %q", stored)
	}
}

func TestRemoteProfileService_ProxyInvalidMethod(t *testing.T) {
	svc := &RemoteProfileService{}
	_, err := svc.Proxy(context.Background(), 1, RemoteProfileProxyRequest{
		Method: "TRACE",
		Path:   "/admin/download-storage",
	})
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", remoteErr.Status)
	}
}

func TestRemoteProfileService_ProxyMissingMethod(t *testing.T) {
	svc := &RemoteProfileService{}
	_, err := svc.Proxy(context.Background(), 1, RemoteProfileProxyRequest{
		Method: "",
		Path:   "/admin/download-storage",
	})
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", remoteErr.Status)
	}
}

func TestRemoteProfileService_RemoteLoginHappyPath(t *testing.T) {
	fixedNow := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/admin/login":
				cookie := &http.Cookie{Name: remoteProfileCookieName, Value: "session-abc", MaxAge: 3600}
				body := `{"authenticated":true,"reset_enabled":true}`
				return newHTTPResponse(http.StatusOK, body, []*http.Cookie{cookie}, "application/json"), nil
			case "/api/v1/admin/session":
				body := `{"authenticated":true,"reset_enabled":true}`
				return newHTTPResponse(http.StatusOK, body, nil, "application/json"), nil
			default:
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
		},
	}
	svc := &RemoteProfileService{
		httpClient: client,
		now:        func() time.Time { return fixedNow },
	}

	session, remoteSessionID, expiresAt, err := svc.remoteLogin(context.Background(), "http://example.com/api/v1", "admin@example.com", "password", administration.RemoteProfileSessionMetadata{ConnectorID: "connector-test"})
	if err != nil {
		t.Fatalf("remoteLogin returned error: %v", err)
	}
	if session != "session-abc" {
		t.Fatalf("expected session-abc, got %q", session)
	}
	if remoteSessionID != "" {
		t.Fatalf("expected empty remote session id when not provided, got %q", remoteSessionID)
	}
	if expiresAt == nil || !expiresAt.Equal(fixedNow.Add(time.Hour)) {
		t.Fatalf("expected expiry %v, got %v", fixedNow.Add(time.Hour), expiresAt)
	}
}

func TestRemoteProfileService_RemoteLoginMissingCookie(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/login" {
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
			body := `{"authenticated":true,"reset_enabled":true}`
			return newHTTPResponse(http.StatusOK, body, nil, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{
		httpClient: client,
		now:        time.Now,
	}

	_, _, _, err := svc.remoteLogin(context.Background(), "http://example.com/api/v1", "admin@example.com", "password", administration.RemoteProfileSessionMetadata{ConnectorID: "connector-test"})
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", remoteErr.Status)
	}
}

func TestRemoteProfileService_RemoteLoginNotAuthenticated(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/login" {
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
			cookie := &http.Cookie{Name: remoteProfileCookieName, Value: "session-abc"}
			body := `{"authenticated":false}`
			return newHTTPResponse(http.StatusOK, body, []*http.Cookie{cookie}, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{httpClient: client, now: time.Now}

	_, _, _, err := svc.remoteLogin(context.Background(), "http://example.com/api/v1", "admin@example.com", "password", administration.RemoteProfileSessionMetadata{ConnectorID: "connector-test"})
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", remoteErr.Status)
	}
}

func TestRemoteProfileService_RemoteLoginSessionVerificationFails(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/admin/login":
				cookie := &http.Cookie{Name: remoteProfileCookieName, Value: "session-abc"}
				body := `{"authenticated":true}`
				return newHTTPResponse(http.StatusOK, body, []*http.Cookie{cookie}, "application/json"), nil
			case "/api/v1/admin/session":
				body := `{"authenticated":false}`
				return newHTTPResponse(http.StatusOK, body, nil, "application/json"), nil
			default:
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
		},
	}
	svc := &RemoteProfileService{httpClient: client, now: time.Now}

	_, _, _, err := svc.remoteLogin(context.Background(), "http://example.com/api/v1", "admin@example.com", "password", administration.RemoteProfileSessionMetadata{ConnectorID: "connector-test"})
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", remoteErr.Status)
	}
}

func TestRemoteProfileService_RemoteLoginInvalidJSON(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/login" {
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
			cookie := &http.Cookie{Name: remoteProfileCookieName, Value: "session-abc"}
			return newHTTPResponse(http.StatusOK, "{invalid", []*http.Cookie{cookie}, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{httpClient: client, now: time.Now}

	_, _, _, err := svc.remoteLogin(context.Background(), "http://example.com/api/v1", "admin@example.com", "password", administration.RemoteProfileSessionMetadata{ConnectorID: "connector-test"})
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestRemoteProfileService_RemoteSessionCheckUnauthorized(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/session" {
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
			return newHTTPResponse(http.StatusUnauthorized, `{"authenticated":false}`, nil, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{httpClient: client, now: time.Now}

	authenticated, err := svc.remoteSessionCheck(context.Background(), "http://example.com/api/v1", "session-abc")
	if err != nil {
		t.Fatalf("remoteSessionCheck returned error: %v", err)
	}
	if authenticated {
		t.Fatalf("expected authenticated=false")
	}
}

func TestRemoteProfileService_RemoteSessionCheckServerError(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/session" {
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
			return newHTTPResponse(http.StatusInternalServerError, `{"error":"boom"}`, nil, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{httpClient: client, now: time.Now}

	_, err := svc.remoteSessionCheck(context.Background(), "http://example.com/api/v1", "session-abc")
	var remoteErr *RemoteProfileError
	if !errors.As(err, &remoteErr) {
		t.Fatalf("expected RemoteProfileError, got %v", err)
	}
	if remoteErr.Status != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", remoteErr.Status)
	}
	if remoteErr.ErrorType != ApiErrorTypeServerError {
		t.Fatalf("expected server_error, got %s", remoteErr.ErrorType)
	}
}

func TestRemoteProfileService_RemoteSessionCheckInvalidJSON(t *testing.T) {
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/admin/session" {
				return newHTTPResponse(http.StatusNotFound, "missing", nil, "text/plain"), nil
			}
			return newHTTPResponse(http.StatusOK, "{invalid", nil, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{httpClient: client, now: time.Now}

	_, err := svc.remoteSessionCheck(context.Background(), "http://example.com/api/v1", "session-abc")
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestBuildRemoteURL(t *testing.T) {
	svc := &RemoteProfileService{}
	urlValue, err := svc.buildRemoteURL("https://example.com/api/v1", "/admin/download-storage", map[string]string{
		"platform": "windows",
		"":         "ignored",
	})
	if err != nil {
		t.Fatalf("buildRemoteURL returned error: %v", err)
	}
	if !strings.HasPrefix(urlValue, "https://example.com/api/v1/admin/download-storage") {
		t.Fatalf("unexpected base url: %s", urlValue)
	}
	if !strings.Contains(urlValue, "platform=windows") {
		t.Fatalf("expected query string, got %s", urlValue)
	}
}

func TestRemoteProfileRecord_ToProfile_ExpiredSession(t *testing.T) {
	now := time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC)
	rec := &remoteProfileRecord{
		ID:               1,
		Tag:              "prod",
		APIBase:          "https://example.com/api/v1",
		Status:           remoteProfileStatusActive,
		EncryptedSession: sql.NullString{Valid: true, String: "session"},
		SessionExpiresAt: sql.NullTime{Valid: true, Time: now.Add(-1 * time.Hour)},
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}

	profile := rec.toProfile(now)
	if profile.Status != remoteProfileStatusExpired {
		t.Fatalf("expected status expired, got %s", profile.Status)
	}
	if !profile.HasSession {
		t.Fatalf("expected HasSession true when encrypted session present")
	}
}

func TestRemoteProfileRecord_ToProfile_NotExpired(t *testing.T) {
	now := time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC)
	rec := &remoteProfileRecord{
		ID:               2,
		Tag:              "staging",
		APIBase:          "https://example.com/api/v1",
		Status:           remoteProfileStatusActive,
		EncryptedSession: sql.NullString{Valid: true, String: "session"},
		SessionExpiresAt: sql.NullTime{Valid: true, Time: now.Add(1 * time.Hour)},
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}

	profile := rec.toProfile(now)
	if profile.Status != remoteProfileStatusActive {
		t.Fatalf("expected status active, got %s", profile.Status)
	}
}

func TestClassifyRemoteErrorTimeout(t *testing.T) {
	err := classifyRemoteError(timeoutError{})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if err.Status != http.StatusGatewayTimeout {
		t.Fatalf("expected status 504, got %d", err.Status)
	}
	if err.ErrorType != ApiErrorTypeTimeout {
		t.Fatalf("expected timeout error_type, got %s", err.ErrorType)
	}
}

func TestClassifyRemoteErrorDeadline(t *testing.T) {
	err := classifyRemoteError(context.DeadlineExceeded)
	if err == nil {
		t.Fatalf("expected error")
	}
	if err.Status != http.StatusGatewayTimeout {
		t.Fatalf("expected status 504, got %d", err.Status)
	}
}

func TestExtractRemoteErrorMessage(t *testing.T) {
	msg := extractRemoteErrorMessage([]byte(`{"error":"bad request"}`))
	if msg != "bad request" {
		t.Fatalf("expected json error message, got %q", msg)
	}
	msg = extractRemoteErrorMessage([]byte("plain text"))
	if msg != "plain text" {
		t.Fatalf("expected plain text, got %q", msg)
	}
	msg = extractRemoteErrorMessage([]byte(" "))
	if msg != "Remote request failed" {
		t.Fatalf("expected default message, got %q", msg)
	}
}

func TestMapRemoteStatus(t *testing.T) {
	if got := mapRemoteStatus(http.StatusInternalServerError); got != http.StatusBadGateway {
		t.Fatalf("expected 502 for 500, got %d", got)
	}
	if got := mapRemoteStatus(http.StatusNotFound); got != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", got)
	}
}

func TestNormalizeRemoteProfileLabel(t *testing.T) {
	if normalizeRemoteProfileLabel("  ") != nil {
		t.Fatalf("expected nil for empty label")
	}
	label := normalizeRemoteProfileLabel("  Hello ")
	if label == nil || *label != "Hello" {
		t.Fatalf("expected trimmed label, got %v", label)
	}
}

func TestNormalizeRemoteProfileAPIBaseStripsQueryAndFragment(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	got, err := normalizeRemoteProfileAPIBase("https://example.com/api/v1?x=1#section")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/api/v1" {
		t.Fatalf("expected base without query/fragment, got %q", got)
	}
}

func TestIsAllowedRemoteProxyPath(t *testing.T) {
	if !isAllowedRemoteProxyPath("/admin/download-storage") {
		t.Fatalf("expected allowlisted path")
	}
	if !isAllowedRemoteProxyPath("/admin/download-storage/subpath") {
		t.Fatalf("expected allowlisted prefix")
	}
	if isAllowedRemoteProxyPath("/admin/users") {
		t.Fatalf("expected disallowed path")
	}
}

func TestReadLimitedBody(t *testing.T) {
	data, err := readLimitedBody(strings.NewReader("hello"), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected hello, got %q", string(data))
	}
	if _, err := readLimitedBody(strings.NewReader("toolong"), 3); err == nil {
		t.Fatalf("expected error for exceeding limit")
	}
}

func TestDoJSONRequestSetsHeadersAndCookies(t *testing.T) {
	var gotAccept, gotContentType, gotCookie string
	client := stubHTTPClient{
		do: func(req *http.Request) (*http.Response, error) {
			gotAccept = req.Header.Get("Accept")
			gotContentType = req.Header.Get("Content-Type")
			if cookie, err := req.Cookie(remoteProfileCookieName); err == nil {
				gotCookie = cookie.Value
			}
			return newHTTPResponse(http.StatusOK, `{}`, nil, "application/json"), nil
		},
	}
	svc := &RemoteProfileService{httpClient: client}

	_, _, err := svc.doJSONRequest(context.Background(), http.MethodPost, "http://example.com/api/v1/admin/login",
		[]byte(`{"email":"admin@example.com"}`), []*http.Cookie{{Name: remoteProfileCookieName, Value: "session-xyz"}})
	if err != nil {
		t.Fatalf("doJSONRequest returned error: %v", err)
	}
	if gotAccept != "application/json" {
		t.Fatalf("expected Accept application/json, got %q", gotAccept)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", gotContentType)
	}
	if gotCookie != "session-xyz" {
		t.Fatalf("expected session cookie, got %q", gotCookie)
	}
}

type stubHTTPClient struct {
	do func(*http.Request) (*http.Response, error)
}

func (s stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if s.do == nil {
		return nil, errors.New("stubHTTPClient do func not set")
	}
	return s.do(req)
}

func newHTTPResponse(status int, body string, cookies []*http.Cookie, contentType string) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	for _, cookie := range cookies {
		if cookie != nil {
			header.Add("Set-Cookie", cookie.String())
		}
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type timeoutError struct{}

func (timeoutError) Error() string { return "timeout" }
func (timeoutError) Timeout() bool { return true }
func (timeoutError) Temporary() bool {
	return true
}
