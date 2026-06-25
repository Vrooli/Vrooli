package main

import (
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdPromptsCatalog(args []string) error {
	fs := flag.NewFlagSet("prompts catalog", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.Get("/prompts/catalog", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PromptCatalogResponse](body)
	if err != nil {
		return err
	}
	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No prompt catalog entries found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("prompts", "skills"),
		})
		return nil
	}

	printPromptCatalogSummary(response.Items)
	printPromptCatalogResults(response.Items)
	printCommandListSection("Next Steps", promptCatalogNextSteps(response.Items))
	return nil
}

// printPromptCatalogSummary prints aggregate counts by group and usage type.
func printPromptCatalogSummary(items []PromptCatalogEntry) {
	groupCounts := map[string]int{}
	usageCounts := map[string]int{}
	for _, item := range items {
		groupCounts[item.Group]++
		usageCounts[item.UsageType]++
	}
	printSection("Summary")
	fmt.Printf("  Found %d prompt catalog entries\n", len(items))
	keys := make([]string, 0, len(groupCounts))
	for key := range groupCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  Group %s: %d\n", key, groupCounts[key])
	}
	usageKeys := make([]string, 0, len(usageCounts))
	for key := range usageCounts {
		usageKeys = append(usageKeys, key)
	}
	sort.Strings(usageKeys)
	for _, key := range usageKeys {
		fmt.Printf("  Usage %s: %d\n", key, usageCounts[key])
	}
}

// printPromptCatalogResults prints the detail block for each catalog entry.
func printPromptCatalogResults(items []PromptCatalogEntry) {
	printSection("Results")
	for _, item := range items {
		fmt.Printf("  %s\n", item.Title)
		fmt.Printf("    Group: %s\n", item.Group)
		fmt.Printf("    Usage: %s (%s)\n", item.UsageType, item.SourceType)
		fmt.Printf("    Trigger: %s\n", item.Trigger)
		fmt.Printf("    Runtime Prompt: %s\n", promptCatalogTarget(item))
		if len(item.BacklogKinds) > 0 {
			fmt.Printf("    Backlog Kinds: %s\n", strings.Join(item.BacklogKinds, ", "))
		}
		if len(item.Modes) > 0 {
			fmt.Printf("    Modes: %s\n", strings.Join(item.Modes, ", "))
		}
		if len(item.Operations) > 0 {
			fmt.Printf("    Operations: %s\n", strings.Join(item.Operations, ", "))
		}
		fmt.Printf("    Purpose: %s\n", item.Purpose)
		if len(item.OutputPaths) > 0 {
			fmt.Printf("    Outputs: %s\n", strings.Join(item.OutputPaths, ", "))
		}
		if strings.TrimSpace(item.ExperimentID) != "" {
			fmt.Printf("    Experiment: %s\n", item.ExperimentID)
		}
		fmt.Println()
	}
}

// promptCatalogNextSteps builds the suggested follow-up commands, seeded from
// the first entry that exposes a skill ID.
func promptCatalogNextSteps(items []PromptCatalogEntry) []string {
	nextSteps := []string{cliCommand("prompts", "skills")}
	for _, item := range items {
		if strings.TrimSpace(item.SkillID) != "" {
			nextSteps = append(nextSteps,
				cliCommand("prompts", "skill-get", "--id", item.SkillID),
				cliCommand("prompts", "preview", "--id", item.SkillID, "--vars", "ITEM_TITLE=Example,ITEM_FOLDER=scenarios/example"),
			)
			if item.Group == "backlog" && item.UsageType == "direct_runtime" {
				nextSteps = append(nextSteps,
					cliCommand("prompts", "simulate", "--kind", firstOr(item.BacklogKinds, "idea"), "--mode", firstOr(item.Modes, "workshop"), "--item-title", "Example", "--item-folder", "scenarios/example"),
				)
			}
			break
		}
	}
	return nextSteps
}

func (a *App) cmdPromptsSkills(args []string) error {
	fs := flag.NewFlagSet("prompts skills", flag.ContinueOnError)
	contains := fs.String("contains", "", "Filter by substring in ID or name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.Get("/prompts/skills", nil)
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
			cliCommand("prompts", "catalog"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d prompt skill(s)\n", len(filtered))
	printSection("Results")
	for _, item := range filtered {
		fmt.Printf("  %s\n", item.ID)
		fmt.Printf("    Name: %s\n", item.Name)
		fmt.Printf("    Usage: %s\n", item.UsageType)
		if len(item.Groups) > 0 {
			fmt.Printf("    Groups: %s\n", strings.Join(item.Groups, ", "))
		}
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

	body, err := a.core.Get("/prompts/skills/"+skillID, nil)
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
	fmt.Printf("  Usage: %s\n", item.UsageType)
	if len(item.Groups) > 0 {
		fmt.Printf("  Groups: %s\n", strings.Join(item.Groups, ", "))
	}
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

	body, err := a.core.Request("PUT", "/prompts/skills/"+skillID, nil, payload)
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

	body, err := a.core.Get("/prompts/skills/"+skillID+"/versions", nil)
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

	body, err := a.core.Request("POST", "/prompts/skills/"+skillID+"/revert/"+strconv.Itoa(version), nil, nil)
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
	body, err := a.core.Request("POST", "/prompts/preview", nil, payload)
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
		cliCommand("prompts", "simulate", "--kind", "idea", "--mode", "workshop", "--item-title", "Example", "--item-folder", "scenarios/example"),
	})
	return nil
}

func (a *App) cmdPromptsSimulate(args []string) error {
	fs := flag.NewFlagSet("prompts simulate", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Workload kind")
	mode := fs.String("mode", "", "Backlog prompt mode (workshop|initialize|finalize)")
	itemName := fs.String("item-name", "", "Backlog item name")
	itemTitle := fs.String("item-title", "", "Backlog item title")
	itemDescription := fs.String("item-description", "", "Backlog item description")
	itemStatus := fs.String("item-status", "", "Backlog item status")
	itemPriority := fs.String("item-priority", "", "Backlog item priority")
	itemTags := fs.String("item-tags", "", "Backlog item tags")
	itemFolder := fs.String("item-folder", "", "Backlog item folder path")
	varsCSV := fs.String("vars", "", "Comma-separated variables (KEY=VALUE)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("kind", *kindFlag); err != nil {
		return fmt.Errorf("usage: prompts simulate --kind KIND [--mode MODE] [--item-title TITLE] [--item-folder PATH] [--vars KEY=VALUE,...] [--json]\n\n%s", err)
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
	if len(vars) > 0 {
		payload["variables"] = vars
	}

	body, err := a.core.Request("POST", "/prompts/simulate", nil, payload)
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
	fmt.Printf("  Simulated prompt for %s (%s)\n", response.Kind, response.Group)
	fmt.Printf("  Catalog Entry: %s\n", response.EntryID)
	fmt.Printf("  Usage: %s\n", response.UsageType)
	fmt.Printf("  Skill: %s\n", response.SkillID)
	if strings.TrimSpace(response.Mode) != "" {
		fmt.Printf("  Mode: %s\n", response.Mode)
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

func (a *App) cmdPromptsExperimentResults(args []string) error {
	fs := flag.NewFlagSet("prompts experiment-results", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Experiment ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *idFlag == "" && fs.NArg() > 0 {
		*idFlag = fs.Arg(0)
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: prompts experiment-results <eid> [--json]\n\n%s", err)
	}
	experimentID := strings.TrimSpace(*idFlag)

	body, err := a.core.Get("/prompts/experiments/"+experimentID+"/results", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ExperimentResultsResponse](body)
	if err != nil {
		return err
	}

	printSection("Experiment Results")
	fmt.Printf("  Experiment: %s\n", response.ExperimentID)
	if strings.TrimSpace(response.Status) != "" {
		fmt.Printf("  Status: %s\n", response.Status)
	}
	fmt.Printf("  Total Outcomes: %d\n", response.TotalOutcomes)
	fmt.Printf("  Analyzed: %s\n", response.AnalyzedAt)

	if len(response.Variants) > 0 {
		printSection("Variant Comparison")
		fmt.Printf("  %-20s %8s %8s %8s %10s %12s\n", "Variant", "Runs", "Ready", "NeedWrk", "FixupRate", "AvgDuration")
		fmt.Printf("  %-20s %8s %8s %8s %10s %12s\n", "-------", "----", "-----", "-------", "---------", "-----------")
		for _, v := range response.Variants {
			dur := ""
			if v.AvgDurationSecs > 0 {
				dur = fmt.Sprintf("%.1fs", v.AvgDurationSecs)
			}
			fmt.Printf("  %-20s %8d %8d %8d %9.1f%% %12s\n",
				v.VariantID, v.TotalRuns, v.ReadyCount, v.NeedsWorkCount,
				v.FixupRate*100, dur)
		}
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("prompts", "catalog"),
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

func promptCatalogTarget(item PromptCatalogEntry) string {
	if strings.TrimSpace(item.SkillID) != "" {
		return item.SkillID
	}
	if strings.TrimSpace(item.Builder) != "" {
		return item.Builder
	}
	return "(unknown)"
}

func firstOr(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	if value := strings.TrimSpace(values[0]); value != "" {
		return value
	}
	return fallback
}

func printPromptTraceSummary(header, subject string, trace PromptTrace) {
	printSection(header)
	fmt.Printf("  %s\n", subject)
	if strings.TrimSpace(trace.Purpose) != "" {
		fmt.Printf("  Purpose: %s\n", trace.Purpose)
	}
	if strings.TrimSpace(trace.CapturedAt) != "" {
		fmt.Printf("  Captured: %s\n", trace.CapturedAt)
	}
	if strings.TrimSpace(trace.PromptRevision) != "" {
		fmt.Printf("  Prompt Revision: %s\n", trace.PromptRevision)
	}
	fmt.Printf("  Used Fallback: %v\n", trace.UsedFallback)
	if strings.TrimSpace(trace.ExperimentID) != "" {
		fmt.Printf("  Experiment: %s\n", trace.ExperimentID)
	}
	if strings.TrimSpace(trace.VariantID) != "" {
		fmt.Printf("  Variant: %s\n", trace.VariantID)
	}
	if strings.TrimSpace(trace.Prompt) != "" {
		printSection("Prompt")
		fmt.Println(trace.Prompt)
	}
}
