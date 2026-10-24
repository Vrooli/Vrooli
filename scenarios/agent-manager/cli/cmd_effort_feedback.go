package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/effortdecision"
	"agent-manager/internal/goalhome"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

type effortFeedbackResult struct {
	GoalHome      string `json:"goal_home"`
	EffortRef     string `json:"effort_ref"`
	RunID         string `json:"run_id"`
	FeedbackID    string `json:"feedback_id"`
	FileStatus    string `json:"file_status"`
	SignalPresent bool   `json:"signal_present"`
	SignalStatus  string `json:"signal_status"`
	SignalSeq     int64  `json:"signal_seq,omitempty"`
	Detail        string `json:"detail,omitempty"`
}

// effortFeedback writes a supervisor entry before recording the durable
// wake signal. The two stores cannot be atomically committed together, so a
// failed signal is explicitly reported as a file-only partial result.
func (a *App) effortFeedback(args []string) error {
	const usage = "usage: agent-manager effort feedback add <goal-home> --effort-ref <ref> --run <run-id> --id <id> --title <text> --detail <text> [--json]"
	if len(args) == 0 || args[0] != "add" {
		return errors.New(usage)
	}
	fs := flag.NewFlagSet("effort feedback add", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	effortRef := fs.String("effort-ref", "", "Exact enrolled effort reference")
	runID := fs.String("run", "", "Exact run carrying the active effort reference")
	id := fs.String("id", "", "Stable FEEDBACK entry ID")
	title := fs.String("title", "", "Single-line feedback title")
	detail := fs.String("detail", "", "Feedback detail")
	dir, rest := goalHomeArg(args[1:])
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	*effortRef, *runID, *id, *title, *detail = strings.TrimSpace(*effortRef), strings.TrimSpace(*runID), strings.TrimSpace(*id), strings.TrimSpace(*title), strings.TrimSpace(*detail)
	if dir == "" || fs.NArg() != 0 || *effortRef == "" || *runID == "" || *id == "" || *title == "" || *detail == "" {
		return errors.New(usage)
	}
	if err := requireUUIDFlag("--run", *runID); err != nil {
		return err
	}
	*runID = strings.ToLower(*runID)
	body, run, err := a.services.Runs.Get(*runID)
	if err != nil {
		return apiError(body, err)
	}
	if run == nil {
		return errors.New("run owner returned no run; FEEDBACK.md was not changed")
	}
	actualRunID := strings.TrimSpace(run.GetId())
	if actualRunID == "" || requireUUIDFlag("run owner ID", actualRunID) != nil || strings.ToLower(actualRunID) != *runID {
		return errors.New("run owner returned a different or invalid run ID; FEEDBACK.md was not changed")
	}
	if !hasSupervisorEffortReference(run, *effortRef) {
		return fmt.Errorf("run %s does not carry a public, verified, active supervisor/orchestrator reference to %s; FEEDBACK.md was not changed", *runID, *effortRef)
	}
	homeRef, err := canonicalFeedbackGoalHome(dir)
	if err != nil {
		return fmt.Errorf("goal home: %w", err)
	}
	if err := a.validateFeedbackGoalHome(*effortRef, homeRef); err != nil {
		return err
	}

	feedbackPath := filepath.Join(dir, goalhome.FeedbackFile)
	original, err := os.ReadFile(feedbackPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	updated, changed, err := goalhome.AddSupervisorFeedback(original, *id, *title, *detail, time.Now().UTC())
	if err != nil {
		return err
	}
	result := effortFeedbackResult{GoalHome: dir, EffortRef: *effortRef, RunID: *runID, FeedbackID: *id, FileStatus: "already_present"}
	if changed {
		if err := writeGoalHomeFile(feedbackPath, original, updated); err != nil {
			return fmt.Errorf("FEEDBACK entry was not written: %w", err)
		}
		result.FileStatus = "written"
	}
	request := &domain.RecordEffortSignalRequest{
		EffortRef: *effortRef,
		Kind:      "feedback",
		RunId:     *runID,
		Detail:    *id + " — " + *title,
		DedupeKey: "feedback:" + *id,
	}
	response := &domain.EffortSignal{}
	body, err = a.services.Watches.call(apiconnect.AgentManagerServiceRecordEffortSignalProcedure, request, response)
	if err != nil {
		result.SignalStatus = "unknown"
		result.Detail = "FEEDBACK entry is present; signal request failed and may have committed. Retry the same ID and content. " + apiError(body, err).Error()
		printEffortFeedbackResult(*jsonOut, result)
		return errors.New(result.Detail)
	}
	if response.GetSeq() < 1 || response.GetEffortRef() != *effortRef || response.GetKind() != "feedback" || response.GetRunId() != *runID || response.GetDetail() != request.Detail {
		result.SignalStatus = "conflict"
		result.Detail = "FEEDBACK entry is present, but the dedupe key resolved to a different signal; inspect the signal log before retrying"
		printEffortFeedbackResult(*jsonOut, result)
		return errors.New(result.Detail)
	}
	result.SignalPresent = true
	result.SignalStatus = "present_or_replayed"
	result.SignalSeq = response.GetSeq()
	printEffortFeedbackResult(*jsonOut, result)
	return nil
}

func canonicalFeedbackGoalHome(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	root := cliutil.ResolveRepoRoot()
	if root == "" {
		return "", errors.New("repository root unknown; feedback must target the enrolled goal home")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || !filepath.IsLocal(rel) {
		return "", errors.New("feedback path must resolve inside the repository to the enrolled goal home")
	}
	return effortdecision.NormalizeGoalHome(filepath.ToSlash(rel))
}

func (a *App) validateFeedbackGoalHome(effortRef, homeRef string) error {
	response := &domain.EffortBoard{}
	request := &domain.GetEffortBoardRequest{EffortRef: effortRef, PageSize: 2}
	body, err := a.services.Watches.call(apiconnect.AgentManagerServiceGetEffortBoardProcedure, request, response)
	if err != nil {
		return fmt.Errorf("cannot confirm the enrolled goal home; FEEDBACK.md was not changed: %w", apiError(body, err))
	}
	if len(response.GetRows()) != 1 {
		return errors.New("effort owner did not return exactly one enrollment row; FEEDBACK.md was not changed")
	}
	row := response.GetRows()[0]
	if row == nil || row.GetEnrollment() == nil || row.GetGoalHome() == nil || row.GetEnrollment().GetEffortRef() != effortRef || row.GetEnrollment().GetWithdrawn() || !strings.HasPrefix(row.GetEnrollment().GetDestinationRef(), "path:") || row.GetGoalHome().GetPath() != homeRef {
		return fmt.Errorf("%s is not the currently enrolled goal home for %s; FEEDBACK.md was not changed", homeRef, effortRef)
	}
	return nil
}

func hasSupervisorEffortReference(run *domain.Run, effortRef string) bool {
	matches := 0
	for _, ref := range run.GetWorkReferences() {
		if ref.GetKind() != "effort" || ref.GetId() != effortRef || !ref.GetVerified() || ref.GetVisibility() != eventdomain.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC || ref.GetState() != eventdomain.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE {
			continue
		}
		if ref.GetRelationship() != "orchestrator" && ref.GetRelationship() != "supervisor" {
			continue
		}
		matches++
	}
	return matches == 1
}

func printEffortFeedbackResult(jsonOut bool, result effortFeedbackResult) {
	if jsonOut {
		data, _ := json.Marshal(result)
		cliutil.PrintJSON(data)
		return
	}
	fmt.Printf("Effort feedback %s: FEEDBACK.md %s; signal %s", result.FeedbackID, result.FileStatus, result.SignalStatus)
	if result.SignalSeq > 0 {
		fmt.Printf(" (seq %d)", result.SignalSeq)
	}
	if result.Detail != "" {
		fmt.Printf("; %s", result.Detail)
	}
	fmt.Println()
}
