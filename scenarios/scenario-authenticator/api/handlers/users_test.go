package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"

	"scenario-authenticator/db"
	"scenario-authenticator/models"
)

func TestNormalizeRoles(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty input ensures user role",
			input:    nil,
			expected: []string{"user"},
		},
		{
			name:     "deduplicates and lowercases",
			input:    []string{"Admin", "ADMIN", "User"},
			expected: []string{"admin", "user"},
		},
		{
			name:     "adds user when missing",
			input:    []string{"moderator"},
			expected: []string{"moderator", "user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeRoles(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d roles, got %d (%v)", len(tt.expected), len(got), got)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Fatalf("expected role %q at index %d, got %q", tt.expected[i], i, got[i])
				}
			}
		})
	}
}

func TestDeleteUserHandler_Success(t *testing.T) {
	// Setup sqlmock
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer sqlDB.Close()

	db.DB = sqlDB
	defer func() { db.DB = nil }()

	// Mock redis using miniredis
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer redisServer.Close()

	db.RedisClient = redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer func() { db.RedisClient = nil }()

	userID := "2b71ad26-4a64-42c5-bd55-812f5da1d6a7"
	email := "temp-admin@vrooli.local"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT email, deleted_at FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"email", "deleted_at"}).AddRow(email, nil))

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET deleted_at = $1, updated_at = $1 WHERE id = $2")).
		WithArgs(sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM user_roles WHERE user_id = $1")).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE api_keys SET revoked_at = $1, updated_at = $1 WHERE user_id = $2 AND revoked_at IS NULL")).
		WithArgs(sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL")).
		WithArgs(sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	expectationQuery := `(?s)SELECT EXISTS \(\s*SELECT 1 FROM information_schema\.tables\s*WHERE table_schema = 'public' AND table_name = \$1\s*\)`

	mock.ExpectQuery(expectationQuery).
		WithArgs("application_users").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectQuery(expectationQuery).
		WithArgs("application_sessions").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectCommit()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audit_logs")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID, nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", userID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, "claims", &models.Claims{
		UserID: "85981e2d-4f87-4eef-b344-4c6b48d4ab0e",
		Roles:  []string{"admin"},
	})
	req = req.WithContext(ctx)
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	DeleteUserHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (%s)", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if success, ok := body["success"].(bool); !ok || !success {
		t.Fatalf("expected success=true, got %v", body)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
