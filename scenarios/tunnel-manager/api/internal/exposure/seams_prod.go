package exposure

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"tunnel-manager/internal/cmdrunner"
)

// FilePortResolver resolves a scenario's fixed UI port from its
// service.json (<root>/<scenario>/.vrooli/service.json, ports.ui.port).
// It is the production PortResolver; tests use a fake.
type FilePortResolver struct {
	Root string
}

// NewFilePortResolver constructs a resolver rooted at the scenarios dir.
func NewFilePortResolver(root string) *FilePortResolver {
	return &FilePortResolver{Root: root}
}

var _ PortResolver = (*FilePortResolver)(nil)

// serviceJSON is the minimal slice of service.json this domain reads. A
// fixed UI port lives at ports.ui.port; ranged/dynamic scenarios omit it.
type serviceJSON struct {
	Ports map[string]struct {
		Port int `json:"port"`
	} `json:"ports"`
}

func (r *FilePortResolver) UIPort(_ context.Context, scenario string) (int, error) {
	path := filepath.Join(r.Root, scenario, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, ErrPortUnresolved{Scenario: scenario, Reason: fmt.Sprintf("service.json not readable: %v", err)}
	}
	var svc serviceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		return 0, ErrPortUnresolved{Scenario: scenario, Reason: fmt.Sprintf("service.json parse: %v", err)}
	}
	ui, ok := svc.Ports["ui"]
	if !ok || ui.Port == 0 {
		return 0, ErrPortUnresolved{Scenario: scenario, Reason: "no fixed UI port declared"}
	}
	return ui.Port, nil
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
