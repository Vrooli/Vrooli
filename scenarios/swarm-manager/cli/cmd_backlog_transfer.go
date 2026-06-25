package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogExport(args []string) error {
	fs := flag.NewFlagSet("backlog export", flag.ContinueOnError)
	kinds := fs.String("kinds", "", "Comma-separated kinds to include (default: all)")
	statuses := fs.String("status", "", "Comma-separated statuses to include (default: non-archived)")
	names := fs.String("names", "", "Comma-separated kind/name pairs for specific items")
	priorityMax := fs.Int("priority-max", 0, "Only items with priority <= this value")
	tags := fs.String("tags", "", "Comma-separated tags (any match)")
	noPRD := fs.Bool("no-prd", false, "Exclude PRD content (smaller file)")
	outPath := fs.String("out", "", "Output file path (default: backlog-export-YYYY-MM-DD.md)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	payload := map[string]any{}
	if strings.TrimSpace(*kinds) != "" {
		payload["kinds"] = strings.Split(*kinds, ",")
	}
	if strings.TrimSpace(*statuses) != "" {
		payload["statuses"] = strings.Split(*statuses, ",")
	}
	if strings.TrimSpace(*names) != "" {
		payload["names"] = strings.Split(*names, ",")
	}
	if *priorityMax > 0 {
		payload["priorityMax"] = *priorityMax
	}
	if strings.TrimSpace(*tags) != "" {
		payload["tags"] = strings.Split(*tags, ",")
	}
	if *noPRD {
		includePrd := false
		payload["includePrd"] = includePrd
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.Request("POST", "/backlog/export", nil, json.RawMessage(bodyBytes))
	if err != nil {
		return err
	}

	if *jsonOut {
		// In JSON mode, output metadata about the export
		result := map[string]any{
			"size_bytes": len(body),
			"format":     "markdown",
		}
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
		return nil
	}

	output := strings.TrimSpace(*outPath)
	if output == "" {
		output = "backlog-export-" + time.Now().Format("2006-01-02") + ".md"
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil && filepath.Dir(output) != "." {
		return fmt.Errorf("prepare output directory: %w", err)
	}
	if err := os.WriteFile(output, body, 0o644); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}

	printSection("Result")
	fmt.Printf("  Exported backlog to %s (%d bytes)\n", output, len(body))
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "import", "--file", output),
		cliCommand("backlog", "import", "--file", output, "--apply"),
	})
	return nil
}

func (a *App) cmdBacklogImport(args []string) error {
	fs := flag.NewFlagSet("backlog import", flag.ContinueOnError)
	fileFlag := fs.String("file", "", "Import file path")
	apply := fs.Bool("apply", false, "Apply changes (default: dry-run only)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("file", *fileFlag); err != nil {
		return fmt.Errorf("usage: backlog import --file FILE [--apply] [--json]\n\n%s", err)
	}

	filePath := strings.TrimSpace(*fileFlag)

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}

	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}
	if *apply {
		if err := writer.WriteField("apply", "true"); err != nil {
			return fmt.Errorf("write apply field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize multipart request: %w", err)
	}

	body, err := a.requestMultipart("POST", "/backlog/import", formBody.Bytes(), writer.FormDataContentType())
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ImportBacklogResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	if response.DryRun {
		fmt.Println("  Mode: dry-run (use --apply to apply changes)")
	} else {
		fmt.Println("  Mode: applied")
	}
	fmt.Printf("  %s\n", response.Summary)

	if len(response.Changes) > 0 {
		printSection("Changes")
		for _, change := range response.Changes {
			fmt.Printf("  [%s] %s\n", change.Action, change.Item)
			for _, detail := range change.Details {
				fmt.Printf("    - %s\n", detail)
			}
		}
	}

	if len(response.Errors) > 0 {
		printSection("Errors")
		for _, e := range response.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	if response.DryRun {
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "import", "--file", filePath, "--apply"),
		})
	} else {
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "list"),
		})
	}
	return nil
}
