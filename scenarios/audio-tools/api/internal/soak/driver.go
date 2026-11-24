//go:build audio_lab

// Package soak owns the one product-path long-form dictation driver. It is
// deliberately behind the API's leased test-isolation boundary: this code can
// exercise the real browser, WebSocket, ledger, and UI without becoming a
// second production capture implementation.
package soak

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"audio-tools/internal/buildidentity"
	"audio-tools/internal/conformance"
	"audio-tools/internal/stt/session"

	corestorage "github.com/vrooli/api-core/storage"
)

const (
	maxLedgerBytes              = 64 << 20
	defaultTurns                = 3
	defaultFeedMS               = 4000
	feedChunkMS                 = 60_000
	turnStepStride              = 4096
	canonicalSampleRate         = 16_000
	minWireRateFraction         = 0.50
	maxWireRateFraction         = 1.50
	maxCaptureCallbackCostRatio = 0.50
)

// Options describes one broker-owned soak. The UI URL and fixed fixture are
// resolved by the scenario owner and are never accepted from a caller.
type Options struct {
	UIURL            string           `json:"-"`
	Surface          string           `json:"surface,omitempty"`
	Fixture          string           `json:"-"`
	Lane             conformance.Lane `json:"lane"`
	Profile          string           `json:"profile"`
	Turns            int              `json:"turns"`
	FeedMS           int              `json:"feed_ms"`
	Fault            string           `json:"fault,omitempty"`
	Reference        string           `json:"reference_text,omitempty"`
	ReferenceCycleMS int              `json:"reference_cycle_ms,omitempty"`
	EngineID         string           `json:"engine_id"`
	ModelID          string           `json:"model_id"`
	Strategy         string           `json:"strategy"`
	Policy           string           `json:"policy"`
	Shape            string           `json:"shape,omitempty"`
	SimulatedMinutes int              `json:"simulated_minutes,omitempty"`
	MobileRecovery   bool             `json:"mobile_recovery,omitempty"`
	fixtureSamples   int64
}

type driver struct {
	client               *http.Client
	opt                  Options
	owner                *browserOwnerCredentials
	mobileRecoveryPassed bool
}

type browserOwnerCredentialsKey struct{}

type browserOwnerCredentials struct {
	identityToken string
	grantReceipt  string
	testLease     string
	brokerURL     string
}

// WithBrowserOwnerCredentials carries only request-scoped bearer receipts.
// They are never accepted in soak options or persisted evidence.
func WithBrowserOwnerCredentials(ctx context.Context, identityToken, grantReceipt, testLease string) context.Context {
	return context.WithValue(ctx, browserOwnerCredentialsKey{}, browserOwnerCredentials{
		identityToken: strings.TrimSpace(identityToken), grantReceipt: strings.TrimSpace(grantReceipt), testLease: strings.TrimSpace(testLease),
	})
}

// WithManagedBrowserOwner records the server-selected BAS endpoint for a
// managed soak. Neither URL nor permission profile is accepted from caller JSON.
func WithManagedBrowserOwner(ctx context.Context, brokerURL string) context.Context {
	credentials, _ := ctx.Value(browserOwnerCredentialsKey{}).(browserOwnerCredentials)
	credentials.brokerURL = strings.TrimRight(strings.TrimSpace(brokerURL), "/")
	return context.WithValue(ctx, browserOwnerCredentialsKey{}, credentials)
}

// FixedDictationFixture resolves the repository-owned Audio Tools reference
// WAV. Managed browser requests never choose a fixture path.
func FixedDictationFixture(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" || !filepath.IsAbs(root) {
		return "", errors.New("Vrooli root is required to resolve the fixed Audio Tools fixture")
	}
	path := filepath.Join(root, "scenarios", "audio-tools", "bas", "fixtures", "dictation-reference.wav")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("fixed Audio Tools dictation fixture is unavailable")
	}
	return path, nil
}

func browserOwnerCredentialsFromContext(ctx context.Context) (browserOwnerCredentials, bool) {
	credentials, ok := ctx.Value(browserOwnerCredentialsKey{}).(browserOwnerCredentials)
	return credentials, ok && credentials.identityToken != "" && credentials.grantReceipt != "" && credentials.testLease != "" && credentials.brokerURL != ""
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
	AudioDeviceEvidence *AudioDeviceEvidence `json:"audio_device_evidence,omitempty"`
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
	Run                  conformance.Run
	AudioDeviceEvidence  *AudioDeviceEvidence
	MobileRecoveryPassed bool
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
	SchemaVersion               int       `json:"schemaVersion"`
	SessionID                   string    `json:"sessionId"`
	State                       string    `json:"state"`
	CapturedSequence            int64     `json:"capturedSequence"`
	CapturedSamples             int64     `json:"capturedSamples"`
	CaptureSampleRate           float64   `json:"captureSampleRate"`
	CaptureCallbackCount        int       `json:"captureCallbackCount"`
	CaptureCallbackMaxCostRatio *float64  `json:"captureCallbackMaxCostRatio"`
	SentSequence                int64     `json:"sentSequence"`
	ProcessedSequence           int64     `json:"processedSequence"`
	RetainedBytes               int64     `json:"retainedBytes"`
	SignalObserved              bool      `json:"signalObserved"`
	StartRequestedAtMS          *float64  `json:"startRequestedAtMs"`
	CaptureReadyAtMS            *float64  `json:"captureReadyAtMs"`
	StartToCaptureReadyMS       *float64  `json:"startToCaptureReadyMs"`
	CaptureStartedAtMS          *float64  `json:"captureStartedAtMs"`
	FirstCaptureAtMS            *float64  `json:"firstCaptureAtMs"`
	LastCaptureAtMS             *float64  `json:"lastCaptureAtMs"`
	FirstSignalAtMS             *float64  `json:"firstSignalAtMs"`
	FirstSentAtMS               *float64  `json:"firstSentAtMs"`
	FirstPartialAtMS            *float64  `json:"firstPartialAtMs"`
	FirstPartialLatencyMS       *float64  `json:"firstPartialLatencyMs"`
	VisibleUpdateGapsMS         []float64 `json:"visibleUpdateGapsMs"`
	VisibleRevisionCount        int       `json:"visibleRevisionCount"`
	CommittedTextLagMS          *float64  `json:"committedTextLagMs"`
	StopToFinalTailMS           *float64  `json:"stopToFinalTailMs"`
	ProviderID                  string    `json:"providerId"`
	ModelID                     string    `json:"modelId"`
	ProviderTier                string    `json:"providerTier"`
	DoneSent                    bool      `json:"doneSent"`
	TerminalReason              string    `json:"terminalReason"`
	StatusCodes                 []string  `json:"statusCodes"`
	ErrorCodes                  []string  `json:"errorCodes"`
}

type pageObservation struct {
	Diagnostic             any    `json:"diag"`
	Transcript             string `json:"final"`
	Interim                string `json:"interim"`
	CoverageVisible        bool   `json:"coverage_visible"`
	ExportAvailable        bool   `json:"export_available"`
	DiagnosticMetadataOnly bool   `json:"diagnostic_metadata_only"`
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
	owner, ok := browserOwnerCredentialsFromContext(ctx)
	if !ok {
		err := errors.New("browser soak requires request-scoped Agent Manager and Test Genie receipts plus the server-selected BAS owner")
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
	d := &driver{client: &http.Client{Timeout: requestTimeout(opt)}, opt: opt, owner: &owner}
	runID := fmt.Sprintf("browser-soak-%d", started.UnixNano())
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
				cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cleanupCancel()
				_ = d.setPersistentMode(cleanupCtx, false)
			}
		}
	}
	if restorePersistentMode != nil {
		defer restorePersistentMode()
	}

	start, err := d.start(ctx)
	if err != nil {
		return Result{Run: failedRunWithID(opt, runID, started, err)}, err
	}
	// Cleanup must not hold a completed owner open forever when the browser
	// driver has a stale session or a background audio page. The conformance
	// result is assembled before this deferred cleanup runs, so bound cleanup
	// independently of the evidence request's potentially long timeout.
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		_ = d.close(closeCtx, start)
	}()

	if err = d.step(ctx, start.SessionID, 0, "navigate", map[string]any{"navigate": map[string]any{"url": d.pageURL()}}); err != nil {
		runErr = err
	} else if err = d.prepareSurface(ctx, start.SessionID); err != nil {
		runErr = err
	}

	observations, maxHeapDelta, runErr := runTurns(ctx, d, start, ledgers, opt, runErr)

	run := buildRun(opt, runID, started, observations, maxHeapDelta)
	if runErr != nil {
		return Result{Run: failedRunWithIDAndCode(opt, runID, started, runErr, run.Code), AudioDeviceEvidence: start.AudioDeviceEvidence, MobileRecoveryPassed: d.mobileRecoveryPassed}, runErr
	}
	return Result{Run: run, AudioDeviceEvidence: start.AudioDeviceEvidence, MobileRecoveryPassed: d.mobileRecoveryPassed}, nil
}

// runTurns owns the ordered browser turn sequence and its heap observations.
func requestTimeout(opt Options) time.Duration {
	base := 10 * time.Minute
	if opt.Lane != conformance.LaneRealtime || opt.Turns <= 0 || opt.FeedMS <= 0 {
		return base
	}
	capture := time.Duration(opt.FeedMS) * time.Duration(opt.Turns) * time.Millisecond
	// Long-form browser runs need enough time for the final browser action,
	// diagnostic collection, lease teardown, and receipt persistence. Five
	// minutes was insufficient at the sixty-minute qualification boundary and
	// caused BAS to return a context deadline before the run could be scored.
	withHeadroom := capture + 10*time.Minute
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
	if strings.TrimSpace(o.UIURL) == "" || strings.TrimSpace(o.Fixture) == "" {
		return errors.New("server-selected ui_url and fixed qualification fixture are required")
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
