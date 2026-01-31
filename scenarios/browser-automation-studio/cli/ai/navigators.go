package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"browser-automation-studio/cli/internal/appctx"
)

// NavigatorInfo matches the API response structure.
type NavigatorInfo struct {
	Type              string           `json:"type"`
	Available         bool             `json:"available"`
	Description       string           `json:"description"`
	CreditPolicy      CreditPolicyInfo `json:"credit_policy"`
	AllowedSources    []string         `json:"allowed_sources"`
	UnavailableReason string           `json:"unavailable_reason,omitempty"`
}

// CreditPolicyInfo matches the API response structure.
type CreditPolicyInfo struct {
	RequiresCredits  bool     `json:"requires_credits"`
	CreditsPerStep   int      `json:"credits_per_step"`
	BypassConditions []string `json:"bypass_conditions"`
}

// NavigatorsResponse matches the API response structure.
type NavigatorsResponse struct {
	Navigators []NavigatorInfo `json:"navigators"`
	Default    string          `json:"default"`
}

func printNavigatorsHelp(cliName string) {
	fmt.Printf("Usage: %s ai navigators [options]\n\n", cliName)
	fmt.Println("List available AI navigation backends.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --json    Output in JSON format")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s ai navigators\n", cliName)
	fmt.Printf("  %s ai navigators --json\n", cliName)
	fmt.Println()
}

func runNavigators(ctx *appctx.Context, args []string) error {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			printNavigatorsHelp(ctx.Name)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(arg, "--") {
				return fmt.Errorf("unknown option: %s", arg)
			}
			return fmt.Errorf("unexpected argument: %s", arg)
		}
	}

	// Call the API
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/ai-navigate/navigators"), nil)
	if err != nil {
		return fmt.Errorf("failed to get navigators: %w", err)
	}

	var response NavigatorsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(response, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	fmt.Println("Available AI Navigation Backends:")
	fmt.Println()

	for _, nav := range response.Navigators {
		availabilityStr := "available"
		if !nav.Available {
			availabilityStr = "unavailable"
		}

		fmt.Printf("  %-14s %s (%s)\n", nav.Type, nav.Description, availabilityStr)

		// Credit policy
		if nav.CreditPolicy.RequiresCredits {
			bypassInfo := ""
			if len(nav.CreditPolicy.BypassConditions) > 0 {
				bypassInfo = fmt.Sprintf(" (bypassed with %s)", strings.Join(nav.CreditPolicy.BypassConditions, " or "))
			}
			fmt.Printf("                Credits: %d/step%s\n", nav.CreditPolicy.CreditsPerStep, bypassInfo)
		} else {
			bypassInfo := ""
			if len(nav.CreditPolicy.BypassConditions) > 0 {
				bypassInfo = fmt.Sprintf(" (%s)", strings.Join(nav.CreditPolicy.BypassConditions, ", "))
			}
			fmt.Printf("                Credits: none%s\n", bypassInfo)
		}

		// Unavailability reason
		if !nav.Available && nav.UnavailableReason != "" {
			fmt.Printf("                Reason: %s\n", nav.UnavailableReason)
		}

		fmt.Println()
	}

	if response.Default != "" {
		fmt.Printf("Default navigator: %s\n", response.Default)
	}

	return nil
}
