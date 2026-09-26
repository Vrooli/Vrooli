package businessaccount

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domain "landing-page-business-suite-api/internal/businessaccount"

	"github.com/gorilla/mux"
)

func TestRoutesListAndCreateBusinessAccounts(t *testing.T) {
	repo := domain.NewMemoryRepository()
	router := mux.NewRouter()
	RegisterRoutes(router, Dependencies{
		Repository: repo,
		UserID:     func(context.Context) string { return "user-1" },
		UserEmail:  func(context.Context) string { return "one@example.com" },
	}, func(next http.HandlerFunc) http.HandlerFunc { return next })

	create := httptest.NewRequest(http.MethodPost, "/api/v1/business-accounts", strings.NewReader(`{"display_name":"Acme"}`))
	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", createResponse.Code, createResponse.Body.String())
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v1/business-accounts", nil)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, list)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d: %s", listResponse.Code, listResponse.Body.String())
	}
	var payload struct {
		Accounts []domain.Account `json:"accounts"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Accounts) != 2 {
		t.Fatalf("accounts = %#v", payload.Accounts)
	}
}
