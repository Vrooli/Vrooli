package authoring

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	authoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring"
	authoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring/authoring_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

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
		Result:      []string{fmt.Sprintf("Started session %s for %q.", sess.GetId(), sess.GetTitle())},
		Changes:     append([]string{fmt.Sprintf("Seeded %d section(s); next: %s.", len(sess.GetSections()), nextLabel(sess.GetCurrentSectionKey()))}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
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
		RetrievalHints: formatStep(resp.Msg.GetStep()),
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
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Submitted section %q (%d violation(s)).", ctx.Flag("section"), len(resp.Msg.GetViolations()))},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
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
			Results:        formatStep(resp.Msg.GetStep()),
			RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
		})
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{"Next section needing input:"},
		ResultsHeading: "Section",
		Results:        []string{formatSection(resp.Msg.GetSection())},
		RetrievalHints: append(formatRecommendedActions(resp.Msg.GetStep()), formatStep(resp.Msg.GetStep())...),
	})
}

func (h *handlers) continueAuthoring(ctx cliapp.RunContext) error {
	resp, err := h.client.ContinueAuthoring(context.Background(), connect.NewRequest(&authoringv1.ContinueAuthoringRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("continue authoring", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Authoring session %s next step: %s.", ctx.Positional("session"), resp.Msg.GetStep().GetTitle())},
		ResultsHeading: "Next action",
		Results:        append(formatContinueAuthoring(resp.Msg), formatStep(resp.Msg.GetStep())...),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
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
		RetrievalHints: append(formatRecommendedActions(resp.Msg.GetStep()), formatStep(resp.Msg.GetStep())...),
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
		Result:      []string{fmt.Sprintf("Autofilled %d of %d source(s).", filled, len(resp.Msg.GetResults()))},
		Changes:     append(results, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) contextSubmit(ctx cliapp.RunContext) error {
	item := &sharedv1.RelevantContextItem{
		Kind:         parseContextKind(ctx.Flag("kind")),
		Label:        ctx.Flag("label"),
		Reason:       ctx.Flag("reason"),
		Instruction:  ctx.Flag("instruction"),
		Command:      ctx.Flag("command"),
		Argv:         parseArgv(ctx.Flag("argv"), ctx.Flag("command")),
		Target:       ctx.Flag("target"),
		Required:     ctx.BoolFlag("required"),
		RepeatPolicy: parseRepeatPolicy(ctx.Flag("repeat")),
		Source:       sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTHORED,
		Status:       sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_READY,
	}
	resp, err := h.client.SubmitRelevantContextItem(context.Background(), connect.NewRequest(&authoringv1.SubmitRelevantContextItemRequest{
		SessionId: ctx.Positional("session"),
		PhaseId:   ctx.Flag("phase"),
		Item:      item,
	}))
	if err != nil {
		return cliapp.WrapAPIError("submit relevant context", err, nil)
	}
	changes := []string{formatContextItem(resp.Msg.GetItem())}
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Submitted relevant context item (%d violation(s)).", len(resp.Msg.GetViolations()))},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) contextList(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRelevantContext(context.Background(), connect.NewRequest(&authoringv1.ListRelevantContextRequest{
		SessionId: ctx.Positional("session"),
		PhaseId:   ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list relevant context", err, nil)
	}
	items := make([]string, 0, len(resp.Msg.GetItems()))
	for _, item := range resp.Msg.GetItems() {
		items = append(items, formatContextItem(item))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Relevant context item(s): %d.", len(resp.Msg.GetItems()))},
		ResultsHeading: "Context",
		Results:        items,
		RetrievalHints: append(formatRecommendedActions(resp.Msg.GetStep()), formatStep(resp.Msg.GetStep())...),
	})
}

func (h *handlers) contextDiscover(ctx cliapp.RunContext) error {
	resp, err := h.client.DiscoverContextCandidates(context.Background(), connect.NewRequest(&authoringv1.DiscoverContextCandidatesRequest{
		SessionId:  ctx.Positional("session"),
		Concepts:   parseList(ctx.Flag("concepts")),
		Complexity: ctx.Flag("complexity"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("discover context candidates", err, nil)
	}
	candidates := make([]string, 0, len(resp.Msg.GetCandidates()))
	for _, candidate := range resp.Msg.GetCandidates() {
		candidates = append(candidates, formatContextCandidate(candidate))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Discovered %d context candidate(s).", len(resp.Msg.GetCandidates()))},
		Changes:     append(candidates, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) contextAccept(ctx cliapp.RunContext) error {
	resp, err := h.client.AcceptContextCandidate(context.Background(), connect.NewRequest(&authoringv1.AcceptContextCandidateRequest{
		SessionId:   ctx.Positional("session"),
		CandidateId: ctx.Positional("candidate"),
		PhaseId:     ctx.Flag("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("accept context candidate", err, nil)
	}
	changes := []string{formatContextCandidate(resp.Msg.GetCandidate()), formatContextItem(resp.Msg.GetItem())}
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Accepted context candidate %s (%d violation(s)).", ctx.Positional("candidate"), len(resp.Msg.GetViolations()))},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) contextReject(ctx cliapp.RunContext) error {
	resp, err := h.client.RejectContextCandidate(context.Background(), connect.NewRequest(&authoringv1.RejectContextCandidateRequest{
		SessionId:   ctx.Positional("session"),
		CandidateId: ctx.Positional("candidate"),
		Reason:      ctx.Flag("reason"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("reject context candidate", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Rejected context candidate %s.", ctx.Positional("candidate"))},
		Changes:     append([]string{formatContextCandidate(resp.Msg.GetCandidate())}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) phaseAdd(ctx cliapp.RunContext) error {
	resp, err := h.client.AddPhase(context.Background(), connect.NewRequest(&authoringv1.AddPhaseRequest{
		SessionId: ctx.Positional("session"),
		Title:     ctx.Flag("title"),
		Intent:    ctx.Flag("intent"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("add phase", err, nil)
	}
	phase := resp.Msg.GetPhase()
	changes := []string{formatPhase(phase)}
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Added phase %d (%s).", phase.GetOrder(), phase.GetId())},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) phaseGet(ctx cliapp.RunContext) error {
	resp, err := h.client.GetPhase(context.Background(), connect.NewRequest(&authoringv1.GetPhaseRequest{
		SessionId: ctx.Positional("session"),
		PhaseId:   ctx.Positional("phase"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get phase", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{formatPhase(resp.Msg.GetPhase())},
		ResultsHeading: "Guided step",
		Results:        formatStep(resp.Msg.GetStep()),
		RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) phaseSubmit(ctx cliapp.RunContext) error {
	resp, err := h.client.SubmitPhaseField(context.Background(), connect.NewRequest(&authoringv1.SubmitPhaseFieldRequest{
		SessionId: ctx.Positional("session"),
		PhaseId:   ctx.Positional("phase"),
		Field:     ctx.Flag("field"),
		Content:   ctx.Flag("content"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("submit phase field", err, nil)
	}
	changes := []string{fmt.Sprintf("Submitted phase field %q.", ctx.Flag("field"))}
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Phase field submitted (%d violation(s)).", len(resp.Msg.GetViolations()))},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) phaseNext(ctx cliapp.RunContext) error {
	resp, err := h.client.NextPhase(context.Background(), connect.NewRequest(&authoringv1.NextPhaseRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("next phase", err, nil)
	}
	if resp.Msg.GetComplete() {
		return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        []string{"All structured phases have required fields."},
			ResultsHeading: "Guided step",
			Results:        formatStep(resp.Msg.GetStep()),
			RetrievalHints: formatRecommendedActions(resp.Msg.GetStep()),
		})
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{"Next phase needing input:"},
		ResultsHeading: "Phase",
		Results:        []string{formatPhase(resp.Msg.GetPhase())},
		RetrievalHints: append(formatRecommendedActions(resp.Msg.GetStep()), formatStep(resp.Msg.GetStep())...),
	})
}

func (h *handlers) preview(ctx cliapp.RunContext) error {
	resp, err := h.client.PreviewPlan(context.Background(), connect.NewRequest(&authoringv1.PreviewPlanRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("preview", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{"Rendered preview of the in-progress plan (not persisted)."},
		ResultsHeading: "Markdown",
		Results:        []string{resp.Msg.GetMarkdown()},
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
		Result:      []string{fmt.Sprintf("Finalized into plan %s (%s).", plan.GetId(), plan.GetSlug())},
		Changes:     append([]string{fmt.Sprintf("Persisted %d phase(s) and %d reference(s).", len(plan.GetPhases()), len(plan.GetReferences()))}, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
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

func formatContextItem(item *sharedv1.RelevantContextItem) string {
	if item == nil {
		return ""
	}
	label := firstNonEmpty(item.GetLabel(), item.GetTarget(), item.GetCommand(), item.GetInstruction(), item.GetKind().String())
	parts := []string{fmt.Sprintf("%s: %s", contextKindLabel(item.GetKind()), label)}
	if item.GetScope() != sharedv1.RelevantContextScope_RELEVANT_CONTEXT_SCOPE_UNSPECIFIED {
		parts = append(parts, "scope="+contextScopeLabel(item.GetScope()))
	}
	if item.GetPhaseId() != "" {
		parts = append(parts, "phase="+item.GetPhaseId())
	}
	if item.GetRepeatPolicy() != sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED {
		parts = append(parts, "repeat="+contextRepeatLabel(item.GetRepeatPolicy()))
	}
	if item.GetCommand() != "" {
		parts = append(parts, "`"+item.GetCommand()+"`")
	}
	return strings.Join(parts, " | ")
}

func formatContextCandidate(candidate *authoringv1.ContextCandidate) string {
	if candidate == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("%s [%s] %s", candidate.GetId(), candidate.GetStatus(), formatContextItem(candidate.GetItem()))}
	if candidate.GetConcept() != "" {
		parts = append(parts, "concept="+candidate.GetConcept())
	}
	if candidate.GetSource() != "" {
		parts = append(parts, "source="+candidate.GetSource())
	}
	if candidate.GetDegraded() {
		parts = append(parts, "degraded="+candidate.GetDetail())
	}
	if candidate.GetRejectionReason() != "" {
		parts = append(parts, "rejected="+candidate.GetRejectionReason())
	}
	return strings.Join(parts, " | ")
}

func formatPhase(phase *authoringv1.PhaseDraft) string {
	return fmt.Sprintf("Phase %d [%s]: %s — %s", phase.GetOrder(), phase.GetId(), phase.GetTitle(), phase.GetIntent())
}

func formatStep(step *sharedv1.GuidedStep) []string {
	if step == nil || strings.TrimSpace(step.GetStepKind()) == "" {
		return nil
	}
	out := []string{fmt.Sprintf("Current Step (%s): %s", step.GetStepKind(), step.GetSummary())}
	for _, input := range step.GetRequiredInputs() {
		out = append(out, "Required input: "+input)
	}
	for _, item := range step.GetInstructions() {
		out = append(out, "- "+item)
	}
	if len(step.GetExamples()) > 0 {
		out = append(out, "Example: "+step.GetExamples()[0])
	}
	for _, action := range step.GetNextActions() {
		if action.GetKind() == sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED {
			continue
		}
		out = append(out, fmt.Sprintf("%s: `%s` — %s", actionKindLabel(action.GetKind()), shellCommand(action.GetArgv()), action.GetReason()))
	}
	return out
}

func formatRecommendedActions(step *sharedv1.GuidedStep) []string {
	if step == nil {
		return nil
	}
	out := make([]string, 0, 1)
	for _, action := range step.GetNextActions() {
		if action.GetKind() != sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED {
			continue
		}
		out = append(out, fmt.Sprintf("`%s` — %s", shellCommand(action.GetArgv()), action.GetReason()))
	}
	return out
}

func actionKindLabel(kind sharedv1.NextActionKind) string {
	switch kind {
	case sharedv1.NextActionKind_NEXT_ACTION_KIND_ALTERNATIVE:
		return "Alternative"
	case sharedv1.NextActionKind_NEXT_ACTION_KIND_OPTIONAL:
		return "Optional"
	case sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOVERY:
		return "Recovery"
	default:
		return "Action"
	}
}

func shellCommand(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, arg := range argv {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.IndexFunc(arg, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' || r == '<' || r == '>' || r == '[' || r == ']' || r == ':' || r == ';' || r == '|' || r == '&'
	}) < 0 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
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
	return parseList(raw)
}

func formatContinueAuthoring(resp *authoringv1.ContinueAuthoringResponse) []string {
	var out []string
	if resp.GetReadyToFinalize() {
		out = append(out, "ready to finalize")
	}
	if sec := resp.GetSection(); sec != nil {
		out = append(out, "section: "+formatSection(sec))
	}
	if ph := resp.GetPhase(); ph != nil {
		out = append(out, fmt.Sprintf("phase: %d %s (%s)", ph.GetOrder(), ph.GetTitle(), ph.GetId()))
	}
	if v := resp.GetViolations(); len(v) > 0 {
		out = append(out, formatViolations(v)...)
	}
	if len(out) == 0 {
		out = append(out, "no additional authoring object returned")
	}
	return out
}

func parseList(raw string) []string {
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

func parseContextKind(raw string) sharedv1.RelevantContextKind {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "skill":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SKILL
	case "doc":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC
	case "command":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND
	case "search":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SEARCH
	case "code_ref":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_CODE_REF
	case "req_ref":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_REQ_REF
	case "note":
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE
	default:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_UNSPECIFIED
	}
}

func parseRepeatPolicy(raw string) sharedv1.RelevantContextRepeatPolicy {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "once_per_execution", "":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ONCE_PER_EXECUTION
	case "on_resume":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ON_RESUME
	case "every_phase":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_EVERY_PHASE
	case "phase_entry":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY
	case "as_needed":
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_AS_NEEDED
	default:
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED
	}
}

func parseArgv(raw, command string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if strings.TrimSpace(command) == "" {
			return nil
		}
		return strings.Fields(command)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func contextKindLabel(kind sharedv1.RelevantContextKind) string {
	return strings.TrimPrefix(strings.ToLower(kind.String()), "relevant_context_kind_")
}

func contextScopeLabel(scope sharedv1.RelevantContextScope) string {
	return strings.TrimPrefix(strings.ToLower(scope.String()), "relevant_context_scope_")
}

func contextRepeatLabel(policy sharedv1.RelevantContextRepeatPolicy) string {
	return strings.TrimPrefix(strings.ToLower(policy.String()), "relevant_context_repeat_policy_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
