package workflows

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

type lintResult struct {
	File   string          `json:"file"`
	Error  string          `json:"error,omitempty"`
	Status string          `json:"status,omitempty"`
	Data   map[string]any  `json:"data,omitempty"`
	Raw    json.RawMessage `json:"raw,omitempty"`
	Valid  *bool           `json:"valid,omitempty"`
}

func runLint(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: browser-automation-studio workflow lint <files...> [--strict] [--json]")
	}

	strict := false
	jsonOutput := false
	files := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			strict = true
		case "--json":
			jsonOutput = true
		case "-h", "--help":
			fmt.Println("Usage: browser-automation-studio workflow lint <files...> [--strict] [--json]")
			return nil
		case "--":
			files = append(files, args[i+1:]...)
			i = len(args)
		default:
			files = append(files, args[i])
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("provide one or more workflow JSON files")
	}

	lintFailed := false
	results := []lintResult{}

	for _, filePath := range files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			lintFailed = true
			if jsonOutput {
				results = append(results, lintResult{File: filePath, Error: "file not found"})
			} else {
				fmt.Printf("ERROR %s\n  File not found\n", filePath)
			}
			continue
		}

		var workflow any
		if err := json.Unmarshal(data, &workflow); err != nil {
			lintFailed = true
			if jsonOutput {
				results = append(results, lintResult{File: filePath, Error: "invalid JSON"})
			} else {
				fmt.Printf("ERROR %s\n  Invalid JSON\n", filePath)
			}
			continue
		}

		payload := map[string]any{
			"workflow": workflow,
			"strict":   strict,
		}

		body, err := ctx.Core.APIClient.Request("POST", ctx.APIPath("/workflows/validate"), nil, payload)
		if err != nil {
			lintFailed = true
			if jsonOutput {
				results = append(results, lintResult{File: filePath, Error: err.Error()})
			} else {
				fmt.Printf("ERROR %s\n  %s\n", filePath, err.Error())
			}
			continue
		}

		var parsed workflowValidationResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			lintFailed = true
			if jsonOutput {
				results = append(results, lintResult{File: filePath, Error: "invalid response"})
			} else {
				fmt.Printf("ERROR %s\n  invalid response\n", filePath)
			}
			continue
		}

		if jsonOutput {
			results = append(results, lintResult{File: filePath, Raw: body, Valid: &parsed.Valid})
		} else {
			if parsed.Valid {
				fmt.Printf("OK %s (%d nodes, %d edges)\n", filePath, parsed.Stats.NodeCount, parsed.Stats.EdgeCount)
			} else {
				fmt.Printf("ERROR %s (%d nodes, %d edges)\n", filePath, parsed.Stats.NodeCount, parsed.Stats.EdgeCount)
			}
			for _, issue := range parsed.Errors {
				fmt.Printf("  [%s] %s", fallback(issue.Code, "error"), issue.Message)
				if issue.NodeID != "" {
					fmt.Printf(" (node: %s)", issue.NodeID)
				}
				if issue.Pointer != "" {
					fmt.Printf(" @ %s", issue.Pointer)
				}
				fmt.Println()
			}
			for _, issue := range parsed.Warnings {
				fmt.Printf("  [warn:%s] %s", fallback(issue.Code, "warning"), issue.Message)
				if issue.NodeID != "" {
					fmt.Printf(" (node: %s)", issue.NodeID)
				}
				if issue.Pointer != "" {
					fmt.Printf(" @ %s", issue.Pointer)
				}
				fmt.Println()
			}
		}

		if !parsed.Valid {
			lintFailed = true
		}
	}

	if jsonOutput {
		payload := map[string]any{"results": results}
		output, _ := json.MarshalIndent(payload, "", "  ")
		cliutil.PrintJSON(output)
	}

	if lintFailed {
		return fmt.Errorf("workflow lint failed")
	}
	return nil
}

func fallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
