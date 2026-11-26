package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestListGraphsFiltersJSONBTags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(_ string, actual string) error {
		if strings.Contains(actual, "jsonb_typeof(tags) = 'array' AND tags @> jsonb_build_array($1::text)") {
			return nil
		}
		return fmt.Errorf("tag filter must use JSONB array containment, got query: %s", actual)
	})))
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer db.Close()

	created := time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery("jsonb tag containment").
		WithArgs("urgent", 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "type", "description", "metadata", "version", "created_by", "created_at", "updated_at", "tags",
		}).AddRow(
			"00000000-0000-0000-0000-000000000001", "Urgent graph", "mind-maps", "fixture", `{}`, 1, "owner", created, created, `["urgent"]`,
		))

	router := gin.New()
	router.Use(DatabaseMiddleware(db))
	router.GET("/api/v1/graphs", NewAPI().ListGraphs)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/graphs?tag=urgent", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 for a matching JSONB tag filter, got %d: %s", response.Code, response.Body.String())
	}
	var result struct {
		Data []Graph `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(result.Data) != 1 || len(result.Data[0].Tags) != 1 || result.Data[0].Tags[0] != "urgent" {
		t.Fatalf("expected the matching tagged graph, got %#v", result.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
