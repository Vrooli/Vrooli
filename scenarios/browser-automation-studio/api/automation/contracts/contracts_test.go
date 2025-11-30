package contracts

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// EngineCapabilities.Validate Tests
// =============================================================================

func TestEngineCapabilitiesValidate(t *testing.T) {
	valid := EngineCapabilities{
		SchemaVersion:         CapabilitiesSchemaVersion,
		Engine:                "browserless",
		MaxConcurrentSessions: 2,
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid capabilities, got error: %v", err)
	}

	invalid := valid
	invalid.Engine = ""
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error when engine is missing")
	}

	invalid = valid
	invalid.MaxConcurrentSessions = 0
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error when max_concurrent_sessions <= 0")
	}

	invalid = valid
	invalid.MaxViewportWidth = -1
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error when max_viewport_width is negative")
	}
}

// Additional EngineCapabilities.Validate tests

func TestEngineCapabilitiesValidate_SchemaVersion(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] rejects empty schema version", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         "",
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
		}
		err := caps.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "schema_version")
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] rejects wrong schema version", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         "wrong-version",
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
		}
		err := caps.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), CapabilitiesSchemaVersion)
	})

	t.Run("accepts current schema version", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
		}
		err := caps.Validate()
		assert.NoError(t, err)
	})
}

func TestEngineCapabilitiesValidate_MaxConcurrentSessions(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] rejects negative max_concurrent_sessions", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: -5,
		}
		err := caps.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_concurrent_sessions")
	})

	t.Run("accepts high max_concurrent_sessions", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1000,
		}
		err := caps.Validate()
		assert.NoError(t, err)
	})
}

func TestEngineCapabilitiesValidate_ViewportDimensions(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] allows zero max_viewport_width", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
			MaxViewportWidth:      0, // Zero means unknown/unbounded
		}
		err := caps.Validate()
		assert.NoError(t, err)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] allows zero max_viewport_height", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
			MaxViewportHeight:     0,
		}
		err := caps.Validate()
		assert.NoError(t, err)
	})

	t.Run("rejects negative max_viewport_height", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
			MaxViewportHeight:     -1,
		}
		err := caps.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_viewport_height")
	})

	t.Run("accepts large viewport dimensions", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
			MaxViewportWidth:      4096,
			MaxViewportHeight:     2160,
		}
		err := caps.Validate()
		assert.NoError(t, err)
	})
}

// =============================================================================
// EngineCapabilities.CheckCompatibility Tests
// =============================================================================

func TestEngineCapabilitiesCheckCompatibility(t *testing.T) {
	caps := EngineCapabilities{
		SchemaVersion:         CapabilitiesSchemaVersion,
		Engine:                "browserless",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    false,
		SupportsHAR:           false,
		SupportsVideo:         true,
		SupportsIframes:       true,
		MaxViewportWidth:      1200,
		MaxViewportHeight:     800,
	}

	req := CapabilityRequirement{
		NeedsParallelTabs: true,
		NeedsHAR:          true,
		NeedsVideo:        true,
		MinViewportWidth:  1300,
		MinViewportHeight: 700,
	}

	gap := caps.CheckCompatibility(req)
	if gap.Satisfied() {
		t.Fatalf("expected gaps, got satisfied result")
	}

	expectedMissing := []string{"parallel_tabs", "har", "viewport_width>=1300"}
	for _, missing := range expectedMissing {
		if !contains(gap.Missing, missing) {
			t.Fatalf("expected missing capability %q, got %+v", missing, gap.Missing)
		}
	}

	// Height should be satisfied; video supported; warnings should be empty.
	if len(gap.Warnings) > 0 {
		t.Fatalf("expected no warnings, got %+v", gap.Warnings)
	}
}

// Additional CheckCompatibility tests

func TestEngineCapabilitiesCheckCompatibility_AllFeatures(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] reports missing file_uploads", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
			SupportsFileUploads:   false,
		}
		req := CapabilityRequirement{NeedsFileUploads: true}

		gap := caps.CheckCompatibility(req)
		assert.False(t, gap.Satisfied())
		assert.Contains(t, gap.Missing, "file_uploads")
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] reports missing downloads", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
			SupportsDownloads:     false,
		}
		req := CapabilityRequirement{NeedsDownloads: true}

		gap := caps.CheckCompatibility(req)
		assert.False(t, gap.Satisfied())
		assert.Contains(t, gap.Missing, "downloads")
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] reports missing tracing", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
			SupportsTracing:       false,
		}
		req := CapabilityRequirement{NeedsTracing: true}

		gap := caps.CheckCompatibility(req)
		assert.False(t, gap.Satisfied())
		assert.Contains(t, gap.Missing, "tracing")
	})

	t.Run("reports missing iframes", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "browserless",
			MaxConcurrentSessions: 1,
			SupportsIframes:       false,
		}
		req := CapabilityRequirement{NeedsIframes: true}

		gap := caps.CheckCompatibility(req)
		assert.False(t, gap.Satisfied())
		assert.Contains(t, gap.Missing, "iframes")
	})
}

func TestEngineCapabilitiesCheckCompatibility_FullCapabilities(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] fully capable engine satisfies all requirements", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "playwright",
			MaxConcurrentSessions: 10,
			AllowsParallelTabs:    true,
			SupportsHAR:           true,
			SupportsVideo:         true,
			SupportsIframes:       true,
			SupportsFileUploads:   true,
			SupportsDownloads:     true,
			SupportsTracing:       true,
			MaxViewportWidth:      4096,
			MaxViewportHeight:     2160,
		}

		req := CapabilityRequirement{
			NeedsParallelTabs: true,
			NeedsHAR:          true,
			NeedsVideo:        true,
			NeedsIframes:      true,
			NeedsFileUploads:  true,
			NeedsDownloads:    true,
			NeedsTracing:      true,
			MinViewportWidth:  1920,
			MinViewportHeight: 1080,
		}

		gap := caps.CheckCompatibility(req)
		assert.True(t, gap.Satisfied())
		assert.Empty(t, gap.Missing)
		assert.Empty(t, gap.Warnings)
	})
}

func TestEngineCapabilitiesCheckCompatibility_NoRequirements(t *testing.T) {
	t.Run("empty requirements are always satisfied", func(t *testing.T) {
		caps := EngineCapabilities{
			SchemaVersion:         CapabilitiesSchemaVersion,
			Engine:                "minimal",
			MaxConcurrentSessions: 1,
		}
		req := CapabilityRequirement{}

		gap := caps.CheckCompatibility(req)
		assert.True(t, gap.Satisfied())
		assert.Empty(t, gap.Missing)
		assert.Empty(t, gap.Warnings)
	})
}

func TestEngineCapabilitiesViewportWarningsWhenUnknownBounds(t *testing.T) {
	caps := EngineCapabilities{
		SchemaVersion:         CapabilitiesSchemaVersion,
		Engine:                "browserless",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    true,
		SupportsHAR:           true,
		SupportsVideo:         true,
		SupportsIframes:       true,
		SupportsFileUploads:   true,
		SupportsDownloads:     true,
		SupportsTracing:       true,
		// MaxViewportWidth/Height left at zero to simulate unknown upper bound.
	}

	req := CapabilityRequirement{
		MinViewportWidth:  1920,
		MinViewportHeight: 1080,
	}

	gap := caps.CheckCompatibility(req)
	if len(gap.Missing) != 0 {
		t.Fatalf("expected no hard missing capabilities, got %+v", gap.Missing)
	}
	if len(gap.Warnings) == 0 {
		t.Fatalf("expected viewport warnings when bounds are unknown")
	}
	assertWarningContains := func(expected string) {
		found := false
		for _, warn := range gap.Warnings {
			if strings.Contains(warn, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected warning containing %q, got %+v", expected, gap.Warnings)
		}
	}
	assertWarningContains("viewport_width>=1920")
	assertWarningContains("viewport_height>=1080")
}

// =============================================================================
// CapabilityGap Tests
// =============================================================================

func TestCapabilityGap_Satisfied(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] empty gap is satisfied", func(t *testing.T) {
		gap := CapabilityGap{}
		assert.True(t, gap.Satisfied())
	})

	t.Run("gap with warnings is still satisfied", func(t *testing.T) {
		gap := CapabilityGap{
			Warnings: []string{"viewport unknown"},
		}
		assert.True(t, gap.Satisfied())
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] gap with missing is not satisfied", func(t *testing.T) {
		gap := CapabilityGap{
			Missing: []string{"parallel_tabs"},
		}
		assert.False(t, gap.Satisfied())
	})

	t.Run("gap with both missing and warnings is not satisfied", func(t *testing.T) {
		gap := CapabilityGap{
			Missing:  []string{"har"},
			Warnings: []string{"viewport unknown"},
		}
		assert.False(t, gap.Satisfied())
	})
}

// =============================================================================
// EventBufferLimits Tests
// =============================================================================

func TestEventBufferLimitsValidate(t *testing.T) {
	if err := DefaultEventBufferLimits.Validate(); err != nil {
		t.Fatalf("expected default limits to be valid, got %v", err)
	}

	limits := EventBufferLimits{PerExecution: 0, PerAttempt: 10}
	if err := limits.Validate(); err == nil {
		t.Fatal("expected per-execution validation error")
	}

	limits = EventBufferLimits{PerExecution: 10, PerAttempt: -1}
	if err := limits.Validate(); err == nil {
		t.Fatal("expected per-attempt validation error")
	}
}

// Additional EventBufferLimits tests

func TestEventBufferLimitsValidate_EdgeCases(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] rejects zero per-attempt", func(t *testing.T) {
		limits := EventBufferLimits{PerExecution: 100, PerAttempt: 0}
		err := limits.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "per-attempt")
	})

	t.Run("accepts minimal valid limits", func(t *testing.T) {
		limits := EventBufferLimits{PerExecution: 1, PerAttempt: 1}
		err := limits.Validate()
		assert.NoError(t, err)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] accepts large limits", func(t *testing.T) {
		limits := EventBufferLimits{PerExecution: 10000, PerAttempt: 1000}
		err := limits.Validate()
		assert.NoError(t, err)
	})
}

func TestDefaultEventBufferLimits(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] default per-execution is positive", func(t *testing.T) {
		assert.Greater(t, DefaultEventBufferLimits.PerExecution, 0)
	})

	t.Run("default per-attempt is positive", func(t *testing.T) {
		assert.Greater(t, DefaultEventBufferLimits.PerAttempt, 0)
	})

	t.Run("default validates successfully", func(t *testing.T) {
		err := DefaultEventBufferLimits.Validate()
		assert.NoError(t, err)
	})
}

// =============================================================================
// StepOutcome Tests
// =============================================================================

func TestStepOutcome_Structure(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can construct minimal success outcome", func(t *testing.T) {
		outcome := StepOutcome{
			SchemaVersion:  StepOutcomeSchemaVersion,
			PayloadVersion: PayloadVersion,
			ExecutionID:    uuid.New(),
			StepIndex:      0,
			Attempt:        1,
			NodeID:         "navigate.1",
			StepType:       "navigate",
			Success:        true,
			StartedAt:      time.Now(),
		}

		assert.True(t, outcome.Success)
		assert.Equal(t, StepOutcomeSchemaVersion, outcome.SchemaVersion)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can construct failure outcome", func(t *testing.T) {
		now := time.Now()
		outcome := StepOutcome{
			SchemaVersion:  StepOutcomeSchemaVersion,
			PayloadVersion: PayloadVersion,
			ExecutionID:    uuid.New(),
			StepIndex:      0,
			Attempt:        1,
			NodeID:         "navigate.1",
			StepType:       "navigate",
			Success:        false,
			StartedAt:      now,
			Failure: &StepFailure{
				Kind:       FailureKindTimeout,
				Code:       "NAV_TIMEOUT",
				Message:    "Navigation timed out after 30s",
				Fatal:      false,
				Retryable:  true,
				OccurredAt: &now,
			},
		}

		assert.False(t, outcome.Success)
		require.NotNil(t, outcome.Failure)
		assert.Equal(t, FailureKindTimeout, outcome.Failure.Kind)
		assert.True(t, outcome.Failure.Retryable)
	})

	t.Run("can include screenshot", func(t *testing.T) {
		outcome := StepOutcome{
			SchemaVersion:  StepOutcomeSchemaVersion,
			PayloadVersion: PayloadVersion,
			StepIndex:      0,
			Attempt:        1,
			NodeID:         "screenshot.1",
			StepType:       "screenshot",
			Success:        true,
			StartedAt:      time.Now(),
			Screenshot: &Screenshot{
				MediaType:   "image/png",
				Width:       1280,
				Height:      720,
				CaptureTime: time.Now(),
			},
		}

		require.NotNil(t, outcome.Screenshot)
		assert.Equal(t, "image/png", outcome.Screenshot.MediaType)
	})
}

// =============================================================================
// StepFailure Tests
// =============================================================================

func TestStepFailure_Kinds(t *testing.T) {
	kinds := []FailureKind{
		FailureKindNone,
		FailureKindEngine,
		FailureKindInfra,
		FailureKindOrchestration,
		FailureKindUser,
		FailureKindTimeout,
		FailureKindCancelled,
	}

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] all failure kinds are distinct", func(t *testing.T) {
		seen := make(map[FailureKind]bool)
		for _, kind := range kinds {
			assert.False(t, seen[kind], "duplicate failure kind: %s", kind)
			seen[kind] = true
		}
	})
}

func TestStepFailure_Sources(t *testing.T) {
	sources := []FailureSource{
		FailureSourceEngine,
		FailureSourceExecutor,
		FailureSourceRecorder,
	}

	t.Run("all failure sources are distinct", func(t *testing.T) {
		seen := make(map[FailureSource]bool)
		for _, source := range sources {
			assert.False(t, seen[source], "duplicate failure source: %s", source)
			seen[source] = true
		}
	})
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestSchemaVersionConstants(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] StepOutcomeSchemaVersion is non-empty", func(t *testing.T) {
		assert.NotEmpty(t, StepOutcomeSchemaVersion)
	})

	t.Run("TelemetrySchemaVersion is non-empty", func(t *testing.T) {
		assert.NotEmpty(t, TelemetrySchemaVersion)
	})

	t.Run("EventEnvelopeSchemaVersion is non-empty", func(t *testing.T) {
		assert.NotEmpty(t, EventEnvelopeSchemaVersion)
	})

	t.Run("CapabilitiesSchemaVersion is non-empty", func(t *testing.T) {
		assert.NotEmpty(t, CapabilitiesSchemaVersion)
	})

	t.Run("PayloadVersion is non-empty", func(t *testing.T) {
		assert.NotEmpty(t, PayloadVersion)
	})
}

func TestSizeLimitConstants(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] DOMSnapshotMaxBytes is positive", func(t *testing.T) {
		assert.Greater(t, DOMSnapshotMaxBytes, 0)
	})

	t.Run("ScreenshotMaxBytes is positive", func(t *testing.T) {
		assert.Greater(t, ScreenshotMaxBytes, 0)
	})

	t.Run("DefaultScreenshotWidth is positive", func(t *testing.T) {
		assert.Greater(t, DefaultScreenshotWidth, 0)
	})

	t.Run("DefaultScreenshotHeight is positive", func(t *testing.T) {
		assert.Greater(t, DefaultScreenshotHeight, 0)
	})

	t.Run("ConsoleEntryMaxBytes is positive", func(t *testing.T) {
		assert.Greater(t, ConsoleEntryMaxBytes, 0)
	})

	t.Run("NetworkPayloadPreviewMaxBytes is positive", func(t *testing.T) {
		assert.Greater(t, NetworkPayloadPreviewMaxBytes, 0)
	})
}

// =============================================================================
// BoundingBox Tests
// =============================================================================

func TestBoundingBox(t *testing.T) {
	t.Run("can represent element position", func(t *testing.T) {
		box := BoundingBox{
			X:      100,
			Y:      200,
			Width:  300,
			Height: 150,
		}
		assert.Equal(t, float64(100), box.X)
		assert.Equal(t, float64(200), box.Y)
		assert.Equal(t, float64(300), box.Width)
		assert.Equal(t, float64(150), box.Height)
	})

	t.Run("can represent fractional positions", func(t *testing.T) {
		box := BoundingBox{
			X:      100.5,
			Y:      200.75,
			Width:  300.25,
			Height: 150.125,
		}
		assert.Equal(t, 100.5, box.X)
		assert.Equal(t, 200.75, box.Y)
	})
}

// =============================================================================
// EventEnvelope Tests
// =============================================================================

func TestEventEnvelope_Structure(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can construct complete envelope", func(t *testing.T) {
		stepIndex := 0
		attempt := 1
		env := EventEnvelope{
			SchemaVersion:  EventEnvelopeSchemaVersion,
			PayloadVersion: PayloadVersion,
			Kind:           EventKindStepCompleted,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			StepIndex:      &stepIndex,
			Attempt:        &attempt,
			Sequence:       1,
			Timestamp:      time.Now().UTC(),
			Payload:        map[string]any{"result": "success"},
		}

		assert.Equal(t, EventEnvelopeSchemaVersion, env.SchemaVersion)
		assert.Equal(t, EventKindStepCompleted, env.Kind)
		assert.NotNil(t, env.StepIndex)
		assert.Equal(t, 0, *env.StepIndex)
	})

	t.Run("supports nil step_index for execution-level events", func(t *testing.T) {
		env := EventEnvelope{
			SchemaVersion:  EventEnvelopeSchemaVersion,
			PayloadVersion: PayloadVersion,
			Kind:           EventKindExecutionStarted,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			StepIndex:      nil,
			Timestamp:      time.Now().UTC(),
		}

		assert.Nil(t, env.StepIndex)
		assert.Equal(t, EventKindExecutionStarted, env.Kind)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can include drop counters", func(t *testing.T) {
		env := EventEnvelope{
			SchemaVersion:  EventEnvelopeSchemaVersion,
			PayloadVersion: PayloadVersion,
			Kind:           EventKindStepTelemetry,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Timestamp:      time.Now().UTC(),
			Drops: DropCounters{
				Dropped:       5,
				OldestDropped: 10,
			},
		}

		assert.Equal(t, uint64(5), env.Drops.Dropped)
		assert.Equal(t, uint64(10), env.Drops.OldestDropped)
	})
}

// =============================================================================
// EventKind Tests
// =============================================================================

func TestEventKind_Enums(t *testing.T) {
	executionKinds := []EventKind{
		EventKindExecutionStarted,
		EventKindExecutionProgress,
		EventKindExecutionCompleted,
		EventKindExecutionFailed,
		EventKindExecutionCancelled,
	}

	stepKinds := []EventKind{
		EventKindStepStarted,
		EventKindStepCompleted,
		EventKindStepFailed,
		EventKindStepScreenshot,
		EventKindStepTelemetry,
		EventKindStepHeartbeat,
	}

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] all execution event kinds are distinct", func(t *testing.T) {
		seen := make(map[EventKind]bool)
		for _, kind := range executionKinds {
			assert.False(t, seen[kind], "duplicate execution event kind: %s", kind)
			seen[kind] = true
		}
	})

	t.Run("all step event kinds are distinct", func(t *testing.T) {
		seen := make(map[EventKind]bool)
		for _, kind := range stepKinds {
			assert.False(t, seen[kind], "duplicate step event kind: %s", kind)
			seen[kind] = true
		}
	})

	t.Run("execution and step kinds don't overlap", func(t *testing.T) {
		executionSet := make(map[EventKind]bool)
		for _, kind := range executionKinds {
			executionSet[kind] = true
		}
		for _, kind := range stepKinds {
			assert.False(t, executionSet[kind], "step kind %s overlaps with execution kinds", kind)
		}
	})
}

// =============================================================================
// TelemetryKind Tests
// =============================================================================

func TestTelemetryKind_Enums(t *testing.T) {
	kinds := []TelemetryKind{
		TelemetryKindHeartbeat,
		TelemetryKindConsole,
		TelemetryKindNetwork,
		TelemetryKindRetry,
		TelemetryKindProgress,
	}

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] all telemetry kinds are distinct", func(t *testing.T) {
		seen := make(map[TelemetryKind]bool)
		for _, kind := range kinds {
			assert.False(t, seen[kind], "duplicate telemetry kind: %s", kind)
			seen[kind] = true
		}
	})

	t.Run("all telemetry kinds are non-empty", func(t *testing.T) {
		for _, kind := range kinds {
			assert.NotEmpty(t, string(kind))
		}
	})
}

// =============================================================================
// StepTelemetry Tests
// =============================================================================

func TestStepTelemetry_Structure(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can construct heartbeat telemetry", func(t *testing.T) {
		tel := StepTelemetry{
			SchemaVersion:  TelemetrySchemaVersion,
			PayloadVersion: PayloadVersion,
			ExecutionID:    uuid.New(),
			StepIndex:      0,
			Attempt:        1,
			Kind:           TelemetryKindHeartbeat,
			Timestamp:      time.Now().UTC(),
			ElapsedMs:      500,
			Heartbeat: &HeartbeatTelemetry{
				Progress: 50,
				Message:  "Waiting for element...",
			},
		}

		assert.Equal(t, TelemetrySchemaVersion, tel.SchemaVersion)
		assert.Equal(t, TelemetryKindHeartbeat, tel.Kind)
		require.NotNil(t, tel.Heartbeat)
		assert.Equal(t, 50, tel.Heartbeat.Progress)
	})

	t.Run("can construct retry telemetry", func(t *testing.T) {
		tel := StepTelemetry{
			SchemaVersion:  TelemetrySchemaVersion,
			PayloadVersion: PayloadVersion,
			ExecutionID:    uuid.New(),
			StepIndex:      0,
			Attempt:        2,
			Kind:           TelemetryKindRetry,
			Timestamp:      time.Now().UTC(),
			Retry: &RetryTelemetry{
				Attempt:     2,
				MaxAttempts: 3,
				Reason: &StepFailure{
					Kind:    FailureKindTimeout,
					Message: "Element not found",
				},
			},
		}

		require.NotNil(t, tel.Retry)
		assert.Equal(t, 2, tel.Retry.Attempt)
		assert.Equal(t, 3, tel.Retry.MaxAttempts)
	})
}

// =============================================================================
// ExecutionPlan Tests
// =============================================================================

func TestExecutionPlan_Structure(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can construct minimal plan", func(t *testing.T) {
		plan := ExecutionPlan{
			SchemaVersion:  ExecutionPlanSchemaVersion,
			PayloadVersion: PayloadVersion,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Instructions: []CompiledInstruction{
				{Index: 0, NodeID: "nav-1", Type: "navigate", Params: map[string]any{"url": "https://example.com"}},
			},
			CreatedAt: time.Now().UTC(),
		}

		assert.Equal(t, ExecutionPlanSchemaVersion, plan.SchemaVersion)
		assert.Len(t, plan.Instructions, 1)
		assert.Equal(t, "navigate", plan.Instructions[0].Type)
	})

	t.Run("can include graph with branching", func(t *testing.T) {
		plan := ExecutionPlan{
			SchemaVersion:  ExecutionPlanSchemaVersion,
			PayloadVersion: PayloadVersion,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Instructions:   []CompiledInstruction{},
			Graph: &PlanGraph{
				Steps: []PlanStep{
					{
						Index:  0,
						NodeID: "branch-1",
						Type:   "branch",
						Outgoing: []PlanEdge{
							{ID: "e1", Target: "step-2", Condition: "true"},
							{ID: "e2", Target: "step-3", Condition: "false"},
						},
					},
				},
			},
			CreatedAt: time.Now().UTC(),
		}

		require.NotNil(t, plan.Graph)
		assert.Len(t, plan.Graph.Steps, 1)
		assert.Len(t, plan.Graph.Steps[0].Outgoing, 2)
	})

	t.Run("can include metadata", func(t *testing.T) {
		plan := ExecutionPlan{
			SchemaVersion:  ExecutionPlanSchemaVersion,
			PayloadVersion: PayloadVersion,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Instructions:   []CompiledInstruction{},
			Metadata: map[string]any{
				"entrySelector":          "[data-testid=ready]",
				"entrySelectorTimeoutMs": 5000,
			},
			CreatedAt: time.Now().UTC(),
		}

		assert.Equal(t, "[data-testid=ready]", plan.Metadata["entrySelector"])
	})
}

// =============================================================================
// CompiledInstruction Tests
// =============================================================================

func TestCompiledInstruction_Structure(t *testing.T) {
	t.Run("can represent navigate instruction", func(t *testing.T) {
		instr := CompiledInstruction{
			Index:  0,
			NodeID: "nav-1",
			Type:   "navigate",
			Params: map[string]any{
				"url":     "https://example.com",
				"timeout": 30000,
			},
		}

		assert.Equal(t, 0, instr.Index)
		assert.Equal(t, "navigate", instr.Type)
		assert.Equal(t, "https://example.com", instr.Params["url"])
	})

	t.Run("can include preload HTML", func(t *testing.T) {
		instr := CompiledInstruction{
			Index:       0,
			NodeID:      "click-1",
			Type:        "click",
			Params:      map[string]any{"selector": "#button"},
			PreloadHTML: "<button id=\"button\">Click</button>",
		}

		assert.NotEmpty(t, instr.PreloadHTML)
	})

	t.Run("can include context and metadata", func(t *testing.T) {
		instr := CompiledInstruction{
			Index:    0,
			NodeID:   "type-1",
			Type:     "type",
			Params:   map[string]any{"selector": "#input", "text": "hello"},
			Context:  map[string]any{"previousStep": "nav-1"},
			Metadata: map[string]string{"source": "recording"},
		}

		assert.Equal(t, "nav-1", instr.Context["previousStep"])
		assert.Equal(t, "recording", instr.Metadata["source"])
	})
}

// =============================================================================
// DOMSnapshot Tests
// =============================================================================

func TestDOMSnapshot_Structure(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can represent complete DOM snapshot", func(t *testing.T) {
		now := time.Now().UTC()
		snap := DOMSnapshot{
			HTML:        "<html><body><h1>Test</h1></body></html>",
			Preview:     "<html><body><h1>...",
			Hash:        "sha256:abc123",
			CollectedAt: now,
			Truncated:   false,
		}

		assert.Contains(t, snap.HTML, "<html>")
		assert.NotEmpty(t, snap.Hash)
		assert.Equal(t, now, snap.CollectedAt)
	})

	t.Run("can represent truncated snapshot", func(t *testing.T) {
		snap := DOMSnapshot{
			HTML:        strings.Repeat("x", 1000),
			Hash:        "sha256:truncated",
			CollectedAt: time.Now().UTC(),
			Truncated:   true,
		}

		assert.True(t, snap.Truncated)
	})
}

// =============================================================================
// ConsoleLogEntry Tests
// =============================================================================

func TestConsoleLogEntry_Structure(t *testing.T) {
	types := []string{"log", "warn", "error", "info", "debug"}

	for _, logType := range types {
		t.Run("supports "+logType+" type", func(t *testing.T) {
			entry := ConsoleLogEntry{
				Type:      logType,
				Text:      "Test message",
				Timestamp: time.Now().UTC(),
			}

			assert.Equal(t, logType, entry.Type)
		})
	}

	t.Run("can include stack trace", func(t *testing.T) {
		entry := ConsoleLogEntry{
			Type:      "error",
			Text:      "Uncaught TypeError",
			Timestamp: time.Now().UTC(),
			Stack:     "at foo.js:10:5\nat bar.js:20:3",
			Location:  "foo.js:10:5",
		}

		assert.NotEmpty(t, entry.Stack)
		assert.NotEmpty(t, entry.Location)
	})
}

// =============================================================================
// NetworkEvent Tests
// =============================================================================

func TestNetworkEvent_Structure(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can represent request event", func(t *testing.T) {
		event := NetworkEvent{
			Type:           "request",
			URL:            "https://api.example.com/users",
			Method:         "POST",
			ResourceType:   "xhr",
			Timestamp:      time.Now().UTC(),
			RequestHeaders: map[string]string{"Content-Type": "application/json"},
		}

		assert.Equal(t, "request", event.Type)
		assert.Equal(t, "POST", event.Method)
	})

	t.Run("can represent response event", func(t *testing.T) {
		event := NetworkEvent{
			Type:                "response",
			URL:                 "https://api.example.com/users",
			Method:              "POST",
			Status:              201,
			OK:                  true,
			Timestamp:           time.Now().UTC(),
			ResponseHeaders:     map[string]string{"Content-Type": "application/json"},
			ResponseBodyPreview: `{"id": 123}`,
		}

		assert.Equal(t, "response", event.Type)
		assert.Equal(t, 201, event.Status)
		assert.True(t, event.OK)
	})

	t.Run("can represent failure event", func(t *testing.T) {
		event := NetworkEvent{
			Type:      "failure",
			URL:       "https://api.example.com/users",
			Failure:   "net::ERR_CONNECTION_REFUSED",
			Timestamp: time.Now().UTC(),
		}

		assert.Equal(t, "failure", event.Type)
		assert.NotEmpty(t, event.Failure)
	})

	t.Run("supports truncated payloads", func(t *testing.T) {
		event := NetworkEvent{
			Type:                "response",
			URL:                 "https://api.example.com/large",
			Status:              200,
			OK:                  true,
			Timestamp:           time.Now().UTC(),
			ResponseBodyPreview: strings.Repeat("x", 1000),
			Truncated:           true,
		}

		assert.True(t, event.Truncated)
	})
}

// =============================================================================
// AssertionOutcome Tests
// =============================================================================

func TestAssertionOutcome_Structure(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] can represent successful assertion", func(t *testing.T) {
		outcome := AssertionOutcome{
			Mode:     "equals",
			Selector: "#heading",
			Expected: "Welcome",
			Actual:   "Welcome",
			Success:  true,
			Message:  "Text matches expected value",
		}

		assert.True(t, outcome.Success)
		assert.Equal(t, outcome.Expected, outcome.Actual)
	})

	t.Run("can represent failed assertion", func(t *testing.T) {
		outcome := AssertionOutcome{
			Mode:     "contains",
			Selector: "#content",
			Expected: "error",
			Actual:   "success message",
			Success:  false,
			Negated:  false,
			Message:  "Expected content to contain 'error'",
		}

		assert.False(t, outcome.Success)
		assert.NotEqual(t, outcome.Expected, outcome.Actual)
	})

	t.Run("supports negated assertions", func(t *testing.T) {
		outcome := AssertionOutcome{
			Mode:     "visible",
			Selector: "#error",
			Success:  true,
			Negated:  true,
			Message:  "Element is not visible (as expected)",
		}

		assert.True(t, outcome.Negated)
	})
}

// =============================================================================
// ConditionOutcome Tests
// =============================================================================

func TestConditionOutcome_Structure(t *testing.T) {
	t.Run("can represent element existence check", func(t *testing.T) {
		outcome := ConditionOutcome{
			Type:     "element_exists",
			Selector: "#login-form",
			Outcome:  true,
		}

		assert.True(t, outcome.Outcome)
	})

	t.Run("can represent variable comparison", func(t *testing.T) {
		outcome := ConditionOutcome{
			Type:     "variable",
			Operator: "equals",
			Variable: "userCount",
			Expected: 10,
			Actual:   10,
			Outcome:  true,
		}

		assert.Equal(t, "equals", outcome.Operator)
	})

	t.Run("can represent expression evaluation", func(t *testing.T) {
		outcome := ConditionOutcome{
			Type:       "expression",
			Expression: "data.items.length > 0",
			Outcome:    true,
		}

		assert.NotEmpty(t, outcome.Expression)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func contains(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}
