package agentsessions

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strings"
)

// sessionDoctrine is the universal band: byte-identical for every session of
// every kind. It states the two invariants that govern all sessions. Everything
// kind-specific belongs in a later band.
const sessionDoctrine = `You are in a Swarm Manager Agent Session: a durable conversation led by a human operator who is present now.

Two rules govern every session.

Propose, never apply. You may recommend any change. Swarm Manager applies it after the operator explicitly accepts a typed proposal. Do not mutate project state as a side effect of conversation.

Resolve in this session. Reach a concrete outcome before the conversation ends: a reviewed proposal, a started transition, a design record, or a recorded reason to do nothing. Do not route the outcome to an autonomous agent's inbox, a team heartbeat, or a queue that only a scheduled loop drains.

Answer first, then ask. Give the best answer available from the context below, then state what you assumed and what would change the answer. Do not spend the operator's turn on clarifying questions alone.`

// subjectForKind is the kind band: byte-identical for every session of one
// kind. It states the subject only. The methodology lives in the skill.
func subjectForKind(kind Kind) string {
	switch kind {
	case KindMetaOrchestration:
		return "Subject: the product. What should exist, and what work makes that true. Shape the operator's raw material into goals, milestones, and backlog items."
	case KindSwarmOperations:
		return "Subject: work that already exists in the ledger. Its true state, what matters most next, and the registered transition that moves it."
	case KindWorkflowAuthoring:
		return "Subject: the machine, not the product. How the operator and agents work together — skills, prompts, workflows, transitions, briefs, session surfaces, and agent profiles."
	default:
		return "Subject: this session's declared kind."
	}
}

func buildInitialPrompt(session Session, message Message, attachments []Attachment) string {
	sections := []promptSection{
		newPromptSection(promptSectionKindDoctrine, "", sessionDoctrine),
		newPromptSection(promptSectionKindSubject, attr("name", string(session.Kind)), kindBand(session)),
	}

	if section, ok := proposalTargetSection(session); ok {
		sections = append(sections, section)
	}

	// Identity is volatile by construction — it is unique per session — so it
	// sits below every stable band. Emitting it earlier is what collapsed the
	// cacheable prefix for every session ever started.
	sections = append(sections, newPromptSection(promptSectionKindIdentity, "", "Session ID: "+session.ID))

	sections = append(sections, contextSections(message.Context)...)
	if section, ok := imagesSection(attachments); ok {
		sections = append(sections, section)
	}
	sections = append(sections, newPromptSection(promptSectionKindOperatorMsg, "", operatorMessageBody(message.Content)))

	return assemblePrompt(sections)
}

func buildContinuationPrompt(message Message, attachments []Attachment) string {
	if len(message.Context) == 0 && len(attachments) == 0 {
		return strings.TrimSpace(message.Content)
	}
	sections := contextSections(message.Context)
	if section, ok := imagesSection(attachments); ok {
		sections = append(sections, section)
	}
	sections = append(sections, newPromptSection(promptSectionKindOperatorMsg, "", operatorMessageBody(message.Content)))
	return assemblePrompt(sections)
}

// assemblePrompt wraps every reference section in one <context> block, ordered
// by volatility, and leaves task-scoped sections outside it in prose. Sorting is
// stable so that sections within a band keep their append order.
func assemblePrompt(sections []promptSection) string {
	ordered := make([]promptSection, len(sections))
	copy(ordered, sections)
	sort.SliceStable(ordered, func(i, j int) bool {
		return promptSectionScopeOf(ordered[i].Kind) < promptSectionScopeOf(ordered[j].Kind)
	})

	var contextParts []string
	var taskParts []string
	for _, section := range ordered {
		body := strings.TrimSpace(section.Content)
		if body == "" {
			continue
		}
		element := promptSectionElement(section.Kind)
		if promptSectionScopeOf(section.Kind) == promptScopeTask {
			taskParts = append(taskParts, fmt.Sprintf("<%s>\n%s\n</%s>", element, body, element))
			continue
		}
		contextParts = append(contextParts, fmt.Sprintf("<%s%s>\n%s\n</%s>", element, section.Attrs, body, element))
	}

	var parts []string
	if len(contextParts) > 0 {
		parts = append(parts, "<context>\n\n"+strings.Join(contextParts, "\n\n")+"\n\n</context>")
	}
	parts = append(parts, taskParts...)
	return strings.Join(parts, "\n\n")
}

// kindBand states the subject and names the authoritative skill. The skill
// carries the complete methodology and is read by the agent before its first
// answer; the startup brief carries only current state.
func kindBand(session Session) string {
	var b strings.Builder
	b.WriteString(subjectForKind(session.Kind))
	if skill := strings.TrimSpace(session.SkillID); skill != "" {
		fmt.Fprintf(&b, "\n\nYour complete methodology is the Prompt Manager skill `%s`. Read it in full before your first answer:\n\n    prompt-manager skill read %s\n\nThe startup brief below is current state, not procedure. Follow the skill.", skill, skill)
	}
	return b.String()
}

func proposalTargetSection(session Session) (promptSection, bool) {
	target := session.ProposalTarget
	if target == nil {
		return promptSection{}, false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "This is a proposal session for %s %q. Return any recommended change as the skill's fenced mutation_list JSON envelope. Never mutate files directly.", target.Type, target.Ref)
	if target.Type == ContextGoal {
		b.WriteString("\n\nCopy the attached goal context Metadata.base_version exactly into the envelope's base_version field. The server rejects goal proposals without that optimistic-concurrency value.")
		b.WriteString("\n\nFor create_milestone or update_milestone, use a goal_milestone object (not milestone) with at least name and title. Goal proposals may use only goal graph operations.")
	}
	attrs := attr("type", string(target.Type)) + attr("ref", target.Ref)
	return newPromptSection(promptSectionKindProposalTarget, attrs, b.String()), true
}

// contextSections splits resolved context into the startup brief and everything
// else. The brief earns its own element because it is the answer-first source:
// naming it lets the agent tell a live state snapshot from operator-chosen
// attachments without parsing a numbered list.
func contextSections(items []ContextItem) []promptSection {
	var brief []ContextItem
	var rest []ContextItem
	for _, item := range items {
		switch item.Type {
		case ContextStartupBrief, ContextOperationsBriefing:
			brief = append(brief, item)
		default:
			rest = append(rest, item)
		}
	}

	var sections []promptSection
	if len(brief) > 0 {
		var b strings.Builder
		b.WriteString("Answer broad status, planning, and authoring questions from this brief. Run at most one targeted drill-down or refresh before your first useful answer.\n")
		writeContextItems(&b, brief)
		sections = append(sections, newPromptSection(promptSectionKindStartupBrief, "", b.String()))
	}
	if len(rest) > 0 {
		var b strings.Builder
		b.WriteString("The operator attached these deliberately. Treat them as the subject of the message unless the message says otherwise.\n")
		writeContextItems(&b, rest)
		sections = append(sections, newPromptSection(promptSectionKindContext, "", b.String()))
	}
	return sections
}

func writeContextItems(b *strings.Builder, items []ContextItem) {
	for _, item := range items {
		fmt.Fprintf(b, "\n<item%s%s%s>\n", attr("type", string(item.Type)), attr("ref", item.Ref), attr("title", item.Title))
		if metadata := strings.TrimSpace(item.MetadataJSON); metadata != "" {
			fmt.Fprintf(b, "<metadata>%s</metadata>\n", metadata)
		}
		if summary := strings.TrimSpace(item.Summary); summary != "" {
			fmt.Fprintf(b, "<summary>\n%s\n</summary>\n", summary)
		}
		b.WriteString("</item>\n")
	}
}

func imagesSection(attachments []Attachment) (promptSection, bool) {
	if len(attachments) == 0 {
		return promptSection{}, false
	}
	var b strings.Builder
	for _, attachment := range attachments {
		fmt.Fprintf(&b, "<image%s%s%s />\n", attr("id", attachment.ID), attr("filename", attachment.Filename), attr("content-type", attachment.ContentType))
	}
	return newPromptSection(promptSectionKindImages, "", b.String()), true
}

func operatorMessageBody(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "(no text supplied — the operator sent context or images only; answer from those)"
	}
	return trimmed
}

// attr renders one escaped XML attribute, or nothing when the value is empty.
func attr(name, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return ` ` + name + `="` + html.EscapeString(trimmed) + `"`
}

func hasContextType(items []ContextItem, target ContextType) bool {
	for _, item := range items {
		if item.Type == target {
			return true
		}
	}
	return false
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

// PreviewResult carries an assembled prompt and which builder produced it.
type PreviewResult struct {
	Prompt  string
	Initial bool
}

// PreviewPrompt assembles the prompt a message would produce, without
// appending the message, spawning a run, or mutating the session.
//
// It calls the same builders Start and Continue call. A client that
// reimplemented the section order or the volatility gradient would produce a
// preview that agrees with nothing, so assembly stays server-owned and this
// method exists purely to expose it.
func (s *Service) PreviewPrompt(ctx context.Context, req ContinueRequest) (PreviewResult, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return PreviewResult{}, err
	}
	session, err := store.LoadSession(strings.TrimSpace(req.SessionID))
	if err != nil {
		return PreviewResult{}, mapStoreError(err)
	}

	initial := session.Status == StatusDraft && strings.TrimSpace(session.RunID) == ""

	refs := req.ContextRefs
	if initial {
		// Mirror Start: the startup brief and any proposal target are added
		// server-side, so a preview that omitted them would understate the
		// prompt by its single largest section.
		refs = refsWithAutoContext(session.Kind, refs, req.AutoContextPolicy, s.startupBriefResolverAvailable())
		if session.ProposalTarget != nil {
			refs = append(refs, ContextRef{Type: session.ProposalTarget.Type, Ref: session.ProposalTarget.Ref})
		}
	}

	contextItems, err := s.resolveMessageContext(ctx, session, refs)
	if err != nil {
		return PreviewResult{}, err
	}

	message := Message{
		Role:          MessageRoleUser,
		Content:       req.Message,
		AttachmentIDs: append([]string(nil), req.AttachmentIDs...),
		Context:       contextItems,
	}
	attachments := sessionAttachmentsByID(session, req.AttachmentIDs)

	if initial {
		return PreviewResult{Prompt: buildInitialPrompt(session, message, attachments), Initial: true}, nil
	}
	return PreviewResult{Prompt: buildContinuationPrompt(message, attachments), Initial: false}, nil
}
