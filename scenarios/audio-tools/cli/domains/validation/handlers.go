package validation

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/domains/stt"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

const (
	minimumQualificationMinutes = 60
	defaultQualificationMinutes = minimumQualificationMinutes
	defaultTTSInterval          = 10 * time.Second
	syntheticFrameDuration      = 100 * time.Millisecond
	syntheticSampleRate         = 16000
	syntheticBytesPerSample     = 2
	syntheticToneHz             = 440
	evidenceFileName            = "expensive-validation.json"
)

type handlers struct {
	core      *cliapp.ScenarioApp
	now       func() time.Time
	newTicker func(time.Duration) *time.Ticker
	getenv    func(string) string
	getwd     func() (string, error)
}

func newHandlersWithClock(core *cliapp.ScenarioApp, now func() time.Time) *handlers {
	return newHandlersWithDependencies(core, now, nil, func(string) string { return "" }, func() (string, error) { return "", errors.New("working directory is not configured") })
}

func newHandlersWithDependencies(core *cliapp.ScenarioApp, now func() time.Time, newTicker func(time.Duration) *time.Ticker, getenv func(string) string, getwd func() (string, error)) *handlers {
	if now == nil {
		now = func() time.Time { return time.Unix(0, 0) }
	}
	if newTicker == nil {
		newTicker = func(time.Duration) *time.Ticker { return nil }
	}
	return &handlers{core: core, now: now, newTicker: newTicker, getenv: getenv, getwd: getwd}
}

// validationEvidence is intentionally a small, append-free artifact. A single
// current record makes freshness unambiguous and keeps the business phase from
// accidentally crediting an old successful run after a newer failure.
type validationEvidence struct {
	Scenario       string    `json:"scenario"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
	Duration       string    `json:"duration"`
	DurationSecond float64   `json:"duration_seconds"`
	TTSRequests    uint64    `json:"tts_requests"`
	TTSFrames      uint64    `json:"tts_frames"`
	TTSBytes       uint64    `json:"tts_bytes"`
	STTChunks      uint64    `json:"stt_chunks"`
	STTBytes       uint64    `json:"stt_bytes"`
	STTEvents      uint64    `json:"stt_events"`
	Checks         []string  `json:"checks"`
}

type validationCounters struct {
	ttsRequests atomic.Uint64
	ttsFrames   atomic.Uint64
	ttsBytes    atomic.Uint64
	sttChunks   atomic.Uint64
	sttBytes    atomic.Uint64
	sttEvents   atomic.Uint64
}

func (c *validationCounters) snapshot() validationEvidence {
	return validationEvidence{
		TTSRequests: c.ttsRequests.Load(),
		TTSFrames:   c.ttsFrames.Load(),
		TTSBytes:    c.ttsBytes.Load(),
		STTChunks:   c.sttChunks.Load(),
		STTBytes:    c.sttBytes.Load(),
		STTEvents:   c.sttEvents.Load(),
	}
}

func (h *handlers) runExpensive(ctx cliapp.RunContext) error {
	minutes := defaultQualificationMinutes
	if raw := strings.TrimSpace(ctx.Flag("duration-minutes")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < minimumQualificationMinutes {
			return fmt.Errorf("--duration-minutes must be at least %d", minimumQualificationMinutes)
		}
		minutes = parsed
	}

	root, err := h.scenarioRoot()
	if err != nil {
		return err
	}
	requirementsDir := filepath.Join(root, "requirements")
	evidencePath := strings.TrimSpace(ctx.Flag("evidence-path"))
	if evidencePath == "" {
		evidencePath = filepath.Join(requirementsDir, "evidence", evidenceFileName)
	}
	return h.runAndRecord(context.Background(), requirementsDir, evidencePath, time.Duration(minutes)*time.Minute, ctx.Stdout(), ctx.JSON())
}

func (h *handlers) runAndRecord(ctx context.Context, requirementsDir, evidencePath string, duration time.Duration, out io.Writer, asJSON bool) error {
	if duration < minimumQualificationMinutes*time.Minute {
		return fmt.Errorf("expensive audio validation requires at least %d minutes", minimumQualificationMinutes)
	}
	if h.core == nil {
		return errors.New("audio-tools validation requires the scenario CLI core")
	}

	started := h.now().UTC()
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	ttsHTTP, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, 0)
	ttsClient := ttsconnect.NewTTSServiceClient(ttsHTTP, baseURL)
	sttClient := stt.NewStreamingSTTClient(h.core, baseURL)
	counters := &validationCounters{}
	errCh := make(chan error, 2)

	go func() { errCh <- runTTSQualification(ctx, ttsClient, counters, h.newTicker) }()
	go func() { errCh <- runDictationQualification(ctx, sttClient, counters, h.newTicker) }()

	var firstErr error
	for range 2 {
		if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && firstErr == nil {
			firstErr = err
			cancel()
		}
	}
	if firstErr != nil {
		return firstErr
	}
	if ctx.Err() != context.DeadlineExceeded {
		return fmt.Errorf("expensive audio validation ended before its duration: %w", ctx.Err())
	}

	finished := h.now().UTC()
	evidence := counters.snapshot()
	evidence.Scenario = "audio-tools"
	evidence.StartedAt = started
	evidence.FinishedAt = finished
	evidence.Duration = finished.Sub(started).String()
	evidence.DurationSecond = finished.Sub(started).Seconds()
	evidence.Checks = []string{
		"hour-scale TTS streaming remained available for the full qualification duration",
		"continuous synthetic PCM dictation streamed in real time without a terminal transport error",
	}
	if err := writeEvidence(evidencePath, evidence); err != nil {
		return err
	}
	updated, err := markOutOfBandValidations(requirementsDir, finished)
	if err != nil {
		return err
	}
	if updated == 0 {
		return fmt.Errorf("no out_of_band validations found under %s", requirementsDir)
	}

	if asJSON {
		return json.NewEncoder(out).Encode(evidence)
	}
	fmt.Fprintf(out, "audio-tools expensive validation passed: %s (%d out-of-band validations refreshed)\n", evidence.Duration, updated)
	fmt.Fprintf(out, "TTS requests=%d frames=%d bytes=%d; STT chunks=%d events=%d bytes=%d\n", evidence.TTSRequests, evidence.TTSFrames, evidence.TTSBytes, evidence.STTChunks, evidence.STTEvents, evidence.STTBytes)
	return nil
}

func (h *handlers) checkFreshness(ctx cliapp.RunContext) error {
	root, err := h.scenarioRoot()
	if err != nil {
		return err
	}
	requirementsDir := strings.TrimSpace(ctx.Flag("requirements-path"))
	if requirementsDir == "" {
		requirementsDir = filepath.Join(root, "requirements")
	}
	stale, err := findStaleValidations(requirementsDir, h.now())
	if err != nil {
		return err
	}
	if len(stale) > 0 {
		return fmt.Errorf("out-of-band audio validation is stale or missing: %s", strings.Join(stale, ", "))
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(map[string]any{"fresh": true, "requirements_path": requirementsDir})
	}
	fmt.Fprintf(ctx.Stdout(), "audio-tools out-of-band validation evidence is fresh (%s)\n", requirementsDir)
	return nil
}

func runTTSQualification(ctx context.Context, client ttsconnect.TTSServiceClient, counters *validationCounters, newTicker func(time.Duration) *time.Ticker) error {
	ticker := newTicker(defaultTTSInterval)
	defer ticker.Stop()
	for {
		if err := synthesizeProbe(ctx, client, counters); err != nil {
			return fmt.Errorf("TTS qualification: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func synthesizeProbe(ctx context.Context, client ttsconnect.TTSServiceClient, counters *validationCounters) error {
	stream, err := client.SynthesizeStream(ctx, connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text:           "This is a continuous audio reliability qualification sentence.",
		Voice:          "voice.neutral.default",
		Speed:          1,
		ResponseFormat: commonv1.ResponseFormat_RESPONSE_FORMAT_WAV,
	}))
	if err != nil {
		return err
	}
	counters.ttsRequests.Add(1)
	seenFinal := false
	for stream.Receive() {
		frame := stream.Msg()
		if frame == nil {
			continue
		}
		counters.ttsFrames.Add(1)
		counters.ttsBytes.Add(uint64(len(frame.Audio)))
		seenFinal = seenFinal || frame.IsFinal
	}
	if err := stream.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if !seenFinal {
		return errors.New("stream ended without a final audio frame")
	}
	return nil
}

func runDictationQualification(ctx context.Context, client sttconnect.STTServiceClient, counters *validationCounters, newTicker func(time.Duration) *time.Ticker) error {
	stream := client.TranscribeStream(ctx)
	if err := stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_Start{Start: &sttv1.StreamStart{
		ProtocolVersion:   2,
		InputFormat:       commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE,
		InputSampleRateHz: syntheticSampleRate,
		SessionId:         "out-of-band-audio-qualification",
		ResumeToken:       "out-of-band-audio-qualification",
	}}}); err != nil {
		return fmt.Errorf("send stream start: %w", err)
	}

	receiveErr := make(chan error, 1)
	go func() {
		for {
			event, err := stream.Receive()
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
					receiveErr <- nil
				} else {
					receiveErr <- err
				}
				return
			}
			if event != nil {
				counters.sttEvents.Add(1)
				if event.GetError() != nil {
					receiveErr <- fmt.Errorf("server stream error %s: %s", event.GetError().Code, event.GetError().Message)
					return
				}
			}
		}
	}()

	ticker := newTicker(syntheticFrameDuration)
	defer ticker.Stop()
	var sequence uint64
	var sampleOffset int64
	for {
		select {
		case <-ctx.Done():
			_ = stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}})
			_ = stream.CloseRequest()
			<-receiveErr
			return ctx.Err()
		case err := <-receiveErr:
			if err != nil {
				return fmt.Errorf("receive stream events: %w", err)
			}
			return nil
		case tick := <-ticker.C:
			_ = tick
			pcm := syntheticPCM(syntheticSampleRate, syntheticFrameDuration, sampleOffset)
			digest := sha256.Sum256(pcm)
			endSample := sampleOffset + int64(len(pcm)/syntheticBytesPerSample)
			if err := stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: &sttv1.StreamAudioChunk{
				Audio: pcm, Sequence: sequence, StartSample: sampleOffset, EndSample: endSample, Sha256: digest[:],
			}}}); err != nil {
				return fmt.Errorf("send synthetic audio chunk: %w", err)
			}
			counters.sttChunks.Add(1)
			counters.sttBytes.Add(uint64(len(pcm)))
			sequence++
			sampleOffset = endSample
		}
	}
}

func syntheticPCM(sampleRate int, duration time.Duration, sampleOffset int64) []byte {
	samples := int(math.Round(float64(sampleRate) * duration.Seconds()))
	pcm := make([]byte, samples*syntheticBytesPerSample)
	for i := 0; i < samples; i++ {
		phase := 2 * math.Pi * syntheticToneHz * float64(sampleOffset+int64(i)) / float64(sampleRate)
		sample := int32(math.Sin(phase) * 12000)
		// The amplitude is deliberately bounded to the int16 range before the
		// wire conversion; this is synthetic PCM, not an unchecked cast.
		if sample < math.MinInt16 {
			sample = math.MinInt16
		}
		if sample > math.MaxInt16 {
			sample = math.MaxInt16
		}
		binary.LittleEndian.PutUint16(pcm[i*2:i*2+2], uint16(int16(sample))) // #nosec G115 -- sample is clamped to int16 bounds above.
	}
	return pcm
}

func (h *handlers) scenarioRoot() (string, error) {
	if configured := strings.TrimSpace(h.getenv("AUDIO_TOOLS_SCENARIO_ROOT")); configured != "" {
		return filepath.Abs(configured)
	}
	directory, err := h.getwd()
	if err != nil {
		return "", fmt.Errorf("resolve audio-tools scenario root: %w", err)
	}
	for {
		candidate := filepath.Join(directory, "scenarios", "audio-tools")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}
		if filepath.Base(directory) == "audio-tools" && filepath.Base(filepath.Dir(directory)) == "scenarios" {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", errors.New("run from the Vrooli repository or set AUDIO_TOOLS_SCENARIO_ROOT")
}

func writeEvidence(path string, evidence validationEvidence) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create evidence directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".expensive-validation-*.tmp")
	if err != nil {
		return fmt.Errorf("create evidence temporary: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evidence); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode validation evidence: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close validation evidence: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("publish validation evidence: %w", err)
	}
	return nil
}

type registryValidation struct {
	Type          string `json:"type"`
	Phase         string `json:"phase"`
	Status        string `json:"status"`
	Ref           string `json:"ref"`
	OutOfBand     bool   `json:"out_of_band"`
	ValidForDays  int    `json:"valid_for_days"`
	LastValidated string `json:"last_validated_at"`
}

type registryRequirement struct {
	ID          string               `json:"id"`
	Validations []registryValidation `json:"validation"`
}

type registryModule struct {
	Requirements []registryRequirement `json:"requirements"`
}

func modulePaths(requirementsDir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(requirementsDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Base(path) == "index.json" || filepath.Dir(path) == filepath.Join(requirementsDir, "evidence") || filepath.Ext(path) != ".json" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan requirements: %w", err)
	}
	return paths, nil
}

func loadModule(path string) (registryModule, map[string]any, error) {
	// #nosec G304 -- paths are discovered from the scenario requirements tree.
	data, err := os.ReadFile(path)
	if err != nil {
		return registryModule{}, nil, err
	}
	var module registryModule
	if err := json.Unmarshal(data, &module); err != nil {
		return registryModule{}, nil, fmt.Errorf("decode %s: %w", path, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return registryModule{}, nil, fmt.Errorf("decode raw %s: %w", path, err)
	}
	return module, raw, nil
}

func markOutOfBandValidations(requirementsDir string, now time.Time) (int, error) {
	paths, err := modulePaths(requirementsDir)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, path := range paths {
		module, raw, err := loadModule(path)
		if err != nil {
			return 0, err
		}
		rawRequirements, ok := raw["requirements"].([]any)
		if !ok {
			continue
		}
		changed := false
		for reqIndex := range module.Requirements {
			for validationIndex := range module.Requirements[reqIndex].Validations {
				validation := &module.Requirements[reqIndex].Validations[validationIndex]
				if !validation.OutOfBand {
					continue
				}
				validation.LastValidated = now.UTC().Format(time.RFC3339)
				changed = true
				updated++
				if reqIndex < len(rawRequirements) {
					if rawValidation, ok := rawRequirements[reqIndex].(map[string]any); ok {
						validations, ok := rawValidation["validation"].([]any)
						if ok && validationIndex < len(validations) {
							if item, ok := validations[validationIndex].(map[string]any); ok {
								item["last_validated_at"] = validation.LastValidated
							}
						}
					}
				}
			}
		}
		if changed {
			if err := writeJSONAtomic(path, raw); err != nil {
				return 0, err
			}
		}
	}
	return updated, nil
}

func findStaleValidations(requirementsDir string, now time.Time) ([]string, error) {
	paths, err := modulePaths(requirementsDir)
	if err != nil {
		return nil, err
	}
	var stale []string
	for _, path := range paths {
		module, _, err := loadModule(path)
		if err != nil {
			return nil, err
		}
		for _, requirement := range module.Requirements {
			for _, validation := range requirement.Validations {
				if !validation.OutOfBand {
					continue
				}
				label := requirement.ID + " (" + validation.Ref + ")"
				if validation.ValidForDays < 1 || strings.TrimSpace(validation.LastValidated) == "" {
					stale = append(stale, label)
					continue
				}
				validatedAt, err := time.Parse(time.RFC3339, validation.LastValidated)
				if err != nil || now.After(validatedAt.Add(time.Duration(validation.ValidForDays)*24*time.Hour)) {
					stale = append(stale, label)
				}
			}
		}
	}
	return stale, nil
}

func writeJSONAtomic(path string, value any) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".requirements-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary requirements file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode requirements: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary requirements file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("publish requirements file: %w", err)
	}
	return nil
}
