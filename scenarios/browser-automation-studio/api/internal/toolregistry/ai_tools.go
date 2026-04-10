// Package toolregistry provides tool definitions for AI-powered capabilities.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// AIToolProvider provides AI-powered analysis and navigation tools.
type AIToolProvider struct{}

// NewAIToolProvider creates a new AIToolProvider.
func NewAIToolProvider() *AIToolProvider {
	return &AIToolProvider{}
}

// Name returns the provider name.
func (p *AIToolProvider) Name() string {
	return "bas-ai"
}

// Categories returns the categories for AI tools.
func (p *AIToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		NewCategory("ai_capabilities", "AI Capabilities", "AI-powered tools for intelligent browser automation", "brain", 6),
	}
}

// Tools returns the AI tool definitions.
func (p *AIToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.aiAnalyzeElementsTool(),
		p.aiNavigateTool(),
		p.getDOMTreeTool(),
	}
}

// aiAnalyzeElementsTool defines the ai_analyze_elements tool.
func (p *AIToolProvider) aiAnalyzeElementsTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"url":   StringParamWithFormat("URL of the page to analyze", "uri"),
			"query": StringParam("Natural language description of elements to find (e.g., 'login button', 'email input field')"),
		},
		[]string{"url", "query"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     120, // AI tools need longer timeout
		RateLimitPerMinute: 5,   // Rate limit due to cost
		CostEstimate:       "high",
		Idempotent:         true,
		Tags:               []string{"ai", "analysis", "elements"},
	})

	return NewToolDefinition(
		"ai_analyze_elements",
		"Use AI to analyze a web page and find elements matching a natural language description. Returns element selectors and information.",
		"ai_capabilities",
		params,
		meta,
	)
}

// aiNavigateTool defines the ai_navigate tool (async, long-running).
func (p *AIToolProvider) aiNavigateTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"url":  StringParamWithFormat("Starting URL for the navigation", "uri"),
			"goal": StringParam("Natural language description of the navigation goal (e.g., 'login with username test@example.com and password secret123')"),
		},
		[]string{"url", "goal"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   true, // AI navigation should require approval
		TimeoutSeconds:     180,  // AI navigation can take a while
		RateLimitPerMinute: 3,
		CostEstimate:       "high",
		LongRunning:        true,
		Idempotent:         false,
		ModifiesState:      true,
		Tags:               []string{"ai", "navigation", "autonomous"},
	})

	meta.AsyncBehavior = NewAsyncBehavior(AsyncBehaviorOpts{
		StatusTool:             "get_ai_navigation_status",
		OperationIdField:       "navigation_id",
		StatusToolIdParam:      "navigation_id",
		PollIntervalSeconds:    3,
		MaxPollDurationSeconds: 600, // 10 minutes
		StatusField:            "status",
		SuccessValues:          []string{"completed", "success"},
		FailureValues:          []string{"failed", "error", "aborted"},
		PendingValues:          []string{"running", "navigating", "analyzing"},
		ErrorField:             "error_message",
		ResultField:            "result",
	})

	return NewToolDefinition(
		"ai_navigate",
		"Use AI to autonomously navigate a website to achieve a goal. The AI will analyze pages, identify relevant elements, and perform actions to reach the specified objective.",
		"ai_capabilities",
		params,
		meta,
	)
}

// getDOMTreeTool defines the get_dom_tree tool.
func (p *AIToolProvider) getDOMTreeTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"url":           StringParamWithFormat("URL of the page to extract DOM from", "uri"),
			"selector":      StringParam("CSS selector to limit extraction to a specific element (optional)"),
			"max_depth":     IntParamWithDefault("Maximum depth of DOM tree to extract", 10),
			"include_text":  BoolParamWithDefault("Include text content in the response", true),
			"include_attrs": BoolParamWithDefault("Include element attributes in the response", true),
		},
		[]string{"url"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     60,
		RateLimitPerMinute: 10,
		CostEstimate:       "medium",
		Idempotent:         true,
		Tags:               []string{"dom", "extraction", "structure"},
	})

	return NewToolDefinition(
		"get_dom_tree",
		"Extract the DOM tree structure from a web page. Useful for understanding page structure and finding elements programmatically.",
		"ai_capabilities",
		params,
		meta,
	)
}
