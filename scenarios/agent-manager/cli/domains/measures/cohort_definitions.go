package measures

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

type requestFunc func(method, path string, query url.Values, body interface{}) ([]byte, error)

// cohortDefinitionCommand manages the durable named cohorts that --cohort
// selects: `measures cohort define|list|show|delete`.
func cohortDefinitionCommand() cliapp.Command {
	return cliapp.Command{
		Name:        "cohort",
		Description: "Define, list, show or delete durable named cohorts [--json]",
		Usage:       "agent-manager measures cohort define <name> --filter-json '{...}' [--change-binding value] | list | show <name> [--limit n] | delete <name>",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{
				{Name: "action", Description: "define | list | show | delete", Required: true},
				{Name: "name", Description: "Cohort name (define, show, delete)"},
			},
			Flags: []cliapp.Flag{
				{Name: "filter-json", Description: "define: InvocationFilter JSON, e.g. '{\"TagPrefix\":\"plan-x\"}'"},
				{Name: "change-binding", Description: "define: change or release binding"},
				{Name: "limit", Description: "show: maximum matched runs (default 100)"},
			},
		},
		RunCtx: func(ctx cliapp.RunContext) error {
			limit := 0
			if raw := ctx.Flag("limit"); raw != "" {
				value, err := strconv.Atoi(raw)
				if err != nil || value < 1 {
					return fmt.Errorf("--limit must be a positive integer")
				}
				limit = value
			}
			body, err := runCohortAction(ctx.Core().Request, ctx.Positional("action"), ctx.Positional("name"), ctx.Flag("filter-json"), ctx.Flag("change-binding"), limit)
			if err != nil {
				return err
			}
			_, err = ctx.Stdout().Write(append(body, '\n'))
			return err
		},
	}
}

func runCohortAction(request requestFunc, action, name, filterJSON, changeBinding string, limit int) ([]byte, error) {
	name = strings.TrimSpace(name)
	if action != "list" && name == "" {
		return nil, fmt.Errorf("cohort %s requires a <name>", action)
	}
	switch action {
	case "list":
		return request("GET", "runs/cohorts", nil, nil)
	case "define":
		if strings.TrimSpace(filterJSON) == "" {
			return nil, fmt.Errorf("--filter-json is required")
		}
		var filter map[string]any
		if err := json.Unmarshal([]byte(filterJSON), &filter); err != nil {
			return nil, fmt.Errorf("--filter-json must be a JSON object: %w", err)
		}
		return request("POST", "runs/cohorts", nil, map[string]any{"name": name, "filter": filter, "changeBinding": changeBinding})
	case "show":
		query := url.Values{}
		if limit > 0 {
			query.Set("limit", strconv.Itoa(limit))
		}
		return request("GET", "runs/cohorts/"+url.PathEscape(name), query, nil)
	case "delete":
		if _, err := request("DELETE", "runs/cohorts/"+url.PathEscape(name), nil, nil); err != nil {
			return nil, err
		}
		return []byte(`{"deleted":true}`), nil
	default:
		return nil, fmt.Errorf("unknown cohort action %q (want define, list, show or delete)", action)
	}
}
