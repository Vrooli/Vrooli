package aimoneyfixtures

import (
	"context"
	"errors"
	"sync"
)

// FakeGatewayPrivate is a deterministic in-memory private gateway. It refuses
// execution without the server-controlled funded marker and memoizes by
// operation ID so retries cannot execute twice.
type FakeGatewayPrivate struct {
	mu          sync.Mutex
	Plans       []GatewayPlanRequest
	Executes    []GatewayExecuteRequest
	Receipts    []GatewayReceiptRequest
	PlanValue   GatewayPlan
	Result      GatewayResult
	PlanErr     error
	ExecuteErr  error
	ReceiptErr  error
	byOperation map[string]GatewayReceipt
}

func (f *FakeGatewayPrivate) Plan(_ context.Context, req GatewayPlanRequest) (GatewayPlan, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Plans = append(f.Plans, req)
	if f.PlanErr != nil {
		return GatewayPlan{}, f.PlanErr
	}
	return f.PlanValue, nil
}

func (f *FakeGatewayPrivate) Execute(_ context.Context, req GatewayExecuteRequest) (GatewayResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !req.Funded {
		return GatewayResult{}, errors.New("funded marker required")
	}
	if f.ExecuteErr != nil {
		return GatewayResult{}, f.ExecuteErr
	}
	if f.byOperation == nil {
		f.byOperation = make(map[string]GatewayReceipt)
	}
	if receipt, ok := f.byOperation[req.OperationID]; ok {
		return receipt.Result, nil
	}
	f.Executes = append(f.Executes, req)
	f.byOperation[req.OperationID] = GatewayReceipt{OperationID: req.OperationID, State: "complete", Result: f.Result}
	return f.Result, nil
}

func (f *FakeGatewayPrivate) Receipt(_ context.Context, req GatewayReceiptRequest) (GatewayReceipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Receipts = append(f.Receipts, req)
	if f.ReceiptErr != nil {
		return GatewayReceipt{}, f.ReceiptErr
	}
	if f.byOperation == nil {
		return GatewayReceipt{}, errors.New("receipt not found")
	}
	r, ok := f.byOperation[req.OperationID]
	if !ok {
		return GatewayReceipt{}, errors.New("receipt not found")
	}
	return r, nil
}
