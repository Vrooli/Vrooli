package tts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectrpc.com/connect"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts/tts_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register builds the `tts` subcommand group covering config, status,
// summarize-config, synthesize, cache, voices, and playback-event posting.
// All RPCs go through the Connect-RPC TTSService.
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
			{Name: "cache-get", Description: "Fetch a cached synthesis (--event-id, --output PATH)", Run: func(args []string) error { return runCacheGet(core, args) }},
			{Name: "event", Description: "Post a TTS playback event (--body-file PATH)", Run: func(args []string) error { return runPostEvent(core, args) }},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) ttsconnect.TTSServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return ttsconnect.NewTTSServiceClient(httpClient, baseURL)
}

// -----------------------------------------------------------------------------
// config
// -----------------------------------------------------------------------------

// configBody mirrors the legacy patch JSON shape ({"autoEnabled":true,...}).
// Pointer fields preserve the "field omitted" vs "field set to zero"
// distinction so config-set --body-file works the same as before.
type configBody struct {
	AutoEnabled *bool    `json:"autoEnabled,omitempty"`
	Backend     *string  `json:"backend,omitempty"`
	KokoroVoice *string  `json:"kokoroVoice,omitempty"`
	KokoroSpeed *float64 `json:"kokoroSpeed,omitempty"`
}

func runConfigGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts config-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	resp, err := newClient(core).GetConfig(context.Background(),
		connect.NewRequest(&ttsv1.GetConfigRequest{}))
	if err != nil {
		return err
	}
	return printConfigReport(resp.Msg.GetConfig(), "TTS config", *jsonOutput)
}

func runConfigSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts config-set")
	bodyFile := fs.String("body-file", "", "Path to a JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var body configBody
	if err := json.Unmarshal(payload, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}
	req := &ttsv1.UpdateConfigRequest{}
	if body.AutoEnabled != nil {
		req.AutoEnabled, req.HasAutoEnabled = *body.AutoEnabled, true
	}
	if body.Backend != nil {
		req.Backend, req.HasBackend = *body.Backend, true
	}
	if body.KokoroVoice != nil {
		req.KokoroVoice, req.HasKokoroVoice = *body.KokoroVoice, true
	}
	if body.KokoroSpeed != nil {
		req.KokoroSpeed, req.HasKokoroSpeed = *body.KokoroSpeed, true
	}

	resp, err := newClient(core).UpdateConfig(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	return printConfigReport(resp.Msg.GetConfig(), "Updated TTS config", *jsonOutput)
}

func printConfigReport(cfg *ttsv1.Config, heading string, asJSON bool) error {
	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Values",
		Results: []string{
			fmt.Sprintf("autoEnabled: %v", cfg.GetAutoEnabled()),
			fmt.Sprintf("backend: %s", cfg.GetBackend()),
			fmt.Sprintf("kokoroVoice: %s", cfg.GetKokoroVoice()),
			fmt.Sprintf("kokoroSpeed: %.2f", cfg.GetKokoroSpeed()),
		},
	}
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// status
// -----------------------------------------------------------------------------

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	resp, err := newClient(core).GetStatus(context.Background(),
		connect.NewRequest(&ttsv1.GetStatusRequest{}))
	if err != nil {
		return err
	}
	st := resp.Msg.GetStatus()
	rows := []string{
		fmt.Sprintf("backend: %s", st.GetConfig().GetBackend()),
		fmt.Sprintf("autoEnabled: %v", st.GetConfig().GetAutoEnabled()),
		fmt.Sprintf("kokoroCapability: %s", st.GetKokoroCapability()),
		fmt.Sprintf("hookRegistered: %v (%s)", st.GetHookRegistered(), st.GetHookCode()),
		fmt.Sprintf("lastPlaybackAt: %s", st.GetLastPlaybackAt()),
	}
	report := cliapp.OperationalReport{
		Status: []string{fmt.Sprintf("Kokoro: %s", st.GetKokoroCapabilityLabel())},
		Triage: []cliapp.TriageGroup{{Heading: "Findings", Items: rows}},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// voices
// -----------------------------------------------------------------------------

func runVoices(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts voices")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	resp, err := newClient(core).ListVoices(context.Background(),
		connect.NewRequest(&ttsv1.ListVoicesRequest{}))
	if err != nil {
		return err
	}
	rows := make([]string, 0, len(resp.Msg.GetVoices()))
	for _, v := range resp.Msg.GetVoices() {
		rows = append(rows, fmt.Sprintf("%s | %s", v.GetId(), v.GetName()))
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

// -----------------------------------------------------------------------------
// summarize config
// -----------------------------------------------------------------------------

type summarizeBody struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	CharThreshold  *int    `json:"charThreshold,omitempty"`
	Level          *string `json:"level,omitempty"`
	Model          *string `json:"model,omitempty"`
	TimeoutSeconds *int    `json:"timeoutSeconds,omitempty"`
}

func runSummarizeGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts summarize-config-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	resp, err := newClient(core).GetSummarizeConfig(context.Background(),
		connect.NewRequest(&ttsv1.GetSummarizeConfigRequest{}))
	if err != nil {
		return err
	}
	return printSummarizeReport(resp.Msg.GetConfig(), "TTS summarize config", *jsonOutput)
}

func runSummarizeSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts summarize-config-set")
	bodyFile := fs.String("body-file", "", "Path to a JSON request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var body summarizeBody
	if err := json.Unmarshal(payload, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}
	req := &ttsv1.UpdateSummarizeConfigRequest{}
	if body.Enabled != nil {
		req.Enabled, req.HasEnabled = *body.Enabled, true
	}
	if body.CharThreshold != nil {
		req.CharThreshold, req.HasCharThreshold = int32(*body.CharThreshold), true
	}
	if body.Level != nil {
		req.Level, req.HasLevel = *body.Level, true
	}
	if body.Model != nil {
		req.Model, req.HasModel = *body.Model, true
	}
	if body.TimeoutSeconds != nil {
		req.TimeoutSeconds, req.HasTimeoutSeconds = int32(*body.TimeoutSeconds), true
	}
	resp, err := newClient(core).UpdateSummarizeConfig(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	return printSummarizeReport(resp.Msg.GetConfig(), "Updated TTS summarize config", *jsonOutput)
}

func printSummarizeReport(cfg *ttsv1.SummarizeConfig, heading string, asJSON bool) error {
	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Values",
		Results: []string{
			fmt.Sprintf("enabled: %v", cfg.GetEnabled()),
			fmt.Sprintf("charThreshold: %d", cfg.GetCharThreshold()),
			fmt.Sprintf("level: %s", cfg.GetLevel()),
			fmt.Sprintf("model: %s", cfg.GetModel()),
			fmt.Sprintf("timeoutSeconds: %d", cfg.GetTimeoutSeconds()),
		},
	}
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// -----------------------------------------------------------------------------
// synthesize
// -----------------------------------------------------------------------------

type synthesizeBody struct {
	Input          string  `json:"input"`
	Voice          string  `json:"voice,omitempty"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
	EventID        string  `json:"event_id,omitempty"`
	Version        string  `json:"version,omitempty"`
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
	var body synthesizeBody
	if err := json.Unmarshal(payload, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}
	resp, err := newClient(core).Synthesize(context.Background(),
		connect.NewRequest(&ttsv1.SynthesizeRequest{
			Input:          body.Input,
			Voice:          body.Voice,
			ResponseFormat: body.ResponseFormat,
			Speed:          body.Speed,
			EventId:        body.EventID,
			Version:        body.Version,
		}))
	if err != nil {
		return err
	}

	if *jsonOutput {
		report := cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Synthesized %d bytes (%s)", len(resp.Msg.GetAudio()), resp.Msg.GetContentType())},
			Changes: []string{fmt.Sprintf("Output: %s", pathOrStdout(*output))},
		}
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return support.WriteOutput(*output, resp.Msg.GetAudio())
}

// -----------------------------------------------------------------------------
// cache-get
// -----------------------------------------------------------------------------

func runCacheGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tts cache-get")
	eventID := fs.String("event-id", "", "Event ID to look up (required)")
	voice := fs.String("voice", "", "Voice ID (defaults server-side to configured kokoroVoice)")
	speed := fs.Float64("speed", 0, "Synthesis speed (defaults to 1.0)")
	version := fs.String("version", "", "Cache variant (defaults to active)")
	output := fs.String("output", "", "Write binary audio to this path (defaults to stdout)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *eventID == "" {
		return fmt.Errorf("--event-id is required")
	}

	resp, err := newClient(core).GetCache(context.Background(),
		connect.NewRequest(&ttsv1.GetCacheRequest{
			EventId: *eventID,
			Voice:   *voice,
			Speed:   *speed,
			Version: *version,
		}))
	if err != nil {
		return err
	}

	if *jsonOutput {
		report := cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Cached synthesis for %s", *eventID)},
			ResultsHeading: "Payload",
			Results: []string{
				fmt.Sprintf("bytes: %d", len(resp.Msg.GetAudio())),
				fmt.Sprintf("contentType: %s", resp.Msg.GetContentType()),
				fmt.Sprintf("output: %s", pathOrStdout(*output)),
			},
		}
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return support.WriteOutput(*output, resp.Msg.GetAudio())
}

// -----------------------------------------------------------------------------
// post event
// -----------------------------------------------------------------------------

type eventBody struct {
	Source    string `json:"source"`
	Stage     string `json:"stage"`
	Backend   string `json:"backend,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Message   string `json:"message,omitempty"`
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
	var body eventBody
	if err := json.Unmarshal(payload, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}
	if _, err := newClient(core).RecordPlaybackEvent(context.Background(),
		connect.NewRequest(&ttsv1.RecordPlaybackEventRequest{
			Event: &ttsv1.PlaybackEvent{
				Source:    body.Source,
				Stage:     body.Stage,
				Backend:   body.Backend,
				SessionId: body.SessionID,
				Message:   body.Message,
			},
		})); err != nil {
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

func pathOrStdout(path string) string {
	if path == "" {
		return "(stdout)"
	}
	return path
}
