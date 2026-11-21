package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"swarm-manager/cli/internal/support"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "search", Description: "Semantic search and related work", Subcommands: []cliapp.Command{
		support.APICommand("query", "Search backlog, goals, and records (<query> [--type backlog|goal|record])", deps.AISearchQuery),
		relatedCommand(),
		support.APICommand("status", "Show search index availability and coverage", deps.AISearchStatus),
		support.APICommand("reindex", "Reconcile search index [--wait]", deps.AISearchReconcile),
		support.APICommand("reindex-status", "Show the current reindex job status", deps.AISearchReconcileStat),
		support.APICommand("reindex-cancel", "Cancel the running reindex job", deps.AISearchReconcileCan),
	}}
}

func relatedCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "related", NeedsAPI: true, Description: "Find related work (<backlog|goal> <name> or backlog <kind> <name>)", Args: cliapp.ArgSchema{
		Positionals: []cliapp.Positional{
			{Name: "target", Description: "Target type", Required: true},
			{Name: "name", Description: "Goal name, or backlog item kind", Required: true},
			{Name: "extra", Description: "Backlog item name", Repeated: true},
		},
		Flags: []cliapp.Flag{{Name: "exclude-historical", Description: "Exclude archived work and records", Bool: true}, {Name: "limit", Description: "Maximum rows per group", Default: "20"}},
	}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*api.GetRelatedResponse, error) {
			limit, err := strconv.ParseInt(op.Flag("limit"), 10, 32)
			if err != nil || limit < 1 || limit > 1000 {
				return nil, fmt.Errorf("--limit must be between 1 and 1000")
			}
			target, name, extra := strings.TrimSpace(op.Positional("target")), strings.TrimSpace(op.Positional("name")), op.Positionals("extra")
			req := &api.GetRelatedRequest{ExcludeHistorical: op.BoolFlag("exclude-historical"), Limit: int32(limit)}
			switch target {
			case "goal":
				if len(extra) != 0 {
					return nil, fmt.Errorf("usage: swarm-manager search related goal <name>")
				}
				req.Target = &api.GetRelatedRequest_Goal{Goal: &api.RelatedGoalTarget{Name: name}}
			case "backlog":
				if len(extra) != 1 || strings.TrimSpace(extra[0]) == "" {
					return nil, fmt.Errorf("usage: swarm-manager search related backlog <kind> <name>")
				}
				req.Target = &api.GetRelatedRequest_Backlog{Backlog: &api.RelatedBacklogTarget{Kind: name, Name: strings.TrimSpace(extra[0])}}
			default:
				return nil, fmt.Errorf("related target must be backlog or goal")
			}
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := apiconnect.NewRelatedServiceClient(h, base).GetRelated(context.Background(), connect.NewRequest(req))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *api.GetRelatedResponse) cliapp.ListReport {
			results := make([]string, 0)
			for _, group := range response.GetGroups() {
				groupName := group.GetName()
				if group.GetDegraded() {
					groupName += " (degraded)"
				}
				for _, row := range group.GetEntities() {
					results = append(results, fmt.Sprintf("%s: %s — %s [%s] (%s)", groupName, row.GetEntityKind(), row.GetTitle(), row.GetStatus(), strings.Join(row.GetReasons(), "; ")))
				}
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Related work groups: %d", len(response.GetGroups()))}, ResultsHeading: "Related Work", Results: results, RetrievalHints: []string{"Use --exclude-historical to restrict to active work."}}
		},
	))
}
