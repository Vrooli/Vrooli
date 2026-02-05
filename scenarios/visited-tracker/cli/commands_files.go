package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdFileGetByPath(args []string) error {
	fs := flag.NewFlagSet("files get-by-path", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", "", "File path to lookup")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	path := strings.TrimSpace(*pathFlag)
	if path == "" && fs.NArg() > 0 {
		path = strings.TrimSpace(fs.Arg(0))
	}
	if path == "" {
		return errors.New("--path is required")
	}

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/campaigns/"+campaignID+"/files/by-path"), buildQuery(map[string]string{"path": path}))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var file trackedFile
	if err := json.Unmarshal(body, &file); err != nil {
		return fmt.Errorf("parse file response: %w", err)
	}

	fmt.Printf("File: %s\n", file.FilePath)
	fmt.Printf("ID: %s\n", file.ID)
	fmt.Printf("Visits: %d\n", file.VisitCount)
	fmt.Printf("Staleness: %.0f\n", file.StalenessScore)
	fmt.Printf("Excluded: %t\n", file.Excluded)
	if file.Notes != nil && strings.TrimSpace(*file.Notes) != "" {
		fmt.Printf("Notes: %s\n", strings.TrimSpace(*file.Notes))
	}
	return nil
}

func (a *App) cmdFileNote(args []string) error {
	fs := flag.NewFlagSet("files note", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fileID := fs.String("file-id", "", "Tracked file ID")
	note := fs.String("note", "", "Note text")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*fileID) == "" {
		return errors.New("--file-id is required")
	}
	if strings.TrimSpace(*note) == "" {
		return errors.New("--note is required")
	}

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{"notes": strings.TrimSpace(*note)}
	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/campaigns/"+campaignID+"/files/"+strings.TrimSpace(*fileID)+"/notes"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("File notes updated")
	fmt.Printf("File ID: %s\n", strings.TrimSpace(*fileID))
	return nil
}

func (a *App) cmdFilePriority(args []string) error {
	fs := flag.NewFlagSet("files priority", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fileID := fs.String("file-id", "", "Tracked file ID")
	weight := fs.String("weight", "", "Priority weight (float)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*fileID) == "" {
		return errors.New("--file-id is required")
	}
	if strings.TrimSpace(*weight) == "" {
		return errors.New("--weight is required")
	}
	parsedWeight, err := strconv.ParseFloat(strings.TrimSpace(*weight), 64)
	if err != nil {
		return errors.New("--weight must be numeric")
	}

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{"priority_weight": parsedWeight}
	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/campaigns/"+campaignID+"/files/"+strings.TrimSpace(*fileID)+"/priority"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("File priority updated")
	fmt.Printf("File ID: %s\n", strings.TrimSpace(*fileID))
	fmt.Printf("Priority weight: %.2f\n", parsedWeight)
	return nil
}

func (a *App) cmdFileExclude(args []string) error {
	fs := flag.NewFlagSet("files exclude", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fileID := fs.String("file-id", "", "Tracked file ID")
	excluded := fs.Bool("excluded", true, "Set excluded state")
	include := fs.Bool("include", false, "Clear exclusion")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*fileID) == "" {
		return errors.New("--file-id is required")
	}

	excludedValue := *excluded
	if *include {
		excludedValue = false
	}

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{"excluded": excludedValue}
	body, err := a.core.APIClient.Request("PATCH", a.apiPath("/campaigns/"+campaignID+"/files/"+strings.TrimSpace(*fileID)+"/exclude"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("File exclusion updated")
	fmt.Printf("File ID: %s\n", strings.TrimSpace(*fileID))
	fmt.Printf("Excluded: %t\n", excludedValue)
	return nil
}
