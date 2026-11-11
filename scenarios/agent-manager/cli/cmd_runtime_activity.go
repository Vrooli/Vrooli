package main

import (
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
)

func (s *RunService) Activity() ([]byte, *apipb.GetRuntimeActivityResponse, error) {
	return typedRPC[apipb.GetRuntimeActivityResponse](s.api, apiconnect.AgentManagerServiceGetRuntimeActivityProcedure, &apipb.GetRuntimeActivityRequest{})
}
func (a *App) runActivity(args []string) error {
	fs := flag.NewFlagSet("run activity", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: agent-manager run activity [--json]")
	}
	body, snapshot, err := a.services.Runs.Activity()
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	if snapshot.ExecutingRuns != nil && snapshot.GetComplete() {
		fmt.Printf("Executing: %d (complete physical observation)\n", snapshot.GetExecutingRuns())
	} else {
		fmt.Printf("Executing total: unknown; physically observed: at least %d\n", snapshot.GetObservedExecutingRuns())
	}
	fmt.Printf("Queued records: %d (not executing work)\n%s\n", snapshot.GetLogicalQueuedRuns(), snapshot.GetReason())
	return nil
}
