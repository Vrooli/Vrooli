package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (a *App) cmdWatch(args []string) error {
	return dispatchSubcommand(args, "watch", map[string]subcommandHandler{
		"create": a.watchCreate, "get": a.watchGet, "list": a.watchList,
		"wait": a.watchWait, "cancel": a.watchCancel, "inspect": a.watchInspect,
		"action": a.watchAction, "actions": a.watchActions,
		"policy-candidate": a.watchPolicyCandidate, "policy-evaluate": a.watchPolicyEvaluate, "policy-assess": a.watchPolicyAssess, "policy-promote": a.watchPolicyPromote, "policy-reject": a.watchPolicyReject, "policy-rollback": a.watchPolicyRollback, "policy-disable": a.watchPolicyDisable,
		"policy-get": a.watchPolicyGet, "policy-outcomes": a.watchPolicyOutcomes,
	})
}

func (a *App) watchPolicyGet(args []string) error {
	fs := flag.NewFlagSet("watch policy-get", flag.ContinueOnError)
	version := fs.String("version", "", "Policy version; omit for active")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, response, err := a.services.Watches.GetPolicy(&domainpb.GetSupervisionPolicyRequest{Version: *version})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("%s %s digest=%s supersedes=%s\n", response.GetPolicy().GetVersion(), response.GetState(), response.GetDigest(), response.GetSupersedes())
	return nil
}

func (a *App) watchPolicyOutcomes(args []string) error {
	fs := flag.NewFlagSet("watch policy-outcomes", flag.ContinueOnError)
	version := fs.String("policy-version", "", "Policy version filter")
	watchID := fs.String("watch-id", "", "Filter before the bounded outcome scan")
	limit := fs.Int("limit", 100, "Maximum outcomes (1-500)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, response, err := a.services.Watches.ListOutcomes(&domainpb.ListSupervisionOutcomesRequest{PolicyVersion: *version, Limit: int32(*limit), WatchId: *watchID})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	for _, outcome := range response.GetOutcomes() {
		fmt.Printf("%s policy=%s predicted=%s observed=%s safety=%t\n", outcome.GetOutcomeId(), outcome.GetPolicyVersion(), outcome.GetPredictedClass(), outcome.GetObservedClass(), outcome.GetSafetyViolation())
	}
	return nil
}

func (a *App) watchAction(args []string) error {
	fs := flag.NewFlagSet("watch action", flag.ContinueOnError)
	kind := fs.String("kind", "", "observe, nudge, park, continue, stop, escalate, or wake-parent")
	target := fs.String("target-run", "", "Target child or parent run")
	requestedBy := fs.String("requested-by", "", "Authenticated requester identity")
	authority := fs.String("authority", "", "system, family-parent, or operator")
	idempotencyKey := fs.String("idempotency-key", "", "Replay-safe action key")
	expectedRevision := fs.Uint64("expected-revision", 0, "Current watch revision")
	message := fs.String("message", "", "Bounded turn message or wake evidence")
	rationale := fs.String("rationale", "", "Action rationale")
	cooldown := fs.Duration("cooldown", 0, "Minimum interval between applied matching actions")
	maximum := fs.Uint("maximum-count", 0, "Maximum applied matching actions")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 || *kind == "" || *requestedBy == "" || *authority == "" || *idempotencyKey == "" || *expectedRevision == 0 {
		return fmt.Errorf("watch action requires a watch id, --kind, --requested-by, --authority, --idempotency-key, and --expected-revision")
	}
	parsedKind, err := parseWatchActionKind(*kind)
	if err != nil {
		return err
	}
	parsedAuthority, err := parseWatchAuthority(*authority)
	if err != nil {
		return err
	}
	request := &domainpb.RequestCohortWatchActionRequest{WatchId: fs.Args()[0], ExpectedWatchRevision: *expectedRevision, IdempotencyKey: *idempotencyKey, Kind: parsedKind, TargetRunId: *target, RequestedBy: *requestedBy, Authority: parsedAuthority, Rationale: *rationale, Message: *message, MaximumCount: uint32(*maximum)}
	if *cooldown > 0 {
		request.Cooldown = durationpb.New(*cooldown)
	}
	body, response, err := a.services.Watches.RequestAction(request)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("%s %s state=%s target=%s replay=%t\n", response.GetAction().GetActionId(), response.GetAction().GetKind(), response.GetAction().GetState(), response.GetAction().GetTargetRunId(), response.GetIdempotentReplay())
	if response.GetAction().GetRejectionReason() != "" {
		fmt.Println(response.GetAction().GetRejectionReason())
	}
	return nil
}

func (a *App) watchActions(args []string) error {
	fs := flag.NewFlagSet("watch actions", flag.ContinueOnError)
	limit := fs.Uint("limit", 100, "Maximum actions (1-200)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("watch actions requires a watch id")
	}
	body, response, err := a.services.Watches.ListActions(&domainpb.ListCohortWatchActionsRequest{WatchId: fs.Args()[0], Limit: uint32(*limit)})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	for _, action := range response.GetActions() {
		fmt.Printf("%s %s state=%s target=%s\n", action.GetActionId(), action.GetKind(), action.GetState(), action.GetTargetRunId())
	}
	return nil
}

func (a *App) watchCreate(args []string) error {
	fs := flag.NewFlagSet("watch create", flag.ContinueOnError)
	specFile := fs.String("spec-file", "", "WatchSpec JSON file")
	idempotencyKey := fs.String("idempotency-key", "", "Replay-safe creation key")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *specFile == "" {
		return fmt.Errorf("watch create requires --spec-file")
	}
	raw, err := os.ReadFile(*specFile)
	if err != nil {
		return err
	}
	spec := &domainpb.WatchSpec{}
	if err := protojson.Unmarshal(raw, spec); err != nil {
		return fmt.Errorf("decode watch spec: %w", err)
	}
	body, response, err := a.services.Watches.Create(&domainpb.CreateCohortWatchRequest{Spec: spec, IdempotencyKey: *idempotencyKey})
	return printWatch(body, response, *jsonOut, err)
}

func (a *App) watchGet(args []string) error {
	fs := flag.NewFlagSet("watch get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("watch get requires a watch id")
	}
	body, response, err := a.services.Watches.Get(&domainpb.GetCohortWatchRequest{WatchId: fs.Args()[0]})
	return printWatch(body, response, *jsonOut, err)
}

func (a *App) watchList(args []string) error {
	fs := flag.NewFlagSet("watch list", flag.ContinueOnError)
	family := fs.String("family-execution", "", "Family execution id")
	status := fs.String("status", "", "active, terminal, canceled, or failed")
	pageSize := fs.Uint("page-size", 50, "Maximum watches (1-200)")
	pageToken := fs.String("page-token", "", "Opaque continuation token")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	parsedStatus, err := parseWatchStatus(*status)
	if err != nil {
		return err
	}
	body, response, err := a.services.Watches.List(&domainpb.ListCohortWatchesRequest{FamilyExecutionId: *family, Status: parsedStatus, PageSize: uint32(*pageSize), PageToken: *pageToken})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	for _, watch := range response.GetWatches() {
		fmt.Printf("%s %s revision=%d family=%s\n", watch.GetWatchId(), watch.GetStatus(), watch.GetRevision(), watch.GetSpec().GetFamilyExecutionId())
	}
	if response.GetNextPageToken() != "" {
		fmt.Printf("Next page token: %s\n", response.GetNextPageToken())
	}
	return nil
}

func (a *App) watchWait(args []string) error {
	fs := flag.NewFlagSet("watch wait", flag.ContinueOnError)
	afterRevision := fs.Uint64("after-revision", 0, "Return after this revision")
	timeout := fs.Duration("timeout", 30*time.Second, "Maximum server wait (capped at 30s)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("watch wait requires a watch id")
	}
	body, response, err := a.services.Watches.Wait(&domainpb.WaitCohortWatchRequest{WatchId: fs.Args()[0], AfterRevision: *afterRevision, Timeout: durationpb.New(*timeout)})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("%s %s revision=%d timed_out=%t\n", response.GetWatch().GetWatchId(), response.GetWatch().GetStatus(), response.GetWatch().GetRevision(), response.GetTimedOut())
	return nil
}

func (a *App) watchCancel(args []string) error {
	fs := flag.NewFlagSet("watch cancel", flag.ContinueOnError)
	expectedRevision := fs.Uint64("expected-revision", 0, "Required current watch revision")
	reason := fs.String("reason", "", "Operator reason")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 || *expectedRevision == 0 {
		return fmt.Errorf("watch cancel requires a watch id and --expected-revision")
	}
	body, response, err := a.services.Watches.Cancel(&domainpb.CancelCohortWatchRequest{WatchId: fs.Args()[0], ExpectedRevision: *expectedRevision, Reason: *reason})
	return printWatch(body, response, *jsonOut, err)
}

func (a *App) watchInspect(args []string) error {
	fs := flag.NewFlagSet("watch inspect", flag.ContinueOnError)
	limit := fs.Uint("event-limit", 100, "Maximum pending events (1-1000)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("watch inspect requires a watch id")
	}
	body, response, err := a.services.Watches.Inspect(&domainpb.InspectCohortWatchRequest{WatchId: fs.Args()[0], EventLimit: uint32(*limit)})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("%s events=%d cursor_reset=%t\n", response.GetWatch().GetWatchId(), len(response.GetEvents()), response.GetCursorResetRequired())
	if response.GetResetReason() != "" {
		fmt.Println(response.GetResetReason())
	}
	return nil
}

func printWatch(body []byte, watch *domainpb.CohortWatch, jsonOut bool, err error) error {
	if err != nil {
		return apiError(body, err)
	}
	if jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("%s %s revision=%d family=%s\n", watch.GetWatchId(), watch.GetStatus(), watch.GetRevision(), watch.GetSpec().GetFamilyExecutionId())
	return nil
}

func parseWatchStatus(value string) (domainpb.WatchStatus, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return domainpb.WatchStatus_WATCH_STATUS_UNSPECIFIED, nil
	case "active":
		return domainpb.WatchStatus_WATCH_STATUS_ACTIVE, nil
	case "terminal":
		return domainpb.WatchStatus_WATCH_STATUS_TERMINAL, nil
	case "canceled", "cancelled":
		return domainpb.WatchStatus_WATCH_STATUS_CANCELED, nil
	case "failed":
		return domainpb.WatchStatus_WATCH_STATUS_FAILED, nil
	default:
		return 0, fmt.Errorf("invalid watch status %q", value)
	}
}

func parseWatchActionKind(value string) (domainpb.WatchActionKind, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), "-", "_"))
	if enum, ok := domainpb.WatchActionKind_value["WATCH_ACTION_KIND_"+normalized]; ok && enum != 0 {
		return domainpb.WatchActionKind(enum), nil
	}
	return 0, fmt.Errorf("invalid watch action kind %q", value)
}

func parseWatchAuthority(value string) (domainpb.WatchAuthority, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), "-", "_"))
	if enum, ok := domainpb.WatchAuthority_value["WATCH_AUTHORITY_"+normalized]; ok && enum != 0 {
		return domainpb.WatchAuthority(enum), nil
	}
	return 0, fmt.Errorf("invalid watch authority %q", value)
}
