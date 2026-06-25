package agentsessions

import (
	"fmt"
	"strings"
)

func buildInitialPrompt(session Session, message Message, attachments []Attachment) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are running a Swarm Manager %s agent session.\n\n", session.Kind)
	fmt.Fprintf(&b, "Use the Prompt Manager skill `%s` as your operating guide.\n", session.SkillID)
	fmt.Fprintf(&b, "Session ID: %s\n", session.ID)
	if hasContextType(message.Context, ContextStartupBrief) {
		b.WriteString("Startup brief context is attached below. For broad status, planning, or authoring questions, answer from this brief first and run at most one targeted refresh/drill-down command before the first useful answer.\n")
	}
	if hasContextType(message.Context, ContextOperationsBriefing) {
		b.WriteString("Operations briefing context is attached below. Answer broad status questions from it before doing exploratory reads.\n")
	}
	writeMessageContext(&b, message.Context)
	writeMessageAttachments(&b, attachments)
	b.WriteString("\nOperator message:\n")
	writeOperatorMessage(&b, message.Content)
	return b.String()
}

func hasContextType(items []ContextItem, target ContextType) bool {
	for _, item := range items {
		if item.Type == target {
			return true
		}
	}
	return false
}

func buildContinuationPrompt(message Message, attachments []Attachment) string {
	if len(message.Context) == 0 && len(attachments) == 0 {
		return strings.TrimSpace(message.Content)
	}
	var b strings.Builder
	writeMessageContext(&b, message.Context)
	writeMessageAttachments(&b, attachments)
	b.WriteString("\nOperator message:\n")
	writeOperatorMessage(&b, message.Content)
	return b.String()
}

func writeMessageContext(b *strings.Builder, contextItems []ContextItem) {
	if len(contextItems) == 0 {
		return
	}
	b.WriteString("\nAttached context:\n")
	for i, item := range contextItems {
		fmt.Fprintf(b, "%d. [%s] %s (%s)\n", i+1, item.Type, item.Title, item.Ref)
		if strings.TrimSpace(item.MetadataJSON) != "" {
			fmt.Fprintf(b, "   Metadata: %s\n", strings.TrimSpace(item.MetadataJSON))
		}
		if strings.TrimSpace(item.Summary) != "" {
			fmt.Fprintf(b, "   Summary: %s\n", strings.TrimSpace(item.Summary))
		}
	}
}

func writeMessageAttachments(b *strings.Builder, attachments []Attachment) {
	if len(attachments) == 0 {
		return
	}
	b.WriteString("\nAttached images:\n")
	for _, attachment := range attachments {
		fmt.Fprintf(b, "- %s: %s (%s)\n", attachment.ID, attachment.Filename, attachment.ContentType)
	}
}

func writeOperatorMessage(b *strings.Builder, content string) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		b.WriteString("(no text supplied)")
		return
	}
	b.WriteString(trimmed)
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
