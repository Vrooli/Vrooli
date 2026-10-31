package snippets

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"

	snippetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/snippets"
	snippetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/snippets/snippets_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

var namedVariablePattern = regexp.MustCompile(`\{\{([a-z][a-z0-9_]*)\}\}`)

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	return &handlers{core: core}
}

// client is intentionally resolved at invocation time. The CLI constructs its
// command tree before parsing global flags, so capturing the generated client
// in Register would bake a stale saved api_base into its absolute URL and make
// a later --api-base override ineffective.
func (h *handlers) client() snippetsconnect.SnippetsServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	return snippetsconnect.NewSnippetsServiceClient(httpClient, baseURL)
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client().ListSnippets(context.Background(), connect.NewRequest(&snippetsv1.ListSnippetsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("snippet list", err, nil)
	}

	snippets := resp.Msg.GetSnippets()
	rows := make([]string, 0, len(snippets)*2)
	for _, snippet := range snippets {
		rows = append(rows, fmt.Sprintf("  %s | %s | used=%d | variables=%d",
			support.ShortID(snippet.GetId()), snippet.GetName(), snippet.GetUseCount(), distinctVariableCount(snippet.GetBody())))
		rows = append(rows, "      "+bodyPreview(snippet.GetBody(), 60))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Snippets: %d", len(snippets))},
		ResultsHeading: "Snippets",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s snippet upsert --body-file snippet.json", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

type upsertBody struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	Color     string `json:"color"`
	Pinned    *bool  `json:"pinned,omitempty"`
	SortOrder int32  `json:"sort_order"`
}

func (h *handlers) upsert(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body upsertBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	req := &snippetsv1.UpsertSnippetRequest{
		Id:        body.ID,
		Name:      body.Name,
		Body:      body.Body,
		Color:     body.Color,
		SortOrder: body.SortOrder,
	}
	if body.Pinned != nil {
		req.Pinned = *body.Pinned
		req.HasPinned = true
	}
	if ctx.FlagProvided("pinned") {
		req.Pinned = ctx.BoolFlag("pinned")
		req.HasPinned = true
	}

	resp, err := h.client().UpsertSnippet(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("snippet upsert", err, nil)
	}
	snippet := resp.Msg.GetSnippet()
	report := cliapp.MutationReport{
		Result: []string{"Saved snippet"},
		Changes: []string{
			fmt.Sprintf("ID: %s", snippet.GetId()),
			fmt.Sprintf("Name: %s", snippet.GetName()),
			fmt.Sprintf("Variables: %d", distinctVariableCount(snippet.GetBody())),
			fmt.Sprintf("Pinned: %t", snippet.GetPinned()),
		},
		NextCommand: []string{fmt.Sprintf("%s snippet list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("snippet-id")
	if id == "" {
		return fmt.Errorf("usage: snippet delete <snippet-id>")
	}
	resp, err := h.client().DeleteSnippet(context.Background(), connect.NewRequest(&snippetsv1.DeleteSnippetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("snippet delete", err, nil)
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Snippet %s deleted=%t", id, resp.Msg.GetDeleted())},
		NextCommand: []string{fmt.Sprintf("%s snippet list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) touch(ctx cliapp.RunContext) error {
	id := ctx.Positional("snippet-id")
	if id == "" {
		return fmt.Errorf("usage: snippet touch <snippet-id>")
	}
	resp, err := h.client().TouchSnippet(context.Background(), connect.NewRequest(&snippetsv1.TouchSnippetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("snippet touch", err, nil)
	}
	snippet := resp.Msg.GetSnippet()
	report := cliapp.MutationReport{Result: []string{fmt.Sprintf("Touched snippet %s; used=%d", id, snippet.GetUseCount())}}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func distinctVariableCount(body string) int {
	variables := make(map[string]struct{})
	for _, match := range namedVariablePattern.FindAllStringSubmatch(body, -1) {
		variables[match[1]] = struct{}{}
	}
	return len(variables)
}

func bodyPreview(body string, limit int) string {
	compact := strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(compact) <= limit {
		return compact
	}
	runes := []rune(compact)
	return string(runes[:limit]) + "…"
}
