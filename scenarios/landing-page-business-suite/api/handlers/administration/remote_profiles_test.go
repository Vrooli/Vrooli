package administration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	admin "landing-page-business-suite-api/internal/administration"
)

func transportDependencies(service RemoteProfileManager) RemoteProfileDependencies {
	writeError := func(w http.ResponseWriter, status int, message, kind string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "error_type": kind})
	}
	return RemoteProfileDependencies{
		Service: service,
		DecodeJSON: func(w http.ResponseWriter, r *http.Request, out any) bool {
			if err := json.NewDecoder(r.Body).Decode(out); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body", "validation")
				return false
			}
			return true
		},
		PathInt64: func(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
			id, err := strconv.ParseInt(mux.Vars(r)[key], 10, 64)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid "+key, "validation")
				return 0, false
			}
			return id, true
		},
		ValidateEmail: func(w http.ResponseWriter, email string) (string, bool) {
			if email == "admin@example.com" {
				return email, true
			}
			writeError(w, http.StatusBadRequest, "Invalid email format", "validation")
			return "", false
		},
		WriteData:   func(w http.ResponseWriter, value any) { _ = json.NewEncoder(w).Encode(value) },
		WriteSimple: func(w http.ResponseWriter) { _ = json.NewEncoder(w).Encode(map[string]bool{"success": true}) },
		WriteError:  writeError,
		LogError:    func(string, map[string]any) {},
	}
}

func withID(request *http.Request, id string) *http.Request {
	return mux.SetURLVars(request, map[string]string{"id": id})
}

func TestListRemoteProfilesReturnsEmptyArray(t *testing.T) {
	handler := ListRemoteProfiles(transportDependencies(remoteProfileStub{list: func(context.Context) ([]admin.RemoteProfile, error) { return nil, nil }}))
	response := httptest.NewRecorder()
	handler(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || response.Body.String() != "{\"profiles\":[]}\n" {
		t.Fatalf("unexpected list response: %d %s", response.Code, response.Body.String())
	}
}

func TestCreateRemoteProfilePassesResolvedEmailAndMapsDomainError(t *testing.T) {
	deps := transportDependencies(remoteProfileStub{create: func(_ context.Context, request admin.RemoteProfileCreateRequest, email string) (*admin.RemoteProfile, error) {
		if request.Tag != "prod" || email != "admin@example.com" {
			t.Fatalf("unexpected create input: %#v %q", request, email)
		}
		return nil, admin.ErrRemoteProfileTagExists
	}})
	deps.ResolveEmail = func(*http.Request) (string, bool) { return "admin@example.com", true }
	response := httptest.NewRecorder()
	CreateRemoteProfile(deps)(response, httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"tag":"prod"}`)))
	if response.Code != http.StatusConflict {
		t.Fatalf("expected conflict, got %d: %s", response.Code, response.Body.String())
	}
}

func TestUpdateAndLoginValidateInputs(t *testing.T) {
	update := UpdateRemoteProfile(transportDependencies(remoteProfileStub{update: func(context.Context, int64, admin.RemoteProfileUpdateRequest) (*admin.RemoteProfile, error) {
		return nil, admin.ErrRemoteProfileNotFound
	}}))
	updateResponse := httptest.NewRecorder()
	update(updateResponse, withID(httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"label":"new"}`)), "7"))
	if updateResponse.Code != http.StatusNotFound {
		t.Fatalf("expected not found, got %d", updateResponse.Code)
	}
	called := false
	login := LoginRemoteProfile(transportDependencies(remoteProfileStub{login: func(context.Context, int64, string, string) (*admin.RemoteProfile, error) {
		called = true
		return nil, nil
	}}))
	loginResponse := httptest.NewRecorder()
	login(loginResponse, withID(httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"invalid"}`)), "7"))
	if called || loginResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid email invoked service=%t status=%d", called, loginResponse.Code)
	}
}

func TestIDOperationsMapRemoteAndUnexpectedErrors(t *testing.T) {
	remote := TestRemoteProfile(transportDependencies(remoteProfileStub{test: func(context.Context, int64) (*admin.RemoteProfile, error) {
		return nil, &admin.RemoteProfileError{Status: http.StatusUnauthorized, ErrorType: "unauthorized", Message: "expired"}
	}}))
	remoteResponse := httptest.NewRecorder()
	remote(remoteResponse, withID(httptest.NewRequest(http.MethodPost, "/", nil), "5"))
	if remoteResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected remote error, got %d", remoteResponse.Code)
	}
	links := RemoteProfileSessionLinks(transportDependencies(remoteProfileStub{sessionLinks: func(context.Context, int64) (*admin.RemoteProfileSessionLinks, error) {
		return nil, errors.New("db down")
	}}))
	linksResponse := httptest.NewRecorder()
	links(linksResponse, withID(httptest.NewRequest(http.MethodGet, "/", nil), "5"))
	if linksResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected server error, got %d", linksResponse.Code)
	}
}

func TestProxyRemoteProfilePreservesResponseAndRejectsInvalidID(t *testing.T) {
	proxy := ProxyRemoteProfile(transportDependencies(remoteProfileStub{proxy: func(_ context.Context, id int64, request admin.RemoteProfileProxyRequest) (*admin.RemoteProxyResponse, error) {
		if id != 42 || request.Path != "/admin/download-storage" {
			t.Fatalf("unexpected proxy input: %d %#v", id, request)
		}
		return &admin.RemoteProxyResponse{StatusCode: http.StatusCreated, ContentType: "text/plain", Body: []byte("ok")}, nil
	}}))
	response := httptest.NewRecorder()
	proxy(response, withID(httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"method":"GET","path":"/admin/download-storage"}`)), "42"))
	if response.Code != http.StatusCreated || response.Header().Get("Content-Type") != "text/plain" || response.Body.String() != "ok" {
		t.Fatalf("unexpected proxy response: %d %q %q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	invalid := httptest.NewRecorder()
	proxy(invalid, withID(httptest.NewRequest(http.MethodPost, "/", nil), "bad"))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid ID failure, got %d", invalid.Code)
	}
}

type remoteProfileStub struct {
	list         func(context.Context) ([]admin.RemoteProfile, error)
	create       func(context.Context, admin.RemoteProfileCreateRequest, string) (*admin.RemoteProfile, error)
	update       func(context.Context, int64, admin.RemoteProfileUpdateRequest) (*admin.RemoteProfile, error)
	login        func(context.Context, int64, string, string) (*admin.RemoteProfile, error)
	test         func(context.Context, int64) (*admin.RemoteProfile, error)
	sessionLinks func(context.Context, int64) (*admin.RemoteProfileSessionLinks, error)
	proxy        func(context.Context, int64, admin.RemoteProfileProxyRequest) (*admin.RemoteProxyResponse, error)
}

func (s remoteProfileStub) List(ctx context.Context) ([]admin.RemoteProfile, error) {
	if s.list != nil {
		return s.list(ctx)
	}
	return nil, nil
}

func (s remoteProfileStub) Create(ctx context.Context, r admin.RemoteProfileCreateRequest, email string) (*admin.RemoteProfile, error) {
	if s.create != nil {
		return s.create(ctx, r, email)
	}
	return nil, nil
}

func (s remoteProfileStub) Update(ctx context.Context, id int64, r admin.RemoteProfileUpdateRequest) (*admin.RemoteProfile, error) {
	if s.update != nil {
		return s.update(ctx, id, r)
	}
	return nil, nil
}
func (s remoteProfileStub) Delete(context.Context, int64) error { return nil }
func (s remoteProfileStub) Login(ctx context.Context, id int64, email, password string) (*admin.RemoteProfile, error) {
	if s.login != nil {
		return s.login(ctx, id, email, password)
	}
	return nil, nil
}

func (s remoteProfileStub) Logout(context.Context, int64) (*admin.RemoteProfile, error) {
	return nil, nil
}

func (s remoteProfileStub) Test(ctx context.Context, id int64) (*admin.RemoteProfile, error) {
	if s.test != nil {
		return s.test(ctx, id)
	}
	return nil, nil
}

func (s remoteProfileStub) SessionLinks(ctx context.Context, id int64) (*admin.RemoteProfileSessionLinks, error) {
	if s.sessionLinks != nil {
		return s.sessionLinks(ctx, id)
	}
	return nil, nil
}

func (s remoteProfileStub) RevokeRemoteSessions(context.Context, int64) (*admin.RemoteProfileSessionLinks, error) {
	return nil, nil
}

func (s remoteProfileStub) Proxy(ctx context.Context, id int64, r admin.RemoteProfileProxyRequest) (*admin.RemoteProxyResponse, error) {
	if s.proxy != nil {
		return s.proxy(ctx, id, r)
	}
	return nil, nil
}
