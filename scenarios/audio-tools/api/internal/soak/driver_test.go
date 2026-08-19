package soak

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"audio-tools/internal/conformance"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestQualityAcrossTurnsExpandsLoopedRealtimeFixtureReference(t *testing.T) {
	observations := []turnObservation{{
		Diagnostic: &diagnostic{CapturedSamples: 32_000},
		Transcript: "the quick brown fox jumps. the quick brown fox jumps.",
	}}
	assertions := qualityAcrossTurns("the quick brown fox jumps.", observations, conformance.LaneRealtime, 16_000)
	for _, assertion := range assertions {
		require.Equal(t, conformance.OutcomePassed, assertion.Outcome, assertion.Name+": "+assertion.Detail)
	}
}

func TestRealtimeAssertionRejectsLatencyGrowth(t *testing.T) {
	observations := []turnObservation{
		{Diagnostic: &diagnostic{FirstPartialLatencyMS: float64Ptr(100)}},
		{Diagnostic: &diagnostic{FirstPartialLatencyMS: float64Ptr(121)}},
	}
	assertion := realtimeAssertion("first_partial_latency_stable", observations, func(d *diagnostic) *float64 { return d.FirstPartialLatencyMS })
	require.Equal(t, conformance.OutcomeFailed, assertion.Outcome)
}

func TestRealtimeAssertionRejectsCommittedLagGrowth(t *testing.T) {
	observations := []turnObservation{
		{Diagnostic: &diagnostic{CommittedTextLagMS: float64Ptr(10)}},
		{Diagnostic: &diagnostic{CommittedTextLagMS: float64Ptr(16)}},
	}
	assertion := realtimeAssertion("committed_text_lag_stable", observations, func(d *diagnostic) *float64 { return d.CommittedTextLagMS })
	require.Equal(t, conformance.OutcomeFailed, assertion.Outcome)
}

func TestRealtimeAssertionAllowsBoundedCommittedLagJitter(t *testing.T) {
	observations := []turnObservation{
		{Diagnostic: &diagnostic{CommittedTextLagMS: float64Ptr(3)}},
		{Diagnostic: &diagnostic{CommittedTextLagMS: float64Ptr(5)}},
	}
	assertion := realtimeAssertion("committed_text_lag_stable", observations, func(d *diagnostic) *float64 { return d.CommittedTextLagMS })
	require.Equal(t, conformance.OutcomePassed, assertion.Outcome, assertion.Detail)
}

func TestRealtimeAssertionRejectsTimingTelemetryDisappearance(t *testing.T) {
	observations := []turnObservation{
		{Diagnostic: &diagnostic{FirstPartialLatencyMS: float64Ptr(100)}},
		{Diagnostic: &diagnostic{}},
	}
	assertion := realtimeAssertion("first_partial_latency_stable", observations, func(d *diagnostic) *float64 { return d.FirstPartialLatencyMS })
	require.Equal(t, conformance.OutcomeNotMeasured, assertion.Outcome)
	require.Contains(t, assertion.Detail, "stopped exposing")
}

func TestRealtimeTimingObservationsExcludeTerminalFlushFromDrift(t *testing.T) {
	periodicFirst := &diagnostic{CommittedTextLagMS: float64Ptr(2)}
	periodicLast := &diagnostic{CommittedTextLagMS: float64Ptr(2)}
	terminalFlush := &diagnostic{CommittedTextLagMS: float64Ptr(104)}
	observations := []turnObservation{{
		Diagnostic:    terminalFlush,
		TimingSamples: []*diagnostic{periodicFirst, periodicLast},
	}}

	timing := realtimeTimingObservations(observations)
	require.Len(t, timing, 2)
	assertion := realtimeAssertion("committed_text_lag_stable", timing, func(d *diagnostic) *float64 {
		return d.CommittedTextLagMS
	})
	require.Equal(t, conformance.OutcomePassed, assertion.Outcome, assertion.Detail)
}

func TestContinuousInterimTextAssertionRequiresEveryRealtimeCheckpoint(t *testing.T) {
	passed := continuousInterimTextAssertion([]turnObservation{{InterimSampleCount: 3, InterimVisibleSamples: 3}})
	require.Equal(t, conformance.OutcomePassed, passed.Outcome)

	failed := continuousInterimTextAssertion([]turnObservation{{InterimSampleCount: 3, InterimVisibleSamples: 2}})
	require.Equal(t, conformance.OutcomeFailed, failed.Outcome)
	require.Contains(t, failed.Detail, "2/3")

	notMeasured := continuousInterimTextAssertion([]turnObservation{{InterimSampleCount: 0}})
	require.Equal(t, conformance.OutcomeNotMeasured, notMeasured.Outcome)
}

func intervalAccounting(d *diagnostic) conformance.Assertion {
	return invariant("interval_accounting_exactly_once", []turnObservation{{Diagnostic: d, ServerSeen: true}}, 0, 4_000, conformance.LaneRealtime, "kyutai", "stt-1b-en_fr")
}

// Every published soak artifact has carried `sent=-1 processed=-1` against a
// positive captured count, and the verdict called that a coverage failure. It
// is not: -1 is the recorder's "this stage never happened" sentinel, so the
// property was never observable. These tests hold the three-way split.
func TestIntervalAccountingReportsUnobservedCursorsAsNotMeasured(t *testing.T) {
	neverSent := intervalAccounting(&diagnostic{
		CapturedSequence: 6_764, SentSequence: -1, ProcessedSequence: -1,
		State: "failed", TerminalReason: "reconnect_exhausted", ErrorCodes: []string{"stream_closed"},
	})
	require.Equal(t, conformance.OutcomeNotMeasured, neverSent.Outcome)
	require.Contains(t, neverSent.Detail, "none reached the socket")
	require.Contains(t, neverSent.Detail, "terminal=reconnect_exhausted")
	require.Contains(t, neverSent.Detail, "errors=[stream_closed]")

	neverAcked := intervalAccounting(&diagnostic{
		CapturedSequence: 12, SentSequence: 12, ProcessedSequence: -1, State: "completed",
	})
	require.Equal(t, conformance.OutcomeNotMeasured, neverAcked.Outcome)
	require.Contains(t, neverAcked.Detail, "acknowledged none")

	neverCaptured := intervalAccounting(&diagnostic{CapturedSequence: -1, SentSequence: -1, ProcessedSequence: -1})
	require.Equal(t, conformance.OutcomeNotMeasured, neverCaptured.Outcome)
	require.Contains(t, neverCaptured.Detail, "no wire interval was captured")
}

// Sequence 0 is a real interval, so a turn that flushed exactly one wire batch
// has capturedSequence == 0. The old `<= 0` guard reported that healthy turn as
// an accounting violation.
func TestIntervalAccountingAcceptsASingleWireInterval(t *testing.T) {
	single := intervalAccounting(&diagnostic{CapturedSequence: 0, SentSequence: 0, ProcessedSequence: 0, SignalObserved: true, State: "completed"})
	require.Equal(t, conformance.OutcomePassed, single.Outcome, single.Detail)
}

func TestIntervalAccountingStillFailsOnGenuineCoverageLoss(t *testing.T) {
	dropped := intervalAccounting(&diagnostic{
		CapturedSequence: 40, SentSequence: 12, ProcessedSequence: 12,
		State: "completed", TerminalReason: "final",
	})
	require.Equal(t, conformance.OutcomeFailed, dropped.Outcome)
	require.Contains(t, dropped.Detail, "captured=40 sent=12")
	require.Contains(t, dropped.Detail, "terminal=final")

	unacked := intervalAccounting(&diagnostic{CapturedSequence: 40, SentSequence: 40, ProcessedSequence: 9, State: "completed"})
	require.Equal(t, conformance.OutcomeFailed, unacked.Outcome)
}

func float64Ptr(value float64) *float64 { return &value }

func TestWAVSampleCountReadsPCMFixtureCycle(t *testing.T) {
	data := make([]byte, 12+8+16+8+8)
	copy(data[0:4], "RIFF")
	copy(data[8:12], "WAVE")
	offset := 12
	copy(data[offset:offset+4], "fmt ")
	binary.LittleEndian.PutUint32(data[offset+4:offset+8], 16)
	binary.LittleEndian.PutUint16(data[offset+8+2:offset+8+4], 1)
	binary.LittleEndian.PutUint16(data[offset+8+4:offset+8+6], 1)
	binary.LittleEndian.PutUint16(data[offset+8+12:offset+8+14], 2)
	offset += 24
	copy(data[offset:offset+4], "data")
	binary.LittleEndian.PutUint32(data[offset+4:offset+8], 8)
	got, err := wavSampleCount(writeTempFile(t, data))
	require.NoError(t, err)
	require.Equal(t, int64(4), got)
}

func writeTempFile(t *testing.T, data []byte) string {
	t.Helper()
	path := t.TempDir() + "/fixture.wav"
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

func TestRunUsesOneBrowserSessionAndEmitsConformanceAssertions(t *testing.T) {
	fixture := t.TempDir() + "/fixture.wav"
	require.NoError(t, os.WriteFile(fixture, []byte("wav"), 0o600))
	var paths []string
	var feedNodes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/session/start":
			var startRequest struct {
				ExecutionID string `json:"execution_id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&startRequest))
			_, uuidErr := uuid.Parse(startRequest.ExecutionID)
			require.NoError(t, uuidErr, "BAS owner execution_id must be recoverable as a UUID")
			_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "browser-session", "lease_id": "lease-1"})
		case strings.HasSuffix(r.URL.Path, "/close"):
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case strings.HasSuffix(r.URL.Path, "/run"):
			var body struct {
				Instruction struct {
					NodeID string `json:"node_id"`
					Action struct {
						Type string `json:"type"`
					} `json:"action"`
				} `json:"instruction"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if strings.HasPrefix(body.Instruction.NodeID, "feed-") {
				feedNodes = append(feedNodes, body.Instruction.NodeID)
			}
			if body.Instruction.Action.Type == "ACTION_TYPE_EVALUATE" {
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "extracted_data": map[string]any{"result": map[string]any{
					"diag":    map[string]any{"schemaVersion": 1, "sessionId": "browser-session", "state": "completed", "capturedSequence": 1, "capturedSamples": 16000, "signalObserved": true, "sentSequence": 0, "processedSequence": 0, "retainedBytes": 0, "doneSent": true, "terminalReason": "final", "committedTextLagMs": 2},
					"final":   "",
					"interim": "speech in progress",
				}}})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	run, err := Run(context.Background(), Options{DriverURL: server.URL, UIURL: "http://ui.test", Fixture: fixture, Lane: conformance.LaneRealtime, Profile: "realistic", Turns: 1, FeedMS: 1000, EngineID: "local", ModelID: "model-1", Strategy: "product", Policy: "default"}, nil)
	require.NoError(t, err)
	require.Equal(t, conformance.SchemaVersion, run.SchemaVersion)
	require.Equal(t, conformance.LaneRealtime, run.Lane)
	require.Contains(t, paths, "/session/start")
	require.Contains(t, paths, "/session/browser-session/close")
	require.Equal(t, []string{"feed-0"}, feedNodes)
	require.NotEmpty(t, run.Assertions)
	for _, assertion := range run.Assertions {
		require.NotEqual(t, conformance.OutcomeFailed, assertion.Outcome, assertion.Name)
	}
}

func TestStepFailsWhenBASReportsPlaybackFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"audio_playback_failure":"pw-cat exited 1"}`))
	}))
	defer server.Close()

	d := &driver{client: server.Client(), opt: Options{DriverURL: server.URL}}
	err := d.step(context.Background(), "session", 1, "wait-recorder", map[string]any{"wait": map[string]any{"duration_ms": 1}})
	require.ErrorContains(t, err, "browser audio playback failed")
	require.ErrorContains(t, err, "pw-cat exited 1")
}

func TestFeedSplitsLongBrowserWaitsWithoutEndingTheTurn(t *testing.T) {
	require.Equal(t, 1, feedChunkCount(60_000))
	require.Equal(t, 3, feedChunkCount(120_001))

	fixture := t.TempDir() + "/fixture.wav"
	require.NoError(t, os.WriteFile(fixture, []byte("wav"), 0o600))
	var feedNodes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/session/start":
			_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "browser-session", "lease_id": "lease-1"})
		case strings.HasSuffix(r.URL.Path, "/close"):
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case strings.HasSuffix(r.URL.Path, "/run"):
			var body struct {
				Instruction struct {
					NodeID string `json:"node_id"`
					Action struct {
						Type string `json:"type"`
					} `json:"action"`
				} `json:"instruction"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if strings.HasPrefix(body.Instruction.NodeID, "feed-") {
				feedNodes = append(feedNodes, body.Instruction.NodeID)
			}
			if body.Instruction.Action.Type == "ACTION_TYPE_EVALUATE" {
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "extracted_data": map[string]any{"result": map[string]any{
					"diag":    map[string]any{"schemaVersion": 1, "sessionId": "browser-session", "state": "completed", "capturedSamples": 16000, "signalObserved": true, "committedTextLagMs": 2},
					"final":   "",
					"interim": "speech in progress",
				}}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err := Run(context.Background(), Options{DriverURL: server.URL, UIURL: "http://ui.test", Fixture: fixture, Lane: conformance.LaneRealtime, Profile: "continuous", Turns: 1, FeedMS: 120_001, EngineID: "local", ModelID: "model-1", Strategy: "product", Policy: "default"}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"feed-0", "feed-1", "feed-2"}, feedNodes)
}

func TestRealtimeShapeUsesProfileWhenAcceleratedShapeIsPresent(t *testing.T) {
	// The CLI supplies shape=burst by default. That value belongs to the
	// accelerated virtual corpus and must not erase the realtime profile.
	o := Options{Lane: conformance.LaneRealtime, Profile: "continuous", Shape: "burst"}
	require.Equal(t, "continuous", realtimeShape(o))
	require.Zero(t, realtimePlaybackPauseMS(o))
	require.Equal(t, "continuous", evidenceShape(o))

	o = Options{Lane: conformance.LaneRealtime, Profile: "realistic", Shape: "burst"}
	require.Equal(t, "realistic", realtimeShape(o))
	require.Equal(t, 250, realtimePlaybackPauseMS(o))
	require.Equal(t, "realistic", evidenceShape(o))
}

func TestExplicitRealtimeShapeOverridesProfile(t *testing.T) {
	o := Options{Lane: conformance.LaneRealtime, Profile: "realistic", Shape: "continuous"}
	require.Equal(t, "continuous", realtimeShape(o))
	require.Zero(t, realtimePlaybackPauseMS(o))
	require.Equal(t, "continuous", evidenceShape(o))
}

func TestSwarmManagerSurfaceUsesSharedComposerLifecycle(t *testing.T) {
	d := &driver{opt: Options{Surface: "swarm-manager", UIURL: "http://swarm.test"}}
	require.Equal(t, "swarm-manager", normalizeSurface(d.opt.Surface))
	require.Equal(t, "http://swarm.test/plan", d.pageURL())
	require.Equal(t, `[data-testid="captures-quick-input-mic"]`, d.productSelector("ready"))
	require.Equal(t, `[data-testid="captures-quick-input-mic"][data-state="recording"]`, d.productSelector("recording"))
	require.Empty(t, d.productSelector("processed"))
	require.Contains(t, d.evaluationExpression(), "captures-quick-composer-interim")
}

func TestValidateOptionsRejectsUnknownSurface(t *testing.T) {
	err := validateOptions(Options{DriverURL: "driver", UIURL: "ui", Fixture: "fixture", Surface: "unknown", Lane: conformance.LaneRealtime, EngineID: "local", ModelID: "model"})
	require.ErrorContains(t, err, "surface must be audio-tools or swarm-manager")
}

func TestValidateOptionsRequiresExactProviderCell(t *testing.T) {
	err := validateOptions(Options{DriverURL: "driver", UIURL: "ui", Fixture: "fixture", Lane: conformance.LaneAccelerated})
	require.ErrorContains(t, err, "engine_id and model_id")
}

func TestValidateOptionsRequiresIndependentReferenceForAcceleratedLane(t *testing.T) {
	err := validateOptions(Options{DriverURL: "driver", UIURL: "ui", Fixture: "fixture", Lane: conformance.LaneAccelerated, EngineID: "local", ModelID: "model"})
	require.ErrorContains(t, err, "independent reference_text")
	require.NoError(t, validateOptions(Options{DriverURL: "driver", UIURL: "ui", Fixture: "fixture", Lane: conformance.LaneAccelerated, Reference: "the quick brown fox jumps.", SimulatedMinutes: 60, EngineID: "local", ModelID: "model"}))
}

func TestRunTimeoutIncludesRealtimeCaptureAndHeadroom(t *testing.T) {
	got := RunTimeout(Options{Lane: conformance.LaneRealtime, Turns: 1, FeedMS: 6 * 60 * 1000})
	require.Equal(t, 11*time.Minute, got)
	require.Equal(t, 10*time.Minute, RunTimeout(Options{Lane: conformance.LaneAccelerated, Turns: 3, FeedMS: 60 * 1000}))
}

func TestPersistEvidencePublishesOneAtomicRunDocument(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	run := conformance.Run{
		SchemaVersion: conformance.SchemaVersion,
		RunID:         "browser-soak-test",
		Lane:          conformance.LaneRealtime,
		Cell:          conformance.Cell{EngineID: "engine", ModelID: "model", Strategy: "product", Policy: "default"},
		Code:          conformance.Code{CapturePackage: "sha256:capture", Server: "sha256:server"},
	}

	path, err := PersistEvidence(run)
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var got conformance.Run
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, run.RunID, got.RunID)
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, strings.HasPrefix(entry.Name(), ".browser-soak-test.json.tmp-"), entry.Name())
	}
}

func TestPersistEvidenceUsesPortableStorageOutsideCheckout(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	run := conformance.Run{
		SchemaVersion: conformance.SchemaVersion,
		RunID:         "portable-soak-test",
		Lane:          conformance.LaneRealtime,
		Cell:          conformance.Cell{EngineID: "engine", ModelID: "model", Strategy: "product", Policy: "default"},
		Code:          conformance.Code{CapturePackage: "sha256:capture", Server: "sha256:server"},
	}

	path, err := PersistEvidence(run)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(storageRoot, "data", "vrooli", "audio-tools", "coverage", run.RunID+".json"), path)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var got conformance.Run
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, run.RunID, got.RunID)
}
