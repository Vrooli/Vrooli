package integrationstatus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
	"swarm-manager/internal/transitions"
)

func TestScenarioCheckerReportsConfiguredAvailability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	status, err := (ScenarioChecker{Scenario: "plan-manager", Required: true, DegradedBehavior: "plan work is parked", ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, Now: func() time.Time { return now }, FreshFor: time.Minute}).Check(context.Background())
	if err != nil || !status.Configured || status.Availability != Available || !status.FreshUntil.Equal(now.Add(time.Minute)) {
		t.Fatalf("status=%#v err=%v", status, err)
	}
}

func TestScenarioCheckerReportsUnconfiguredWithoutTransportError(t *testing.T) {
	status, err := (ScenarioChecker{Scenario: "missing", DegradedBehavior: "work is parked", ResolveURL: func(context.Context, string) (string, error) { return "", nil }}).Check(context.Background())
	if err != nil || status.Availability != Unconfigured || status.Configured {
		t.Fatalf("status=%#v err=%v", status, err)
	}
}

func TestScenarioCheckerPreservesResolverFailureAndRecovers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	for _, cause := range []error{
		&discovery.Error{Kind: discovery.ErrScenarioNotRunning, Scenario: "git-control-tower", Err: errors.New("scenario is stopped")},
		&discovery.Error{Kind: discovery.ErrTimeout, Scenario: "git-control-tower", Err: context.DeadlineExceeded},
	} {
		t.Run(cause.Error(), func(t *testing.T) {
			resolveErr := cause
			checker := ScenarioChecker{Scenario: "git-control-tower", Required: true, DegradedBehavior: "regression-gated work is parked", ResolveURL: func(context.Context, string) (string, error) {
				if resolveErr != nil {
					return "", resolveErr
				}
				return server.URL, nil
			}}
			provider := New(map[string]Checker{"git-control-tower": checker})
			definition := transitions.Definition{Key: "plan.execute", Requires: []string{"git-control-tower"}}
			status, err := checker.Check(context.Background())
			if err != nil || status.Availability != Unavailable || !status.Configured || !strings.Contains(status.Diagnostic, cause.Error()) {
				t.Fatalf("resolver failure lost its unavailable diagnosis: status=%#v err=%v", status, err)
			}
			if err := provider.Preflight(context.Background(), definition); err == nil || !strings.Contains(err.Error(), cause.Error()) {
				t.Fatalf("preflight did not retain original resolver cause: %v", err)
			}
			resolveErr = nil
			if err := provider.Preflight(context.Background(), definition); err != nil {
				t.Fatalf("restored original dependency should admit without reconfiguration: %v", err)
			}
		})
	}
}

func TestPlanExecuteRequirementsHaveRequiredStartup(t *testing.T) {
	root := filepath.Join("..", "..", "..", ".vrooli")
	registry, err := transitions.LoadDir(filepath.Join(root, "swarm-transitions"))
	if err != nil {
		t.Fatal(err)
	}
	definition, ok := registry.Get("plan.execute")
	if !ok {
		t.Fatal("plan.execute is missing")
	}
	data, err := os.ReadFile(filepath.Join(root, "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Dependencies struct {
			Scenarios map[string]struct {
				Enabled       bool   `json:"enabled"`
				Required      bool   `json:"required"`
				StartupPolicy string `json:"startup_policy"`
			} `json:"scenarios"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, name := range definition.Requires {
		dependency, exists := manifest.Dependencies.Scenarios[name]
		if !exists || !dependency.Enabled || !dependency.Required || dependency.StartupPolicy != "must_start" {
			t.Errorf("plan.execute requires %s, but lifecycle cannot guarantee its startup: %+v", name, dependency)
		}
	}
}
