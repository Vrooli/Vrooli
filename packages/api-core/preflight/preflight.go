// Package preflight provides startup checks for Vrooli scenario APIs.
//
// Preflight combines multiple startup checks that must run before any API
// initialization. It handles:
//   - Staleness detection and auto-rebuild
//   - Lifecycle management verification
//
// Usage:
//
//	func main() {
//	    if preflight.Run(preflight.Config{ScenarioName: "my-scenario"}) {
//	        return // Process was re-exec'd after rebuild
//	    }
//	    // Safe to initialize server, DB connections, etc.
//	}
package preflight

import (
	"fmt"
	"os"

	"github.com/vrooli/api-core/staleness"
)

// Environment variable that indicates the API is managed by the lifecycle system
const LifecycleManagedEnvVar = "VROOLI_LIFECYCLE_MANAGED"

// Config configures the preflight checks.
type Config struct {
	// ScenarioName is used in error messages to guide users.
	// Example: "my-scenario" produces "vrooli scenario start my-scenario"
	ScenarioName string

	// DisableStaleness skips the staleness check entirely.
	// Useful in production where rebuilds aren't desired.
	DisableStaleness bool

	// SkipRebuild detects staleness but doesn't rebuild.
	// Useful for debugging or when rebuild should be handled externally.
	SkipRebuild bool

	// DisableLifecycleGuard skips the lifecycle management check.
	// Useful for testing or development outside the lifecycle system.
	DisableLifecycleGuard bool

	// Logger overrides the default stderr logger.
	Logger func(format string, args ...interface{})

	// StalenessConfig provides additional staleness checker configuration.
	// Most scenarios can leave this nil for defaults.
	StalenessConfig *staleness.CheckerConfig
}

// Run executes all preflight checks in the correct order.
//
// Returns true if the process was re-exec'd after a rebuild. In this case,
// the caller should return immediately from main() as the new process has
// already started.
//
// If the lifecycle guard fails, Run exits the process with an error message
// guiding the user to use the lifecycle system.
//
// Check order:
//  1. Staleness check (may trigger rebuild and re-exec)
//  2. Lifecycle guard (may exit with error)
//
// This order is critical: staleness check preserves environment variables
// during re-exec, so VROOLI_LIFECYCLE_MANAGED remains set.
func Run(cfg Config) bool {
	// 1. Staleness check FIRST
	// This may rebuild the binary and re-exec, preserving all env vars
	if !cfg.DisableStaleness {
		staleCfg := staleness.CheckerConfig{
			SkipRebuild: cfg.SkipRebuild,
			Logger:      cfg.Logger,
		}

		// Merge any custom staleness config
		if cfg.StalenessConfig != nil {
			if cfg.StalenessConfig.APIDir != "" {
				staleCfg.APIDir = cfg.StalenessConfig.APIDir
			}
			if cfg.StalenessConfig.BinaryPath != "" {
				staleCfg.BinaryPath = cfg.StalenessConfig.BinaryPath
			}
			if cfg.StalenessConfig.CommandRunner != nil {
				staleCfg.CommandRunner = cfg.StalenessConfig.CommandRunner
			}
			if cfg.StalenessConfig.Reexec != nil {
				staleCfg.Reexec = cfg.StalenessConfig.Reexec
			}
			if cfg.StalenessConfig.LookPath != nil {
				staleCfg.LookPath = cfg.StalenessConfig.LookPath
			}
		}

		checker := staleness.NewChecker(staleCfg)
		if checker.CheckAndMaybeRebuild() {
			return true // Process was re-exec'd
		}
	}

	// 2. Lifecycle guard SECOND
	// Check that we're running under the lifecycle system
	if !cfg.DisableLifecycleGuard {
		if os.Getenv(LifecycleManagedEnvVar) != "true" {
			scenarioName := cfg.ScenarioName
			if scenarioName == "" {
				scenarioName = "<scenario-name>"
			}

			msg := fmt.Sprintf(`This binary must be run through the Vrooli lifecycle system.

Instead, use:
   vrooli scenario start %s

The lifecycle system provides environment variables, port allocation,
and dependency management automatically. Direct execution is not supported.
`, scenarioName)

			if cfg.Logger != nil {
				cfg.Logger("%s", msg)
			} else {
				fmt.Fprint(os.Stderr, msg)
			}
			os.Exit(1)
		}
	}

	return false // All checks passed
}

// MustRun is like Run but panics instead of returning on re-exec.
// This is a convenience for scenarios that prefer panic-based flow control.
//
// Usage:
//
//	func main() {
//	    preflight.MustRun(preflight.Config{ScenarioName: "my-scenario"})
//	    // Only reached if all checks pass and no re-exec occurred
//	}
func MustRun(cfg Config) {
	if Run(cfg) {
		// This shouldn't be reached because re-exec replaces the process,
		// but just in case the re-exec fails silently:
		panic("preflight: unexpected return after re-exec")
	}
}
