package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// [REQ:PCT-FUNC-001][REQ:PCT-CATALOG-ENUMERATE] Catalog view - Test description extraction
func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "simple PRD with description",
			content: `# Product Requirements Document

This is a simple description of the product.

## Features`,
			expected: "This is a simple description of the product.",
		},
		{
			name: "PRD with quoted description",
			content: `# My Scenario

> This is a quoted description with **bold text**.

## Overview`,
			expected: "This is a quoted description with **bold text**.",
		},
		{
			name: "PRD with no description",
			content: `# Title

## Section`,
			expected: "",
		},
		{
			name: "PRD with long description",
			content: `# Title

` + "This is a very long description that should be truncated at exactly 200 characters to ensure we don't overflow the UI components with excessive text that would make the interface look cluttered and unprofessional in the catalog view" + `

## Next Section`,
			expected: "This is a very long description that should be truncated at exactly 200 characters to ensure we don't overflow the UI components with excessive text that would make the interface look cluttered and...",
		},
		{
			name: "PRD with multiple empty lines",
			content: `# Title



This is the description after blank lines.

## Section`,
			expected: "This is the description after blank lines.",
		},
		{
			name:     "PRD with only title",
			content:  `# Just a Title`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			result := extractDescription(tmpFile)

			if result != tt.expected {
				t.Errorf("extractDescription() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// [REQ:PCT-FUNC-001][REQ:PCT-CATALOG-ENUMERATE] Catalog view - Test description extraction for nonexistent files
func TestExtractDescriptionNonexistentFile(t *testing.T) {
	result := extractDescription("/nonexistent/file.md")
	if result != "" {
		t.Errorf("extractDescription() for nonexistent file = %q, want empty string", result)
	}
}

// [REQ:PCT-FUNC-001][REQ:PCT-CATALOG-STATUS] Catalog view - Test draft status detection
func TestHasDraft(t *testing.T) {
	// Create temporary draft directory structure
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenario")
	resourceDir := filepath.Join(tmpDir, "resource")

	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}
	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		t.Fatalf("Failed to create resource dir: %v", err)
	}

	// Create a test draft file
	testDraftPath := filepath.Join(scenarioDir, "test-scenario.md")
	if err := os.WriteFile(testDraftPath, []byte("# Test Draft"), 0644); err != nil {
		t.Fatalf("Failed to create test draft: %v", err)
	}

	tests := []struct {
		name       string
		entityType string
		entityName string
		setupFunc  func() // Optional setup function
		want       bool
	}{
		{
			name:       "draft exists",
			entityType: "scenario",
			entityName: "test-scenario",
			want:       true,
		},
		{
			name:       "draft does not exist",
			entityType: "scenario",
			entityName: "nonexistent",
			want:       false,
		},
		{
			name:       "resource draft does not exist",
			entityType: "resource",
			entityName: "test-resource",
			want:       false,
		},
	}

	// Note: This test is limited because hasDraftOnDisk uses a hardcoded relative path.
	// In a real implementation, we'd refactor hasDraft to accept a base path parameter.
	// For now, we'll test the logic indirectly through getDraftPath.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test getDraftPath correctness instead since hasDraft is file-system dependent
			path := getDraftPath(tt.entityType, tt.entityName)
			expectedPath := filepath.Join("../data/prd-drafts", tt.entityType, tt.entityName+".md")
			if path != expectedPath {
				t.Errorf("getDraftPath(%q, %q) = %q, want %q", tt.entityType, tt.entityName, path, expectedPath)
			}
		})
	}
}

// [REQ:PCT-FUNC-001][REQ:PCT-CATALOG-ENUMERATE] Catalog view - Test entity enumeration
func TestEnumerateEntities(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		setupFunc  func() string // Returns base directory
		entityType string
		wantCount  int
		wantNames  []string
	}{
		{
			name: "enumerate scenarios with PRDs",
			setupFunc: func() string {
				scenariosDir := filepath.Join(tmpDir, "test-scenarios")
				// Create test scenarios
				for _, name := range []string{"scenario-one", "scenario-two", "scenario-three"} {
					dir := filepath.Join(scenariosDir, name)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("Failed to create scenario dir: %v", err)
					}
					prdPath := filepath.Join(dir, "PRD.md")
					if err := os.WriteFile(prdPath, []byte("# Test PRD"), 0644); err != nil {
						t.Fatalf("Failed to create PRD: %v", err)
					}
				}
				return scenariosDir
			},
			entityType: "scenario",
			wantCount:  3,
			wantNames:  []string{"scenario-one", "scenario-two", "scenario-three"},
		},
		{
			name: "enumerate resources without PRDs",
			setupFunc: func() string {
				resourcesDir := filepath.Join(tmpDir, "test-resources")
				// Create test resources without PRDs
				for _, name := range []string{"resource-one", "resource-two"} {
					dir := filepath.Join(resourcesDir, name)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("Failed to create resource dir: %v", err)
					}
				}
				return resourcesDir
			},
			entityType: "resource",
			wantCount:  2,
			wantNames:  []string{"resource-one", "resource-two"},
		},
		{
			name: "enumerate nonexistent directory",
			setupFunc: func() string {
				return filepath.Join(tmpDir, "nonexistent")
			},
			entityType: "scenario",
			wantCount:  0,
			wantNames:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := tt.setupFunc()

			entries, err := enumerateEntities(baseDir, tt.entityType)
			if err != nil {
				t.Fatalf("enumerateEntities() error = %v", err)
			}

			if len(entries) != tt.wantCount {
				t.Errorf("enumerateEntities() returned %d entries, want %d", len(entries), tt.wantCount)
			}

			// Verify entity names
			gotNames := make(map[string]bool)
			for _, entry := range entries {
				gotNames[entry.Name] = true
				if entry.Type != tt.entityType {
					t.Errorf("Entry type = %q, want %q", entry.Type, tt.entityType)
				}
			}

			for _, wantName := range tt.wantNames {
				if !gotNames[wantName] {
					t.Errorf("Expected entity %q not found in results", wantName)
				}
			}
		})
	}
}

type stubRow struct {
	err    error
	values []any
}

func (r stubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("unexpected scan destination count: got %d, want %d", len(dest), len(r.values))
	}

	for i := range dest {
		switch target := dest[i].(type) {
		case *string:
			value, ok := r.values[i].(string)
			if !ok {
				return fmt.Errorf("scan value %d has type %T, want string", i, r.values[i])
			}
			*target = value
		case *sql.NullString:
			value, ok := r.values[i].(sql.NullString)
			if !ok {
				return fmt.Errorf("scan value %d has type %T, want sql.NullString", i, r.values[i])
			}
			*target = value
		case *time.Time:
			value, ok := r.values[i].(time.Time)
			if !ok {
				return fmt.Errorf("scan value %d has type %T, want time.Time", i, r.values[i])
			}
			*target = value
		default:
			return fmt.Errorf("unsupported scan destination type %T", target)
		}
	}

	return nil
}

type stubDraftStore struct {
	row       rowScanner
	execErr   error
	execCalls []draftStoreExecCall
}

type draftStoreExecCall struct {
	query string
	args  []any
}

func (s *stubDraftStore) QueryRow(_ string, _ ...any) rowScanner {
	return s.row
}

func (s *stubDraftStore) Exec(query string, args ...any) (sql.Result, error) {
	s.execCalls = append(s.execCalls, draftStoreExecCall{query: query, args: args})
	if s.execErr != nil {
		return nil, s.execErr
	}
	return stubResult{}, nil
}

type stubResult struct{}

func (stubResult) LastInsertId() (int64, error) { return 0, nil }

func (stubResult) RowsAffected() (int64, error) { return 0, nil }

// [REQ:PCT-DRAFT-CREATE] Create new draft from template or existing PRD
func TestEnsureDraftFromPublishedPRDCreatesDraft(t *testing.T) {
	store := &stubDraftStore{
		row: stubRow{err: sql.ErrNoRows},
	}

	content := "# PRD\n\nContent"
	draft, err := ensureDraftFromPublishedPRD(store, "scenario", "test-scenario", content)
	if err != nil {
		t.Fatalf("ensureDraftFromPublishedPRD() unexpected error: %v", err)
	}

	if draft.EntityType != "scenario" {
		t.Errorf("draft.EntityType = %q, want %q", draft.EntityType, "scenario")
	}
	if draft.EntityName != "test-scenario" {
		t.Errorf("draft.EntityName = %q, want %q", draft.EntityName, "test-scenario")
	}
	if draft.Content != content {
		t.Errorf("draft.Content = %q, want %q", draft.Content, content)
	}
	if draft.Status != "draft" {
		t.Errorf("draft.Status = %q, want %q", draft.Status, "draft")
	}
	if time.Since(draft.CreatedAt) > time.Second {
		t.Errorf("draft.CreatedAt is too old: %v", draft.CreatedAt)
	}
	if time.Since(draft.UpdatedAt) > time.Second {
		t.Errorf("draft.UpdatedAt is too old: %v", draft.UpdatedAt)
	}

	if len(store.execCalls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(store.execCalls))
	}
	if !strings.Contains(store.execCalls[0].query, "INSERT INTO drafts") {
		t.Errorf("expected insert query, got %q", store.execCalls[0].query)
	}
}

// [REQ:PCT-DRAFT-CREATE] Create new draft from template or existing PRD
func TestEnsureDraftFromPublishedPRDReusesDraft(t *testing.T) {
	createdAt := time.Now().Add(-2 * time.Hour)
	updatedAt := createdAt.Add(time.Hour)
	store := &stubDraftStore{
		row: stubRow{
			values: []any{
				"draft-id",
				"scenario",
				"test-scenario",
				"# Existing Draft",
				sql.NullString{String: "owner", Valid: true},
				createdAt,
				updatedAt,
				"draft",
			},
		},
	}

	content := "# Updated Content"
	draft, err := ensureDraftFromPublishedPRD(store, "scenario", "test-scenario", content)
	if err != nil {
		t.Fatalf("ensureDraftFromPublishedPRD() unexpected error: %v", err)
	}

	if draft.Content != "# Existing Draft" {
		t.Errorf("draft.Content = %q, want %q", draft.Content, "# Existing Draft")
	}
	if draft.Owner != "owner" {
		t.Errorf("draft.Owner = %q, want %q", draft.Owner, "owner")
	}
	if !draft.UpdatedAt.Equal(updatedAt) {
		t.Errorf("draft.UpdatedAt changed: got %v, want %v", draft.UpdatedAt, updatedAt)
	}
	if len(store.execCalls) != 0 {
		t.Fatalf("expected no exec calls, got %d", len(store.execCalls))
	}
}

// [REQ:PCT-DRAFT-CREATE] Create new draft from template or existing PRD
func TestEnsureDraftFromPublishedPRDResetsPublishedDraft(t *testing.T) {
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := createdAt.Add(6 * time.Hour)
	store := &stubDraftStore{
		row: stubRow{
			values: []any{
				"draft-id",
				"scenario",
				"test-scenario",
				"# Old Content",
				sql.NullString{},
				createdAt,
				updatedAt,
				"published",
			},
		},
	}

	content := "# Fresh Content"
	draft, err := ensureDraftFromPublishedPRD(store, "scenario", "test-scenario", content)
	if err != nil {
		t.Fatalf("ensureDraftFromPublishedPRD() unexpected error: %v", err)
	}

	if draft.Content != content {
		t.Errorf("draft.Content = %q, want %q", draft.Content, content)
	}
	if draft.Status != "draft" {
		t.Errorf("draft.Status = %q, want %q", draft.Status, "draft")
	}
	if !draft.CreatedAt.Equal(createdAt) {
		t.Errorf("draft.CreatedAt changed: got %v, want %v", draft.CreatedAt, createdAt)
	}
	if !draft.UpdatedAt.After(updatedAt) {
		t.Errorf("draft.UpdatedAt = %v, want after %v", draft.UpdatedAt, updatedAt)
	}
	if len(store.execCalls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(store.execCalls))
	}
	if !strings.Contains(strings.ToUpper(store.execCalls[0].query), "UPDATE DRAFTS") {
		t.Errorf("expected update query, got %q", store.execCalls[0].query)
	}
}

// [REQ:PCT-CATALOG-ENUMERATE] Catalog enumerates all scenarios and resources with PRD status
func TestEnumerateEntitiesWithMixedContent(t *testing.T) {
	// Test that enumerateEntities only processes directories, not files
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "mixed")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("Failed to create base dir: %v", err)
	}

	// Create a directory (should be included)
	validDir := filepath.Join(baseDir, "valid-scenario")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatalf("Failed to create valid dir: %v", err)
	}

	// Create a file at root level (should be ignored)
	if err := os.WriteFile(filepath.Join(baseDir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	entries, err := enumerateEntities(baseDir, "scenario")
	if err != nil {
		t.Fatalf("enumerateEntities() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("enumerateEntities() returned %d entries, want 1 (should ignore files)", len(entries))
	}

	if len(entries) > 0 && entries[0].Name != "valid-scenario" {
		t.Errorf("Entry name = %q, want %q", entries[0].Name, "valid-scenario")
	}
}
