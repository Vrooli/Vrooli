package main

import (
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdPromptsMap(args []string) error {
	fs := flag.NewFlagSet("prompts map", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.getV1("/prompts/map", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptMapResponse](body)
	if err != nil {
		return err
	}
	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No prompt bindings found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("prompts", "skills"),
		})
		return nil
	}

	areaCounts := map[string]int{}
	for _, item := range response.Items {
		areaCounts[item.Area]++
	}
	printSection("Summary")
	fmt.Printf("  Found %d prompt binding(s)\n", len(response.Items))
	keys := make([]string, 0, len(areaCounts))
	for key := range areaCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  %s: %d\n", key, areaCounts[key])
	}

	printSection("Results")
	for _, item := range response.Items {
		fmt.Printf("  %s\n", item.Trigger)
		fmt.Printf("    Skill: %s\n", item.SkillID)
		if strings.TrimSpace(item.Kind) != "" {
			fmt.Printf("    Kind: %s\n", item.Kind)
		}
		if strings.TrimSpace(item.Mode) != "" {
			fmt.Printf("    Mode: %s\n", item.Mode)
		}
		if strings.TrimSpace(item.Operation) != "" {
			fmt.Printf("    Operation: %s\n", item.Operation)
		}
		fmt.Printf("    Purpose: %s\n", item.Purpose)
		fmt.Println()
	}

	first := response.Items[0]
	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "skills"),
		cliCommand("prompts", "skill-get", "--id", first.SkillID),
		cliCommand("prompts", "preview", "--id", first.SkillID, "--vars", "ITEM_TITLE=Example,ITEM_FOLDER=scenarios/example"),
	})
	return nil
}

func (a *App) cmdPromptsSkills(args []string) error {
	fs := flag.NewFlagSet("prompts skills", flag.ContinueOnError)
	contains := fs.String("contains", "", "Filter by substring in ID or name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.getV1("/prompts/skills", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptSkillsResponse](body)
	if err != nil {
		return err
	}
	filtered := response.Items
	if needle := strings.ToLower(strings.TrimSpace(*contains)); needle != "" {
		next := make([]PromptSkillSummary, 0, len(filtered))
		for _, item := range filtered {
			if strings.Contains(strings.ToLower(item.ID), needle) ||
				strings.Contains(strings.ToLower(item.Name), needle) {
				next = append(next, item)
			}
		}
		filtered = next
	}

	if len(filtered) == 0 {
		printSection("Summary")
		fmt.Println("  No prompt skills found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("prompts", "map"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d prompt skill(s)\n", len(filtered))
	printSection("Results")
	for _, item := range filtered {
		fmt.Printf("  %s\n", item.ID)
		fmt.Printf("    Name: %s\n", item.Name)
		fmt.Printf("    Trigger Paths: %d\n", item.TriggerCount)
		if strings.TrimSpace(item.UpdatedAt) != "" {
			fmt.Printf("    Updated: %s\n", item.UpdatedAt)
		}
		fmt.Printf("    Draft: %v\n", item.Draft)
		if strings.TrimSpace(item.Description) != "" {
			fmt.Printf("    Description: %s\n", item.Description)
		}
		fmt.Println()
	}

	first := filtered[0]
	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "skill-get", "--id", first.ID),
		cliCommand("prompts", "skill-versions", "--id", first.ID),
		cliCommand("prompts", "preview", "--id", first.ID, "--vars", "ITEM_TITLE=Example,ITEM_FOLDER=scenarios/example"),
	})
	return nil
}

func (a *App) cmdPromptsSkillGet(args []string) error {
	fs := flag.NewFlagSet("prompts skill-get", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Skill ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: prompts skill-get --id ID [--json]\n\n%s", err)
	}
	skillID := strings.TrimSpace(*idFlag)

	body, err := a.getV1("/prompts/skills/"+skillID, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptSkillResponse](body)
	if err != nil {
		return err
	}
	item := response.Item
	printSection("Summary")
	fmt.Printf("  %s (%s)\n", item.ID, item.Name)
	printSection("Details")
	fmt.Printf("  Trigger Paths: %d\n", item.TriggerCount)
	fmt.Printf("  Draft: %v\n", item.Draft)
	if strings.TrimSpace(item.DefaultScope) != "" {
		fmt.Printf("  Default Scope: %s\n", item.DefaultScope)
	}
	if strings.TrimSpace(item.ImpactSummary) != "" {
		fmt.Printf("  Impact: %s\n", item.ImpactSummary)
	}
	if len(item.RequiredMissing) > 0 {
		fmt.Printf("  Missing Required Vars: %s\n", strings.Join(item.RequiredMissing, ", "))
	}
	if strings.TrimSpace(item.Description) != "" {
		fmt.Printf("  Description: %s\n", item.Description)
	}
	if strings.TrimSpace(item.UpdatedAt) != "" {
		fmt.Printf("  Updated: %s\n", item.UpdatedAt)
	}
	if strings.TrimSpace(item.CurrentContent) != "" {
		printSection("Prompt")
		fmt.Println(item.CurrentContent)
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "preview", "--id", item.ID, "--vars", "ITEM_TITLE=Example,ITEM_FOLDER=scenarios/example"),
		cliCommand("prompts", "skill-update", "--id", item.ID, "--data", "'{\"draft\":true}'"),
		cliCommand("prompts", "skill-versions", "--id", item.ID),
	})
	return nil
}

func (a *App) cmdPromptsSkillUpdate(args []string) error {
	fs := flag.NewFlagSet("prompts skill-update", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Skill ID")
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("id", *idFlag, "data", *data); err != nil {
		return fmt.Errorf("usage: prompts skill-update --id ID --data JSON [--json]\n\n%s", err)
	}
	skillID := strings.TrimSpace(*idFlag)
	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	body, err := a.requestV1("PUT", "/prompts/skills/"+skillID, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptSkillResponse](body)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  Updated prompt skill: %s\n", response.Item.ID)
	printSection("What Changed")
	fmt.Printf("  Draft: %v\n", response.Item.Draft)
	if strings.TrimSpace(response.Item.UpdatedAt) != "" {
		fmt.Printf("  Updated: %s\n", response.Item.UpdatedAt)
	}
	if len(response.Item.RequiredMissing) > 0 {
		fmt.Printf("  Missing Required Vars: %s\n", strings.Join(response.Item.RequiredMissing, ", "))
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "skill-get", "--id", response.Item.ID),
		cliCommand("prompts", "skill-versions", "--id", response.Item.ID),
	})
	return nil
}

func (a *App) cmdPromptsSkillVersions(args []string) error {
	fs := flag.NewFlagSet("prompts skill-versions", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Skill ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: prompts skill-versions --id ID [--json]\n\n%s", err)
	}
	skillID := strings.TrimSpace(*idFlag)

	body, err := a.getV1("/prompts/skills/"+skillID+"/versions", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptSkillVersionsResponse](body)
	if err != nil {
		return err
	}
	printSection("Summary")
	fmt.Printf("  Skill: %s\n", response.SkillID)
	fmt.Printf("  Current Version: %d\n", response.Current)
	fmt.Printf("  Stored Versions: %d\n", len(response.Versions))

	if len(response.Versions) > 0 {
		printSection("Results")
		for _, version := range response.Versions {
			fmt.Printf("  v%d\n", version.Version)
			fmt.Printf("    Name: %s\n", version.Name)
			if strings.TrimSpace(version.UpdatedAt) != "" {
				fmt.Printf("    Updated: %s\n", version.UpdatedAt)
			}
			if strings.TrimSpace(version.CreatedBy) != "" {
				fmt.Printf("    Author: %s\n", version.CreatedBy)
			}
			fmt.Println()
		}
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "skill-get", "--id", skillID),
		cliCommand("prompts", "skill-revert", "--id", skillID, "--version", "<version>"),
	})
	return nil
}

func (a *App) cmdPromptsSkillRevert(args []string) error {
	fs := flag.NewFlagSet("prompts skill-revert", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Skill ID")
	versionFlag := fs.Int("version", 0, "Version number to revert to")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: prompts skill-revert --id ID --version VERSION [--json]\n\n%s", err)
	}
	skillID := strings.TrimSpace(*idFlag)
	version := *versionFlag
	if version <= 0 {
		return fmt.Errorf("usage: prompts skill-revert --id ID --version VERSION [--json]\n\nversion must be a positive integer")
	}

	body, err := a.requestV1("POST", "/prompts/skills/"+skillID+"/revert/"+strconv.Itoa(version), nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptSkillResponse](body)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  Reverted prompt skill %s to version %d\n", skillID, version)
	printSection("What Changed")
	if strings.TrimSpace(response.Item.UpdatedAt) != "" {
		fmt.Printf("  Updated: %s\n", response.Item.UpdatedAt)
	}
	if len(response.Item.RequiredMissing) > 0 {
		fmt.Printf("  Missing Required Vars: %s\n", strings.Join(response.Item.RequiredMissing, ", "))
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "skill-get", "--id", skillID),
		cliCommand("prompts", "preview", "--id", skillID, "--vars", "ITEM_TITLE=Example,ITEM_FOLDER=scenarios/example"),
	})
	return nil
}

func (a *App) cmdPromptsPreview(args []string) error {
	fs := flag.NewFlagSet("prompts preview", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Skill ID")
	withScope := fs.Bool("with-scope", false, "Include prompt scope metadata")
	varsCSV := fs.String("vars", "", "Comma-separated variables (KEY=VALUE)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: prompts preview --id ID [--vars KEY=VALUE,...] [--with-scope] [--json]\n\n%s", err)
	}
	skillID := strings.TrimSpace(*idFlag)
	vars, err := parseKVCSV(*varsCSV)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"skill_id":   skillID,
		"variables":  vars,
		"with_scope": *withScope,
	}
	body, err := a.requestV1("POST", "/prompts/preview", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptPreviewResponse](body)
	if err != nil {
		return err
	}
	printSection("Summary")
	fmt.Printf("  Rendered prompt for skill: %s\n", response.SkillID)
	fmt.Printf("  With Scope: %v\n", response.WithScope)
	fmt.Printf("  Variables: %d\n", len(response.Variables))
	printSection("Prompt")
	fmt.Println(response.Prompt)
	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "skill-get", "--id", response.SkillID),
		cliCommand("prompts", "simulate", "--kind", "idea", "--mode", "clarify", "--item-title", "Example", "--item-folder", "scenarios/example"),
	})
	return nil
}

func (a *App) cmdPromptsSimulate(args []string) error {
	fs := flag.NewFlagSet("prompts simulate", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Workload kind")
	mode := fs.String("mode", "", "Research mode (clarify|suggest|enhance|research)")
	operation := fs.String("operation", "", "Operation mode (generator|improver)")
	itemName := fs.String("item-name", "", "Backlog item name")
	itemTitle := fs.String("item-title", "", "Backlog item title")
	itemDescription := fs.String("item-description", "", "Backlog item description")
	itemStatus := fs.String("item-status", "", "Backlog item status")
	itemPriority := fs.String("item-priority", "", "Backlog item priority")
	itemTags := fs.String("item-tags", "", "Backlog item tags")
	itemFolder := fs.String("item-folder", "", "Backlog item folder path")
	researchTarget := fs.String("research-target", "", "Research target (idea|fix|execute|unspecified)")
	varsCSV := fs.String("vars", "", "Comma-separated variables (KEY=VALUE)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("kind", *kindFlag); err != nil {
		return fmt.Errorf("usage: prompts simulate --kind KIND [--mode MODE] [--operation OP] [--item-title TITLE] [--item-folder PATH] [--vars KEY=VALUE,...] [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	vars, err := parseKVCSV(*varsCSV)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"kind": kind,
	}
	if value := strings.TrimSpace(*mode); value != "" {
		payload["mode"] = value
	}
	if value := strings.TrimSpace(*operation); value != "" {
		payload["operation"] = value
	}
	if value := strings.TrimSpace(*itemName); value != "" {
		payload["item_name"] = value
	}
	if value := strings.TrimSpace(*itemTitle); value != "" {
		payload["item_title"] = value
	}
	if value := strings.TrimSpace(*itemDescription); value != "" {
		payload["item_description"] = value
	}
	if value := strings.TrimSpace(*itemStatus); value != "" {
		payload["item_status"] = value
	}
	if value := strings.TrimSpace(*itemPriority); value != "" {
		payload["item_priority"] = value
	}
	if value := strings.TrimSpace(*itemTags); value != "" {
		payload["item_tags"] = value
	}
	if value := strings.TrimSpace(*itemFolder); value != "" {
		payload["item_folder"] = value
	}
	if value := strings.TrimSpace(*researchTarget); value != "" {
		payload["research_target"] = value
	}
	if len(vars) > 0 {
		payload["variables"] = vars
	}

	body, err := a.requestV1("POST", "/prompts/simulate", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptSimulateResponse](body)
	if err != nil {
		return err
	}
	printSection("Summary")
	fmt.Printf("  Simulated prompt for %s (%s)\n", response.Kind, response.Area)
	fmt.Printf("  Skill: %s\n", response.SkillID)
	if strings.TrimSpace(response.Mode) != "" {
		fmt.Printf("  Mode: %s\n", response.Mode)
	}
	if strings.TrimSpace(response.Operation) != "" {
		fmt.Printf("  Operation: %s\n", response.Operation)
	}
	fmt.Printf("  Variables: %d\n", len(response.Variables))
	printSection("Prompt")
	fmt.Println(response.Prompt)
	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "skill-get", "--id", response.SkillID),
		cliCommand("prompts", "preview", "--id", response.SkillID, "--vars", "ITEM_TITLE=Example,ITEM_FOLDER=scenarios/example"),
	})
	return nil
}

func parseKVCSV(raw string) (map[string]string, error) {
	result := map[string]string{}
	for _, entry := range cliutil.ParseCSV(raw) {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid variable %q (expected KEY=VALUE)", trimmed)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid variable %q (key cannot be empty)", trimmed)
		}
		result[key] = strings.TrimSpace(parts[1])
	}
	return result, nil
}

func printPromptTraceSummary(header, subject string, trace PromptTrace) {
	printSection(header)
	fmt.Printf("  %s\n", subject)
	fmt.Printf("  Skill ID: %s\n", trace.SkillID)
	if strings.TrimSpace(trace.Purpose) != "" {
		fmt.Printf("  Purpose: %s\n", trace.Purpose)
	}
	if strings.TrimSpace(trace.CapturedAt) != "" {
		fmt.Printf("  Captured: %s\n", trace.CapturedAt)
	}
	fmt.Printf("  Used Fallback: %v\n", trace.UsedFallback)
	fmt.Printf("  Variables: %d\n", len(trace.Variables))
	if strings.TrimSpace(trace.Prompt) != "" {
		printSection("Prompt")
		fmt.Println(trace.Prompt)
	}
}
