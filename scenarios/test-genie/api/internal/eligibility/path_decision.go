package eligibility

// Path identifies which path the playbooks phase chose for a run.
//
// PathRouted means the in-place routed e2e path was taken. Every other value
// records *why* the destructive playbooks were refused. There is no
// restart-based fallback — playbooks is routed-or-refuse.
type Path string

const (
	// PathRouted: the scenario qualified and the routed e2e path ran.
	PathRouted Path = "routed"

	// PathRefusedIsolation: storage-health could not statically prove test-DB
	// isolation (ROUTED_SEAMS_UNWIRED, or STORAGE_ISOLATION_UNVERIFIED for a
	// non-Go API). BlockingFindings is populated.
	PathRefusedIsolation Path = "refused_isolation"

	// PathRefusedProviderUnreachable: the storage-health validation could not be
	// completed (provider down, network error). "Isolation cannot be verified"
	// is treated as "not certifiably isolated" — fail closed.
	PathRefusedProviderUnreachable Path = "refused_provider_unreachable"

	// PathRefusedPreflight: eligibility passed but a routed-path pre-flight check
	// failed (DSN extraction or routing-service client resolution).
	// PreflightFailure identifies which one.
	PathRefusedPreflight Path = "refused_preflight"

	// PathRefusedProductionMode: the target scenario is running in production
	// mode (or otherwise has the dev-only RoutingService disabled); the routed
	// surface is not available, so destructive playbooks cannot be isolated.
	PathRefusedProductionMode Path = "refused_production_mode"
)

// PreflightFailure identifies which routed pre-flight check failed.
type PreflightFailure string

const (
	PreflightFailureNone               PreflightFailure = ""
	PreflightFailureNoDSN              PreflightFailure = "no_test_dsn"
	PreflightFailureRoutingUnreachable PreflightFailure = "routing_unreachable"
)

// PathDecision is the consolidated record of which path the playbooks phase
// took for a given run and why. It is the single source of truth for the
// structured log block emitted at the top of each playbooks run.
type PathDecision struct {
	Path             Path
	Reason           string
	BlockingFindings []IsolationFinding
	// Unverified mirrors Eligibility.Unverified for PathRefusedIsolation: the
	// disqualification was a non-Go STORAGE_ISOLATION_UNVERIFIED fail-safe.
	Unverified       bool
	PreflightFailure PreflightFailure

	// RoutingBaseURL, LeaseID, DSNDriver are populated for routed runs so the
	// log block can name the active install.
	RoutingBaseURL string
	LeaseID        string
	DSNDriver      string
}

// IsRouted reports whether the playbooks phase will execute the in-place routed
// path for this decision.
func (d PathDecision) IsRouted() bool { return d.Path == PathRouted }

// IsRefused reports whether destructive playbooks were refused.
func (d PathDecision) IsRefused() bool { return d.Path != PathRouted }
