package bindings

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	domainconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExerciseObservation is the verified, durable signal used by binding
// conditions. It deliberately contains no local invocation-ledger fields:
// serving telemetry and cross-scenario exercise evidence are separate facts.
type ExerciseObservation struct {
	TargetScenario          string
	Operation               string
	Invocations             int64
	DistinctVerifiedCallers int64
	UnattributedRemainder   int64
	LastInvokedAt           string
}

// ExerciseReader reads the durable receipt projection owned by vrooli-events.
// Implementations must return all operations for a target so the registry can
// map a receipt operation to its manifest binding without guessing a single
// operation at the transport boundary.
type ExerciseReader interface {
	Aggregate(context.Context, string, time.Time, time.Time) ([]ExerciseObservation, error)
}

type receiptExerciseReader struct {
	resolver ReachabilityResolver
	client   connect.HTTPClient
}

// NewReceiptExerciseReader creates the production cross-scenario exercise
// reader. The resolver is injected so tests can use a static URL and so the
// runtime remains portable across lifecycle-managed ports and nodes.
func NewReceiptExerciseReader(resolver ReachabilityResolver, client connect.HTTPClient) ExerciseReader {
	if resolver == nil {
		resolver = ResolveTargetURLDefault
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &receiptExerciseReader{resolver: resolver, client: client}
}

func (r *receiptExerciseReader) Aggregate(ctx context.Context, target string, since, until time.Time) ([]ExerciseObservation, error) {
	baseURL, err := r.resolver(ctx, "vrooli-events")
	if err != nil {
		return nil, fmt.Errorf("resolve vrooli-events: %w", err)
	}
	request := &domain.ReceiptAggregateRequest{
		Since:          timestamppb.New(since.UTC()),
		Until:          timestamppb.New(until.UTC()),
		TargetScenario: target,
	}
	client := domainconnect.NewReceiptAggregateServiceClient(r.client, strings.TrimRight(baseURL, "/"))
	response, err := client.AggregateReceipts(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, fmt.Errorf("aggregate receipts for %s: %w", target, err)
	}
	out := make([]ExerciseObservation, 0, len(response.Msg.GetAggregates()))
	for _, aggregate := range response.Msg.GetAggregates() {
		if aggregate == nil {
			continue
		}
		out = append(out, ExerciseObservation{
			TargetScenario:          aggregate.GetTargetScenario(),
			Operation:               aggregate.GetOperation(),
			Invocations:             aggregate.GetInvocationCount(),
			DistinctVerifiedCallers: aggregate.GetDistinctVerifiedCallers(),
			UnattributedRemainder:   aggregate.GetUnattributedRemainder(),
			LastInvokedAt:           aggregate.GetLastInvokedAt(),
		})
	}
	return out, nil
}

func exerciseMatchesBinding(observation ExerciseObservation, binding interface {
	GetId() string
	GetGroup() string
	GetCommand() string
	GetService() string
	GetMethod() string
}) bool {
	operation := strings.ToLower(strings.TrimSpace(observation.Operation))
	if operation == "" {
		return false
	}
	candidates := []string{
		binding.GetId(),
		binding.GetGroup() + "/" + binding.GetCommand(),
		binding.GetCommand(),
		binding.GetService() + "/" + binding.GetMethod(),
		binding.GetMethod(),
	}
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.Trim(candidate, "/"))
		if candidate == "" || candidate == "/" {
			continue
		}
		if strings.HasSuffix(operation, "/"+candidate) || operation == candidate || strings.HasSuffix(operation, "."+candidate) {
			return true
		}
	}
	return false
}

func exerciseForBinding(binding interface {
	GetId() string
	GetGroup() string
	GetCommand() string
	GetService() string
	GetMethod() string
}, observations []ExerciseObservation, available bool) *bindingsv1.ExerciseCondition {
	if !available {
		return &bindingsv1.ExerciseCondition{Family: &bindingsv1.ConditionFamily{
			Status: bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED,
			Reason: "exercise receipt aggregate unavailable",
		}}
	}
	exercise := &bindingsv1.ExerciseCondition{Family: &bindingsv1.ConditionFamily{
		Status: bindingsv1.ConditionStatus_CONDITION_STATUS_DORMANT,
		Reason: "exercise.invocations=0",
	}}
	for _, observation := range observations {
		if !exerciseMatchesBinding(observation, binding) {
			continue
		}
		exercise.Invocations += observation.Invocations
		exercise.DistinctCallers += observation.DistinctVerifiedCallers
		exercise.UnattributedRemainder += observation.UnattributedRemainder
		if observation.LastInvokedAt > exercise.LastInvokedAt {
			exercise.LastInvokedAt = observation.LastInvokedAt
		}
	}
	if exercise.Invocations > 0 {
		exercise.Family.Status = bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY
		exercise.Family.Reason = "verified receipt exercise observed"
	}
	return exercise
}
