package tracking

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"visited-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "visited-tracker"

func Register(core *cliapp.ScenarioApp, campaignID *string) cliapp.CommandGroup {
	resolver := support.Resolver{Core: core, CampaignID: campaignID}
	return cliapp.CommandGroup{
		Title: "Tracking",
		Commands: []cliapp.Command{
			{Name: "visit", NeedsAPI: true, Description: "Record file visits", Run: func(args []string) error { return runVisit(core, &resolver, args) }},
			{Name: "adjust-visit", NeedsAPI: true, Description: "Adjust a file visit count", Run: func(args []string) error { return runAdjustVisit(core, &resolver, args) }},
			{Name: "exclude", NeedsAPI: true, Description: "Bulk-exclude files from a campaign", Run: func(args []string) error { return runExclude(core, &resolver, args) }},
			{Name: "sync", NeedsAPI: true, Description: "Sync campaign file structure", Run: func(args []string) error { return runSync(core, &resolver, args) }},
		},
	}
}

func runVisit(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	opts := visitOptions{}
	if err := opts.parse(args); err != nil {
		return err
	}
	if len(opts.files) == 0 && len(opts.fileNotePaths) == 0 {
		return errors.New("at least one file is required")
	}
	allFiles := support.NormalizePathList(append(opts.files, opts.fileNotePaths...))
	if len(allFiles) == 0 {
		return errors.New("at least one file is required")
	}
	campaignID, err := resolver.ResolveCampaignID(opts.auto, opts.jsonOutput)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{"files": allFiles}
	if noteMap := support.BuildFileNotes(allFiles, opts.note, opts.fileNotePaths, opts.fileNoteNotes); len(noteMap) > 0 {
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
	body, err := core.Request("POST", "/campaigns/"+campaignID+"/visit", nil, payload)
	if err != nil {
		return err
	}
	var response support.VisitResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse visit response: %w", err)
	}
	report := cliapp.MutationReport{
		Result:      []string{"Visit recorded successfully", fmt.Sprintf("Files recorded: %d", response.Recorded)},
		Changes:     append(renderFiles("Files", response.Files), renderFiles("Unmatched patterns", response.Unmatched)...),
		NextCommand: []string{cliName + " coverage --campaign-id " + campaignID, cliName + " least-visited --campaign-id " + campaignID},
	}
	if opts.note != "" {
		report.Changes = append([]string{"Note: " + opts.note}, report.Changes...)
	}
	if opts.jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runAdjustVisit(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	opts := adjustVisitOptions{}
	if err := opts.parse(args); err != nil {
		return err
	}
	campaignID, err := resolver.ResolveCampaignID(opts.auto, opts.jsonOutput)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/campaigns/"+campaignID+"/adjust-visit", nil, map[string]interface{}{"file_id": opts.fileID, "action": opts.action})
	if err != nil {
		return err
	}
	var response support.AdjustVisitResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse adjust-visit response: %w", err)
	}
	report := cliapp.MutationReport{
		Result:      []string{"Visit count updated", "File ID: " + response.FileID},
		Changes:     []string{fmt.Sprintf("Action: %s", response.Action), fmt.Sprintf("Current visits: %d", response.VisitCount)},
		NextCommand: []string{cliName + " files get-by-path --path <file-path>"},
	}
	if opts.jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runExclude(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	opts := excludeOptions{}
	if err := opts.parse(args); err != nil {
		return err
	}
	if len(opts.files) == 0 && len(opts.fileReasonPaths) == 0 {
		return errors.New("at least one file is required")
	}
	allFiles := support.NormalizePathList(append(opts.files, opts.fileReasonPaths...))
	if len(allFiles) == 0 {
		return errors.New("at least one file is required")
	}
	campaignID, err := resolver.ResolveCampaignID(opts.auto, opts.jsonOutput)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{"files": allFiles, "excluded": true}
	if strings.TrimSpace(opts.reason) != "" {
		payload["reason"] = strings.TrimSpace(opts.reason)
	}
	if noteMap := support.BuildFileNotes(allFiles, opts.reason, opts.fileReasonPaths, opts.fileReasonNotes); len(noteMap) > 0 {
		payload["file_notes"] = noteMap
	}
	body, err := core.Request("POST", "/campaigns/"+campaignID+"/files/exclude", nil, payload)
	if err != nil {
		return err
	}
	var response support.ExcludeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse exclude response: %w", err)
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Excluded %d file(s)", response.ExcludedCount)},
		Changes:     append(renderFiles("Files", response.Files), renderFiles("Unmatched patterns", response.Unmatched)...),
		NextCommand: []string{cliName + " most-stale --campaign-id " + campaignID, cliName + " coverage --campaign-id " + campaignID},
	}
	if opts.reason != "" {
		report.Changes = append([]string{"Reason: " + opts.reason}, report.Changes...)
	}
	if opts.jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSync(core *cliapp.ScenarioApp, resolver *support.Resolver, args []string) error {
	fs, jsonOutput, err := support.ParseFlags("sync", args)
	if err != nil {
		return err
	}
	location := fs.String("location", "", "Campaign location")
	tag := fs.String("tag", "", "Campaign tag")
	pattern := fs.String("pattern", "", "File pattern")
	name := fs.String("name", "", "Campaign name")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	campaignID, err := resolver.ResolveCampaignID(support.CampaignAutoOptions{Location: *location, Tag: *tag, Pattern: *pattern, Name: *name}, *jsonOutput)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/campaigns/"+campaignID+"/structure/sync", nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	var response support.SyncResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse sync response: %w", err)
	}
	report := cliapp.MutationReport{
		Result: []string{"Campaign structure synced", "Campaign ID: " + campaignID},
		Changes: []string{
			fmt.Sprintf("Added: %d", response.Added),
			fmt.Sprintf("Removed: %d", response.Removed),
			fmt.Sprintf("Moved: %d", response.Moved),
			fmt.Sprintf("Total: %d", response.Total),
		},
		NextCommand: []string{cliName + " coverage --campaign-id " + campaignID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderFiles(label string, items []string) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, label+": "+item)
	}
	return lines
}

type visitOptions struct {
	files         []string
	context       string
	agent         string
	conversation  string
	note          string
	fileNotePaths []string
	fileNoteNotes []string
	auto          support.CampaignAutoOptions
	jsonOutput    bool
}

func (o *visitOptions) parse(args []string) error {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--context":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--context")
			}
			o.context = strings.TrimSpace(args[i+1])
			i++
		case "--agent":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--agent")
			}
			o.agent = strings.TrimSpace(args[i+1])
			i++
		case "--conversation":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--conversation")
			}
			o.conversation = strings.TrimSpace(args[i+1])
			i++
		case "--note":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--note")
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
				return support.ErrMissingFlagValue("--location")
			}
			o.auto.Location = strings.TrimSpace(args[i+1])
			i++
		case "--tag":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--tag")
			}
			o.auto.Tag = strings.TrimSpace(args[i+1])
			i++
		case "--pattern":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--pattern")
			}
			o.auto.Pattern = strings.TrimSpace(args[i+1])
			i++
		case "--name":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--name")
			}
			o.auto.Name = strings.TrimSpace(args[i+1])
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
	auto       support.CampaignAutoOptions
	jsonOutput bool
}

func (o *adjustVisitOptions) parse(args []string) error {
	o.action = "increment"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file-id":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--file-id")
			}
			o.fileID = strings.TrimSpace(args[i+1])
			i++
		case "--action":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--action")
			}
			o.action = strings.TrimSpace(args[i+1])
			i++
		case "--increment":
			o.action = "increment"
		case "--decrement":
			o.action = "decrement"
		case "--location":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--location")
			}
			o.auto.Location = strings.TrimSpace(args[i+1])
			i++
		case "--tag":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--tag")
			}
			o.auto.Tag = strings.TrimSpace(args[i+1])
			i++
		case "--pattern":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--pattern")
			}
			o.auto.Pattern = strings.TrimSpace(args[i+1])
			i++
		case "--name":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--name")
			}
			o.auto.Name = strings.TrimSpace(args[i+1])
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
	auto            support.CampaignAutoOptions
	jsonOutput      bool
}

func (o *excludeOptions) parse(args []string) error {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--reason":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--reason")
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
				return support.ErrMissingFlagValue("--location")
			}
			o.auto.Location = strings.TrimSpace(args[i+1])
			i++
		case "--tag":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--tag")
			}
			o.auto.Tag = strings.TrimSpace(args[i+1])
			i++
		case "--pattern":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--pattern")
			}
			o.auto.Pattern = strings.TrimSpace(args[i+1])
			i++
		case "--name":
			if i+1 >= len(args) {
				return support.ErrMissingFlagValue("--name")
			}
			o.auto.Name = strings.TrimSpace(args[i+1])
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
