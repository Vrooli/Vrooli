package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type TestArtifactSummary struct {
	Count      int      `json:"count"`
	TotalBytes int64    `json:"total_bytes"`
	Paths      []string `json:"paths,omitempty"`
}

type TestArtifactCleanupResult struct {
	RemovedCount int   `json:"removed_count"`
	FreedBytes   int64 `json:"freed_bytes"`
}

func testArtifactPaths() []string {
	var paths []string
	if entries, err := os.ReadDir("/tmp"); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "scenario-to-desktop-test-") {
				paths = append(paths, filepath.Join("/tmp", name))
			}
		}
	}
	// Fallback to glob in case readdir fails for some reason.
	if len(paths) == 0 {
		if globbed, _ := filepath.Glob("/tmp/scenario-to-desktop-test-*"); len(globbed) > 0 {
			paths = globbed
		}
	}
	return paths
}

func sizeOfPath(root string) int64 {
	var total int64
	filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info, err := d.Info(); err == nil && info != nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func summarizeTestArtifacts(maxPaths int) TestArtifactSummary {
	paths := testArtifactPaths()
	summary := TestArtifactSummary{}
	if len(paths) == 0 {
		return summary
	}

	limit := maxPaths
	if limit <= 0 || limit > len(paths) {
		limit = len(paths)
	}

	for idx, p := range paths {
		if idx < limit {
			summary.Paths = append(summary.Paths, p)
		}
		summary.Count++
		summary.TotalBytes += sizeOfPath(p)
	}
	return summary
}

// listTestArtifactsHandler reports how many CI/test artifacts remain in /tmp.
func (s *Server) listTestArtifactsHandler(w http.ResponseWriter, _ *http.Request) {
	summary := summarizeTestArtifacts(5)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// cleanupTestArtifactsHandler deletes CI/test artifact directories from /tmp.
func (s *Server) cleanupTestArtifactsHandler(w http.ResponseWriter, _ *http.Request) {
	paths := testArtifactPaths()
	var result TestArtifactCleanupResult
	for _, p := range paths {
		result.FreedBytes += sizeOfPath(p)
		if err := os.RemoveAll(p); err == nil {
			result.RemovedCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
