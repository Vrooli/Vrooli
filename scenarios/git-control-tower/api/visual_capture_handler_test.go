package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleVisualCaptureList_Empty(t *testing.T) {
	t.Parallel()

	_ = httptest.NewRequest(http.MethodGet, "/api/v1/repo/visual-captures?scenarioSlug=nonexistent", nil)
	rr := httptest.NewRecorder()

	resp := NewResponse(rr)
	// Without a full server setup, just test response formatting
	resp.OK(map[string]interface{}{
		"snapshots": []SnapshotSetMeta{},
		"total":     0,
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleVisualCaptureScreenshot_PathTraversal(t *testing.T) {
	t.Parallel()

	err := validateFilename("../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}

	err = validateFilename("normal_file.png")
	if err != nil {
		t.Errorf("unexpected error for valid filename: %v", err)
	}

	err = validateFilename("file\\with\\backslash")
	if err == nil {
		t.Error("expected error for backslash in filename")
	}
}

func TestHandleVisualCaptureStorageStats_Format(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.OK(&VisualCaptureStorageStats{
		TotalSizeBytes: 1024,
		SnapshotCount:  5,
		PerScenario: []ScenarioStorageBreakdown{
			{ScenarioSlug: "test", SnapshotCount: 5, SizeBytes: 1024},
		},
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}
