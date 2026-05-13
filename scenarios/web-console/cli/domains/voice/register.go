package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"connectrpc.com/connect"

	voicev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice"
	voiceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice/voice_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register builds the `voice` subcommand group covering transcription, voice
// stream config, wake word template, and speaker verification (config,
// status, profiles). All RPCs go through the Connect-RPC VoiceService.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "voice",
		Description: "Voice transcription, stream config, wake word, and speaker verification",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "transcribe", Description: "Transcribe audio bytes via Whisper (--audio-file PATH)", Run: func(args []string) error { return runTranscribe(core, args) }},
			{Name: "config-get", Aliases: []string{"config"}, Description: "Show voice stream config", Run: func(args []string) error { return runConfigGet(core, args) }},
			{Name: "config-set", Description: "Update voice stream config (--body-file PATH)", Run: func(args []string) error { return runConfigSet(core, args) }},
			{Name: "wakeword-get", Description: "Show wake word config", Run: func(args []string) error { return runWakewordGet(core, args) }},
			{Name: "wakeword-set", Description: "Upload/update a wake word template (--body-file PATH)", Run: func(args []string) error { return runWakewordSet(core, args) }},
			{Name: "wakeword-delete", Description: "Delete the wake word template", Run: func(args []string) error { return runWakewordDelete(core, args) }},
			{Name: "speaker-config-get", Description: "Show speaker verification config", Run: func(args []string) error { return runSpeakerConfigGet(core, args) }},
			{Name: "speaker-config-set", Description: "Update speaker verification config (--body-file PATH)", Run: func(args []string) error { return runSpeakerConfigSet(core, args) }},
			{Name: "speaker-status", Description: "Show speaker verification status", Run: func(args []string) error { return runSpeakerStatus(core, args) }},
			{Name: "speaker-profiles", Description: "List speaker verification profiles", Run: func(args []string) error { return runSpeakerProfiles(core, args) }},
			{Name: "speaker-enroll", Description: "Enroll a speaker profile from audio bytes (--audio-file PATH, --body-file PATH)", Run: func(args []string) error { return runSpeakerEnroll(core, args) }},
			{Name: "speaker-clear", Description: "Clear the bound speaker profile for this client", Run: func(args []string) error { return runSpeakerClear(core, args) }},
			{Name: "speaker-remove", Description: "Remove a speaker profile from the active list (--profile-id ID)", Run: func(args []string) error { return runSpeakerRemove(core, args) }},
			{Name: "speaker-delete", Description: "Hard-delete a speaker profile (--profile-id ID)", Run: func(args []string) error { return runSpeakerDelete(core, args) }},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) voiceconnect.VoiceServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return voiceconnect.NewVoiceServiceClient(httpClient, baseURL)
}

// -----------------------------------------------------------------------------
// Transcribe
// -----------------------------------------------------------------------------

func runTranscribe(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice transcribe")
	audioFile := fs.String("audio-file", "", "Path to an audio file (required)")
	language := fs.String("language", "", "BCP-47 language hint (empty = auto-detect)")
	skip := fs.Bool("skip-speaker-verification", false, "Bypass the speaker-verification gate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *audioFile == "" {
		return fmt.Errorf("--audio-file is required")
	}
	audio, err := os.ReadFile(*audioFile)
	if err != nil {
		return fmt.Errorf("read audio file: %w", err)
	}
	resp, err := newClient(core).Transcribe(context.Background(), connect.NewRequest(&voicev1.TranscribeRequest{
		Audio:                   audio,
		Language:                *language,
		SkipSpeakerVerification: *skip,
	}))
	if err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{"Transcription result"},
		ResultsHeading: "Text",
		Results:        []string{resp.Msg.GetText()},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// Stream config
// -----------------------------------------------------------------------------

type streamConfigBody struct {
	FlushIntervalMs   *int32   `json:"flushIntervalMs,omitempty"`
	MinDeltaBytes     *int32   `json:"minDeltaBytes,omitempty"`
	OverlapBytes      *int32   `json:"overlapBytes,omitempty"`
	PersistentMode    *bool    `json:"persistentMode,omitempty"`
	WakeWordEnabled   *bool    `json:"wakeWordEnabled,omitempty"`
	WakeWordThreshold *float64 `json:"wakeWordThreshold,omitempty"`
	SegmentSilenceMs  *int32   `json:"segmentSilenceMs,omitempty"`
}

func runConfigGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice config-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	resp, err := newClient(core).GetStreamConfig(context.Background(),
		connect.NewRequest(&voicev1.GetStreamConfigRequest{}))
	if err != nil {
		return err
	}
	return printStreamConfig(resp.Msg.GetConfig(), "Voice stream config", *jsonOutput)
}

func runConfigSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice config-set")
	bodyFile := fs.String("body-file", "", "Path to a JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var body streamConfigBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("parse body: %w", err)
	}
	req := &voicev1.UpdateStreamConfigRequest{}
	if body.FlushIntervalMs != nil {
		req.FlushIntervalMs = *body.FlushIntervalMs
		req.HasFlushIntervalMs = true
	}
	if body.MinDeltaBytes != nil {
		req.MinDeltaBytes = *body.MinDeltaBytes
		req.HasMinDeltaBytes = true
	}
	if body.OverlapBytes != nil {
		req.OverlapBytes = *body.OverlapBytes
		req.HasOverlapBytes = true
	}
	if body.PersistentMode != nil {
		req.PersistentMode = *body.PersistentMode
		req.HasPersistentMode = true
	}
	if body.WakeWordEnabled != nil {
		req.WakeWordEnabled = *body.WakeWordEnabled
		req.HasWakeWordEnabled = true
	}
	if body.WakeWordThreshold != nil {
		req.WakeWordThreshold = *body.WakeWordThreshold
		req.HasWakeWordThreshold = true
	}
	if body.SegmentSilenceMs != nil {
		req.SegmentSilenceMs = *body.SegmentSilenceMs
		req.HasSegmentSilenceMs = true
	}
	resp, err := newClient(core).UpdateStreamConfig(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	return printStreamConfig(resp.Msg.GetConfig(), "Updated voice stream config", *jsonOutput)
}

func printStreamConfig(cfg *voicev1.StreamConfig, heading string, jsonOutput bool) error {
	rows := []string{
		fmt.Sprintf("flushIntervalMs: %d", cfg.GetFlushIntervalMs()),
		fmt.Sprintf("minDeltaBytes: %d", cfg.GetMinDeltaBytes()),
		fmt.Sprintf("overlapBytes: %d", cfg.GetOverlapBytes()),
		fmt.Sprintf("persistentMode: %t", cfg.GetPersistentMode()),
		fmt.Sprintf("wakeWordEnabled: %t", cfg.GetWakeWordEnabled()),
		fmt.Sprintf("wakeWordThreshold: %.2f", cfg.GetWakeWordThreshold()),
		fmt.Sprintf("segmentSilenceMs: %d", cfg.GetSegmentSilenceMs()),
	}
	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Values",
		Results:        rows,
	}
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// Wake word
// -----------------------------------------------------------------------------

func runWakewordGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice wakeword-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	resp, err := newClient(core).GetWakeWordConfig(context.Background(),
		connect.NewRequest(&voicev1.GetWakeWordConfigRequest{}))
	if err != nil {
		return err
	}
	return printWakeWord(resp.Msg.GetConfig(), "Wake word config", *jsonOutput)
}

func runWakewordSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice wakeword-set")
	bodyFile := fs.String("body-file", "", "Path to a WakeWordTemplate JSON file (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	resp, err := newClient(core).UpdateWakeWordTemplate(context.Background(),
		connect.NewRequest(&voicev1.UpdateWakeWordTemplateRequest{TemplateJson: string(raw)}))
	if err != nil {
		return err
	}
	return printWakeWord(resp.Msg.GetConfig(), "Updated wake word template", *jsonOutput)
}

func runWakewordDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice wakeword-delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if _, err := newClient(core).DeleteWakeWordTemplate(context.Background(),
		connect.NewRequest(&voicev1.DeleteWakeWordTemplateRequest{})); err != nil {
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

func printWakeWord(cfg *voicev1.WakeWordConfig, heading string, jsonOutput bool) error {
	rows := []string{
		fmt.Sprintf("configured: %t", cfg.GetConfigured()),
		fmt.Sprintf("templateBytes: %d", len(cfg.GetTemplateJson())),
	}
	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Values",
		Results:        rows,
	}
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// Speaker config
// -----------------------------------------------------------------------------

type speakerConfigBody struct {
	Enabled                     *bool     `json:"enabled,omitempty"`
	ProfileIDs                  *[]string `json:"profileIds,omitempty"`
	Threshold                   *float64  `json:"threshold,omitempty"`
	Mode                        *string   `json:"mode,omitempty"`
	RejectBehavior              *string   `json:"rejectBehavior,omitempty"`
	FallbackWithoutVerification *bool     `json:"fallbackWithoutVerification,omitempty"`
	ExtractionEnabled           *bool     `json:"extractionEnabled,omitempty"`
}

func runSpeakerConfigGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-config-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	resp, err := newClient(core).GetSpeakerConfig(context.Background(),
		connect.NewRequest(&voicev1.GetSpeakerConfigRequest{}))
	if err != nil {
		return err
	}
	return printSpeakerConfig(resp.Msg.GetConfig(), "Speaker verification config", *jsonOutput)
}

func runSpeakerConfigSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-config-set")
	bodyFile := fs.String("body-file", "", "Path to a JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var body speakerConfigBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("parse body: %w", err)
	}
	req := &voicev1.UpdateSpeakerConfigRequest{}
	if body.Enabled != nil {
		req.Enabled = *body.Enabled
		req.HasEnabled = true
	}
	if body.ProfileIDs != nil {
		req.ProfileIds = *body.ProfileIDs
		req.HasProfileIds = true
	}
	if body.Threshold != nil {
		req.Threshold = *body.Threshold
		req.HasThreshold = true
	}
	if body.Mode != nil {
		req.Mode = *body.Mode
		req.HasMode = true
	}
	if body.RejectBehavior != nil {
		req.RejectBehavior = *body.RejectBehavior
		req.HasRejectBehavior = true
	}
	if body.FallbackWithoutVerification != nil {
		req.FallbackWithoutVerification = *body.FallbackWithoutVerification
		req.HasFallbackWithoutVerification = true
	}
	if body.ExtractionEnabled != nil {
		req.ExtractionEnabled = *body.ExtractionEnabled
		req.HasExtractionEnabled = true
	}
	resp, err := newClient(core).UpdateSpeakerConfig(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	return printSpeakerConfig(resp.Msg.GetConfig(), "Updated speaker config", *jsonOutput)
}

func printSpeakerConfig(cfg *voicev1.SpeakerConfig, heading string, jsonOutput bool) error {
	ids := cfg.GetProfileIds()
	sort.Strings(ids)
	rows := []string{
		fmt.Sprintf("enabled: %t", cfg.GetEnabled()),
		fmt.Sprintf("profileIds: %v", ids),
		fmt.Sprintf("threshold: %.3f", cfg.GetThreshold()),
		fmt.Sprintf("mode: %s", cfg.GetMode()),
		fmt.Sprintf("rejectBehavior: %s", cfg.GetRejectBehavior()),
		fmt.Sprintf("fallbackWithoutVerification: %t", cfg.GetFallbackWithoutVerification()),
		fmt.Sprintf("extractionEnabled: %t", cfg.GetExtractionEnabled()),
	}
	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Values",
		Results:        rows,
	}
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSpeakerStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	resp, err := newClient(core).GetSpeakerStatus(context.Background(),
		connect.NewRequest(&voicev1.GetSpeakerStatusRequest{}))
	if err != nil {
		return err
	}
	st := resp.Msg.GetStatus()
	rows := []string{
		fmt.Sprintf("capability: %s", st.GetCapability()),
		fmt.Sprintf("capabilityLabel: %s", st.GetCapabilityLabel()),
		fmt.Sprintf("resourceReady: %t", st.GetResourceReady()),
		fmt.Sprintf("profileConfigured: %t", st.GetProfileConfigured()),
		fmt.Sprintf("profileExists: %t", st.GetProfileExists()),
		fmt.Sprintf("profileCount: %d", st.GetProfileCount()),
		fmt.Sprintf("checkedAt: %s", st.GetCheckedAt()),
	}
	report := cliapp.OperationalReport{
		Status: []string{fmt.Sprintf("Speaker verification: %s", st.GetCapability())},
		Triage: []cliapp.TriageGroup{{Heading: "Findings", Items: rows}},
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
	resp, err := newClient(core).ListSpeakerProfiles(context.Background(),
		connect.NewRequest(&voicev1.ListSpeakerProfilesRequest{}))
	if err != nil {
		return err
	}
	profiles := resp.Msg.GetProfiles()
	rows := make([]string, 0, len(profiles))
	for _, p := range profiles {
		rows = append(rows, fmt.Sprintf("%s | %s | model=%s | rate=%d",
			support.ShortID(p.GetId()), p.GetDisplayName(), p.GetModelName(), p.GetSampleRate()))
	}
	if len(rows) == 0 {
		rows = []string{"(no profiles)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Speaker profiles: %d", resp.Msg.GetCount())},
		ResultsHeading: "Profiles",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

type enrollBody struct {
	ProfileID   string `json:"profileId"`
	DisplayName string `json:"displayName"`
	Notes       string `json:"notes"`
	AddToActive *bool  `json:"addToActive,omitempty"`
	Enable      *bool  `json:"enable,omitempty"`
}

func runSpeakerEnroll(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-enroll")
	audioFile := fs.String("audio-file", "", "Path to an enrollment audio file (required)")
	bodyFile := fs.String("body-file", "", "Optional JSON metadata (profileId/displayName/notes/addToActive/enable)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *audioFile == "" {
		return fmt.Errorf("--audio-file is required")
	}
	audio, err := os.ReadFile(*audioFile)
	if err != nil {
		return fmt.Errorf("read audio file: %w", err)
	}
	req := &voicev1.EnrollSpeakerProfileRequest{Audio: audio}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, false)
		if err != nil {
			return err
		}
		var body enrollBody
		if err := json.Unmarshal(raw, &body); err != nil {
			return fmt.Errorf("parse body: %w", err)
		}
		req.ProfileId = body.ProfileID
		req.DisplayName = body.DisplayName
		req.Notes = body.Notes
		if body.AddToActive != nil {
			req.AddToActive = *body.AddToActive
			req.HasAddToActive = true
		}
		if body.Enable != nil {
			req.Enable = *body.Enable
			req.HasEnable = true
		}
	}
	resp, err := newClient(core).EnrollSpeakerProfile(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	en := resp.Msg.GetEnrollment()
	changes := []string{
		fmt.Sprintf("profileId: %s", en.GetProfileId()),
		fmt.Sprintf("displayName: %s", en.GetDisplayName()),
		fmt.Sprintf("modelName: %s", en.GetModelName()),
		fmt.Sprintf("audioSeconds: %.2f", en.GetEnrollmentAudioSeconds()),
	}
	report := cliapp.MutationReport{
		Result:      []string{"Enrolled speaker profile"},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s voice speaker-profiles", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSpeakerClear(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-clear")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if _, err := newClient(core).ClearSpeakerProfileBinding(context.Background(),
		connect.NewRequest(&voicev1.ClearSpeakerProfileBindingRequest{})); err != nil {
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
	fs := support.NewFlagSet("voice speaker-remove")
	profileID := fs.String("profile-id", "", "Speaker profile ID (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *profileID == "" {
		return fmt.Errorf("--profile-id is required")
	}
	resp, err := newClient(core).RemoveSpeakerProfile(context.Background(),
		connect.NewRequest(&voicev1.RemoveSpeakerProfileRequest{ProfileId: *profileID}))
	if err != nil {
		return err
	}
	return printSpeakerConfig(resp.Msg.GetConfig(), "Removed speaker profile", *jsonOutput)
}

func runSpeakerDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("voice speaker-delete")
	profileID := fs.String("profile-id", "", "Speaker profile ID (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *profileID == "" {
		return fmt.Errorf("--profile-id is required")
	}
	resp, err := newClient(core).DeleteSpeakerProfile(context.Background(),
		connect.NewRequest(&voicev1.DeleteSpeakerProfileRequest{ProfileId: *profileID}))
	if err != nil {
		return err
	}
	return printSpeakerConfig(resp.Msg.GetConfig(), "Deleted speaker profile", *jsonOutput)
}
