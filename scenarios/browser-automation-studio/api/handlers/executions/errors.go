package executions

import "errors"

var (
	errInvalidExecutionID   = errors.New("invalid execution id")
	errExecutionNotFound    = errors.New("execution not found")
	errMissingCleanupToken  = errors.New("cleanup_token is required")
	errSeedSchedulerMissing = errors.New("seed cleanup manager unavailable")
)
