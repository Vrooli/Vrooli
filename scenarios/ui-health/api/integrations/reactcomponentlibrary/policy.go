package reactcomponentlibrary

import "time"

// Policy is the retry/timeout/required-vs-optional policy the integration
// adapter enforces (interop-steer §12).
type Policy struct {
	// PerCallTimeout bounds any single ScanScenario call.
	PerCallTimeout time.Duration
	// MaxRetries is the bounded retry count on transport failure.
	MaxRetries int
	// Required gates fail-fast behavior. ui-health's RCL dependency is
	// optional (degraded_behavior: "React scenarios are skipped during
	// reindex"); set Required=false in production.
	Required bool
}

// DefaultPolicy returns the policy ui-health uses in production: optional
// dependency, 30s per call, 1 retry after re-resolution.
func DefaultPolicy() Policy {
	return Policy{
		PerCallTimeout: 30 * time.Second,
		MaxRetries:     1,
		Required:       false,
	}
}
