package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Binding-override administration: `agent-operations overrides {list|set|clear}`.
// list is read-only; set/clear are MUTATING (they administer the override
// documents in domain storage). Resolution is SNAPSHOT-at-Invoke: a change
// affects only operations started afterwards.

func (a *App) cmdAgentOpsOverrides(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-operations overrides {list|set|clear} [options]")
	}
	action, rest := args[0], args[1:]
	switch action {
	case "list":
		return a.cmdAgentOpsOverridesList(rest)
	case "set":
		return a.cmdAgentOpsOverridesSet(rest)
	case "clear":
		return a.cmdAgentOpsOverridesClear(rest)
	default:
		return fmt.Errorf("unknown overrides action %q (want list|set|clear)", action)
	}
}

func agentOpsOwnerFlags(fs *flag.FlagSet) (*string, *string) {
	return fs.String("owner-kind", "", "override owner kind: backlog-item|initiative"),
		fs.String("owner", "", "owner id (backlog item kind/name, or initiative name)")
}

func (a *App) cmdAgentOpsOverridesList(args []string) error {
	fs := flag.NewFlagSet("agent-operations overrides list", flag.ContinueOnError)
	kind, owner := agentOpsOwnerFlags(fs)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *owner)
	if err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().ListBindingOverrides(context.Background(), connect.NewRequest(&apipb.AgentOpsListBindingOverridesRequest{Owner: sel}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	docs := resp.Msg.GetOverrides()
	printSection(fmt.Sprintf("Binding overrides (%d)", len(docs)))
	if len(docs) == 0 {
		fmt.Println("  (no overrides stored at this owner's layer)")
		return nil
	}
	for _, d := range docs {
		b := d.GetBinding()
		disabled := ""
		if b.GetDisabled() {
			disabled = " [DISABLED]"
		}
		version := b.GetOperationVersion()
		if version == "" {
			version = "*"
		}
		fmt.Printf("  %s@%s -> %s@%s%s\n", b.GetOperation(), version, b.GetMode(), b.GetModeRevision(), disabled)
		fmt.Printf("    file     : %s (updated %s)\n", d.GetFile(), d.GetUpdatedAt())
		fmt.Printf("    revision : %s\n", d.GetRevision())
	}
	return nil
}

func (a *App) cmdAgentOpsOverridesSet(args []string) error {
	fs := flag.NewFlagSet("agent-operations overrides set", flag.ContinueOnError)
	kind, owner := agentOpsOwnerFlags(fs)
	operation := fs.String("operation", "", "operation id")
	version := fs.String("version", "", "exact operation version pin (optional; empty binds across versions)")
	mode := fs.String("mode", "", "operating mode the override selects")
	modeRevision := fs.String("mode-revision", "", "exact mode content revision to pin")
	disabled := fs.Bool("disabled", false, "write a fail-closed veto: refuse the operation for this scope")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *owner)
	if err != nil {
		return err
	}
	if *operation == "" {
		return fmt.Errorf("--operation is required")
	}
	if *mode == "" || *modeRevision == "" {
		return fmt.Errorf("--mode and --mode-revision are required")
	}
	if a.globalDry {
		fmt.Println("[dry-run] would write binding override (no mutation performed)")
		return nil
	}
	resp, err := a.agentOperationsClient().PutBindingOverride(context.Background(), connect.NewRequest(&apipb.AgentOpsPutBindingOverrideRequest{
		Owner: sel, Operation: *operation, OperationVersion: *version,
		Mode: *mode, ModeRevision: *modeRevision, Disabled: *disabled,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	b := resp.Msg.GetStored()
	printSection("Binding override stored")
	fmt.Printf("  operation : %s\n", b.GetOperation())
	fmt.Printf("  mode      : %s@%s\n", b.GetMode(), b.GetModeRevision())
	fmt.Printf("  layer     : %s\n", agentOpsLayerLabel(b.GetLayer()))
	fmt.Printf("  file      : %s\n", resp.Msg.GetFile())
	fmt.Printf("  revision  : %s\n", resp.Msg.GetRevision())
	fmt.Println("  note      : resolution is snapshot-at-invoke; running operations keep their pinned binding")
	return nil
}

func (a *App) cmdAgentOpsOverridesClear(args []string) error {
	fs := flag.NewFlagSet("agent-operations overrides clear", flag.ContinueOnError)
	kind, owner := agentOpsOwnerFlags(fs)
	operation := fs.String("operation", "", "operation id")
	version := fs.String("version", "", "exact version pin of the stored override (optional)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *owner)
	if err != nil {
		return err
	}
	if *operation == "" {
		return fmt.Errorf("--operation is required")
	}
	if a.globalDry {
		fmt.Println("[dry-run] would delete binding override (no mutation performed)")
		return nil
	}
	resp, err := a.agentOperationsClient().DeleteBindingOverride(context.Background(), connect.NewRequest(&apipb.AgentOpsDeleteBindingOverrideRequest{
		Owner: sel, Operation: *operation, OperationVersion: *version,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	if resp.Msg.GetFound() {
		fmt.Printf("Override for %s cleared.\n", *operation)
	} else {
		fmt.Printf("No override for %s was stored at this owner's layer (nothing deleted).\n", *operation)
	}
	return nil
}
