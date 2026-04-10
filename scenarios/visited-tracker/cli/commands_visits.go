package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdVisit(args []string) error {
	opts := visitOptions{}
	if err := opts.parse(args); err != nil {
		return err
	}

	if len(opts.files) == 0 && len(opts.fileNotePaths) == 0 {
		return errors.New("at least one file is required")
	}

	allFiles := normalizePathList(append(opts.files, opts.fileNotePaths...))
	if len(allFiles) == 0 {
		return errors.New("at least one file is required")
	}

	campaignID, err := a.resolveCampaignID(opts.auto, opts.jsonOutput)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"files": allFiles,
	}

	if noteMap := buildFileNotes(allFiles, opts.note, opts.fileNotePaths, opts.fileNoteNotes); len(noteMap) > 0 {
		payload["file_notes"] = noteMap
	}
	if opts.context != "" {
		payload["context"] = opts.context
	}
	if opts.agent != "" {
		payload["agent"] = opts.agent
	}
	if opts.conversation != "" {
		payload["conversation_id"] = opts.conversation
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns/"+campaignID+"/visit"), nil, payload)
	if err != nil {
		return err
	}

	if opts.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var response visitResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse visit response: %w", err)
	}

	fmt.Println("Visit recorded successfully")
	fmt.Printf("Files recorded: %d\n", response.Recorded)
	if opts.note != "" {
		fmt.Printf("Note: %s\n", opts.note)
	} else if len(opts.fileNotePaths) > 0 {
		fmt.Println("Notes: file-specific")
	}

	for _, file := range response.Files {
		fmt.Printf("  %s\n", file)
	}
	if len(response.Unmatched) > 0 {
		fmt.Printf("Unmatched patterns: %s\n", strings.Join(response.Unmatched, ", "))
	}

	return nil
}

func (a *App) cmdAdjustVisit(args []string) error {
	opts := adjustVisitOptions{}
	if err := opts.parse(args); err != nil {
		return err
	}

	campaignID, err := a.resolveCampaignID(opts.auto, opts.jsonOutput)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"file_id": opts.fileID,
		"action":  opts.action,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns/"+campaignID+"/adjust-visit"), nil, payload)
	if err != nil {
		return err
	}

	if opts.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var response adjustVisitResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse adjust-visit response: %w", err)
	}

	fmt.Printf("Visit count %sed for file %s\n", response.Action, response.FileID)
	fmt.Printf("Current visits: %d\n", response.VisitCount)
	return nil
}

func (a *App) cmdExclude(args []string) error {
	opts := excludeOptions{}
	if err := opts.parse(args); err != nil {
		return err
	}

	if len(opts.files) == 0 && len(opts.fileReasonPaths) == 0 {
		return errors.New("at least one file is required")
	}

	allFiles := normalizePathList(append(opts.files, opts.fileReasonPaths...))
	if len(allFiles) == 0 {
		return errors.New("at least one file is required")
	}

	campaignID, err := a.resolveCampaignID(opts.auto, opts.jsonOutput)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"files":    allFiles,
		"excluded": true,
	}

	if strings.TrimSpace(opts.reason) != "" {
		reason := strings.TrimSpace(opts.reason)
		payload["reason"] = reason
	}
	if noteMap := buildFileNotes(allFiles, opts.reason, opts.fileReasonPaths, opts.fileReasonNotes); len(noteMap) > 0 {
		payload["file_notes"] = noteMap
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns/"+campaignID+"/files/exclude"), nil, payload)
	if err != nil {
		return err
	}

	if opts.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var response excludeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse exclude response: %w", err)
	}

	fmt.Printf("Excluded %d file(s)\n", response.ExcludedCount)
	if opts.reason != "" {
		fmt.Printf("Reason: %s\n", opts.reason)
	} else if len(opts.fileReasonPaths) > 0 {
		fmt.Println("Reasons: file-specific")
	}

	for _, file := range response.Files {
		fmt.Printf("  %s\n", file)
	}
	if len(response.Unmatched) > 0 {
		fmt.Printf("Unmatched patterns: %s\n", strings.Join(response.Unmatched, ", "))
	}

	return nil
}

type visitOptions struct {
	files         []string
	context       string
	agent         string
	conversation  string
	note          string
	fileNotePaths []string
	fileNoteNotes []string
	auto          campaignAutoOptions
	jsonOutput    bool
}

func (o *visitOptions) parse(args []string) error {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--context":
			if i+1 >= len(args) {
				return errMissingFlagValue("--context")
			}
			o.context = strings.TrimSpace(args[i+1])
			i++
		case "--agent":
			if i+1 >= len(args) {
				return errMissingFlagValue("--agent")
			}
			o.agent = strings.TrimSpace(args[i+1])
			i++
		case "--conversation":
			if i+1 >= len(args) {
				return errMissingFlagValue("--conversation")
			}
			o.conversation = strings.TrimSpace(args[i+1])
			i++
		case "--note":
			if i+1 >= len(args) {
				return errMissingFlagValue("--note")
			}
			o.note = strings.TrimSpace(args[i+1])
			i++
		case "--file-note":
			if i+2 >= len(args) {
				return errors.New("--file-note requires PATH and NOTE")
			}
			o.fileNotePaths = append(o.fileNotePaths, strings.TrimSpace(args[i+1]))
			o.fileNoteNotes = append(o.fileNoteNotes, strings.TrimSpace(args[i+2]))
			i += 2
		case "--location":
			if i+1 >= len(args) {
				return errMissingFlagValue("--location")
			}
			o.auto.location = strings.TrimSpace(args[i+1])
			i++
		case "--tag":
			if i+1 >= len(args) {
				return errMissingFlagValue("--tag")
			}
			o.auto.tag = strings.TrimSpace(args[i+1])
			i++
		case "--pattern":
			if i+1 >= len(args) {
				return errMissingFlagValue("--pattern")
			}
			o.auto.pattern = strings.TrimSpace(args[i+1])
			i++
		case "--name":
			if i+1 >= len(args) {
				return errMissingFlagValue("--name")
			}
			o.auto.name = strings.TrimSpace(args[i+1])
			i++
		case "--json":
			o.jsonOutput = true
		case "--":
			o.files = append(o.files, args[i+1:]...)
			return nil
		default:
			o.files = append(o.files, strings.TrimSpace(args[i]))
		}
	}
	return nil
}

type adjustVisitOptions struct {
	fileID     string
	action     string
	auto       campaignAutoOptions
	jsonOutput bool
}

func (o *adjustVisitOptions) parse(args []string) error {
	o.action = "increment"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file-id":
			if i+1 >= len(args) {
				return errMissingFlagValue("--file-id")
			}
			o.fileID = strings.TrimSpace(args[i+1])
			i++
		case "--action":
			if i+1 >= len(args) {
				return errMissingFlagValue("--action")
			}
			o.action = strings.TrimSpace(args[i+1])
			i++
		case "--increment":
			o.action = "increment"
		case "--decrement":
			o.action = "decrement"
		case "--location":
			if i+1 >= len(args) {
				return errMissingFlagValue("--location")
			}
			o.auto.location = strings.TrimSpace(args[i+1])
			i++
		case "--tag":
			if i+1 >= len(args) {
				return errMissingFlagValue("--tag")
			}
			o.auto.tag = strings.TrimSpace(args[i+1])
			i++
		case "--pattern":
			if i+1 >= len(args) {
				return errMissingFlagValue("--pattern")
			}
			o.auto.pattern = strings.TrimSpace(args[i+1])
			i++
		case "--name":
			if i+1 >= len(args) {
				return errMissingFlagValue("--name")
			}
			o.auto.name = strings.TrimSpace(args[i+1])
			i++
		case "--json":
			o.jsonOutput = true
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if strings.TrimSpace(o.fileID) == "" {
		return errors.New("--file-id is required")
	}
	if o.action != "increment" && o.action != "decrement" {
		return errors.New("--action must be increment or decrement")
	}
	return nil
}

type excludeOptions struct {
	files           []string
	reason          string
	fileReasonPaths []string
	fileReasonNotes []string
	auto            campaignAutoOptions
	jsonOutput      bool
}

func (o *excludeOptions) parse(args []string) error {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--reason":
			if i+1 >= len(args) {
				return errMissingFlagValue("--reason")
			}
			o.reason = strings.TrimSpace(args[i+1])
			i++
		case "--file-reason":
			if i+2 >= len(args) {
				return errors.New("--file-reason requires PATH and REASON")
			}
			o.fileReasonPaths = append(o.fileReasonPaths, strings.TrimSpace(args[i+1]))
			o.fileReasonNotes = append(o.fileReasonNotes, strings.TrimSpace(args[i+2]))
			i += 2
		case "--location":
			if i+1 >= len(args) {
				return errMissingFlagValue("--location")
			}
			o.auto.location = strings.TrimSpace(args[i+1])
			i++
		case "--tag":
			if i+1 >= len(args) {
				return errMissingFlagValue("--tag")
			}
			o.auto.tag = strings.TrimSpace(args[i+1])
			i++
		case "--pattern":
			if i+1 >= len(args) {
				return errMissingFlagValue("--pattern")
			}
			o.auto.pattern = strings.TrimSpace(args[i+1])
			i++
		case "--name":
			if i+1 >= len(args) {
				return errMissingFlagValue("--name")
			}
			o.auto.name = strings.TrimSpace(args[i+1])
			i++
		case "--json":
			o.jsonOutput = true
		case "--":
			o.files = append(o.files, args[i+1:]...)
			return nil
		default:
			o.files = append(o.files, strings.TrimSpace(args[i]))
		}
	}
	return nil
}

func buildFileNotes(allFiles []string, globalNote string, perPaths []string, perNotes []string) map[string]string {
	result := make(map[string]string)
	globalNote = strings.TrimSpace(globalNote)
	if globalNote != "" {
		for _, file := range allFiles {
			result[file] = globalNote
		}
	}
	for i, path := range perPaths {
		if i >= len(perNotes) {
			break
		}
		note := strings.TrimSpace(perNotes[i])
		if strings.TrimSpace(path) == "" || note == "" {
			continue
		}
		result[strings.TrimSpace(path)] = note
	}
	return result
}
