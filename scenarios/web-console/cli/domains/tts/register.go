package tts

import (
	"fmt"
	"os"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `tts` subcommand group for text-to-speech config,
// status, voices, summarize-config, synthesize, cache lookup, and event post.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "tts",
		Description: "Text-to-speech configuration, status, and synthesis",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "config-get", Aliases: []string{"config"}, Description: "Show TTS config", Run: func(args []string) error { return runConfigGet(core, args) }},
			{Name: "config-set", Description: "Update TTS config (--body-file PATH)", Run: func(args []string) error { return runConfigSet(core, args) }},
			{Name: "status", Description: "Show TTS runtime status", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "voices", Description: "List available TTS voices", Run: func(args []string) error { return runVoices(core, args) }},
			{Name: "summarize-config-get", Description: "Show TTS summarize config", Run: func(args []string) error { return runSummarizeGet(core, args) }},
			{Name: "summarize-config-set", Description: "Update TTS summarize config (--body-file PATH)", Run: func(args []string) error { return runSummarizeSet(core, args) }},
			{Name: "synthesize", Description: "Synthesize speech (--body-file PATH, --output PATH)", Run: func(args []string) error { return runSynthesize(core, args) }},
			{Name: "cache-get", Description: "Fetch a cached synthesis by event id (--output PATH)", Run: func(args []string) error { return runCacheGet(core, args) }},
			{Name: "event", Description: "Post a TTS playback event (--body-file PATH)", Run: func(args []string) error { return runPostEvent(core, args) }},
		},
	}
}

func runConfigGet(core *cliapp.ScenarioApp, args []string) error {
	return runSimpleGet(core, args, "tts config-get", "/tts/config", "TTS config")
}

func runConfigSet(core *cliapp.ScenarioApp, args []string) error {
	return runBodyPut(core, args, "tts config-set", "/tts/config", "Updated TTS config",
		fmt.Sprintf("%s tts config-get", support.CLIName))
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/tts/status", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	status := "unknown"
	if v, ok := payload["status"].(string); ok && v != "" {
		status = v
	}

	report := cliapp.OperationalReport{
		Status: []string{fmt.Sprintf("TTS status: %s", status)},
		Triage: []cliapp.TriageGroup{{Heading: "Findings", Items: support.MapRows(payload)}},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runVoices(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts voices")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/tts/voices", nil)
	if err != nil {
		return err
	}

	// The payload can be either an array or an envelope-wrapped object; fall
	// back to a generic map render when it isn't a flat list.
	var voices []map[string]interface{}
	rows := []string{}
	if err := support.Decode(body, &voices); err == nil && voices != nil {
		for _, v := range voices {
			name, _ := v["name"].(string)
			if name == "" {
				name, _ = v["id"].(string)
			}
			lang, _ := v["language"].(string)
			rows = append(rows, fmt.Sprintf("%s | %s", name, lang))
		}
	}
	if len(rows) == 0 {
		var obj map[string]interface{}
		_ = support.Decode(body, &obj)
		rows = support.MapRows(obj)
	}

	report := cliapp.ListReport{
		Summary:        []string{"TTS voices"},
		ResultsHeading: "Voices",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSummarizeGet(core *cliapp.ScenarioApp, args []string) error {
	return runSimpleGet(core, args, "tts summarize-config-get", "/tts/summarize/config", "TTS summarize config")
}

func runSummarizeSet(core *cliapp.ScenarioApp, args []string) error {
	return runBodyPut(core, args, "tts summarize-config-set", "/tts/summarize/config", "Updated TTS summarize config",
		fmt.Sprintf("%s tts summarize-config-get", support.CLIName))
}

func runSynthesize(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts synthesize")
	bodyFile := fs.String("body-file", "", "Path to JSON request body (required)")
	output := fs.String("output", "", "Write binary audio to this path (defaults to stdout)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/tts/synthesize", nil, payload)
	if err != nil {
		return err
	}

	// Streamed audio is a binary payload; JSON mode surfaces metadata instead.
	if *jsonOutput {
		report := cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Synthesized %d bytes", len(body))},
			Changes: []string{fmt.Sprintf("Output: %s", pathOrStdout(*output))},
		}
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return support.WriteOutput(*output, body)
}

func runCacheGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts cache-get")
	output := fs.String("output", "", "Write binary audio to this path (defaults to stdout)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tts cache-get <event-id> [--output PATH]")
	}
	eventID := fs.Arg(0)

	body, err := core.Get("/tts/cache/"+eventID, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		report := cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Cached synthesis for %s", eventID)},
			ResultsHeading: "Payload",
			Results:        []string{fmt.Sprintf("bytes: %d", len(body)), fmt.Sprintf("output: %s", pathOrStdout(*output))},
		}
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return support.WriteOutput(*output, body)
}

func runPostEvent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts event")
	bodyFile := fs.String("body-file", "", "Path to JSON playback event body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("POST", "/tts/events", nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Recorded TTS event"},
		NextCommand: []string{fmt.Sprintf("%s tts status", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSimpleGet(core *cliapp.ScenarioApp, args []string, name, path, heading string) error {
	fs := support.NewFlagSet(name)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Values",
		Results:        support.MapRows(payload),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runBodyPut(core *cliapp.ScenarioApp, args []string, name, path, result, nextCmd string) error {
	fs := support.NewFlagSet(name)
	bodyFile := fs.String("body-file", "", "Path to a JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", path, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{result},
		NextCommand: []string{nextCmd},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func pathOrStdout(path string) string {
	if path == "" {
		return "(stdout)"
	}
	return path
}
