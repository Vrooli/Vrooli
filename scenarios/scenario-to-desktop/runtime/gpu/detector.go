// Package gpu provides GPU detection and requirement enforcement for the bundle runtime.
package gpu

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
)

// Status holds the result of GPU detection.
type Status struct {
	Available bool   // Whether a usable GPU was detected
	Method    string // Detection method used (env_override, nvidia-smi, system_profiler, wmic, probe)
	Reason    string // Human-readable explanation
	Backends  []string
	Facts     map[string]string
}

// Detector abstracts GPU detection for testing.
type Detector interface {
	// Detect probes the system for GPU availability.
	Detect() Status
}

// RealDetector implements Detector using system commands.
type RealDetector struct {
	CommandRunner infra.CommandRunner
	EnvReader     infra.EnvReader
	Collect       func(context.Context) (hostinventory.Snapshot, error)
}

// NewDetector creates a new RealDetector with the given dependencies.
func NewDetector(cmdRunner infra.CommandRunner, envReader infra.EnvReader) *RealDetector {
	return &RealDetector{
		CommandRunner: cmdRunner,
		EnvReader:     envReader,
	}
}

// Detect implements Detector interface.
// It uses the shared host inventory authority for GPU availability.
// Detection can be overridden via the BUNDLE_GPU_AVAILABLE environment variable.
func (d *RealDetector) Detect() Status {
	// Allow environment override for testing or forcing behavior.
	override := strings.TrimSpace(d.EnvReader.Getenv("BUNDLE_GPU_AVAILABLE"))
	switch strings.ToLower(override) {
	case "1", "true", "yes", "on":
		return Status{Available: true, Method: "env_override", Reason: "forced available via BUNDLE_GPU_AVAILABLE", Backends: []string{"gpu"}}
	case "0", "false", "no", "off":
		return Status{Available: false, Method: "env_override", Reason: "forced unavailable via BUNDLE_GPU_AVAILABLE"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	snapshot, err := d.collectSnapshot(ctx)
	if err != nil {
		return Status{Available: false, Method: "hostinventory", Reason: "host inventory collection failed"}
	}
	return statusFromSnapshot(snapshot)
}

func (d *RealDetector) collectSnapshot(ctx context.Context) (hostinventory.Snapshot, error) {
	if d.Collect != nil {
		return d.Collect(ctx)
	}
	collector := hostinventory.SystemCollector()
	collector.Commands = inventoryCommandRunner{runner: d.CommandRunner}
	return collector.Collect(ctx)
}

func statusFromSnapshot(snapshot hostinventory.Snapshot) Status {
	facts := snapshot.AcceleratorFacts()
	backends := strings.Split(facts[hostinventory.FactAccelBackends], ",")
	if len(snapshot.GPUs) > 0 {
		method := snapshot.GPUs[0].Source
		if method == "" {
			method = "hostinventory"
		}
		return Status{Available: len(snapshot.GPUs) > 0, Method: method, Reason: gpuReason(method), Backends: backends, Facts: facts}
	}
	if len(facts) > 0 {
		return Status{Available: false, Method: "probe", Reason: "no GPU detected", Backends: backends, Facts: facts}
	}
	return Status{Available: false, Method: "probe", Reason: "no GPU detected"}
}

func gpuReason(method string) string {
	switch method {
	case "nvidia-smi":
		return "nvidia gpu detected"
	case "system_profiler":
		return "GPU reported by system_profiler"
	case "wmic":
		return "GPU reported by wmic"
	case "darwin-unified-memory":
		return "Apple unified-memory GPU inferred"
	default:
		return "GPU reported by host inventory"
	}
}

type inventoryCommandRunner struct {
	runner infra.CommandRunner
}

func (r inventoryCommandRunner) LookPath(file string) (string, error) {
	return r.runner.LookPath(file)
}

func (r inventoryCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.runner.Output(ctx, name, args...)
}

// Ensure RealDetector implements Detector.
var _ Detector = (*RealDetector)(nil)
