// Package domain defines the core domain entities for agent-manager.
//
// This file contains context formatting logic for building agent prompts
// with structured context attachments. The formatting is designed to be
// optimal for AI consumption with clear hierarchy and metadata.

package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatContextForPrompt formats context attachments into XML-like blocks
// suitable for appending to an agent prompt. Attachments are sorted by
// priority (high first) and formatted with summaries and content hints.
//
// Example output:
//
//	<context key="error-info" type="note" priority="high" format="text" summary="TLS validation failed on port 443">
//	Failed Step: verify_origin
//	Error: Cannot negotiate ALPN protocol
//	</context>
func FormatContextForPrompt(attachments []ContextAttachment) string {
	if len(attachments) == 0 {
		return ""
	}

	// Sort by priority: high > medium > low > unset
	sorted := sortByPriority(attachments)

	var builder strings.Builder
	builder.WriteString("\n\n")

	for i, att := range sorted {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(formatSingleContext(att))
	}

	return builder.String()
}

// sortByPriority returns attachments sorted by priority (high first).
func sortByPriority(attachments []ContextAttachment) []ContextAttachment {
	priorityOrder := map[string]int{
		"high":   0,
		"medium": 1,
		"low":    2,
		"":       3,
	}

	// Make a copy to avoid modifying the original
	sorted := make([]ContextAttachment, len(attachments))
	copy(sorted, attachments)

	// Simple insertion sort (stable, good for small lists)
	for i := 1; i < len(sorted); i++ {
		j := i
		for j > 0 && priorityOrder[sorted[j].Priority] < priorityOrder[sorted[j-1].Priority] {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			j--
		}
	}

	return sorted
}

func formatSingleContext(att ContextAttachment) string {
	var builder strings.Builder

	// Build opening tag with attributes
	builder.WriteString("<context")

	if att.Key != "" {
		builder.WriteString(fmt.Sprintf(` key="%s"`, escapeXMLAttr(att.Key)))
	}

	builder.WriteString(fmt.Sprintf(` type="%s"`, att.Type))

	// Include new metadata fields
	if att.Priority != "" {
		builder.WriteString(fmt.Sprintf(` priority="%s"`, escapeXMLAttr(att.Priority)))
	}

	if att.Format != "" {
		builder.WriteString(fmt.Sprintf(` format="%s"`, escapeXMLAttr(att.Format)))
	}

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

	// Add summary as attribute if present (one-line TL;DR)
	if att.Summary != "" {
		builder.WriteString(fmt.Sprintf(` summary="%s"`, escapeXMLAttr(att.Summary)))
	}

	builder.WriteString(">\n")

	// Content with format-aware rendering
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
			return formatContentByType(att.Content, att.Format)
		}
		return fmt.Sprintf("[File: %s - content to be loaded by agent]", att.Path)
	case "link":
		if att.Content != "" {
			return formatContentByType(att.Content, att.Format)
		}
		return fmt.Sprintf("[Link: %s]", att.URL)
	case "note":
		return formatContentByType(att.Content, att.Format)
	default:
		return formatContentByType(att.Content, att.Format)
	}
}

// formatContentByType applies format-specific processing to content.
func formatContentByType(content, format string) string {
	if content == "" {
		return ""
	}

	switch format {
	case "json":
		return formatJSON(content)
	case "yaml":
		// YAML is already human-readable, just ensure consistent indentation
		return content
	case "log":
		// Log content often has pipe-separated entries - split for readability
		return formatLogContent(content)
	case "markdown", "text", "":
		return content
	default:
		return content
	}
}

// formatJSON attempts to pretty-print JSON content.
// If the content is not valid JSON, returns it as-is.
func formatJSON(content string) string {
	content = strings.TrimSpace(content)

	// Check if it looks like JSON
	if !strings.HasPrefix(content, "{") && !strings.HasPrefix(content, "[") {
		return content
	}

	// Try to parse and re-marshal with indentation
	var data interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return content // Not valid JSON, return as-is
	}

	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return content
	}

	return string(formatted)
}

// formatLogContent splits pipe-separated log entries into readable lines.
func formatLogContent(content string) string {
	// Common pattern: log entries separated by " | "
	if strings.Contains(content, " | ") {
		parts := strings.Split(content, " | ")
		var builder strings.Builder
		for i, part := range parts {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(strings.TrimSpace(part))
		}
		return builder.String()
	}
	return content
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
//
// The base prompt should contain the core task instructions. Context attachments
// are appended after the prompt, sorted by priority, with proper formatting.
func BuildPromptWithContext(basePrompt string, attachments []ContextAttachment) string {
	if len(attachments) == 0 {
		return basePrompt
	}

	contextSection := FormatContextForPrompt(attachments)
	return basePrompt + contextSection
}
