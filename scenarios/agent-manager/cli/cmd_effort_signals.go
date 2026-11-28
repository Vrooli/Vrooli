// This file implements the effort signal log commands (DL-2): reading the log
// a check-in dispatcher polls, recording a feedback or step_back signal, and
// the step_back signal epoch-check records.
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// effortSignals is `agent-manager effort signals <effort> [--after N] [--limit N] [--json]`.
func (a *App) effortSignals(args []string) error {
	const usage = "usage: agent-manager effort signals <effort-ref> [--after N] [--limit N] [--json]"
	fs := flag.NewFlagSet("effort signals", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	after := fs.Int64("after", 0, "Only signals with seq greater than this cursor")
	limit := fs.Int("limit", 100, "Maximum signals (1-500)")
	ref, rest := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	if ref == "" || fs.NArg() != 0 || *after < 0 || *limit < 1 || *limit > 500 {
		return fmt.Errorf(usage)
	}
	response := &pb.ListEffortSignalsResponse{}
	body, err := callRPC(a.services.Watches.api, apiconnect.AgentManagerServiceListEffortSignalsProcedure, &pb.ListEffortSignalsRequest{EffortRef: ref, AfterSeq: *after, Limit: int32(*limit)}, response)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Effort %s: latest seq %d, %d signal(s) after %d\n", ref, response.GetLatestSeq(), len(response.GetSignals()), *after)
	for _, signal := range response.GetSignals() {
		run := signal.GetRunId()
		if run == "" {
			run = "-"
		}
		fmt.Printf("%6d  %s  %-16s run=%s  %s\n", signal.GetSeq(), effortTimestamp(signal.GetCreatedAt()), signal.GetKind(), run, signal.GetDetail())
	}
	return nil
}

// effortSignal is `agent-manager effort signal <effort> --kind feedback|step_back
// [--detail text] [--run id] [--dedupe-key key] [--json]`. Whoever writes an
// operator or supervisor FEEDBACK.md entry records a feedback signal, so the
// next check-in starts on it. The dedupe key makes a repeat a no-op; by
// default it is kind:run:detail.
func (a *App) effortSignal(args []string) error {
	const usage = "usage: agent-manager effort signal <effort-ref> --kind feedback|step_back [--detail text] [--run run-id] [--dedupe-key key] [--json]"
	fs := flag.NewFlagSet("effort signal", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	kind := fs.String("kind", "", "feedback or step_back")
	detail := fs.String("detail", "", "What happened, e.g. \"FB-070 operator: re-aim at journeys\"")
	runID := fs.String("run", "", "Run of the effort the signal names (optional)")
	dedupeKey := fs.String("dedupe-key", "", "Same key, same signal (default kind:run:detail)")
	ref, rest := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	if ref == "" || fs.NArg() != 0 || (*kind != "feedback" && *kind != "step_back") {
		return fmt.Errorf(usage)
	}
	request := &pb.RecordEffortSignalRequest{EffortRef: ref, Kind: *kind, RunId: strings.TrimSpace(*runID), Detail: strings.TrimSpace(*detail), DedupeKey: strings.TrimSpace(*dedupeKey)}
	response := &pb.EffortSignal{}
	body, err := callRPC(a.services.Watches.api, apiconnect.AgentManagerServiceRecordEffortSignalProcedure, request, response)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Effort %s signal %d: %s %s\n", ref, response.GetSeq(), response.GetKind(), response.GetDetail())
	return nil
}

// recordStepBack records one step_back signal for a fired step-back. The
// dedupe key is the epoch file, its triggers and its log length, so rerunning
// epoch-check on an unchanged epoch records nothing new.
func (a *App) recordStepBack(effortRef, path string, triggers []string, logLines int) (*pb.EffortSignal, error) {
	runID := ""
	if identity := cliutil.DetectIdentity(); identity.IsIdentityPresent() {
		if verified, err := identity.VerifyIdentity(); err == nil && verified != nil && verified.Valid && verified.Claims != nil {
			runID = strings.TrimSpace(verified.Claims.RunID)
		}
	}
	request := &pb.RecordEffortSignalRequest{
		EffortRef: effortRef,
		Kind:      "step_back",
		RunId:     runID,
		Detail:    "STEP_BACK " + strings.Join(triggers, "; ") + " (epoch " + path + ")",
		DedupeKey: fmt.Sprintf("step_back:%s:%s:%d", path, strings.Join(triggers, ";"), logLines),
	}
	response := &pb.EffortSignal{}
	body, err := callRPC(a.services.Watches.api, apiconnect.AgentManagerServiceRecordEffortSignalProcedure, request, response)
	if err != nil {
		return nil, apiError(body, err)
	}
	return response, nil
}
