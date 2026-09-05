package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/repo-contract-go/cliinvoke"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Exit codes. The scheduler unit reads them: 0 is a clean stop, everything
// else is a failure it restarts and, past its burst limit, escalates.
const (
	// exitSignal: SIGINT or SIGTERM was received and honoured.
	exitSignal = 0
	// exitNoRoot: the repository root could not be resolved.
	exitNoRoot = 2
	// exitNonHealable: three consecutive failures the loop cannot heal by
	// retrying (usage error, missing binary, failed preflight). The status
	// file is written first so the escalation target can read why.
	exitNonHealable = 3
	// exitStateUnwritable: the state directory cannot be created or written.
	exitStateUnwritable = 4
)

// nonHealableExitThreshold is how many consecutive non-healable failures
// turn the loop's failure into the scheduler's.
const nonHealableExitThreshold = 3

// Config holds loop configuration.
type Config struct {
	// TickInterval is the health tick period and the pause between
	// non-healable heal attempts.
	TickInterval time.Duration
	// MaxFailures is how many consecutive tick failures it takes before the
	// loop asks whether the API is still alive at all.
	MaxFailures int
	// StartupTimeout bounds waitForAPIHealthy after a lifecycle command.
	StartupTimeout time.Duration
	// HealthCheckInterval is the poll period inside waitForAPIHealthy.
	HealthCheckInterval time.Duration
	VrooliRoot          string
	ScenarioName        string
	// FixedBaseURL, when set by --api-url, replaces port detection.
	FixedBaseURL string
	// HealthEndpoint and TickEndpoint are derived from the adopted port or
	// FixedBaseURL; setBaseURL is the only writer.
	HealthEndpoint     string
	TickEndpoint       string
	ManageAPILifecycle bool
	// APIPort is the identity-verified port currently in use; LastKnownPort
	// survives a loss of the API so detection can try it first.
	APIPort       string
	LastKnownPort string
	// VrooliCmdPath is the resolved CLI, empty when resolution failed;
	// VrooliResolveErr says why.
	VrooliCmdPath    string
	VrooliResolveErr error
	// ProbePorts are the historical allocations tried as a last resort.
	ProbePorts []int
}

// defaultProbePorts are the ports vrooli has historically allocated to the
// autoheal API: the historical defaults, the start of the 15000-19999
// scenario range, and its middle.
var defaultProbePorts = []int{
	19761, 19762, 19763, 19764, 19765,
	15000, 15001, 15002, 15003, 15004,
	18000, 18001, 18002, 18003, 18004,
}

// options is what the command line produced.
type options struct {
	config        *Config
	installTarget string
	selfTest      bool
	vrooliBin     string
}

// parseFlags reads the command line. The --interval, --max-failures,
// --vrooli-bin and --install-self flags are part of the unit and safeguard
// contract and must keep their names and meanings.
func parseFlags(args []string) (options, error) {
	fs := flag.NewFlagSet("vrooli-autoheal-loop", flag.ContinueOnError)
	interval := fs.Int("interval", 60, "Tick interval in seconds")
	apiURL := fs.String("api-url", "", "API base URL (auto-detected if not specified)")
	maxFailures := fs.Int("max-failures", 3, "Consecutive tick failures before the API's liveness is questioned")
	noManageAPI := fs.Bool("no-manage-api", false, "Disable API lifecycle management")
	vrooliBin := fs.String("vrooli-bin", "", "Path to the vrooli CLI to use for lifecycle operations")
	installTarget := fs.String("install-self", "", "Atomically install this executable at the target path")
	selfTest := fs.Bool("self-test", false, "Run the preflight, print its JSON result, and exit 0 (ok) or 3 (failed)")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if *apiURL != "" {
		if err := validateLocalEndpoint(*apiURL); err != nil {
			return options{}, fmt.Errorf("invalid --api-url: %w", err)
		}
	}
	return options{
		config: &Config{
			TickInterval:        time.Duration(*interval) * time.Second,
			MaxFailures:         *maxFailures,
			StartupTimeout:      120 * time.Second,
			HealthCheckInterval: 5 * time.Second,
			ScenarioName:        "vrooli-autoheal",
			FixedBaseURL:        strings.TrimRight(*apiURL, "/"),
			ManageAPILifecycle:  !*noManageAPI,
			ProbePorts:          defaultProbePorts,
		},
		installTarget: *installTarget,
		selfTest:      *selfTest,
		vrooliBin:     strings.TrimSpace(*vrooliBin),
	}, nil
}

// setBaseURL is the one place the HTTP endpoints are derived.
func (config *Config) setBaseURL(base string) {
	config.HealthEndpoint = base + "/health"
	config.TickEndpoint = base + "/api/v1/tick"
}

// runtimeHomeEntry resolves operator runtime paths through the repository
// contract. Callers must not construct home/.vrooli paths themselves.
func runtimeHomeEntry(key string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	path, err := repocontract.RuntimeHomeEntryPath(home, key)
	if err != nil {
		return ""
	}
	return path
}

func resolveVrooliRoot() string {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}

// resolveVrooliBinary resolves the CLI once at start through the one invoker
// seam: explicit flag, VROOLI_BIN, the runtime home's bin entry, then PATH. A
// miss leaves the path empty and the preflight names it.
func (config *Config) resolveVrooliBinary(explicit string) {
	home, _ := os.UserHomeDir()
	path, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{Explicit: explicit, RuntimeHome: home})
	config.VrooliCmdPath, config.VrooliResolveErr = path, err
}

// sleepCtx waits for d or until ctx is done, reporting whether the full
// duration elapsed. Every wait in the loop goes through it so a shutdown
// signal is never delayed by a timer.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// validateLocalEndpoint keeps the watchdog's outbound HTTP surface bound to
// the local autoheal API. The endpoint is configurable for tests and local
// proxies, but a remote or user-info-bearing URL must never become a server-
// side request target.
func validateLocalEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("endpoint scheme must be http or https")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("endpoint host is required")
	}
	if parsed.User != nil {
		return fmt.Errorf("endpoint user-info is not allowed")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("endpoint host %q is not local", parsed.Hostname())
	}
	return nil
}

func localHealthEndpoint(rawPort string) (string, error) {
	port, err := strconv.Atoi(strings.TrimSpace(rawPort))
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("invalid local API port %q", rawPort)
	}
	endpoint := "http://localhost:" + strconv.Itoa(port) + "/health"
	if err := validateLocalEndpoint(endpoint); err != nil {
		return "", err
	}
	return endpoint, nil
}
