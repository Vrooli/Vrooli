package commerce

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// LocalTestServiceOrderProvider returns deterministic test-mode receipts and
// performs no network or payment operation. Production checkout adapters must
// be separate implementations with their own authorization and refund rules.
type LocalTestServiceOrderProvider struct{}

func NewLocalTestServiceOrderProvider() *LocalTestServiceOrderProvider {
	return &LocalTestServiceOrderProvider{}
}

func (p *LocalTestServiceOrderProvider) Authorize(_ context.Context, request ServiceOrderAuthorizationRequest) (ServiceOrderAuthorizationReceipt, error) {
	if p == nil || !localTestDoor(request.Door) || request.OrderID == "" || request.JobID == "" || request.IdempotencyKey == "" || request.PaymentSignature != "" {
		return ServiceOrderAuthorizationReceipt{}, errors.New("local test payment provider accepts only browser, browsing-agent, or MCP orders without payment signatures")
	}
	identity := stableTestOrderIdentity("authorize", request.IdempotencyKey)
	return ServiceOrderAuthorizationReceipt{
		EventID: "local-test-authorization:" + identity, AuthorizationID: "local-test-auth:" + identity,
		Sandbox: true, Approved: true,
	}, nil
}

func (p *LocalTestServiceOrderProvider) Capture(_ context.Context, request ServiceOrderCaptureRequest) (ServiceOrderCaptureReceipt, error) {
	if p == nil || !localTestDoor(request.Door) || request.OrderID == "" || request.JobID == "" || request.AuthorizationID == "" || request.IdempotencyKey == "" {
		return ServiceOrderCaptureReceipt{}, errors.New("local test payment provider requires a non-x402 order and durable authorization")
	}
	identity := stableTestOrderIdentity("capture", request.IdempotencyKey)
	return ServiceOrderCaptureReceipt{
		EventID: "local-test-capture-event:" + identity, CaptureID: "local-test-capture:" + identity,
		Sandbox: true, Captured: true,
	}, nil
}

func (p *LocalTestServiceOrderProvider) Void(_ context.Context, request ServiceOrderVoidRequest) (ServiceOrderVoidReceipt, error) {
	if p == nil || !localTestDoor(request.Door) || request.OrderID == "" || request.JobID == "" || request.AuthorizationID == "" || request.IdempotencyKey == "" {
		return ServiceOrderVoidReceipt{}, errors.New("local test payment provider requires a non-x402 order and durable authorization")
	}
	identity := stableTestOrderIdentity("void", request.IdempotencyKey)
	return ServiceOrderVoidReceipt{
		EventID: "local-test-void-event:" + identity, VoidID: "local-test-void:" + identity,
		Sandbox: true, Voided: true,
	}, nil
}

type ServiceOrderProviderRouter struct {
	LocalTest ServiceOrderSandboxProvider
	Treasury  ServiceOrderSandboxProvider
}

func NewServiceOrderProviderRouter(localTest, treasury ServiceOrderSandboxProvider) (*ServiceOrderProviderRouter, error) {
	if localTest == nil || treasury == nil {
		return nil, errors.New("service-order provider router requires local test and Treasury providers")
	}
	return &ServiceOrderProviderRouter{LocalTest: localTest, Treasury: treasury}, nil
}

func (p *ServiceOrderProviderRouter) provider(door string) (ServiceOrderSandboxProvider, error) {
	if p == nil {
		return nil, errors.New("service-order provider router is unavailable")
	}
	if localTestDoor(door) {
		return p.LocalTest, nil
	}
	if door == ServiceOrderDoorAgentRun {
		return p.Treasury, nil
	}
	return nil, errors.New("unsupported service-order payment door")
}

func (p *ServiceOrderProviderRouter) Authorize(ctx context.Context, request ServiceOrderAuthorizationRequest) (ServiceOrderAuthorizationReceipt, error) {
	provider, err := p.provider(request.Door)
	if err != nil {
		return ServiceOrderAuthorizationReceipt{}, err
	}
	return provider.Authorize(ctx, request)
}

func (p *ServiceOrderProviderRouter) Capture(ctx context.Context, request ServiceOrderCaptureRequest) (ServiceOrderCaptureReceipt, error) {
	provider, err := p.provider(request.Door)
	if err != nil {
		return ServiceOrderCaptureReceipt{}, err
	}
	return provider.Capture(ctx, request)
}

func (p *ServiceOrderProviderRouter) Void(ctx context.Context, request ServiceOrderVoidRequest) (ServiceOrderVoidReceipt, error) {
	provider, err := p.provider(request.Door)
	if err != nil {
		return ServiceOrderVoidReceipt{}, err
	}
	return provider.Void(ctx, request)
}

func localTestDoor(door string) bool {
	switch strings.TrimSpace(door) {
	case ServiceOrderDoorBrowser, ServiceOrderDoorBrowserAgent, ServiceOrderDoorMCP:
		return true
	default:
		return false
	}
}

func stableTestOrderIdentity(operation, key string) string {
	digest := sha256.Sum256([]byte("lpbs-local-test:" + operation + ":" + key))
	return hex.EncodeToString(digest[:])
}
