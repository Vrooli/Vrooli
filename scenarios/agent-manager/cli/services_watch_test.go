package main

import (
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestWatchServiceUsesGeneratedConnectProcedures(t *testing.T) { // [REQ:REQ-P2-008]
	services, recorder := newContractServices(t)
	calls := []func() error{
		func() error { _, _, err := services.Watches.Create(&domainpb.CreateCohortWatchRequest{}); return err },
		func() error { _, _, err := services.Watches.Get(&domainpb.GetCohortWatchRequest{}); return err },
		func() error { _, _, err := services.Watches.List(&domainpb.ListCohortWatchesRequest{}); return err },
		func() error { _, _, err := services.Watches.Wait(&domainpb.WaitCohortWatchRequest{}); return err },
		func() error { _, _, err := services.Watches.Cancel(&domainpb.CancelCohortWatchRequest{}); return err },
		func() error { _, _, err := services.Watches.Inspect(&domainpb.InspectCohortWatchRequest{}); return err },
		func() error {
			_, _, err := services.Watches.RequestAction(&domainpb.RequestCohortWatchActionRequest{})
			return err
		},
		func() error {
			_, _, err := services.Watches.ListActions(&domainpb.ListCohortWatchActionsRequest{})
			return err
		},
	}
	for _, call := range calls {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"CreateCohortWatch", "GetCohortWatch", "ListCohortWatches", "WaitCohortWatch", "CancelCohortWatch", "InspectCohortWatch", "RequestCohortWatchAction", "ListCohortWatchActions"}
	requests := recorder.Requests()
	if len(requests) != len(want) {
		t.Fatalf("requests=%+v", requests)
	}
	for i, suffix := range want {
		if requests[i].Method != "POST" || requests[i].Path != "/agent_manager.v1.AgentManagerService/"+suffix {
			t.Fatalf("request[%d]=%+v", i, requests[i])
		}
	}
}
