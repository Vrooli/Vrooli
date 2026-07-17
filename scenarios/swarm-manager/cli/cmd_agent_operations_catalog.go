package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Catalog + compatibility + resolved-bindings projections over
// AgentOperationsService. Thin Connect clients: the server computes every
// verdict and every winning binding; the CLI only renders the typed result.

// agentOpsEnumLabel projects a proto enum wire name onto the kebab-case domain
// vocabulary (e.g. AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE with the layer
// prefix -> "initiative-override").
func agentOpsEnumLabel(name, prefix string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(name, prefix), "_", "-"))
}

func agentOpsLayerLabel(layer fmt.Stringer) string {
	return agentOpsEnumLabel(layer.String(), "AGENT_OPS_BINDING_LAYER_")
}

func agentOpsTargetKindLabel(kind fmt.Stringer) string {
	return agentOpsEnumLabel(kind.String(), "OPERATING_MODE_TARGET_KIND_")
}

func (a *App) cmdAgentOpsCatalog(args []string) error {
	fs := flag.NewFlagSet("agent-operations catalog", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().ListOperationCatalog(context.Background(), connect.NewRequest(&apipb.AgentOpsListOperationCatalogRequest{}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	entries := resp.Msg.GetEntries()
	printSection(fmt.Sprintf("Operation catalog (%d)", len(entries)))
	if len(entries) == 0 {
		fmt.Println("  (no authored operation contracts)")
		return nil
	}
	for _, e := range entries {
		c := e.GetContract()
		targets := make([]string, 0, len(e.GetCompatibleTargets()))
		for _, k := range e.GetCompatibleTargets() {
			targets = append(targets, agentOpsTargetKindLabel(k))
		}
		fmt.Printf("  %s@%s\n", c.GetId(), c.GetVersion())
		if c.GetSummary() != "" {
			fmt.Printf("    summary  : %s\n", c.GetSummary())
		}
		fmt.Printf("    targets  : %s\n", strings.Join(targets, ", "))
		fmt.Printf("    revision : %s\n", e.GetRevision())
	}
	return nil
}

func (a *App) cmdAgentOpsCompatibleModes(args []string) error {
	fs := flag.NewFlagSet("agent-operations compatible-modes", flag.ContinueOnError)
	kind, target := agentOpsTargetFlags(fs)
	operation := fs.String("operation", "", "narrow verdicts to one catalog operation id (optional)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *target)
	if err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().ListCompatibleModes(context.Background(), connect.NewRequest(&apipb.AgentOpsListCompatibleModesRequest{
		Target: sel, Operation: *operation,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	modes := resp.Msg.GetModes()
	printSection(fmt.Sprintf("Compatible modes (%d)", len(modes)))
	if len(modes) == 0 {
		fmt.Println("  (no authored modes)")
		return nil
	}
	for _, m := range modes {
		fmt.Printf("  %s@%s (target: %s)\n", m.GetMode(), m.GetModeRevision(), agentOpsTargetKindLabel(m.GetTargetKind()))
		for _, v := range m.GetVerdicts() {
			if v.GetCompatible() {
				fmt.Printf("    ✓ %s@%s\n", v.GetOperation(), v.GetOperationVersion())
			} else {
				fmt.Printf("    ✗ %s@%s — %s\n", v.GetOperation(), v.GetOperationVersion(), v.GetReason())
			}
		}
	}
	return nil
}

func (a *App) cmdAgentOpsBindings(args []string) error {
	fs := flag.NewFlagSet("agent-operations bindings", flag.ContinueOnError)
	kind, target := agentOpsTargetFlags(fs)
	verbose := fs.Bool("verbose", false, "show every contributing layer, not just the winner")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *target)
	if err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().GetResolvedBindings(context.Background(), connect.NewRequest(&apipb.AgentOpsGetResolvedBindingsRequest{Target: sel}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	ops := resp.Msg.GetOperations()
	printSection(fmt.Sprintf("Resolved bindings (%d operations)", len(ops)))
	if len(ops) == 0 {
		fmt.Println("  (no catalog operation is compatible with this target)")
		return nil
	}
	for _, op := range ops {
		if !op.GetResolved() {
			fmt.Printf("  %s@%s -> UNRESOLVED (%s: %s)\n", op.GetOperation(), op.GetOperationVersion(), op.GetError(), op.GetErrorMessage())
			continue
		}
		b := op.GetBinding()
		fmt.Printf("  %s@%s -> %s@%s [%s]\n", op.GetOperation(), op.GetOperationVersion(), b.GetMode(), b.GetModeRevision(), agentOpsLayerLabel(b.GetLayer()))
		if op.GetPolicyId() != "" {
			fmt.Printf("    policy : %s (%s)\n", op.GetPolicyId(), op.GetPolicyRevision())
		}
		if *verbose {
			for _, c := range op.GetContributions() {
				marker := " "
				if c.GetWinning() {
					marker = "*"
				}
				cb := c.GetBinding()
				owner := ""
				if cb.GetOwner() != nil {
					owner = fmt.Sprintf(" owner=%s/%s", cb.GetOwner().GetKind(), cb.GetOwner().GetId())
				}
				fmt.Printf("    %s [%s] %s@%s%s\n", marker, agentOpsLayerLabel(cb.GetLayer()), cb.GetMode(), cb.GetModeRevision(), owner)
			}
		}
	}
	return nil
}
