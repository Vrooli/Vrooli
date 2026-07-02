package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGreenfield_NoRawSetSizeOutsideGatedPaths enforces the plan
// constraint that SIGWINCH-by-SetSize only fires via
// `maybeSIGWINCHRecovery` or the public `Resize` method. Any other
// call site on these files is a regression (see
// Checks both session.go (Resize) and broadcast.go
// (maybeSIGWINCHRecovery) since the method was moved there in the
// decomposition phase.
func TestGreenfield_NoRawSetSizeOutsideGatedPaths(t *testing.T) {
	for _, file := range []string{"session/session.go", "session/broadcast.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		re := regexp.MustCompile(`s\.pty\.SetSize\(`)
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if !re.MatchString(line) {
				continue
			}
			// Allowed: inside maybeSIGWINCHRecovery or Resize.
			enclosing := findEnclosingFunc(lines, i)
			switch enclosing {
			case "maybeSIGWINCHRecovery", "Resize":
				// ok
			default:
				t.Errorf("%s:%d SetSize outside gated path (enclosing func=%q): %q",
					file, i+1, enclosing, strings.TrimSpace(line))
			}
		}
	}
}

func TestGreenfield_AudioToolsDependencyIsLazyDegraded(t *testing.T) {
	type scenarioDependency struct {
		Required         bool   `json:"required"`
		StartupPolicy    string `json:"startup_policy"`
		DegradedBehavior string `json:"degraded_behavior"`
	}
	var service struct {
		Dependencies struct {
			Scenarios map[string]scenarioDependency `json:"scenarios"`
		} `json:"dependencies"`
	}

	raw, err := os.ReadFile("../.vrooli/service.json")
	if err != nil {
		t.Fatalf("read service metadata: %v", err)
	}
	if err := json.Unmarshal(raw, &service); err != nil {
		t.Fatalf("parse service metadata: %v", err)
	}
	dep, ok := service.Dependencies.Scenarios["audio-tools"]
	if !ok {
		t.Fatalf("audio-tools dependency missing from service metadata")
	}
	if dep.Required {
		t.Fatalf("audio-tools must stay optional for terminal boot")
	}
	if dep.StartupPolicy != "try_start" {
		t.Fatalf("audio-tools startup_policy = %q, want try_start", dep.StartupPolicy)
	}
	lower := strings.ToLower(dep.DegradedBehavior)
	for _, banned := range []string{"fail fast", "fails fast", "required at startup"} {
		if strings.Contains(lower, banned) {
			t.Fatalf("audio-tools degraded_behavior reintroduced fail-fast language: %q", dep.DegradedBehavior)
		}
	}
	if !strings.Contains(lower, "terminal workspace boots") {
		t.Fatalf("audio-tools degraded_behavior should state terminal boot remains available: %q", dep.DegradedBehavior)
	}
}

// TestGreenfield_NoRawPtmxWriteOutsidePTYFiles enforces that raw
// ptmx.Write(...) calls only appear inside pty.go / pty_tmux.go.
// Anywhere else means a caller bypassed the PTY interface's kind-
// aware WriteInput — exactly the Bug A regression we refuse to ship
// again. See refactor plan §10.4 and §14.2.
func TestGreenfield_NoRawPtmxWriteOutsidePTYFiles(t *testing.T) {
	allowed := map[string]bool{
		"pty.go":      true,
		"pty_tmux.go": true,
	}
	re := regexp.MustCompile(`\bptmx\.Write\(`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		if allowed[f] {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if re.Match(b) {
			t.Errorf("%s calls ptmx.Write(...) — must route through PTY.WriteInput instead", f)
		}
	}
}

// TestGreenfield_PTYInterfaceHasNoLegacyWrite enforces the greenfield
// rule that the old PTY.Write method was deleted, not kept as a
// compat alias. Every call site must go through WriteInput with an
// explicit pty.InputKind. Looks for the exact legacy signature in pty.go.
func TestGreenfield_PTYInterfaceHasNoLegacyWrite(t *testing.T) {
	b, err := os.ReadFile("pty.go")
	if err != nil {
		t.Fatalf("read pty.go: %v", err)
	}
	// The PTY interface body contains `Read(...)` and `WriteInput(...)`
	// but MUST NOT contain a raw `Write(p []byte) (int, error)`
	// method declaration (the legacy shape).
	legacy := regexp.MustCompile(`\n\s*Write\(p \[\]byte\) \(int, error\)`)
	if legacy.Match(b) {
		t.Errorf("pty.go still declares the legacy PTY.Write(p []byte) (int, error) method")
	}
}

// TestGreenfield_NoReferencesToRemovedPlans ensures the deleted plan
// filenames are not referenced from in-tree source files. Dangling
// references would indicate the deletion left the codebase in an
// inconsistent state. The comparison is case-sensitive and built at
// runtime from parts so THIS test's own source doesn't count as a
// reference.
func TestGreenfield_NoReferencesToRemovedPlans(t *testing.T) {
	// Assembled at runtime so this test file doesn't literally
	// contain the full filenames.
	removed := []string{
		"terminal-session-" + "rework" + "-implementation-plan",
		"terminal-session-" + "rework" + "-phase-2-implementation-plan",
		"persistent-terminal-" + "input-protection" + "-plan",
		"detachable-" + "sessions-implementation" + "-plan",
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		// Exclude this test itself (it contains the assembly, which
		// at the byte level matches).
		if f == "greenfield_assertions_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, name := range removed {
			if strings.Contains(string(b), name) {
				t.Errorf("%s references deleted plan %q", f, name)
			}
		}
	}
}

// findEnclosingFunc walks backward from lineIdx to find the most
// recent `func` line and returns the function name.
func findEnclosingFunc(lines []string, lineIdx int) string {
	re := regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?([A-Za-z_][A-Za-z0-9_]*)`)
	for i := lineIdx; i >= 0; i-- {
		if m := re.FindStringSubmatch(lines[i]); m != nil {
			return m[1]
		}
	}
	return ""
}

// TestGreenfield_WorkspaceStoreNotInPackageMain enforces that workspace
// persistence code lives in internal/workspace, not package main. The
// migration deleted workspace_store{,_mem,_sql,_shim}.go; resurrecting
// any of them — or the old package-main type names — is a regression.
func TestGreenfield_WorkspaceStoreNotInPackageMain(t *testing.T) {
	forbiddenFiles := []string{
		"workspace_store.go",
		"workspace_store_mem.go",
		"workspace_store_sql.go",
		"workspace_store_shim.go",
	}
	for _, f := range forbiddenFiles {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("%s reappeared in package main — workspace persistence belongs in internal/workspace", f)
		}
	}

	// Word-bounded so e.g. events.WorkspaceLayoutUpdated (constant name)
	// or TestSQLStore_* (renamed) don't false-positive.
	forbiddenSymbols := []string{
		`\bWorkspaceStore\b`,
		`\bMemWorkspaceStore\b`,
		`\bSQLWorkspaceStore\b`,
		`\bWorkspacePane\b`,
		`\bNewMemWorkspaceStore\b`,
		`\bNewSQLWorkspaceStore\b`,
		`\bworkspaceStoreShim\b`,
		`\bnewWorkspaceStoreShim\b`,
	}
	res := make([]*regexp.Regexp, len(forbiddenSymbols))
	for i, p := range forbiddenSymbols {
		res[i] = regexp.MustCompile(p)
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if f == "greenfield_assertions_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for i, re := range res {
			if re.Match(b) {
				t.Errorf("%s references legacy workspace symbol %q — use internal/workspace types", f, forbiddenSymbols[i])
			}
		}
	}
}

// TestGreenfield_AIDomainNotInPackageMain enforces that AI provider /
// config-store / system-context code lives in internal/ai, not package
// main. The migration deleted ai_provider{,_config,_config_sql}.go,
// ai_system_context.go, and ai_service_shim.go; resurrecting any of
// them — or the old package-main type names — is a regression.
func TestGreenfield_AIDomainNotInPackageMain(t *testing.T) {
	forbiddenFiles := []string{
		"ai_provider.go",
		"ai_provider_config.go",
		"ai_provider_config_sql.go",
		"ai_system_context.go",
		"ai_service_shim.go",
	}
	for _, f := range forbiddenFiles {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("%s reappeared in package main — AI domain belongs in internal/ai", f)
		}
	}

	forbiddenSymbols := []string{
		`\bAIProviderChain\b`,
		`\bNewAIProviderChain\b`,
		`\bAIProviderConfigStore\b`,
		`\bNewAIProviderConfigStore\b`,
		`\bSQLAIConfigStore\b`,
		`\bNewSQLAIConfigStore\b`,
		`\bAIConfigStore\b`,
		`\baiServiceShim\b`,
		`\bnewAIServiceShim\b`,
		`\bproviderHealthTracker\b`,
		`\bbuildCommandSystemPrompt\b`,
		`\bbuildSuggestSystemPrompt\b`,
		`\bnewAIServiceShim\b`,
	}
	res := make([]*regexp.Regexp, len(forbiddenSymbols))
	for i, p := range forbiddenSymbols {
		res[i] = regexp.MustCompile(p)
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if f == "greenfield_assertions_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for i, re := range res {
			if re.Match(b) {
				t.Errorf("%s references legacy AI symbol %q — use internal/ai types", f, forbiddenSymbols[i])
			}
		}
	}
}

// TestGreenfield_VoiceDomainNotInPackageMain enforces that voice config,
// stream WS, transcribe, wake-word, and speaker-verification code lives in
// internal/voice, not package main. The migration deleted voice_config.go,
// voice_transcribe.go, voice_stream_ws.go, voice_service_shim.go,
// speaker_verification{,_client,_config}.go, and wakeword_handlers.go;
// resurrecting any of them — or the old package-main type names — is a
// regression.
func TestGreenfield_VoiceDomainNotInPackageMain(t *testing.T) {
	forbiddenFiles := []string{
		"voice_config.go",
		"voice_transcribe.go",
		"voice_stream_ws.go",
		"voice_service_shim.go",
		"speaker_verification.go",
		"speaker_verification_client.go",
		"speaker_verification_config.go",
		"wakeword_handlers.go",
	}
	for _, f := range forbiddenFiles {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("%s reappeared in package main — voice domain belongs in internal/voice", f)
		}
	}

	forbiddenSymbols := []string{
		`\bVoiceStreamConfig\b`,
		`\bVoiceStreamConfigPatch\b`,
		`\bDefaultVoiceStreamConfig\b`,
		`\bSpeakerVerificationConfig\b`,
		`\bSpeakerVerificationConfigPatch\b`,
		`\bDefaultSpeakerVerificationConfig\b`,
		`\bSpeakerVerificationResourceClient\b`,
		`\bSpeakerVerificationProfile\b`,
		`\bSpeakerVerificationProfileList\b`,
		`\bSpeakerVerificationResourceInfo\b`,
		`\bSpeakerVerificationResourceReady\b`,
		`\bSpeakerVerificationEnrollmentResponse\b`,
		`\bSpeakerVerificationResult\b`,
		`\bvoiceServiceShim\b`,
		`\bnewVoiceServiceShim\b`,
		`\bspeakerVerificationGateDecision\b`,
		`\bevaluateSpeakerVerification\b`,
		`\bextractTargetSpeaker\b`,
		`\bisWhisperHallucination\b`,
		`\bdeduplicateOverlap\b`,
		`\btranscribeBytes\b`,
		`\bhandleVoiceStreamWS\b`,
		`\bloadVoiceConfig\b`,
		`\bsaveVoiceConfig\b`,
		`\bloadSpeakerVerificationConfig\b`,
		`\bsaveSpeakerVerificationConfig\b`,
		`\bloadWakeWordTemplate\b`,
		`\bsaveWakeWordTemplate\b`,
		`\bdeleteWakeWordTemplate\b`,
		`\bvalidateWakeWordTemplate\b`,
	}
	res := make([]*regexp.Regexp, len(forbiddenSymbols))
	for i, p := range forbiddenSymbols {
		res[i] = regexp.MustCompile(p)
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if f == "greenfield_assertions_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for i, re := range res {
			if re.Match(b) {
				t.Errorf("%s references legacy voice symbol %q — use internal/voice types", f, forbiddenSymbols[i])
			}
		}
	}
}

// TestGreenfield_TTSReusableCoreNotInPackageMain enforces that reusable TTS
// text-processing primitives live under api/internal/tts, not in package
// main. Extraction-prep work moved NormalizeTextForSpeech,
// SplitIntoSpeechParagraphs, and TTSMaxChunkLength into the internal package
// so the future audio-tools scenario can own them without taking
// web-console-specific glue along for the ride.
func TestGreenfield_TTSReusableCoreNotInPackageMain(t *testing.T) {
	// Files whose entire content moved into internal/tts. tts_cache.go
	// remains in package main but holds only Server-method orchestration
	// glue (invalidate / preSynthesize / synthesizeParagraphs).
	forbiddenFiles := []string{
		"tts_normalizer.go",
		"tts_chunker.go",
		"tts_summarizer.go",
		"tts_summarization_service.go",
		"tts_summarize_config.go",
	}
	for _, f := range forbiddenFiles {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("%s reappeared in package main — reusable TTS core belongs in internal/tts", f)
		}
	}

	// Definitions of these symbols must only appear in internal/tts. Their
	// references from package main go through `inttts.*`, so the `func `
	// prefix here scopes the check to definitions.
	forbiddenDefinitions := []string{
		`func NormalizeTextForSpeech\(`,
		`func SplitIntoSpeechParagraphs\(`,
		`const TTSMaxChunkLength`,
		`type TTSCache `,
		`type TTSCacheKey `,
		`type TTSCacheEntry `,
		`func NewTTSCache\(`,
		`type TTSSummarizer `,
		`func NewTTSSummarizer\(`,
		`type TTSSummarizationService `,
		`func NewTTSSummarizationService\(`,
		`type TTSConfig `,
		`type TTSSummarizeConfig `,
		`func DefaultTTSConfig\(`,
		`func DefaultTTSSummarizeConfig\(`,
		`func loadTTSConfig\(`,
		`func saveTTSConfig\(`,
		`func loadTTSSummarizeConfig\(`,
		`func saveTTSSummarizeConfig\(`,
	}
	res := make([]*regexp.Regexp, len(forbiddenDefinitions))
	for i, p := range forbiddenDefinitions {
		res[i] = regexp.MustCompile(p)
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if f == "greenfield_assertions_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for i, re := range res {
			if re.Match(b) {
				t.Errorf("%s defines %q — definition must live in internal/tts", f, forbiddenDefinitions[i])
			}
		}
	}
}

// TestGreenfield_InternalAudioDomainsDoNotImportHandlers enforces the
// dependency direction required for audio-tools extraction: the remaining
// internal audio packages (internal/audio, internal/audioports) must not
// import any web-console/handlers/* package. Handlers depend on internal
// domain types via alias; the reverse direction would re-couple the domain
// to the transport. internal/voice and internal/tts are gone — the audio-
// tools adoption made web-console own neither.
func TestGreenfield_InternalAudioDomainsDoNotImportHandlers(t *testing.T) {
	internalDirs := []string{
		"internal/audio",
		"internal/audioports",
	}
	handlerImport := regexp.MustCompile(`"web-console/handlers/[^"]+"`)
	for _, dir := range internalDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			b, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if m := handlerImport.Find(b); m != nil {
				t.Errorf("%s imports %s — internal audio domains must not depend on handler packages", path, string(m))
			}
		}
	}
}

// TestGreenfield_NoAudioPackagesInWebConsole locks in the audio-tools
// adoption. After the cutover the following packages must never reappear:
//
//   - web-console/internal/voice (Whisper / wake word / speaker verification)
//   - web-console/internal/tts (Kokoro / summarizer / cache / chunker / normalizer)
//   - web-console/handlers/voice / handlers/tts (REST/Connect TTS+voice handlers)
//   - @vrooli/proto-types/web-console/v1/voice / .../v1/tts (the proto schemas)
//
// All audio synthesis, voice listing, summarization, transcription,
// speaker-verification, and wake-word lives in audio-tools now. Web-console
// reaches it through audioports.Remote* adapters + the local audio-integration UI module.
func TestGreenfield_NoAudioPackagesInWebConsole(t *testing.T) {
	deletedDirs := []string{
		"internal/voice",
		"internal/tts",
		"handlers/voice",
		"handlers/tts",
	}
	for _, dir := range deletedDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			t.Errorf("%s reappeared — audio-tools adoption removed it; all audio flows through audioports.Remote*", dir)
		}
	}

	forbiddenImports := []*regexp.Regexp{
		regexp.MustCompile(`"web-console/internal/voice"`),
		regexp.MustCompile(`"web-console/internal/tts"`),
		regexp.MustCompile(`"web-console/handlers/voice"`),
		regexp.MustCompile(`"web-console/handlers/tts"`),
		regexp.MustCompile(`"github\.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice"`),
		regexp.MustCompile(`"github\.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts"`),
	}
	checkFile := func(path string) {
		b, err := os.ReadFile(path)
		if err != nil {
			return
		}
		for _, re := range forbiddenImports {
			if loc := re.FindIndex(b); loc != nil {
				t.Errorf("%s imports forbidden audio package %s — must use audioports.* adapters instead", path, re.String())
			}
		}
	}
	rootFiles, err := filepath.Glob("*.go")
	if err == nil {
		for _, f := range rootFiles {
			if f == "greenfield_assertions_test.go" {
				continue
			}
			checkFile(f)
		}
	}
	internalDirs, err := os.ReadDir("internal")
	if err == nil {
		for _, e := range internalDirs {
			if !e.IsDir() {
				continue
			}
			subFiles, err := filepath.Glob(filepath.Join("internal", e.Name(), "*.go"))
			if err != nil {
				continue
			}
			for _, f := range subFiles {
				checkFile(f)
			}
		}
	}
}

// TestGreenfield_OrchestrationRoutesThroughAudioPorts ensures the
// conversation/TTS orchestration sites in package main do NOT call the raw
// internal/tts text-pipeline functions directly — they must go through the
// audioports.SpeechTextProcessor seam so a future audio-tools-backed
// implementation can be swapped in without touching orchestration.
//
// The check looks for `inttts.NormalizeTextForSpeech(` and
// `inttts.SplitIntoSpeechParagraphs(` in package-main .go files (excluding
// the audioports local adapter and any test). If they reappear, the seam
// has been bypassed.
func TestGreenfield_OrchestrationRoutesThroughAudioPorts(t *testing.T) {
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`\binttts\.NormalizeTextForSpeech\(`),
		regexp.MustCompile(`\binttts\.SplitIntoSpeechParagraphs\(`),
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, re := range forbidden {
			if loc := re.FindIndex(b); loc != nil {
				t.Errorf("%s calls %s directly — must route through audioports.SpeechTextProcessor instead", f, re.String())
			}
		}
	}
}

// TestGreenfield_OrchestrationGoesThroughAudioPorts enforces that
// orchestration files in package main do NOT call the raw STT/TTS provider
// surfaces directly — they must go through audioports.SpeechToText /
// audioports.TextToSpeech so the future audio-tools client can be swapped in
// without touching orchestration glue.
//
// The summarize pipeline (SummarizationService / SummarizeRequest /
// SummarizeResult / SummarizeErrorMessage / ErrSummarizeCoolingDown) is
// explicitly NOT routed through audioports in this pass — see
// docs/internal/PROBLEMS.md §10. This assertion therefore scopes to STT/TTS
// provider symbols and never to summarize symbols.
func TestGreenfield_OrchestrationGoesThroughAudioPorts(t *testing.T) {
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`\bs\.voiceService\.Transcribe\b`),
		regexp.MustCompile(`internal/voice\.Service\.Transcribe`),
	}
	// main.go is exempt: it constructs the audioports.Remote* adapters from
	// the audio-tools integration client and is the single seam where the
	// underlying provider is named.
	exempt := map[string]bool{
		"main.go":                       true,
		"greenfield_assertions_test.go": true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		if exempt[f] {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, re := range forbidden {
			if loc := re.FindIndex(b); loc != nil {
				t.Errorf("%s references provider-direct symbol %s — must route through audioports.SpeechToText/TextToSpeech instead", f, re.String())
			}
		}
	}
}

// TestGreenfield_NoLegacyHistorySymbols enforces that the deleted
// raw-byte history protocol stays deleted. New code must use the
// snapshot-replay flow through terminal.Emulator.
func TestGreenfield_NoLegacyHistorySymbols(t *testing.T) {
	forbidden := []string{
		"outputHistory",
		"appendHistory",
		"OfflineBufferMax",
		"WC_OFFLINE_BUFFER_MAX",
		"history_offset",
		"PTYStateTracker",
		"snapToCleanBoundary",
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if f == "greenfield_assertions_test.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		s := string(b)
		for _, sym := range forbidden {
			if strings.Contains(s, sym) {
				t.Errorf("%s references deleted symbol %q", f, sym)
			}
		}
	}
}
