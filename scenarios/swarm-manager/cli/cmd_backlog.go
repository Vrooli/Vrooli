package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogList(args []string) error {
	fs := flag.NewFlagSet("backlog list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if fs.NArg() > 0 {
		kinds := strings.Join(fs.Args(), ",")
		if strings.TrimSpace(kinds) != "" {
			query.Set("kinds", kinds)
		}
	}

	body, err := a.getV1("/backlog", query)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ListBacklogResponse](body)
	if err != nil {
		return err
	}

	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No backlog items found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "create", "'{\"name\":\"my-idea\",\"title\":\"My Idea\",\"kind\":\"idea\"}'"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d backlog item(s)\n", len(response.Items))
	if kinds := strings.TrimSpace(query.Get("kinds")); kinds != "" {
		fmt.Printf("  Filtered kinds: %s\n", kinds)
	}

	printSection("Results")
	for _, item := range response.Items {
		fmt.Printf("  [%s] %s (priority: %d, status: %s)\n", item.Kind, item.Name, item.Priority, item.Status)
		fmt.Printf("    Title: %s\n", item.Title)
		if len(item.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(item.Tags, ", "))
		}
		if item.Kind == "research" && item.ResearchTarget != "" {
			fmt.Printf("    Target: %s\n", item.ResearchTarget)
		}
		fmt.Println()
	}

	first := response.Items[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("backlog", "get", "<kind>", "<name>"),
		cliCommand("backlog", "get", first.Kind, first.Name),
		cliCommand("backlog", "files", first.Kind, first.Name),
		cliCommand("backlog", "queue", first.Kind, first.Name),
	})
	return nil
}

func (a *App) cmdBacklogGet(args []string) error {
	fs := flag.NewFlagSet("backlog get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: backlog get <kind> <name> [--json]")
	}
	kind := fs.Arg(0)
	name := fs.Arg(1)

	body, err := a.getV1("/backlog/"+kind+"/"+name, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Summary")
	fmt.Printf("  %s/%s (%s)\n", item.Kind, item.Name, item.Status)

	printSection("Details")
	fmt.Printf("  Name: %s\n", item.Name)
	fmt.Printf("  Kind: %s\n", item.Kind)
	fmt.Printf("  Title: %s\n", item.Title)
	fmt.Printf("  Description: %s\n", item.Description)
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	if len(item.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(item.Tags, ", "))
	}
	if item.ResearchTarget != "" {
		fmt.Printf("  Research Target: %s\n", item.ResearchTarget)
	}
	fmt.Printf("  Created: %s\n", item.Created)
	fmt.Printf("  Updated: %s\n", item.Updated)

	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "files", item.Kind, item.Name),
		cliCommand("backlog", "update", item.Kind, item.Name, "'{\"status\":\"ready\"}'"),
		cliCommand("backlog", "queue", item.Kind, item.Name),
	})
	return nil
}

func (a *App) cmdBacklogCreate(args []string) error {
	fs := flag.NewFlagSet("backlog create", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: backlog create <json-or-@file> [--json]\n\nExample:\n  backlog create '{\"name\":\"my-idea\",\"title\":\"My Idea\",\"kind\":\"idea\"}'")
	}

	payload, err := parseJSONArg(fs.Args())
	if err != nil {
		return err
	}

	var req CreateBacklogRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if req.Name == "" || req.Title == "" || req.Kind == "" {
		return fmt.Errorf("name, title, and kind are required fields")
	}

	body, err := a.requestV1("POST", "/backlog", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Result")
	fmt.Printf("  Created backlog item: %s/%s\n", item.Kind, item.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", item.Kind, item.Name),
		cliCommand("backlog", "files", item.Kind, item.Name),
		cliCommand("backlog", "queue", item.Kind, item.Name),
	})
	return nil
}

func (a *App) cmdBacklogUpdate(args []string) error {
	fs := flag.NewFlagSet("backlog update", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 3 {
		return fmt.Errorf("usage: backlog update <kind> <name> <json-or-@file> [--json]\n\nExample:\n  backlog update idea my-idea '{\"title\":\"Updated Title\",\"status\":\"ready\"}'")
	}

	kind := fs.Arg(0)
	name := fs.Arg(1)
	payload, err := parseJSONArg(fs.Args()[2:])
	if err != nil {
		return err
	}

	var update map[string]any
	if err := json.Unmarshal(payload, &update); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.requestV1("PUT", "/backlog/"+kind+"/"+name, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}
	item := response.Item

	printSection("Result")
	fmt.Printf("  Updated backlog item: %s/%s\n", item.Kind, item.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", item.Status)
	fmt.Printf("  Priority: %d\n", item.Priority)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", item.Kind, item.Name),
		cliCommand("backlog", "queue", item.Kind, item.Name),
	})
	return nil
}

func (a *App) cmdBacklogDelete(args []string) error {
	fs := flag.NewFlagSet("backlog delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: backlog delete <kind> <name> [--json]")
	}
	kind := fs.Arg(0)
	name := fs.Arg(1)

	body, err := a.requestV1("DELETE", "/backlog/"+kind+"/"+name, nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	printSection("Result")
	fmt.Printf("  Deleted backlog item: %s/%s\n", kind, name)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "list"),
		cliCommand("backlog", "create", "'{\"name\":\"new-item\",\"title\":\"New Item\",\"kind\":\"idea\"}'"),
	})
	return nil
}

func (a *App) cmdBacklogFiles(args []string) error {
	fs := flag.NewFlagSet("backlog files", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: backlog files <kind> <name> [--json]")
	}
	kind := strings.TrimSpace(fs.Arg(0))
	name := strings.TrimSpace(fs.Arg(1))

	body, err := a.getV1("/backlog/"+kind+"/"+name+"/files", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogFilesResponse](body)
	if err != nil {
		return err
	}
	if len(response.Files) == 0 {
		printSection("Summary")
		fmt.Printf("  No files found for %s/%s.\n", kind, name)
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "file-upload", kind, name, "<local-file>"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d file node(s) for %s/%s\n", len(response.Files), kind, name)
	printSection("Results")
	printTree(response.Files,
		func(item BacklogFile) []BacklogFile { return item.Children },
		func(item BacklogFile) string {
			if item.Type == "directory" {
				return item.Name + "/"
			}
			return fmt.Sprintf("%s (%d bytes)", item.Name, item.Size)
		},
		0,
	)
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("backlog", "file-get", kind, name, "<path>"),
		cliCommand("backlog", "file-upload", kind, name, "<local-file>", "--path", "docs"),
	})
	return nil
}

func (a *App) cmdBacklogQueue(args []string) error {
	fs := flag.NewFlagSet("backlog queue", flag.ContinueOnError)
	mode, delaySeconds, operation, startedBy := addExecutionOptionsFlags(fs)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: backlog queue <kind> <name> [--mode manual|scheduled|yolo] [--delay-seconds N] [--operation generator|improver] [--started-by NAME] [--json]")
	}

	opts, err := parseExecutionOptions(mode, delaySeconds, operation, startedBy, false)
	if err != nil {
		return err
	}

	kind := fs.Arg(0)
	name := fs.Arg(1)
	payload, err := json.Marshal(map[string]any{
		"operation":     opts.operation,
		"mode":          opts.mode,
		"delay_seconds": opts.delaySeconds,
		"started_by":    opts.startedBy,
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.requestV1("POST", "/backlog/"+kind+"/"+name+"/queue", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[QueueBacklogResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  Queued backlog item: %s/%s\n", response.Item.Kind, response.Item.Name)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", response.Item.Status)
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	if response.RunID != "" {
		fmt.Printf("  Run ID: %s\n", response.RunID)
	}
	fmt.Printf("  Mode: %s\n", opts.mode)
	if opts.mode == "scheduled" {
		fmt.Printf("  Delay Seconds: %d\n", opts.delaySeconds)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("execution", "list", "--backlog-kind", response.Item.Kind, "--backlog-name", response.Item.Name),
		cliCommand("execution", "get", "<execution-id>"),
	})
	return nil
}

func (a *App) cmdBacklogResearch(args []string) error {
	fs := flag.NewFlagSet("backlog research", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: backlog research <kind> <name> [json-or-@file] [--json]\n\nExample:\n  backlog research idea my-idea '{\"prompt\":\"Focus on risks\"}'")
	}
	kind := fs.Arg(0)
	name := fs.Arg(1)

	var payload json.RawMessage
	if fs.NArg() > 2 {
		parsed, err := parseJSONArg(fs.Args()[2:])
		if err != nil {
			return err
		}
		payload = parsed
	}

	body, err := a.requestV1("POST", "/backlog/"+kind+"/"+name+"/research", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ResearchResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  Started research for %s/%s\n", kind, name)
	printSection("What Changed")
	fmt.Printf("  Task ID: %s\n", response.TaskID)
	fmt.Printf("  Run ID: %s\n", response.RunID)
	fmt.Printf("  Base URL: %s\n", response.BaseURL)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", kind, name),
		cliCommand("execution", "list", "--backlog-kind", kind, "--backlog-name", name),
	})
	return nil
}

func (a *App) cmdBacklogPromptTrace(args []string) error {
	fs := flag.NewFlagSet("backlog prompt-trace", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: backlog prompt-trace <kind> <name> [--json]")
	}
	kind := strings.TrimSpace(fs.Arg(0))
	name := strings.TrimSpace(fs.Arg(1))
	if kind == "" || name == "" {
		return fmt.Errorf("kind and name are required")
	}

	body, err := a.getV1("/backlog/"+kind+"/"+name+"/prompt-trace", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptTraceResponse](body)
	if err != nil {
		return err
	}
	printPromptTraceSummary(
		"Summary",
		fmt.Sprintf("Prompt trace for backlog item %s/%s", kind, name),
		response.Trace,
	)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "research", kind, name),
		cliCommand("backlog", "get", kind, name),
	})
	return nil
}

func (a *App) cmdBacklogConvert(args []string) error {
	fs := flag.NewFlagSet("backlog convert", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 3 {
		return fmt.Errorf("usage: backlog convert <kind> <name> <target-kind> [target-name] [--json]")
	}
	kind := fs.Arg(0)
	name := fs.Arg(1)
	targetKind := fs.Arg(2)
	targetName := ""
	if fs.NArg() > 3 {
		targetName = strings.Join(fs.Args()[3:], " ")
	}

	payload := map[string]string{
		"targetKind": targetKind,
	}
	if strings.TrimSpace(targetName) != "" {
		payload["targetName"] = targetName
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.requestV1("POST", "/backlog/"+kind+"/"+name+"/convert", nil, json.RawMessage(bodyBytes))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[BacklogItemResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  Converted backlog item: %s/%s -> %s/%s\n", kind, name, response.Item.Kind, response.Item.Name)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", response.Item.Kind, response.Item.Name),
		cliCommand("backlog", "list", response.Item.Kind),
	})
	return nil
}

func (a *App) cmdBacklogFileGet(args []string) error {
	fs := flag.NewFlagSet("backlog file get", flag.ContinueOnError)
	outPath := fs.String("out", "", "Write file content to local path instead of stdout")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 3 {
		return fmt.Errorf("usage: backlog file get <kind> <name> <path> [--out local-path] [--json]")
	}
	kind := strings.TrimSpace(fs.Arg(0))
	name := strings.TrimSpace(fs.Arg(1))
	filePath := strings.TrimSpace(fs.Arg(2))
	if kind == "" || name == "" || filePath == "" {
		return fmt.Errorf("kind, name, and path are required")
	}

	body, err := a.getV1("/backlog/"+kind+"/"+name+"/files/"+filePath, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	if strings.TrimSpace(*outPath) == "" {
		fmt.Print(string(body))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil && filepath.Dir(*outPath) != "." {
		return fmt.Errorf("prepare output directory: %w", err)
	}
	if err := os.WriteFile(*outPath, body, 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	printSection("Result")
	fmt.Printf("  Saved file to %s\n", *outPath)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "files", kind, name),
	})
	return nil
}

func (a *App) cmdBacklogFile(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: backlog file <get|upload> ...")
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "get":
		return a.cmdBacklogFileGet(args[1:])
	case "upload":
		return a.cmdBacklogFileUpload(args[1:])
	default:
		return fmt.Errorf("unknown backlog file subcommand %q (expected get or upload)", args[0])
	}
}

func (a *App) cmdBacklogFileUpload(args []string) error {
	fs := flag.NewFlagSet("backlog file upload", flag.ContinueOnError)
	targetPath := fs.String("path", "", "Optional directory path within backlog item")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 3 {
		return fmt.Errorf("usage: backlog file upload <kind> <name> <local-file> [--path backlog/subdir] [--json]")
	}

	kind := strings.TrimSpace(fs.Arg(0))
	name := strings.TrimSpace(fs.Arg(1))
	localFile := strings.TrimSpace(fs.Arg(2))
	if kind == "" || name == "" || localFile == "" {
		return fmt.Errorf("kind, name, and local-file are required")
	}

	file, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()

	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)
	part, err := writer.CreateFormFile("file", filepath.Base(localFile))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}
	if strings.TrimSpace(*targetPath) != "" {
		if err := writer.WriteField("path", strings.TrimSpace(*targetPath)); err != nil {
			return fmt.Errorf("write path field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize multipart request: %w", err)
	}

	respBody, err := a.requestMultipartV1("POST", "/backlog/"+kind+"/"+name+"/files", formBody.Bytes(), writer.FormDataContentType())
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(respBody)
		return nil
	}

	parsed, err := decodeResponse[BacklogFileResponse](respBody)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  Uploaded file: %s\n", parsed.File.Path)
	printSection("What Changed")
	fmt.Printf("  Name: %s\n", parsed.File.Name)
	fmt.Printf("  Size: %d bytes\n", parsed.File.Size)
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "files", kind, name),
		cliCommand("backlog", "file-get", kind, name, parsed.File.Path),
	})
	return nil
}

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

	body, err := a.requestV1("POST", "/backlog/export", nil, json.RawMessage(bodyBytes))
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
		cliCommand("backlog", "import", output),
		cliCommand("backlog", "import", output, "--apply"),
	})
	return nil
}

func (a *App) cmdBacklogImport(args []string) error {
	fs := flag.NewFlagSet("backlog import", flag.ContinueOnError)
	apply := fs.Bool("apply", false, "Apply changes (default: dry-run only)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: backlog import <file> [--apply] [--json]")
	}

	filePath := strings.TrimSpace(fs.Arg(0))
	if filePath == "" {
		return fmt.Errorf("file path is required")
	}

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

	body, err := a.requestMultipartV1("POST", "/backlog/import", formBody.Bytes(), writer.FormDataContentType())
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
			cliCommand("backlog", "import", filePath, "--apply"),
		})
	} else {
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "list"),
		})
	}
	return nil
}
