// Package services contains business logic orchestration.
// This file handles search-hub-backed command discovery for completion context.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"agent-inbox/domain"
	"agent-inbox/integrations"
)

const (
	commandDiscoveryLimit   = 8
	commandDiscoveryTimeout = 5 * time.Second
)

// CommandDiscovery finds scenario CLI commands relevant to a chat turn.
type CommandDiscovery interface {
	DiscoverCommands(ctx context.Context, req CommandDiscoveryRequest) (CommandDiscoveryResult, error)
}

// CommandDiscoveryRequest contains the prompt material used for command search.
type CommandDiscoveryRequest struct {
	ChatID   string
	Query    string
	Messages []domain.Message
	Limit    int
}

// CommandDiscoveryResult is the command context that may be injected into a completion.
type CommandDiscoveryResult struct {
	Commands   []DiscoveredCommand
	Diagnostic string
}

// DiscoveredCommand is a normalized Search Hub command hit.
type DiscoveredCommand struct {
	Title         string
	Path          string
	Snippet       string
	ProviderID    string
	ProviderGroup string
	Score         float64
	RerankScore   float64
}

// SearchHubCommandDiscovery shells out to the Search Hub CLI. The service seam
// keeps completion preparation independent of Search Hub transport details.
type SearchHubCommandDiscovery struct {
	Binary string
}

// NewSearchHubCommandDiscovery creates the production Search Hub adapter.
func NewSearchHubCommandDiscovery() *SearchHubCommandDiscovery {
	return &SearchHubCommandDiscovery{Binary: "search-hub"}
}

// DiscoverCommands queries Search Hub for cli-health command records.
func (d *SearchHubCommandDiscovery) DiscoverCommands(ctx context.Context, req CommandDiscoveryRequest) (CommandDiscoveryResult, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		query = buildCommandDiscoveryQuery(req.Messages)
	}
	if query == "" {
		return CommandDiscoveryResult{Diagnostic: "command discovery skipped: no user query text"}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = commandDiscoveryLimit
	}

	binary := d.Binary
	if binary == "" {
		binary = "search-hub"
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandDiscoveryTimeout)
	defer cancel()

	args := []string{"query", query, "--type", "command", "--limit", fmt.Sprintf("%d", limit), "--json"}
	out, err := exec.CommandContext(cmdCtx, binary, args...).CombinedOutput()
	if err != nil {
		return CommandDiscoveryResult{}, fmt.Errorf("search-hub command discovery failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	var payload searchHubQueryResponse
	if err := json.Unmarshal(out, &payload); err != nil {
		return CommandDiscoveryResult{}, fmt.Errorf("decode search-hub command discovery response: %w", err)
	}

	commands := make([]DiscoveredCommand, 0, len(payload.Ranked))
	for _, hit := range payload.Ranked {
		if hit.Type != "" && hit.Type != "command" {
			continue
		}
		commands = append(commands, DiscoveredCommand{
			Title:         hit.Title,
			Path:          hit.Path,
			Snippet:       hit.Snippet,
			ProviderID:    hit.ProviderID,
			ProviderGroup: hit.ProviderGroup,
			Score:         hit.Score,
			RerankScore:   hit.RerankScore,
		})
	}

	if len(commands) == 0 {
		return CommandDiscoveryResult{Diagnostic: "command discovery returned no command records"}, nil
	}
	return CommandDiscoveryResult{Commands: commands}, nil
}

type searchHubQueryResponse struct {
	Ranked []searchHubHit `json:"ranked"`
}

type searchHubHit struct {
	ProviderID    string  `json:"provider_id"`
	ProviderGroup string  `json:"provider_group"`
	Type          string  `json:"type"`
	Title         string  `json:"title"`
	Snippet       string  `json:"snippet"`
	Path          string  `json:"path"`
	Score         float64 `json:"score"`
	RerankScore   float64 `json:"rerank_score"`
}

func (s *CompletionService) maybeInjectCommandContext(ctx context.Context, chatID string, messages []domain.Message, orMessages []integrations.OpenRouterMessage) ([]integrations.OpenRouterMessage, string) {
	if s.commandDiscovery == nil {
		return orMessages, ""
	}

	query := buildCommandDiscoveryQuery(messages)
	result, err := s.commandDiscovery.DiscoverCommands(ctx, CommandDiscoveryRequest{
		ChatID:   chatID,
		Query:    query,
		Messages: messages,
		Limit:    commandDiscoveryLimit,
	})
	if err != nil {
		diagnostic := err.Error()
		log.Printf("[WARN] command discovery degraded: %v", err)
		return orMessages, diagnostic
	}
	if result.Diagnostic != "" {
		log.Printf("[DEBUG] command discovery diagnostic: %s", result.Diagnostic)
	}
	if len(result.Commands) == 0 {
		return orMessages, result.Diagnostic
	}

	contextMessage := buildCommandContextMessage(result.Commands)
	orMessages = append([]integrations.OpenRouterMessage{
		{Role: "system", Content: contextMessage},
	}, orMessages...)
	log.Printf("[DEBUG] Injected %d discovered command records", len(result.Commands))
	return orMessages, result.Diagnostic
}

func buildCommandDiscoveryQuery(messages []domain.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		if text := strings.TrimSpace(messages[i].Content); text != "" {
			return text
		}
	}
	return ""
}

func buildCommandContextMessage(commands []DiscoveredCommand) string {
	var b strings.Builder
	b.WriteString("Relevant Vrooli CLI command capabilities discovered via search-hub/cli-health. ")
	b.WriteString("Use these as command/capability context when planning. Do not treat them as OpenRouter function tools.\n")
	for i, command := range commands {
		if i >= commandDiscoveryLimit {
			break
		}
		title := strings.TrimSpace(command.Title)
		if title == "" {
			title = strings.TrimSpace(command.Path)
		}
		b.WriteString(fmt.Sprintf("- %s", title))
		if path := strings.TrimSpace(command.Path); path != "" && path != title {
			b.WriteString(fmt.Sprintf(" (`%s`)", path))
		}
		if snippet := strings.TrimSpace(command.Snippet); snippet != "" && snippet != title {
			b.WriteString(": ")
			b.WriteString(snippet)
		}
		if command.ProviderID != "" {
			b.WriteString(fmt.Sprintf(" [%s]", command.ProviderID))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
