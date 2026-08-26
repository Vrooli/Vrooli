package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"

	_ "modernc.org/sqlite"

	filePreviewH "web-console/handlers/file_preview"
	settingsH "web-console/handlers/settings"
	"web-console/internal/capabilities"
	"web-console/internal/filepreview"
)

func TestProductionCapabilityCheckersCoverCatalogue(t *testing.T) {
	checkers := newCapabilityCheckers("http://127.0.0.1:1", "", "", "", "")
	known := make(map[string]struct{}, len(capabilities.Known))
	for _, def := range capabilities.Known {
		known[def.ID] = struct{}{}
	}
	for id := range checkers {
		if _, ok := known[id]; !ok {
			t.Fatalf("production checker %q is missing from capabilities.Known", id)
		}
	}
	for id := range known {
		if _, ok := checkers[id]; !ok {
			t.Fatalf("catalogue capability %q has no production checker", id)
		}
	}
}

func TestParseCleanupMaxBytesAcceptsHumanAndRawValues(t *testing.T) {
	if got := parseCleanupMaxBytes("1GiB"); got != 1<<30 {
		t.Fatalf("1GiB parsed as %d", got)
	}
	if got := parseCleanupMaxBytes("1073741824"); got != 1<<30 {
		t.Fatalf("raw bytes parsed as %d", got)
	}
	if got := parseCleanupMaxBytes("not-a-size"); got != 0 {
		t.Fatalf("invalid size parsed as %d", got)
	}
}

func TestServerPolicyHookStatusAndDiagnostics(t *testing.T) {
	srv := newFakeTestServer()
	if got := srv.getSummarizeAutoPolicy(); got != defaultSummarizeAutoPolicy() {
		t.Fatalf("default summarize policy = %#v", got)
	}
	srv.SetSummarizeAutoPolicy(SummarizeAutoPolicy{Enabled: true, CharThreshold: 900, Level: "heavy", TimeoutSeconds: 30})
	if got := srv.getSummarizeAutoPolicy(); got.CharThreshold != 900 || !got.Enabled {
		t.Fatalf("set summarize policy = %#v", got)
	}

	t.Setenv("API_PORT", "9911")
	t.Setenv("CLAUDE_PROJECT_SETTINGS", filepath.Join(t.TempDir(), "settings.json"))
	srv.hookAuthToken = "secret-token"
	if ok, code, _, _ := srv.getClaudeHookStatus(); ok || code != "hook_missing_file" {
		t.Fatalf("missing hook status = %v/%s", ok, code)
	}
	settings := `{"hooks":{"Stop":[{"hooks":[{"_id":"web-console-tts","type":"http","url":"http://localhost:9911/api/v1/hooks/stop","headers":{"X-Hook-Token":"secret-token"}}]}]}}`
	if err := os.WriteFile(os.Getenv("CLAUDE_PROJECT_SETTINGS"), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}
	if ok, code, _, _ := srv.getClaudeHookStatus(); !ok || code != "hook_registered" {
		t.Fatalf("registered hook status = %v/%s", ok, code)
	}
	if err := os.WriteFile(os.Getenv("CLAUDE_PROJECT_SETTINGS"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ok, code, _, _ := srv.getClaudeHookStatus(); ok || code != "hook_invalid_json" {
		t.Fatalf("invalid hook status = %v/%s", ok, code)
	}

	if got := srv.expectedClaudeHookURL(); got != "http://localhost:9911/api/v1/hooks/stop" {
		t.Fatalf("expected hook URL = %q", got)
	}
	t.Setenv("API_PORT", "")
	if got := srv.expectedClaudeHookURL(); got != "" {
		t.Fatalf("empty port hook URL = %q", got)
	}
}

func TestHookTokenAndLegacyFilePrimitives(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hook-token.txt")
	if token := loadOrCreateHookToken(path); len(token) != 64 {
		t.Fatalf("generated hook token length = %d", len(token))
	}
	if token := loadOrCreateHookToken(path); len(token) != 64 {
		t.Fatalf("loaded hook token length = %d", len(token))
	}
	if err := os.WriteFile(path, []byte(" saved-token \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loadOrCreateHookToken(path); got != "saved-token" {
		t.Fatalf("saved hook token = %q", got)
	}
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "nested", "copy.txt")
	if err := os.WriteFile(src, []byte("payload"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := copyFileWithMode(src, dst, 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "payload" {
		t.Fatalf("copied file = %q/%v", data, err)
	}
	if err := copyFileIfExists(filepath.Join(dir, "missing"), filepath.Join(dir, "ignored")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(legacyDBCandidates(), ","), "web-console") {
		t.Fatal("legacy DB candidates did not include web-console")
	}
}

func TestNewServerInitializesCapabilityCatalogue(t *testing.T) {
	root := t.TempDir()
	for _, class := range []string{"config", "data", "cache", "logs", "state"} {
		t.Setenv("VROOLI_"+strings.ToUpper(class)+"_ROOT", filepath.Join(root, class))
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "xdg-state"))
	t.Setenv("WEB_CONSOLE_BRIDGE_URL", "")
	t.Setenv("AUDIO_TOOLS_URL", "http://127.0.0.1:1")

	dsn := resolveSQLiteDSN()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	routed := database.NewFromPrimary(db)
	srv := NewServer(routed)
	t.Cleanup(func() {
		srv.sweeper.Stop()
		srv.sessions.StopReattachWatchdog()
		srv.sessions.Shutdown()
		if srv.codexTailer != nil {
			srv.codexTailer.Stop()
		}
		if srv.claudeTailer != nil {
			srv.claudeTailer.Stop()
		}
		if srv.grokTailer != nil {
			srv.grokTailer.Stop()
		}
		if srv.opencodeWatcher != nil {
			srv.opencodeWatcher.Stop()
		}
		_ = db.Close()
	})

	described, err := srv.capabilities.Describe(t.Context())
	if err != nil {
		t.Fatalf("describe capabilities: %v", err)
	}
	var catalogue struct {
		Definitions []struct {
			ID string `json:"id"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(described, &catalogue); err != nil {
		t.Fatalf("decode capability catalogue: %v", err)
	}
	ids := make(map[string]bool, len(catalogue.Definitions))
	for _, definition := range catalogue.Definitions {
		ids[definition.ID] = true
	}
	for _, id := range []string{"audio-tools", "ollama", "openrouter", "session-backend-standard", "session-backend-persistent", "vrooli-bridge"} {
		if !ids[id] {
			t.Errorf("capability catalogue missing %q; got %v", id, ids)
		}
	}
}

func TestSettingsAdapterValidatesAndUpdatesDefaults(t *testing.T) {
	srv := newFakeTestServer()
	srv.backendRegistry = InitDefaultRegistry()
	adapter := newSettingsAdapter(srv)

	initial := adapter.GetDefaults()
	if initial.DefaultBackend == "" || initial.DefaultPolicy.Mode == "" {
		t.Fatalf("initial defaults = %+v", initial)
	}
	unknown := "missing-backend"
	if _, err := adapter.UpdateDefaults(settingsH.UpdateDefaultsRequest{DefaultBackend: &unknown}); err == nil {
		t.Fatal("unknown backend update returned nil error")
	}
	validBackend := "standard"
	validPolicy := settingsH.Policy{Mode: "preset", Duration: "1h"}
	updated, err := adapter.UpdateDefaults(settingsH.UpdateDefaultsRequest{
		DefaultBackend: &validBackend,
		DefaultPolicy:  &validPolicy,
	})
	if err != nil {
		t.Fatalf("valid defaults update: %v", err)
	}
	if updated.DefaultBackend != validBackend || updated.DefaultPolicy != validPolicy {
		t.Fatalf("updated defaults = %+v, want backend/policy %q/%+v", updated, validBackend, validPolicy)
	}
	invalidPolicy := settingsH.Policy{Mode: "custom", Duration: "not-a-duration"}
	if _, err := adapter.UpdateDefaults(settingsH.UpdateDefaultsRequest{DefaultPolicy: &invalidPolicy}); err == nil {
		t.Fatal("invalid policy update returned nil error")
	}
}

func TestHookAndPreviewErrorProjections(t *testing.T) {
	if got := (hookStopRequest{AssistantResponse: "fallback"}).assistantText(); got != "fallback" {
		t.Fatalf("assistant fallback = %q", got)
	}
	if got := (hookStopRequest{AssistantResponse: "fallback", LastAssistantMessage: "preferred"}).assistantText(); got != "preferred" {
		t.Fatalf("assistant preferred = %q", got)
	}

	cases := []struct {
		code filepreview.Code
		want error
	}{
		{filepreview.CodeInvalid, filePreviewH.ErrInvalidArgument},
		{filepreview.CodeNotAllowed, filePreviewH.ErrPermissionDenied},
		{filepreview.CodeNotPreviewable, filePreviewH.ErrPreviewUnavailable},
		{filepreview.CodeNotFound, filePreviewH.ErrNotFound},
		{filepreview.CodeUnresolvable, filePreviewH.ErrNotFound},
		{filepreview.CodeStale, filePreviewH.ErrStale},
	}
	for _, tc := range cases {
		err := mapFilePreviewError(&filepreview.Error{Code: tc.code, Message: "mapped"})
		if !errors.Is(err, tc.want) {
			t.Errorf("map %s = %v, want sentinel %v", tc.code, err, tc.want)
		}
	}
}

func TestTTSStatusSnapshotsReturnCopiesAndEmptyState(t *testing.T) {
	srv := &Server{}
	if result, at := srv.getLastTTSRouting(); result != nil || !at.IsZero() {
		t.Fatalf("empty routing snapshot = %v/%v", result, at)
	}
	if event, at := srv.getLastTTSAck(); event != nil || !at.IsZero() {
		t.Fatalf("empty ack snapshot = %v/%v", event, at)
	}
	srv.recordLastTTSRouting(ConversationAppendResult{Source: "hook", Code: "accepted", SessionID: "s1"})
	routing, _ := srv.getLastTTSRouting()
	if routing == nil || routing.Source != "hook" {
		t.Fatalf("routing snapshot = %+v", routing)
	}
	srv.recordTTSAck(TTSClientAck{Source: "ui", Stage: "started", SessionID: "s1"})
	ack, _ := srv.getLastTTSAck()
	if ack == nil || ack.Stage != "started" {
		t.Fatalf("ack snapshot = %+v", ack)
	}
	srv.recordTTSPlaybackEvent(TTSPlaybackEvent{Source: "ui", Stage: "played", SessionID: "s1"})
	playback, _ := srv.getLastTTSPlaybackEvent()
	if playback == nil || playback.Stage != "played" {
		t.Fatalf("playback snapshot = %+v", playback)
	}
}

func TestTTSHookConfigPatchAndLoadDefaults(t *testing.T) {
	base := DefaultTTSHookConfig()
	auto, backend, muted := true, "kokoro", false
	got := (TTSHookConfigPatch{AutoEnabled: &auto, Backend: &backend, StartMuted: &muted}).Apply(base)
	if got.AutoEnabled != auto || got.Backend != backend || got.StartMuted != muted {
		t.Fatalf("applied TTS config = %+v", got)
	}
	invalidBackend := "not-a-backend"
	if got := (TTSHookConfigPatch{Backend: &invalidBackend}).Apply(base); got.Backend != base.Backend {
		t.Fatalf("invalid backend changed config to %q", got.Backend)
	}
	missing, err := loadTTSHookConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil || missing != base {
		t.Fatalf("missing config = %+v/%v", missing, err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"autoEnabled":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadTTSHookConfig(path)
	if err != nil || !loaded.AutoEnabled || loaded.Backend != "auto" {
		t.Fatalf("loaded config = %+v/%v", loaded, err)
	}
}

func TestContentDispositionAndEnvironmentHelpers(t *testing.T) {
	unsafe := "bad\"name" + `\` + "\n.txt"
	if got := sanitizeContentDispositionFilename(unsafe); got != "bad_name__.txt" {
		t.Fatalf("sanitized filename = %q", got)
	}
	t.Setenv("WEB_CONSOLE_TEST_VALUE", "set")
	if got := getEnvOrDefault("WEB_CONSOLE_TEST_VALUE", "fallback"); got != "set" {
		t.Fatalf("set environment value = %q", got)
	}
	t.Setenv("WEB_CONSOLE_TEST_VALUE", "")
	if got := getEnvOrDefault("WEB_CONSOLE_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("empty environment value = %q", got)
	}
}
