package voice

import (
	"fmt"
	"os"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `voice` subcommand group covering voice config, wake
// word templates, and speaker verification surfaces under /voice.
// Audio transcription and streaming endpoints are deliberately omitted: they
// require multipart uploads or WebSocket streams, which a thin CLI shouldn't
// orchestrate on behalf of the API.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "voice",
		Description: "Voice configuration, wake word, and speaker verification",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "config-get", Aliases: []string{"config"}, Description: "Show voice config", Run: func(args []string) error { return runConfigGet(core, args) }},
			{Name: "config-set", Description: "Update voice config (--body-file PATH)", Run: func(args []string) error { return runConfigSet(core, args) }},
			{Name: "wakeword-get", Description: "Show wake word config", Run: func(args []string) error { return runWakewordGet(core, args) }},
			{Name: "wakeword-set", Description: "Upload/update a wake word template (--body-file PATH)", Run: func(args []string) error { return runWakewordSet(core, args) }},
			{Name: "wakeword-delete", Description: "Delete the wake word template", Run: func(args []string) error { return runWakewordDelete(core, args) }},
			{Name: "speaker-config-get", Description: "Show speaker verification config", Run: func(args []string) error { return runSpeakerConfigGet(core, args) }},
			{Name: "speaker-config-set", Description: "Update speaker verification config (--body-file PATH)", Run: func(args []string) error { return runSpeakerConfigSet(core, args) }},
			{Name: "speaker-status", Description: "Show speaker verification status", Run: func(args []string) error { return runSpeakerStatus(core, args) }},
			{Name: "speaker-profiles", Description: "List speaker verification profiles", Run: func(args []string) error { return runSpeakerProfiles(core, args) }},
			{Name: "speaker-enroll", Description: "Enroll a new speaker profile (--body-file PATH)", Run: func(args []string) error { return runSpeakerEnroll(core, args) }},
			{Name: "speaker-clear", Description: "Clear the bound speaker profile for this client", Run: func(args []string) error { return runSpeakerClear(core, args) }},
			{Name: "speaker-remove", Description: "Remove a speaker profile (--body-file PATH)", Run: func(args []string) error { return runSpeakerRemove(core, args) }},
			{Name: "speaker-delete", Description: "Hard-delete a speaker profile (--body-file PATH)", Run: func(args []string) error { return runSpeakerDelete(core, args) }},
		},
	}
}

func runConfigGet(core *cliapp.ScenarioApp, args []string) error {
	return runGet(core, args, "voice config-get", "/voice/config", "Voice config")
}

func runConfigSet(core *cliapp.ScenarioApp, args []string) error {
	return runBodyPut(core, args, "voice config-set", "/voice/config", "Updated voice config",
		fmt.Sprintf("%s voice config-get", support.CLIName))
}

func runWakewordGet(core *cliapp.ScenarioApp, args []string) error {
	return runGet(core, args, "voice wakeword-get", "/voice/wakeword", "Wake word config")
}

func runWakewordSet(core *cliapp.ScenarioApp, args []string) error {
	return runBodyPut(core, args, "voice wakeword-set", "/voice/wakeword", "Updated wake word template",
		fmt.Sprintf("%s voice wakeword-get", support.CLIName))
}

func runWakewordDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice wakeword-delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if _, err := core.Request("DELETE", "/voice/wakeword", nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Deleted wake word template"},
		NextCommand: []string{fmt.Sprintf("%s voice wakeword-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSpeakerConfigGet(core *cliapp.ScenarioApp, args []string) error {
	return runGet(core, args, "voice speaker-config-get", "/voice/speaker/config", "Speaker verification config")
}

func runSpeakerConfigSet(core *cliapp.ScenarioApp, args []string) error {
	return runBodyPut(core, args, "voice speaker-config-set", "/voice/speaker/config", "Updated speaker config",
		fmt.Sprintf("%s voice speaker-config-get", support.CLIName))
}

func runSpeakerStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/voice/speaker/status", nil)
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
		Status: []string{fmt.Sprintf("Speaker verification: %s", status)},
		Triage: []cliapp.TriageGroup{{Heading: "Findings", Items: support.MapRows(payload)}},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runSpeakerProfiles(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-profiles")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/voice/speaker/profiles", nil)
	if err != nil {
		return err
	}
	var profiles []support.VoiceProfile
	if err := support.Decode(body, &profiles); err != nil {
		return err
	}

	rows := make([]string, 0, len(profiles))
	for _, p := range profiles {
		rows = append(rows, fmt.Sprintf("%s | speaker=%s | status=%s",
			support.ShortID(p.ID), p.Speaker, p.Status))
	}
	if len(rows) == 0 {
		rows = []string{"(no profiles)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Speaker profiles: %d", len(profiles))},
		ResultsHeading: "Profiles",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSpeakerEnroll(core *cliapp.ScenarioApp, args []string) error {
	return runBodyMutation(core, args, "voice speaker-enroll", "POST", "/voice/speaker/enroll", "Enrolled speaker profile",
		fmt.Sprintf("%s voice speaker-profiles", support.CLIName))
}

func runSpeakerClear(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-clear")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if _, err := core.Request("DELETE", "/voice/speaker/profile", nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Cleared speaker profile binding for this client"},
		NextCommand: []string{fmt.Sprintf("%s voice speaker-status", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSpeakerRemove(core *cliapp.ScenarioApp, args []string) error {
	return runBodyMutation(core, args, "voice speaker-remove", "POST", "/voice/speaker/profile/remove", "Removed speaker profile",
		fmt.Sprintf("%s voice speaker-profiles", support.CLIName))
}

func runSpeakerDelete(core *cliapp.ScenarioApp, args []string) error {
	return runBodyMutation(core, args, "voice speaker-delete", "POST", "/voice/speaker/profile/delete", "Deleted speaker profile",
		fmt.Sprintf("%s voice speaker-profiles", support.CLIName))
}

// runGet is a small helper that GETs a config surface and renders it as a map.
func runGet(core *cliapp.ScenarioApp, args []string, name, path, heading string) error {
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

// runBodyPut PUTs a JSON body-file payload and returns a mutation report.
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

// runBodyMutation POSTs a JSON body-file payload and renders changes.
func runBodyMutation(core *cliapp.ScenarioApp, args []string, name, method, path, result, nextCmd string) error {
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
	body, err := core.Request(method, path, nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	_ = support.Decode(body, &resp)

	report := cliapp.MutationReport{
		Result:      []string{result},
		Changes:     support.MapRows(resp),
		NextCommand: []string{nextCmd},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
