package routing

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	"search-hub/internal/aigateway"
)

const searchHubRerankPrivacySource = "search_hub.llm_reranking.query_and_candidate_metadata"

type scenarioURLResolver = aigateway.URLResolver

func newGatewayLLMReranker(resolve scenarioURLResolver, client connect.HTTPClient) aisearch.Reranker {
	return aisearch.NewLLMRerankerWithRunner("", gatewayRerankGenerateRunner(resolve, client))
}

// gatewayRerankGenerateRunner adapts ai-go/search's legacy generation seam to
// the public typed Gateway API. The role policy owns the output cap; the legacy
// CLI-shaped max-token argument is parsed for validation but is not allowed to
// override that policy.
func gatewayRerankGenerateRunner(resolve scenarioURLResolver, client connect.HTTPClient) aisearch.GenerateRunner {
	generate := aigateway.NewTextGenerator(resolve, client, searchHubRerankPrivacySource)
	return func(ctx context.Context, args []string, prompt []byte) ([]byte, error) {
		role, err := parseRerankGatewayArgs(args)
		if err != nil {
			return nil, err
		}
		return generate(ctx, role, string(prompt), 0)
	}
}

func parseRerankGatewayArgs(args []string) (string, error) {
	role := ""
	maxTokensSeen := false
	// ai-go/search's compatibility seam prefixes its former executable name.
	// This typed adapter ignores that argv slot and accepts only the task
	// options it needs; it never shells the named binary.
	start := 0
	if len(args) > 0 {
		start = 1
	}
	for i := start; i < len(args); i++ {
		switch args[i] {
		case "gateway", "generate", "--json", "--prompt-stdin":
			continue
		case "--temperature":
			if i+1 >= len(args) || args[i+1] != "0" {
				return "", errors.New("rerank adapter only supports deterministic temperature 0")
			}
			i++
		case "--max-tokens":
			if i+1 >= len(args) {
				return "", errors.New("rerank adapter is missing its max-token value")
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return "", fmt.Errorf("invalid rerank max-token value: %w", err)
			}
			if value <= 0 {
				return "", errors.New("rerank max-token value must be positive")
			}
			maxTokensSeen = true
			i++
		case "--role":
			if i+1 >= len(args) {
				return "", errors.New("rerank adapter is missing its role")
			}
			role = strings.TrimSpace(args[i+1])
			if role == "" {
				return "", errors.New("rerank role is empty")
			}
			i++
		default:
			return "", fmt.Errorf("unsupported rerank option %q", args[i])
		}
	}
	if role == "" || !maxTokensSeen {
		return "", errors.New("rerank role and max-token bound are required")
	}
	return role, nil
}
