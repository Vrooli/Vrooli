package main

import (
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestInvestigationAndWatchServicesUseTheirDeclaredRPCProcedures(t *testing.T) { // [REQ:REQ-P2-008]
	services, recorder := newContractServices(t)
	calls := []struct {
		name string
		path string
		call func() error
	}{
		{"investigation start", "/agent_manager.v1.AgentManagerService/StartInvestigation", func() error {
			_, _, err := services.Investigations.Start(&apipb.StartInvestigationRequest{})
			return err
		}},
		{"investigation get", "/agent_manager.v1.AgentManagerService/GetInvestigation", func() error {
			_, _, err := services.Investigations.Get(&apipb.GetInvestigationRequest{InvestigationId: "investigation-1"})
			return err
		}},
		{"investigation list", "/agent_manager.v1.AgentManagerService/ListInvestigations", func() error {
			_, _, err := services.Investigations.List(&apipb.ListInvestigationsRequest{Limit: 17})
			return err
		}},
		{"investigation wait", "/agent_manager.v1.AgentManagerService/WaitInvestigation", func() error {
			_, _, err := services.Investigations.Wait(&apipb.WaitInvestigationRequest{InvestigationId: "investigation-2"})
			return err
		}},
		{"investigation cancel", "/agent_manager.v1.AgentManagerService/CancelInvestigation", func() error {
			_, _, err := services.Investigations.Cancel(&apipb.CancelInvestigationRequest{InvestigationId: "investigation-3"})
			return err
		}},
		{"watch create", "/agent_manager.v1.AgentManagerService/CreateCohortWatch", func() error { _, _, err := services.Watches.Create(&domainpb.CreateCohortWatchRequest{}); return err }},
		{"watch get", "/agent_manager.v1.AgentManagerService/GetCohortWatch", func() error {
			_, _, err := services.Watches.Get(&domainpb.GetCohortWatchRequest{WatchId: "watch-1"})
			return err
		}},
		{"watch list", "/agent_manager.v1.AgentManagerService/ListCohortWatches", func() error {
			_, _, err := services.Watches.List(&domainpb.ListCohortWatchesRequest{PageSize: 23})
			return err
		}},
		{"watch wait", "/agent_manager.v1.AgentManagerService/WaitCohortWatch", func() error {
			_, _, err := services.Watches.Wait(&domainpb.WaitCohortWatchRequest{WatchId: "watch-2"})
			return err
		}},
		{"watch cancel", "/agent_manager.v1.AgentManagerService/CancelCohortWatch", func() error {
			_, _, err := services.Watches.Cancel(&domainpb.CancelCohortWatchRequest{WatchId: "watch-3"})
			return err
		}},
		{"watch inspect", "/agent_manager.v1.AgentManagerService/InspectCohortWatch", func() error {
			_, _, err := services.Watches.Inspect(&domainpb.InspectCohortWatchRequest{WatchId: "watch-4"})
			return err
		}},
		{"watch request action", "/agent_manager.v1.AgentManagerService/RequestCohortWatchAction", func() error {
			_, _, err := services.Watches.RequestAction(&domainpb.RequestCohortWatchActionRequest{})
			return err
		}},
		{"watch list actions", "/agent_manager.v1.AgentManagerService/ListCohortWatchActions", func() error {
			_, _, err := services.Watches.ListActions(&domainpb.ListCohortWatchActionsRequest{})
			return err
		}},
		{"watch policy", "/agent_manager.v1.AgentManagerService/GetSupervisionPolicy", func() error {
			_, _, err := services.Watches.GetPolicy(&domainpb.GetSupervisionPolicyRequest{})
			return err
		}},
		{"watch outcomes", "/agent_manager.v1.AgentManagerService/ListSupervisionOutcomes", func() error {
			_, _, err := services.Watches.ListOutcomes(&domainpb.ListSupervisionOutcomesRequest{})
			return err
		}},
		{"policy candidate", "/agent_manager.v1.AgentManagerService/CreateSupervisionPolicyCandidate", func() error {
			_, _, err := services.Watches.PolicyCandidate(&domainpb.CreateSupervisionPolicyCandidateRequest{})
			return err
		}},
		{"policy evaluate", "/agent_manager.v1.AgentManagerService/EvaluateSupervisionPolicy", func() error {
			_, _, err := services.Watches.PolicyEvaluate(&domainpb.EvaluateSupervisionPolicyRequest{})
			return err
		}},
		{"policy assess", "/agent_manager.v1.AgentManagerService/RecordSupervisionOutcome", func() error {
			_, _, err := services.Watches.PolicyAssess(&domainpb.RecordSupervisionOutcomeRequest{})
			return err
		}},
		{"policy promote", "/agent_manager.v1.AgentManagerService/PromoteSupervisionPolicy", func() error {
			_, _, err := services.Watches.PolicyPromote(&domainpb.PromoteSupervisionPolicyRequest{})
			return err
		}},
		{"policy reject", "/agent_manager.v1.AgentManagerService/RejectSupervisionPolicy", func() error {
			_, _, err := services.Watches.PolicyReject(&domainpb.RejectSupervisionPolicyRequest{})
			return err
		}},
		{"policy rollback", "/agent_manager.v1.AgentManagerService/RollbackSupervisionPolicy", func() error {
			_, _, err := services.Watches.PolicyRollback(&domainpb.RollbackSupervisionPolicyRequest{})
			return err
		}},
		{"policy disable", "/agent_manager.v1.AgentManagerService/SetSupervisionPolicyDisabled", func() error {
			_, _, err := services.Watches.PolicyDisable(&domainpb.SetSupervisionPolicyDisabledRequest{})
			return err
		}},
	}
	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err != nil {
				t.Fatal(err)
			}
			request := recorder.Requests()[len(recorder.Requests())-1]
			if request.Method != "POST" || request.Path != tc.path {
				t.Fatalf("request=%+v, want POST %s", request, tc.path)
			}
		})
	}
	if got := len(recorder.Requests()); got != len(calls) {
		t.Fatalf("requests=%d, want one request for each of %d service methods", got, len(calls))
	}
}
