package inference

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/caller"
)

// RouteRecorder writes route events, the gateway's one record of every
// inference request: who asked, for which role, what ran and how long it took.
// The routing repository implements it, so typed inference and ExecuteRoute
// share one log.
type RouteRecorder interface {
	Create(ctx context.Context, ev *routingv1.RouteEvidence) error
}

// ServiceOption configures a Service.
type ServiceOption func(*Service)

// WithRouteRecorder makes the service record a route event for every typed
// request. scenarios names the scenario an observed caller belongs to, since
// typed requests declare none.
func WithRouteRecorder(recorder RouteRecorder, scenarios caller.Scenarios) ServiceOption {
	return func(s *Service) { s.routes, s.scenarios = recorder, scenarios }
}

// Operations recorded for typed inference, one per public RPC.
const (
	OperationRun    = "inference.run"
	OperationEmbed  = "inference.embed"
	OperationDecide = "inference.decide"
)

// callerWait bounds how long a route event waits for its caller to be
// identified. Recording happens after the response, so the wait never delays
// a caller; the bound only matters when /proc is unreadable.
const callerWait = 5 * time.Second

// routeRequest is what a typed request declares about itself.
type routeRequest struct {
	operation, role   string
	profile           sharedv1.Profile
	privacyClass      sharedv1.PrivacyClass
	privacyProvenance string
}

// routeOutcome is what came back, read from the RPC's own response.
type routeOutcome struct {
	provider, model, resourceRole, locality string
	err                                     *inferencev1.InferenceError
	usage                                   *inferencev1.Usage
}

// beginRoute starts one request's route record. The returned finish writes it
// once the response is known.
func (s *Service) beginRoute(ctx context.Context, request routeRequest) (context.Context, func(routeOutcome)) {
	if s.routes == nil {
		return ctx, func(routeOutcome) {}
	}
	started := time.Now()
	ctx, trace := withRouteTrace(ctx)
	return ctx, func(outcome routeOutcome) {
		ev := routeEvent(request, outcome, trace.snapshot(), started, time.Now())
		// The response is already on its way; the caller's identification and
		// the write finish beside it.
		go s.recordRoute(context.WithoutCancel(ctx), ev)
	}
}

func (s *Service) recordRoute(ctx context.Context, ev *routingv1.RouteEvidence) {
	ev.Caller = caller.Await(ctx, callerWait)
	ev.Scenario = s.scenarios.Of(ev.Caller)
	if err := s.routes.Create(ctx, ev); err != nil {
		log.Printf("typed inference route event %s for caller %s: %v", ev.GetEventId(), ev.GetCaller(), err)
	}
}

// routeEvent builds the event for one typed request. The selected candidate is
// the resource call that last ran, which for a Decide answered by its enum
// fallback is the resource role behind the fallback, not the fallback's name;
// a request answered without a resource call, such as a code role, keeps what
// its response reports. Failure reasons carry
// only the stable error code, never provider text, matching the privacy
// boundary ExecuteRoute keeps.
func routeEvent(request routeRequest, outcome routeOutcome, trace routeTraceSnapshot, started, finished time.Time) *routingv1.RouteEvidence {
	total := max(int64(finished.Sub(started)), 0)
	ev := &routingv1.RouteEvidence{
		EventId:                     newRouteID("rt"),
		RequestId:                   newRouteID("req"),
		Operation:                   request.operation,
		Role:                        request.role,
		TaskRole:                    request.role,
		Profile:                     request.profile,
		PrivacyClass:                request.privacyClass,
		PrivacyProvenance:           request.privacyProvenance,
		SelectedProvider:            firstNonEmpty(trace.provider, outcome.provider),
		SelectedModel:               firstNonEmpty(trace.model, outcome.model),
		SelectedResourceRole:        firstNonEmpty(trace.resourceRole, outcome.resourceRole),
		SelectedLocality:            firstNonEmpty(trace.locality, outcome.locality),
		Status:                      routeStatus(outcome.err, trace.attempts),
		FallbackUsed:                trace.candidates > 1,
		InputTokens:                 outcome.usage.GetInputTokens(),
		OutputTokens:                outcome.usage.GetOutputTokens(),
		CostEstimate:                float64(outcome.usage.GetCostMicros()) / 1e6,
		AttachmentsRedacted:         true,
		LatencyMs:                   total / int64(time.Millisecond),
		TotalDurationNs:             total,
		ProviderExecutionDurationNs: trace.executionNS,
		ProviderRuntimeDurationNs:   trace.runtimeNS,
		ProviderAdmissionWaitNs:     trace.admissionNS,
		CreatedAt:                   started.UTC().Format(time.RFC3339Nano),
	}
	if code := outcome.err.GetCode(); outcome.err != nil {
		ev.FailureReasons = []string{code.String()}
	}
	// Overhead is everything that was neither provider work nor waiting for
	// provider admission, the same split ExecuteRoute records.
	providerNS := trace.executionNS
	if trace.runtimeKnown {
		providerNS = trace.runtimeNS + trace.admissionNS
	}
	ev.OverheadDurationNs = max(total-providerNS, 0)
	return ev
}

// routeStatus uses ExecuteRoute's vocabulary: a request that never reached a
// provider and failed was blocked; one that reached a provider and failed
// failed.
func routeStatus(err *inferencev1.InferenceError, attempts int) string {
	switch {
	case err == nil:
		return "succeeded"
	case attempts == 0:
		return "blocked"
	default:
		return "failed"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func newRouteID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func runOutcome(response *inferencev1.RunResponse) routeOutcome {
	return routeOutcome{provider: response.GetProvider(), model: response.GetModel(), err: response.GetError(), usage: response.GetUsage()}
}

func embedOutcome(response *inferencev1.EmbedResponse) routeOutcome {
	return routeOutcome{provider: response.GetProvider(), model: response.GetModel(), err: response.GetError(), usage: response.GetUsage()}
}

func decideOutcome(response *inferencev1.DecideResponse) routeOutcome {
	return routeOutcome{
		provider: response.GetProvider(), model: response.GetModel(), resourceRole: response.GetResourceRole(),
		locality: response.GetLocality(), err: response.GetError(), usage: response.GetUsage(),
	}
}
