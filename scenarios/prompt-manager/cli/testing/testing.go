// Package testing provides CLI commands for skill testing with Ollama.
//
// DOC: docs/reference/cli-commands.md#testing
package testing

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// TestRequest is the request body for testing a skill
type TestRequest struct {
	Role        string            `json:"role"`
	Variables   map[string]string `json:"variables,omitempty"`
	MaxTokens   *int              `json:"maxTokens,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
}

// TestResponse is the response from testing a skill
type TestResponse struct {
	TestID       string    `json:"testId"`
	Role         string    `json:"role"`
	Response     string    `json:"response"`
	ResponseTime float64   `json:"responseTime"`
	TokenCount   int       `json:"tokenCount"`
	TestedAt     time.Time `json:"testedAt"`
}

// TestResult represents a test result in history
type TestResult struct {
	ID           string    `json:"id"`
	SkillID      string    `json:"skillId"`
	Role         string    `json:"role"`
	InputVars    *string   `json:"inputVariables,omitempty"`
	Response     *string   `json:"response,omitempty"`
	ResponseTime *float64  `json:"responseTime,omitempty"`
	TokenCount   *int      `json:"tokenCount,omitempty"`
	Rating       *int      `json:"rating,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	TestedAt     time.Time `json:"testedAt"`
}

// Commands returns the testing command groups using noun-verb pattern.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Testing",
		Commands: []cliapp.Command{
			{
				Name:        "test",
				Aliases:     []string{"tests"},
				NeedsAPI:    true,
				Description: "Test skills with Ollama (run|history)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
		},
	}
}

// route dispatches to the appropriate subcommand.
func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "run", "execute":
		return cmdRun(ctx, subArgs)
	case "history", "results":
		return cmdHistory(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager test <subcommand> [args]

Subcommands:
  run, execute <skill-id>    Test a skill with Ollama
  history, results <skill-id> View test history for a skill`
}

func cmdRun(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	role := fs.String("role", "chat.small", "Ollama role to use")
	vars := fs.String("vars", "", "Variables in key=value,key2=value2 format")
	maxTokens := fs.Int("max-tokens", 1000, "Maximum tokens in response")
	temperature := fs.Float64("temperature", 0.7, "Temperature for generation")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: test run <skill-id> [--role=chat.small] [--vars=key=value] [--max-tokens=1000] [--temperature=0.7]")
	}
	skillID := fs.Arg(0)

	// Parse variables
	variables := make(map[string]string)
	if *vars != "" {
		pairs := strings.Split(*vars, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				variables[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	req := TestRequest{
		Role:        *role,
		Variables:   variables,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	fmt.Printf("Testing skill %s with %s...\n", skillID, *role)

	var resp TestResponse
	if err := ctx.Post(fmt.Sprintf("/skills/%s/test", skillID), req, &resp); err != nil {
		return fmt.Errorf("failed to test skill: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("\nTest completed in %.0fms (%d tokens)\n", resp.ResponseTime, resp.TokenCount)
	fmt.Printf("\nResponse:\n%s\n", resp.Response)
	return nil
}

func cmdHistory(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: test history <skill-id>")
	}
	skillID := fs.Arg(0)

	var results []TestResult
	if err := ctx.Get(fmt.Sprintf("/skills/%s/test-history", skillID), &results); err != nil {
		return fmt.Errorf("failed to get test history: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	if len(results) == 0 {
		fmt.Println("No test history found")
		return nil
	}

	fmt.Printf("Test History for %s:\n", skillID)
	for _, r := range results {
		responseTime := ""
		if r.ResponseTime != nil {
			responseTime = fmt.Sprintf("%.0fms", *r.ResponseTime)
		}
		tokens := ""
		if r.TokenCount != nil {
			tokens = fmt.Sprintf("%d tokens", *r.TokenCount)
		}
		fmt.Printf("  %s - %s - %s %s [%s]\n",
			r.TestedAt.Format("2006-01-02 15:04"),
			r.Role,
			responseTime,
			tokens,
			r.ID[:8])
	}
	return nil
}
