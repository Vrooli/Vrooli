package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type computeAdmissionFixture struct {
	instanceconnect.UnimplementedInstanceServiceHandler
	quote                     *instancev1.SpendReceipt
	instance                  *instancev1.Instance
	quoteCalls, instanceCalls int
}

func (f *computeAdmissionFixture) GetSpendReceipt(_ context.Context, request *connect.Request[instancev1.GetSpendReceiptRequest]) (*connect.Response[instancev1.GetSpendReceiptResponse], error) {
	f.quoteCalls++
	if request.Msg.GetIdempotencyKey() != f.quote.GetIdempotencyKey() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&instancev1.GetSpendReceiptResponse{Receipt: f.quote}), nil
}
func (f *computeAdmissionFixture) GetInstance(_ context.Context, request *connect.Request[instancev1.GetInstanceRequest]) (*connect.Response[instancev1.GetInstanceResponse], error) {
	f.instanceCalls++
	if request.Msg.GetId() != f.instance.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&instancev1.GetInstanceResponse{Instance: f.instance}), nil
}

func TestCloudComputeAdmissionVerifiesStoredA2ReceiptAndRunningTarget(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	fixture := a2ComputeAdmissionFixture(now)
	path, handler := instanceconnect.NewInstanceServiceHandler(fixture)
	router := http.NewServeMux()
	router.Handle(path, handler)
	service := httptest.NewServer(router)
	defer service.Close()
	verifier := &cloudComputeAdmissionVerifier{resolve: func(context.Context) (string, error) { return service.URL, nil }, client: service.Client(), now: func() time.Time { return now }}
	if err := verifier.VerifyCloudComputeAdmission(context.Background(), "quote-key", "cm-instance"); err != nil {
		t.Fatalf("A2 receipt should verify: %v", err)
	}
	if fixture.quoteCalls != 1 || fixture.instanceCalls != 1 {
		t.Fatalf("read-only calls quote=%d instance=%d", fixture.quoteCalls, fixture.instanceCalls)
	}
}

func TestCloudComputeAdmissionRefusesOverCapBeforeInstanceAccept(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	fixture := a2ComputeAdmissionFixture(now)
	fixture.quote.TotalMinorPerMonth = 710
	path, handler := instanceconnect.NewInstanceServiceHandler(fixture)
	router := http.NewServeMux()
	router.Handle(path, handler)
	service := httptest.NewServer(router)
	defer service.Close()
	verifier := &cloudComputeAdmissionVerifier{resolve: func(context.Context) (string, error) { return service.URL, nil }, client: service.Client(), now: func() time.Time { return now }}
	if err := verifier.VerifyCloudComputeAdmission(context.Background(), "quote-key", "cm-instance"); err == nil {
		t.Fatal("over-cap quote was accepted")
	}
	if fixture.instanceCalls != 0 {
		t.Fatal("instance lookup should stop after quote cap refusal")
	}
}

func TestCloudComputeAdmissionRefusesStaleQuoteAndNonRunningInstance(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		mutate func(*computeAdmissionFixture)
	}{
		{name: "stale quote", mutate: func(f *computeAdmissionFixture) { f.quote.ExpiresAt = timestamppb.New(now.Add(-time.Second)) }},
		{name: "non-running instance", mutate: func(f *computeAdmissionFixture) { f.instance.State = instancev1.InstanceState_INSTANCE_STATE_REQUESTED }},
		{name: "quote binds other instance", mutate: func(f *computeAdmissionFixture) { f.quote.InstanceId = "cm-other" }},
		{name: "quote component total mismatch", mutate: func(f *computeAdmissionFixture) { f.quote.TotalMinorPerMonth = 708 }},
		{name: "offer ID mismatch", mutate: func(f *computeAdmissionFixture) { f.quote.OfferId = "cx33-nbg1" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := a2ComputeAdmissionFixture(now)
			tc.mutate(fixture)
			path, handler := instanceconnect.NewInstanceServiceHandler(fixture)
			router := http.NewServeMux()
			router.Handle(path, handler)
			service := httptest.NewServer(router)
			defer service.Close()
			verifier := &cloudComputeAdmissionVerifier{resolve: func(context.Context) (string, error) { return service.URL, nil }, client: service.Client(), now: func() time.Time { return now }}
			if err := verifier.VerifyCloudComputeAdmission(context.Background(), "quote-key", "cm-instance"); err == nil {
				t.Fatal("invalid upstream facts were accepted")
			}
		})
	}
}

func a2ComputeAdmissionFixture(now time.Time) *computeAdmissionFixture {
	return &computeAdmissionFixture{
		quote:    &instancev1.SpendReceipt{IdempotencyKey: "quote-key", Provider: "hetzner", OfferId: "cx23-nbg1", Region: "nbg1", Size: "cx23", Ipv4: true, Term: "monthly", Currency: "USD", ServerMinorPerMonth: 649, Ipv4MinorPerMonth: 60, TotalMinorPerMonth: 709, TaxBasis: "pre-tax", SourceUrl: "https://prices.example/fixture-only", SourceVersion: "fixture-only-v1", ObservedAt: timestamppb.New(now.Add(-time.Minute)), ExpiresAt: timestamppb.New(now.Add(time.Minute)), QuoteState: "approved", CapCurrency: "USD", CapMonthlyMinor: 709, CapTaxBasis: "pre-tax", AdmissionDecision: "approved", InstanceId: "cm-instance"},
		instance: &instancev1.Instance{Id: "cm-instance", Provider: "hetzner", Region: "nbg1", Size: "cx23", Address: "203.0.113.8", State: instancev1.InstanceState_INSTANCE_STATE_RUNNING, ExpiresAt: timestamppb.New(now.Add(time.Hour))},
	}
}
