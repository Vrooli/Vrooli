package exposure

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"tunnel-manager/internal/cmdrunner"
	"tunnel-manager/internal/manifest"
)

// ScenarioFileResolver resolves contract-owned scenario files.
type ScenarioFileResolver interface {
	ServiceFile(scenario string) (string, error)
}

// FilePortResolver resolves a scenario's fixed UI port from its contract-owned
// service manifest. It is the production PortResolver; tests use a fake.
type FilePortResolver struct {
	Files ScenarioFileResolver
}

// NewFilePortResolver constructs a resolver backed by the shared scenario
// file resolver.
func NewFilePortResolver(files ScenarioFileResolver) *FilePortResolver {
	return &FilePortResolver{Files: files}
}

var _ PortResolver = (*FilePortResolver)(nil)

// serviceJSON is the minimal slice of service.json this domain reads. A
// fixed UI port lives at ports.ui.port; ranged/dynamic scenarios omit it.
type serviceJSON struct {
	Ports map[string]struct {
		Port int `json:"port"`
	} `json:"ports"`
	Components map[string]struct {
		Run struct {
			Readiness struct {
				Type string `json:"type"`
				Path string `json:"path"`
			} `json:"readiness"`
		} `json:"run"`
	} `json:"components"`
	Lifecycle struct {
		Health struct {
			Endpoints map[string]string `json:"endpoints"`
		} `json:"health"`
	} `json:"lifecycle"`
}

func (r *FilePortResolver) loadService(scenario string) (serviceJSON, error) {
	if r == nil || r.Files == nil {
		return serviceJSON{}, ErrPortUnresolved{Scenario: scenario, Reason: "scenario file resolver not configured"}
	}
	path, err := r.Files.ServiceFile(scenario)
	if err != nil {
		return serviceJSON{}, ErrPortUnresolved{Scenario: scenario, Reason: fmt.Sprintf("resolve service.json: %v", err)}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return serviceJSON{}, ErrPortUnresolved{Scenario: scenario, Reason: fmt.Sprintf("service.json not readable: %v", err)}
	}
	var svc serviceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		return serviceJSON{}, ErrPortUnresolved{Scenario: scenario, Reason: fmt.Sprintf("service.json parse: %v", err)}
	}
	return svc, nil
}

func (r *FilePortResolver) UIPort(_ context.Context, scenario string) (int, error) {
	svc, err := r.loadService(scenario)
	if err != nil {
		return 0, err
	}
	ui, ok := svc.Ports["ui"]
	if !ok || ui.Port == 0 {
		return 0, ErrPortUnresolved{Scenario: scenario, Reason: "no fixed UI port declared"}
	}
	return ui.Port, nil
}

func (r *FilePortResolver) HealthPath(_ context.Context, scenario string) (string, error) {
	svc, err := r.loadService(scenario)
	if err != nil {
		return "", err
	}
	ui, ok := svc.Components["ui"]
	if ok && ui.Run.Readiness.Type == "http" && ui.Run.Readiness.Path != "" {
		return ui.Run.Readiness.Path, nil
	}
	if path := svc.Lifecycle.Health.Endpoints["ui"]; path != "" {
		return path, nil
	}
	return manifest.DefaultHealthPath, nil
}

// CLIRunner ensures a scenario is running by shelling `vrooli scenario
// start <scenario>` through the cmdrunner seam. It is the production
// Runner; tests use a fake.
//
// Latency fast-path: `vrooli scenario start` is idempotent but SLOW. When a
// Ports resolver is wired, EnsureRunning first does a cheap TCP dial to the
// scenario's *current* fixed UI port (from service.json). A successful dial
// means the process is already serving on the right port, so it skips.
// If the dial fails (e.g. first expose after TM pinned a fixed port for a
// previously ranged scenario, or stale registry port), it forces stop+start
// so the lifecycle binds the declared port. Only cold or port-changed cases pay
// the (necessary) start cost.
type CLIRunner struct {
	Runner cmdrunner.Runner
	// Ports resolves the scenario's fixed UI port for the already-running
	// probe. Optional: when nil, EnsureRunning always shells start (old
	// behaviour) — correctness is preserved, only the fast-path is disabled.
	Ports PortResolver
	// Dial probes whether the UI port is already accepting connections.
	// Optional: defaults to a 300ms TCP dial. Injected in tests.
	Dial func(ctx context.Context, port int) bool
}

// NewCLIRunner constructs the production Runner with the already-running
// fast-path enabled via the supplied PortResolver.
func NewCLIRunner(runner cmdrunner.Runner, ports PortResolver) *CLIRunner {
	return &CLIRunner{Runner: runner, Ports: ports}
}

var _ Runner = (*CLIRunner)(nil)

func (r *CLIRunner) EnsureRunning(ctx context.Context, scenario string) error {
	if r.alreadyRunning(ctx, scenario) {
		return nil
	}
	// The target fixed port (from service.json after any recent TM pin) is not
	// listening. Force a clean stop+start cycle. This ensures lifecycle
	// re-reads the (possibly just updated) service.json and binds the correct
	// fixed port. Common on first `exposure expose` for ranged scenarios.
	// Stop is always safe (idempotent). We only pay this cost when the dial
	// fast-path fails.
	_, _ = r.Runner(ctx, "vrooli", "scenario", "stop", scenario)
	if _, err := r.Runner(ctx, "vrooli", "scenario", "start", scenario); err != nil {
		return fmt.Errorf("ensure %q running: %w", scenario, err)
	}
	return nil
}

// alreadyRunning reports whether the scenario's fixed UI port is already
// accepting connections, so EnsureRunning can skip the slow start shell. It is
// best-effort: a ranged scenario (no fixed port) or an unwired resolver returns
// false, falling back to the (idempotent) start.
func (r *CLIRunner) alreadyRunning(ctx context.Context, scenario string) bool {
	if r.Ports == nil {
		return false
	}
	port, err := r.Ports.UIPort(ctx, scenario)
	if err != nil || port <= 0 {
		return false
	}
	dial := r.Dial
	if dial == nil {
		dial = dialPort
	}
	return dial(ctx, port)
}

// dialPort is the production probe: a short TCP dial to localhost:<port>.
func dialPort(ctx context.Context, port int) bool {
	d := net.Dialer{Timeout: 300 * time.Millisecond}
	dctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	conn, err := d.DialContext(dctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
