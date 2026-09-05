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

func (h *handlers) archiveRetention(ctx cliapp.RunContext) error {
	resp, err := h.client.GetArchiveRetention(context.Background(), connect.NewRequest(&sessionsv1.GetArchiveRetentionRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get archive retention", err, nil)
	}
	policy, stats := resp.Msg.GetPolicy(), resp.Msg.GetStats()
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Archived entries: %d; total measured bytes: %d", stats.GetEntryCount(), stats.GetTotalBytes())},
		ResultsHeading: "Archive retention",
		Results: []string{
			fmt.Sprintf("message_less_age_days: %d", policy.GetMessageLessAgeDays()),
			fmt.Sprintf("agent_home_age_days: %d", policy.GetAgentHomeAgeDays()),
			fmt.Sprintf("max_bytes: %d", policy.GetMaxBytes()),
			fmt.Sprintf("message_count: %d", stats.GetMessageCount()),
			fmt.Sprintf("transcript_bytes: %d", stats.GetTranscriptBytes()),
			fmt.Sprintf("agent_home_bytes: %d", stats.GetAgentHomeBytes()),
		},
		RetrievalHints: []string{fmt.Sprintf("%s session archive-prune", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) archivePrune(ctx cliapp.RunContext) error {
	apply := strings.EqualFold(ctx.Flag("apply"), "true")
	resp, err := h.client.PruneArchive(context.Background(), connect.NewRequest(&sessionsv1.PruneArchiveRequest{Apply: apply}))
	if err != nil {
		return cliapp.WrapAPIError("prune archive", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetActions())+1)
	for _, action := range resp.Msg.GetActions() {
		results = append(results, fmt.Sprintf("%s: %s (%d bytes, applied=%t)", action.GetSessionId(), action.GetKind(), action.GetBytes(), action.GetApplied()))
	}
	if len(results) == 0 {
		results = append(results, "No prune actions matched the configured policy")
	}
	report := cliapp.MutationReport{
		Result: results,
		NextCommand: []string{
			fmt.Sprintf("dry_run: %t", resp.Msg.GetDryRun()),
			fmt.Sprintf("reclaimed_bytes: %d", resp.Msg.GetReclaimedBytes()),
		},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
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
		fmt.Sprintf("Survives restart: %t", sess.GetSurvivesRestart()),
	}
	if t := sess.GetCreatedAt(); t != "" {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTime(t)))
	}
	results = append(results, fmt.Sprintf("Origin: %s", originString(sess.GetOrigin())))
	if owner := sess.GetOwner(); owner != "" {
		results = append(results, fmt.Sprintf("Owner: %s", owner))
	}
	if label := sess.GetDisplayLabel(); label != "" {
		results = append(results, fmt.Sprintf("Label: %s", label))
	}
	if target := sess.GetTarget(); target != nil {
		results = append(results, fmt.Sprintf("Target: %s (%s)", target.GetLabel(), target.GetId()))
		results = append(results, fmt.Sprintf("Target state: %s; dispatchable=%t", target.GetState().String(), target.GetDispatchable()))
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
	Shell                string         `json:"shell,omitempty"`
	Cols                 int32          `json:"cols,omitempty"`
	Rows                 int32          `json:"rows,omitempty"`
	Backend              string         `json:"backend,omitempty"`
	Policy               *policyBody    `json:"policy,omitempty"`
	LaunchCommand        string         `json:"launchCommand,omitempty"`
	AgentType            string         `json:"agentType,omitempty"`
	Origin               string         `json:"origin,omitempty"`
	Owner                string         `json:"owner,omitempty"`
	DisplayLabel         string         `json:"displayLabel,omitempty"`
	TargetID             string         `json:"targetId,omitempty"`
	WorkingDir           string         `json:"workingDir,omitempty"`
	ExecuteLaunchCommand bool           `json:"executeLaunchCommand,omitempty"`
	Extra                map[string]any `json:"-"`
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
		Shell:                strings.TrimSpace(ctx.Flag("shell")),
		Cols:                 int32(cols),
		Rows:                 int32(rows),
		Backend:              strings.TrimSpace(ctx.Flag("backend")),
		LaunchCommand:        strings.TrimSpace(ctx.Flag("launch-command")),
		Origin:               strings.TrimSpace(ctx.Flag("origin")),
		Owner:                strings.TrimSpace(ctx.Flag("owner")),
		DisplayLabel:         strings.TrimSpace(ctx.Flag("label")),
		TargetID:             strings.TrimSpace(ctx.Flag("target")),
		WorkingDir:           strings.TrimSpace(ctx.Flag("working-dir")),
		ExecuteLaunchCommand: ctx.BoolFlag("execute-launch-command"),
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

	// Resolve origin after the --body-file merge so either source can set it.
	// Omitted origin stays UNSPECIFIED; the server normalizes it to PROGRAMMATIC.
	origin, err := parseOrigin(body.Origin)
	if err != nil {
		return err
	}

	req := connect.NewRequest(&sessionsv1.CreateRequest{
		Shell:                body.Shell,
		Cols:                 body.Cols,
		Rows:                 body.Rows,
		Backend:              body.Backend,
		LaunchCommand:        body.LaunchCommand,
		AgentType:            body.AgentType,
		Origin:               origin,
		Owner:                body.Owner,
		DisplayLabel:         body.DisplayLabel,
		TargetId:             body.TargetID,
		WorkingDir:           body.WorkingDir,
		ExecuteLaunchCommand: body.ExecuteLaunchCommand,
	})
	if body.Policy != nil {
		req.Msg.Policy = &sessionsv1.ExpirationPolicy{Mode: body.Policy.Mode, Duration: body.Policy.Duration}
		req.Msg.HasPolicy = true
	}
	if key := strings.TrimSpace(ctx.Flag("idempotency-key")); key != "" {
		req.Header().Set("X-Idempotency-Key", key)
	}

	resp, err := h.client.Create(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError("session create", err, nil)
	}
	sess := resp.Msg.GetSession()

	changes := []string{
		fmt.Sprintf("Backend: %s", sess.GetBackend()),
		fmt.Sprintf("Shell: %s", sess.GetShell()),
		fmt.Sprintf("Origin: %s", originString(sess.GetOrigin())),
	}
	if owner := sess.GetOwner(); owner != "" {
		changes = append(changes, fmt.Sprintf("Owner: %s", owner))
	}
	if label := sess.GetDisplayLabel(); label != "" {
		changes = append(changes, fmt.Sprintf("Label: %s", label))
	}
	if target := sess.GetTarget(); target != nil {
		changes = append(changes, fmt.Sprintf("Target: %s (%s)", target.GetLabel(), target.GetId()))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created session %s", sess.GetId())},
		Changes:     changes,
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
		row := fmt.Sprintf("%s | shell=%s | backend=%s | %dx%d | origin=%s",
			support.ShortID(s.GetId()), s.GetShell(), s.GetBackend(), s.GetCols(), s.GetRows(), originString(s.GetOrigin()))
		if owner := s.GetOwner(); owner != "" {
			row += " | owner=" + owner
		}
		if target := s.GetTarget(); target != nil {
			row += " | target=" + target.GetLabel() + " (" + target.GetId() + ")"
		}
		rows = append(rows, row)
	}
	return rows
}

// parseOrigin maps the human-facing --origin value onto the proto enum. An
// omitted value stays UNSPECIFIED (the server normalizes it to PROGRAMMATIC);
// an unrecognized value is a client-side error.
func parseOrigin(raw string) (sessionsv1.SessionOrigin, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_UNSPECIFIED, nil
	case "ui":
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_UI, nil
	case "programmatic":
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC, nil
	case "remote":
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE, nil
	default:
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_UNSPECIFIED,
			fmt.Errorf("invalid --origin %q (want ui|programmatic|remote)", raw)
	}
}

// originString renders the proto enum as its human-facing token.
func originString(o sessionsv1.SessionOrigin) string {
	switch o {
	case sessionsv1.SessionOrigin_SESSION_ORIGIN_UI:
		return "ui"
	case sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC:
		return "programmatic"
	case sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE:
		return "remote"
	default:
		return "unspecified"
	}
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
