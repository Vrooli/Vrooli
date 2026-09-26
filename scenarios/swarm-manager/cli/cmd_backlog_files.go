package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogFiles(args []string) error {
	fs := flag.NewFlagSet("backlog files", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog files --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/backlog/"+kind+"/"+name+"/files", nil)
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
			cliCommand("backlog", "file-upload", "--kind", kind, "--name", name, "--file", "<local-file>"),
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
		cliCommand("backlog", "file-get", "--kind", kind, "--name", name, "--path", "<path>"),
		cliCommand("backlog", "file-upload", "--kind", kind, "--name", name, "--path", "<path>", "--file", "<local-file>"),
	})
	return nil
}

func (a *App) cmdBacklogFileGet(args []string) error {
	fs := flag.NewFlagSet("backlog file-get", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	pathFlag := fs.String("path", "", "File path within backlog item")
	outPath := fs.String("out", "", "Write file content to local path instead of stdout")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag, "path", *pathFlag); err != nil {
		return fmt.Errorf("usage: backlog file-get --kind KIND --name NAME --path PATH [--out local-path] [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	filePath := strings.TrimSpace(*pathFlag)

	body, err := a.core.Get("/backlog/"+kind+"/"+name+"/files/"+filePath, nil)
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
		cliCommand("backlog", "files", "--kind", kind, "--name", name),
	})
	return nil
}

func (a *App) cmdBacklogFileUpload(args []string) error {
	fs := flag.NewFlagSet("backlog file-upload", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	serverPath := fs.String("path", "", "Full server-side destination path (e.g. workshop/round-001.json)")
	localFile := fs.String("file", "", "Local file path to upload")
	contentStr := fs.String("content", "", "Inline content string to upload (⚠️  prefer --stdin to avoid shell quoting issues)")
	stdinFlag := fs.Bool("stdin", false, "Read content from stdin (safest for content with special characters)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog file-upload --kind KIND --name NAME --path PATH (--stdin|--file FILE|--content CONTENT) [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	fileStr := strings.TrimSpace(*localFile)
	contentVal := *contentStr

	// Read from stdin if --stdin is set
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
		return fmt.Errorf("usage: backlog file-upload --kind KIND --name NAME --path PATH (--stdin|--file FILE|--content CONTENT) [--json]\n\neither --stdin, --file, or --content is required")
	}
	if fileStr != "" && contentVal != "" {
		return fmt.Errorf("usage: backlog file-upload --kind KIND --name NAME --path PATH (--stdin|--file FILE|--content CONTENT) [--json]\n\n--file and --content are mutually exclusive")
	}

	sp := strings.TrimSpace(*serverPath)

	pathRequiredErr := fmt.Errorf("usage: backlog file-upload --kind KIND --name NAME --path PATH --content CONTENT [--json]\n\n--path is required when using --content")
	formBody, contentType, err := buildFileUploadMultipart(sp, fileStr, contentVal, pathRequiredErr)
	if err != nil {
		return err
	}

	respBody, err := a.requestMultipart("POST", "/backlog/"+kind+"/"+name+"/files", formBody, contentType)
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
		cliCommand("backlog", "files", "--kind", kind, "--name", name),
		cliCommand("backlog", "file-get", "--kind", kind, "--name", name, "--path", parsed.File.Path),
	})
	return nil
}

// buildFileUploadMultipart constructs the multipart/form-data body shared by the
// backlog and milestones file-upload commands. Exactly one of contentVal or
// fileStr must be set (callers validate this). In content mode the server path
// is required and pathRequiredErr is returned when it is empty; in file mode the
// server path defaults to the local file's base name. Returns the encoded body
// and its Content-Type header value.
func buildFileUploadMultipart(serverPath, fileStr, contentVal string, pathRequiredErr error) ([]byte, string, error) {
	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)

	if contentVal != "" {
		if serverPath == "" {
			return nil, "", pathRequiredErr
		}
		if err := writeUploadPart(writer, serverPath, strings.NewReader(contentVal), "copy content"); err != nil {
			return nil, "", err
		}
	} else {
		if serverPath == "" {
			serverPath = filepath.Base(fileStr)
		}
		file, err := os.Open(fileStr)
		if err != nil {
			return nil, "", fmt.Errorf("open local file: %w", err)
		}
		defer file.Close()
		if err := writeUploadPart(writer, serverPath, file, "copy file content"); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("finalize multipart request: %w", err)
	}
	return formBody.Bytes(), writer.FormDataContentType(), nil
}

// writeUploadPart writes the "file" part (named by serverPath's base) from src
// and, when serverPath has a directory component, the "path" form field.
// copyErrLabel prefixes the error wrapping a copy failure.
func writeUploadPart(writer *multipart.Writer, serverPath string, src io.Reader, copyErrLabel string) error {
	serverDir := filepath.Dir(serverPath)
	serverFile := filepath.Base(serverPath)

	part, err := writer.CreateFormFile("file", serverFile)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, src); err != nil {
		return fmt.Errorf("%s: %w", copyErrLabel, err)
	}
	if serverDir != "." && serverDir != "" {
		if err := writer.WriteField("path", serverDir); err != nil {
			return fmt.Errorf("write path field: %w", err)
		}
	}
	return nil
}
