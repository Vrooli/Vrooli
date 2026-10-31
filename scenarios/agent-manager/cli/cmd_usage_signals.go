// This file holds `usage signals status|off|on|purge`: the runtime switch for
// operator signals (labels a local model puts on your messages). Changes are
// operator-only; the API owns authority, the switch file and the purge.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"agent-manager/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

const usageSignalsUsage = `agent-manager usage signals status [--json]
  agent-manager usage signals off [--json]
  agent-manager usage signals on [--json]
  agent-manager usage signals purge --yes [--json]`

const usageSignalsHelp = `Operator signals are labels (reported failure, correction, frustration, satisfaction, neutral) that a local decision model puts on your messages after agent work. Message text is read only to classify it, goes to local models only, and is never stored; only the label and its probability are.

status   shows whether signals are on, what decides it, and how many labels are stored. Agents may read it.
off      stops reading messages and classifying at once; no restart. Stored labels stay until purged.
on       turns them back on. AGENT_MANAGER_OPERATOR_SIGNALS=off in the environment still forces them off.
purge    deletes every stored label (requires --yes). Neither off nor purge needs a restart.

off, on and purge need verified operator authority; an agent run is refused with 403.`

func (a *App) usageSignalsCommand() cliapp.Command {
	command := support.Command("signals", "Operator-signal switch and stored labels: status|off|on|purge [--json]", a.usageSignals)
	command.Usage = usageSignalsUsage
	command.HelpText = usageSignalsHelp
	return command
}

func (a *App) usageSignals(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Println("agent-manager usage signals " + strings.TrimPrefix(usageSignalsUsage, "agent-manager usage signals "))
		fmt.Println()
		fmt.Println(usageSignalsHelp)
		return nil
	}
	fs := flag.NewFlagSet("usage signals "+args[0], flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	yes := fs.Bool("yes", false, "confirm purge")
	fs.Bool("local-owner", false, "accepted for symmetry with other operator commands")
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	ctx := context.Background()
	switch args[0] {
	case "status":
		response, err := a.modeClient(false).UsageSignalsStatus(ctx, connect.NewRequest(&measurepb.UsageSignalsStatusRequest{}))
		if err != nil {
			return modeRPCError(err)
		}
		if *jsonOut {
			return printProtoJSON(response.Msg)
		}
		printSignalsStatus(response.Msg)
		return nil
	case "off", "on", "purge":
		if args[0] == "purge" && !*yes {
			return errors.New("purge deletes every stored operator-signal label; re-run with --yes")
		}
		response, err := a.modeClient(true).SetUsageSignals(ctx, connect.NewRequest(&measurepb.SetUsageSignalsRequest{Action: args[0]}))
		if err != nil {
			return modeRPCError(err)
		}
		if *jsonOut {
			return printProtoJSON(response.Msg)
		}
		if args[0] == "purge" {
			fmt.Printf("Purged %d stored operator-signal labels.\n", response.Msg.GetPurged())
		}
		printSignalsStatus(response.Msg.GetStatus())
		return nil
	}
	return fmt.Errorf("unknown usage signals subcommand %q; expected status, off, on or purge", args[0])
}

func printSignalsStatus(s *measurepb.UsageSignalsStatusResponse) {
	state := "off"
	if s.GetEnabled() {
		state = "on"
	}
	fmt.Printf("Operator signals are %s (decided by: %s); %d labels stored.\n", state, s.GetSource(), s.GetStored())
	if s.GetChangedBy() != "" {
		fmt.Printf("Last changed %s by %s.\n", s.GetChangedAt(), s.GetChangedBy())
	}
	if s.GetNote() != "" {
		fmt.Println(s.GetNote())
	}
}
