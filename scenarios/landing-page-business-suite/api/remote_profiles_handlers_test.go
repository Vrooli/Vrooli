package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestHandleAdminRemoteProfiles_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	svc := newRemoteProfileServiceForTest(db, nil)
	sessionMgr := initSessionManager()
	server := &Server{db: db, sessionManager: sessionMgr, remoteProfileService: svc}

	createReq := []byte(`{"tag":"prod","label":"Production","api_base":"https://example.com/api/v1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles", bytes.NewBuffer(createReq))
	attachAdminSession(t, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(handleAdminCreateRemoteProfile(server.remoteProfileService, server.sessionAdminEmail))(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/remote-profiles", nil)
	attachAdminSession(t, listReq, defaultAdminEmail)
	listResp := httptest.NewRecorder()

	server.requireAdmin(handleAdminListRemoteProfiles(server.remoteProfileService))(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listResp.Code, listResp.Body.String())
	}

	var payload struct {
		Profiles []RemoteProfile `json:"profiles"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(payload.Profiles))
	}
	if payload.Profiles[0].Tag != "prod" {
		t.Fatalf("expected tag prod, got %s", payload.Profiles[0].Tag)
	}
}

func TestHandleAdminRemoteProfiles_ListEmpty(t *testing.T) {
	handler := handleAdminListRemoteProfiles(remoteProfileManagerStub{
		listFn: func(_ context.Context) ([]RemoteProfile, error) {
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/remote-profiles", nil)
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var payload struct {
		Profiles []RemoteProfile `json:"profiles"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Profiles == nil {
		t.Fatalf("expected profiles to be non-nil")
	}
	if len(payload.Profiles) != 0 {
		t.Fatalf("expected empty profiles, got %d", len(payload.Profiles))
	}
}

func TestHandleAdminRemoteProfiles_CreateUsesResolver(t *testing.T) {
	handler := handleAdminCreateRemoteProfile(remoteProfileManagerStub{
		createFn: func(_ context.Context, req RemoteProfileCreateRequest, createdByEmail string) (*RemoteProfile, error) {
			if createdByEmail != "admin@example.com" {
				t.Fatalf("expected resolver email, got %q", createdByEmail)
			}
			return &RemoteProfile{
				ID:      1,
				Tag:     req.Tag,
				APIBase: req.APIBase,
				Status:  remoteProfileStatusUnknown,
			}, nil
		},
	}, func(_ *http.Request) (string, bool) {
		return "admin@example.com", true
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles",
		bytes.NewBufferString(`{"tag":"prod","label":"Production","api_base":"https://example.com/api/v1"}`))
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	var profile RemoteProfile
	if err := json.Unmarshal(resp.Body.Bytes(), &profile); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if profile.Tag != "prod" {
		t.Fatalf("expected tag prod, got %s", profile.Tag)
	}
}

func TestHandleAdminRemoteProfiles_CreateTagConflict(t *testing.T) {
	handler := handleAdminCreateRemoteProfile(remoteProfileManagerStub{
		createFn: func(_ context.Context, _ RemoteProfileCreateRequest, _ string) (*RemoteProfile, error) {
			return nil, ErrRemoteProfileTagExists
		},
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles",
		bytes.NewBufferString(`{"tag":"prod","label":"Production","api_base":"https://example.com/api/v1"}`))
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.Code)
	}
}

func TestHandleAdminRemoteProfiles_UpdateNotFound(t *testing.T) {
	handler := handleAdminUpdateRemoteProfile(remoteProfileManagerStub{
		updateFn: func(_ context.Context, _ int64, _ RemoteProfileUpdateRequest) (*RemoteProfile, error) {
			return nil, ErrRemoteProfileNotFound
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/remote-profiles/12",
		bytes.NewBufferString(`{"label":"Updated"}`))
	req = mux.SetURLVars(req, map[string]string{"id": "12"})
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}

func TestHandleAdminRemoteProfiles_DeleteSuccess(t *testing.T) {
	handler := handleAdminDeleteRemoteProfile(remoteProfileManagerStub{
		deleteFn: func(_ context.Context, id int64) error {
			if id != 12 {
				t.Fatalf("expected id 12, got %d", id)
			}
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/remote-profiles/12", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "12"})
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["success"] != true {
		t.Fatalf("expected success true, got %v", payload["success"])
	}
}

func TestHandleAdminRemoteProfileLogin_InvalidEmail(t *testing.T) {
	called := false
	handler := handleAdminRemoteProfileLogin(remoteProfileManagerStub{
		loginFn: func(_ context.Context, _ int64, _ string, _ string) (*RemoteProfile, error) {
			called = true
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles/22/login",
		bytes.NewBufferString(`{"email":"invalid","password":"secret"}`))
	req = mux.SetURLVars(req, map[string]string{"id": "22"})
	resp := httptest.NewRecorder()

	handler(resp, req)

	if called {
		t.Fatalf("expected login not to be called for invalid email")
	}
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestHandleAdminRemoteProfileLogin_RemoteError(t *testing.T) {
	handler := handleAdminRemoteProfileLogin(remoteProfileManagerStub{
		loginFn: func(_ context.Context, _ int64, _ string, _ string) (*RemoteProfile, error) {
			return nil, &RemoteProfileError{
				Status:    http.StatusUnauthorized,
				ErrorType: ApiErrorTypeUnauthorized,
				Message:   "unauthorized",
			}
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles/22/login",
		bytes.NewBufferString(`{"email":"admin@example.com","password":"secret"}`))
	req = mux.SetURLVars(req, map[string]string{"id": "22"})
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestHandleAdminRemoteProfileLogout_Success(t *testing.T) {
	handler := handleAdminRemoteProfileLogout(remoteProfileManagerStub{
		logoutFn: func(_ context.Context, id int64) (*RemoteProfile, error) {
			if id != 33 {
				t.Fatalf("expected id 33, got %d", id)
			}
			return &RemoteProfile{ID: id, Tag: "prod", APIBase: "https://example.com/api/v1"}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles/33/logout", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "33"})
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHandleAdminRemoteProfileTest_RemoteError(t *testing.T) {
	handler := handleAdminRemoteProfileTest(remoteProfileManagerStub{
		testFn: func(_ context.Context, _ int64) (*RemoteProfile, error) {
			return nil, &RemoteProfileError{
				Status:    http.StatusUnauthorized,
				ErrorType: ApiErrorTypeUnauthorized,
				Message:   "expired",
			}
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles/44/test", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "44"})
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestHandleAdminRemoteProfileProxy_ContentType(t *testing.T) {
	handler := handleAdminRemoteProfileProxy(remoteProfileManagerStub{
		proxyFn: func(_ context.Context, id int64, req RemoteProfileProxyRequest) (*RemoteProxyResponse, error) {
			if id != 42 {
				t.Fatalf("expected id 42, got %d", id)
			}
			if req.Method != "GET" || req.Path != "/admin/download-storage" {
				t.Fatalf("unexpected proxy request: %#v", req)
			}
			return &RemoteProxyResponse{
				StatusCode:  http.StatusCreated,
				Body:        []byte("ok"),
				ContentType: "text/plain",
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles/42/proxy",
		bytes.NewBufferString(`{"method":"GET","path":"/admin/download-storage"}`))
	req = mux.SetURLVars(req, map[string]string{"id": "42"})
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "text/plain" {
		t.Fatalf("expected content-type text/plain, got %q", got)
	}
	if resp.Body.String() != "ok" {
		t.Fatalf("expected body ok, got %q", resp.Body.String())
	}
}

func TestHandleAdminRemoteProfileProxy_DefaultContentType(t *testing.T) {
	handler := handleAdminRemoteProfileProxy(remoteProfileManagerStub{
		proxyFn: func(_ context.Context, id int64, _ RemoteProfileProxyRequest) (*RemoteProxyResponse, error) {
			if id != 7 {
				t.Fatalf("expected id 7, got %d", id)
			}
			return &RemoteProxyResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(`{"ok":true}`),
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles/7/proxy",
		bytes.NewBufferString(`{"method":"GET","path":"/admin/download-storage"}`))
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	resp := httptest.NewRecorder()

	handler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", got)
	}
}

func TestWriteRemoteProfileErrorMapping(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		status    int
		errorType string
	}{
		{
			name:      "not-found",
			err:       ErrRemoteProfileNotFound,
			status:    http.StatusNotFound,
			errorType: ApiErrorTypeNotFound,
		},
		{
			name:      "tag-exists",
			err:       ErrRemoteProfileTagExists,
			status:    http.StatusConflict,
			errorType: ApiErrorTypeValidation,
		},
		{
			name:      "session-missing",
			err:       ErrRemoteProfileSessionMissing,
			status:    http.StatusConflict,
			errorType: ApiErrorTypeValidation,
		},
		{
			name:      "disallowed-path",
			err:       ErrRemoteProfileDisallowedPath,
			status:    http.StatusForbidden,
			errorType: ApiErrorTypeForbidden,
		},
		{
			name:      "invalid",
			err:       ErrRemoteProfileInvalid,
			status:    http.StatusBadRequest,
			errorType: ApiErrorTypeValidation,
		},
		{
			name: "remote-profile-error",
			err: &RemoteProfileError{
				Status:    http.StatusUnauthorized,
				ErrorType: ApiErrorTypeUnauthorized,
				Message:   "unauthorized",
			},
			status:    http.StatusUnauthorized,
			errorType: ApiErrorTypeUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handled := writeRemoteProfileError(rec, tt.err)
			if !handled {
				t.Fatalf("expected error to be handled")
			}
			if rec.Code != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, rec.Code)
			}
			var payload ApiErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if payload.ErrorType != tt.errorType {
				t.Fatalf("expected error_type %q, got %q", tt.errorType, payload.ErrorType)
			}
		})
	}
}

type remoteProfileManagerStub struct {
	listFn   func(context.Context) ([]RemoteProfile, error)
	createFn func(context.Context, RemoteProfileCreateRequest, string) (*RemoteProfile, error)
	updateFn func(context.Context, int64, RemoteProfileUpdateRequest) (*RemoteProfile, error)
	deleteFn func(context.Context, int64) error
	loginFn  func(context.Context, int64, string, string) (*RemoteProfile, error)
	logoutFn func(context.Context, int64) (*RemoteProfile, error)
	testFn   func(context.Context, int64) (*RemoteProfile, error)
	proxyFn  func(context.Context, int64, RemoteProfileProxyRequest) (*RemoteProxyResponse, error)
}

func (s remoteProfileManagerStub) List(ctx context.Context) ([]RemoteProfile, error) {
	if s.listFn != nil {
		return s.listFn(ctx)
	}
	return nil, nil
}

func (s remoteProfileManagerStub) Create(ctx context.Context, req RemoteProfileCreateRequest, createdByEmail string) (*RemoteProfile, error) {
	if s.createFn != nil {
		return s.createFn(ctx, req, createdByEmail)
	}
	return nil, nil
}

func (s remoteProfileManagerStub) Update(ctx context.Context, id int64, req RemoteProfileUpdateRequest) (*RemoteProfile, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, id, req)
	}
	return nil, nil
}

func (s remoteProfileManagerStub) Delete(ctx context.Context, id int64) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func (s remoteProfileManagerStub) Login(ctx context.Context, id int64, email string, password string) (*RemoteProfile, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, id, email, password)
	}
	return nil, nil
}

func (s remoteProfileManagerStub) Logout(ctx context.Context, id int64) (*RemoteProfile, error) {
	if s.logoutFn != nil {
		return s.logoutFn(ctx, id)
	}
	return nil, nil
}

func (s remoteProfileManagerStub) Test(ctx context.Context, id int64) (*RemoteProfile, error) {
	if s.testFn != nil {
		return s.testFn(ctx, id)
	}
	return nil, nil
}

func (s remoteProfileManagerStub) Proxy(ctx context.Context, id int64, req RemoteProfileProxyRequest) (*RemoteProxyResponse, error) {
	if s.proxyFn != nil {
		return s.proxyFn(ctx, id, req)
	}
	return nil, nil
}
