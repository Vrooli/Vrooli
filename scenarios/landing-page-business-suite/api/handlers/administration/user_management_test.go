package administration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	admin "landing-page-business-suite-api/internal/administration"
)

func userManagementTestDeps(service UserManagementService) UserManagementDependencies {
	return UserManagementDependencies{Service: service, Path: func(r *http.Request, k string) (string, bool) { v := mux.Vars(r)[k]; return v, v != "" }, WriteJSON: func(w http.ResponseWriter, v any) { _ = json.NewEncoder(w).Encode(v) }, WriteError: func(w http.ResponseWriter, status int, message string) { http.Error(w, message, status) }, Log: func(string, map[string]any) {}, LogError: func(string, map[string]any) {}}
}

func TestUserManagementTransportValidatesIdentifiersAndWritesSuccess(t *testing.T) {
	deps := userManagementTestDeps(userManagementStub{revoke: func(_ context.Context, user, session string) (bool, error) {
		if user != "u" || session != "s" {
			t.Fatalf("unexpected ids")
		}
		return true, nil
	}, revokeAll: func(context.Context, string) (int64, error) { return 2, nil }})
	missing := httptest.NewRecorder()
	GetUser(deps)(missing, httptest.NewRequest(http.MethodGet, "/", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("got %d", missing.Code)
	}
	revoke := httptest.NewRecorder()
	RevokeUserSession(deps)(revoke, mux.SetURLVars(httptest.NewRequest(http.MethodDelete, "/", nil), map[string]string{"id": "u", "sid": "s"}))
	if revoke.Code != http.StatusOK {
		t.Fatalf("got %d", revoke.Code)
	}
	all := httptest.NewRecorder()
	RevokeAllUserSessions(deps)(all, mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/", nil), map[string]string{"id": "u"}))
	if all.Code != http.StatusOK {
		t.Fatalf("got %d", all.Code)
	}
}

type userManagementStub struct {
	revoke    func(context.Context, string, string) (bool, error)
	revokeAll func(context.Context, string) (int64, error)
}

func (s userManagementStub) List(context.Context, string, int, int) (admin.UsersListResponse, error) {
	return admin.UsersListResponse{}, nil
}

func (s userManagementStub) Get(context.Context, string) (*admin.UserAccountResponse, error) {
	return &admin.UserAccountResponse{}, nil
}

func (s userManagementStub) ListSessions(context.Context, string) ([]admin.UserSessionResponse, error) {
	return nil, nil
}

func (s userManagementStub) RevokeSession(c context.Context, u, id string) (bool, error) {
	return s.revoke(c, u, id)
}

func (s userManagementStub) RevokeAllSessions(c context.Context, u string) (int64, error) {
	return s.revokeAll(c, u)
}
