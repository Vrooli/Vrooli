package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"
)

// AINavigateRequest matches the API request structure.
type AINavigateRequest struct {
	SessionID     string `json:"session_id"`
	Prompt        string `json:"prompt"`
	Model         string `json:"model"`
	MaxSteps      int    `json:"max_steps,omitempty"`
	APIKey        string `json:"api_key,omitempty"`
	NavigatorType string `json:"navigator_type,omitempty"`
}

// AINavigateResponse matches the API response structure.
type AINavigateResponse struct {
	NavigationID  string `json:"navigation_id"`
	Status        string `json:"status"`
	Model         string `json:"model"`
	MaxSteps      int    `json:"max_steps"`
	NavigatorType string `json:"navigator_type"`
}

func printNavigateHelp(cliName string) {
	fmt.Printf("Usage: %s ai navigate [options]\n\n", cliName)
	fmt.Println("Start AI-driven browser navigation.")
	fmt.Println()
	fmt.Println("Required Options:")
	fmt.Println("  --session <id>       Browser session ID to navigate")
	fmt.Println("  --prompt <text>      Natural language navigation instruction")
	fmt.Println()
	fmt.Println("Optional Options:")
	fmt.Println("  --model <model>      Vision model to use (default: gpt-4o)")
	fmt.Println("  --max-steps <n>      Maximum navigation steps (default: 20, max: 100)")
	fmt.Println("  --navigator <type>   Navigator backend: playwright | claude_code")
	fmt.Println("  --api-key <key>      BYOK API key for AI provider")
	fmt.Println("  --json               Output in JSON format")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s ai navigate --session abc123 --prompt \"Click the login button\"\n", cliName)
	fmt.Printf("  %s ai navigate --session abc123 --prompt \"Fill in email\" --navigator playwright\n", cliName)
	fmt.Printf("  %s ai navigate --session abc123 --prompt \"Search for products\" --model claude-sonnet-4\n", cliName)
	fmt.Println()
}

func runNavigate(ctx *appctx.Context, args []string) error {
	var (
		sessionID     string
		prompt        string
		model         = "gpt-4o"
		maxSteps      = 0
		navigatorType string
		apiKey        string
		jsonOutput    = false
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			printNavigateHelp(ctx.Name)
			return nil
		case "--session":
			if i+1 >= len(args) {
				return fmt.Errorf("--session requires a value")
			}
			sessionID = strings.TrimSpace(args[i+1])
			i++
		case "--prompt":
			if i+1 >= len(args) {
				return fmt.Errorf("--prompt requires a value")
			}
			prompt = strings.TrimSpace(args[i+1])
			i++
		case "--model":
			if i+1 >= len(args) {
				return fmt.Errorf("--model requires a value")
			}
			model = strings.TrimSpace(args[i+1])
			i++
		case "--max-steps":
			if i+1 >= len(args) {
				return fmt.Errorf("--max-steps requires a value")
			}
			var n int
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err != nil {
				return fmt.Errorf("--max-steps must be a number")
			}
			maxSteps = n
			i++
		case "--navigator":
			if i+1 >= len(args) {
				return fmt.Errorf("--navigator requires a value")
			}
			navigatorType = strings.TrimSpace(args[i+1])
			i++
		case "--api-key":
			if i+1 >= len(args) {
				return fmt.Errorf("--api-key requires a value")
			}
			apiKey = strings.TrimSpace(args[i+1])
			i++
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			return fmt.Errorf("unexpected argument: %s", args[i])
		}
	}

	// Validate required fields
	if sessionID == "" {
		return fmt.Errorf("--session is required")
	}
	if prompt == "" {
		return fmt.Errorf("--prompt is required")
	}

	// Build request
	request := AINavigateRequest{
		SessionID:     sessionID,
		Prompt:        prompt,
		Model:         model,
		MaxSteps:      maxSteps,
		NavigatorType: navigatorType,
		APIKey:        apiKey,
	}

	// Call the API using our custom Do function that sets X-Client-Source header
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	statusCode, body, err := api.Do(ctx, http.MethodPost, "/ai-navigate", nil, payload, nil)
	if err != nil {
		return fmt.Errorf("failed to start navigation: %w", err)
	}
	if statusCode >= 400 {
		return fmt.Errorf("failed to start navigation: api error (%d): %s", statusCode, string(body))
	}

	var response AINavigateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(response, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	fmt.Println("AI Navigation Started")
	fmt.Printf("Navigation ID: %s\n", response.NavigationID)
	fmt.Printf("Status: %s\n", response.Status)
	fmt.Printf("Model: %s\n", response.Model)
	fmt.Printf("Max Steps: %d\n", response.MaxSteps)
	fmt.Printf("Navigator: %s\n", response.NavigatorType)
	fmt.Println()
	fmt.Printf("Use '%s ai navigate-status %s' to check progress.\n", ctx.Name, response.NavigationID)

	return nil
}
