package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"plan-manager/cli/internal/steprender"

	"connectrpc.com/connect"

	repocontract "github.com/vrooli/repo-contract-go"

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
	handle := firstNonEmpty(sess.GetPlanSlug(), sess.GetId())
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Started session %s for %q.", handle, sess.GetTitle())},
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
	changes := []string{fmt.Sprintf("Next: %s.", nextLabel(resp.Msg.GetProgress().GetCurrentSectionKey()))}
	changes = append(changes, formatProgress(resp.Msg.GetProgress())...)
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("%s (%d violation(s)).", summaryLine(resp.Msg.GetSummary()), len(resp.Msg.GetViolations()))},
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
	results = append(results, formatProgress(resp.Msg.GetProgress())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Autofilled %d of %d source(s).", filled, len(resp.Msg.GetResults()))},
		Changes:     append(results, formatStep(resp.Msg.GetStep())...),
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) contextSubmit(ctx cliapp.RunContext) error {
	argv, err := parseContextArgv(ctx.Flag("argv-json"), ctx.Flag("argv"), ctx.Flag("command"))
	if err != nil {
		return err
	}
	item := &sharedv1.RelevantContextItem{
		Kind:         parseContextKind(ctx.Flag("kind")),
		Label:        ctx.Flag("label"),
		Reason:       ctx.Flag("reason"),
		Instruction:  ctx.Flag("instruction"),
		Command:      ctx.Flag("command"),
		Argv:         argv,
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
	// A violation-rejected item is NOT appended — the gate is still open. That
	// must be unmissable: violations lead, the result says NOT accepted, and
	// the command exits non-zero.
	if len(resp.Msg.GetViolations()) > 0 && !resp.Msg.GetAccepted() {
		changes := formatViolations(resp.Msg.GetViolations())
		changes = append(changes, formatStep(resp.Msg.GetStep())...)
		if renderErr := cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:      []string{"NOT accepted — the context item was rejected and the phase requirement is still unmet."},
			Changes:     changes,
			NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
		}); renderErr != nil {
			return renderErr
		}
		return fmt.Errorf("context item rejected with %d violation(s); the gate is still open", len(resp.Msg.GetViolations()))
	}
	changes := []string{formatContextItem(resp.Msg.GetItem())}
	changes = append(changes, formatProgress(resp.Msg.GetProgress())...)
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{"Accepted relevant context item."},
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

func (h *handlers) contextUpdate(ctx cliapp.RunContext) error {
	argv, err := parseContextArgv(ctx.Flag("argv-json"), ctx.Flag("argv"), ctx.Flag("command"))
	if err != nil {
		return err
	}
	item := &sharedv1.RelevantContextItem{
		Id:           ctx.Positional("item"),
		Kind:         parseContextKind(ctx.Flag("kind")),
		Label:        ctx.Flag("label"),
		Reason:       ctx.Flag("reason"),
		Instruction:  ctx.Flag("instruction"),
		Command:      ctx.Flag("command"),
		Argv:         argv,
		Target:       ctx.Flag("target"),
		Required:     ctx.BoolFlag("required"),
		RepeatPolicy: parseRepeatPolicy(ctx.Flag("repeat")),
		Source:       sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTHORED,
		Status:       sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_READY,
	}
	resp, err := h.client.UpdateRelevantContextItem(context.Background(), connect.NewRequest(&authoringv1.UpdateRelevantContextItemRequest{
		SessionId: ctx.Positional("session"),
		PhaseId:   ctx.Flag("phase"),
		ItemId:    ctx.Positional("item"),
		Item:      item,
	}))
	if err != nil {
		return cliapp.WrapAPIError("update relevant context", err, nil)
	}
	changes := []string{formatContextItem(resp.Msg.GetItem())}
	changes = append(changes, formatProgress(resp.Msg.GetProgress())...)
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("%s (%d violation(s)).", summaryLine(resp.Msg.GetSummary()), len(resp.Msg.GetViolations()))},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) contextRemove(ctx cliapp.RunContext) error {
	resp, err := h.client.RemoveRelevantContextItem(context.Background(), connect.NewRequest(&authoringv1.RemoveRelevantContextItemRequest{
		SessionId: ctx.Positional("session"),
		PhaseId:   ctx.Flag("phase"),
		ItemId:    ctx.Positional("item"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("remove relevant context", err, nil)
	}
	changes := formatProgress(resp.Msg.GetProgress())
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("%s (%d violation(s)).", summaryLine(resp.Msg.GetSummary()), len(resp.Msg.GetViolations()))},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) getSession(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSession(context.Background(), connect.NewRequest(&authoringv1.GetSessionRequest{
		SessionId: ctx.Positional("session"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get session", err, nil)
	}
	sess := resp.Msg.GetSession()
	sections := make([]string, 0, len(sess.GetSections()))
	for _, sec := range sess.GetSections() {
		sections = append(sections, formatSection(sec))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Session %s (%q): %d section(s), %d phase(s), %d context item(s).", sess.GetId(), sess.GetTitle(), len(sess.GetSections()), len(sess.GetPhaseDrafts()), len(sess.GetRelevantContext()))},
		ResultsHeading: "Sections",
		Results:        sections,
		RetrievalHints: formatStep(resp.Msg.GetStep()),
	})
}

func (h *handlers) skillPack(ctx cliapp.RunContext) error {
	resp, err := h.client.DiscoverSkillPack(context.Background(), connect.NewRequest(&authoringv1.DiscoverSkillPackRequest{
		SessionId:  ctx.Positional("session"),
		PhaseId:    ctx.Flag("phase"),
		Concepts:   parseList(ctx.Flag("concepts")),
		Complexity: ctx.Flag("complexity"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("discover skill pack", err, nil)
	}
	changes := make([]string, 0, len(resp.Msg.GetAddedItems())+len(resp.Msg.GetKeptItems())+12)
	if summary := strings.TrimSpace(resp.Msg.GetResultsSummary()); summary != "" {
		changes = append(changes, summary)
	}
	if budget := strings.TrimSpace(resp.Msg.GetBudgetStatus()); budget != "" {
		changes = append(changes, "budget: "+budget)
	}
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	if resp.Msg.GetDegraded() {
		changes = append(changes, "degraded: "+resp.Msg.GetDegradedReason())
	}
	if read := strings.TrimSpace(resp.Msg.GetReadCommand()); read != "" {
		changes = append(changes, "read: "+read)
	}
	if recommended := strings.TrimSpace(resp.Msg.GetRecommendedReadCommand()); recommended != "" && recommended != resp.Msg.GetReadCommand() {
		changes = append(changes, "recommended read: "+recommended)
	}
	for _, item := range resp.Msg.GetAddedItems() {
		changes = append(changes, "added: "+formatContextItem(item))
	}
	for _, item := range resp.Msg.GetKeptItems() {
		changes = append(changes, "kept: "+formatContextItem(item))
	}
	changes = append(changes, formatProgress(resp.Msg.GetProgress())...)
	changes = append(changes, "search-hub: run `search-hub query \"<intent>\" --type record,doc,skill` directly for docs/records/code, then add only durable context or references.")
	changes = append(changes, "edit: use author context-submit/context-update/context-remove to adjust relevant context.")
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Skill pack: %d added, %d already present.", len(resp.Msg.GetAddedItems()), len(resp.Msg.GetKeptItems()))},
		Changes:     changes,
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
	// Optional --set pairs make add+fill one command: the new phase's fields
	// land in a single batch call right after the add.
	if items, batchErr := batchFieldWrites(ctx, phase.GetId()); batchErr != nil {
		return batchErr
	} else if len(items) > 0 {
		batchResp, batchErr := h.client.SubmitFields(context.Background(), connect.NewRequest(&authoringv1.SubmitFieldsRequest{
			SessionId: ctx.Positional("session"),
			Items:     items,
		}))
		if batchErr != nil {
			return cliapp.WrapAPIError("fill added phase", batchErr, nil)
		}
		return renderSubmitFields(ctx, fmt.Sprintf("Added phase %d (%s); field batch", phase.GetOrder(), phase.GetId()), items, batchResp)
	}
	changes := []string{formatPhase(phase)}
	changes = append(changes, formatProgress(resp.Msg.GetProgress())...)
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	result := fmt.Sprintf("Added phase %d (%s).", phase.GetOrder(), phase.GetId())
	if s := summaryLine(resp.Msg.GetSummary()); s != "" {
		result = fmt.Sprintf("%s — %s.", strings.TrimRight(result, "."), s)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{result},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

func (h *handlers) phaseMove(ctx cliapp.RunContext) error {
	resp, err := h.client.MovePhase(context.Background(), connect.NewRequest(&authoringv1.MovePhaseRequest{
		SessionId:     ctx.Positional("session"),
		PhaseId:       ctx.Positional("phase"),
		BeforePhaseId: ctx.Flag("before"),
		AfterPhaseId:  ctx.Flag("after"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("move phase", err, nil)
	}
	changes := []string{formatPhase(resp.Msg.GetPhase())}
	changes = append(changes, formatProgress(resp.Msg.GetProgress())...)
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("%s (%d violation(s)).", summaryLine(resp.Msg.GetSummary()), len(resp.Msg.GetViolations()))},
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

// batchFieldWrites parses repeated --set / --set-file flag values into
// FieldWrite items. Key grammar: with a phaseRef the key is a bare phase field
// name; session-scope keys are either a section key or `<phase-ref>.<field>`.
// A --set-file value is `key=path` (optionally `key=@path`); the file's
// contents become the field content — the escape hatch for long content.
func batchFieldWrites(ctx cliapp.RunContext, phaseRef string) ([]*authoringv1.FieldWrite, error) {
	var items []*authoringv1.FieldWrite
	appendPair := func(raw string, fromFile bool) error {
		flag := "--set"
		if fromFile {
			flag = "--set-file"
		}
		key, val, ok := strings.Cut(raw, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return fmt.Errorf("%s %q: expected <key>=<content>", flag, raw)
		}
		if fromFile {
			path := strings.TrimPrefix(strings.TrimSpace(val), "@")
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("%s %s: %w", flag, key, err)
			}
			val = string(data)
		}
		item := &authoringv1.FieldWrite{Content: val}
		switch {
		case phaseRef != "":
			item.Scope = &authoringv1.FieldWrite_Phase{Phase: &authoringv1.PhaseFieldRef{PhaseRef: phaseRef, Field: key}}
		case strings.Contains(key, "."):
			ref, field, _ := strings.Cut(key, ".")
			item.Scope = &authoringv1.FieldWrite_Phase{Phase: &authoringv1.PhaseFieldRef{PhaseRef: ref, Field: field}}
		default:
			item.Scope = &authoringv1.FieldWrite_SectionKey{SectionKey: key}
		}
		items = append(items, item)
		return nil
	}
	for _, raw := range ctx.FlagValues("set") {
		if err := appendPair(raw, false); err != nil {
			return nil, err
		}
	}
	for _, raw := range ctx.FlagValues("set-file") {
		if err := appendPair(raw, true); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// fieldWriteLabel names one batch item for the per-item result line.
func fieldWriteLabel(item *authoringv1.FieldWrite) string {
	if phase := item.GetPhase(); phase != nil {
		return phase.GetPhaseRef() + "." + phase.GetField()
	}
	return item.GetSectionKey()
}

// renderSubmitFields renders the per-item batch outcome: every accepted and
// rejected field named on its own line, then progress and the guided step.
func renderSubmitFields(ctx cliapp.RunContext, verb string, items []*authoringv1.FieldWrite, resp *connect.Response[authoringv1.SubmitFieldsResponse]) error {
	results := resp.Msg.GetResults()
	accepted := 0
	lines := make([]string, 0, len(results))
	for _, result := range results {
		label := ""
		if i := int(result.GetIndex()); i >= 0 && i < len(items) {
			label = fieldWriteLabel(items[i])
		}
		if result.GetAccepted() {
			accepted++
			lines = append(lines, fmt.Sprintf("✔ %s — %s", label, result.GetSummary()))
			continue
		}
		line := fmt.Sprintf("✖ %s — REJECTED: %s", label, result.GetSummary())
		lines = append(lines, line)
		for _, violation := range result.GetViolations() {
			if msg := violation.GetMessage(); msg != "" && !strings.Contains(line, msg) {
				lines = append(lines, "    "+msg)
			}
		}
	}
	changes := lines
	changes = append(changes, formatProgress(resp.Msg.GetProgress())...)
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	result := fmt.Sprintf("%s: %d of %d field write(s) accepted.", verb, accepted, len(results))
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{result},
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

// submit is the session-scope batch: repeated --set section_key=content and/or
// --set <phase-ref>.<field>=content in ONE call. For an already-written plan
// document, `plans import` remains the whole-document path.
func (h *handlers) submit(ctx cliapp.RunContext) error {
	items, err := batchFieldWrites(ctx, "")
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("provide at least one --set <key>=<content> (section key or <phase-ref>.<field>); for a whole already-written plan document use `plans import`")
	}
	resp, err := h.client.SubmitFields(context.Background(), connect.NewRequest(&authoringv1.SubmitFieldsRequest{
		SessionId: ctx.Positional("session"),
		Items:     items,
	}))
	if err != nil {
		return cliapp.WrapAPIError("submit fields", err, nil)
	}
	return renderSubmitFields(ctx, "Batch submit", items, resp)
}

func (h *handlers) phaseSubmit(ctx cliapp.RunContext) error {
	phaseRef := ctx.Positional("phase")
	if items, err := batchFieldWrites(ctx, phaseRef); err != nil {
		return err
	} else if len(items) > 0 {
		if ctx.Flag("field") != "" || ctx.Flag("content") != "" {
			return fmt.Errorf("use either --set pairs or the single --field/--content form, not both")
		}
		resp, err := h.client.SubmitFields(context.Background(), connect.NewRequest(&authoringv1.SubmitFieldsRequest{
			SessionId: ctx.Positional("session"),
			Items:     items,
		}))
		if err != nil {
			return cliapp.WrapAPIError("submit phase fields", err, nil)
		}
		return renderSubmitFields(ctx, fmt.Sprintf("Phase %s batch", phaseRef), items, resp)
	}
	if ctx.Flag("field") == "" {
		return fmt.Errorf("provide --field/--content or one or more --set <field>=<content> pairs")
	}
	resp, err := h.client.SubmitPhaseField(context.Background(), connect.NewRequest(&authoringv1.SubmitPhaseFieldRequest{
		SessionId: ctx.Positional("session"),
		PhaseId:   phaseRef,
		Field:     ctx.Flag("field"),
		Content:   ctx.Flag("content"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("submit phase field", err, nil)
	}
	changes := []string{formatPhase(resp.Msg.GetPhase())}
	changes = append(changes, formatProgress(resp.Msg.GetProgress())...)
	if v := resp.Msg.GetViolations(); len(v) > 0 {
		changes = append(changes, formatViolations(v)...)
	}
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("%s (%d violation(s)).", summaryLine(resp.Msg.GetSummary()), len(resp.Msg.GetViolations()))},
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
		SessionId:     ctx.Positional("session"),
		WorkspaceRoot: finalizeWorkspaceRoot(ctx.Flag("workspace")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("finalize", err, nil)
	}
	if ctx.JSON() {
		if ctx.BoolFlag("full") {
			return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
		}
		return json.NewEncoder(ctx.Stdout()).Encode(compactFinalizePlan(resp.Msg))
	}
	changes := formatFinalizePlan(resp.Msg, ctx.BoolFlag("full"))
	changes = append(changes, formatStep(resp.Msg.GetStep())...)
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      finalizeResultLines(resp.Msg),
		Changes:     changes,
		NextCommand: formatRecommendedActions(resp.Msg.GetStep()),
	})
}

// finalizeWorkspaceRoot resolves the workspace root to stamp on the finalized
// plan: the explicit --workspace flag, else the resolved repo root (matching
// how the plans commands scope reads), else empty (unscoped).
func finalizeWorkspaceRoot(raw string) string {
	root := strings.TrimSpace(raw)
	if root == "" {
		resolved, err := repocontract.ResolveRepoRoot()
		if err != nil {
			return ""
		}
		root = resolved
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return filepath.Clean(abs)
}

// finalizeResultLines leads with the honest persistence report: where the plan
// row lives, then identity, then the mirror outcome. An idempotent re-run and a
// failed mirror write are called out loudly instead of blending into happy
// output.
func finalizeResultLines(msg *authoringv1.FinalizeResponse) []string {
	plan := msg.GetPlan()
	handle := firstNonEmpty(plan.GetSlug(), plan.GetId())
	var out []string
	if msg.GetAlreadyFinalized() {
		out = append(out, fmt.Sprintf("Already finalized at %s → plan %s (no new plan written).",
			firstNonEmpty(msg.GetFinalizedAt(), "an earlier run"), handle))
	} else {
		out = append(out, fmt.Sprintf("Finalized plan %s.", handle))
	}
	store := firstNonEmpty(msg.GetStorePath(), "unknown (server did not report a store path)")
	workspace := firstNonEmpty(msg.GetWorkspaceRoot(), "unscoped")
	out = append(out, fmt.Sprintf("plan persisted: %s (workspace %s)", store, workspace))
	if mirror := msg.GetMirror(); mirror != nil {
		status := mirrorStatusString(mirror)
		switch status {
		case "fresh":
			out = append(out, "mirror: fresh — "+firstNonEmpty(mirror.GetPath(), mirror.GetRelativePath()))
		case "write_failed":
			out = append(out,
				"WARNING: mirror write FAILED — the plan IS persisted in SQLite (source of truth) but its markdown mirror was not written.",
				"  reason: "+firstNonEmpty(mirror.GetLastError(), "not reported"),
				"  recover: plan-manager plans reconcile --repair-mirrors",
			)
		default:
			if status != "" && status != "unspecified" {
				out = append(out, "mirror: "+status)
			}
		}
	}
	return out
}

type finalizePlanJSON struct {
	ID               string   `json:"id,omitempty"`
	Slug             string   `json:"slug,omitempty"`
	Title            string   `json:"title,omitempty"`
	Status           string   `json:"status,omitempty"`
	StorePath        string   `json:"store_path,omitempty"`
	Workspace        string   `json:"workspace,omitempty"`
	AlreadyFinalized bool     `json:"already_finalized,omitempty"`
	FinalizedAt      string   `json:"finalized_at,omitempty"`
	MirrorPath       string   `json:"mirror_path,omitempty"`
	MirrorStatus     string   `json:"mirror_status,omitempty"`
	MirrorError      string   `json:"mirror_error,omitempty"`
	NextCommands     []string `json:"next_commands,omitempty"`
}

func compactFinalizePlan(msg *authoringv1.FinalizeResponse) finalizePlanJSON {
	plan := msg.GetPlan()
	step := msg.GetStep()
	if plan == nil {
		return finalizePlanJSON{NextCommands: formatRecommendedActions(step)}
	}
	out := finalizePlanJSON{
		ID:               plan.GetId(),
		Slug:             plan.GetSlug(),
		Title:            plan.GetTitle(),
		Status:           planStatusString(plan),
		StorePath:        msg.GetStorePath(),
		Workspace:        msg.GetWorkspaceRoot(),
		AlreadyFinalized: msg.GetAlreadyFinalized(),
		FinalizedAt:      msg.GetFinalizedAt(),
		NextCommands:     formatRecommendedActions(step),
	}
	if mirror := firstMirror(msg.GetMirror(), plan.GetMirror()); mirror != nil {
		out.MirrorStatus = mirrorStatusString(mirror)
		out.MirrorPath = firstNonEmpty(mirror.GetRelativePath(), mirror.GetPath())
		out.MirrorError = mirror.GetLastError()
	}
	return out
}

// firstMirror prefers the response-level computed mirror (threaded from the
// write) over the plan echo's read-model state.
func firstMirror(candidates ...*sharedv1.RenderedPlanMirror) *sharedv1.RenderedPlanMirror {
	for _, m := range candidates {
		if m != nil {
			return m
		}
	}
	return nil
}

func formatFinalizePlan(msg *authoringv1.FinalizeResponse, full bool) []string {
	plan := msg.GetPlan()
	if plan == nil {
		return nil
	}
	out := []string{
		"id: " + plan.GetId(),
		"slug: " + plan.GetSlug(),
		"title: " + plan.GetTitle(),
		"status: " + planStatusString(plan),
	}
	if mirror := firstMirror(msg.GetMirror(), plan.GetMirror()); mirror != nil {
		status := mirrorStatusString(mirror)
		if status != "unspecified" || mirror.GetPath() != "" || mirror.GetRelativePath() != "" {
			out = append(out, "mirror_status: "+status)
			if mirror.GetRelativePath() != "" {
				out = append(out, "mirror_path: "+mirror.GetRelativePath())
			} else if mirror.GetPath() != "" {
				out = append(out, "mirror_path: "+mirror.GetPath())
			}
		}
	}
	if full {
		out = append(out,
			fmt.Sprintf("phases: %d", len(plan.GetPhases())),
			fmt.Sprintf("references: %d", len(plan.GetReferences())),
			"content_hash: "+plan.GetContentHash(),
		)
	}
	return out
}

func planStatusString(plan *sharedv1.Plan) string {
	if plan == nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(plan.GetStatus().String()), "plan_status_")
}

func mirrorStatusString(mirror *sharedv1.RenderedPlanMirror) string {
	if mirror == nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(mirror.GetStatus().String()), "rendered_mirror_status_")
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

// formatProgress renders the compact navigation snapshot every mutation returns
// in place of the full session graph: filled/total counts plus the remaining
// required inputs the agent still owes before finalize.
func formatProgress(p *authoringv1.AuthoringProgress) []string {
	if p == nil {
		return nil
	}
	ready := ""
	if p.GetReadyToFinalize() {
		ready = " — ready to finalize"
	}
	out := []string{fmt.Sprintf("Progress: %d/%d mandatory section(s), %d/%d phase(s)%s.",
		p.GetMandatorySectionsFilled(), p.GetMandatorySectionsTotal(),
		p.GetPhasesComplete(), p.GetPhasesTotal(), ready)}
	for _, r := range p.GetRemainingRequiredInputs() {
		out = append(out, "Remaining: "+r)
	}
	return out
}

// summaryLine renders the one-line mutation acknowledgement.
func summaryLine(s *authoringv1.AuthoringMutationSummary) string {
	if s == nil {
		return ""
	}
	return s.GetSummary()
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

func formatPhase(phase *authoringv1.PhaseDraft) string {
	return fmt.Sprintf("Phase %d [%s]: %s — %s", phase.GetOrder(), phase.GetId(), phase.GetTitle(), phase.GetIntent())
}

func formatStep(step *sharedv1.GuidedStep) []string {
	return steprender.StepLines(step)
}

func formatRecommendedActions(step *sharedv1.GuidedStep) []string {
	return steprender.RecommendedActions(step)
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
	out = append(out, formatProgress(resp.GetProgress())...)
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
	case "":
		// Unset means "let the server pick the scope-appropriate default"
		// (phase_entry for phase scope, once_per_execution for global) — do NOT
		// hard-code once_per_execution here, which silently mis-set phase context.
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED
	case "once_per_execution":
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

func parseContextArgv(rawJSON, rawCSV, command string) ([]string, error) {
	rawJSON = strings.TrimSpace(rawJSON)
	if rawJSON != "" {
		var out []string
		if err := json.Unmarshal([]byte(rawJSON), &out); err != nil {
			return nil, fmt.Errorf("invalid --argv-json: %w", err)
		}
		return trimEmptyArgs(out), nil
	}
	return parseArgv(rawCSV, command)
}

func parseArgv(raw, command string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if strings.TrimSpace(command) == "" {
			return nil, nil
		}
		return splitShellArgs(command)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out, nil
}

func trimEmptyArgs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, arg := range in {
		if strings.TrimSpace(arg) != "" {
			out = append(out, arg)
		}
	}
	return out
}

func splitShellArgs(command string) ([]string, error) {
	var (
		out         []string
		b           strings.Builder
		quote       rune
		escaped     bool
		tokenActive bool
	)
	for _, r := range command {
		switch {
		case escaped:
			b.WriteRune(r)
			tokenActive = true
			escaped = false
		case r == '\\':
			escaped = true
			tokenActive = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
				tokenActive = true
			}
		case r == '\'' || r == '"':
			quote = r
			tokenActive = true
		case unicode.IsSpace(r):
			if tokenActive {
				out = append(out, b.String())
				b.Reset()
				tokenActive = false
			}
		default:
			b.WriteRune(r)
			tokenActive = true
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted argument in --command")
	}
	if tokenActive {
		out = append(out, b.String())
	}
	return out, nil
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
