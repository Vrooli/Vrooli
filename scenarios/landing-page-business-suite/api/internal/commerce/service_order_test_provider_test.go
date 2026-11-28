package commerce

import (
	"context"
	"testing"
)

func TestLocalTestServiceOrderProviderIsDeterministicAndNonNetworked(t *testing.T) {
	provider := NewLocalTestServiceOrderProvider()
	request := ServiceOrderAuthorizationRequest{OrderID: "order-1", JobID: "job-1", IdempotencyKey: "order-1:authorize", Door: ServiceOrderDoorBrowser}
	first, err := provider.Authorize(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := provider.Authorize(context.Background(), request)
	if err != nil || replay != first {
		t.Fatalf("authorization replay = %+v, %v; want %+v", replay, err, first)
	}
	captureRequest := ServiceOrderCaptureRequest{OrderID: "order-1", JobID: "job-1", AuthorizationID: first.AuthorizationID, IdempotencyKey: "order-1:capture", Door: ServiceOrderDoorBrowser}
	capture, err := provider.Capture(context.Background(), captureRequest)
	if err != nil || !capture.Sandbox || !capture.Captured || capture.CaptureID == "" {
		t.Fatalf("Capture() = %+v, %v", capture, err)
	}
}

func TestServiceOrderProviderRouterKeepsTreasuryOnlyOnAgentRunDoor(t *testing.T) {
	local := NewLocalTestServiceOrderProvider()
	remote := &recordingOrderProvider{authorization: ServiceOrderAuthorizationReceipt{EventID: "treasury-auth", AuthorizationID: "treasury-admission", Sandbox: true, Approved: true}}
	router, err := NewServiceOrderProviderRouter(local, remote)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := router.Authorize(context.Background(), ServiceOrderAuthorizationRequest{
		OrderID: "order-1", JobID: "job-1", IdempotencyKey: "browser-key", Door: ServiceOrderDoorBrowser,
	}); err != nil {
		t.Fatal(err)
	}
	if remote.authorizeCalls != 0 {
		t.Fatalf("browser checkout reached Treasury %d times", remote.authorizeCalls)
	}
	request := ServiceOrderAuthorizationRequest{OrderID: "order-2", JobID: "job-2", IdempotencyKey: "x402-key", Door: ServiceOrderDoorAgentRun, PaymentSignature: "signed-test-payment"}
	got, err := router.Authorize(context.Background(), request)
	if err != nil || got.AuthorizationID != "treasury-admission" || remote.authorizeCalls != 1 {
		t.Fatalf("Agent Manager route = %+v, %v; Treasury calls=%d", got, err, remote.authorizeCalls)
	}
}

type recordingOrderProvider struct {
	authorization  ServiceOrderAuthorizationReceipt
	authorizeCalls int
}

func (p *recordingOrderProvider) Authorize(context.Context, ServiceOrderAuthorizationRequest) (ServiceOrderAuthorizationReceipt, error) {
	p.authorizeCalls++
	return p.authorization, nil
}

func (*recordingOrderProvider) Capture(context.Context, ServiceOrderCaptureRequest) (ServiceOrderCaptureReceipt, error) {
	return ServiceOrderCaptureReceipt{}, nil
}

func (*recordingOrderProvider) Void(context.Context, ServiceOrderVoidRequest) (ServiceOrderVoidReceipt, error) {
	return ServiceOrderVoidReceipt{}, nil
}
