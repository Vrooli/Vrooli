package privilegebroker

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/signal"
)

const (
	runParameterA = 2
)

// RunServiceCommand is intentionally an internal entry point invoked only by
// the root-owned systemd unit. It exposes no user-facing generic command.
func RunServiceCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "serve" {
		fmt.Fprintln(stderr, "privilege broker requires the serve command")
		return runParameterA
	}
	flags := flag.NewFlagSet("privilege-broker serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	socket := flags.String("socket", DefaultSocketPath, "")
	allowedUID := flags.Uint("allowed-uid", 0, "")
	socketGID := flags.Int("socket-gid", -1, "")
	auditPath := flags.String("audit-path", defaultAuditPath, "")
	runtimeHomeRoot := flags.String("runtime-home-root", "", "")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *allowedUID == 0 || *allowedUID > uint(^uint32(0)) {
		if err == nil {
			fmt.Fprintln(stderr, "valid --allowed-uid is required")
		}
		return runParameterA
	}
	runtimeRepair := executeRuntimeHomeRepair
	if *runtimeHomeRoot != "" {
		fixedRoot := *runtimeHomeRoot
		runtimeRepair = func(ctx context.Context, subject RuntimeHomeSubject) Result {
			return executeRuntimeHomeRepairAt(ctx, subject, fixedRoot)
		}
	}
	b, err := New(Config{SocketPath: *socket, AllowedUID: uint32(*allowedUID), SocketGID: *socketGID, AuditPath: *auditPath, RuntimeHomeRepair: runtimeRepair})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), serviceSignals()...)
	defer stop()
	if err := b.Serve(ctx); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
