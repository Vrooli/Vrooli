package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

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

	body, err := a.core.Get("/initiatives/"+name+"/files", nil)
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

	body, err := a.core.Get("/initiatives/"+name+"/files/"+filePath, nil)
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

	pathRequiredErr := fmt.Errorf("--path is required when using --content or --stdin")
	formBody, contentType, err := buildFileUploadMultipart(sp, fileStr, contentVal, pathRequiredErr)
	if err != nil {
		return err
	}

	respBody, err := a.requestMultipart("POST", "/initiatives/"+name+"/files", formBody, contentType)
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

	body, err := a.core.Request("PATCH", "/initiatives/"+name+"/files", nil, payloadBytes)
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
