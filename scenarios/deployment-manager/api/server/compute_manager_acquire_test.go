package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"deployment-manager/releases"
)

type acquireCMFixture struct {
	requestCalls    int
	acquireCalls    int
	acquireErr      error
	acquireReceipt  *instancev1.SpendReceipt
	requestErr      error
	requestReceipt  *instancev1.SpendReceipt
	receipt         *instancev1.SpendReceipt
	instance        *instancev1.Instance
	readErr         error
	enrollCalls     int
	enrollErr       error
	enrollment      *instancev1.EnrollmentReceipt
	enrollmentReads int
	notFoundFirst   bool
	acquireState    string
	acquireReason   string
}

func (f *acquireCMFixture) RequestInstance(_ context.Context, req *connect.Request[instancev1.RequestInstanceRequest]) (*connect.Response[instancev1.RequestInstanceResponse], error) {
	f.requestCalls++
	if req.Msg.GetProvider() != "hetzner" || req.Msg.GetSize() != "cx23" || req.Msg.GetIdempotencyKey() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	if f.requestErr != nil {
		f.receipt = f.requestReceipt
		return nil, f.requestErr
	}
	return connect.NewResponse(&instancev1.RequestInstanceResponse{Instance: f.instance}), nil
}
func (f *acquireCMFixture) AcquireInstance(_ context.Context, req *connect.Request[instancev1.AcquireInstanceRequest]) (*connect.Response[instancev1.AcquireInstanceResponse], error) {
	f.acquireCalls++
	if req.Msg.GetProfileId() != webConsoleStagingProfile || req.Msg.GetIdempotencyKey() == "" || (req.Msg.GetRegion() != "nbg1" && req.Msg.GetRegion() != "fsn1") {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	if f.acquireErr != nil {
		f.acquireReceipt = f.requestReceipt
		f.receipt = f.requestReceipt
		return nil, f.acquireErr
	}
	f.acquireReceipt = f.receipt
	return connect.NewResponse(&instancev1.AcquireInstanceResponse{Instance: f.instance, SpendReceipt: f.receipt}), nil
}
func (f *acquireCMFixture) GetAcquireReceipt(context.Context, *connect.Request[instancev1.GetAcquireReceiptRequest]) (*connect.Response[instancev1.GetAcquireReceiptResponse], error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	if f.acquireReceipt == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&instancev1.GetAcquireReceiptResponse{Instance: f.instance, SpendReceipt: f.acquireReceipt, State: f.acquireState, Reason: f.acquireReason}), nil
}
func (f *acquireCMFixture) EnrollInstance(_ context.Context, req *connect.Request[instancev1.EnrollInstanceRequest]) (*connect.Response[instancev1.EnrollInstanceResponse], error) {
	f.enrollCalls++
	if req.Msg.GetIdempotencyKey() == "" || req.Msg.GetAcquisitionIdempotencyKey() == "" || req.Msg.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	if f.enrollErr != nil {
		return nil, f.enrollErr
	}
	return connect.NewResponse(&instancev1.EnrollInstanceResponse{Instance: f.instance, Receipt: f.enrollment}), nil
}
func (f *acquireCMFixture) GetEnrollmentReceipt(context.Context, *connect.Request[instancev1.GetEnrollmentReceiptRequest]) (*connect.Response[instancev1.GetEnrollmentReceiptResponse], error) {
	f.enrollmentReads++
	if f.readErr != nil {
		return nil, f.readErr
	}
	if f.notFoundFirst && f.enrollmentReads == 1 {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	if f.enrollment == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&instancev1.GetEnrollmentReceiptResponse{Instance: f.instance, Receipt: f.enrollment}), nil
}
func (f *acquireCMFixture) GetSpendReceipt(context.Context, *connect.Request[instancev1.GetSpendReceiptRequest]) (*connect.Response[instancev1.GetSpendReceiptResponse], error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	if f.receipt == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&instancev1.GetSpendReceiptResponse{Receipt: f.receipt}), nil
}
func (f *acquireCMFixture) GetInstance(context.Context, *connect.Request[instancev1.GetInstanceRequest]) (*connect.Response[instancev1.GetInstanceResponse], error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	if f.instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&instancev1.GetInstanceResponse{Instance: f.instance}), nil
}
func (*acquireCMFixture) ListInstances(context.Context, *connect.Request[instancev1.ListInstancesRequest]) (*connect.Response[instancev1.ListInstancesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*acquireCMFixture) ExtendInstance(context.Context, *connect.Request[instancev1.ExtendInstanceRequest]) (*connect.Response[instancev1.ExtendInstanceResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*acquireCMFixture) DestroyInstance(context.Context, *connect.Request[instancev1.DestroyInstanceRequest]) (*connect.Response[instancev1.DestroyInstanceResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*acquireCMFixture) GetDestroyReceipt(context.Context, *connect.Request[instancev1.GetDestroyReceiptRequest]) (*connect.Response[instancev1.GetDestroyReceiptResponse], error) {
	return nil, connect.NewError(connect.CodeNotFound, nil)
}
func (*acquireCMFixture) SnapshotInstance(context.Context, *connect.Request[instancev1.SnapshotInstanceRequest]) (*connect.Response[instancev1.SnapshotInstanceResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*acquireCMFixture) ResetInstance(context.Context, *connect.Request[instancev1.ResetInstanceRequest]) (*connect.Response[instancev1.ResetInstanceResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*acquireCMFixture) AdoptInstance(context.Context, *connect.Request[instancev1.AdoptInstanceRequest]) (*connect.Response[instancev1.AdoptInstanceResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*acquireCMFixture) SubmitAdoptionCredential(context.Context, *connect.Request[instancev1.SubmitAdoptionCredentialRequest]) (*connect.Response[instancev1.SubmitAdoptionCredentialResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestCloudComputeAcquirerReconcilesLostRequestFromPersistedReceipt(t *testing.T) {
	fixture, now := newAcquireCMFixture()
	fixture.acquireErr = connect.NewError(connect.CodeUnavailable, nil) // response lost after the fixture persisted the receipt
	fixture.requestReceipt = approvedAcquireReceipt("stable-key", "cm-fixture-1", now)
	server := acquireCMServer(t, fixture)
	defer server.Close()
	acquirer := &cloudComputeAcquirer{resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client(), now: func() time.Time { return now }}
	result, err := acquirer.AcquireCloudCompute(context.Background(), approvedAcquireRequest("stable-key"))
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceID != "cm-fixture-1" || result.QuoteIdempotencyKey != "stable-key" || fixture.acquireCalls != 1 {
		t.Fatalf("result=%+v AcquireInstance calls=%d", result, fixture.acquireCalls)
	}
	var exact instancev1.GetAcquireReceiptResponse
	if err := protojson.Unmarshal(result.Receipt, &exact); err != nil || exact.GetState() != "acquired" || exact.GetInstance().GetId() != "cm-fixture-1" || exact.GetSpendReceipt().GetIdempotencyKey() != "stable-key" {
		t.Fatalf("receipt is not the full generated GetAcquireReceipt response: %+v err=%v", &exact, err)
	}
}

func TestCloudComputeAcquirerReadsExistingReceiptBeforeAnyRequest(t *testing.T) {
	fixture, now := newAcquireCMFixture()
	fixture.acquireReceipt = approvedAcquireReceipt("stable-key", "cm-fixture-1", now)
	fixture.receipt = fixture.acquireReceipt
	server := acquireCMServer(t, fixture)
	defer server.Close()
	acquirer := &cloudComputeAcquirer{resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client(), now: func() time.Time { return now }}
	result, err := acquirer.AcquireCloudCompute(context.Background(), approvedAcquireRequest("stable-key"))
	if err != nil {
		t.Fatal(err)
	}
	if result.InstanceID != "cm-fixture-1" || fixture.acquireCalls != 0 {
		t.Fatalf("result=%+v calls=%d", result, fixture.acquireCalls)
	}
}

func TestCloudComputeAcquirerRefusesPersistedOverCapReceiptBeforeRequest(t *testing.T) {
	fixture, now := newAcquireCMFixture()
	fixture.acquireReceipt = approvedAcquireReceipt("stable-key", "cm-fixture-1", now)
	fixture.acquireReceipt.ServerMinorPerMonth = 710
	fixture.acquireReceipt.TotalMinorPerMonth = 710
	fixture.receipt = fixture.acquireReceipt
	server := acquireCMServer(t, fixture)
	defer server.Close()
	acquirer := &cloudComputeAcquirer{resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client(), now: func() time.Time { return now }}
	if _, err := acquirer.AcquireCloudCompute(context.Background(), approvedAcquireRequest("stable-key")); err == nil || !strings.Contains(err.Error(), "A2") || fixture.acquireCalls != 0 {
		t.Fatalf("err=%v calls=%d", err, fixture.acquireCalls)
	}
}

func TestCloudComputeAcquirerTreatsStoredBlockedStateAsRefusalWithoutTarget(t *testing.T) {
	fixture, now := newAcquireCMFixture()
	fixture.instance = nil
	fixture.acquireReceipt = approvedAcquireReceipt("stable-key", "", now)
	fixture.acquireReceipt.AdmissionDecision = "refused"
	fixture.acquireState = "blocked"
	fixture.acquireReason = "quote exceeds the approved cap"
	server := acquireCMServer(t, fixture)
	defer server.Close()
	acquirer := &cloudComputeAcquirer{resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client(), now: func() time.Time { return now }}
	if _, err := acquirer.AcquireCloudCompute(context.Background(), approvedAcquireRequest("stable-key")); err == nil || errors.Is(err, releases.ErrCloudAcquireAmbiguous) || !strings.Contains(err.Error(), "blocked") || fixture.acquireCalls != 0 {
		t.Fatalf("err=%v acquire calls=%d", err, fixture.acquireCalls)
	}
}

func TestCloudComputeAcquirerKeepsMissingReceiptReadbackAmbiguous(t *testing.T) {
	fixture, _ := newAcquireCMFixture()
	fixture.readErr = connect.NewError(connect.CodeUnavailable, nil)
	server := acquireCMServer(t, fixture)
	defer server.Close()
	acquirer := &cloudComputeAcquirer{resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client()}
	_, err := acquirer.AcquireCloudCompute(context.Background(), approvedAcquireRequest("stable-key"))
	if err == nil || !strings.Contains(err.Error(), "ambiguous") || fixture.acquireCalls != 0 {
		t.Fatalf("err=%v calls=%d", err, fixture.acquireCalls)
	}
}

func TestCloudComputeAcquirerDoesNotCallLostBeginWithoutReceiptAmbiguous(t *testing.T) {
	fixture, _ := newAcquireCMFixture()
	fixture.acquireErr = connect.NewError(connect.CodeUnavailable, nil)
	server := acquireCMServer(t, fixture)
	defer server.Close()
	acquirer := &cloudComputeAcquirer{resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client()}
	_, err := acquirer.AcquireCloudCompute(context.Background(), approvedAcquireRequest("stable-key"))
	if err == nil || !errors.Is(err, releases.ErrCloudAcquireAmbiguous) || fixture.acquireCalls != 1 {
		t.Fatalf("err=%v calls=%d", err, fixture.acquireCalls)
	}
}

func TestCloudComputeEnrollerUsesStableReceiptAndAcceptsOnlyExactReadyEvidence(t *testing.T) {
	tests := []struct {
		name      string
		state     instancev1.EnrollmentState
		machine   string
		wantError error
	}{
		{name: "persisted ready", state: instancev1.EnrollmentState_ENROLLMENT_STATE_READY, machine: "bridge-fixture-1"},
		{name: "awaiting human/operator enrollment", state: instancev1.EnrollmentState_ENROLLMENT_STATE_AWAITING_ENROLLMENT, wantError: releases.ErrCloudEnrollmentPending},
		{name: "enrolling", state: instancev1.EnrollmentState_ENROLLMENT_STATE_ENROLLING, wantError: releases.ErrCloudEnrollmentInProgress},
		{name: "ambiguous", state: instancev1.EnrollmentState_ENROLLMENT_STATE_AMBIGUOUS, wantError: releases.ErrCloudEnrollmentAmbiguous},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture, _ := newAcquireCMFixture()
			fixture.enrollment = readyOrPendingEnrollment("stable-enrollment", tc.state, tc.machine)
			fixture.instance.BridgeMachineId = tc.machine
			server := acquireCMServer(t, fixture)
			defer server.Close()
			adapter := &cloudComputeAcquirer{resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client()}
			result, err := adapter.EnrollCloudCompute(context.Background(), releases.CloudEnrollmentRequest{IdempotencyKey: "stable-enrollment", AcquisitionIdempotencyKey: "stable-acquire", InstanceID: "cm-fixture-1"})
			if tc.wantError != nil {
				if err == nil || !errors.Is(err, tc.wantError) || fixture.enrollCalls != 0 {
					t.Fatalf("result=%+v err=%v enroll calls=%d", result, err, fixture.enrollCalls)
				}
				return
			}
			if err != nil || result == nil || result.State != "ready" || result.InstanceID != "cm-fixture-1" || result.BridgeMachineID != tc.machine || result.BridgeOperationID != "bridge-op-fixture" || fixture.enrollCalls != 0 {
				t.Fatalf("result=%+v err=%v enroll calls=%d", result, err, fixture.enrollCalls)
			}
		})
	}
}

func TestCloudComputeEnrollerReconcilesLostEnrollReplyAndRejectsWrongTarget(t *testing.T) {
	fixture, _ := newAcquireCMFixture()
	fixture.notFoundFirst = true
	fixture.enrollErr = connect.NewError(connect.CodeUnavailable, nil)
	fixture.enrollment = readyOrPendingEnrollment("stable-enrollment", instancev1.EnrollmentState_ENROLLMENT_STATE_READY, "bridge-fixture-1")
	fixture.instance.BridgeMachineId = "bridge-fixture-1"
	server := acquireCMServer(t, fixture)
	defer server.Close()
	adapter := &cloudComputeAcquirer{resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client()}
	result, err := adapter.EnrollCloudCompute(context.Background(), releases.CloudEnrollmentRequest{IdempotencyKey: "stable-enrollment", AcquisitionIdempotencyKey: "stable-acquire", InstanceID: "cm-fixture-1"})
	if err != nil || result == nil || result.State != "ready" || fixture.enrollCalls != 1 || fixture.enrollmentReads != 2 {
		t.Fatalf("result=%+v err=%v calls=%d reads=%d", result, err, fixture.enrollCalls, fixture.enrollmentReads)
	}
	_, err = adapter.EnrollCloudCompute(context.Background(), releases.CloudEnrollmentRequest{IdempotencyKey: "stable-enrollment", AcquisitionIdempotencyKey: "stable-acquire", InstanceID: "cm-other"})
	if err == nil || !errors.Is(err, releases.ErrCloudEnrollmentAmbiguous) || fixture.enrollCalls != 1 {
		t.Fatalf("wrong target err=%v calls=%d", err, fixture.enrollCalls)
	}
}

func readyOrPendingEnrollment(key string, state instancev1.EnrollmentState, machine string) *instancev1.EnrollmentReceipt {
	return &instancev1.EnrollmentReceipt{IdempotencyKey: key, AcquisitionIdempotencyKey: "stable-acquire", InstanceId: "cm-fixture-1", State: state, BridgeMachineId: machine, BridgeOperationId: "bridge-op-fixture"}
}

func approvedAcquireRequest(key string) releases.CloudAcquireRequest {
	return releases.CloudAcquireRequest{IdempotencyKey: key, Provider: "hetzner", Region: "nbg1", Size: "cx23", CapCurrency: "USD", CapMonthlyMinor: 709, CapTaxBasis: "pre-tax"}
}
func newAcquireCMFixture() (*acquireCMFixture, time.Time) {
	now := time.Now().UTC()
	return &acquireCMFixture{acquireState: "acquired", instance: &instancev1.Instance{Id: "cm-fixture-1", Provider: "hetzner", Size: "cx23", Region: "nbg1", Address: "203.0.113.8", Tenant: "deployment-manager", State: instancev1.InstanceState_INSTANCE_STATE_RUNNING, ExpiresAt: timestamppb.New(now.Add(time.Hour))}}, now
}
func approvedAcquireReceipt(key, id string, now time.Time) *instancev1.SpendReceipt {
	// Values and URLs below are isolated test fixtures, never live price evidence.
	return &instancev1.SpendReceipt{IdempotencyKey: key, Provider: "hetzner", OfferId: "cx23-nbg1", Region: "nbg1", Size: "cx23", Ipv4: true, Term: "monthly", Currency: "USD", ServerMinorPerMonth: 649, Ipv4MinorPerMonth: 60, TotalMinorPerMonth: 709, TaxBasis: "pre-tax", SourceUrl: "https://fixture.invalid/price", SourceVersion: "fixture-only", ObservedAt: timestamppb.New(now.Add(-time.Minute)), ExpiresAt: timestamppb.New(now.Add(time.Minute)), QuoteState: "approved", CapCurrency: "USD", CapMonthlyMinor: 709, CapTaxBasis: "pre-tax", AdmissionDecision: "approved", InstanceId: id}
}
func acquireCMServer(t *testing.T, fixture *acquireCMFixture) *httptest.Server {
	t.Helper()
	path, handler := instanceconnect.NewInstanceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return httptest.NewServer(mux)
}
