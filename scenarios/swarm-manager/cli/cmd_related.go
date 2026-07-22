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
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

func (a *App) relatedClient() apiconnect.RelatedServiceClient {
	h, b := cliapp.NewConnectHTTPClient(a.core)
	return apiconnect.NewRelatedServiceClient(h, b)
}
func (a *App) cmdRelatedBacklog(args []string) error { return a.cmdRelated(args, true) }
func (a *App) cmdRelatedGoal(args []string) error    { return a.cmdRelated(args, false) }
func (a *App) cmdRelated(args []string, backlog bool) error {
	fs := flag.NewFlagSet("related", flag.ContinueOnError)
	exclude := fs.Bool("exclude-historical", false, "exclude archived work and records")
	limit := fs.Int("limit", 20, "maximum rows per group")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	pos := fs.Args()
	req := &api.GetRelatedRequest{ExcludeHistorical: *exclude, Limit: int32(*limit)}
	if backlog {
		if len(pos) != 2 {
			return fmt.Errorf("usage: swarm-manager related backlog <kind> <name>")
		}
		req.Target = &api.GetRelatedRequest_Backlog{Backlog: &api.RelatedBacklogTarget{Kind: strings.TrimSpace(pos[0]), Name: strings.TrimSpace(pos[1])}}
	} else {
		if len(pos) != 1 {
			return fmt.Errorf("usage: swarm-manager related goal <name>")
		}
		req.Target = &api.GetRelatedRequest_Goal{Goal: &api.RelatedGoalTarget{Name: strings.TrimSpace(pos[0])}}
	}
	response, err := a.relatedClient().GetRelated(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, response.Msg)
	}
	fmt.Println("Related work")
	for _, group := range response.Msg.Groups {
		fmt.Printf("\n%s", group.Name)
		if group.Degraded {
			fmt.Print(" (degraded)")
		}
		fmt.Println(":")
		if len(group.Entities) == 0 {
			fmt.Println("  No results.")
		}
		for _, row := range group.Entities {
			fmt.Printf("  %s — %s [%s] (%s)\n", row.EntityKind, row.Title, row.Status, strings.Join(row.Reasons, "; "))
		}
	}
	fmt.Println("\nRetrieval Hints: use --json for generated wire output; use --exclude-historical to restrict to active work.")
	return nil
}
