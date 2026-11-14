package handoffrules

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

	handoffrulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/handoffrules"
	handoffrulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/handoffrules/handoffrules_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client handoffrulesconnect.HandoffRulesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: handoffrulesconnect.NewHandoffRulesServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRules(context.Background(), connect.NewRequest(&handoffrulesv1.ListRulesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("handoff-rule list", err, nil)
	}

	rules := resp.Msg.GetRules()
	rows := make([]string, 0, len(rules))
	for _, r := range rules {
		state := "off"
		if r.GetEnabled() {
			state = "on"
		}
		rows = append(rows, fmt.Sprintf("  %s | %s | %s | %s | %s",
			support.ShortID(r.GetId()), r.GetName(), state, r.GetSource(), r.GetPattern()))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Capture rules: %d", len(rules)),
			"Rules never send anything. They only decide when a suggestion appears.",
		},
		ResultsHeading: "Rules",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s handoff-rule upsert --body-file rule.json", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

type ruleUpsertBody struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Enabled   bool     `json:"enabled"`
	Source    string   `json:"source"`
	Pattern   string   `json:"pattern"`
	Surfaces  []string `json:"surfaces"`
	SortOrder int32    `json:"sort_order"`
}

func (h *handlers) upsert(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body ruleUpsertBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	resp, err := h.client.UpsertRule(context.Background(), connect.NewRequest(&handoffrulesv1.UpsertRuleRequest{
		Id:        body.ID,
		Name:      body.Name,
		Enabled:   body.Enabled,
		Source:    body.Source,
		Pattern:   body.Pattern,
		Surfaces:  body.Surfaces,
		SortOrder: body.SortOrder,
	}))
	if err != nil {
		return cliapp.WrapAPIError("handoff-rule upsert", err, nil)
	}
	r := resp.Msg.GetRule()

	report := cliapp.MutationReport{
		Result: []string{"Saved capture rule"},
		Changes: []string{
			fmt.Sprintf("ID: %s", r.GetId()),
			fmt.Sprintf("Name: %s", r.GetName()),
			fmt.Sprintf("Enabled: %t", r.GetEnabled()),
			fmt.Sprintf("Source: %s", r.GetSource()),
		},
		NextCommand: []string{fmt.Sprintf("%s handoff-rule list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("rule-id")
	if id == "" {
		return fmt.Errorf("usage: handoff-rule delete <rule-id>")
	}

	if _, err := h.client.DeleteRule(context.Background(), connect.NewRequest(&handoffrulesv1.DeleteRuleRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("handoff-rule delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted capture rule %s", id)},
		NextCommand: []string{fmt.Sprintf("%s handoff-rule list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}
