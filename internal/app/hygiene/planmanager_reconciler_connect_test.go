package hygiene

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/demand"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

type reconcilePlansStub struct {
	plansconnect.UnimplementedPlansServiceHandler
	request *plansv1.ReconcilePlansRequest
}

func (s *reconcilePlansStub) ReconcilePlans(_ context.Context, req *connect.Request[plansv1.ReconcilePlansRequest]) (*connect.Response[plansv1.ReconcilePlansResponse], error) {
	s.request = req.Msg
	return connect.NewResponse(&plansv1.ReconcilePlansResponse{
		DryRun: true,
		Items: []*plansv1.ReconcilePlanItem{
			nil,
			{
				Action:                  plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_REPAIRED,
				PlanId:                  "plan-1",
				Slug:                    "plan-one",
				Title:                   "Plan One",
				SourcePath:              "/repo/plans/one.md",
				Mirror:                  &sharedv1.RenderedPlanMirror{Path: "/home/plans/one.md", Status: sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_FRESH},
				SourceUntouched:         true,
				SourceRetirementPlanned: true,
				Error:                   "diagnostic",
			},
		},
	}), nil
}

type reconcileDemandStub struct {
	mu       sync.Mutex
	acquired []demand.AcquireRequest
	released []string
}

func (s *reconcileDemandStub) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acquired = append(s.acquired, req)
	return demand.Lease{LeaseID: req.LeaseID}, nil
}

func (s *reconcileDemandStub) Renew(context.Context, string, time.Duration) (demand.Lease, error) {
	return demand.Lease{}, nil
}

func (s *reconcileDemandStub) Release(_ context.Context, leaseID, _ string) (demand.Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.released = append(s.released, leaseID)
	return demand.Lease{LeaseID: leaseID}, nil
}

func TestPlanManagerReconcilerUsesUniqueConcurrentDemandLeases(t *testing.T) {
	stub := &reconcileDemandStub{}
	reconciler := planManagerReconciler{demand: stub, demandConsumerID: "test:reconciler"}
	const calls = 32
	var wg sync.WaitGroup
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := reconciler.acquireDemand(context.Background(), "reconcile-plans")
			if err != nil {
				t.Errorf("acquireDemand: %v", err)
				return
			}
			if release != nil {
				release()
			}
		}()
	}
	wg.Wait()
	if got := len(stub.acquired); got != calls {
		t.Fatalf("acquired leases = %d, want %d", got, calls)
	}
	seen := make(map[string]struct{}, calls)
	for _, req := range stub.acquired {
		if _, ok := seen[req.LeaseID]; ok {
			t.Fatalf("duplicate lease ID %q", req.LeaseID)
		}
		seen[req.LeaseID] = struct{}{}
	}
}

func TestPlanManagerReconcilerUsesGeneratedConnectRequestAndProjection(t *testing.T) {
	owner := &reconcilePlansStub{}
	path, handler := plansconnect.NewPlansServiceHandler(owner)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	demandStub := &reconcileDemandStub{}
	reconciler := planManagerReconciler{
		client:           server.Client(),
		baseURL:          server.URL,
		demand:           demandStub,
		demandConsumerID: "test:plan-reconciler",
	}

	report, err := reconciler.ReconcilePlans(context.Background(), PlanReconcileRequest{
		DryRun:          true,
		RepairMirrors:   true,
		SourceIntake:    true,
		RetireSources:   true,
		IncludeArchived: true,
		SkipExisting:    true,
		WorkspaceRoot:   "/repo",
	})
	if err != nil {
		t.Fatalf("ReconcilePlans: %v", err)
	}
	request := owner.request
	if request == nil || !request.GetDryRun() || !request.GetRepairMirrors() || !request.GetSourceIntake() || !request.GetRetireSources() || !request.GetIncludeArchived() {
		t.Fatalf("reconcile request flags = %#v", request)
	}
	if request.GetConflictPolicy() != plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_SKIP_EXISTING || !request.GetSourceRuntimeHomePlans() || !request.GetSourceDocsPlans() || !request.GetSourceRepoPlans() || request.GetWorkspace().GetRoot() != "/repo" {
		t.Fatalf("reconcile request policy/sources/scope = %#v", request)
	}
	if !report.DryRun || len(report.Items) != 1 {
		t.Fatalf("report = %#v", report)
	}
	item := report.Items[0]
	if item.Action != planmanagerReconcilerMirrorRepaired || item.PlanID != "plan-1" || item.MirrorPath != "/home/plans/one.md" || item.MirrorStatus != "fresh" || !item.SourceUntouched || !item.SourceRetirementPlanned || item.SourceRemoved || item.Error != "diagnostic" {
		t.Fatalf("item projection = %#v", item)
	}
	demandStub.mu.Lock()
	defer demandStub.mu.Unlock()
	if len(demandStub.acquired) != 1 || len(demandStub.released) != 1 || demandStub.acquired[0].ConsumerID != "test:plan-reconciler" || demandStub.released[0] != demandStub.acquired[0].LeaseID {
		t.Fatalf("demand lease lifecycle = acquired %#v released %#v", demandStub.acquired, demandStub.released)
	}
}

func TestPlanManagerReconcilerResolvesCurrentURLForEachOperation(t *testing.T) {
	firstOwner := &reconcilePlansStub{}
	firstPath, firstHandler := plansconnect.NewPlansServiceHandler(firstOwner)
	firstMux := http.NewServeMux()
	firstMux.Handle(firstPath, firstHandler)
	firstServer := httptest.NewServer(firstMux)
	t.Cleanup(firstServer.Close)

	secondOwner := &reconcilePlansStub{}
	secondPath, secondHandler := plansconnect.NewPlansServiceHandler(secondOwner)
	secondMux := http.NewServeMux()
	secondMux.Handle(secondPath, secondHandler)
	secondServer := httptest.NewServer(secondMux)
	t.Cleanup(secondServer.Close)

	currentURL := firstServer.URL
	resolutions := 0
	reconciler := planManagerReconciler{
		client: http.DefaultClient,
		resolveURL: func(context.Context) (string, error) {
			resolutions++
			return currentURL, nil
		},
	}
	if _, err := reconciler.ReconcilePlans(context.Background(), PlanReconcileRequest{DryRun: true}); err != nil {
		t.Fatalf("first ReconcilePlans: %v", err)
	}
	if firstOwner.request == nil {
		t.Fatal("first resolved endpoint received no request")
	}
	currentURL = secondServer.URL
	if _, err := reconciler.ReconcilePlans(context.Background(), PlanReconcileRequest{DryRun: true}); err != nil {
		t.Fatalf("second ReconcilePlans: %v", err)
	}
	if secondOwner.request == nil || resolutions != 2 {
		t.Fatalf("second resolved endpoint request=%v resolutions=%d, want request and 2 resolutions", secondOwner.request != nil, resolutions)
	}
}

func TestPlanManagerEnumUnknownValuesRemainVisible(t *testing.T) {
	if got := reconcileActionString(plansv1.ReconcileAction(777)); got != "unknown(777)" {
		t.Fatalf("unknown reconcile action = %q", got)
	}
	if got := mirrorStatusString(sharedv1.RenderedMirrorStatus(888)); got != "unknown(888)" {
		t.Fatalf("unknown mirror status = %q", got)
	}
}
