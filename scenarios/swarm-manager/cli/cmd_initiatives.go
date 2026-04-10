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

	"github.com/vrooli/cli-core/cliutil"
)

func isValidInitiativeStatus(status string) bool {
	switch status {
	case "active", "completed":
		return true
	default:
		return false
	}
}

func (a *App) cmdInitiativesList(args []string) error {
	fs := flag.NewFlagSet("initiatives list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.getV1("/initiatives", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ListInitiativesResponse](body)
	if err != nil {
		return err
	}

	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No initiatives found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("initiatives", "create", "--data", "'{\"name\":\"my-init\",\"title\":\"My Initiative\"}'"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d initiative(s)\n", len(response.Items))

	printSection("Initiatives")
	for _, item := range response.Items {
		init := item.Initiative
		rollup := item.Rollup
		fmt.Printf("  %s (%s)\n", init.Name, init.Status)
		fmt.Printf("    Title: %s\n", init.Title)
		if init.Description != "" {
			fmt.Printf("    Description: %s\n", init.Description)
		}
		fmt.Printf("    Items: %d total, %d completed, %d in-progress, %d failed, %d pending\n",
			rollup.Total, rollup.Completed, rollup.InProgress, rollup.Failed, rollup.Pending)
		if len(init.Items) > 0 {
			fmt.Printf("    References: %s\n", strings.Join(init.Items, ", "))
		}
		fmt.Println()
	}

	first := response.Items[0].Initiative
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "get", "--name", first.Name),
	})
	return nil
}

func (a *App) cmdInitiativesGet(args []string) error {
	fs := flag.NewFlagSet("initiatives get", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives get --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.getV1("/initiatives/"+name, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[InitiativeResponse](body)
	if err != nil {
		return err
	}
	init := response.Initiative
	rollup := response.Rollup

	printSection("Initiative")
	fmt.Printf("  Name: %s\n", init.Name)
	fmt.Printf("  Title: %s\n", init.Title)
	if init.Description != "" {
		fmt.Printf("  Description: %s\n", init.Description)
	}
	fmt.Printf("  Status: %s\n", init.Status)
	fmt.Printf("  Created: %s\n", init.Created)
	fmt.Printf("  Updated: %s\n", init.Updated)

	printSection("Rollup")
	fmt.Printf("  Total: %d\n", rollup.Total)
	fmt.Printf("  Completed: %d\n", rollup.Completed)
	fmt.Printf("  In Progress: %d\n", rollup.InProgress)
	fmt.Printf("  Failed: %d\n", rollup.Failed)
	fmt.Printf("  Pending: %d\n", rollup.Pending)

	if len(init.Items) > 0 {
		printSection("Items")
		for _, item := range init.Items {
			fmt.Printf("  - %s\n", item)
		}
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "update", "--name", init.Name, "--data", "'{\"title\":\"...\"}'"),
		cliCommand("initiatives", "delete", "--name", init.Name),
	})
	return nil
}

func (a *App) cmdInitiativesCreate(args []string) error {
	fs := flag.NewFlagSet("initiatives create", flag.ContinueOnError)
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("data", *data); err != nil {
		return fmt.Errorf("usage: initiatives create --data JSON [--json]\n\nExample:\n  initiatives create --data '{\"name\":\"my-init\",\"title\":\"My Initiative\"}'\n\n%s", err)
	}

	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	var req InitiativeCreateRequest
	if err := decodeJSONStrict(payload, &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if req.Name == "" || req.Title == "" {
		return fmt.Errorf("name and title are required fields")
	}
	if status := strings.TrimSpace(req.Status); status != "" && !isValidInitiativeStatus(status) {
		return fmt.Errorf("status must be active or completed")
	}

	body, err := a.requestV1("POST", "/initiatives", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[InitiativeResponse](body)
	if err != nil {
		return err
	}

	printSection("Created")
	fmt.Printf("  Name: %s\n", response.Initiative.Name)
	fmt.Printf("  Title: %s\n", response.Initiative.Title)
	fmt.Printf("  Status: %s\n", response.Initiative.Status)

	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "get", "--name", response.Initiative.Name),
	})
	return nil
}

func (a *App) cmdInitiativesUpdate(args []string) error {
	fs := flag.NewFlagSet("initiatives update", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "data", *data); err != nil {
		return fmt.Errorf("usage: initiatives update --name NAME --data JSON [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}
	var req InitiativeUpdateRequest
	if err := decodeJSONStrict(payload, &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if !req.HasChanges() {
		return fmt.Errorf("at least one field must be provided")
	}
	if req.Status != nil && !isValidInitiativeStatus(strings.TrimSpace(*req.Status)) {
		return fmt.Errorf("status must be active or completed")
	}

	body, err := a.requestV1("PUT", "/initiatives/"+name, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[InitiativeResponse](body)
	if err != nil {
		return err
	}

	printSection("Updated")
	fmt.Printf("  Name: %s\n", response.Initiative.Name)
	fmt.Printf("  Title: %s\n", response.Initiative.Title)
	fmt.Printf("  Status: %s\n", response.Initiative.Status)
	return nil
}

func (a *App) cmdInitiativesDelete(args []string) error {
	fs := flag.NewFlagSet("initiatives delete", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives delete --name NAME\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	_, err := a.requestV1("DELETE", "/initiatives/"+name, nil, nil)
	if err != nil {
		return err
	}

	printSection("Deleted")
	fmt.Printf("  Initiative %q deleted.\n", name)
	return nil
}

func (a *App) cmdInitiativesAddItems(args []string) error {
	fs := flag.NewFlagSet("initiatives add-items", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	itemsFlag := fs.String("items", "", "Comma-separated item references (kind/name)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "items", *itemsFlag); err != nil {
		return fmt.Errorf("usage: initiatives add-items --name NAME --items kind/name,kind/name [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	items := parseCommaSeparated(*itemsFlag)
	if len(items) == 0 {
		return fmt.Errorf("at least one item reference is required")
	}

	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return err
	}

	body, err := a.requestV1("POST", "/initiatives/"+name+"/items", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[InitiativeResponse](body)
	if err != nil {
		return err
	}

	printSection("Items Added")
	fmt.Printf("  Initiative: %s\n", response.Initiative.Name)
	fmt.Printf("  Total items: %d\n", len(response.Initiative.Items))
	for _, item := range response.Initiative.Items {
		fmt.Printf("  - %s\n", item)
	}
	return nil
}

func (a *App) cmdInitiativesRemoveItems(args []string) error {
	fs := flag.NewFlagSet("initiatives remove-items", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	itemsFlag := fs.String("items", "", "Comma-separated item references (kind/name)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "items", *itemsFlag); err != nil {
		return fmt.Errorf("usage: initiatives remove-items --name NAME --items kind/name,kind/name [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	items := parseCommaSeparated(*itemsFlag)
	if len(items) == 0 {
		return fmt.Errorf("at least one item reference is required")
	}

	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return err
	}

	body, err := a.requestV1("DELETE", "/initiatives/"+name+"/items", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[InitiativeResponse](body)
	if err != nil {
		return err
	}

	printSection("Items Removed")
	fmt.Printf("  Initiative: %s\n", response.Initiative.Name)
	fmt.Printf("  Remaining items: %d\n", len(response.Initiative.Items))
	for _, item := range response.Initiative.Items {
		fmt.Printf("  - %s\n", item)
	}
	return nil
}

func (a *App) cmdInitiativesFiles(args []string) error {
	fs := flag.NewFlagSet("initiatives files", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives files --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.getV1("/initiatives/"+name+"/files", nil)
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
		fmt.Printf("  No files found for initiative %s.\n", name)
		printCommandListSection("Next Steps", []string{
			cliCommand("initiatives", "file-upload", "--name", name, "--path", "<file>", "--stdin"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d file node(s) for initiative %s\n", len(response.Files), name)
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
		cliCommand("initiatives", "file-get", "--name", name, "--path", "<path>"),
		cliCommand("initiatives", "file-upload", "--name", name, "--path", "<path>", "--stdin"),
	})
	return nil
}

func (a *App) cmdInitiativesFileGet(args []string) error {
	fs := flag.NewFlagSet("initiatives file-get", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	pathFlag := fs.String("path", "", "File path within initiative")
	outPath := fs.String("out", "", "Write file content to local path instead of stdout")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "path", *pathFlag); err != nil {
		return fmt.Errorf("usage: initiatives file-get --name NAME --path PATH [--out local-path] [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	filePath := strings.TrimSpace(*pathFlag)

	body, err := a.getV1("/initiatives/"+name+"/files/"+filePath, nil)
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
	if dir := filepath.Dir(*outPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("prepare output directory: %w", err)
		}
	}
	if err := os.WriteFile(*outPath, body, 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	printSection("Result")
	fmt.Printf("  Saved file to %s\n", *outPath)
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "files", "--name", name),
	})
	return nil
}

func (a *App) cmdInitiativesFileUpload(args []string) error {
	fs := flag.NewFlagSet("initiatives file-upload", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	serverPath := fs.String("path", "", "Full server-side destination path")
	localFile := fs.String("file", "", "Local file path to upload")
	contentStr := fs.String("content", "", "Inline content string to upload")
	stdinFlag := fs.Bool("stdin", false, "Read content from stdin")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives file-upload --name NAME --path PATH (--stdin|--file FILE|--content CONTENT) [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	fileStr := strings.TrimSpace(*localFile)
	contentVal := *contentStr

	if *stdinFlag {
		if fileStr != "" || contentVal != "" {
			return fmt.Errorf("--stdin cannot be combined with --file or --content")
		}
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		contentVal = string(stdinBytes)
	}

	if fileStr == "" && contentVal == "" {
		return fmt.Errorf("usage: initiatives file-upload --name NAME --path PATH (--stdin|--file FILE|--content CONTENT) [--json]\n\neither --stdin, --file, or --content is required")
	}
	if fileStr != "" && contentVal != "" {
		return fmt.Errorf("--file and --content are mutually exclusive")
	}

	sp := strings.TrimSpace(*serverPath)

	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)

	if contentVal != "" {
		if sp == "" {
			return fmt.Errorf("--path is required when using --content or --stdin")
		}
		serverDir := filepath.Dir(sp)
		serverFile := filepath.Base(sp)

		part, err := writer.CreateFormFile("file", serverFile)
		if err != nil {
			return fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, strings.NewReader(contentVal)); err != nil {
			return fmt.Errorf("copy content: %w", err)
		}
		if serverDir != "." && serverDir != "" {
			if err := writer.WriteField("path", serverDir); err != nil {
				return fmt.Errorf("write path field: %w", err)
			}
		}
	} else {
		if sp == "" {
			sp = filepath.Base(fileStr)
		}
		serverDir := filepath.Dir(sp)
		serverFile := filepath.Base(sp)

		file, err := os.Open(fileStr)
		if err != nil {
			return fmt.Errorf("open local file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("file", serverFile)
		if err != nil {
			return fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			return fmt.Errorf("copy file content: %w", err)
		}
		if serverDir != "." && serverDir != "" {
			if err := writer.WriteField("path", serverDir); err != nil {
				return fmt.Errorf("write path field: %w", err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize multipart request: %w", err)
	}

	respBody, err := a.requestMultipartV1("POST", "/initiatives/"+name+"/files", formBody.Bytes(), writer.FormDataContentType())
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
		cliCommand("initiatives", "files", "--name", name),
		cliCommand("initiatives", "file-get", "--name", name, "--path", parsed.File.Path),
	})
	return nil
}

func (a *App) cmdInitiativesFileOp(args []string) error {
	fs := flag.NewFlagSet("initiatives file-op", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	opFlag := fs.String("op", "", "Operation: delete, rename, move, copy")
	sourceFlag := fs.String("source", "", "Source file path")
	destFlag := fs.String("dest", "", "Destination file path (for rename/move/copy)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "op", *opFlag, "source", *sourceFlag); err != nil {
		return fmt.Errorf("usage: initiatives file-op --name NAME --op OP --source PATH [--dest PATH] [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	op := strings.ToLower(strings.TrimSpace(*opFlag))
	source := strings.TrimSpace(*sourceFlag)
	dest := strings.TrimSpace(*destFlag)

	switch op {
	case "delete", "rename", "move", "copy":
		// valid
	default:
		return fmt.Errorf("unsupported operation %q: must be delete, rename, move, or copy", op)
	}
	if op != "delete" && dest == "" {
		return fmt.Errorf("--dest is required for %s operation", op)
	}

	payload := map[string]string{
		"operation":   op,
		"source_path": source,
	}
	if dest != "" {
		payload["destination_path"] = dest
	}
	payloadBytes, _ := json.Marshal(payload)

	body, err := a.requestV1("PATCH", "/initiatives/"+name+"/files", nil, payloadBytes)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	printSection("Result")
	if op == "delete" {
		fmt.Printf("  Deleted: %s\n", source)
	} else {
		fmt.Printf("  %s: %s → %s\n", strings.ToUpper(op[:1])+op[1:], source, dest)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "files", "--name", name),
	})
	return nil
}

// parseCommaSeparated splits a comma-separated string and trims whitespace.
func parseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
