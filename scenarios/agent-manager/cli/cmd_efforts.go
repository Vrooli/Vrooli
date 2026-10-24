package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (a *App) cmdEffort(args []string) error {
	if len(args) == 0 {
		args = []string{"board"}
	}
	command := args[0]
	// Goal-home commands run locally over the goal-home files.
	switch command {
	case "exact-setup":
		return a.effortExactSetup(args[1:])
	case "exact-receipt":
		return a.effortExactReceipt(args[1:])
	case "exact-reconcile":
		return a.effortExactReconcile(args[1:])
	case "epoch-check":
		return a.effortEpochCheck(args[1:])
	case "lint":
		return a.effortLint(args[1:])
	case "handoff":
		return a.effortHandoff(args[1:])
	case "migrate":
		return a.effortMigrate(args[1:])
	case "delivery":
		return a.effortDelivery(args[1:])
	case "park":
		return a.effortPark(args[1:])
	case "signals":
		return a.effortSignals(args[1:])
	case "signal":
		return a.effortSignal(args[1:])
	case "feedback":
		return a.effortFeedback(args[1:])
	case "decision":
		return a.effortDecision(args[1:])
	}
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
	case "board":
		path = apiconnect.AgentManagerServiceGetEffortBoardProcedure
		request = &pb.GetEffortBoardRequest{EffortRef: *ref, PageSize: uint32(*limit), PageToken: *cursor}
		response = &pb.EffortBoard{}
	case "compact":
		// The compact owner read uses the same single joined AM projection as
		// board. It only changes presentation, so detail references can be
		// followed explicitly without repeating board/transcript reads.
		path = apiconnect.AgentManagerServiceGetEffortBoardProcedure
		request = &pb.GetEffortBoardRequest{EffortRef: *ref, PageSize: uint32(*limit), PageToken: *cursor}
		response = &pb.EffortBoard{}
	case "list":
		path = apiconnect.AgentManagerServiceListEffortsProcedure
		request = &pb.ListEffortsRequest{PageSize: uint32(*limit), PageToken: *cursor}
		response = &pb.ListEffortsResponse{}
	case "discover":
		path = apiconnect.AgentManagerServiceReconcileEffortDiscoveryProcedure
		request = &pb.ReconcileEffortDiscoveryRequest{}
		response = &pb.EffortDiscovery{}
	case "enroll":
		path = apiconnect.AgentManagerServiceEnrollEffortProcedure
		request = &pb.EnrollEffortRequest{}
		response = &pb.EffortEnrollment{}
		mutation = true
	case "reconcile-metadata":
		path = apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure
		request = &pb.ReconcileEffortMetadataRequest{}
		response = &pb.EffortEnrollment{}
		mutation = true
	case "withdraw":
		path = apiconnect.AgentManagerServiceWithdrawEffortProcedure
		request = &pb.WithdrawEffortRequest{}
		response = &pb.EffortEnrollment{}
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
	if *localOwner && !mutation && command != "discover" {
		return fmt.Errorf("--local-owner is only for explicit operator mutations")
	}
	service, err := a.effortService(*localOwner, request)
	if err != nil {
		return err
	}
	body, err := service.call(path, request, response)
	if err != nil {
		return apiError(body, err)
	}
	return printEffortResponse(command, *jsonOut, response, body)
}

// effortService retains the same operator exchange and ambient agent credentials.
func (a *App) effortService(localOwner bool, request proto.Message) (*WatchService, error) {
	service := a.services.Watches
	if localOwner {
		if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) != "" {
			return nil, fmt.Errorf("--local-owner is unavailable inside an identified agent run; use this operation's supported owner or granted-agent path")
		}
		if scoped, ok := request.(interface{ GetAuthority() pb.WatchAuthority }); ok && scoped.GetAuthority() != pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR {
			return nil, fmt.Errorf("--local-owner requires operator authority")
		}
		if err := cliutil.ConfirmLocalOwnerEligible(context.Background()); err != nil {
			return nil, fmt.Errorf("local owner caller authority unavailable: %w", err)
		}
		login, err := authn.ExchangeLocalMachinePrincipal(context.Background())
		if err != nil {
			return nil, fmt.Errorf("local owner exchange unavailable: %w", err)
		}
		service = &WatchService{api: service.api.WithToken(login.Tokens.AccessToken)}
	}
	return service, nil
}

func printEffortResponse(command string, jsonOut bool, response proto.Message, body []byte) error {
	if jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	switch out := response.(type) {
	case *pb.EffortBoard:
		if command == "compact" {
			printCompactEffortBoard(out)
		} else {
			printEffortBoard(out)
		}
	case *pb.ListEffortsResponse:
		for _, e := range out.Efforts {
			fmt.Printf("%s  %s  revision=%d withdrawn=%t\n", e.EffortRef, e.DisplayName, e.Revision, e.Withdrawn)
		}
		printEffortPage(out.NextPageToken)
	case *pb.EffortEnrollment:
		fmt.Printf("%s revision=%d withdrawn=%t\n", out.EffortRef, out.Revision, out.Withdrawn)
	case *pb.EffortDiscovery:
		fmt.Printf("Discovery generation=%d scanned=%d limit=%d partial=%t cursor=%s\n", out.Generation, out.ScannedCount, out.ScanLimit, out.Partial, out.ScanCursor)
		for _, f := range out.Findings {
			fmt.Printf("%s: %s — %s\n", f.Source, f.Code, f.Reason)
		}
	}
	return nil
}

// effortDelivery composes an exact owner enrollment, retaining the existing
// authority, revision fencing and idempotency path instead of a second writer.
func (a *App) effortDelivery(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-manager effort delivery <start|pause|resume|stop> --effort-ref <ref> [--commission <id> --budget 24h --workers 2] [--reason <text>] [--local-owner] [--json]")
	}
	action := args[0]
	if action != "start" && action != "pause" && action != "resume" && action != "stop" {
		return fmt.Errorf("unknown delivery action %q", action)
	}
	fs := flag.NewFlagSet("effort delivery", flag.ContinueOnError)
	ref := fs.String("effort-ref", "", "Exact enrolled effort")
	commission := fs.String("commission", "", "New commission ID (start only)")
	budget := fs.Duration("budget", 24*time.Hour, "Commission duration (start only)")
	workers := fs.Uint("workers", 2, "Maximum independent workers (start only)")
	reason := fs.String("reason", "", "Pause or stop disposition")
	key := fs.String("idempotency-key", "", "Stable mutation request identity")
	localOwner := fs.Bool("local-owner", false, "Use the existing explicit local operator exchange")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	if *ref == "" || fs.NArg() != 0 {
		return errors.New("delivery requires --effort-ref")
	}
	if action == "start" && (*commission == "" || *budget < time.Second || *budget > 30*24*time.Hour || *workers < 1 || *workers > 32) {
		return errors.New("start requires --commission, a 1s–30d budget and 1–32 workers")
	}
	if (action == "pause" || action == "stop") && strings.TrimSpace(*reason) == "" {
		return errors.New("pause and stop require --reason")
	}
	service, err := a.effortService(*localOwner, &pb.EnrollEffortRequest{})
	if err != nil {
		return err
	}
	board := &pb.EffortBoard{}
	if _, err := service.call(apiconnect.AgentManagerServiceGetEffortBoardProcedure, &pb.GetEffortBoardRequest{EffortRef: *ref, PageSize: 1}, board); err != nil {
		return err
	}
	if len(board.Rows) != 1 || board.Rows[0].GetEnrollment().GetEffortRef() != *ref {
		return errors.New("exact delivery enrollment unavailable")
	}
	e := proto.Clone(board.Rows[0].Enrollment).(*pb.EffortEnrollment)
	switch action {
	case "start":
		e.Delivery = &pb.EffortDelivery{CommissionId: *commission, BudgetSeconds: int64(*budget / time.Second), MaxWorkers: uint32(*workers), State: "running"}
	default:
		if e.Delivery == nil {
			return errors.New("effort has no delivery commission")
		}
		e.Delivery.Reason = *reason
		switch action {
		case "pause":
			e.Delivery.State = "paused"
		case "resume":
			e.Delivery.State = "running"
		case "stop":
			e.Delivery.State = "stopped"
		}
	}
	if *key == "" {
		*key = fmt.Sprintf("delivery:%s:%s:%s:%d", *ref, e.Delivery.CommissionId, action, e.Revision)
	}
	request := &pb.EnrollEffortRequest{Enrollment: e, ExpectedRevision: e.Revision, IdempotencyKey: *key}
	response := &pb.EffortEnrollment{}
	body, err := service.call(apiconnect.AgentManagerServiceEnrollEffortProcedure, request, response)
	if err != nil {
		return apiError(body, err)
	}
	return printEffortResponse("enroll", *jsonOut, response, body)
}

func printDeliveryPressure(p *pb.EffortDeliveryPressure) {
	if p == nil {
		return
	}
	fmt.Printf("    Delivery %s: %s; active=%ds budget=%ds paused=%ds; hard stop=%s\n", p.CommissionId, p.Phase, p.ElapsedSeconds, p.BudgetSeconds, p.PausedSeconds, effortTimestamp(p.HardStopAt))
	fmt.Printf("      %s\n", p.Instruction)
	if p.Disposition != "" {
		fmt.Printf("      %s\n", p.Disposition)
	}
}

func printCompactEffortBoard(b *pb.EffortBoard) {
	fmt.Printf("Efforts: %d active; observed=%s; partial=%t\n", b.GetActiveCount(), effortTimestamp(b.GetObservedAt()), b.GetPartial())
	if quotas := b.GetQuotaObservations(); len(quotas) > 0 {
		fmt.Println("Quota observations (shared owner cut):")
		for _, quota := range quotas {
			if quota == nil {
				continue
			}
			used := "unknown"
			if quota.UsedPercent != nil {
				used = fmt.Sprintf("%.2f%%", quota.GetUsedPercent())
			}
			fmt.Printf("  %s/%s/%s standing=%s used=%s reset=%s source=%s evidence=%s\n", quota.GetProvider(), quota.GetPool(), quota.GetWindow(), quota.GetStanding(), used, effortTimestamp(quota.GetResetAt()), effortKnown(quota.GetSourceRunId()), effortKnown(quota.GetEvidenceRef()))
		}
	}
	for _, r := range b.GetRows() {
		e := r.GetEnrollment()
		fmt.Printf("\n%s (%s)\n  Standing: runtime=%s outcome=%s freshness=%s target=%s @ %s\n", e.GetDisplayName(), e.GetEffortRef(), r.GetRuntimeState(), r.GetOutcomeStanding().GetState(), strings.TrimPrefix(r.GetFreshness().String(), "EFFORT_FRESHNESS_"), e.GetDestinationRef(), e.GetTargetRevision())
		fmt.Printf("  Changed evidence: %s\n", strings.Join(r.GetEvidenceRefs(), ","))
		if usage := r.GetUsage(); usage != nil {
			tokens, cost := "unknown", "unknown"
			if usage.Tokens != nil {
				tokens = fmt.Sprint(usage.GetTokens())
			}
			if usage.ReportedCostUsd != nil {
				cost = fmt.Sprintf("$%.4f", usage.GetReportedCostUsd())
			}
			fmt.Printf("  Usage: tokens=%s reported-cost=%s observed-runs=%d/%d\n", tokens, cost, usage.GetObservedRuns(), usage.GetDeclaredRuns())
		} else {
			fmt.Println("  Usage: unknown")
		}
		if waits := r.GetPendingOperations(); len(waits) > 0 {
			fmt.Printf("  Named waits: %s\n", strings.Join(waits, ", "))
		} else {
			fmt.Println("  Named waits: none")
		}
		printGoalHome(r.GetGoalHome())
		printDeliveryPressure(r.GetDeliveryPressure())
		refs := append([]string{}, r.GetEvidenceRefs()...)
		refs = append(refs, r.GetPendingOperations()...)
		refs = append(refs, "agent-manager:GetEffortBoard:"+e.GetEffortRef())
		fmt.Printf("  Detail refs: %s\n", strings.Join(boundedEffortRefs(refs, 32), ", "))
	}
	if findings := b.GetDiscovery().GetFindings(); len(findings) > 0 {
		for _, f := range findings {
			fmt.Printf("Discovery %s: %s\n", f.GetSource(), f.GetReason())
		}
	}
	printEffortPage(b.GetNextPageToken())
}

func effortTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil || !ts.IsValid() {
		return "unknown"
	}
	return ts.AsTime().Format(time.RFC3339)
}

func uniqueEffortRefs(refs []string) []string {
	seen := make(map[string]struct{}, len(refs))
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func boundedEffortRefs(refs []string, limit int) []string {
	out := uniqueEffortRefs(refs)
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func printEffortPage(token string) {
	if token != "" {
		fmt.Printf("Next page: --page-token %q\n", token)
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
		printGoalHome(r.GetGoalHome())
		printDeliveryPressure(r.GetDeliveryPressure())
		for _, v := range r.Blockers {
			fmt.Println("  Blocker: " + v)
		}
		for _, v := range r.PendingOperations {
			fmt.Println("  Pending: " + v)
		}
		for _, v := range r.Limitations {
			fmt.Println("  Limitation: " + v)
		}
	}
	for _, f := range b.GetDiscovery().GetFindings() {
		fmt.Printf("Discovery %s: %s\n", f.Source, f.Reason)
	}
	printEffortPage(b.NextPageToken)
}

// printGoalHome prints a registered goal home's facts (DL-9). Absent facts
// print as unknown, never zero.
func printGoalHome(g *pb.EffortGoalHome) {
	if g == nil {
		return
	}
	number := func(v *float64) string {
		if v == nil {
			return "unknown"
		}
		return strconv.FormatFloat(*v, 'f', -1, 64)
	}
	fmt.Printf("  Goal: %s (%s, read %s)\n", g.GetPath(), g.GetState(), effortTimestamp(g.GetReadAt()))
	destination := "unknown"
	if g.GetDestinationMetric() != "" {
		destination = fmt.Sprintf("%s %s %s", g.GetDestinationMetric(), g.GetDestinationOp(), number(g.DestinationTarget))
	}
	fmt.Printf("    Destination: %s; current: %s (as of %s)\n", destination, number(g.CurrentValue), effortTimestamp(g.GetCurrentAsOf()))
	fmt.Printf("    Active unit: %s %s workers=%s; last accepted: %s at %s yield=%s\n", effortKnown(g.GetActiveUnit()), g.GetActiveStatus(), effortKnown(strings.Join(g.GetActiveWorkers(), ",")), effortKnown(g.GetLastAcceptedUnit()), effortTimestamp(g.GetLastAcceptedAt()), effortKnown(g.GetLastYield()))
	workarounds := "unknown"
	if g.OpenWorkarounds != nil {
		workarounds = fmt.Sprint(g.GetOpenWorkarounds())
	}
	fmt.Printf("    Open: workarounds=%s feedback=%s; spend per accepted unit=%s weighted (measured %d); canary=%s\n", workarounds, effortKnown(strings.Join(g.GetOpenFeedbackIds(), ",")), number(g.WeightedTokensPerAcceptedUnit), g.GetAcceptedUnitsMeasured(), effortKnown(g.GetCanaryStatus()))
	for _, item := range g.GetNeedsOperator() {
		fmt.Println("    Needs operator: " + item)
	}
	for _, d := range g.GetOpenDecisions() {
		fmt.Println("    Decision: " + decisionLine(d))
	}
	for _, usage := range g.GetCapUsage() {
		limit := "none"
		if usage.Cap != nil {
			limit = fmt.Sprint(usage.GetCap())
		}
		fmt.Printf("    Cap: %s run=%s weighted=%s cap=%s mode=%s\n", effortKnown(usage.GetRole()), usage.GetRunId(), number(usage.WeightedTokens), limit, usage.GetMode())
	}
	if findings := g.GetLintFindings(); len(findings) > 0 {
		fmt.Println("    Lint: " + strings.Join(findings, ", "))
	}
	for _, v := range g.GetLimitations() {
		fmt.Println("    Goal limitation: " + v)
	}
	for _, row := range g.SurfaceScoreboard {
		fmt.Printf("      Surface %s: %s target=%s owner=%s\n", row.Surface, row.State, row.Target, effortKnown(row.OwnerRun))
	}
	for _, unit := range g.ActiveUnits {
		fmt.Printf("      Unit %s: %s workers=%s\n", unit.UnitId, unit.Status, effortKnown(strings.Join(unit.Workers, ",")))
	}
}

func effortKnown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
