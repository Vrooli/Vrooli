package hostapp

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostinventory"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func hostDesktopSessionSpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{Name: "desktop-session", Summary: "Inspect a live Linux desktop session and X-server peer", Handler: "desktop-session"}
}
func (app *App) runDesktopSession(ctx *CommandContext, args []string) error {
	flags := flag.NewFlagSet("host desktop-session", flag.ContinueOnError)
	flags.SetOutput(ctx.Stderr)
	id := flags.String("session-id", "", "Exact logind session ID")
	peerPID := flags.Int("peer-pid", 0, "Kernel-authenticated X-server peer PID")
	jsonOutput := flags.Bool("json", false, "Emit canonical JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected desktop-session arguments")
	}
	facts, err := hostinventory.SystemCollector().InspectDesktopSession(context.Background(), *id, *peerPID)
	if err != nil {
		return err
	}
	if *jsonOutput || ctx.Globals.JSON {
		return cliout.WriteProtoJSON(ctx.Stdout, &cliv1.CliDesktopSessionFacts{SessionId: facts.SessionID, Uid: facts.UID, PeerPid: int64(facts.PeerPID), PeerUid: facts.PeerUID, Type: facts.Type, Active: facts.Active, Locked: facts.Locked, Remote: facts.Remote, Matched: facts.Matched, Reason: facts.Reason, ObservedAt: facts.ObservedAt.UTC().Format(time.RFC3339Nano)})
	}
	_, err = fmt.Fprintf(ctx.Stdout, "Session %s: %s\n", facts.SessionID, facts.Reason)
	return err
}
