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

func (a *App) cmdCapturesList(args []string) error {
	fs := flag.NewFlagSet("captures list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.getV1("/captures", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ListCapturesResponse](body)
	if err != nil {
		return err
	}

	if len(response.Captures) == 0 {
		printSection("Summary")
		fmt.Println("  No captures found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("captures", "create", "--text", "'My idea or thought'"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d capture(s)\n", len(response.Captures))

	printSection("Results")
	for _, cap := range response.Captures {
		preview := cap.Text
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		fmt.Printf("  [%s] %s  %s  %s\n", cap.Status, cap.ID, preview, cap.Created)
	}

	first := response.Captures[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("captures", "get", "--id", first.ID),
		cliCommand("captures", "classify", "--id", first.ID),
		cliCommand("captures", "delete", "--id", first.ID),
	})
	return nil
}

func (a *App) cmdCapturesCreate(args []string) error {
	fs := flag.NewFlagSet("captures create", flag.ContinueOnError)
	textFlag := fs.String("text", "", "Capture text (required)")
	var fileFlags stringSlice
	fs.Var(&fileFlags, "file", "File to attach (can be repeated)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("text", *textFlag); err != nil {
		return fmt.Errorf("usage: captures create --text TEXT [--file FILE]... [--json]\n\n%s", err)
	}

	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)

	if err := writer.WriteField("text", *textFlag); err != nil {
		return fmt.Errorf("write text field: %w", err)
	}

	for _, filePath := range fileFlags {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("open file %s: %w", filePath, err)
		}
		part, err := writer.CreateFormFile("files", filepath.Base(filePath))
		if err != nil {
			file.Close()
			return fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			file.Close()
			return fmt.Errorf("copy file content: %w", err)
		}
		file.Close()
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize multipart request: %w", err)
	}

	body, err := a.requestMultipartV1("POST", "/captures", formBody.Bytes(), writer.FormDataContentType())
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[CaptureCreateResponse](body)
	if err != nil {
		return err
	}
	cap := response.Capture

	printSection("Result")
	fmt.Printf("  Created capture: %s\n", cap.ID)
	printSection("What Changed")
	fmt.Printf("  Status: %s\n", cap.Status)
	if response.RunID != "" {
		fmt.Printf("  Classification run: %s\n", response.RunID)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("captures", "get", "--id", cap.ID),
		cliCommand("captures", "classify", "--id", cap.ID),
	})
	return nil
}

func (a *App) cmdCapturesGet(args []string) error {
	fs := flag.NewFlagSet("captures get", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Capture ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: captures get --id ID [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	body, err := a.getV1("/captures/"+id, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[CaptureResponse](body)
	if err != nil {
		return err
	}
	cap := response.Capture

	printSection("Summary")
	fmt.Printf("  %s (%s)\n", cap.ID, cap.Status)

	printSection("Details")
	fmt.Printf("  ID: %s\n", cap.ID)
	fmt.Printf("  Status: %s\n", cap.Status)
	fmt.Printf("  Text: %s\n", cap.Text)
	fmt.Printf("  Created: %s\n", cap.Created)
	if len(cap.Attachments) > 0 {
		fmt.Printf("  Attachments: %s\n", strings.Join(cap.Attachments, ", "))
	}

	if cap.Classification != nil {
		printSection("Classification")
		fmt.Printf("  Classified at: %s\n", cap.Classification.ClassifiedAt)
		for i, item := range cap.Classification.Items {
			fmt.Printf("  [%d] %s: %s (priority: %d, confidence: %.0f%%)\n",
				i+1, item.Kind, item.Title, item.Priority, item.Confidence*100)
			if item.Description != "" {
				fmt.Printf("      %s\n", item.Description)
			}
			if len(item.Tags) > 0 {
				fmt.Printf("      Tags: %s\n", strings.Join(item.Tags, ", "))
			}
		}
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("captures", "classify", "--id", cap.ID),
		cliCommand("captures", "delete", "--id", cap.ID),
	})
	return nil
}

func (a *App) cmdCapturesDelete(args []string) error {
	fs := flag.NewFlagSet("captures delete", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Capture ID")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: captures delete --id ID\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	if _, err := a.requestV1("DELETE", "/captures/"+id, nil, nil); err != nil {
		return err
	}

	fmt.Printf("Deleted capture %s\n", id)
	return nil
}

func (a *App) cmdCapturesClassify(args []string) error {
	fs := flag.NewFlagSet("captures classify", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Capture ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: captures classify --id ID [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	body, err := a.requestV1("POST", "/captures/"+id+"/classify", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	type classifyResponse struct {
		TaskID  string `json:"task_id"`
		RunID   string `json:"run_id"`
		BaseURL string `json:"base_url"`
		Created string `json:"created"`
	}
	response, err := decodeResponse[classifyResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  Classification triggered for capture: %s\n", id)
	if response.RunID != "" {
		fmt.Printf("  Run ID: %s\n", response.RunID)
	}
	if response.TaskID != "" {
		fmt.Printf("  Task ID: %s\n", response.TaskID)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("captures", "get", "--id", id),
		cliCommand("agent-manager", "run-get", "--id", response.RunID),
	})
	return nil
}
