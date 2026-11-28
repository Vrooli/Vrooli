package dependencygovernance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	governanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_v1connect"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"
)

type slowInstallFixture struct {
	calls atomic.Int32
}

func (f *slowInstallFixture) Install(ctx context.Context, _ installgateway.Resolution) (string, error) {
	f.calls.Add(1)
	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return "staged fixture", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestInstallTransportPreservesReceiptPastOrdinaryHTTPWriteTimeout(t *testing.T) {
	for _, test := range []struct {
		name   string
		budget bool
	}{
		{"ordinary server loses receipt", false},
		{"install transport delivers receipt", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeRegistry(t, root, `{"records":[{"ecosystem":"npm","package_name":"react","version_range":"19.0.0","state":"approved","rationale":"Isolated fixture","allowed_surfaces":["ui"]}]}`)
			scenariosDir := filepath.Join(root, "scenarios")
			ui := filepath.Join(scenariosDir, "fixture", "ui")
			if err := os.MkdirAll(ui, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(ui, "pnpm-lock.yaml"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			installer := &slowInstallFixture{}
			router := gin.New()
			RegisterConnectRoutes(router, func() string { return scenariosDir }, func(h *connectHandler) {
				withTestDiskAdmission(t, h)
				h.installer = installer
			})
			var handler http.Handler = router
			if test.budget {
				var err error
				handler, err = WithInstallTransport(handler)
				if err != nil {
					t.Fatal(err)
				}
			}
			server := httptest.NewUnstartedServer(handler)
			server.Config.WriteTimeout = 75 * time.Millisecond
			server.Start()
			defer server.Close()

			client := governanceconnect.NewDependencyGovernanceServiceClient(server.Client(), server.URL)
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			response, err := client.InstallDependency(ctx, connect.NewRequest(&governancev1.InstallDependencyRequest{
				Scenario: "fixture", Surface: "ui", Ecosystem: "npm",
				PackageName: "react", Version: "19.0.0", Apply: true,
			}))
			if installer.calls.Load() != 1 {
				t.Fatal("request did not reach the governed installer")
			}
			if !test.budget {
				if err == nil || !strings.Contains(err.Error(), "EOF") {
					t.Fatalf("expected lost HTTP response after write timeout, got %v", err)
				}
				return
			}
			if err != nil || !response.Msg.GetInstalled() || response.Msg.GetVerdict() != "approved" {
				t.Fatalf("governed receipt = %v, %v", response, err)
			}
		})
	}
}

func TestInstallTransportLeavesOtherMethodsAndProceduresAlone(t *testing.T) {
	handler, err := WithInstallTransport(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, governanceconnect.DependencyGovernanceServiceInstallDependencyProcedure, nil),
		httptest.NewRequest(http.MethodPost, governanceconnect.DependencyGovernanceServiceListApprovedDependenciesProcedure, nil),
	} {
		// The recorder cannot change write deadlines. The middleware would
		// refuse the request before dispatch if it tried to do so here.
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s %s received %d", request.Method, request.URL.Path, response.Code)
		}
	}
}
