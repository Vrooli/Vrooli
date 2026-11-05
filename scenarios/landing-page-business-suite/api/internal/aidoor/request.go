package aidoor

import (
	"context"
	"errors"
	"fmt"
)

type (
	requestStatus string
	requestEvent  string
)

const (
	requestAccepted    requestStatus = "accepted"
	requestPlanned     requestStatus = "planned"
	requestReserved    requestStatus = "reserved"
	requestStreaming   requestStatus = "streaming"
	requestReconciling requestStatus = "reconciling"
	requestReceipt     requestStatus = "receipt"
	requestSettled     requestStatus = "settled"
	requestFailed      requestStatus = "failed"
)

const (
	requestPlan       requestEvent = "plan"
	requestReserve    requestEvent = "reserve"
	requestDispatch   requestEvent = "dispatch"
	requestRecover    requestEvent = "recover"
	requestDisconnect requestEvent = "disconnect"
	requestReceive    requestEvent = "receive"
	requestCommit     requestEvent = "commit"
	requestDefer      requestEvent = "defer"
	requestReject     requestEvent = "reject"
	requestRelease    requestEvent = "release"
)

// transitionRequest owns request ordering, not durable wallet or Gateway state.
// request.spec.json independently specifies every pair via explicit rows and
// fail-closed defaults; conformance and real journey traces consume that spec.
func transitionRequest(state requestStatus, event requestEvent) (requestStatus, error) {
	switch event {
	case requestPlan:
		if state == requestAccepted {
			return requestPlanned, nil
		}
	case requestReserve:
		if state == requestPlanned {
			return requestReserved, nil
		}
	case requestDispatch:
		if state == requestReserved {
			return requestStreaming, nil
		}
	case requestRecover:
		if state == requestReserved {
			return requestReconciling, nil
		}
	case requestDisconnect:
		if state == requestStreaming {
			return state, nil
		}
	case requestReceive:
		if state == requestStreaming || state == requestReconciling {
			return requestReceipt, nil
		}
	case requestCommit:
		if state == requestReceipt {
			return requestSettled, nil
		}
	case requestDefer:
		if state == requestReserved || state == requestStreaming || state == requestReceipt {
			return requestReconciling, nil
		}
	case requestReject:
		if state == requestAccepted || state == requestPlanned {
			return requestFailed, nil
		}
	case requestRelease:
		if state == requestReserved || state == requestStreaming {
			return requestFailed, nil
		}
	}
	return state, fmt.Errorf("invalid door transition %s/%s", state, event)
}

// RequestTransition is optional telemetry, never an authority or recovery store.
// A trace starts at accepted for a request or reconciling after process recovery.
type RequestTransition struct {
	OperationID     string
	From, Event, To string
	Rejected        bool
}

type requestLifecycle struct {
	door      *Door
	operation string
	state     requestStatus
}

func (d *Door) requestLifecycle(operation string, state requestStatus) *requestLifecycle {
	return &requestLifecycle{door: d, operation: operation, state: state}
}

func (f *requestLifecycle) advance(event requestEvent) error {
	next, err := transitionRequest(f.state, event)
	if f.door.ObserveTransition != nil {
		f.door.ObserveTransition(RequestTransition{f.operation, string(f.state), string(event), string(next), err != nil})
	}
	f.state = next
	return err
}

// Returning without a known outcome leaves the durable hold to reconciliation.
// A request that never acquired a hold is a definitive local rejection.
func (f *requestLifecycle) finish() {
	switch f.state {
	case requestAccepted, requestPlanned:
		_ = f.advance(requestReject)
	case requestReserved, requestStreaming, requestReceipt:
		_ = f.advance(requestDefer)
	}
}

func (f *requestLifecycle) settle(ctx context.Context, key Key, result receipt) error {
	if result.OperationID != f.operation {
		return errors.New("receipt operation mismatch")
	}
	if f.state == requestSettled {
		return nil // A repeated private receipt cannot call the wallet again.
	}
	if !result.CostKnown || (result.State != "completed" && result.State != "unknown" && !(result.State == "failed" && result.Credits == 0)) {
		return errors.New("receipt outcome is unresolved")
	}
	if err := f.advance(requestReceive); err != nil {
		return err
	}
	if err := f.door.settle(ctx, key, result); err != nil {
		_ = f.advance(requestDefer)
		return err
	}
	return f.advance(requestCommit)
}
