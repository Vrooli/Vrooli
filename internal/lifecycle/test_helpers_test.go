package lifecycle

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

var lifecycleFixturePorts = struct {
	sync.Mutex
	used map[string]map[int]struct{}
}{used: make(map[string]map[int]struct{})}

func writeLifecycleFixture(t *testing.T, root, name string) {
	t.Helper()
	writeLifecycleFixtureManifest(t, root, lifecycleFixtureManifest(name))
}

func cleanupRunner(t *testing.T, runner *Runner, name string, opts StopOptions) {
	t.Helper()
	t.Cleanup(func() {
		if err := runner.Stop(name, opts); err != nil {
			t.Errorf("Stop(%q, %#v) during cleanup: %v", name, opts, err)
		}
	})
}

func writeLifecycleFixtureManifest(t *testing.T, root string, manifest scenario.ServiceManifest) {
	t.Helper()

	if strings.TrimSpace(manifest.Service.Name) == "" {
		t.Fatalf("fixture manifest is missing service name")
	}
	// The real lifecycle allocator must never be coupled to a shared fixed
	// port. Allocate the fixture's API port from the OS and release it before
	// the runner starts; the runner then exercises its normal declared-port
	// path with an isolated value.
	if apiPort, ok := manifest.Ports["api"]; ok && apiPort.Port == nil && apiPort.Range == "22000-22010" {
		var listener net.Listener
		var err error
		lifecycleFixturePorts.Lock()
		defer lifecycleFixturePorts.Unlock()
		used := lifecycleFixturePorts.used[root]
		if used == nil {
			used = make(map[int]struct{})
			lifecycleFixturePorts.used[root] = used
		}
		// Port 0 is OS allocated, but Linux places it in the ephemeral band,
		// which the repository deliberately rejects for scenario APIs. Ask the
		// OS to bind candidates in the canonical API band instead.
		for port := 15000; port <= 19999; port++ {
			if _, alreadyUsed := used[port]; alreadyUsed {
				continue
			}
			listener, err = net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
			if err == nil {
				used[port] = struct{}{}
				break
			}
		}
		if err != nil || listener == nil {
			t.Fatalf("allocate lifecycle fixture port in API band: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		if err := listener.Close(); err != nil {
			t.Fatalf("release lifecycle fixture port %d: %v", port, err)
		}
		apiPort.Range = ""
		apiPort.Port = &port
		manifest.Ports["api"] = apiPort
		if health := manifest.Lifecycle.Health; health != nil {
			for i := range health.Checks {
				health.Checks[i].Target = strings.ReplaceAll(health.Checks[i].Target, "${API_PORT}", strconv.Itoa(port))
			}
		}
	}

	testresource.WritePortRegistry(t, root, nil)
	// Smoke tests construct a real Runner, which loads the root manifest during
	// host-requirement resolution. Write an empty project manifest so resolution
	// succeeds with an empty scope; tests that exercise host requirements can
	// overwrite this fixture.
	testscenario.WriteProjectService(t, root, testscenario.ProjectServiceManifest())
	// Fixture repositories live under the OS temp dir, outside any checkout.
	// Go's VCS stamping would probe whatever ancestor .git it finds (a stray
	// /tmp/.git or ~/.git fails every build with "error obtaining VCS
	// status"), so fixture scenario builds never stamp.
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(`{"build.go_flags":{"develop":["-trimpath","-buildvcs=false"],"distribution":["-trimpath","-buildvcs=false"],"scenario":["-trimpath","-buildvcs=false"]}}`), 0o644); err != nil {
		t.Fatalf("write build flag policy: %v", err)
	}
	testscenario.WriteScenarioService(t, root, manifest.Service.Name, manifest)
	for _, component := range manifest.Components {
		if component.Build.Kind != "go_module" || component.Build.Output != "api/mock-api" {
			continue
		}
		apiDir := filepath.Join(root, "scenarios", manifest.Service.Name, "api")
		if err := os.MkdirAll(apiDir, 0o755); err != nil {
			t.Fatal(err)
		}
		_, helperFile, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatal("resolve lifecycle fixture module path")
		}
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(helperFile), "../.."))
		fixtureModule := fmt.Sprintf("module fixture\n\ngo 1.25.0\n\nrequire (\n\tconnectrpc.com/connect v1.19.2\n\tgithub.com/vrooli/vrooli/packages/proto v0.0.0\n)\n\nreplace github.com/vrooli/vrooli/packages/proto => %s\n", filepath.Join(repoRoot, "packages", "proto"))
		if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte(fixtureModule), 0o600); err != nil {
			t.Fatal(err)
		}
		fixtureSums := map[string]struct{}{}
		for _, sumPath := range []string{filepath.Join(repoRoot, "go.sum"), filepath.Join(repoRoot, "packages", "proto", "go.sum")} {
			data, err := os.ReadFile(sumPath)
			if err != nil {
				t.Fatalf("read fixture module sums %q: %v", sumPath, err)
			}
			for _, line := range strings.Split(string(data), "\n") {
				if strings.TrimSpace(line) != "" {
					fixtureSums[line] = struct{}{}
				}
			}
		}
		sumLines := make([]string, 0, len(fixtureSums))
		for line := range fixtureSums {
			sumLines = append(sumLines, line)
		}
		sort.Strings(sumLines)
		if err := os.WriteFile(filepath.Join(apiDir, "go.sum"), []byte(strings.Join(sumLines, "\n")+"\n"), 0o600); err != nil {
			t.Fatalf("write fixture module sums: %v", err)
		}
		mainSource := `package main
import (
 "context"
 "encoding/json"
 "fmt"
 "net/http"
 "os"
 "path/filepath"
 "sync"
 "connectrpc.com/connect"
 commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
 commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)
type maintenance struct { commonv1connect.UnimplementedLifecycleMaintenanceServiceHandler; mu sync.Mutex; op string; closed, drained bool; statePath string }
type maintenanceState struct { OperationID string ` + "`json:\"operation_id\"`" + `; Closed bool ` + "`json:\"closed\"`" + `; Drained bool ` + "`json:\"drained\"`" + ` }
func newMaintenance() *maintenance { home:=os.Getenv("VROOLI_HOME"); if home=="" { home=os.TempDir() }; dir:=filepath.Join(home,"fixture-maintenance"); _=os.MkdirAll(dir,0700); s:=&maintenance{statePath:filepath.Join(dir,os.Getenv("VROOLI_SCENARIO")+".json")}; if data,err:=os.ReadFile(s.statePath); err==nil { var state maintenanceState; if json.Unmarshal(data,&state)==nil { s.op=state.OperationID; s.closed=state.Closed; s.drained=state.Drained } }; return s }
func (s *maintenance) persist() error { data,err:=json.Marshal(maintenanceState{OperationID:s.op,Closed:s.closed,Drained:s.drained}); if err!=nil{return err}; return os.WriteFile(s.statePath,data,0600) }
func (s *maintenance) standing() *commonv1.LifecycleStanding { return &commonv1.LifecycleStanding{Provider: os.Getenv("VROOLI_SCENARIO"), ProviderInstanceId: "fixture-provider", OperationId: s.op, FenceToken: "fixture-fence", Revision: 1, AdmissionClosed: s.closed, InventoryComplete: true, Drained: s.drained, ComponentStopReady: s.drained} }
func (s *maintenance) Status(context.Context, *connect.Request[commonv1.LifecycleStatusRequest]) (*connect.Response[commonv1.LifecycleStatusResponse], error) { s.mu.Lock(); defer s.mu.Unlock(); return connect.NewResponse(&commonv1.LifecycleStatusResponse{Standing:s.standing()}),nil }
func (s *maintenance) Prepare(_ context.Context, r *connect.Request[commonv1.LifecyclePrepareRequest]) (*connect.Response[commonv1.LifecyclePrepareResponse], error) { s.mu.Lock(); defer s.mu.Unlock(); s.op=r.Msg.GetOperationId(); s.closed=true; if err:=s.persist(); err!=nil{return nil,connect.NewError(connect.CodeInternal,err)}; return connect.NewResponse(&commonv1.LifecyclePrepareResponse{Standing:s.standing()}),nil }
func (s *maintenance) Drain(_ context.Context, r *connect.Request[commonv1.LifecycleDrainRequest]) (*connect.Response[commonv1.LifecycleDrainResponse], error) { s.mu.Lock(); defer s.mu.Unlock(); if r.Msg.GetOperationId()!=s.op || r.Msg.GetFenceToken()!="fixture-fence" || r.Msg.GetRevision()!=1 { return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("unexpected fixture fence identity")) }; s.drained=true; if err:=s.persist(); err!=nil{return nil,connect.NewError(connect.CodeInternal,err)}; return connect.NewResponse(&commonv1.LifecycleDrainResponse{Standing:s.standing()}),nil }
func (s *maintenance) Resume(_ context.Context, r *connect.Request[commonv1.LifecycleResumeRequest]) (*connect.Response[commonv1.LifecycleResumeResponse], error) { s.mu.Lock(); defer s.mu.Unlock(); if r.Msg.GetOperationId()!=s.op || r.Msg.GetFenceToken()!="fixture-fence" || r.Msg.GetRevision()!=1 { return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("unexpected fixture fence identity")) }; s.closed=false; s.drained=false; if err:=s.persist(); err!=nil{return nil,connect.NewError(connect.CodeInternal,err)}; return connect.NewResponse(&commonv1.LifecycleResumeResponse{}),nil }
func main() {
 mux := http.NewServeMux()
 mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
  _, _ = fmt.Fprintf(w, "{\"status\":\"healthy\",\"service\":\"fixture\",\"timestamp\":\"2026-05-08T12:00:00Z\",\"readiness\":true,\"build_identity\":\"%s\"}", os.Getenv("VROOLI_BUILD_IDENTITY"))
 })
 _, handler := commonv1connect.NewLifecycleMaintenanceServiceHandler(newMaintenance())
 mux.Handle("/common.v1.LifecycleMaintenanceService/", handler)
 _ = http.ListenAndServe("127.0.0.1:"+os.Getenv("API_PORT"), mux)
}
`
		if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(mainSource), 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = apiDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fixture dependency resolution: %v\n%s", err, out)
		}
	}
}

func lifecycleFixtureManifest(name string) scenario.ServiceManifest {
	return scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			Name:        name,
			DisplayName: "Lifecycle " + name,
			Description: "Lifecycle validation fixture",
			Version:     "0.1.0",
		},
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
				Range:  "22000-22010",
			},
		},
		Components: map[string]scenario.Component{
			"api": {
				Role:  "api",
				Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api", Output: "api/mock-api"},
				Run:   scenario.ComponentRun{Argv: []string{"{{bin.api}}"}, CWD: "api", Port: "api"},
			},
		},
		Lifecycle: scenario.Lifecycle{
			Version: "2.0.0",
			Health: &scenario.HealthConfig{
				Checks: []scenario.HealthCheck{
					{
						Name:     "api",
						Type:     "http",
						Target:   "http://127.0.0.1:${API_PORT}/health",
						Critical: true,
						Timeout:  1000,
					},
				},
				StartupGracePeriod: 250,
				Timeout:            5000,
				Interval:           250,
			},
			Setup:   scenario.Phase{},
			Develop: scenario.Phase{},
		},
	}
}
