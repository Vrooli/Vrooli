package resources

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/types/known/structpb"

	"test-genie/internal/structure/types"
)

// HealthChecker validates that a scenario's required resources are healthy.
type HealthChecker interface {
	// Check verifies every required resource is running and healthy.
	Check(ctx context.Context) HealthResult
}

// HealthResult represents the outcome of health checking.
type HealthResult struct {
	// Success indicates whether all required resources are healthy.
	Success bool

	// Error contains the validation error, if any.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass types.FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains detailed observations.
	Observations []types.Observation
}

// ResourceStatusFetcher abstracts live resource-health retrieval — the typed
// `vrooli resource status --json` contract — so the checker is unit-testable.
// *vroolicli.Client satisfies this directly.
type ResourceStatusFetcher interface {
	ResourceStatuses(ctx context.Context) (*cliv1.ResourceStatusesResponse, error)
}

// checker is the default implementation of HealthChecker. It cross-references
// the scenario's declared required resources (from its service manifest)
// against live health reported by the Vrooli CLI.
type checker struct {
	requiredResources []string
	fetcher           ResourceStatusFetcher
	logWriter         io.Writer
}

// NewChecker creates a resource health checker. requiredResources is the set of
// resource names the scenario declares as required; fetcher supplies live
// resource health from the Vrooli CLI.
func NewChecker(requiredResources []string, fetcher ResourceStatusFetcher, logWriter io.Writer) HealthChecker {
	return &checker{
		requiredResources: requiredResources,
		fetcher:           fetcher,
		logWriter:         logWriter,
	}
}

// Check implements HealthChecker.
func (c *checker) Check(ctx context.Context) HealthResult {
	if len(c.requiredResources) == 0 {
		return HealthResult{
			Success: true,
			Observations: []types.Observation{
				types.NewInfoObservation("scenario declares no required resources; skipping resource health check"),
			},
		}
	}

	resp, err := c.fetcher.ResourceStatuses(ctx)
	if err != nil {
		// A read failure is a real gate failure: we cannot confirm that the
		// scenario's required resources are up, so we must not pass. (This is the
		// correctness fix — the previous implementation parsed scenario-status
		// fields the CLI no longer emits and silently always passed.)
		c.logWarn("resource status unavailable: %v", err)
		return HealthResult{
			Success:      false,
			Error:        fmt.Errorf("unable to read resource status: %w", err),
			FailureClass: types.FailureClass("missing_dependency"),
			Remediation:  "Ensure the vrooli CLI is available and resources are installed, then rerun the dependency phase.",
		}
	}

	byName := make(map[string]*cliv1.ResourceStatus, len(resp.GetResources()))
	for _, rs := range resp.GetResources() {
		if name := rs.GetResource().GetName(); name != "" {
			byName[name] = rs
		}
	}

	var observations []types.Observation
	var failures []string
	for _, name := range c.requiredResources {
		rs, ok := byName[name]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s (not found)", name))
			continue
		}

		running := rs.GetRunning()
		healthy, known := resourceHealthy(rs)
		switch {
		case !running:
			failures = append(failures, fmt.Sprintf("%s (running=false)", name))
		case known && !healthy:
			failures = append(failures, fmt.Sprintf("%s (running=true healthy=false)", name))
		case !known:
			observations = append(observations, types.NewSuccessObservation(
				fmt.Sprintf("resource running (health not probed): %s", name)))
		default:
			observations = append(observations, types.NewSuccessObservation(
				fmt.Sprintf("resource healthy: %s", name)))
		}
	}

	if len(failures) > 0 {
		sort.Strings(failures)
		return HealthResult{
			Success:      false,
			Error:        fmt.Errorf("required resources unhealthy: %s", strings.Join(failures, ", ")),
			FailureClass: types.FailureClass("missing_dependency"),
			Remediation:  "Start the missing resources (see `vrooli resource status`) or restart the scenario before rerunning tests.",
			Observations: observations,
		}
	}

	return HealthResult{
		Success:      true,
		Observations: observations,
	}
}

// resourceHealthy reads the tri-state `healthy` probe from a resource status
// (a bool when the probe was evaluated, null when it was not). known is false
// when the resource has no evaluated health probe, in which case the caller
// treats a running resource as acceptable rather than failing on absent data.
func resourceHealthy(rs *cliv1.ResourceStatus) (healthy bool, known bool) {
	v := rs.GetHealthy()
	if v == nil {
		return false, false
	}
	if b, ok := v.GetKind().(*structpb.Value_BoolValue); ok {
		return b.BoolValue, true
	}
	return false, false
}

// logWarn writes a warning message to the log.
func (c *checker) logWarn(format string, args ...interface{}) {
	if c.logWriter == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(c.logWriter, "[WARNING] %s\n", msg)
}

var _ HealthChecker = (*checker)(nil)
