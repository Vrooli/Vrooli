package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdExport(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: export <file> [--format json] [--include-history] [--patterns PATTERN]")
	}

	filePath, err := ensureFilePath(args[0])
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "json", "Export format (json only)")
	includeHistory := fs.Bool("include-history", false, "Include visit history")
	patterns := fs.String("patterns", "", "Comma-separated patterns")
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}

	_ = format
	_ = includeHistory

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{}, true)
	if err != nil {
		return err
	}

	query := map[string]string{}
	if strings.TrimSpace(*patterns) != "" {
		query["patterns"] = strings.TrimSpace(*patterns)
	}
	if strings.TrimSpace(*format) != "" {
		query["format"] = strings.TrimSpace(*format)
	}
	if *includeHistory {
		query["include_history"] = "true"
	}

	body, err := a.core.APIClient.Get(a.apiPath("/campaigns/"+campaignID+"/export"), buildQuery(query))
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, body, 0o600); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}

	fmt.Println("Export completed")
	fmt.Printf("File: %s\n", filePath)
	return nil
}

func (a *App) cmdImport(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: import <file> [--merge true|false] [--format json] [--json]")
	}

	filePath, err := ensureFilePath(args[0])
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	mergeValue := fs.String("merge", "", "Merge with existing campaign (true/false)")
	format := fs.String("format", "json", "Import format (json only)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}

	_ = format

	payload, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}

	var campaignData map[string]interface{}
	if err := json.Unmarshal(payload, &campaignData); err != nil {
		return fmt.Errorf("invalid campaign data: %w", err)
	}

	query := map[string]string{}
	mergeVal := strings.TrimSpace(*mergeValue)
	if mergeVal != "" {
		if mergeVal == "true" || mergeVal == "false" {
			query["merge"] = mergeVal
		} else {
			query["merge"] = "true"
		}
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns/import"), buildQuery(query), campaignData)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var response importResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse import response: %w", err)
	}

	message := strings.TrimSpace(response.Message)
	if message == "" {
		message = "Campaign imported"
	}
	fmt.Println(message)
	if response.Campaign.ID != "" {
		fmt.Printf("ID: %s\n", response.Campaign.ID)
	}
	return nil
}
