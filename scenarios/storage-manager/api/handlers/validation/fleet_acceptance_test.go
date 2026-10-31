package validation

import (
	"encoding/json"
	corestorage "github.com/vrooli/api-core/storage"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	// Fleet drift is operator data, not a unit-test prerequisite. Exercise every
	// owner kind against a bounded repository and leave live fleet certification
	// to the explicit validate-fleet operation.
	root := fleetAcceptanceFixture(t)
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
	kinds := map[string]bool{}
	for _, owner := range report.Reports {
		kinds[owner.OwnerKind] = true
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
	for _, kind := range []string{"scenario", "resource", "tool", "safeguard"} {
		if !kinds[kind] {
			t.Fatalf("fleet omitted owner kind %s: %#v", kind, report.Reports)
		}
	}
	if len(severe) > 0 {
		t.Fatalf("validate fleet returned unexpected error or blocker findings: %v", severe)
	}
	if coverageFindings == 0 {
		t.Fatal("validate fleet did not surface any undeclared class-root coverage findings")
	}
}

func fleetAcceptanceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, ".vrooli"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("SCENARIO_DATA_DIR", "")
	t.Setenv("SCENARIO_NAME", "")
	t.Setenv("VROOLI_SCENARIO", "")
	files := map[string]string{
		"scenarios/demo/.vrooli/service.json":     `{"service":{"name":"demo"},"storage":{"entries":{}}}`,
		"resources/demo/resource.json":            `{"name":"demo","storage":{"entries":{}}}`,
		"internal/tools/demo/tool.json":           `{"name":"demo","storage":{"entries":{}}}`,
		"internal/safeguards/demo/safeguard.json": `{"name":"demo","storage":{"entries":{}}}`,
	}
	for relative, data := range files {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// An undeclared real class-root entry must remain visible in the report.
	owner := corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "demo", ManifestPath: filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json")}
	classRoot, err := corestorage.ResolveOwnerStoragePath(root, owner, corestorage.StorageEntry{Rung: corestorage.RungOwned, Kind: "dir", Class: corestorage.ClassData}, corestorage.PlatformLinux, corestorage.PlatformSeams{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(classRoot, root+string(filepath.Separator)) {
		t.Fatalf("fixture escaped its temporary root: %s", classRoot)
	}
	unknown := filepath.Join(classRoot, "undeclared")
	if err := os.MkdirAll(filepath.Dir(unknown), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unknown, []byte("fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}
