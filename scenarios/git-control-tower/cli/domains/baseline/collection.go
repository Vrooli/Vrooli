package baseline

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/operationstanding"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

// renderedExitError lets the top-level runner map an already-rendered result
// to its process code. Command helpers must never call os.Exit: doing so can
// truncate JSON and makes the result impossible to test as a value.
type renderedExitError struct{ code int }

func (e renderedExitError) Error() string {
	return fmt.Sprintf("operation finished with exit code %d", e.code)
}
func (e renderedExitError) ExitCode() int { return e.code }
func (e renderedExitError) Silent() bool  { return true }

// stringListFlag deliberately uses repeated flags rather than a comma grammar:
// scenario names and future baseline identities remain opaque values.
type stringListFlag []string

func (v *stringListFlag) String() string { return strings.Join(*v, ",") }
func (v *stringListFlag) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func runCollection(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("collection requires capture, show, extend, diff, or delete")
	}
	switch args[0] {
	case "capture":
		return runCollectionCapture(core, args[1:])
	case "show":
		return runCollectionShow(core, args[1:])
	case "extend":
		return runCollectionExtend(core, args[1:])
	case "diff":
		return runCollectionDiff(core, args[1:])
	case "delete":
		return runCollectionDelete(core, args[1:])
	default:
		return fmt.Errorf("unknown collection command %q (use capture, show, extend, diff, or delete)", args[0])
	}
}

func runCollectionDiff(core *cliapp.ScenarioApp, args []string) error {
	if len(args) > 0 && args[0] == "status" {
		return runCollectionDiffStatus(core, args[1:])
	}
	if len(args) > 0 && args[0] == "wait" {
		return runCollectionDiffWait(core, args[1:])
	}
	fs := newFlagSet("baseline collection diff")
	name, branch, asJSON := collectionFlags(fs)
	operationID := fs.String("operation-id", "", "Durable idempotency key for this collection diff (required)")
	wait := fs.Bool("wait", false, "Convenience: wait once on the started parent operation")
	var members, scenarios stringListFlag
	fs.Var(&members, "member", "Captured collection member to diff; repeat to narrow selection (default: all)")
	fs.Var(&scenarios, "scenario", "Deprecated alias for --member")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*operationID) == "" {
		return fmt.Errorf("--name and --operation-id are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), snapshotStartCeiling)
	defer cancel()
	selection := append([]string(nil), members...)
	selection = append(selection, scenarios...)
	resp, err := clientFactory(core).StartCollectionDiff(ctx, connect.NewRequest(&baselinesv1.StartCollectionDiffRequest{Name: *name, Branch: *branch, Scenarios: selection, OperationId: *operationID}))
	if err != nil {
		return err
	}
	if *asJSON {
		if !*wait {
			return printJSON(resp.Msg)
		}
		return runCollectionDiffWait(core, collectionFollowupArgs(*name, *branch, "--operation-id", *operationID, "--json"))
	}
	fmt.Printf("Collection diff %q (%s): %s\n", *name, resp.Msg.GetOperationId(), resp.Msg.GetClassification())
	for _, member := range resp.Msg.GetMembers() {
		fmt.Printf("  %-18s %-14s run=%s\n", member.GetScenario(), member.GetStatus(), member.GetRunId())
	}
	fmt.Printf("  wait once: %s\n", collectionFollowupCommand("diff wait", *name, *branch, "--operation-id", resp.Msg.GetOperationId()))
	fmt.Println("  Ctrl-C detaches; rerun that exact status command to recover. Do not poll.")
	if *wait {
		return runCollectionDiffWait(core, collectionFollowupArgs(*name, *branch, "--operation-id", *operationID))
	}
	return nil
}

func runCollectionDiffStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection diff status")
	name, branch, asJSON := collectionFlags(fs)
	operationID := fs.String("operation-id", "", "Durable collection diff operation id (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*operationID) == "" {
		return fmt.Errorf("--name and --operation-id are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), snapshotStartCeiling)
	defer cancel()
	resp, err := clientFactory(core).GetCollectionDiffStatus(ctx, connect.NewRequest(&baselinesv1.GetCollectionDiffStatusRequest{Name: *name, Branch: *branch, OperationId: *operationID}))
	if err != nil {
		return err
	}
	if *asJSON {
		if err := printJSON(resp.Msg); err != nil {
			return err
		}
		return nil
	}
	fmt.Printf("Collection diff %q (%s): %s\n", *name, resp.Msg.GetOperationId(), resp.Msg.GetClassification())
	for _, member := range resp.Msg.GetMembers() {
		fmt.Printf("  %-18s %-14s run=%s\n", member.GetScenario(), member.GetStatus(), member.GetRunId())
	}
	printCollectionStanding(resp.Msg.GetStanding())
	return nil
}

func runCollectionDiffWait(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection diff wait")
	name, branch, asJSON := collectionFlags(fs)
	operationID := fs.String("operation-id", "", "Durable collection diff operation id (required)")
	timeout := fs.Int("timeout", 0, "Max seconds to wait (0 = until terminal)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*operationID) == "" {
		return fmt.Errorf("--name and --operation-id are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), baselineClientTimeout)
	defer cancel()
	resp, err := clientFactory(core).WaitCollectionDiff(ctx, connect.NewRequest(&baselinesv1.WaitCollectionDiffRequest{Name: *name, Branch: *branch, OperationId: *operationID, TimeoutSeconds: int32(*timeout)}))
	if err != nil {
		return err
	}
	if *asJSON {
		if err := printJSON(resp.Msg); err != nil {
			return err
		}
	} else {
		fmt.Printf("Collection diff %q (%s): %s\n", *name, resp.Msg.GetOperationId(), resp.Msg.GetClassification())
		for _, member := range resp.Msg.GetMembers() {
			fmt.Printf("  %-18s %-14s run=%s\n", member.GetScenario(), member.GetStatus(), member.GetRunId())
		}
		printCollectionStanding(resp.Msg.GetStanding())
	}
	if resp.Msg.GetDetached() {
		return renderedExitError{code: 124}
	}
	return renderedExitForStanding(resp.Msg.GetStanding(), resp.Msg.GetClassification())
}

func renderedExitForStanding(standing interface{ GetLifecycle() string }, classification string) error {
	if standing.GetLifecycle() != "terminal" {
		return renderedExitError{code: 124}
	}
	if code := exitCodeForVerdict(classification); code != exitOK {
		return renderedExitError{code: code}
	}
	return nil
}

func printCollectionStanding(standing *commonv1.OperationStanding) {
	if err := operationstanding.WriteText(os.Stdout, standing); err != nil {
		fmt.Printf("  lifecycle rendering failed: %v\n", err)
	}
}

func collectionFlags(fs *flag.FlagSet) (name, branch *string, json *bool) {
	name = fs.String("name", "", "Collection name (required)")
	branch = fs.String("branch", "", "Git branch (default: current)")
	json = fs.Bool("json", false, "Emit JSON")
	return name, branch, json
}

// collectionFollowupArgs produces a parse-safe argument list for a reattach
// command. A blank --branch value would make Go's flag parser consume the next
// flag (commonly --wait) as its value, silently defeating the reattach.
func collectionFollowupArgs(name, branch string, tail ...string) []string {
	args := []string{"--name", name}
	if branch = strings.TrimSpace(branch); branch != "" {
		args = append(args, "--branch", branch)
	}
	return append(args, tail...)
}

func collectionFollowupCommand(action, name, branch string, tail ...string) string {
	args := append([]string{"git-control-tower", "baseline", "collection", action}, collectionFollowupArgs(name, branch, tail...)...)
	return strings.Join(args, " ")
}

func runCollectionCapture(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection capture")
	name, branch, asJSON := collectionFlags(fs)
	includeIgnored := fs.Bool("include-ignored", false, "Include ignored source-evidence files explicitly")
	retainContent := fs.Bool("retain-content", false, "Retain bounded source text explicitly (default: metadata only)")
	var members, paths stringListFlag
	fs.Var(&members, "member", "Required member as scenario[:baseline-name]; repeat for every scenario")
	fs.Var(&paths, "path", "Safe repo-relative source-evidence glob; repeat (informational only)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || len(members) == 0 {
		return fmt.Errorf("--name and at least one --member are required")
	}
	targets := make([]*baselinesv1.CollectionTarget, 0, len(members))
	for _, raw := range members {
		target, err := parseCollectionMember(raw, *name)
		if err != nil {
			return err
		}
		targets = append(targets, target)
	}
	ctx, cancel := context.WithTimeout(context.Background(), snapshotStartCeiling)
	defer cancel()
	resp, err := clientFactory(core).StartCollectionCapture(ctx, connect.NewRequest(&baselinesv1.StartCollectionCaptureRequest{Name: *name, Branch: *branch, Targets: targets, PathSelections: paths, IncludeIgnored: *includeIgnored, RetainContent: *retainContent, CreatedBy: "agent"}))
	if err != nil {
		renderPathSnapshotPolicyError(err, *asJSON)
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	printCollection(resp.Msg.GetCollection(), resp.Msg.GetResumed())
	fmt.Printf("  wait once: %s\n", collectionFollowupCommand("show", *name, *branch, "--wait"))
	fmt.Println("  Ctrl-C detaches; rerun that exact command to recover. Do not poll.")
	return nil
}

func parseCollectionMember(raw, defaultName string) (*baselinesv1.CollectionTarget, error) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if parts[0] == "" {
		return nil, fmt.Errorf("collection member scenario is required")
	}
	baseline := defaultName
	if len(parts) == 2 {
		baseline = strings.TrimSpace(parts[1])
	}
	if baseline == "" {
		return nil, fmt.Errorf("collection member %q baseline name is required", parts[0])
	}
	return &baselinesv1.CollectionTarget{Scenario: parts[0], BaselineName: baseline, Required: true}, nil
}

func runCollectionShow(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection show")
	name, branch, asJSON := collectionFlags(fs)
	wait := fs.Bool("wait", false, "Block server-side while pending collection members finalize")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), baselineClientTimeout)
	defer cancel()
	var collection *baselinesv1.BaselineCollection
	var standing *commonv1.OperationStanding
	detached := false
	if *wait {
		resp, err := clientFactory(core).WaitCollectionCapture(ctx, connect.NewRequest(&baselinesv1.WaitCollectionCaptureRequest{Name: *name, Branch: *branch}))
		if err != nil {
			return err
		}
		collection, standing, detached = resp.Msg.GetCollection(), resp.Msg.GetStanding(), resp.Msg.GetDetached()
	} else {
		resp, err := clientFactory(core).GetCollectionStatus(ctx, connect.NewRequest(&baselinesv1.GetCollectionStatusRequest{Name: *name, Branch: *branch}))
		if err != nil {
			return err
		}
		collection, standing = resp.Msg.GetCollection(), resp.Msg.GetStanding()
	}
	if *asJSON {
		var payload any = &baselinesv1.GetCollectionStatusResponse{Collection: collection, Standing: standing}
		if *wait {
			payload = &baselinesv1.WaitCollectionCaptureResponse{Collection: collection, Standing: standing, Detached: detached}
		}
		if err := printJSON(payload); err != nil {
			return err
		}
		if detached {
			return renderedExitError{code: 124}
		}
		return nil
	}
	printCollection(collection, false)
	printCollectionStanding(standing)
	if detached {
		return renderedExitError{code: 124}
	}
	return nil
}

func runCollectionExtend(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection extend")
	name, branch, asJSON := collectionFlags(fs)
	var members stringListFlag
	fs.Var(&members, "member", "New pre-edit member as scenario[:baseline-name]; repeat as needed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || len(members) == 0 {
		return fmt.Errorf("--name and at least one --member are required")
	}
	targets := make([]*baselinesv1.CollectionTarget, 0, len(members))
	for _, raw := range members {
		target, err := parseCollectionMember(raw, *name)
		if err != nil {
			return err
		}
		targets = append(targets, target)
	}
	ctx, cancel := context.WithTimeout(context.Background(), snapshotStartCeiling)
	defer cancel()
	resp, err := clientFactory(core).ExtendCollection(ctx, connect.NewRequest(&baselinesv1.ExtendCollectionRequest{Name: *name, Branch: *branch, Targets: targets, CreatedBy: "agent"}))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	fmt.Printf("Extended collection %q with: %s\n", *name, strings.Join(resp.Msg.GetAddedScenarios(), ", "))
	printCollection(resp.Msg.GetCollection(), resp.Msg.GetResumed())
	fmt.Printf("  wait once: %s\n", collectionFollowupCommand("show", *name, *branch, "--wait"))
	return nil
}

func runCollectionDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection delete")
	name, branch, asJSON := collectionFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}
	resp, err := clientFactory(core).DeleteCollection(context.Background(), connect.NewRequest(&baselinesv1.DeleteCollectionRequest{Name: *name, Branch: *branch}))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	fmt.Printf("Deleted collection %q\n", *name)
	return nil
}
