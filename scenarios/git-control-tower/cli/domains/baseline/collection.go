package baseline

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

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
		return fmt.Errorf("collection requires capture, show, diff, or delete")
	}
	switch args[0] {
	case "capture":
		return runCollectionCapture(core, args[1:])
	case "show":
		return runCollectionShow(core, args[1:])
	case "diff":
		return runCollectionDiff(core, args[1:])
	case "delete":
		return runCollectionDelete(core, args[1:])
	default:
		return fmt.Errorf("unknown collection command %q (use capture, show, diff, or delete)", args[0])
	}
}

func runCollectionDiff(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection diff")
	name, branch, asJSON := collectionFlags(fs)
	operationID := fs.String("operation-id", "", "Durable idempotency key for this collection diff (required)")
	wait := fs.Bool("wait", false, "Block server-side until started collection diff children reach terminal checkpoints")
	var scenarios stringListFlag
	fs.Var(&scenarios, "scenario", "Member scenario to diff; repeat to narrow selection (default: all)")
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
	resp, err := clientFactory(core).StartCollectionDiff(ctx, connect.NewRequest(&baselinesv1.StartCollectionDiffRequest{Name: *name, Branch: *branch, Scenarios: []string(scenarios), OperationId: *operationID}))
	if err != nil {
		return err
	}
	if *wait {
		settled, err := clientFactory(core).GetCollectionDiff(ctx, connect.NewRequest(&baselinesv1.GetCollectionDiffRequest{Name: *name, Branch: *branch, OperationId: *operationID, Wait: true}))
		if err != nil {
			return err
		}
		if *asJSON {
			return printJSON(settled.Msg)
		}
		fmt.Printf("Collection diff %q (%s): %s\n", *name, settled.Msg.GetOperationId(), settled.Msg.GetClassification())
		for _, member := range settled.Msg.GetMembers() {
			fmt.Printf("  %-18s %-14s run=%s\n", member.GetScenario(), member.GetStatus(), member.GetRunId())
		}
		return nil
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	fmt.Printf("Collection diff %q (%s): %s\n", *name, resp.Msg.GetOperationId(), resp.Msg.GetClassification())
	for _, member := range resp.Msg.GetMembers() {
		fmt.Printf("  %-18s %-14s run=%s\n", member.GetScenario(), member.GetStatus(), member.GetRunId())
	}
	return nil
}

func collectionFlags(fs *flag.FlagSet) (name, branch *string, json *bool) {
	name = fs.String("name", "", "Collection name (required)")
	branch = fs.String("branch", "", "Git branch (default: current)")
	json = fs.Bool("json", false, "Emit JSON")
	return name, branch, json
}

func runCollectionCapture(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("baseline collection capture")
	name, branch, asJSON := collectionFlags(fs)
	var members stringListFlag
	fs.Var(&members, "member", "Required member as scenario[:baseline-name]; repeat for every scenario")
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
	resp, err := clientFactory(core).StartCollectionCapture(ctx, connect.NewRequest(&baselinesv1.StartCollectionCaptureRequest{Name: *name, Branch: *branch, Targets: targets, CreatedBy: "agent"}))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	printCollection(resp.Msg.GetCollection(), resp.Msg.GetResumed())
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
		return printJSON(resp.Msg)
	}
	printCollection(resp.Msg.GetCollection(), false)
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
