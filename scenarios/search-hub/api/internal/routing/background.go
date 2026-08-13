package routing

import "context"

type (
	backgroundEvaluationContextKey  struct{}
	backgroundEvaluationProviderKey struct{}
	recoveryProbeContextKey         struct{}
	failureRecoveryProbeContextKey  struct{}
)

// WithBackgroundEvaluation marks a query issued by an unattended evaluator.
// The router uses this internal scheduling signal to avoid competing with
// interactive traffic for classifier/reranker capacity.
func WithBackgroundEvaluation(ctx context.Context) context.Context {
	return context.WithValue(ctx, backgroundEvaluationContextKey{}, true)
}

// WithBackgroundEvaluationProvider narrows an evaluator query to the suite's
// registered provider. This keeps an unavailable sibling provider from
// stretching the scheduler's cycle or competing with interactive traffic.
func WithBackgroundEvaluationProvider(ctx context.Context, providerID string) context.Context {
	return context.WithValue(WithBackgroundEvaluation(ctx), backgroundEvaluationProviderKey{}, providerID)
}

// WithRecoveryProbe marks a bounded, unattended request that is testing a
// provider after its zero-yield decay window. It still uses the explicit
// provider-targeting path, but its result is recorded as automatic recovery
// evidence rather than operator-selected traffic.
func WithRecoveryProbe(ctx context.Context) context.Context {
	return context.WithValue(WithBackgroundEvaluation(ctx), recoveryProbeContextKey{}, true)
}

// WithFailureRecoveryProbe marks a probe that already claimed an elapsed
// transport-breaker cooldown. It bypasses normal breaker admission once;
// the result still closes or reopens that breaker through normal accounting.
func WithFailureRecoveryProbe(ctx context.Context) context.Context {
	return context.WithValue(WithRecoveryProbe(ctx), failureRecoveryProbeContextKey{}, true)
}

func isBackgroundEvaluation(ctx context.Context) bool {
	value, _ := ctx.Value(backgroundEvaluationContextKey{}).(bool)
	return value
}

func backgroundEvaluationProvider(ctx context.Context) string {
	value, _ := ctx.Value(backgroundEvaluationProviderKey{}).(string)
	return value
}

func isRecoveryProbe(ctx context.Context) bool {
	value, _ := ctx.Value(recoveryProbeContextKey{}).(bool)
	return value
}

func isFailureRecoveryProbe(ctx context.Context) bool {
	value, _ := ctx.Value(failureRecoveryProbeContextKey{}).(bool)
	return value
}
