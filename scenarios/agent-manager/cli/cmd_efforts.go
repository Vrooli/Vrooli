package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliutil"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (a *App) cmdEffort(args []string) error {
	if len(args) == 0 {
		args = []string{"board"}
	}
	command := args[0]
	fs := flag.NewFlagSet("effort "+command, flag.ContinueOnError)
	file := fs.String("request-file", "", "Typed RPC request JSON for mutation")
	localOwner := fs.Bool("local-owner", false, "Explicit local operator exchange; no persisted token or agent fallback")
	ref := fs.String("effort-ref", "", "Exact effort reference")
	limit := fs.Uint("page-size", 50, "Maximum rows (1-100)")
	cursor := fs.String("page-token", "", "Next page token from prior response")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected effort arguments")
	}
	if *limit < 1 || *limit > 100 {
		return fmt.Errorf("page-size must be 1-100")
	}
	var path string
	var request, response proto.Message
	mutation := false
	switch command {
	case "issue-dispatch":
		path = apiconnect.AgentManagerServiceIssueSupervisorDispatchProcedure
		request, response, mutation = &api.IssueSupervisorDispatchRequest{}, &pb.EffortEnrollment{}, true
	case "revoke-dispatch":
		path = apiconnect.AgentManagerServiceRevokeSupervisorDispatchProcedure
		request, response, mutation = &api.RevokeSupervisorDispatchRequest{}, &pb.EffortEnrollment{}, true
	case "board":
		path = apiconnect.AgentManagerServiceGetEffortBoardProcedure
		request = &pb.GetEffortBoardRequest{EffortRef: *ref, PageSize: uint32(*limit), PageToken: *cursor}
		response = &pb.EffortBoard{}
	case "list":
		path = apiconnect.AgentManagerServiceListEffortsProcedure
		request = &pb.ListEffortsRequest{PageSize: uint32(*limit), PageToken: *cursor}
		response = &pb.ListEffortsResponse{}
	case "directives":
		path = apiconnect.AgentManagerServiceListEffortDirectivesProcedure
		request = &pb.ListEffortDirectivesRequest{EffortRef: *ref, PageSize: uint32(*limit), PageToken: *cursor}
		response = &pb.ListEffortDirectivesResponse{}
	case "discover":
		path = apiconnect.AgentManagerServiceReconcileEffortDiscoveryProcedure
		request = &pb.ReconcileEffortDiscoveryRequest{}
		response = &pb.EffortDiscovery{}
	case "enroll":
		path = apiconnect.AgentManagerServiceEnrollEffortProcedure
		request = &pb.EnrollEffortRequest{}
		response = &pb.EffortEnrollment{}
		mutation = true
	case "withdraw":
		path = apiconnect.AgentManagerServiceWithdrawEffortProcedure
		request = &pb.WithdrawEffortRequest{}
		response = &pb.EffortEnrollment{}
		mutation = true
	case "direct":
		path = apiconnect.AgentManagerServiceRequestEffortDirectiveProcedure
		request = &pb.RequestEffortDirectiveRequest{}
		response = &pb.EffortDirective{}
		mutation = true
	case "update-directive":
		path = apiconnect.AgentManagerServiceUpdateEffortDirectiveProcedure
		request = &pb.UpdateEffortDirectiveRequest{}
		response = &pb.EffortDirective{}
		mutation = true
	case "assess":
		path = apiconnect.AgentManagerServiceRecordEffortAssessmentProcedure
		request = &pb.RecordEffortAssessmentRequest{}
		response = &pb.EffortAssessment{}
		mutation = true
	default:
		return fmt.Errorf("unknown effort operation %q", command)
	}
	if mutation && *file == "" {
		return fmt.Errorf("effort %s requires --request-file with the typed RPC request", command)
	}
	if *file != "" {
		raw, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		if err = protojson.Unmarshal(raw, request); err != nil {
			return fmt.Errorf("decode effort request: %w", err)
		}
	}
	service := a.services.Watches
	if *localOwner {
		if !mutation && command != "discover" {
			return fmt.Errorf("--local-owner is only for explicit operator mutations")
		}
		if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) != "" {
			return fmt.Errorf("--local-owner is unavailable inside an identified agent run; use the run's granted authority")
		}
		if scoped, ok := request.(interface{ GetAuthority() pb.WatchAuthority }); ok && scoped.GetAuthority() != pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR {
			return fmt.Errorf("--local-owner requires operator authority")
		}
		login, err := authn.ExchangeLocalMachinePrincipal(context.Background())
		if err != nil {
			return fmt.Errorf("local owner exchange unavailable: %w", err)
		}
		service = &WatchService{api: service.api.WithToken(login.Tokens.AccessToken)}
	}
	body, err := service.call(path, request, response)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	switch out := response.(type) {
	case *pb.EffortBoard:
		printEffortBoard(out)
	case *pb.ListEffortsResponse:
		for _, e := range out.Efforts {
			fmt.Printf("%s  %s  revision=%d withdrawn=%t\n", e.EffortRef, e.DisplayName, e.Revision, e.Withdrawn)
		}
		printEffortPage(out.NextPageToken)
	case *pb.EffortEnrollment:
		fmt.Printf("%s revision=%d withdrawn=%t\n", out.EffortRef, out.Revision, out.Withdrawn)
		if grant := out.GetDispatchAuthorization(); grant != nil {
			fmt.Printf("Dispatcher %s team=%s member=%s profile=%s expires=%s runs=%d/%d minimum-interval=%ds revoked=%t\n", grant.AuthorizationId, grant.TeamId, grant.MemberId, grant.ProfileKey, grant.ExpiresAt.AsTime().Format(time.RFC3339), grant.DispatchedRuns, grant.MaximumRuns, grant.MinimumIntervalSeconds, grant.RevokedAt != nil)
		}
	case *pb.EffortDiscovery:
		fmt.Printf("Discovery generation=%d scanned=%d limit=%d partial=%t cursor=%s\n", out.Generation, out.ScannedCount, out.ScanLimit, out.Partial, out.ScanCursor)
		for _, f := range out.Findings {
			fmt.Printf("%s: %s — %s\n", f.Source, f.Code, f.Reason)
		}
	case *pb.EffortDirective:
		printEffortDirective(out)
	case *pb.EffortAssessment:
		fmt.Printf("Assessment %s disposition=%s benefit=%s shared-operation=%s\n", out.AssessmentId, out.Disposition, out.Benefit, out.SharedOperationRef)
	case *pb.ListEffortDirectivesResponse:
		for _, d := range out.Directives {
			printEffortDirective(d)
		}
		printEffortPage(out.NextPageToken)
	}
	return nil
}
func printEffortPage(token string) {
	if token != "" {
		fmt.Printf("Next page: --page-token %q\n", token)
	}
}
func printEffortDirective(d *pb.EffortDirective) {
	fmt.Printf("%s effort=%s revision=%d delivery=%s acknowledgment=%s assessment=%s\n", d.DirectiveId, d.EffortRef, d.Revision, strings.TrimPrefix(d.Delivery.String(), "EFFORT_DIRECTIVE_DELIVERY_"), strings.TrimPrefix(d.Acknowledgment.String(), "EFFORT_DIRECTIVE_ACKNOWLEDGMENT_"), d.Assessment)
	if d.DeliveryReason != "" {
		fmt.Println("  " + d.DeliveryReason)
	}
	if expectation := d.GetRecoveryExpectation(); expectation != nil {
		if d.RecoveredRunId != "" {
			fmt.Printf("  Replacement run: %s; predecessor: %s\n", d.RecoveredRunId, d.TargetRunId)
		}
		fmt.Printf("  Recovery: %s; progress condition: %s\n", d.GetRecoveryVerification().GetState(), expectation.ProgressCondition)
		if v := d.GetRecoveryVerification(); v != nil && v.Reason != "" {
			fmt.Printf("  Verification: %s; verifier=%s; next=%s\n", v.Reason, v.Verifier, v.NextOwnerCondition)
		}
	}
}
func printEffortBoard(b *pb.EffortBoard) {
	fmt.Printf("Efforts: %d active; discovery generation=%d; partial=%t\n", b.ActiveCount, b.GetDiscovery().GetGeneration(), b.Partial)
	if b.ObservedAt != nil {
		fmt.Printf("Observed: %s\n", b.ObservedAt.AsTime().Format(time.RFC3339))
	}
	if b.GetDiscovery().GetLastSuccessfulScanAt() != nil {
		fmt.Printf("Last successful scan: %s\n", b.Discovery.LastSuccessfulScanAt.AsTime().Format(time.RFC3339))
	}
	for _, r := range b.Rows {
		e := r.GetEnrollment()
		tokens, cost := "unknown", "unknown"
		if r.Usage != nil && r.Usage.Tokens != nil {
			tokens = fmt.Sprint(r.GetUsage().GetTokens())
		}
		if r.Usage != nil && r.Usage.ReportedCostUsd != nil {
			cost = fmt.Sprintf("$%.4f", r.GetUsage().GetReportedCostUsd())
		}
		fmt.Printf("\n%s (%s)\n  Runtime: %s; outcome: %s (%s); freshness: %s\n  Target: %s @ %s\n  Usage: tokens=%s reported-cost=%s\n  Next: %s — %s\n", e.GetDisplayName(), e.GetEffortRef(), r.RuntimeState, r.GetOutcomeStanding().GetState(), r.GetOutcomeStanding().GetAttribution(), strings.TrimPrefix(r.Freshness.String(), "EFFORT_FRESHNESS_"), e.GetDestinationRef(), e.GetTargetRevision(), tokens, cost, r.NextAction, r.Rationale)
		for _, a := range r.Assignments {
			fmt.Printf("  %s: %s/%s run=%s state=%s %s\n", a.GetSubject().GetRole(), a.GetSubject().GetOwner(), a.GetSubject().GetReference(), a.GetSubject().GetRunId(), a.RuntimeState, a.UnavailableReason)
			fmt.Printf("    Requested: runner=%s model=%s reasoning=%s; effective: runner=%s model=%s reasoning=%s\n", effortKnown(a.RequestedRunner), effortKnown(a.RequestedModel), effortKnown(a.RequestedReasoning), effortKnown(a.EffectiveRunner), effortKnown(a.EffectiveModel), effortKnown(a.EffectiveReasoning))
		}
		if a := r.LastAssessment; a != nil {
			fmt.Printf("  Assessment: %s %s benefit=%s evidence=%s shared-operation=%s allowance=%s (cost unknown/unallocated unless reported)\n", a.AssessmentId, a.Disposition, a.Benefit, a.SourceLedgerRef, a.SharedOperationRef, effortKnown(a.AllowanceRef))
		}
		for _, v := range r.Blockers {
			fmt.Println("  Blocker: " + v)
		}
		for _, v := range r.PendingOperations {
			fmt.Println("  Pending: " + v)
		}
		for _, v := range r.Limitations {
			fmt.Println("  Limitation: " + v)
		}
		for _, d := range r.Directives {
			printEffortDirective(d)
		}
	}
	for _, f := range b.GetDiscovery().GetFindings() {
		fmt.Printf("Discovery %s: %s\n", f.Source, f.Reason)
	}
	printEffortPage(b.NextPageToken)
}

func effortKnown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
