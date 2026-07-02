// Package wizard is the CLI's contract-authoring surface: start / answer /
// preview / apply over the WizardService session RPCs, plus an interactive
// TTY interview (`wizard start --interactive`) that loops the same RPCs.
package wizard

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	wizardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/wizard"
	wizardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/wizard/wizard_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "wizard"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"WizardService.StartSession":    h.start,
		"WizardService.SubmitAnswers":   h.answer,
		"WizardService.PreviewScaffold": h.preview,
		"WizardService.ApplyScaffold":   h.apply,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("wizard: load from manifest: %w", err)
	}
	return group, nil
}

type handlers struct {
	core   *cliapp.ScenarioApp
	client wizardconnect.WizardServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: wizardconnect.NewWizardServiceClient(httpClient, baseURL)}
}

func (h *handlers) startSession(scenario string, reset bool) (*wizardv1.SessionState, error) {
	resp, err := h.client.StartSession(context.Background(), connect.NewRequest(&wizardv1.StartSessionRequest{
		Scenario: scenario,
		Reset_:   reset,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	state, err := h.startSession(ctx.Positional("scenario"), ctx.BoolFlag("reset"))
	if err != nil {
		return cliapp.WrapAPIError("start wizard", err, nil)
	}
	if ctx.BoolFlag("interactive") {
		return h.interactive(ctx, state)
	}
	return renderState(ctx, state)
}

// interactive loops the remaining questions on the TTY, submitting each
// answer through the same SubmitAnswers RPC the UI uses.
func (h *handlers) interactive(ctx cliapp.RunContext, state *wizardv1.SessionState) error {
	reader := bufio.NewReader(os.Stdin)
	questions := map[string]*wizardv1.Question{}
	for _, q := range state.Questions {
		questions[q.Id] = q
	}
	for len(state.Remaining) > 0 {
		qid := state.Remaining[0]
		q := questions[qid]
		if q == nil {
			return fmt.Errorf("server referenced unknown question %q", qid)
		}
		fmt.Printf("\n%s\n", q.Prompt)
		if q.Help != "" {
			fmt.Printf("  (%s)\n", q.Help)
		}
		answer := &wizardv1.Answer{QuestionId: q.Id}
		if q.Kind == "ot_list" {
			fmt.Println("  One target per line as `Title :: description`; empty line to finish.")
			for {
				fmt.Print("> ")
				line, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				line = strings.TrimSpace(line)
				if line == "" {
					break
				}
				parts := strings.SplitN(line, "::", 2)
				t := &wizardv1.OperationalTargetAnswer{Title: strings.TrimSpace(parts[0])}
				if len(parts) == 2 {
					t.Description = strings.TrimSpace(parts[1])
				}
				answer.Targets = append(answer.Targets, t)
			}
		} else {
			fmt.Println("  Multi-line answer; finish with a single `.` on its own line.")
			var lines []string
			for {
				fmt.Print("> ")
				line, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				if strings.TrimSpace(line) == "." {
					break
				}
				lines = append(lines, strings.TrimRight(line, "\n"))
			}
			answer.Text = strings.Join(lines, "\n")
		}
		resp, err := h.client.SubmitAnswers(context.Background(), connect.NewRequest(&wizardv1.SubmitAnswersRequest{
			SessionId: state.SessionId,
			Answers:   []*wizardv1.Answer{answer},
		}))
		if err != nil {
			return cliapp.WrapAPIError("submit answer", err, nil)
		}
		state = resp.Msg
		if a, ok := state.Answers[qid]; ok && a.InvalidReason != "" {
			fmt.Printf("  ✗ %s — try again.\n", a.InvalidReason)
		}
	}
	fmt.Println("\nAll required questions answered.")
	fmt.Printf("Next: `business-health wizard preview %s` (diffs), then `business-health wizard apply %s`.\n", state.Scenario, state.Scenario)
	return nil
}

func (h *handlers) answer(ctx cliapp.RunContext) error {
	state, err := h.startSession(ctx.Positional("scenario"), false)
	if err != nil {
		return cliapp.WrapAPIError("resume wizard", err, nil)
	}
	var answers []*wizardv1.Answer
	if file := ctx.Flag("answers"); file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read answers file: %w", err)
		}
		var raw []struct {
			QuestionID string `json:"question_id"`
			Text       string `json:"text"`
			Targets    []struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"targets"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parse answers file: %w", err)
		}
		for _, r := range raw {
			a := &wizardv1.Answer{QuestionId: r.QuestionID, Text: r.Text}
			for _, t := range r.Targets {
				a.Targets = append(a.Targets, &wizardv1.OperationalTargetAnswer{Title: t.Title, Description: t.Description})
			}
			answers = append(answers, a)
		}
	} else if q := ctx.Flag("question"); q != "" {
		answers = append(answers, &wizardv1.Answer{QuestionId: q, Text: ctx.Flag("text")})
	} else {
		return fmt.Errorf("provide --answers file.json or --question/--text")
	}
	resp, err := h.client.SubmitAnswers(context.Background(), connect.NewRequest(&wizardv1.SubmitAnswersRequest{
		SessionId: state.SessionId,
		Answers:   answers,
	}))
	if err != nil {
		return cliapp.WrapAPIError("submit answers", err, nil)
	}
	return renderState(ctx, resp.Msg)
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	state, err := h.startSession(ctx.Positional("scenario"), false)
	if err != nil {
		return cliapp.WrapAPIError("resume wizard", err, nil)
	}
	resp, err := h.client.PreviewScaffold(context.Background(), connect.NewRequest(&wizardv1.PreviewScaffoldRequest{
		SessionId: state.SessionId,
	}))
	if err != nil {
		return cliapp.WrapAPIError("preview scaffold", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Files))
	for _, f := range resp.Msg.Files {
		verb := "update"
		if f.Before == "" {
			verb = "create"
		}
		results = append(results, fmt.Sprintf("%s %s (%d bytes)", verb, f.Path, len(f.After)))
	}
	summary := []string{fmt.Sprintf("%d file(s) would be written.", len(resp.Msg.Files))}
	if len(resp.Msg.Blocking) > 0 {
		summary = append(summary, "Blocked on: "+strings.Join(resp.Msg.Blocking, ", "))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Scaffold (dry-run)",
		Results:        results,
		RetrievalHints: []string{"`wizard apply <scenario>` — write these files (refuses while blocked)"},
	})
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	state, err := h.startSession(ctx.Positional("scenario"), false)
	if err != nil {
		return cliapp.WrapAPIError("resume wizard", err, nil)
	}
	resp, err := h.client.ApplyScaffold(context.Background(), connect.NewRequest(&wizardv1.ApplyScaffoldRequest{
		SessionId: state.SessionId,
		Apply:     true,
	}))
	if err != nil {
		return cliapp.WrapAPIError("apply scaffold", err, nil)
	}
	results := append([]string{}, resp.Msg.Written...)
	if len(resp.Msg.ResidualFindings) > 0 {
		results = append(results, "Residual findings (expected none):")
		results = append(results, resp.Msg.ResidualFindings...)
	} else {
		results = append(results, "Post-apply validation: clean (the round-trip guarantee held).")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Wrote %d file(s).", len(resp.Msg.Written))},
		Changes: results,
		NextCommand: []string{
			"`business-health validate scenario <scenario>` — full assessment",
			"`business-health matrix show <scenario>` — traceability",
		},
	})
}

func renderState(ctx cliapp.RunContext, state *wizardv1.SessionState) error {
	results := make([]string, 0, len(state.Remaining))
	questions := map[string]*wizardv1.Question{}
	for _, q := range state.Questions {
		questions[q.Id] = q
	}
	for _, id := range state.Remaining {
		prompt := id
		if q := questions[id]; q != nil {
			prompt = fmt.Sprintf("%s — %s", id, q.Prompt)
		}
		if a, ok := state.Answers[id]; ok && a.InvalidReason != "" {
			prompt += " (invalid: " + a.InvalidReason + ")"
		}
		results = append(results, prompt)
	}
	if len(results) == 0 {
		results = append(results, "All required questions answered — preview and apply when ready.")
	}
	summary := []string{fmt.Sprintf("Session %s for %s: %d answered, %d remaining.", state.SessionId, state.Scenario, len(state.Answers), len(state.Remaining))}
	for _, hint := range state.Hints {
		summary = append(summary, fmt.Sprintf("Similar capability: %s (%s, score %.2f)", hint.Capability, hint.Anchor, hint.Score))
	}
	return cliapp.RenderProtoList(ctx, state, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Remaining questions",
		Results:        results,
		RetrievalHints: []string{
			"`wizard answer <scenario> --answers file.json` — bulk answers",
			"`wizard start <scenario> --interactive` — TTY interview",
		},
	})
}
