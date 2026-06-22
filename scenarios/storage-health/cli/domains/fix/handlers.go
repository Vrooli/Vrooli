package fix

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunContext func
// has typed access to the generated Connect client without re-resolving it.
type handlers struct {
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

// fixTimeout is the HTTP client timeout for the Fix RPCs. The fixers are static
// (AST + filesystem) so they are fast; a 60s ceiling mirrors the validate path.
const fixTimeout = 60 * time.Second

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, fixTimeout)
	return &handlers{client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)}
}

// preview calls ScenarioValidationService.PreviewFix and renders the candidate
// edits without writing anything.
func (h *handlers) preview(ctx cliapp.RunContext) error {
	return h.run(ctx, false)
}

// apply calls ScenarioValidationService.ApplyFix and renders what changed.
func (h *handlers) apply(ctx cliapp.RunContext) error {
	return h.run(ctx, true)
}

func (h *handlers) run(ctx cliapp.RunContext, apply bool) error {
	name := ctx.Positional("name")
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: name,
		RuleIds:  splitCSV(ctx.FlagValues("rule")),
	})

	var (
		resp *connect.Response[scenariovalidationv1.FixResponse]
		err  error
	)
	if apply {
		resp, err = h.client.ApplyFix(context.Background(), req)
	} else {
		resp, err = h.client.PreviewFix(context.Background(), req)
	}
	if err != nil {
		verb := "preview"
		if apply {
			verb = "apply"
		}
		return cliapp.WrapAPIError(fmt.Sprintf("%s storage autofix for %q", verb, name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fix response")
	}

	msg := resp.Msg
	changes := make([]string, 0, len(msg.GetCandidates()))
	for _, c := range msg.GetCandidates() {
		changes = append(changes, fmt.Sprintf("[%s] %s — %s (applied=%v)",
			c.GetRuleId(), c.GetFilePath(), c.GetDescription(), c.GetApplied()))
	}

	verb := "preview"
	if msg.GetApplied() {
		verb = "applied"
	}
	result := []string{fmt.Sprintf("%s %s: %d storage fix candidate(s).", verb, msg.GetScenario(), len(msg.GetCandidates()))}
	result = append(result, msg.GetMessages()...)

	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("`storage-health validate scenario %s --json` - re-check storage findings", name),
		},
	})
}

// splitCSV flattens repeated and comma-separated --rule flag values into a
// deduplicated finding-code list.
func splitCSV(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}
