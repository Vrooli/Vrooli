package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client sessionsconnect.SessionsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: sessionsconnect.NewSessionsServiceClient(httpClient, baseURL),
	}
}

// -----------------------------------------------------------------------------
// list / get / create / delete
// -----------------------------------------------------------------------------

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.List(context.Background(), connect.NewRequest(&sessionsv1.ListRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("session list", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active sessions: %d", len(resp.Msg.GetSessions()))},
		ResultsHeading: "Sessions",
		Results:        sessionRows(resp.Msg.GetSessions()),
		RetrievalHints: []string{
			fmt.Sprintf("%s session get <session-id>", support.CLIName),
			fmt.Sprintf("%s session policy-get <session-id>", support.CLIName),
		},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("session-id")
	if id == "" {
		return fmt.Errorf("usage: session get <session-id>")
	}

	resp, err := h.client.Get(context.Background(), connect.NewRequest(&sessionsv1.GetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("session get", err, nil)
	}
	sess := resp.Msg.GetSession()

	results := []string{
		fmt.Sprintf("ID: %s", sess.GetId()),
		fmt.Sprintf("Shell: %s", sess.GetShell()),
		fmt.Sprintf("Backend: %s", sess.GetBackend()),
		fmt.Sprintf("Size: %dx%d", sess.GetCols(), sess.GetRows()),
		fmt.Sprintf("Busy: %t", sess.GetBusy()),
		fmt.Sprintf("Survives restart: %t", sess.GetSurvivesRestart()),
	}
	if t := sess.GetCreatedAt(); t != "" {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTime(t)))
	}
	results = append(results, fmt.Sprintf("Policy: %s", policyString(sess.GetPolicy())))

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Session: %s (%s)", sess.GetId(), sess.GetBackend())},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s session policy-get %s", support.CLIName, sess.GetId()),
			fmt.Sprintf("%s conversation get --session %s", support.CLIName, sess.GetId()),
		},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

// createBody mirrors CreateRequest's JSON form so --body-file is supported
// uniformly. Pointer fields toggle the has_policy flag server-side.
type createBody struct {
	Shell         string         `json:"shell,omitempty"`
	Cols          int32          `json:"cols,omitempty"`
	Rows          int32          `json:"rows,omitempty"`
	Backend       string         `json:"backend,omitempty"`
	Policy        *policyBody    `json:"policy,omitempty"`
	LaunchCommand string         `json:"launchCommand,omitempty"`
	AgentType     string         `json:"agentType,omitempty"`
	Extra         map[string]any `json:"-"`
}

type policyBody struct {
	Mode     string `json:"mode"`
	Duration string `json:"duration,omitempty"`
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	cols, err := atoiFlag(ctx.Flag("cols"))
	if err != nil {
		return err
	}
	rows, err := atoiFlag(ctx.Flag("rows"))
	if err != nil {
		return err
	}

	body := createBody{
		Shell:   strings.TrimSpace(ctx.Flag("shell")),
		Cols:    int32(cols),
		Rows:    int32(rows),
		Backend: strings.TrimSpace(ctx.Flag("backend")),
	}
	if bodyFile := strings.TrimSpace(ctx.Flag("body-file")); bodyFile != "" {
		raw, err := support.ReadJSONFile(bodyFile, true)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return fmt.Errorf("decode --body-file: %w", err)
		}
	}

	req := connect.NewRequest(&sessionsv1.CreateRequest{
		Shell:         body.Shell,
		Cols:          body.Cols,
		Rows:          body.Rows,
		Backend:       body.Backend,
		LaunchCommand: body.LaunchCommand,
		AgentType:     body.AgentType,
	})
	if body.Policy != nil {
		req.Msg.Policy = &sessionsv1.ExpirationPolicy{Mode: body.Policy.Mode, Duration: body.Policy.Duration}
		req.Msg.HasPolicy = true
	}

	resp, err := h.client.Create(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError("session create", err, nil)
	}
	sess := resp.Msg.GetSession()

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Created session %s", sess.GetId())},
		Changes: []string{
			fmt.Sprintf("Backend: %s", sess.GetBackend()),
			fmt.Sprintf("Shell: %s", sess.GetShell()),
		},
		NextCommand: []string{fmt.Sprintf("%s session get %s", support.CLIName, sess.GetId())},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("session-id")
	if id == "" {
		return fmt.Errorf("usage: session delete <session-id>")
	}

	if _, err := h.client.Delete(context.Background(),
		connect.NewRequest(&sessionsv1.DeleteRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("session delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted session %s", id)},
		NextCommand: []string{fmt.Sprintf("%s session list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

// -----------------------------------------------------------------------------
// policy
// -----------------------------------------------------------------------------

func (h *handlers) policyGet(ctx cliapp.RunContext) error {
	id := ctx.Positional("session-id")
	if id == "" {
		return fmt.Errorf("usage: session policy-get <session-id>")
	}

	resp, err := h.client.GetPolicy(context.Background(), connect.NewRequest(&sessionsv1.GetPolicyRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("session policy-get", err, nil)
	}
	view := resp.Msg.GetPolicy()

	results := []string{
		fmt.Sprintf("Session: %s", view.GetSessionId()),
		fmt.Sprintf("Policy: %s", policyString(view.GetPolicy())),
	}
	if view.GetHasExpiry() {
		results = append(results,
			fmt.Sprintf("Expires at: %s", support.FormatTime(view.GetExpiresAt())),
			fmt.Sprintf("TTL: %.0fs", view.GetTtlSeconds()),
		)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Policy for session %s", view.GetSessionId())},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s session policy-set %s --mode <mode>", support.CLIName, view.GetSessionId())},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) policySet(ctx cliapp.RunContext) error {
	id := ctx.Positional("session-id")
	if id == "" {
		return fmt.Errorf("usage: session policy-set <session-id> --mode <mode> [--duration <dur>]")
	}

	var policy policyBody
	if bodyFile := strings.TrimSpace(ctx.Flag("body-file")); bodyFile != "" {
		raw, err := support.ReadJSONFile(bodyFile, true)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &policy); err != nil {
			return fmt.Errorf("decode --body-file: %w", err)
		}
	} else {
		mode := ctx.Flag("mode")
		if strings.TrimSpace(mode) == "" {
			return fmt.Errorf("--mode is required (or provide --body-file)")
		}
		policy = policyBody{Mode: mode, Duration: ctx.Flag("duration")}
	}

	resp, err := h.client.UpdatePolicy(context.Background(), connect.NewRequest(&sessionsv1.UpdatePolicyRequest{
		Id: id,
		Policy: &sessionsv1.ExpirationPolicy{
			Mode:     policy.Mode,
			Duration: policy.Duration,
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("session policy-set", err, nil)
	}
	view := resp.Msg.GetPolicy()

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated policy for session %s", view.GetSessionId())},
		Changes:     []string{fmt.Sprintf("Policy: %s", policyString(view.GetPolicy()))},
		NextCommand: []string{fmt.Sprintf("%s session policy-get %s", support.CLIName, view.GetSessionId())},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

// -----------------------------------------------------------------------------
// recovery
// -----------------------------------------------------------------------------

func (h *handlers) listRecoverable(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRecoverable(context.Background(),
		connect.NewRequest(&sessionsv1.ListRecoverableRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("session list-recoverable", err, nil)
	}
	rows := resp.Msg.GetSessions()

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recoverable sessions: %d", len(rows))},
		ResultsHeading: "Recoverable",
		Results:        recoverableRows(rows),
		RetrievalHints: []string{
			fmt.Sprintf("%s session recover <session-id>", support.CLIName),
			fmt.Sprintf("%s session dismiss <session-id>", support.CLIName),
		},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) recover(ctx cliapp.RunContext) error {
	id := ctx.Positional("session-id")
	if id == "" {
		return fmt.Errorf("usage: session recover <session-id>")
	}

	resp, err := h.client.Recover(context.Background(), connect.NewRequest(&sessionsv1.RecoverRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("session recover", err, nil)
	}
	res := resp.Msg

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Recovered %s -> %s", support.ShortID(res.GetOldSessionId()), support.ShortID(res.GetNewSessionId()))},
		Changes: []string{
			fmt.Sprintf("Agent: %s", res.GetAgentType()),
			fmt.Sprintf("Pasted: %s", strings.TrimSpace(res.GetCommandSent())),
			fmt.Sprintf("CODEX_HOME copied: %t", res.GetCodexHomeCopied()),
		},
		NextCommand: []string{fmt.Sprintf("%s session get %s", support.CLIName, res.GetNewSessionId())},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) dismiss(ctx cliapp.RunContext) error {
	id := ctx.Positional("session-id")
	if id == "" {
		return fmt.Errorf("usage: session dismiss <session-id>")
	}
	if _, err := h.client.DismissRecoverable(context.Background(),
		connect.NewRequest(&sessionsv1.DismissRecoverableRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("session dismiss", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Dismissed orphan session %s (on-disk state preserved)", id)},
		NextCommand: []string{fmt.Sprintf("%s session list-recoverable", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

// -----------------------------------------------------------------------------
// formatting helpers
// -----------------------------------------------------------------------------

// atoiFlag mirrors the old fs.Int(...,0,...) default: an unset (empty) flag
// parses as 0; a non-empty value must be a valid integer.
func atoiFlag(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func sessionRows(sessions []*sessionsv1.Session) []string {
	if len(sessions) == 0 {
		return []string{"No active sessions"}
	}
	rows := make([]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, fmt.Sprintf("%s | shell=%s | backend=%s | %dx%d | busy=%t",
			support.ShortID(s.GetId()), s.GetShell(), s.GetBackend(), s.GetCols(), s.GetRows(), s.GetBusy()))
	}
	return rows
}

func recoverableRows(rows []*sessionsv1.RecoverableSession) []string {
	if len(rows) == 0 {
		return []string{"No orphaned persistent sessions awaiting recovery"}
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		flag := ""
		if !r.GetRecoverable() {
			flag = " (NOT RECOVERABLE: " + r.GetNotRecoverableReason() + ")"
		}
		out = append(out, fmt.Sprintf("%s | agent=%s | session=%s | orphaned=%s%s",
			support.ShortID(r.GetId()),
			defaultIfEmpty(r.GetAgentType(), "none"),
			defaultIfEmpty(support.ShortID(r.GetAgentSessionId()), "-"),
			support.FormatTime(r.GetOrphanedAt()),
			flag,
		))
	}
	return out
}

func policyString(p *sessionsv1.ExpirationPolicy) string {
	if p == nil || p.GetMode() == "" {
		return "never"
	}
	if p.GetDuration() != "" {
		return p.GetMode() + "/" + p.GetDuration()
	}
	return p.GetMode()
}

func defaultIfEmpty(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

// Build-time guard: ensures time import isn't accidentally removed when
// the file is regenerated; FormatTime accepts strings, not time.Time, so
// nothing imports it directly today.
var _ = time.RFC3339
