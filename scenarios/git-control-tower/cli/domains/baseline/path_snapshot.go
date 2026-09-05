package baseline

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

// runPathSnapshot exposes informational source evidence. It intentionally
// never maps a delta to a process exit verdict: only Test Genie diffs are a
// behavioral oracle.
func runPathSnapshot(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("path requires estimate, capture, show, diff, or delete")
	}
	switch args[0] {
	case "estimate":
		return runPathEstimate(core, args[1:])
	case "capture":
		return runPathCapture(core, args[1:])
	case "show":
		return runPathShow(core, args[1:])
	case "diff":
		return runPathDiff(core, args[1:])
	case "delete":
		return runPathDelete(core, args[1:])
	default:
		return fmt.Errorf("unknown path command %q (use estimate, capture, show, diff, or delete)", args[0])
	}
}

func pathFlags(fs *flag.FlagSet) (name, branch *string, asJSON *bool) {
	name = fs.String("name", "", "Path snapshot name (required)")
	branch = fs.String("branch", "", "Git branch (default: current)")
	asJSON = fs.Bool("json", false, "Emit JSON")
	return name, branch, asJSON
}

func runPathCapture(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline path capture")
	name, branch, asJSON := pathFlags(fs)
	retention := fs.Duration("retention", 0, "Bounded source-evidence retention (default: 168h; maximum: 720h)")
	includeIgnored := fs.Bool("include-ignored", false, "Include ignored files explicitly")
	retainContent := fs.Bool("retain-content", false, "Retain bounded text bodies explicitly (default: metadata only)")
	var selections stringListFlag
	fs.Var(&selections, "path", "Safe repo-relative glob; repeat for every selection")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || len(selections) == 0 {
		return fmt.Errorf("--name and at least one --path are required")
	}
	if *retention < 0 || *retention%time.Second != 0 {
		return fmt.Errorf("--retention must be a whole, positive number of seconds")
	}
	resp, err := clientFactory(core).CapturePathSnapshot(context.Background(), connect.NewRequest(&baselinesv1.CapturePathSnapshotRequest{Name: *name, Branch: *branch, Selections: []string(selections), RetentionSeconds: int64(retention.Seconds()), IncludeIgnored: *includeIgnored, RetainContent: *retainContent}))
	if err != nil {
		renderPathSnapshotPolicyError(err, *asJSON)
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	printPathSnapshot(resp.Msg.GetSnapshot(), resp.Msg.GetResumed())
	return nil
}

// renderPathSnapshotPolicyError unwraps the typed server detail rather than
// leaving a direct capture caller with only a generic precondition error. The
// estimate uses the exact same report, so the repair selections are truthful to
// the capture that was refused before it wrote any evidence.
func renderPathSnapshotPolicyError(err error, asJSON bool) {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return
	}
	for _, detail := range connectErr.Details() {
		msg, detailErr := detail.Value()
		if detailErr != nil {
			continue
		}
		violation, ok := msg.(*baselinesv1.PathSnapshotPolicyViolation)
		if !ok || violation.GetEstimate() == nil {
			continue
		}
		if asJSON {
			_ = printJSON(&baselinesv1.EstimatePathSnapshotResponse{Estimate: violation.GetEstimate()})
			return
		}
		fmt.Println("Source evidence capture refused before writing a snapshot:")
		printPathEstimate(violation.GetEstimate())
		return
	}
}

func runPathEstimate(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline path estimate")
	asJSON := fs.Bool("json", false, "Emit JSON")
	includeIgnored := fs.Bool("include-ignored", false, "Include ignored files explicitly")
	retainContent := fs.Bool("retain-content", false, "Project bounded text-body retention")
	var selections stringListFlag
	fs.Var(&selections, "path", "Safe repo-relative glob; repeat for every selection")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(selections) == 0 {
		return fmt.Errorf("at least one --path is required")
	}
	resp, err := clientFactory(core).EstimatePathSnapshot(context.Background(), connect.NewRequest(&baselinesv1.EstimatePathSnapshotRequest{Selections: []string(selections), IncludeIgnored: *includeIgnored, RetainContent: *retainContent}))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	printPathEstimate(resp.Msg.GetEstimate())
	return nil
}

func printPathEstimate(estimate *baselinesv1.PathSnapshotEstimate) {
	if estimate == nil {
		fmt.Println("No source-evidence estimate returned")
		return
	}
	status := "safe"
	if estimate.GetRepairRequired() {
		status = "repair required"
	}
	mode := "metadata only"
	if estimate.GetRetainContent() {
		mode = "retained content"
	}
	fmt.Printf("Source evidence estimate: %s (%s; policy v%d)\n", status, mode, estimate.GetPolicyVersion())
	fmt.Printf("  eligible: %d files, %d bytes; ignored excluded: %d files, %d bytes\n", estimate.GetEligibleFiles(), estimate.GetEligibleBytes(), estimate.GetExcludedIgnoredFiles(), estimate.GetExcludedIgnoredBytes())
	fmt.Printf("  projected retained content: %d bytes\n", estimate.GetRetainedContentBytes())
	for _, issue := range estimate.GetIssues() {
		fmt.Printf("  %s [%s]: %s\n", issue.GetSeverity(), issue.GetCode(), issue.GetDetail())
	}
	for _, contributor := range estimate.GetTopContributors() {
		fmt.Printf("  contributor: %s (%d files, %d bytes)\n", contributor.GetPath(), contributor.GetFiles(), contributor.GetBytes())
	}
	for _, recommendation := range estimate.GetRecommendations() {
		fmt.Printf("  repair: --path %s  # %s\n", recommendation.GetSelection(), recommendation.GetReason())
	}
}

func runPathShow(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline path show")
	name, branch, asJSON := pathFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}
	resp, err := clientFactory(core).GetPathSnapshot(context.Background(), connect.NewRequest(&baselinesv1.GetPathSnapshotRequest{Name: *name, Branch: *branch}))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	printPathSnapshot(resp.Msg.GetSnapshot(), false)
	return nil
}

func runPathDiff(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline path diff")
	before := fs.String("before", "", "Before snapshot name (required)")
	after := fs.String("after", "", "After snapshot name (required)")
	branch := fs.String("branch", "", "Git branch (default: current)")
	asJSON := fs.Bool("json", false, "Emit JSON")
	var selections stringListFlag
	fs.Var(&selections, "path", "Safe repo-relative glob to limit the informational review; repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*before) == "" || strings.TrimSpace(*after) == "" {
		return fmt.Errorf("--before and --after are required")
	}
	resp, err := clientFactory(core).DiffPathSnapshots(context.Background(), connect.NewRequest(&baselinesv1.DiffPathSnapshotsRequest{BeforeName: *before, AfterName: *after, Branch: *branch, Selections: []string(selections)}))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	fmt.Printf("Source evidence (%s): %d path deltas\n", resp.Msg.GetClassification(), len(resp.Msg.GetDeltas()))
	for _, delta := range resp.Msg.GetDeltas() {
		fmt.Printf("  %-10s %s\n", delta.GetStatus(), delta.GetPath())
	}
	return nil
}

func runPathDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline path delete")
	name, branch, asJSON := pathFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}
	resp, err := clientFactory(core).DeletePathSnapshot(context.Background(), connect.NewRequest(&baselinesv1.DeletePathSnapshotRequest{Name: *name, Branch: *branch}))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	fmt.Printf("Deleted source evidence snapshot %q\n", *name)
	return nil
}

func printPathSnapshot(snapshot *baselinesv1.PathSnapshot, resumed bool) {
	if snapshot == nil {
		fmt.Println("No source evidence snapshot returned")
		return
	}
	verb := "Captured"
	if resumed {
		verb = "Resumed"
	}
	mode := "metadata only"
	if snapshot.GetRetainContent() {
		mode = "retained content"
	}
	fmt.Printf("%s informational source evidence %q (%d entries; %s; policy v%d; expires %s; bytes are never displayed)\n", verb, snapshot.GetName(), len(snapshot.GetEntries()), mode, snapshot.GetPolicyVersion(), snapshot.GetExpiresAt())
}
