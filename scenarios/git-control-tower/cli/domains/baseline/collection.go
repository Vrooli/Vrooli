package baseline

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

var collectionExit = os.Exit

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
	fs := newFlagSet("baseline collection diff")
	name, branch, asJSON := collectionFlags(fs)
	operationID := fs.String("operation-id", "", "Durable idempotency key for this collection diff (required)")
	wait := fs.Bool("wait", false, "Block server-side until started collection diff children reach terminal checkpoints")
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
		// Preserve the old convenience flag without maintaining a separate wait
		// lifecycle. The status endpoint is the durable producer authority.
		return runCollectionDiffStatus(core, collectionFollowupArgs(*name, *branch, "--operation-id", *operationID, "--wait", "--json"))
	}
	fmt.Printf("Collection diff %q (%s): %s\n", *name, resp.Msg.GetOperationId(), resp.Msg.GetClassification())
	for _, member := range resp.Msg.GetMembers() {
		fmt.Printf("  %-18s %-14s run=%s\n", member.GetScenario(), member.GetStatus(), member.GetRunId())
	}
	fmt.Printf("  wait once: %s\n", collectionFollowupCommand("diff status", *name, *branch, "--operation-id", resp.Msg.GetOperationId(), "--wait"))
	fmt.Println("  Ctrl-C detaches; rerun that exact status command to recover. Do not poll.")
	if *wait {
		return runCollectionDiffStatus(core, collectionFollowupArgs(*name, *branch, "--operation-id", *operationID, "--wait"))
	}
	return nil
}

func runCollectionDiffStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection diff status")
	name, branch, asJSON := collectionFlags(fs)
	operationID := fs.String("operation-id", "", "Durable collection diff operation id (required)")
	wait := fs.Bool("wait", false, "Wait once for this durable producer operation; Ctrl-C detaches")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*operationID) == "" {
		return fmt.Errorf("--name and --operation-id are required")
	}
	timeout := snapshotStartCeiling
	if *wait {
		timeout = baselineClientTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := clientFactory(core).GetCollectionDiff(ctx, connect.NewRequest(&baselinesv1.GetCollectionDiffRequest{Name: *name, Branch: *branch, OperationId: *operationID, Wait: *wait}))
	if err != nil {
		return err
	}
	if *asJSON {
		if err := printJSON(resp.Msg); err != nil {
			return err
		}
		if code := collectionDiffStatusExitCode(resp.Msg, *wait); code != exitOK {
			collectionExit(code)
		}
		return nil
	}
	fmt.Printf("Collection diff %q (%s): %s\n", *name, resp.Msg.GetOperationId(), resp.Msg.GetClassification())
	for _, member := range resp.Msg.GetMembers() {
		fmt.Printf("  %-18s %-14s run=%s\n", member.GetScenario(), member.GetStatus(), member.GetRunId())
	}
	if outcome := resp.Msg.GetWaitOutcome(); outcome != nil && outcome.GetKind() != "complete" {
		fmt.Printf("  wait outcome: %s — %s\n", outcome.GetKind(), outcome.GetDetail())
		for _, command := range outcome.GetRecoveryCommands() {
			fmt.Printf("  reattach once: %s\n", command)
		}
	}
	if code := collectionDiffStatusExitCode(resp.Msg, *wait); code != exitOK {
		collectionExit(code)
	}
	return nil
}

func collectionDiffStatusExitCode(resp *baselinesv1.GetCollectionDiffResponse, waited bool) int {
	if resp == nil {
		if waited {
			return exitNotReady
		}
		return exitOK
	}
	if outcome := resp.GetWaitOutcome(); outcome != nil {
		if outcome.GetKind() != "complete" {
			if waited {
				return exitNotReady
			}
			return exitOK
		}
	} else if waited {
		return exitNotReady
	}
	return exitCodeForVerdict(resp.GetClassification())
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
	timeout := snapshotStartCeiling
	if *wait {
		timeout = baselineClientTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := clientFactory(core).GetCollection(ctx, connect.NewRequest(&baselinesv1.GetCollectionRequest{Name: *name, Branch: *branch, Wait: *wait}))
	if err != nil {
		return err
	}
	if *asJSON {
		if err := printJSON(resp.Msg); err != nil {
			return err
		}
		if *wait && !resp.Msg.GetCollection().GetCoverage().GetComplete() {
			collectionExit(exitNotReady)
		}
		return nil
	}
	printCollection(resp.Msg.GetCollection(), false)
	if !resp.Msg.GetCollection().GetCoverage().GetComplete() {
		fmt.Printf("  reattach once: %s\n", collectionFollowupCommand("show", *name, *branch, "--wait"))
		if outcome := resp.Msg.GetWaitOutcome(); outcome != nil && outcome.GetDetail() != "" {
			fmt.Printf("  wait outcome: %s — %s\n", outcome.GetKind(), outcome.GetDetail())
		}
		if *wait {
			collectionExit(exitNotReady)
		}
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
