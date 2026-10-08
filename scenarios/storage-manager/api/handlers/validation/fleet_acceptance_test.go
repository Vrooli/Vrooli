package validation

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

type fleetAcceptanceReport struct {
	Reports    []fleetAcceptanceOwner `json:"reports"`
	ErrorCount int                    `json:"error_count"`
}

type fleetAcceptanceOwner struct {
	OwnerKind string                   `json:"owner_kind"`
	OwnerID   string                   `json:"owner_id"`
	Findings  []fleetAcceptanceFinding `json:"findings"`
}

type fleetAcceptanceFinding struct {
	Code     string `json:"code"`
	Severity int    `json:"severity"`
}

// TestFleetValidationAcceptance exercises the same REST surface used by
// `storage-manager validate fleet`. This is the product acceptance gate for
// all owner kinds; Test Genie remains responsible for scenario-shaped storage
// analysis only.
func TestFleetValidationAcceptance(t *testing.T) {
	root := findRepoRoot(t)
	router := mux.NewRouter()
	Module(log.New(io.Discard, "", 0), root).Mount(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/validation/validate/fleet?platform=linux", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("validate fleet status = %d, body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.Bytes()
	var report fleetAcceptanceReport
	if err := json.Unmarshal(body, &report); err != nil {
		t.Fatalf("decode fleet report: %v", err)
	}
	if len(report.Reports) == 0 {
		t.Fatal("validate fleet returned no owner reports")
	}
	var severe []string
	coverageFindings := 0
	for _, owner := range report.Reports {
		for _, finding := range owner.Findings {
			if finding.Code == "STORAGE_PATH_UNCOVERED" {
				coverageFindings++
				continue
			}
			if finding.Code == "STORAGE_PATH_NOT_PORTABLE" {
				continue
			}
			if finding.Severity >= 2 {
				severe = append(severe, owner.OwnerKind+"/"+owner.OwnerID+":"+finding.Code)
			}
		}
	}
	if len(severe) > 0 {
		t.Fatalf("validate fleet returned unexpected error or blocker findings: %v", severe)
	}
	if coverageFindings == 0 {
		t.Fatal("validate fleet did not surface any undeclared class-root coverage findings")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repository root not found above %s", dir)
		}
		dir = parent
	}
}
