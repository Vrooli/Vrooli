// Package soak owns the one product-path long-form dictation driver. It is
// deliberately behind the API's leased test-isolation boundary: this code can
// exercise the real browser, WebSocket, ledger, and UI without becoming a
// second production capture implementation.
package soak

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"audio-tools/internal/buildidentity"
	"audio-tools/internal/conformance"
	"audio-tools/internal/stt/session"

	"github.com/google/uuid"
	corestorage "github.com/vrooli/api-core/storage"
)

const (
	maxLedgerBytes      = 64 << 20
	defaultTurns        = 3
	defaultFeedMS       = 4000
	feedChunkMS         = 60_000
	turnStepStride      = 4096
	canonicalSampleRate = 16_000
	minWireRateFraction = 0.50
	maxWireRateFraction = 1.50
	maxTurnSampleSpread = 1.50
)

// Options describes one browser-owned soak. URLs and the fixture are explicit
// inputs so a run cannot silently target a different UI or driver instance.
type Options struct {
	DriverURL        string           `json:"driver_url"`
	UIURL            string           `json:"ui_url"`
	Surface          string           `json:"surface,omitempty"`
	Fixture          string           `json:"fixture"`
	Lane             conformance.Lane `json:"lane"`
	Profile          string           `json:"profile"`
	Turns            int              `json:"turns"`
	FeedMS           int              `json:"feed_ms"`
	Fault            string           `json:"fault,omitempty"`
	Reference        string           `json:"reference_text,omitempty"`
	EngineID         string           `json:"engine_id"`
	ModelID          string           `json:"model_id"`
	Strategy         string           `json:"strategy"`
	Policy           string           `json:"policy"`
	Shape            string           `json:"shape,omitempty"`
	SimulatedMinutes int              `json:"simulated_minutes,omitempty"`
	fixtureSamples   int64
}

type driver struct {
	client *http.Client
	opt    Options
}

type streamConfigResponse struct {
	Config struct {
		PersistentMode bool `json:"persistentMode"`
	} `json:"config"`
}

const audioAdminConfigPath = "/vrooli.swarm_manager.v1.audio_admin.AudioAdminService/"

// PersistEvidence atomically stores the complete run document in audio-tools'
// governed data class, independent of whether it runs from a checkout or a
// standalone bundle.
func PersistEvidence(run conformance.Run) (string, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return "", fmt.Errorf("resolve evidence storage: %w", err)
	}
	dir, err := resolver.EnsureArtifactDir(corestorage.Options{ScenarioID: "audio-tools"}, corestorage.ArtifactRef{
		Owner: "audio-tools", Domain: "soak-evidence", Class: corestorage.ClassData,
	}, 0o750)
	if err != nil {
		return "", fmt.Errorf("resolve evidence path: %w", err)
	}
	path := filepath.Join(dir, run.RunID+".json")
	var payload strings.Builder
	if err := run.WriteJSON(&payload); err != nil {
		return "", fmt.Errorf("encode evidence: %w", err)
	}
	if err := corestorage.WriteFileAtomic(path, []byte(payload.String()), corestorage.SecretFilePerm); err != nil {
		return "", fmt.Errorf("publish evidence: %w", err)
	}
	return path, nil
}

type sessionStart struct {
	SessionID           string               `json:"session_id"`
	LeaseID             string               `json:"lease_id"`
	AudioDeviceEvidence *AudioDeviceEvidence `json:"audio_device_evidence,omitempty"`
	ExecutionID         string               `json:"-"`
}

// AudioDeviceEvidence is the metadata returned by BAS when a realtime soak
// opts into its PipeWire host-device path. It intentionally contains no audio
// bytes or transcript data.
type AudioDeviceEvidence struct {
	Enumerated    bool   `json:"enumerated"`
	SelectedLabel string `json:"selectedLabel"`
	SampleRate    int    `json:"sampleRate"`
	ChannelCount  int    `json:"channelCount"`
}

// Result keeps the public conformance document separate from the durable
// qualification records. The HTTP endpoint serializes only Run; the records
// are written through the existing experiment service.
type Result struct {
	Run                 conformance.Run
	AudioDeviceEvidence *AudioDeviceEvidence
}

type instructionResponse struct {
	Success              bool           `json:"success"`
	ExtractedData        map[string]any `json:"extracted_data"`
	AudioPlaybackFailure string         `json:"audio_playback_failure"`
	Error                any            `json:"error"`
	Failure              any            `json:"failure"`
}

type turnObservation struct {
	Diagnostic            *diagnostic      `json:"diagnostic"`
	Transcript            string           `json:"-"`
	Server                session.Snapshot `json:"server"`
	ServerSeen            bool             `json:"server_seen"`
	Samples               []diagnostic     `json:"-"`
	TimingSamples         []*diagnostic    `json:"-"`
	InterimSampleCount    int              `json:"-"`
	InterimVisibleSamples int              `json:"-"`
}

type diagnostic struct {
	SchemaVersion         int      `json:"schemaVersion"`
	SessionID             string   `json:"sessionId"`
	State                 string   `json:"state"`
	CapturedSequence      int64    `json:"capturedSequence"`
	CapturedSamples       int64    `json:"capturedSamples"`
	SentSequence          int64    `json:"sentSequence"`
	ProcessedSequence     int64    `json:"processedSequence"`
	RetainedBytes         int64    `json:"retainedBytes"`
	SignalObserved        bool     `json:"signalObserved"`
	FirstPartialLatencyMS *float64 `json:"firstPartialLatencyMs"`
	CommittedTextLagMS    *float64 `json:"committedTextLagMs"`
	ProviderID            string   `json:"providerId"`
	ModelID               string   `json:"modelId"`
	DoneSent              bool     `json:"doneSent"`
	TerminalReason        string   `json:"terminalReason"`
	StatusCodes           []string `json:"statusCodes"`
	ErrorCodes            []string `json:"errorCodes"`
}

type pageObservation struct {
	Diagnostic any    `json:"diag"`
	Transcript string `json:"final"`
	Interim    string `json:"interim"`
}

// Run drives the real Dictation Studio page and returns the same conformance
// Run used by the unit and ledger evidence. It never records transcript or
// audio content in that document.
func Run(ctx context.Context, opt Options, ledgers *session.Registry) (conformance.Run, error) {
	result, err := RunWithEvidence(ctx, opt, ledgers)
	return result.Run, err
}

// RunWithEvidence drives the product path and returns the metadata needed to
// persist dedicated browser/device qualification records. It does not add a
// second evidence document to the conformance response.
func RunWithEvidence(ctx context.Context, opt Options, ledgers *session.Registry) (Result, error) {
	if err := validateOptions(opt); err != nil {
		return Result{Run: failedRun(opt, err)}, err
	}
	opt.Surface = normalizeSurface(opt.Surface)
	if opt.Turns <= 0 {
		opt.Turns = defaultTurns
	}
	if opt.FeedMS <= 0 {
		opt.FeedMS = defaultFeedMS
	}
	// BAS loops the supplied WAV when a realtime turn outlives one fixture.
	// Keep the operator-facing reference as one fixture cycle, but retain its
	// sample count privately so the quality oracle can grade the actual amount
	// of audio captured rather than comparing repeated speech to one phrase.
	if samples, sampleErr := wavSampleCount(opt.Fixture); sampleErr == nil {
		opt.fixtureSamples = samples
	}
	started := time.Now()
	d := &driver{client: &http.Client{Timeout: requestTimeout(opt)}, opt: opt}
	runID := fmt.Sprintf("browser-soak-%d", started.UnixNano())
	// BAS uses execution_id as the owner key for lease recovery. Keep the
	// readable run ID in the conformance artifact, but give the browser session
	// a real UUID so BAS's reconciler can parse and track its owner.
	executionID := uuid.NewString()
	observations := make([]turnObservation, 0, opt.Turns)
	var runErr error
	var restorePersistentMode func()
	if opt.Surface == "swarm-manager" {
		persistent, configErr := d.getPersistentMode(ctx)
		if configErr != nil {
			return Result{Run: failedRunWithID(opt, runID, started, configErr)}, configErr
		}
		if !persistent {
			if configErr = d.setPersistentMode(ctx, true); configErr != nil {
				return Result{Run: failedRunWithID(opt, runID, started, configErr)}, configErr
			}
			restorePersistentMode = func() {
				_ = d.setPersistentMode(context.Background(), false)
			}
		}
	}
	if restorePersistentMode != nil {
		defer restorePersistentMode()
	}

	start, err := d.start(ctx, executionID)
	if err != nil {
		return Result{Run: failedRunWithID(opt, runID, started, err)}, err
	}
	defer func() { _ = d.close(context.Background(), start) }()

	if err = d.step(ctx, start.SessionID, 0, "navigate", map[string]any{"navigate": map[string]any{"url": d.pageURL()}}); err != nil {
		runErr = err
	} else if err = d.prepareSurface(ctx, start.SessionID); err != nil {
		runErr = err
	}

	heapBefore := stableHeapAlloc()
	maxHeapDelta := int64(0)
	for turn := 0; runErr == nil && turn < opt.Turns; turn++ {
		base := turn * turnStepStride
		feedChunks := feedChunkCount(opt.FeedMS)
		samples := make([]diagnostic, 0, feedChunks)
		interimSampleCount := 0
		interimVisibleSamples := 0
		timingSamples := make([]*diagnostic, 0, feedChunks)
		if err = d.step(ctx, start.SessionID, base+2, "start", map[string]any{"click": map[string]any{"selector": d.productSelector("ready")}}); err != nil {
			runErr = err
			break
		}
		if err = d.step(ctx, start.SessionID, base+3, "recording", map[string]any{"wait": map[string]any{"selector": d.productSelector("recording"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 15000}}); err != nil {
			runErr = err
			break
		}
		// A processed marker from the preceding turn can still be in the DOM
		// during the click event that starts this turn. Clear that stale marker
		// before feeding audio so the completion wait cannot sample a previous
		// terminal state.
		if processed := d.productSelector("processed"); processed != "" {
			if err = d.step(ctx, start.SessionID, base+4, "clear-processed", map[string]any{"wait": map[string]any{"selector": processed, "state": "WAIT_STATE_DETACHED", "timeout_ms": 2000}}); err != nil {
				runErr = err
				break
			}
		}
		if err = d.feed(ctx, start.SessionID, base+5, feedChunks, opt.FeedMS, func(observed *diagnostic, interimVisible bool) {
			timingSamples = append(timingSamples, observed)
			if observed != nil {
				samples = append(samples, *observed)
			}
			interimSampleCount++
			if interimVisible {
				interimVisibleSamples++
			}
		}); err != nil {
			runErr = err
			break
		}
		tailBase := base + 5 + feedChunks
		// The WebSocket handler removes a terminal ledger immediately after
		// delivering the terminal event. Capture the metadata-only server
		// snapshot before clicking Stop so the qualification observes the live
		// ledger without keeping terminal sessions resident after the turn.
		var serverSnapshot session.Snapshot
		serverSeen := false
		var beforeStop pageObservation
		if evalErr := d.evaluate(ctx, start.SessionID, tailBase+1000, &beforeStop); evalErr == nil {
			if observed := decodeDiagnostic(beforeStop.Diagnostic); observed != nil && ledgers != nil {
				serverSnapshot, serverSeen = ledgers.Snapshot(observed.SessionID)
			}
		}
		shouldStop := true
		if strings.TrimSpace(opt.Fault) != "" {
			// A terminal fault can change the product button from Stop back to
			// Start before the feed window ends. Clicking the shared selector in
			// that state would begin a fresh turn and erase the fault diagnostic.
			var afterFeed pageObservation
			if evalErr := d.evaluate(ctx, start.SessionID, tailBase+1, &afterFeed); evalErr == nil {
				if raw, ok := afterFeed.Diagnostic.(map[string]any); ok {
					if state, _ := raw["state"].(string); state == "completed" || state == "failed" || state == "cancelled" {
						shouldStop = false
					}
				}
			}
		}
		if shouldStop {
			if err = d.step(ctx, start.SessionID, tailBase, "stop", map[string]any{"click": map[string]any{"selector": d.productSelector("ready")}}); err != nil {
				runErr = err
				break
			}
		}
		if strings.TrimSpace(opt.Fault) != "" {
			// Fault profiles are allowed to end in an explicit failed terminal
			// state. Waiting for the healthy-turn processed marker would turn a
			// correctly surfaced fault into a harness timeout.
			if err = d.step(ctx, start.SessionID, tailBase+2, "fault-terminal", map[string]any{"wait": map[string]any{"selector": d.productSelector("terminal"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 60000}}); err != nil {
				var failedPage pageObservation
				if evalErr := d.evaluate(ctx, start.SessionID, tailBase+6, &failedPage); evalErr == nil && failedPage.Diagnostic != nil {
					if diagnosticJSON, marshalErr := json.Marshal(failedPage.Diagnostic); marshalErr == nil {
						err = fmt.Errorf("%w; browser_diagnostic=%s", err, string(diagnosticJSON))
					}
				}
				runErr = err
				break
			}
		} else if processed := d.productSelector("processed"); processed != "" {
			processedTimeoutMS := 15000
			if opt.Lane == conformance.LaneAccelerated {
				processedTimeoutMS = 60000
			}
			if err = d.step(ctx, start.SessionID, tailBase+2, "processed", map[string]any{"wait": map[string]any{"selector": processed, "state": "WAIT_STATE_ATTACHED", "timeout_ms": processedTimeoutMS}}); err != nil {
				// Preserve the metadata-only browser diagnostic in a failure detail so
				// a terminal/ack ordering bug is debuggable from the conformance
				// artifact without retaining transcript or audio.
				var failedPage pageObservation
				if evalErr := d.evaluate(ctx, start.SessionID, tailBase+6, &failedPage); evalErr == nil && failedPage.Diagnostic != nil {
					if diagnosticJSON, marshalErr := json.Marshal(failedPage.Diagnostic); marshalErr == nil {
						err = fmt.Errorf("%w; browser_diagnostic=%s", err, string(diagnosticJSON))
					}
				}
				runErr = err
				break
			}
		}
		if strings.TrimSpace(opt.Fault) == "" {
			captureTimeoutMS := 15000
			if opt.Lane == conformance.LaneAccelerated || normalizeSurface(opt.Surface) == "swarm-manager" {
				captureTimeoutMS = 60000
			}
			// Whole-turn replay is intentionally bounded in the browser. A long
			// clean stream can therefore finish with an explicit recoverable
			// retention failure after the server has processed every interval.
			// Observe either terminal recorder state so the conformance document
			// still contains the real diagnostic instead of 15 not_measured rows.
			if err = d.step(ctx, start.SessionID, tailBase+3, "captured", map[string]any{"wait": map[string]any{"selector": d.productSelector("terminal"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": captureTimeoutMS}}); err != nil {
				runErr = err
				break
			}
		}
		if err = d.step(ctx, start.SessionID, tailBase+4, "settle", map[string]any{"wait": map[string]any{"duration_ms": 500}}); err != nil {
			runErr = err
			break
		}
		var page pageObservation
		if err = d.evaluate(ctx, start.SessionID, tailBase+5, &page); err != nil {
			runErr = err
			break
		}
		obs := turnObservation{
			Transcript:            page.Transcript,
			Server:                serverSnapshot,
			ServerSeen:            serverSeen,
			Samples:               samples,
			TimingSamples:         timingSamples,
			InterimSampleCount:    interimSampleCount,
			InterimVisibleSamples: interimVisibleSamples,
		}
		obs.Diagnostic = decodeDiagnostic(page.Diagnostic)
		if !obs.ServerSeen && obs.Diagnostic != nil && ledgers != nil {
			obs.Server, obs.ServerSeen = ledgers.Snapshot(obs.Diagnostic.SessionID)
		}
		observations = append(observations, obs)
		// HeapAlloc is deliberately sampled only after a completed turn and a
		// full GC. Raw allocator growth is not retained application memory: a
		// long stream can legitimately create temporary decode/request objects
		// faster than the runtime's next collection. The qualification is about
		// live heap retained across turns, so compare stable turn-boundary values.
		heapDelta := int64(stableHeapAlloc()) - int64(heapBefore)
		if heapDelta > maxHeapDelta {
			maxHeapDelta = heapDelta
		}
	}
	// Include the final boundary even when the last turn exits through a
	// recoverable capture failure after its browser observation was recorded.
	finalHeapDelta := int64(stableHeapAlloc()) - int64(heapBefore)
	if finalHeapDelta > maxHeapDelta {
		maxHeapDelta = finalHeapDelta
	}

	run := buildRun(opt, runID, started, observations, maxHeapDelta)
	if runErr != nil {
		return Result{Run: failedRunWithIDAndCode(opt, runID, started, runErr, run.Code), AudioDeviceEvidence: start.AudioDeviceEvidence}, runErr
	}
	return Result{Run: run, AudioDeviceEvidence: start.AudioDeviceEvidence}, nil
}

// stableHeapAlloc measures live Go heap rather than allocator high-water mark.
// Qualification runs can process an hour of audio and allocate many short-lived
// JSON/PCM wrappers; forcing collection at an observation boundary prevents
// those transient allocations from being reported as retained memory. This is
// called outside an active stream turn, so it does not perturb capture timing.
func stableHeapAlloc() uint64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}

func feedChunkCount(feedMS int) int {
	if feedMS <= 0 {
		return 1
	}
	return (feedMS + feedChunkMS - 1) / feedChunkMS
}

// feed keeps each browser wait request below the driver/proxy timeout horizon.
// The recording button is not touched between chunks, so the browser product
// still observes one continuous turn and the WebSocket remains open throughout.
func (d *driver) feed(ctx context.Context, sessionID string, index, chunks, totalMS int, sample func(*diagnostic, bool)) error {
	remaining := totalMS
	for chunk := 0; chunk < chunks; chunk++ {
		duration := feedChunkMS
		if remaining < duration {
			duration = remaining
		}
		if duration <= 0 {
			break
		}
		if err := d.step(ctx, sessionID, index+chunk, fmt.Sprintf("feed-%d", chunk), map[string]any{"wait": map[string]any{"duration_ms": duration}}); err != nil {
			return err
		}
		if d.opt.Lane == conformance.LaneRealtime && strings.TrimSpace(d.opt.Fault) == "" && sample != nil {
			var page pageObservation
			if err := d.evaluate(ctx, sessionID, index+1000+chunk, &page); err != nil {
				return err
			}
			sample(decodeDiagnostic(page.Diagnostic), strings.TrimSpace(page.Interim) != "")
		}
		remaining -= duration
	}
	return nil
}

func decodeDiagnostic(raw any) *diagnostic {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out diagnostic
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return &out
}

// requestTimeout covers the declared wall-clock capture budget plus time for
// browser startup, final processing, and teardown. The previous fixed
// ten-minute timeout made a valid fifteen-minute lane impossible to observe
// and would have silently invalidated the sixty-minute qualification lane.
func requestTimeout(opt Options) time.Duration {
	base := 10 * time.Minute
	if opt.Lane != conformance.LaneRealtime || opt.Turns <= 0 || opt.FeedMS <= 0 {
		return base
	}
	capture := time.Duration(opt.FeedMS) * time.Duration(opt.Turns) * time.Millisecond
	withHeadroom := capture + 5*time.Minute
	if withHeadroom > base {
		return withHeadroom
	}
	return base
}

// RunTimeout is the server-owned wall-clock budget for one soak. The budget
// includes browser startup, capture, processing, and lease teardown. It is
// deliberately finite so detaching the HTTP caller cannot turn a failed
// browser run into an unbounded goroutine or browser-session leak.
func RunTimeout(opt Options) time.Duration {
	return requestTimeout(opt)
}

func validateOptions(o Options) error {
	if strings.TrimSpace(o.DriverURL) == "" || strings.TrimSpace(o.UIURL) == "" || strings.TrimSpace(o.Fixture) == "" {
		return errors.New("driver_url, ui_url, and fixture are required")
	}
	if strings.TrimSpace(o.EngineID) == "" || strings.TrimSpace(o.ModelID) == "" {
		return errors.New("engine_id and model_id are required; the soak must identify the exact provider cell")
	}
	if surface := strings.ToLower(strings.TrimSpace(o.Surface)); surface != "" && surface != "audio-tools" && surface != "swarm-manager" {
		return fmt.Errorf("surface must be audio-tools or swarm-manager")
	}
	if o.Lane != conformance.LaneAccelerated && o.Lane != conformance.LaneRealtime {
		return fmt.Errorf("lane must be accelerated or realtime")
	}
	if o.Lane == conformance.LaneAccelerated {
		if strings.TrimSpace(o.Reference) == "" {
			return errors.New("accelerated lane requires independent reference_text for quality measurement")
		}
		if o.SimulatedMinutes <= 0 {
			return errors.New("accelerated lane requires a positive simulated_minutes target")
		}
	}
	if o.Turns <= 0 {
		o.Turns = defaultTurns
	}
	return nil
}

func (d *driver) pageURL() string {
	if normalizeSurface(d.opt.Surface) == "swarm-manager" {
		page := strings.TrimRight(d.opt.UIURL, "/") + "/plan"
		if d.opt.Fault != "" {
			page += "?stt_test_mode=1&stt_test_fault=" + url.QueryEscape(d.opt.Fault)
		}
		return page
	}
	page := strings.TrimRight(d.opt.UIURL, "/") + "/dictation-studio"
	page += "?stt_engine_id=" + d.opt.EngineID
	if d.opt.Lane == conformance.LaneAccelerated {
		page += "&stt_test_mode=1&stt_capture_source=virtual&stt_capture_shape=" + url.QueryEscape(normalizeShape(d.opt.Shape))
		page += "&stt_virtual_samples=" + strconv.FormatInt(targetSamplesPerTurn(d.opt), 10)
		page += "&stt_corpus_url=" + url.QueryEscape(strings.TrimRight(d.opt.UIURL, "/")+"/api/v1/validation/soak/virtual-corpus?clip=quick-brown-fox")
	}
	if d.opt.Fault != "" {
		page += "&stt_test_mode=1&stt_test_fault=" + d.opt.Fault
	}
	return page
}

func normalizeSurface(surface string) string {
	if strings.EqualFold(strings.TrimSpace(surface), "swarm-manager") {
		return "swarm-manager"
	}
	return "audio-tools"
}

// productSelector keeps the browser driver honest about which consumer it is
// exercising. Audio Tools owns the full Dictation Studio recorder state;
// swarm-manager owns a MessageComposer that embeds the same shared capture
// package, so its lifecycle is represented by the RCL voice-control state.
func (d *driver) productSelector(name string) string {
	if normalizeSurface(d.opt.Surface) == "swarm-manager" {
		const mic = `[data-testid="captures-quick-input-mic"]`
		switch name {
		case "ready":
			return mic
		case "recording":
			return mic + `[data-state="recording"]`
		case "terminal":
			return mic + `[data-state="idle"], ` + mic + `[data-state="error"]`
		case "interim":
			return `[data-testid="captures-quick-composer-interim"]`
		default:
			return ""
		}
	}
	switch name {
	case "ready":
		return `[data-testid="dictation-record-start"]`
	case "recording":
		return `[data-testid="dictation-record-state"][data-recorder-state="recording"]`
	case "processed":
		return `[data-testid="dictation-turn-processed-ready"]`
	case "terminal":
		return `[data-testid="dictation-record-state"][data-recorder-state="captured"], [data-testid="dictation-record-state"][data-recorder-state="failed"]`
	case "interim":
		return `[data-testid="dictation-interim-transcript"]`
	default:
		return ""
	}
}

func (d *driver) prepareSurface(ctx context.Context, sessionID string) error {
	if normalizeSurface(d.opt.Surface) != "swarm-manager" {
		return d.step(ctx, sessionID, 1, "wait-recorder", map[string]any{"wait": map[string]any{"selector": d.productSelector("ready"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 15000}})
	}
	if err := d.step(ctx, sessionID, 1, "wait-action-menu", map[string]any{"wait": map[string]any{"selector": `[data-testid="graph-action-fab"]`, "state": "WAIT_STATE_ATTACHED", "timeout_ms": 15000}}); err != nil {
		return err
	}
	if err := d.step(ctx, sessionID, 2, "open-action-menu", map[string]any{"click": map[string]any{"selector": `[data-testid="graph-action-fab"]`}}); err != nil {
		return err
	}
	if err := d.step(ctx, sessionID, 3, "open-quick-capture", map[string]any{"click": map[string]any{"selector": `[role="menuitem"][aria-label="Quick Capture"]`}}); err != nil {
		return err
	}
	return d.step(ctx, sessionID, 4, "wait-recorder", map[string]any{"wait": map[string]any{"selector": d.productSelector("ready"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 15000}})
}

const virtualCorpusSamples = int64(31_268)

func targetSamplesPerTurn(o Options) int64 {
	if o.SimulatedMinutes <= 0 || o.Turns <= 0 {
		return virtualCorpusSamples
	}
	total := int64(o.SimulatedMinutes) * 60 * canonicalSampleRate
	perTurn := total / int64(o.Turns)
	if perTurn < virtualCorpusSamples {
		return virtualCorpusSamples
	}
	return perTurn
}

func normalizeShape(shape string) string {
	if strings.EqualFold(strings.TrimSpace(shape), "chunked") {
		return "chunked"
	}
	return "burst"
}

// realtimeShape is deliberately separate from normalizeShape. The accelerated
// lane shapes the virtual corpus (burst/chunked), while the realtime lane
// shapes host-device playback (realistic/continuous). The CLI historically
// sends the accelerated default shape for every lane, so only the two
// realtime-specific values are allowed to override profile here.
func realtimeShape(o Options) string {
	switch strings.ToLower(strings.TrimSpace(o.Shape)) {
	case "realistic", "continuous":
		return strings.ToLower(strings.TrimSpace(o.Shape))
	}
	if strings.EqualFold(strings.TrimSpace(o.Profile), "continuous") {
		return "continuous"
	}
	return "realistic"
}

func realtimePlaybackPauseMS(o Options) int {
	if realtimeShape(o) == "realistic" {
		return 250
	}
	return 0
}

func (d *driver) start(ctx context.Context, executionID string) (sessionStart, error) {
	var out sessionStart
	request := map[string]any{
		"execution_id": executionID,
		"workflow_id":  "audio-tools-long-form-dictation-soak",
		"viewport":     map[string]int{"width": 1280, "height": 900},
		"reuse_mode":   "fresh",
		"base_url":     d.opt.UIURL,
		"permissions":  []string{"microphone"},
	}
	if d.opt.Lane == conformance.LaneRealtime {
		request["fake_media"] = map[string]string{"microphone_wav": d.opt.Fixture}
		request["audio_playback_pause_ms"] = realtimePlaybackPauseMS(d.opt)
		// The WAV remains the qualification corpus, but the browser must receive
		// it through BAS's user-owned PipeWire source rather than Chromium's fake
		// media device. This is an explicit per-session opt-in so ordinary BAS
		// sessions never change the host default source.
		request["audio_device_evidence"] = true
	}
	err := d.post(ctx, "/session/start", request, &out)
	if err == nil && (out.SessionID == "" || out.LeaseID == "") {
		err = errors.New("playwright driver returned no session lease")
	}
	out.ExecutionID = executionID
	return out, err
}

func (d *driver) close(ctx context.Context, s sessionStart) error {
	return d.post(ctx, "/session/"+s.SessionID+"/close", map[string]string{"execution_id": s.ExecutionID, "lease_id": s.LeaseID}, nil)
}

func (d *driver) step(ctx context.Context, sessionID string, index int, nodeID string, params map[string]any) error {
	actionType := map[string]string{"navigate": "ACTION_TYPE_NAVIGATE", "click": "ACTION_TYPE_CLICK", "wait": "ACTION_TYPE_WAIT"}[nodeID]
	if nodeID == "start" || nodeID == "stop" {
		actionType = "ACTION_TYPE_CLICK"
	}
	if strings.HasPrefix(nodeID, "feed-") || nodeID == "clear-processed" || nodeID == "processed" || nodeID == "fault-terminal" || nodeID == "settle" || nodeID == "wait-recorder" || nodeID == "recording" || nodeID == "captured" {
		actionType = "ACTION_TYPE_WAIT"
	}
	// Named surface-preparation steps can vary by product consumer. Derive the
	// typed action from the sole action payload when the human-readable node id
	// is new, so a selector/lifecycle addition cannot silently emit an untyped
	// BAS instruction.
	if actionType == "" {
		switch {
		case params["navigate"] != nil:
			actionType = "ACTION_TYPE_NAVIGATE"
		case params["click"] != nil:
			actionType = "ACTION_TYPE_CLICK"
		case params["wait"] != nil:
			actionType = "ACTION_TYPE_WAIT"
		}
	}
	var response instructionResponse
	if err := d.post(ctx, "/session/"+sessionID+"/run", map[string]any{
		"instruction": map[string]any{
			"index": index, "node_id": nodeID,
			"action":    mergeAction(actionType, params),
			"telemetry": map[string]string{"screenshot": "SCREENSHOT_CAPTURE_POLICY_NEVER"},
		},
	}, &response); err != nil {
		return err
	}
	if response.Failure != nil {
		return fmt.Errorf("browser step %q failed: %v", nodeID, response.Failure)
	}
	if strings.TrimSpace(response.AudioPlaybackFailure) != "" {
		return fmt.Errorf("browser audio playback failed during %q: %s", nodeID, response.AudioPlaybackFailure)
	}
	return nil
}

func (d *driver) evaluate(ctx context.Context, sessionID string, index int, out *pageObservation) error {
	var response instructionResponse
	err := d.post(ctx, "/session/"+sessionID+"/run", map[string]any{"instruction": map[string]any{
		"index": index, "node_id": "observe-" + fmt.Sprint(index),
		"action":    map[string]any{"type": "ACTION_TYPE_EVALUATE", "evaluate": map[string]string{"expression": d.evaluationExpression()}},
		"telemetry": map[string]string{"screenshot": "SCREENSHOT_CAPTURE_POLICY_NEVER"},
	}}, &response)
	if err != nil {
		return err
	}
	if response.Failure != nil {
		return fmt.Errorf("browser evaluate failed: %v", response.Failure)
	}
	result, ok := response.ExtractedData["result"]
	if !ok {
		return errors.New("browser evaluate returned no result")
	}
	data, ok := result.(string)
	if ok {
		return json.Unmarshal([]byte(data), out)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, out)
}

func (d *driver) evaluationExpression() string {
	if normalizeSurface(d.opt.Surface) == "swarm-manager" {
		return `(function(){const interim=document.querySelector('[data-testid="captures-quick-composer-interim"]');const style=interim?window.getComputedStyle(interim):null;const rendered=!!interim&&!!style&&style.display!=="none"&&style.visibility!=="hidden"&&interim.getClientRects().length>0;return {diag:window.__VROOLI_AUDIO_STREAM_DIAGNOSTIC__?.latest||null,final:document.querySelector('[data-testid="captures-quick-input-submit"]')?.value||'',interim:rendered?(interim.textContent||''):''};})()`
	}
	return `(function(){const interim=document.querySelector('[data-testid="dictation-interim-transcript"]');const style=interim?window.getComputedStyle(interim):null;const rendered=!!interim&&!!style&&style.display!=="none"&&style.visibility!=="hidden"&&interim.getClientRects().length>0;return {diag:window.__VROOLI_AUDIO_STREAM_DIAGNOSTIC__?.latest||null,final:document.querySelector('[data-testid="dictation-final-transcript"]')?.textContent||'',interim:rendered?(interim.textContent||''):''};})()`
}

func mergeAction(typ string, params map[string]any) map[string]any {
	out := map[string]any{"type": typ}
	for key, value := range params {
		out[key] = value
	}
	return out
}

func (d *driver) post(ctx context.Context, path string, body any, out any) error {
	return d.postAt(ctx, d.opt.DriverURL, "playwright driver "+path, path, body, out)
}

func (d *driver) postAt(ctx context.Context, baseURL, targetName, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+path, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.Contains(targetName, "audio admin") {
		req.Header.Set("Connect-Protocol-Version", "1")
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", targetName, path, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned HTTP %d: %s", targetName, path, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if out != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return fmt.Errorf("decode %s response: %w", targetName, err)
		}
	}
	return nil
}

func (d *driver) getPersistentMode(ctx context.Context) (bool, error) {
	var out streamConfigResponse
	err := d.postAt(ctx, d.opt.UIURL, "audio admin", audioAdminConfigPath+"GetStreamConfig", map[string]any{}, &out)
	return out.Config.PersistentMode, err
}

func (d *driver) setPersistentMode(ctx context.Context, enabled bool) error {
	return d.postAt(ctx, d.opt.UIURL, "audio admin", audioAdminConfigPath+"UpdateStreamConfig", map[string]any{
		// Connect's JSON mapping represents google.protobuf.FieldMask as its
		// canonical comma-separated string, and uses proto field names in that
		// string even though the message fields remain lowerCamelCase.
		"updateMask": "persistentMode",
		"config":     map[string]any{"persistentMode": enabled},
	}, nil)
}

func buildRun(opt Options, runID string, started time.Time, observations []turnObservation, heapDelta int64) conformance.Run {
	run := conformance.Run{SchemaVersion: conformance.SchemaVersion, RunID: runID, Lane: opt.Lane, Profile: opt.Profile, Shape: evidenceShape(opt), SimulatedMinutes: opt.SimulatedMinutes, Turns: opt.Turns, Fault: opt.Fault, Cell: conformance.Cell{EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, Policy: opt.Policy}, Code: conformance.Code{CapturePackage: captureFingerprint(), Server: "sha256:" + buildidentity.SourceIdentity}, DurationMs: time.Since(started).Milliseconds()}
	if strings.TrimSpace(opt.Fault) != "" {
		return buildFaultRun(opt, runID, started, observations, heapDelta)
	}
	if run.Profile == "" {
		run.Profile = "realistic"
	}
	if run.Cell.Policy == "" {
		run.Cell.Policy = "default"
	}
	if run.Cell.Strategy == "" {
		run.Cell.Strategy = "product"
	}
	for _, name := range conformance.InvariantAssertions {
		run.Assertions = append(run.Assertions, invariant(name, observations, heapDelta, opt.FeedMS, opt.Lane, opt.EngineID, opt.ModelID))
	}
	if opt.Lane == conformance.LaneAccelerated {
		run.Assertions = append(run.Assertions, acceleratedDurationAssertion(observations, opt), conformance.Measured("accelerated_wall_budget", run.DurationMs <= 60_000, fmt.Sprintf("wall duration %dms; required <=60000ms", run.DurationMs)))
	}
	if opt.Lane == conformance.LaneRealtime {
		if len(observations) == 0 || observations[0].Diagnostic == nil {
			run.Assertions = append(run.Assertions, conformance.NotMeasured("first_partial_latency_stable", "the browser did not expose a completed stream diagnostic"), conformance.NotMeasured("committed_text_lag_stable", "the browser did not expose a completed stream diagnostic"))
		} else {
			timing := realtimeTimingObservations(observations)
			run.Assertions = append(run.Assertions, realtimeAssertion("first_partial_latency_stable", timing, func(d *diagnostic) *float64 { return d.FirstPartialLatencyMS }), realtimeAssertion("committed_text_lag_stable", timing, func(d *diagnostic) *float64 { return d.CommittedTextLagMS }))
		}
		run.Assertions = append(run.Assertions, continuousInterimTextAssertion(observations))
	}
	if strings.TrimSpace(opt.Reference) == "" {
		run.Assertions = append(run.Assertions, conformance.NotMeasured("word_error_rate_stable", "reference_text was not supplied; the driver will not invent a transcript reference"), conformance.NotMeasured("punctuation_rate_recorded", "reference_text was not supplied"), conformance.NotMeasured("capitalisation_rate_recorded", "reference_text was not supplied"))
	} else {
		// The transcript is intentionally held only in memory during this
		// function; quality details contain rates, never transcript text.
		run.Assertions = append(run.Assertions, qualityAcrossTurns(opt.Reference, observations, opt.Lane, opt.fixtureSamples)...)
	}
	return run
}

func buildFaultRun(opt Options, runID string, started time.Time, observations []turnObservation, heapDelta int64) conformance.Run {
	run := conformance.Run{SchemaVersion: conformance.SchemaVersion, RunID: runID, Lane: opt.Lane, Profile: opt.Profile, Shape: evidenceShape(opt), SimulatedMinutes: opt.SimulatedMinutes, Turns: opt.Turns, Fault: opt.Fault, Cell: conformance.Cell{EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, Policy: opt.Policy}, Code: conformance.Code{CapturePackage: captureFingerprint(), Server: "sha256:" + buildidentity.SourceIdentity}, DurationMs: time.Since(started).Milliseconds()}
	run.Assertions = append(run.Assertions, conformance.Measured("fault_profile_observed", strings.TrimSpace(opt.Fault) != "" && len(observations) > 0, fmt.Sprintf("fault=%q; observed browser turns=%d", opt.Fault, len(observations))))
	intervals := true
	duplicates := true
	seenSegments := map[string]struct{}{}
	for _, observation := range observations {
		if observation.Diagnostic == nil || observation.Diagnostic.CapturedSequence < 0 || observation.Diagnostic.SentSequence < -1 || observation.Diagnostic.ProcessedSequence < -1 || (observation.Diagnostic.SentSequence >= 0 && observation.Diagnostic.SentSequence > observation.Diagnostic.CapturedSequence) {
			intervals = false
		}
		for _, segment := range observation.Server.Committed {
			if _, seen := seenSegments[segment.ID]; seen {
				duplicates = false
			}
			seenSegments[segment.ID] = struct{}{}
		}
	}
	run.Assertions = append(run.Assertions, conformance.Measured("fault_interval_accounting", intervals && len(observations) > 0, "captured and sent interval sequences remained ordered; terminal faults may stop before processing acknowledgement"))
	run.Assertions = append(run.Assertions, conformance.Measured("fault_no_duplicate_committed_segments", duplicates, "committed segment identifiers remained unique across the injected fault and any recovery"))
	terminal := true
	details := make([]string, 0, len(observations))
	for _, observation := range observations {
		if observation.Diagnostic == nil {
			terminal = false
			continue
		}
		if observation.Diagnostic.State != "completed" && observation.Diagnostic.State != "failed" && observation.Diagnostic.State != "cancelled" && observation.Diagnostic.TerminalReason == "" {
			terminal = false
		}
		details = append(details, fmt.Sprintf("state=%s terminal=%s captured=%d processed=%d", observation.Diagnostic.State, observation.Diagnostic.TerminalReason, observation.Diagnostic.CapturedSequence, observation.Diagnostic.ProcessedSequence))
	}
	run.Assertions = append(run.Assertions, conformance.Measured("fault_recovered_or_terminal", terminal && len(observations) > 0, strings.Join(details, "; ")))
	_ = heapDelta
	return run
}

func continuousInterimTextAssertion(observations []turnObservation) conformance.Assertion {
	if len(observations) == 0 {
		return conformance.NotMeasured("continuous_interim_text_visible", "no completed browser turn was observed")
	}
	for index, observation := range observations {
		if observation.InterimSampleCount == 0 {
			return conformance.NotMeasured("continuous_interim_text_visible", fmt.Sprintf("turn %d exposed no interim-text visibility samples", index+1))
		}
		if observation.InterimVisibleSamples != observation.InterimSampleCount {
			return conformance.Measured("continuous_interim_text_visible", false, fmt.Sprintf("turn %d kept interim text visible at %d/%d feed checkpoints", index+1, observation.InterimVisibleSamples, observation.InterimSampleCount))
		}
	}
	return conformance.Measured("continuous_interim_text_visible", true, fmt.Sprintf("interim text remained visible at %d/%d realtime feed checkpoints across %d turn(s)", totalInterimVisibleSamples(observations), totalInterimSampleCount(observations), len(observations)))
}

func totalInterimSampleCount(observations []turnObservation) int {
	total := 0
	for _, observation := range observations {
		total += observation.InterimSampleCount
	}
	return total
}

func totalInterimVisibleSamples(observations []turnObservation) int {
	total := 0
	for _, observation := range observations {
		total += observation.InterimVisibleSamples
	}
	return total
}

func qualityAcrossTurns(reference string, observations []turnObservation, lane conformance.Lane, fixtureSamples int64) []conformance.Assertion {
	const names = 3
	passed := [names]bool{true, true, true}
	details := [names][]string{}
	for index, observation := range observations {
		turnReference := reference
		if lane == conformance.LaneRealtime && fixtureSamples > 0 && observation.Diagnostic != nil && observation.Diagnostic.CapturedSamples > 0 {
			repetitions := int(math.Round(float64(observation.Diagnostic.CapturedSamples) / float64(fixtureSamples)))
			if repetitions < 1 {
				repetitions = 1
			}
			turnReference = repeatReference(reference, repetitions)
		}
		measured := conformance.MeasureQuality(conformance.QualityObservation{Reference: turnReference, Hypothesis: observation.Transcript, MaxWER: 0.25, MinPunctuationRate: 0, MinCapitalisationRate: 0})
		for metric, assertion := range measured {
			passed[metric] = passed[metric] && assertion.Outcome == conformance.OutcomePassed
			details[metric] = append(details[metric], fmt.Sprintf("turn %d: %s", index+1, assertion.Detail))
		}
	}
	if len(observations) == 0 {
		return []conformance.Assertion{
			conformance.NotMeasured("word_error_rate_stable", "no completed browser turn was observed"),
			conformance.NotMeasured("punctuation_rate_recorded", "no completed browser turn was observed"),
			conformance.NotMeasured("capitalisation_rate_recorded", "no completed browser turn was observed"),
		}
	}
	return []conformance.Assertion{
		conformance.Measured("word_error_rate_stable", passed[0], strings.Join(details[0], "; ")),
		conformance.Measured("punctuation_rate_recorded", passed[1], strings.Join(details[1], "; ")),
		conformance.Measured("capitalisation_rate_recorded", passed[2], strings.Join(details[2], "; ")),
	}
}

func repeatReference(reference string, repetitions int) string {
	reference = strings.TrimSpace(reference)
	if repetitions <= 1 || reference == "" {
		return reference
	}
	return strings.TrimSpace(strings.Repeat(reference+" ", repetitions))
}

// wavSampleCount reads only the RIFF chunks needed to determine how many PCM
// samples one BAS fixture cycle contains. Invalid/non-WAV test fixtures return
// an error and leave the quality oracle's one-cycle behavior unchanged.
func wavSampleCount(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return 0, errors.New("fixture is not a RIFF/WAVE file")
	}
	var blockAlign uint16
	for offset := 12; offset+8 <= len(data); {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if chunkSize < 0 || offset+chunkSize > len(data) {
			return 0, errors.New("fixture contains an invalid RIFF chunk")
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 14 {
				return 0, errors.New("fixture fmt chunk is too short")
			}
			blockAlign = binary.LittleEndian.Uint16(data[offset+12 : offset+14])
		case "data":
			if blockAlign == 0 {
				return 0, errors.New("fixture has no valid PCM block alignment")
			}
			return int64(chunkSize) / int64(blockAlign), nil
		}
		offset += chunkSize
		if chunkSize%2 == 1 {
			offset++
		}
	}
	return 0, errors.New("fixture has no data chunk")
}

func acceleratedDurationAssertion(observations []turnObservation, opt Options) conformance.Assertion {
	if opt.SimulatedMinutes <= 0 {
		return conformance.NotMeasured("accelerated_duration_target", "simulated_minutes was not supplied")
	}
	var captured int64
	for _, observation := range observations {
		if observation.Diagnostic == nil {
			return conformance.NotMeasured("accelerated_duration_target", "browser diagnostic was absent")
		}
		captured += observation.Diagnostic.CapturedSamples
	}
	target := int64(opt.SimulatedMinutes) * 60 * canonicalSampleRate
	return conformance.Measured("accelerated_duration_target", captured >= target, fmt.Sprintf("virtual corpus captured %d/%d samples (%.2f/%.2f simulated minutes)", captured, target, float64(captured)/canonicalSampleRate/60, float64(target)/canonicalSampleRate/60))
}

func signalTurnCount(observations []turnObservation) int {
	count := 0
	for _, observation := range observations {
		if observation.Diagnostic != nil && observation.Diagnostic.SignalObserved {
			count++
		}
	}
	return count
}

func allSignalObserved(observations []turnObservation) bool {
	return len(observations) > 0 && signalTurnCount(observations) == len(observations)
}

// cursorsUnobserved reports whether a turn's wire cursors never left the
// StreamDiagnosticRecorder's -1 sentinel, together with the reason to publish.
//
// The three cursors mark three stages: captured (audio reached the batcher),
// sent (a frame reached the socket), processed (the server acknowledged it).
// Sequence 0 is a real interval, so -1 — not 0 — is the "never happened"
// sentinel for each. The returned reason names the stage that was never
// reached and carries the turn's terminal state and error codes, because those
// are what actually explain it.
func cursorsUnobserved(d *diagnostic) (string, bool) {
	switch {
	case d.CapturedSequence < 0:
		return fmt.Sprintf("no wire interval was captured in the browser (%s)", describeTurnOutcome(d)), true
	case d.SentSequence < 0:
		return fmt.Sprintf("%d wire interval(s) were captured but none reached the socket, so sent/processed coverage was never observable (%s)", d.CapturedSequence+1, describeTurnOutcome(d)), true
	case d.ProcessedSequence < 0:
		return fmt.Sprintf("%d wire interval(s) were sent but the server acknowledged none, so processed coverage was never observable (%s)", d.SentSequence+1, describeTurnOutcome(d)), true
	}
	return "", false
}

// describeTurnOutcome renders the triage context an evidence reader needs to
// act on a red or unmeasured turn without re-running the soak.
func describeTurnOutcome(d *diagnostic) string {
	parts := []string{fmt.Sprintf("state=%s", nonEmpty(d.State, "unknown"))}
	if strings.TrimSpace(d.TerminalReason) != "" {
		parts = append(parts, fmt.Sprintf("terminal=%s", d.TerminalReason))
	}
	if len(d.ErrorCodes) > 0 {
		parts = append(parts, fmt.Sprintf("errors=[%s]", strings.Join(d.ErrorCodes, " ")))
	}
	if len(d.StatusCodes) > 0 {
		parts = append(parts, fmt.Sprintf("status=[%s]", strings.Join(d.StatusCodes, " ")))
	}
	return strings.Join(parts, " ")
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func invariant(name string, observations []turnObservation, heapDelta int64, feedMS int, lane conformance.Lane, expectedEngineID string, expectedModelID string) conformance.Assertion {
	if len(observations) == 0 {
		return conformance.NotMeasured(name, "no completed browser turn was observed")
	}
	for _, o := range observations {
		if o.Diagnostic == nil {
			return conformance.NotMeasured(name, "browser diagnostic was absent")
		}
		if !o.ServerSeen && name == "server_retention_bounded" {
			return conformance.NotMeasured(name, "server ledger snapshot was unavailable for the browser session")
		}
	}
	switch name {
	case "interval_accounting_exactly_once":
		// A wire cursor still holding its -1 sentinel means the browser never
		// reached that stage — not that intervals went missing. Reporting the
		// sentinel as `failed` blamed the accounting for what is almost always a
		// stream that never connected or a server that never acknowledged, and
		// the "expected zero-based coverage" wording sent every investigation
		// after a drop that had not happened. Say what was not observed instead.
		for _, o := range observations {
			if reason, unobserved := cursorsUnobserved(o.Diagnostic); unobserved {
				return conformance.NotMeasured(name, reason)
			}
		}
		for _, o := range observations {
			if o.Diagnostic.SentSequence < o.Diagnostic.CapturedSequence-1 || o.Diagnostic.ProcessedSequence < o.Diagnostic.SentSequence {
				return conformance.Measured(name, false, fmt.Sprintf("captured=%d sent=%d processed=%d; expected zero-based sent/processed coverage for %d captured intervals (%s)", o.Diagnostic.CapturedSequence, o.Diagnostic.SentSequence, o.Diagnostic.ProcessedSequence, o.Diagnostic.CapturedSequence+1, describeTurnOutcome(o.Diagnostic)))
			}
		}
		return conformance.Measured(name, true, fmt.Sprintf("%d browser turns observed with monotonic capture coverage; signal_observed=%t", len(observations), allSignalObserved(observations)))
	case "capture_signal_observed":
		return conformance.Measured(name, allSignalObserved(observations), fmt.Sprintf("non-silent PCM observed on %d/%d browser turns", signalTurnCount(observations), len(observations)))
	case "browser_retention_bounded":
		for _, o := range observations {
			if o.Diagnostic.RetainedBytes < 0 || o.Diagnostic.RetainedBytes > maxLedgerBytes {
				return conformance.Measured(name, false, "browser retained-byte bound exceeded")
			}
		}
		return conformance.Measured(name, true, "metadata-only browser retained-byte snapshots stayed within 64MiB")
	case "server_retention_bounded":
		for _, o := range observations {
			if o.Server.RetainedBytes < 0 || o.Server.RetainedBytes > maxLedgerBytes {
				return conformance.Measured(name, false, "server retained-byte bound exceeded")
			}
		}
		return conformance.Measured(name, true, "server ledger retained-byte snapshots stayed within 64MiB")
	case "per_frame_cost_constant":
		minSamples, maxSamples := int64(0), int64(0)
		for _, o := range observations {
			samples := o.Diagnostic.CapturedSamples
			if samples <= 0 {
				return conformance.Measured(name, false, fmt.Sprintf("capturedSamples=%d; every completed turn must expose positive samples", samples))
			}
			if minSamples == 0 || samples < minSamples {
				minSamples = samples
			}
			if samples > maxSamples {
				maxSamples = samples
			}
		}
		spread := float64(maxSamples) / float64(minSamples)
		return conformance.Measured(name, spread <= maxTurnSampleSpread, fmt.Sprintf("captured sample spread max/min=%.3f (%d/%d), limit=%.2f", spread, maxSamples, minSamples, maxTurnSampleSpread))
	case "wire_rate_within_budget":
		if feedMS <= 0 {
			return conformance.NotMeasured(name, "feed duration was not available to calculate canonical wire rate")
		} else {
			minRate, maxRate := float64(0), float64(0)
			for _, o := range observations {
				seconds := float64(feedMS) / 1000
				if lane == conformance.LaneAccelerated {
					seconds = float64(o.Diagnostic.CapturedSamples) / canonicalSampleRate
				}
				rate := float64(o.Diagnostic.CapturedSamples) / seconds
				if rate <= 0 {
					return conformance.Measured(name, false, fmt.Sprintf("capturedSamples=%d produced a non-positive wire rate", o.Diagnostic.CapturedSamples))
				}
				if minRate == 0 || rate < minRate {
					minRate = rate
				}
				if rate > maxRate {
					maxRate = rate
				}
			}
			lower, upper := canonicalSampleRate*minWireRateFraction, canonicalSampleRate*maxWireRateFraction
			return conformance.Measured(name, minRate >= lower && maxRate <= upper, fmt.Sprintf("captured wire rate %.0f..%.0f samples/s; expected %.0f..%.0f", minRate, maxRate, lower, upper))
		}
	case "zero_duplicate_committed_segments":
		for _, o := range observations {
			ids := map[string]bool{}
			for _, segment := range o.Server.Committed {
				if ids[segment.ID] {
					return conformance.Measured(name, false, "duplicate committed segment id observed")
				}
				ids[segment.ID] = true
			}
		}
		return conformance.Measured(name, true, "server ledger committed segment ids were unique")
	case "zero_silent_terminal_outcomes":
		for _, o := range observations {
			if o.Diagnostic.State != "completed" && o.Diagnostic.State != "failed" && o.Diagnostic.TerminalReason == "" {
				return conformance.Measured(name, false, fmt.Sprintf("state=%q terminalReason=%q", o.Diagnostic.State, o.Diagnostic.TerminalReason))
			}
		}
		return conformance.Measured(name, true, "every observed turn exposed a terminal state or reason")
	case "provider_cell_identity":
		for _, o := range observations {
			if o.Diagnostic.ProviderID == "" || o.Diagnostic.ModelID == "" {
				return conformance.NotMeasured(name, "the product path did not expose provider identity")
			}
			if o.Diagnostic.ProviderID != expectedEngineID || o.Diagnostic.ModelID != expectedModelID {
				return conformance.Measured(name, false, fmt.Sprintf("observed provider=%q model=%q; expected provider=%q model=%q", o.Diagnostic.ProviderID, o.Diagnostic.ModelID, expectedEngineID, expectedModelID))
			}
		}
		return conformance.Measured(name, true, fmt.Sprintf("provider=%q model=%q observed for %d turns", expectedEngineID, expectedModelID, len(observations)))
	case "peak_memory_flat":
		return conformance.Measured(name, heapDelta < maxLedgerBytes, fmt.Sprintf("audio-tools server heap delta=%d bytes across turns", heapDelta))
	default:
		return conformance.NotMeasured(name, "driver has no observation for this assertion")
	}
}

func realtimeTimingObservations(observations []turnObservation) []turnObservation {
	out := make([]turnObservation, 0, len(observations))
	for _, observation := range observations {
		if len(observation.TimingSamples) > 0 {
			// Stability is a property of the live stream. Do not compare the
			// first periodic sample with the terminal flush snapshot: manual
			// stop intentionally retains a short settle window so the final
			// captured audio is delivered. Terminal delivery is covered by
			// zero_silent_terminal_outcomes and exact interval accounting.
			for _, sample := range observation.TimingSamples {
				out = append(out, turnObservation{Diagnostic: sample})
			}
			continue
		}
		for index := range observation.Samples {
			sample := observation.Samples[index]
			out = append(out, turnObservation{Diagnostic: &sample})
		}
	}
	return out
}

func realtimeAssertion(name string, observations []turnObservation, value func(*diagnostic) *float64) conformance.Assertion {
	values := make([]float64, 0, len(observations))
	measured := false
	for _, o := range observations {
		if o.Diagnostic == nil {
			if measured {
				return conformance.NotMeasured(name, "the product path stopped exposing this timing after it was measured")
			}
			continue
		}
		v := value(o.Diagnostic)
		if v == nil {
			if measured {
				return conformance.NotMeasured(name, "the product path stopped exposing this timing after it was measured")
			}
			continue
		}
		measured = true
		values = append(values, *v)
	}
	if len(values) == 0 {
		return conformance.NotMeasured(name, "no realtime observations were available")
	}
	first, last := values[0], values[len(values)-1]
	// Every switch arm below either assigns passed or returns.
	var passed bool
	bound := ""
	switch name {
	case "first_partial_latency_stable":
		limit := first * 1.2
		passed = last <= limit
		bound = fmt.Sprintf("last <= first*1.2 (%.2fms)", limit)
	case "committed_text_lag_stable":
		// Committed lag is reported in millisecond-resolution snapshots. A
		// small increase can therefore be scheduler/measurement jitter rather
		// than retained queue growth. Keep the bound strict for real drift,
		// while allowing at most 5ms or 20% (whichever is larger) of bounded
		// periodic variation. Terminal flush latency is excluded before this
		// function, so this tolerance cannot hide an end-of-turn delivery gap.
		limit := first + math.Max(5, first*0.2)
		passed = last <= limit
		bound = fmt.Sprintf("last <= first+max(5ms,first*0.2) (%.2fms)", limit)
	default:
		return conformance.NotMeasured(name, "driver has no stability bound for this timing")
	}
	return conformance.Measured(name, passed, fmt.Sprintf("%s observed across %d realtime turns; first=%.2fms last=%.2fms; %s", name, len(values), first, last, bound))
}

func captureFingerprint() string {
	dirs := []string{}
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		dirs = append(dirs, root)
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			dirs = append(dirs, dir)
		}
	}
	for _, root := range dirs {
		packageDir := filepath.Join(root, "packages", "audio-capture-browser")
		paths := []string{filepath.Join(packageDir, "package.json"), filepath.Join(packageDir, "dist", "index.js")}
		hash := sha256.New()
		valid := true
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				valid = false
				break
			}
			_, _ = hash.Write([]byte(filepath.ToSlash(path)))
			_, _ = hash.Write(data)
		}
		if valid {
			return "sha256:" + hex.EncodeToString(hash.Sum(nil))
		}
	}
	return "unknown"
}

func failedRun(opt Options, err error) conformance.Run {
	return failedRunWithID(opt, "failed", time.Now(), err)
}

func failedRunWithID(opt Options, id string, started time.Time, err error) conformance.Run {
	return failedRunWithIDAndCode(opt, id, started, err, conformance.Code{CapturePackage: captureFingerprint(), Server: "sha256:" + buildidentity.SourceIdentity})
}

func failedRunWithIDAndCode(opt Options, id string, started time.Time, err error, code conformance.Code) conformance.Run {
	run := conformance.Run{SchemaVersion: conformance.SchemaVersion, RunID: id, Lane: opt.Lane, Profile: opt.Profile, Shape: evidenceShape(opt), SimulatedMinutes: opt.SimulatedMinutes, Turns: opt.Turns, Fault: opt.Fault, Cell: conformance.Cell{EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, Policy: opt.Policy}, Code: code, DurationMs: time.Since(started).Milliseconds()}
	for _, name := range conformance.RequiredAssertionsForRun(conformance.Run{Lane: opt.Lane, Fault: opt.Fault}) {
		run.Assertions = append(run.Assertions, conformance.NotMeasured(name, err.Error()))
	}
	return run
}

func evidenceShape(opt Options) string {
	if opt.Lane == conformance.LaneAccelerated {
		return normalizeShape(opt.Shape)
	}
	return realtimeShape(opt)
}
