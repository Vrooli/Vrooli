package versions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	versionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core            *cliapp.ScenarioApp
	client          versionsconnect.VersionsServiceClient
	lifecycleClient versionsconnect.VersionLifecycleServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:            core,
		client:          versionsconnect.NewVersionsServiceClient(httpClient, baseURL),
		lifecycleClient: versionsconnect.NewVersionLifecycleServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) retireCandidates(ctx cliapp.RunContext) error {
	resp, err := h.lifecycleClient.ListRetireCandidates(context.Background(), connect.NewRequest(&versionsv1.ListRetireCandidatesRequest{ComponentId: ctx.Flag("component-id")}))
	if err != nil {
		return cliapp.WrapAPIError("list retire candidates", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Candidates))
	for _, item := range resp.Msg.Candidates {
		results = append(results, fmt.Sprintf("%s %s@%s status=%s", item.ComponentId, item.LibraryId, item.Version, item.Status))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d retire candidate(s).", len(results))}, ResultsHeading: "Retire candidates", Results: results})
}

func (h *handlers) progression(ctx cliapp.RunContext) error {
	resp, err := h.lifecycleClient.ListVersionLedger(context.Background(), connect.NewRequest(&versionsv1.ListVersionLedgerRequest{LibraryId: ctx.Flag("library-id"), Window: ctx.Flag("window")}))
	if err != nil {
		return cliapp.WrapAPIError("list version progression", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Rows))
	for _, row := range resp.Msg.Rows {
		results = append(results, fmt.Sprintf("%s@%s state=%s gates=%d/%d tests=%d pass=%.2f adopters=%d size=%d LOC", row.LibraryId, row.Version, row.LifecycleState, row.GatePassCount, row.GateFailCount, row.TestRuns, row.TestPassRate, row.AdoptionCurrent, row.LinesOfCode))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d ledger row(s).", len(results))}, ResultsHeading: "Version progression", Results: results})
}

func (h *handlers) transition(ctx cliapp.RunContext, method string) error {
	confirm := false
	if method == "retire" {
		confirm = ctx.Flag("confirm") != ""
	}
	req := &versionsv1.VersionLifecycleRequest{ComponentId: ctx.Positional("component-id"), Version: ctx.Positional("version"), Confirm: confirm}
	var resp *connect.Response[versionsv1.VersionLifecycleResponse]
	var err error
	switch method {
	case "deprecate":
		resp, err = h.lifecycleClient.DeprecateVersion(context.Background(), connect.NewRequest(req))
	case "archive":
		resp, err = h.lifecycleClient.ArchiveVersion(context.Background(), connect.NewRequest(req))
	default:
		resp, err = h.lifecycleClient.RetireVersion(context.Background(), connect.NewRequest(req))
	}
	if err != nil {
		return cliapp.WrapAPIError(method+" version", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%s %s@%s.", method, req.ComponentId, req.Version)}, ResultsHeading: "Version lifecycle", Results: []string{fmt.Sprintf("%s state=%s", resp.Msg.Version.GetVersion(), resp.Msg.LifecycleState)}})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	req := &versionsv1.ListVersionsRequest{ComponentId: ctx.Positional("component-id")}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListVersions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list versions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no list response")
	}
	results := make([]string, 0, len(resp.Msg.Versions))
	for _, v := range resp.Msg.Versions {
		results = append(results, formatVersion(v))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d version(s).", len(resp.Msg.Versions))},
		ResultsHeading: "Versions",
		Results:        results,
		RetrievalHints: []string{
			"`versions show <component-id> <version> --with-content` — full body",
			"`versions diff <component-id> <from> <to>` — line-by-line diff",
		},
	})
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	req := &versionsv1.GetVersionRequest{
		ComponentId:    ctx.Positional("component-id"),
		Version:        ctx.Positional("version"),
		IncludeContent: ctx.Flag("with-content") != "",
	}
	resp, err := h.client.GetVersion(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("get version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Version == nil {
		return fmt.Errorf("server returned no version")
	}
	results := []string{formatVersion(resp.Msg.Version)}
	if req.IncludeContent {
		results = append(results, "--- content ---", resp.Msg.Content)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Version %s for %s.", req.Version, req.ComponentId)},
		ResultsHeading: "Version",
		Results:        results,
	})
}

func (h *handlers) diff(ctx cliapp.RunContext) error {
	req := &versionsv1.DiffVersionsRequest{
		ComponentId: ctx.Positional("component-id"),
		From:        ctx.Positional("from"),
		To:          ctx.Positional("to"),
	}
	resp, err := h.client.DiffVersions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("diff versions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no diff response")
	}
	results := make([]string, 0, len(resp.Msg.Rows))
	for _, r := range resp.Msg.Rows {
		results = append(results, formatDiffRow(r))
	}
	summary := fmt.Sprintf("%s → %s : +%d / -%d (%d rows)",
		req.From, req.To, resp.Msg.Additions, resp.Msg.Removals, len(resp.Msg.Rows))
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Diff",
		Results:        results,
	})
}

func formatVersion(v *versionsv1.Version) string {
	if v == nil {
		return "(nil)"
	}
	created := "?"
	if v.CreatedAt != nil {
		created = v.CreatedAt.AsTime().Format(time.RFC3339)
	}
	sha := v.ContentSha256
	if len(sha) > 12 {
		sha = sha[:12]
	}
	return fmt.Sprintf("%s — v=%s sha=%s first-seen=%s required-tokens=%v required-token-patterns=%v changelog=%q",
		v.Id, v.Version, sha, created, v.RequiredTokens, v.RequiredTokenPatterns, v.ChangelogMd)
}

func formatDiffRow(r *versionsv1.DiffRow) string {
	if r == nil {
		return ""
	}
	left := formatDiffCell(r.Left)
	right := formatDiffCell(r.Right)
	sep := "  |  "
	return strings.Join([]string{left, sep, right}, "")
}

func formatDiffCell(c *versionsv1.DiffCell) string {
	if c == nil {
		return ""
	}
	marker := opMarker(c.Op)
	if c.Op == versionsv1.DiffOp_DIFF_OP_EMPTY {
		return fmt.Sprintf("%s %4s %s", marker, "", "")
	}
	return fmt.Sprintf("%s %4d %s", marker, c.LineNumber, c.Text)
}

func opMarker(op versionsv1.DiffOp) string {
	switch op {
	case versionsv1.DiffOp_DIFF_OP_ADD:
		return "+"
	case versionsv1.DiffOp_DIFF_OP_REMOVE:
		return "-"
	case versionsv1.DiffOp_DIFF_OP_EQUAL:
		return " "
	case versionsv1.DiffOp_DIFF_OP_EMPTY:
		return " "
	}
	return "?"
}
