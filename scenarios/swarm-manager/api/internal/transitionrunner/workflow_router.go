package transitionrunner

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"swarm-manager/internal/agentmanager"

	"google.golang.org/protobuf/types/known/structpb"
)

// WorkflowRouter selects an Agent Manager transport by declared workflow key.
// It lets a composition root use specialized transports in tests without
// leaking workflow lifecycle calls back into subject packages.
type WorkflowRouter struct {
	defaultInvoker agentmanager.WorkflowInvoker
	byKey          map[string]agentmanager.WorkflowInvoker
	mu             sync.RWMutex
	byExecution    map[string]agentmanager.WorkflowInvoker
}

func NewWorkflowRouter(defaultInvoker agentmanager.WorkflowInvoker, routes map[string]agentmanager.WorkflowInvoker) *WorkflowRouter {
	copyRoutes := make(map[string]agentmanager.WorkflowInvoker, len(routes))
	for key, invoker := range routes {
		if strings.TrimSpace(key) != "" && invoker != nil {
			copyRoutes[key] = invoker
		}
	}
	return &WorkflowRouter{defaultInvoker: defaultInvoker, byKey: copyRoutes, byExecution: map[string]agentmanager.WorkflowInvoker{}}
}

func (r *WorkflowRouter) StartWorkflow(ctx context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	invoker, err := r.forKey(invocation.WorkflowKey)
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	started, err := invoker.StartWorkflow(ctx, invocation)
	if err == nil && strings.TrimSpace(started.ExecutionID) != "" {
		r.mu.Lock()
		r.byExecution[started.ExecutionID] = invoker
		r.mu.Unlock()
	}
	return started, err
}

func (r *WorkflowRouter) CollectWorkflow(ctx context.Context, executionID string) (agentmanager.InvocationCompletion, error) {
	r.mu.RLock()
	invoker := r.byExecution[executionID]
	r.mu.RUnlock()
	if invoker != nil {
		return invoker.CollectWorkflow(ctx, executionID)
	}
	if r.defaultInvoker == nil {
		return agentmanager.InvocationCompletion{}, agentmanager.ErrNotAvailable
	}
	return r.defaultInvoker.CollectWorkflow(ctx, executionID)
}

func (r *WorkflowRouter) SignalWorkflow(ctx context.Context, executionID, signal string, payload *structpb.Value, idempotencyKey string) error {
	r.mu.RLock()
	invoker := r.byExecution[executionID]
	r.mu.RUnlock()
	if signaler, ok := invoker.(interface {
		SignalWorkflow(context.Context, string, string, *structpb.Value, string) error
	}); ok {
		return signaler.SignalWorkflow(ctx, executionID, signal, payload, idempotencyKey)
	}
	return fmt.Errorf("workflow signaling is not supported")
}

// CancelWorkflow mirrors SignalWorkflow: route to the transport that started
// this execution, falling back to the default. Without it the router silently
// dropped cancellation for every routed transition.
func (r *WorkflowRouter) CancelWorkflow(ctx context.Context, executionID, idempotencyKey, reason string) error {
	r.mu.RLock()
	invoker := r.byExecution[executionID]
	r.mu.RUnlock()
	if invoker == nil {
		invoker = r.defaultInvoker
	}
	if canceler, ok := invoker.(interface {
		CancelWorkflow(context.Context, string, string, string) error
	}); ok {
		return canceler.CancelWorkflow(ctx, executionID, idempotencyKey, reason)
	}
	return fmt.Errorf("workflow cancellation is not supported")
}

func (r *WorkflowRouter) forKey(key string) (agentmanager.WorkflowInvoker, error) {
	if invoker := r.byKey[key]; invoker != nil {
		return invoker, nil
	}
	if r.defaultInvoker != nil {
		return r.defaultInvoker, nil
	}
	return nil, fmt.Errorf("workflow transport for %q is not configured", key)
}
