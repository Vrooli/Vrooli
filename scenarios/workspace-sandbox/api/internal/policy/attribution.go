package policy

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/types"
)

// DefaultAttributionPolicy implements AttributionPolicy using configuration.
type DefaultAttributionPolicy struct {
	config   config.PolicyConfig
	template *template.Template
}

// NewDefaultAttributionPolicy creates an attribution policy from config.
func NewDefaultAttributionPolicy(cfg config.PolicyConfig) (*DefaultAttributionPolicy, error) {
	tmpl, err := template.New("commit").Parse(cfg.CommitMessageTemplate)
	if err != nil {
		return nil, fmt.Errorf("invalid commit message template: %w", err)
	}

	return &DefaultAttributionPolicy{
		config:   cfg,
		template: tmpl,
	}, nil
}

// GetCommitAuthor returns the author string for the commit.
func (p *DefaultAttributionPolicy) GetCommitAuthor(ctx context.Context, sandbox *types.Sandbox, actor string) string {
	switch p.config.CommitAuthorMode {
	case "reviewer":
		// Use the reviewing user as author
		if actor != "" {
			return fmt.Sprintf("%s <noreply@workspace-sandbox.local>", actor)
		}
		return "Workspace Sandbox <noreply@workspace-sandbox.local>"

	case "coauthored":
		// Primary author is the sandbox owner (agent), reviewer is co-author
		if sandbox.Owner != "" {
			return fmt.Sprintf("%s <noreply@workspace-sandbox.local>", sandbox.Owner)
		}
		return "Workspace Sandbox <noreply@workspace-sandbox.local>"

	default: // "agent"
		// Use the sandbox owner (typically an agent) as author
		if sandbox.Owner != "" {
			return fmt.Sprintf("%s <noreply@workspace-sandbox.local>", sandbox.Owner)
		}
		return "Workspace Sandbox <noreply@workspace-sandbox.local>"
	}
}

// commitTemplateData provides data for commit message templates.
type commitTemplateData struct {
	SandboxID string
	FileCount int
	Actor     string
	Owner     string
	ScopePath string
}

// GetCommitMessage returns the formatted commit message.
func (p *DefaultAttributionPolicy) GetCommitMessage(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange, userMessage string) string {
	// If user provided a message, use it directly
	if userMessage != "" {
		return userMessage
	}

	// Use template to generate message
	data := commitTemplateData{
		SandboxID: sandbox.ID.String(),
		FileCount: len(changes),
		Actor:     sandbox.Owner,
		Owner:     sandbox.Owner,
		ScopePath: sandbox.ScopePath,
	}

	var buf bytes.Buffer
	if err := p.template.Execute(&buf, data); err != nil {
		// Fallback to simple message
		return fmt.Sprintf("Apply sandbox changes (%d files)", len(changes))
	}

	return buf.String()
}

// GetCoAuthors returns any co-author lines to append to the commit message.
func (p *DefaultAttributionPolicy) GetCoAuthors(ctx context.Context, sandbox *types.Sandbox, actor string) []string {
	if p.config.CommitAuthorMode != "coauthored" {
		return nil
	}

	// In coauthored mode, add the reviewer as co-author
	if actor != "" && actor != sandbox.Owner {
		return []string{
			fmt.Sprintf("Co-authored-by: %s <noreply@workspace-sandbox.local>", actor),
		}
	}

	return nil
}

// FormatCommitMessage combines the message with co-author lines.
func FormatCommitMessage(message string, coAuthors []string) string {
	if len(coAuthors) == 0 {
		return message
	}

	return message + "\n\n" + strings.Join(coAuthors, "\n")
}

// Verify interface is implemented.
var _ AttributionPolicy = (*DefaultAttributionPolicy)(nil)
