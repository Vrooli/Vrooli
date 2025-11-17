package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

type fileResult struct {
	File   string                    `json:"file"`
	Result *workflowvalidator.Result `json:"result,omitempty"`
	Error  string                    `json:"error,omitempty"`
}

func main() {
	strict := flag.Bool("strict", false, "Treat warnings as errors")
	jsonOutput := flag.Bool("json", false, "Emit machine-readable JSON")
	selectorRoot := flag.String("selector-root", "", "Override selector root directory")
	flag.Parse()

	inputs := flag.Args()
	if len(inputs) == 0 {
		fmt.Fprintln(os.Stderr, "workflow-lint: one or more workflow files are required")
		os.Exit(1)
	}

	files := expandInputs(inputs)
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "workflow-lint: no files matched the provided inputs")
		os.Exit(1)
	}

	validator, err := workflowvalidator.NewValidator()
	if err != nil {
		fmt.Fprintf(os.Stderr, "workflow-lint: failed to initialize validator: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	options := workflowvalidator.Options{
		Strict:       *strict,
		SelectorRoot: *selectorRoot,
	}
	var results []fileResult
	exitCode := 0

	for _, file := range files {
		result := fileResult{File: file}
		payload, readErr := os.ReadFile(file)
		if readErr != nil {
			result.Error = readErr.Error()
			exitCode = 1
			results = append(results, result)
			continue
		}

		var definition map[string]any
		if err := json.Unmarshal(payload, &definition); err != nil {
			result.Error = fmt.Sprintf("invalid JSON: %v", err)
			exitCode = 1
			results = append(results, result)
			continue
		}

		validation, err := validator.Validate(ctx, definition, options)
		if err != nil {
			result.Error = err.Error()
			exitCode = 1
			results = append(results, result)
			continue
		}

		workflowvalidator.SortIssues(validation.Errors)
		workflowvalidator.SortIssues(validation.Warnings)
		result.Result = validation
		if !validation.Valid {
			exitCode = 1
		}
		results = append(results, result)
	}

	if *jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"results": results}); err != nil {
			fmt.Fprintf(os.Stderr, "workflow-lint: failed to encode JSON: %v\n", err)
			os.Exit(1)
		}
		os.Exit(exitCode)
	}

	for _, res := range results {
		printHuman(res)
	}

	os.Exit(exitCode)
}

func expandInputs(inputs []string) []string {
	seen := map[string]struct{}{}
	var files []string
	for _, input := range inputs {
		matches, err := filepath.Glob(input)
		if err == nil && len(matches) > 0 {
			sort.Strings(matches)
			for _, match := range matches {
				if info, statErr := os.Stat(match); statErr == nil && !info.IsDir() {
					if _, exists := seen[match]; !exists {
						seen[match] = struct{}{}
						files = append(files, match)
					}
				}
			}
			continue
		}

		if info, err := os.Stat(input); err == nil && !info.IsDir() {
			if _, exists := seen[input]; !exists {
				seen[input] = struct{}{}
				files = append(files, input)
			}
		}
	}
	sort.Strings(files)
	return files
}

func printHuman(res fileResult) {
	if res.Error != "" {
		fmt.Printf("✗ %s\n  %s\n", res.File, res.Error)
		return
	}

	stats := res.Result.Stats
	if res.Result.Valid {
		fmt.Printf("✓ %s (%d nodes, %d edges)\n", res.File, stats.NodeCount, stats.EdgeCount)
	} else {
		fmt.Printf("✗ %s (%d nodes, %d edges)\n", res.File, stats.NodeCount, stats.EdgeCount)
	}

	if len(res.Result.Errors) > 0 {
		for _, issue := range res.Result.Errors {
			fmt.Printf("  [%s] %s", issue.Code, issue.Message)
			if issue.NodeID != "" {
				fmt.Printf(" (node: %s)", issue.NodeID)
			}
			if issue.Pointer != "" {
				fmt.Printf(" @ %s", issue.Pointer)
			}
			fmt.Println()
		}
	}

	if len(res.Result.Warnings) > 0 {
		for _, issue := range res.Result.Warnings {
			fmt.Printf("  [warn:%s] %s", issue.Code, issue.Message)
			if issue.NodeID != "" {
				fmt.Printf(" (node: %s)", issue.NodeID)
			}
			if issue.Pointer != "" {
				fmt.Printf(" @ %s", issue.Pointer)
			}
			fmt.Println()
		}
	}
}
