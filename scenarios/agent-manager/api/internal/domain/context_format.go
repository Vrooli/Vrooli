// Package domain defines the core domain entities for agent-manager.
//
// This file contains context formatting logic for building agent prompts
// with structured context attachments.

package domain

import (
	"fmt"
	"strings"
)

// FormatContextForPrompt formats context attachments into XML-like blocks
// suitable for appending to an agent prompt.
//
// Example output:
//
//	<context key="error-logs" type="file" path="/var/log/app.log">
//	[file content or placeholder]
//	</context>
func FormatContextForPrompt(attachments []ContextAttachment) string {
	if len(attachments) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n\n")

	for i, att := range attachments {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(formatSingleContext(att))
	}

	return builder.String()
}

func formatSingleContext(att ContextAttachment) string {
	var builder strings.Builder

	// Build opening tag with attributes
	builder.WriteString("<context")

	if att.Key != "" {
		builder.WriteString(fmt.Sprintf(` key="%s"`, escapeXMLAttr(att.Key)))
	}

	builder.WriteString(fmt.Sprintf(` type="%s"`, att.Type))

	if len(att.Tags) > 0 {
		builder.WriteString(fmt.Sprintf(` tags="%s"`, escapeXMLAttr(strings.Join(att.Tags, ","))))
	}

	if att.Label != "" {
		builder.WriteString(fmt.Sprintf(` label="%s"`, escapeXMLAttr(att.Label)))
	}

	// Type-specific attributes
	switch att.Type {
	case "file":
		if att.Path != "" {
			builder.WriteString(fmt.Sprintf(` path="%s"`, escapeXMLAttr(att.Path)))
		}
	case "link":
		if att.URL != "" {
			builder.WriteString(fmt.Sprintf(` url="%s"`, escapeXMLAttr(att.URL)))
		}
	}

	builder.WriteString(">\n")

	// Content
	content := getContextContent(att)
	if content != "" {
		builder.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			builder.WriteString("\n")
		}
	}

	builder.WriteString("</context>")

	return builder.String()
}

func getContextContent(att ContextAttachment) string {
	switch att.Type {
	case "file":
		// For files, content may be pre-loaded or we just provide the path hint
		if att.Content != "" {
			return att.Content
		}
		return fmt.Sprintf("[File: %s - content to be loaded by agent]", att.Path)
	case "link":
		if att.Content != "" {
			return att.Content // Description or fetched content
		}
		return fmt.Sprintf("[Link: %s]", att.URL)
	case "note":
		return att.Content
	default:
		return att.Content
	}
}

func escapeXMLAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// BuildPromptWithContext combines a base prompt with formatted context attachments.
// This is the primary function for constructing the final prompt sent to agents.
func BuildPromptWithContext(basePrompt string, attachments []ContextAttachment) string {
	if len(attachments) == 0 {
		return basePrompt
	}

	contextSection := FormatContextForPrompt(attachments)
	return basePrompt + contextSection
}
