package baseline

import (
	"context"
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
		return fmt.Errorf("path requires capture, show, diff, or delete")
	}
	switch args[0] {
	case "capture":
		return runPathCapture(core, args[1:])
	case "show":
		return runPathShow(core, args[1:])
	case "diff":
		return runPathDiff(core, args[1:])
	case "delete":
		return runPathDelete(core, args[1:])
	default:
		return fmt.Errorf("unknown path command %q (use capture, show, diff, or delete)", args[0])
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
	resp, err := clientFactory(core).CapturePathSnapshot(context.Background(), connect.NewRequest(&baselinesv1.CapturePathSnapshotRequest{Name: *name, Branch: *branch, Selections: []string(selections), RetentionSeconds: int64(retention.Seconds())}))
	if err != nil {
		return err
	}
	if *asJSON {
		return printJSON(resp.Msg)
	}
	printPathSnapshot(resp.Msg.GetSnapshot(), resp.Msg.GetResumed())
	return nil
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
	fmt.Printf("%s informational source evidence %q (%d entries; expires %s; bytes are never displayed)\n", verb, snapshot.GetName(), len(snapshot.GetEntries()), snapshot.GetExpiresAt())
}
