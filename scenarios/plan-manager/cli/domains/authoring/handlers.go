package authoring

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	authoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring"
	authoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring/authoring_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client authoringconnect.AuthoringServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: authoringconnect.NewAuthoringServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	resp, err := h.client.StartSession(context.Background(), connect.NewRequest(&authoringv1.StartSessionRequest{
		Title:      ctx.Flag("title"),
		Slug:       ctx.Flag("slug"),
		TemplateId: ctx.Flag("template"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("start authoring session", err, nil)
	}
	sess := resp.Msg.GetSession()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Started session %s for %q.", sess.GetId(), sess.GetTitle())},
		Changes: []string{fmt.Sprintf("Seeded %d section(s); next: %s.", len(sess.GetSections()), nextLabel(sess.GetCurrentSectionKey()))},
		NextCommand: []string{
			fmt.Sprintf("`author next %s` — the next section that needs input", sess.GetId()),
		},
	})
}

func (h *handlers) sectionGet(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSection(context.Background(), connect.NewRequest(&authoringv1.GetSectionRequest{
		SessionId:  ctx.Positional("session"),
		SectionKey: ctx.Flag("section"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get section", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{formatSection(resp.Msg.GetSection())},
		ResultsHeading: "Content",
		Results:        []string{resp.Msg.GetSection().GetContent()},
	})
}

func (h *handlers) sectionSubmit(ctx cliapp.RunContext) error {
	resp, err := h.client.SubmitSection(context.Background(), connect.NewRequest(&authoringv1.SubmitSectionRequest{
		SessionId:  ctx.Positional("session"),
		SectionKey: ctx.Flag("section"),
		Content:    ctx.Flag("content"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("submit section", err, nil)
	}
	changes := []string{fmt.Sprintf("Next: %s.", nextLabel(resp.Msg.GetSession().GetCurrentSectionKey()))}
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Submitted section %q (%d violation(s)).", ctx.Flag("section"), len(resp.Msg.GetViolations()))},
		Changes: changes,
	})
}

func (h *handlers) next(ctx cliapp.RunContext) error {
	resp, err := h.client.Next(context.Background(), connect.NewRequest(&authoringv1.NextRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("next section", err, nil)
	}
	if resp.Msg.GetComplete() {
		return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        []string{"All mandatory sections are filled."},
			ResultsHeading: "Next step",
			Results:        []string{"Run `author validate <session>` then `author finalize <session>`."},
		})
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{"Next section needing input:"},
		ResultsHeading: "Section",
		Results:        []string{formatSection(resp.Msg.GetSection())},
		RetrievalHints: []string{fmt.Sprintf("`author section-submit %s --section %s --content '…'`", ctx.Positional("session"), resp.Msg.GetSection().GetKey())},
	})
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	resp, err := h.client.ValidateStructure(context.Background(), connect.NewRequest(&authoringv1.ValidateStructureRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate structure", err, nil)
	}
	verdict := "INVALID"
	if resp.Msg.GetValid() {
		verdict = "valid"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Structure is %s (%d violation(s)).", verdict, len(resp.Msg.GetViolations()))},
		ResultsHeading: "Violations",
		Results:        formatViolations(resp.Msg.GetViolations()),
	})
}

func (h *handlers) autofill(ctx cliapp.RunContext) error {
	resp, err := h.client.Autofill(context.Background(), connect.NewRequest(&authoringv1.AutofillRequest{
		SessionId: ctx.Positional("session"),
		Sources:   parseSources(ctx.Flag("sources")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("autofill", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetResults()))
	filled := 0
	for _, r := range resp.Msg.GetResults() {
		results = append(results, formatAutofillResult(r))
		if r.GetFilled() {
			filled++
		}
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Autofilled %d of %d source(s).", filled, len(resp.Msg.GetResults()))},
		Changes: results,
	})
}

func (h *handlers) finalize(ctx cliapp.RunContext) error {
	resp, err := h.client.Finalize(context.Background(), connect.NewRequest(&authoringv1.FinalizeRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("finalize", err, nil)
	}
	plan := resp.Msg.GetPlan()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Finalized into plan %s (%s).", plan.GetId(), plan.GetSlug())},
		Changes: []string{fmt.Sprintf("Persisted %d phase(s) and %d reference(s).", len(plan.GetPhases()), len(plan.GetReferences()))},
		NextCommand: []string{
			fmt.Sprintf("`plans get %s` — view the structured plan", plan.GetSlug()),
		},
	})
}

func formatSection(sec *authoringv1.Section) string {
	state := "empty"
	switch {
	case sec.GetAutofilled():
		state = "autofilled"
	case sec.GetFilled():
		state = "filled"
	}
	tag := "optional"
	if sec.GetMandatory() {
		tag = "mandatory"
	}
	return fmt.Sprintf("[%s] %s (%s, %s)", sec.GetKey(), sec.GetLabel(), tag, state)
}

func formatViolations(violations []*authoringv1.StructureViolation) []string {
	out := make([]string, 0, len(violations))
	for _, v := range violations {
		out = append(out, fmt.Sprintf("%s: %s", v.GetSectionKey(), v.GetMessage()))
	}
	return out
}

func formatAutofillResult(r *authoringv1.AutofillResult) string {
	if r.GetDegraded() {
		return fmt.Sprintf("%s → degraded (%s)", r.GetSource(), r.GetDetail())
	}
	return fmt.Sprintf("%s → filled %s", r.GetSource(), r.GetSectionKey())
}

func nextLabel(key string) string {
	if strings.TrimSpace(key) == "" {
		return "complete"
	}
	return key
}

// parseSources splits the comma-separated --sources flag into the requested
// autofill source list. An empty flag yields nil, which the service treats as
// "run all sources".
func parseSources(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
