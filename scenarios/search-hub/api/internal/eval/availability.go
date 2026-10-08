package eval

import (
	"context"
	"errors"
	"strings"
	"time"
)

type caseTimeoutKey struct{}

// WithCaseTimeout lets the scheduler set a per-provider-call budget without
// changing the runner seam used by hand-run evaluations and tests.
func WithCaseTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, caseTimeoutKey{}, timeout)
}

func caseTimeout(ctx context.Context) time.Duration {
	if timeout, ok := ctx.Value(caseTimeoutKey{}).(time.Duration); ok && timeout > 0 {
		return timeout
	}
	return 0
}

// unavailableError identifies failures that say something about the serving
// substrate rather than retrieval quality. Keep this deliberately conservative:
// ordinary provider/application errors remain visible as error cases.
func unavailableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	reason := strings.ToLower(err.Error())
	for _, marker := range []string{
		"connection refused",
		"connection reset",
		"no such host",
		"network is unreachable",
		"provider unreachable",
		"http 500",
		"http 501",
		"http 502",
		"http 503",
		"http 504",
		"http 505",
		"http 506",
		"http 507",
		"http 508",
		"http 509",
		"http 510",
		"http 511",
	} {
		if strings.Contains(reason, marker) {
			return true
		}
	}
	return false
}
