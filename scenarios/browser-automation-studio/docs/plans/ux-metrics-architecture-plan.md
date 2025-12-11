# UX Metrics Foundation Architecture Proposal

> **Status**: Proposed
> **Author**: Claude (Architecture Design Session)
> **Date**: 2025-12-11
> **Target**: browser-automation-studio scenario

## Executive Summary

This proposal defines a **clean, maintainable, testable** architecture for UX metrics in browser-automation-studio. The design follows screaming architecture principles—meaning anyone looking at the codebase will immediately understand its purpose and organization.

### Key Design Decisions

1. **EventSink Decorator Pattern**: Zero changes to existing executor code
2. **Interface-Driven**: Every component has testing seams
3. **Domain Isolation**: Contracts package with no external dependencies
4. **Entitlement-Gated**: Pro tier and above only

---

## Table of Contents

1. [Domain Model](#1-domain-model)
2. [Service Layer Architecture](#2-service-layer-architecture)
3. [Storage Architecture](#3-storage-architecture)
4. [API & WebSocket Contracts](#4-api--websocket-contracts)
5. [UI Consumption](#5-ui-consumption)
6. [Testing Seams](#6-testing-seams)
7. [Integration Points](#7-integration-points)
8. [Migration Path](#8-migration-path)
9. [File Structure](#9-file-structure)
10. [Screaming Architecture Audit](#10-screaming-architecture-audit-checklist)

---

## 1. Domain Model

### 1.1 Core Concepts

The UX metrics domain is centered around three concepts:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        UX METRICS DOMAIN                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  InteractionTrace      CursorPath          FrictionScore           │
│  ┌─────────────┐       ┌─────────────┐     ┌─────────────┐         │
│  │ ExecutionID │       │ Points[]    │     │ Overall     │         │
│  │ StepIndex   │       │ Duration    │     │ PerStep[]   │         │
│  │ ElementID   │       │ Distance    │     │ Signals[]   │         │
│  │ Action      │       │ Velocity    │     └─────────────┘         │
│  │ Timestamp   │       │ Acceleration│                              │
│  │ Duration    │       │ Directness  │                              │
│  │ Position    │       │ ZigZagScore │                              │
│  │ Metadata    │       └─────────────┘                              │
│  └─────────────┘                                                    │
│                                                                     │
│  SessionMetrics        StepMetrics         ExecutionMetrics         │
│  ┌─────────────┐       ┌─────────────┐     ┌─────────────┐         │
│  │ TotalClicks │       │ TimeToAct   │     │ TotalDuration│        │
│  │ TotalTime   │       │ CursorPath  │     │ StepCount    │        │
│  │ PageViews   │       │ Retries     │     │ FailureRate  │        │
│  │ ErrorCount  │       │ FrictionSig │     │ AvgStepTime  │        │
│  └─────────────┘       └─────────────┘     │ FrictionScore│        │
│                                             └─────────────┘         │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Domain Types (Contracts)

**File: `/api/services/uxmetrics/contracts/types.go`**

```go
package contracts

import (
    "time"
    "github.com/google/uuid"
)

// InteractionTrace captures a single user interaction event.
// These are the atomic building blocks of UX metrics.
type InteractionTrace struct {
    ID            uuid.UUID         `json:"id"`
    ExecutionID   uuid.UUID         `json:"execution_id"`
    StepIndex     int               `json:"step_index"`
    ActionType    ActionType        `json:"action_type"`
    ElementID     string            `json:"element_id,omitempty"`
    Selector      string            `json:"selector,omitempty"`
    Position      *Point            `json:"position,omitempty"`
    Timestamp     time.Time         `json:"timestamp"`
    DurationMs    int64             `json:"duration_ms,omitempty"`
    Success       bool              `json:"success"`
    Metadata      map[string]any    `json:"metadata,omitempty"`
}

type ActionType string

const (
    ActionClick      ActionType = "click"
    ActionType_      ActionType = "type"
    ActionScroll     ActionType = "scroll"
    ActionNavigation ActionType = "navigation"
    ActionWait       ActionType = "wait"
    ActionHover      ActionType = "hover"
    ActionDrag       ActionType = "drag"
)

// CursorPath captures the full trajectory of cursor movement for a step.
type CursorPath struct {
    StepIndex      int             `json:"step_index"`
    Points         []TimedPoint    `json:"points"`
    TotalDistancePx float64        `json:"total_distance_px"`
    DurationMs     int64           `json:"duration_ms"`
    DirectDistance float64         `json:"direct_distance_px"` // Straight-line start->end
    Directness     float64         `json:"directness"`         // 0-1, 1 = perfectly straight
    ZigZagScore    float64         `json:"zigzag_score"`       // Higher = more erratic
    AverageSpeed   float64         `json:"average_speed_px_ms"`
    MaxSpeed       float64         `json:"max_speed_px_ms"`
    Hesitations    int             `json:"hesitation_count"`   // Pauses > 200ms
}

type TimedPoint struct {
    X         float64   `json:"x"`
    Y         float64   `json:"y"`
    Timestamp time.Time `json:"timestamp"`
}

type Point struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
}

// FrictionSignal indicates a potential usability issue detected in the flow.
type FrictionSignal struct {
    Type        FrictionType   `json:"type"`
    StepIndex   int            `json:"step_index"`
    Severity    Severity       `json:"severity"`     // low, medium, high
    Score       float64        `json:"score"`        // Normalized 0-100
    Description string         `json:"description"`
    Evidence    map[string]any `json:"evidence,omitempty"`
}

type FrictionType string

const (
    FrictionExcessiveTime    FrictionType = "excessive_time"
    FrictionZigZagPath       FrictionType = "zigzag_path"
    FrictionMultipleRetries  FrictionType = "multiple_retries"
    FrictionRapidClicks      FrictionType = "rapid_clicks"
    FrictionLongHesitation   FrictionType = "long_hesitation"
    FrictionBackNavigation   FrictionType = "back_navigation"
    FrictionElementMiss      FrictionType = "element_miss"
)

type Severity string

const (
    SeverityLow    Severity = "low"
    SeverityMedium Severity = "medium"
    SeverityHigh   Severity = "high"
)

// StepMetrics aggregates UX metrics for a single step.
type StepMetrics struct {
    StepIndex        int              `json:"step_index"`
    NodeID           string           `json:"node_id"`
    StepType         string           `json:"step_type"`
    TimeToActionMs   int64            `json:"time_to_action_ms"`  // Time until user initiated action
    ActionDurationMs int64            `json:"action_duration_ms"` // Time to complete action
    TotalDurationMs  int64            `json:"total_duration_ms"`
    CursorPath       *CursorPath      `json:"cursor_path,omitempty"`
    RetryCount       int              `json:"retry_count"`
    FrictionSignals  []FrictionSignal `json:"friction_signals,omitempty"`
    FrictionScore    float64          `json:"friction_score"` // 0-100, lower is better
}

// ExecutionMetrics aggregates UX metrics for an entire execution.
type ExecutionMetrics struct {
    ExecutionID       uuid.UUID        `json:"execution_id"`
    WorkflowID        uuid.UUID        `json:"workflow_id"`
    ComputedAt        time.Time        `json:"computed_at"`
    TotalDurationMs   int64            `json:"total_duration_ms"`
    StepCount         int              `json:"step_count"`
    SuccessfulSteps   int              `json:"successful_steps"`
    FailedSteps       int              `json:"failed_steps"`
    TotalRetries      int              `json:"total_retries"`
    AvgStepDurationMs float64          `json:"avg_step_duration_ms"`
    TotalCursorDist   float64          `json:"total_cursor_distance_px"`
    OverallFriction   float64          `json:"overall_friction_score"` // 0-100
    FrictionSignals   []FrictionSignal `json:"friction_signals"`
    StepMetrics       []StepMetrics    `json:"step_metrics"`
    Summary           *MetricsSummary  `json:"summary,omitempty"`
}

// MetricsSummary provides human-readable insights.
type MetricsSummary struct {
    HighFrictionSteps  []int    `json:"high_friction_steps,omitempty"`
    SlowestSteps       []int    `json:"slowest_steps,omitempty"`
    TopFrictionTypes   []string `json:"top_friction_types,omitempty"`
    RecommendedActions []string `json:"recommended_actions,omitempty"`
}
```

---

## 2. Service Layer Architecture

### 2.1 Service Responsibilities (Screaming Architecture)

```
/api/services/uxmetrics/
├── contracts/           # Domain types (exported)
│   └── types.go
├── collector/           # Data ingestion
│   ├── collector.go     # EventSink decorator - captures raw data
│   └── collector_test.go
├── analyzer/            # Metrics computation
│   ├── analyzer.go      # Computes metrics from raw traces
│   ├── friction.go      # Friction signal detection algorithms
│   ├── cursor.go        # Cursor path analysis
│   └── analyzer_test.go
├── repository/          # Persistence (interface)
│   ├── repository.go    # Interface definition
│   └── postgres.go      # PostgreSQL implementation
├── service.go           # Orchestration facade
├── service_test.go
└── interfaces.go        # Public interfaces
```

### 2.2 Interface Definitions

**File: `/api/services/uxmetrics/interfaces.go`**

```go
package uxmetrics

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// Collector is the ingestion interface - implements automation EventSink.
// It sits in the event pipeline and captures interaction data passively.
type Collector interface {
    // OnStepOutcome receives step completion data from the executor.
    OnStepOutcome(ctx context.Context, executionID uuid.UUID, outcome StepOutcomeData) error

    // OnCursorUpdate receives cursor position updates during recording.
    OnCursorUpdate(ctx context.Context, executionID uuid.UUID, stepIndex int, point contracts.TimedPoint) error

    // FlushExecution finalizes data collection for an execution.
    FlushExecution(ctx context.Context, executionID uuid.UUID) error
}

// StepOutcomeData is the subset of StepOutcome we consume (dependency inversion).
type StepOutcomeData struct {
    StepIndex     int
    NodeID        string
    StepType      string
    Success       bool
    DurationMs    int
    ClickPosition *contracts.Point
    CursorTrail   []contracts.Point
    StartedAt     time.Time
    CompletedAt   *time.Time
}

// Analyzer computes metrics from raw interaction data.
type Analyzer interface {
    // AnalyzeExecution computes all metrics for a completed execution.
    AnalyzeExecution(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error)

    // AnalyzeStep computes metrics for a single step (for real-time).
    AnalyzeStep(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error)
}

// Repository handles persistence of UX metrics data.
type Repository interface {
    // Raw data persistence (write path)
    SaveInteractionTrace(ctx context.Context, trace *contracts.InteractionTrace) error
    SaveCursorPath(ctx context.Context, executionID uuid.UUID, path *contracts.CursorPath) error

    // Computed metrics persistence
    SaveExecutionMetrics(ctx context.Context, metrics *contracts.ExecutionMetrics) error

    // Query path
    GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error)
    GetStepMetrics(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error)
    ListInteractionTraces(ctx context.Context, executionID uuid.UUID) ([]contracts.InteractionTrace, error)
    GetCursorPath(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.CursorPath, error)

    // Aggregation queries (for dashboards)
    GetWorkflowMetricsAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*WorkflowMetricsAggregate, error)
}

// WorkflowMetricsAggregate provides workflow-level trends.
type WorkflowMetricsAggregate struct {
    WorkflowID           uuid.UUID `json:"workflow_id"`
    ExecutionCount       int       `json:"execution_count"`
    AvgFrictionScore     float64   `json:"avg_friction_score"`
    AvgDurationMs        float64   `json:"avg_duration_ms"`
    TrendDirection       string    `json:"trend_direction"` // improving, degrading, stable
    HighFrictionStepFreq map[int]int `json:"high_friction_step_frequency"`
}

// Service is the public facade for the UX metrics subsystem.
type Service interface {
    Collector() Collector
    Analyzer() Analyzer

    // High-level operations
    GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error)
    ComputeAndSaveMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error)
}
```

### 2.3 Collector Implementation (EventSink Decorator)

**File: `/api/services/uxmetrics/collector/collector.go`**

```go
package collector

import (
    "context"
    "math"
    "sync"
    "time"

    "github.com/google/uuid"
    autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
    autoevents "github.com/vrooli/browser-automation-studio/automation/events"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// Collector implements both automation/events.Sink and uxmetrics.Collector.
// It decorates the existing EventSink to passively capture UX data.
type Collector struct {
    delegate autoevents.Sink  // The underlying sink (e.g., WSHubSink)
    repo     uxmetrics.Repository

    // In-flight cursor paths (flushed on step completion)
    cursorBuffers sync.Map // map[executionID+stepIndex][]TimedPoint
}

// NewCollector creates a collector that wraps an existing EventSink.
func NewCollector(delegate autoevents.Sink, repo uxmetrics.Repository) *Collector {
    return &Collector{
        delegate: delegate,
        repo:     repo,
    }
}

// Publish implements automation/events.Sink.
// This is the key integration point - every event flows through here.
func (c *Collector) Publish(ctx context.Context, event autocontracts.EventEnvelope) error {
    // Extract UX-relevant data from events
    switch event.Kind {
    case autocontracts.EventKindStepCompleted:
        if payload, ok := event.Payload.(*autocontracts.StepOutcome); ok {
            c.onStepCompleted(ctx, event.ExecutionID, payload)
        }
    case autocontracts.EventKindStepFailed:
        if payload, ok := event.Payload.(*autocontracts.StepOutcome); ok {
            c.onStepFailed(ctx, event.ExecutionID, payload)
        }
    }

    // Always delegate to underlying sink
    return c.delegate.Publish(ctx, event)
}

// Limits delegates to the underlying sink.
func (c *Collector) Limits() autocontracts.EventBufferLimits {
    return c.delegate.Limits()
}

// OnStepOutcome implements uxmetrics.Collector for direct calls.
func (c *Collector) OnStepOutcome(ctx context.Context, executionID uuid.UUID, outcome uxmetrics.StepOutcomeData) error {
    // Convert to internal processing
    return c.processStepOutcome(ctx, executionID, outcome, true)
}

// OnCursorUpdate implements uxmetrics.Collector.
func (c *Collector) OnCursorUpdate(ctx context.Context, executionID uuid.UUID, stepIndex int, point contracts.TimedPoint) error {
    key := cursorBufferKey(executionID, stepIndex)

    val, _ := c.cursorBuffers.LoadOrStore(key, &[]contracts.TimedPoint{})
    buffer := val.(*[]contracts.TimedPoint)
    *buffer = append(*buffer, point)

    return nil
}

// FlushExecution implements uxmetrics.Collector.
func (c *Collector) FlushExecution(ctx context.Context, executionID uuid.UUID) error {
    // Clean up any remaining cursor buffers for this execution
    c.cursorBuffers.Range(func(key, value any) bool {
        if k, ok := key.(string); ok && len(k) > 36 && k[:36] == executionID.String() {
            c.cursorBuffers.Delete(key)
        }
        return true
    })
    return nil
}

func (c *Collector) onStepCompleted(ctx context.Context, executionID uuid.UUID, outcome *autocontracts.StepOutcome) {
    // Convert cursor trail to cursor path
    if len(outcome.CursorTrail) > 0 {
        path := c.buildCursorPath(outcome.StepIndex, outcome.CursorTrail, outcome.StartedAt, outcome.CompletedAt)
        _ = c.repo.SaveCursorPath(ctx, executionID, path)
    }

    // Create interaction trace
    var selector string
    if outcome.FocusedElement != nil {
        selector = outcome.FocusedElement.Selector
    }

    trace := &contracts.InteractionTrace{
        ID:          uuid.New(),
        ExecutionID: executionID,
        StepIndex:   outcome.StepIndex,
        ActionType:  mapStepTypeToAction(outcome.StepType),
        Selector:    selector,
        Position:    convertPoint(outcome.ClickPosition),
        Timestamp:   outcome.StartedAt,
        DurationMs:  int64(outcome.DurationMs),
        Success:     outcome.Success,
    }
    _ = c.repo.SaveInteractionTrace(ctx, trace)
}

func (c *Collector) onStepFailed(ctx context.Context, executionID uuid.UUID, outcome *autocontracts.StepOutcome) {
    // Same as completed but with Success=false
    c.onStepCompleted(ctx, executionID, outcome)
}

func (c *Collector) processStepOutcome(ctx context.Context, executionID uuid.UUID, outcome uxmetrics.StepOutcomeData, success bool) error {
    // Build cursor path from trail
    if len(outcome.CursorTrail) > 0 {
        trail := make([]autocontracts.CursorPosition, len(outcome.CursorTrail))
        for i, p := range outcome.CursorTrail {
            trail[i] = autocontracts.CursorPosition{
                Point: autocontracts.Point{X: p.X, Y: p.Y},
            }
        }
        path := c.buildCursorPath(outcome.StepIndex, trail, outcome.StartedAt, outcome.CompletedAt)
        if err := c.repo.SaveCursorPath(ctx, executionID, path); err != nil {
            return err
        }
    }

    // Create interaction trace
    trace := &contracts.InteractionTrace{
        ID:          uuid.New(),
        ExecutionID: executionID,
        StepIndex:   outcome.StepIndex,
        ActionType:  mapStepTypeToAction(outcome.StepType),
        Position:    outcome.ClickPosition,
        Timestamp:   outcome.StartedAt,
        DurationMs:  int64(outcome.DurationMs),
        Success:     success,
    }
    return c.repo.SaveInteractionTrace(ctx, trace)
}

func (c *Collector) buildCursorPath(stepIndex int, trail []autocontracts.CursorPosition, start time.Time, end *time.Time) *contracts.CursorPath {
    points := make([]contracts.TimedPoint, len(trail))
    totalDist := 0.0
    maxSpeed := 0.0
    hesitations := 0

    for i, pos := range trail {
        points[i] = contracts.TimedPoint{
            X:         pos.Point.X,
            Y:         pos.Point.Y,
            Timestamp: pos.RecordedAt,
        }

        if i > 0 {
            dx := pos.Point.X - trail[i-1].Point.X
            dy := pos.Point.Y - trail[i-1].Point.Y
            dist := math.Sqrt(dx*dx + dy*dy)
            totalDist += dist

            dt := pos.RecordedAt.Sub(trail[i-1].RecordedAt).Milliseconds()
            if dt > 0 {
                speed := dist / float64(dt)
                if speed > maxSpeed {
                    maxSpeed = speed
                }
            }
            if dt > 200 { // Hesitation threshold
                hesitations++
            }
        }
    }

    // Calculate directness (straight-line efficiency)
    directDist := 0.0
    if len(points) >= 2 {
        first, last := points[0], points[len(points)-1]
        dx := last.X - first.X
        dy := last.Y - first.Y
        directDist = math.Sqrt(dx*dx + dy*dy)
    }

    directness := 1.0
    if totalDist > 0 && directDist > 0 {
        directness = directDist / totalDist
    }

    durationMs := int64(0)
    if end != nil {
        durationMs = end.Sub(start).Milliseconds()
    }

    avgSpeed := 0.0
    if durationMs > 0 {
        avgSpeed = totalDist / float64(durationMs)
    }

    return &contracts.CursorPath{
        StepIndex:       stepIndex,
        Points:          points,
        TotalDistancePx: totalDist,
        DirectDistance:  directDist,
        DurationMs:      durationMs,
        Directness:      directness,
        ZigZagScore:     1.0 - directness, // Inverse of directness
        AverageSpeed:    avgSpeed,
        MaxSpeed:        maxSpeed,
        Hesitations:     hesitations,
    }
}

func cursorBufferKey(executionID uuid.UUID, stepIndex int) string {
    return executionID.String() + ":" + string(rune(stepIndex))
}

func mapStepTypeToAction(stepType string) contracts.ActionType {
    switch stepType {
    case "click":
        return contracts.ActionClick
    case "type", "input":
        return contracts.ActionType_
    case "scroll":
        return contracts.ActionScroll
    case "navigate":
        return contracts.ActionNavigation
    case "wait":
        return contracts.ActionWait
    case "hover":
        return contracts.ActionHover
    case "drag", "dragdrop":
        return contracts.ActionDrag
    default:
        return contracts.ActionClick
    }
}

func convertPoint(p *autocontracts.Point) *contracts.Point {
    if p == nil {
        return nil
    }
    return &contracts.Point{X: p.X, Y: p.Y}
}

// Compile-time interface checks
var _ autoevents.Sink = (*Collector)(nil)
var _ uxmetrics.Collector = (*Collector)(nil)
```

### 2.4 Analyzer Implementation

**File: `/api/services/uxmetrics/analyzer/analyzer.go`**

```go
package analyzer

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// Config holds tunable parameters for friction detection.
type Config struct {
    ExcessiveStepDurationMs  int64   // Default: 10000 (10s)
    ZigZagThreshold          float64 // Default: 0.3 (directness < 0.7)
    HesitationThresholdMs    int64   // Default: 3000 (3s)
    RapidClickWindowMs       int64   // Default: 500
    RapidClickCountThreshold int     // Default: 3
}

// DefaultConfig provides sensible defaults for friction detection.
var DefaultConfig = Config{
    ExcessiveStepDurationMs:  10000,
    ZigZagThreshold:          0.3,
    HesitationThresholdMs:    3000,
    RapidClickWindowMs:       500,
    RapidClickCountThreshold: 3,
}

// Analyzer computes UX metrics from collected interaction data.
type Analyzer struct {
    repo   uxmetrics.Repository
    config Config
}

// NewAnalyzer creates an analyzer with the given repository and config.
func NewAnalyzer(repo uxmetrics.Repository, config *Config) *Analyzer {
    cfg := DefaultConfig
    if config != nil {
        cfg = *config
    }
    return &Analyzer{repo: repo, config: cfg}
}

// AnalyzeExecution computes full metrics for a completed execution.
func (a *Analyzer) AnalyzeExecution(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
    traces, err := a.repo.ListInteractionTraces(ctx, executionID)
    if err != nil {
        return nil, err
    }

    stepMetrics := make([]contracts.StepMetrics, 0)
    allSignals := make([]contracts.FrictionSignal, 0)

    // Group traces by step
    stepTraces := groupByStep(traces)

    for stepIndex, sTraces := range stepTraces {
        cursorPath, _ := a.repo.GetCursorPath(ctx, executionID, stepIndex)

        sm := a.analyzeStepTraces(stepIndex, sTraces, cursorPath)
        stepMetrics = append(stepMetrics, sm)
        allSignals = append(allSignals, sm.FrictionSignals...)
    }

    // Compute aggregate metrics
    totalDuration := int64(0)
    totalRetries := 0
    successCount := 0
    failCount := 0
    totalCursorDist := 0.0
    frictionSum := 0.0

    for _, sm := range stepMetrics {
        totalDuration += sm.TotalDurationMs
        totalRetries += sm.RetryCount
        if sm.CursorPath != nil {
            totalCursorDist += sm.CursorPath.TotalDistancePx
        }
        frictionSum += sm.FrictionScore
    }

    // Count successes/failures from traces
    for _, t := range traces {
        if t.Success {
            successCount++
        } else {
            failCount++
        }
    }

    avgDuration := 0.0
    overallFriction := 0.0
    if len(stepMetrics) > 0 {
        avgDuration = float64(totalDuration) / float64(len(stepMetrics))
        overallFriction = frictionSum / float64(len(stepMetrics))
    }

    return &contracts.ExecutionMetrics{
        ExecutionID:       executionID,
        ComputedAt:        time.Now().UTC(),
        TotalDurationMs:   totalDuration,
        StepCount:         len(stepMetrics),
        SuccessfulSteps:   successCount,
        FailedSteps:       failCount,
        TotalRetries:      totalRetries,
        AvgStepDurationMs: avgDuration,
        TotalCursorDist:   totalCursorDist,
        OverallFriction:   overallFriction,
        FrictionSignals:   allSignals,
        StepMetrics:       stepMetrics,
        Summary:           a.generateSummary(stepMetrics, allSignals),
    }, nil
}

// AnalyzeStep computes metrics for a single step.
func (a *Analyzer) AnalyzeStep(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error) {
    traces, err := a.repo.ListInteractionTraces(ctx, executionID)
    if err != nil {
        return nil, err
    }

    // Filter to just this step
    stepTraces := make([]contracts.InteractionTrace, 0)
    for _, t := range traces {
        if t.StepIndex == stepIndex {
            stepTraces = append(stepTraces, t)
        }
    }

    cursorPath, _ := a.repo.GetCursorPath(ctx, executionID, stepIndex)
    sm := a.analyzeStepTraces(stepIndex, stepTraces, cursorPath)
    return &sm, nil
}

func (a *Analyzer) analyzeStepTraces(stepIndex int, traces []contracts.InteractionTrace, cursorPath *contracts.CursorPath) contracts.StepMetrics {
    signals := make([]contracts.FrictionSignal, 0)

    totalDuration := int64(0)
    var nodeID, stepType string
    for _, t := range traces {
        totalDuration += t.DurationMs
        if nodeID == "" {
            nodeID = t.ElementID
        }
        if stepType == "" {
            stepType = string(t.ActionType)
        }
    }

    // Detect friction signals
    signals = append(signals, a.detectExcessiveTime(stepIndex, totalDuration)...)
    if cursorPath != nil {
        signals = append(signals, a.detectZigZag(stepIndex, cursorPath)...)
        signals = append(signals, a.detectHesitations(stepIndex, cursorPath)...)
    }
    signals = append(signals, a.detectRapidClicks(stepIndex, traces)...)

    // Calculate friction score (weighted average of signal severities)
    frictionScore := a.calculateFrictionScore(signals)

    return contracts.StepMetrics{
        StepIndex:       stepIndex,
        NodeID:          nodeID,
        StepType:        stepType,
        TotalDurationMs: totalDuration,
        CursorPath:      cursorPath,
        FrictionSignals: signals,
        FrictionScore:   frictionScore,
    }
}

func (a *Analyzer) generateSummary(stepMetrics []contracts.StepMetrics, signals []contracts.FrictionSignal) *contracts.MetricsSummary {
    summary := &contracts.MetricsSummary{
        HighFrictionSteps:  make([]int, 0),
        SlowestSteps:       make([]int, 0),
        TopFrictionTypes:   make([]string, 0),
        RecommendedActions: make([]string, 0),
    }

    // Find high friction steps (score > 50)
    for _, sm := range stepMetrics {
        if sm.FrictionScore > 50 {
            summary.HighFrictionSteps = append(summary.HighFrictionSteps, sm.StepIndex)
        }
    }

    // Count friction types
    typeCounts := make(map[contracts.FrictionType]int)
    for _, s := range signals {
        typeCounts[s.Type]++
    }

    // Get top friction types
    for ft, count := range typeCounts {
        if count > 0 {
            summary.TopFrictionTypes = append(summary.TopFrictionTypes, string(ft))
        }
    }

    // Generate recommendations based on friction types
    if typeCounts[contracts.FrictionZigZagPath] > 0 {
        summary.RecommendedActions = append(summary.RecommendedActions,
            "Consider improving element visibility or placement to reduce cursor wandering")
    }
    if typeCounts[contracts.FrictionExcessiveTime] > 0 {
        summary.RecommendedActions = append(summary.RecommendedActions,
            "Review slow steps for performance optimization opportunities")
    }
    if typeCounts[contracts.FrictionRapidClicks] > 0 {
        summary.RecommendedActions = append(summary.RecommendedActions,
            "Check for unresponsive UI elements that may frustrate users")
    }

    return summary
}

func groupByStep(traces []contracts.InteractionTrace) map[int][]contracts.InteractionTrace {
    groups := make(map[int][]contracts.InteractionTrace)
    for _, t := range traces {
        groups[t.StepIndex] = append(groups[t.StepIndex], t)
    }
    return groups
}

// Compile-time interface check
var _ uxmetrics.Analyzer = (*Analyzer)(nil)
```

**File: `/api/services/uxmetrics/analyzer/friction.go`**

```go
package analyzer

import (
    "fmt"
    "math"

    "github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// detectExcessiveTime checks if a step took too long.
func (a *Analyzer) detectExcessiveTime(stepIndex int, durationMs int64) []contracts.FrictionSignal {
    if durationMs <= a.config.ExcessiveStepDurationMs {
        return nil
    }

    severity := contracts.SeverityLow
    score := float64(durationMs-a.config.ExcessiveStepDurationMs) / float64(a.config.ExcessiveStepDurationMs) * 50
    if score > 75 {
        severity = contracts.SeverityHigh
    } else if score > 50 {
        severity = contracts.SeverityMedium
    }

    return []contracts.FrictionSignal{{
        Type:        contracts.FrictionExcessiveTime,
        StepIndex:   stepIndex,
        Severity:    severity,
        Score:       math.Min(score, 100),
        Description: fmt.Sprintf("Step took %dms, expected <%dms", durationMs, a.config.ExcessiveStepDurationMs),
        Evidence: map[string]any{
            "actual_ms":   durationMs,
            "expected_ms": a.config.ExcessiveStepDurationMs,
        },
    }}
}

// detectZigZag checks for erratic cursor movement.
func (a *Analyzer) detectZigZag(stepIndex int, path *contracts.CursorPath) []contracts.FrictionSignal {
    if path.Directness >= (1.0 - a.config.ZigZagThreshold) {
        return nil
    }

    score := (1.0 - path.Directness) * 100
    severity := contracts.SeverityLow
    if score > 60 {
        severity = contracts.SeverityHigh
    } else if score > 40 {
        severity = contracts.SeverityMedium
    }

    return []contracts.FrictionSignal{{
        Type:        contracts.FrictionZigZagPath,
        StepIndex:   stepIndex,
        Severity:    severity,
        Score:       score,
        Description: fmt.Sprintf("Cursor path efficiency %.0f%% (%.0fpx traveled, %.0fpx direct)",
            path.Directness*100, path.TotalDistancePx, path.DirectDistance),
        Evidence: map[string]any{
            "directness":      path.Directness,
            "total_distance":  path.TotalDistancePx,
            "direct_distance": path.DirectDistance,
            "zigzag_score":    path.ZigZagScore,
        },
    }}
}

// detectHesitations checks for pauses in cursor movement.
func (a *Analyzer) detectHesitations(stepIndex int, path *contracts.CursorPath) []contracts.FrictionSignal {
    if path.Hesitations == 0 {
        return nil
    }

    score := float64(path.Hesitations) * 15 // 15 points per hesitation
    severity := contracts.SeverityLow
    if path.Hesitations >= 3 {
        severity = contracts.SeverityHigh
    } else if path.Hesitations >= 2 {
        severity = contracts.SeverityMedium
    }

    return []contracts.FrictionSignal{{
        Type:        contracts.FrictionLongHesitation,
        StepIndex:   stepIndex,
        Severity:    severity,
        Score:       math.Min(score, 100),
        Description: fmt.Sprintf("%d hesitation(s) detected (>200ms pauses)", path.Hesitations),
        Evidence: map[string]any{
            "hesitation_count": path.Hesitations,
        },
    }}
}

// detectRapidClicks checks for frustrated clicking.
func (a *Analyzer) detectRapidClicks(stepIndex int, traces []contracts.InteractionTrace) []contracts.FrictionSignal {
    clicks := filterClicks(traces)
    if len(clicks) < a.config.RapidClickCountThreshold {
        return nil
    }

    // Check for rapid succession
    rapidCount := 0
    for i := 1; i < len(clicks); i++ {
        gap := clicks[i].Timestamp.Sub(clicks[i-1].Timestamp).Milliseconds()
        if gap < a.config.RapidClickWindowMs {
            rapidCount++
        }
    }

    if rapidCount < a.config.RapidClickCountThreshold-1 {
        return nil
    }

    score := float64(rapidCount) * 25
    return []contracts.FrictionSignal{{
        Type:        contracts.FrictionRapidClicks,
        StepIndex:   stepIndex,
        Severity:    contracts.SeverityMedium,
        Score:       math.Min(score, 100),
        Description: fmt.Sprintf("%d rapid clicks detected (possible frustration)", rapidCount+1),
        Evidence: map[string]any{
            "rapid_click_count": rapidCount + 1,
            "window_ms":         a.config.RapidClickWindowMs,
        },
    }}
}

func (a *Analyzer) calculateFrictionScore(signals []contracts.FrictionSignal) float64 {
    if len(signals) == 0 {
        return 0
    }

    // Weighted by severity
    weights := map[contracts.Severity]float64{
        contracts.SeverityLow:    0.5,
        contracts.SeverityMedium: 1.0,
        contracts.SeverityHigh:   2.0,
    }

    totalWeight := 0.0
    weightedSum := 0.0
    for _, s := range signals {
        w := weights[s.Severity]
        totalWeight += w
        weightedSum += s.Score * w
    }

    return math.Min(weightedSum/totalWeight, 100)
}

func filterClicks(traces []contracts.InteractionTrace) []contracts.InteractionTrace {
    clicks := make([]contracts.InteractionTrace, 0)
    for _, t := range traces {
        if t.ActionType == contracts.ActionClick {
            clicks = append(clicks, t)
        }
    }
    return clicks
}
```

---

## 3. Storage Architecture

### 3.1 Database Schema

**File: Schema addition for UX metrics (add to existing migrations)**

```sql
-- UX Metrics Schema Extension
-- Follows existing patterns in schema.sql

-- Raw interaction traces (write-heavy, append-only)
CREATE TABLE ux_interaction_traces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    element_id VARCHAR(255),
    selector TEXT,
    position_x DOUBLE PRECISION,
    position_y DOUBLE PRECISION,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_ms BIGINT,
    success BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ux_traces_execution ON ux_interaction_traces(execution_id);
CREATE INDEX idx_ux_traces_step ON ux_interaction_traces(execution_id, step_index);

-- Cursor paths (one per step, stores full trajectory)
CREATE TABLE ux_cursor_paths (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    points JSONB NOT NULL,  -- Array of {x, y, timestamp}
    total_distance_px DOUBLE PRECISION,
    direct_distance_px DOUBLE PRECISION,
    duration_ms BIGINT,
    directness DOUBLE PRECISION,
    zigzag_score DOUBLE PRECISION,
    average_speed DOUBLE PRECISION,
    max_speed DOUBLE PRECISION,
    hesitation_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(execution_id, step_index)
);

CREATE INDEX idx_ux_cursor_execution ON ux_cursor_paths(execution_id);

-- Computed execution metrics (materialized for fast queries)
CREATE TABLE ux_execution_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    computed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    total_duration_ms BIGINT,
    step_count INTEGER,
    successful_steps INTEGER,
    failed_steps INTEGER,
    total_retries INTEGER,
    avg_step_duration_ms DOUBLE PRECISION,
    total_cursor_distance DOUBLE PRECISION,
    overall_friction_score DOUBLE PRECISION,
    friction_signals JSONB DEFAULT '[]',
    step_metrics JSONB DEFAULT '[]',
    summary JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(execution_id)
);

CREATE INDEX idx_ux_metrics_workflow ON ux_execution_metrics(workflow_id);
CREATE INDEX idx_ux_metrics_friction ON ux_execution_metrics(overall_friction_score);
```

### 3.2 Repository Implementation

**File: `/api/services/uxmetrics/repository/postgres.go`**

```go
package repository

import (
    "context"
    "database/sql"
    "encoding/json"

    "github.com/google/uuid"
    "github.com/vrooli/browser-automation-studio/database"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// PostgresRepository implements uxmetrics.Repository using PostgreSQL.
type PostgresRepository struct {
    db *database.DB
}

// NewPostgresRepository creates a new PostgreSQL-backed repository.
func NewPostgresRepository(db *database.DB) *PostgresRepository {
    return &PostgresRepository{db: db}
}

// SaveInteractionTrace persists a single interaction trace.
func (r *PostgresRepository) SaveInteractionTrace(ctx context.Context, trace *contracts.InteractionTrace) error {
    query := `
        INSERT INTO ux_interaction_traces
        (id, execution_id, step_index, action_type, element_id, selector,
         position_x, position_y, timestamp, duration_ms, success, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `

    var posX, posY *float64
    if trace.Position != nil {
        posX, posY = &trace.Position.X, &trace.Position.Y
    }

    metadataJSON, _ := json.Marshal(trace.Metadata)

    _, err := r.db.ExecContext(ctx, query,
        trace.ID, trace.ExecutionID, trace.StepIndex, trace.ActionType,
        trace.ElementID, trace.Selector, posX, posY,
        trace.Timestamp, trace.DurationMs, trace.Success, metadataJSON,
    )
    return err
}

// SaveCursorPath persists a cursor path for a step.
func (r *PostgresRepository) SaveCursorPath(ctx context.Context, executionID uuid.UUID, path *contracts.CursorPath) error {
    query := `
        INSERT INTO ux_cursor_paths
        (execution_id, step_index, points, total_distance_px, direct_distance_px,
         duration_ms, directness, zigzag_score, average_speed, max_speed, hesitation_count)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (execution_id, step_index)
        DO UPDATE SET points = EXCLUDED.points,
                      total_distance_px = EXCLUDED.total_distance_px,
                      directness = EXCLUDED.directness,
                      zigzag_score = EXCLUDED.zigzag_score
    `

    pointsJSON, _ := json.Marshal(path.Points)

    _, err := r.db.ExecContext(ctx, query,
        executionID, path.StepIndex, pointsJSON, path.TotalDistancePx,
        path.DirectDistance, path.DurationMs, path.Directness,
        path.ZigZagScore, path.AverageSpeed, path.MaxSpeed, path.Hesitations,
    )
    return err
}

// SaveExecutionMetrics persists computed execution metrics.
func (r *PostgresRepository) SaveExecutionMetrics(ctx context.Context, metrics *contracts.ExecutionMetrics) error {
    query := `
        INSERT INTO ux_execution_metrics
        (execution_id, workflow_id, computed_at, total_duration_ms, step_count,
         successful_steps, failed_steps, total_retries, avg_step_duration_ms,
         total_cursor_distance, overall_friction_score, friction_signals,
         step_metrics, summary)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        ON CONFLICT (execution_id) DO UPDATE SET
            computed_at = EXCLUDED.computed_at,
            overall_friction_score = EXCLUDED.overall_friction_score,
            friction_signals = EXCLUDED.friction_signals,
            step_metrics = EXCLUDED.step_metrics,
            summary = EXCLUDED.summary
    `

    signalsJSON, _ := json.Marshal(metrics.FrictionSignals)
    stepMetricsJSON, _ := json.Marshal(metrics.StepMetrics)
    summaryJSON, _ := json.Marshal(metrics.Summary)

    _, err := r.db.ExecContext(ctx, query,
        metrics.ExecutionID, metrics.WorkflowID, metrics.ComputedAt,
        metrics.TotalDurationMs, metrics.StepCount, metrics.SuccessfulSteps,
        metrics.FailedSteps, metrics.TotalRetries, metrics.AvgStepDurationMs,
        metrics.TotalCursorDist, metrics.OverallFriction,
        signalsJSON, stepMetricsJSON, summaryJSON,
    )
    return err
}

// GetExecutionMetrics retrieves computed metrics for an execution.
func (r *PostgresRepository) GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
    query := `
        SELECT execution_id, workflow_id, computed_at, total_duration_ms,
               step_count, successful_steps, failed_steps, total_retries,
               avg_step_duration_ms, total_cursor_distance, overall_friction_score,
               friction_signals, step_metrics, summary
        FROM ux_execution_metrics
        WHERE execution_id = $1
    `

    var m contracts.ExecutionMetrics
    var signalsJSON, stepMetricsJSON, summaryJSON []byte

    err := r.db.QueryRowContext(ctx, query, executionID).Scan(
        &m.ExecutionID, &m.WorkflowID, &m.ComputedAt, &m.TotalDurationMs,
        &m.StepCount, &m.SuccessfulSteps, &m.FailedSteps, &m.TotalRetries,
        &m.AvgStepDurationMs, &m.TotalCursorDist, &m.OverallFriction,
        &signalsJSON, &stepMetricsJSON, &summaryJSON,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    _ = json.Unmarshal(signalsJSON, &m.FrictionSignals)
    _ = json.Unmarshal(stepMetricsJSON, &m.StepMetrics)
    _ = json.Unmarshal(summaryJSON, &m.Summary)

    return &m, nil
}

// GetStepMetrics retrieves metrics for a specific step.
func (r *PostgresRepository) GetStepMetrics(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error) {
    metrics, err := r.GetExecutionMetrics(ctx, executionID)
    if err != nil || metrics == nil {
        return nil, err
    }

    for _, sm := range metrics.StepMetrics {
        if sm.StepIndex == stepIndex {
            return &sm, nil
        }
    }
    return nil, nil
}

// ListInteractionTraces retrieves all interaction traces for an execution.
func (r *PostgresRepository) ListInteractionTraces(ctx context.Context, executionID uuid.UUID) ([]contracts.InteractionTrace, error) {
    query := `
        SELECT id, execution_id, step_index, action_type, element_id, selector,
               position_x, position_y, timestamp, duration_ms, success, metadata
        FROM ux_interaction_traces
        WHERE execution_id = $1
        ORDER BY timestamp ASC
    `

    rows, err := r.db.QueryContext(ctx, query, executionID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    traces := make([]contracts.InteractionTrace, 0)
    for rows.Next() {
        var t contracts.InteractionTrace
        var posX, posY sql.NullFloat64
        var metadataJSON []byte

        err := rows.Scan(
            &t.ID, &t.ExecutionID, &t.StepIndex, &t.ActionType,
            &t.ElementID, &t.Selector, &posX, &posY,
            &t.Timestamp, &t.DurationMs, &t.Success, &metadataJSON,
        )
        if err != nil {
            return nil, err
        }

        if posX.Valid && posY.Valid {
            t.Position = &contracts.Point{X: posX.Float64, Y: posY.Float64}
        }
        _ = json.Unmarshal(metadataJSON, &t.Metadata)

        traces = append(traces, t)
    }

    return traces, rows.Err()
}

// GetCursorPath retrieves the cursor path for a specific step.
func (r *PostgresRepository) GetCursorPath(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.CursorPath, error) {
    query := `
        SELECT step_index, points, total_distance_px, direct_distance_px,
               duration_ms, directness, zigzag_score, average_speed, max_speed, hesitation_count
        FROM ux_cursor_paths
        WHERE execution_id = $1 AND step_index = $2
    `

    var path contracts.CursorPath
    var pointsJSON []byte

    err := r.db.QueryRowContext(ctx, query, executionID, stepIndex).Scan(
        &path.StepIndex, &pointsJSON, &path.TotalDistancePx, &path.DirectDistance,
        &path.DurationMs, &path.Directness, &path.ZigZagScore,
        &path.AverageSpeed, &path.MaxSpeed, &path.Hesitations,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    _ = json.Unmarshal(pointsJSON, &path.Points)
    return &path, nil
}

// GetWorkflowMetricsAggregate computes aggregate metrics across executions.
func (r *PostgresRepository) GetWorkflowMetricsAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*uxmetrics.WorkflowMetricsAggregate, error) {
    query := `
        SELECT
            COUNT(*) as execution_count,
            AVG(overall_friction_score) as avg_friction,
            AVG(total_duration_ms) as avg_duration
        FROM ux_execution_metrics
        WHERE workflow_id = $1
        ORDER BY computed_at DESC
        LIMIT $2
    `

    var agg uxmetrics.WorkflowMetricsAggregate
    agg.WorkflowID = workflowID

    err := r.db.QueryRowContext(ctx, query, workflowID, limit).Scan(
        &agg.ExecutionCount, &agg.AvgFrictionScore, &agg.AvgDurationMs,
    )
    if err != nil {
        return nil, err
    }

    // TODO: Compute trend direction and high friction step frequency
    agg.TrendDirection = "stable"
    agg.HighFrictionStepFreq = make(map[int]int)

    return &agg, nil
}

// Compile-time interface check
var _ uxmetrics.Repository = (*PostgresRepository)(nil)
```

---

## 4. API & WebSocket Contracts

### 4.1 REST API Endpoints

```
GET  /api/v1/executions/{id}/ux-metrics
     → Returns ExecutionMetrics

GET  /api/v1/executions/{id}/ux-metrics/steps/{stepIndex}
     → Returns StepMetrics

GET  /api/v1/executions/{id}/ux-metrics/friction
     → Returns []FrictionSignal (filtered by severity query param)

GET  /api/v1/workflows/{id}/ux-metrics/aggregate
     → Returns WorkflowMetricsAggregate (last N executions)

POST /api/v1/executions/{id}/ux-metrics/compute
     → Triggers metric computation for completed execution
```

### 4.2 Handler Implementation

**File: `/api/handlers/uxmetrics.go`**

```go
package handlers

import (
    "net/http"
    "strconv"

    "github.com/google/uuid"
    "github.com/vrooli/browser-automation-studio/internal/httpjson"
    "github.com/vrooli/browser-automation-studio/services/entitlement"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics"
)

// UXMetricsHandler handles UX metrics HTTP endpoints.
type UXMetricsHandler struct {
    service uxmetrics.Service
}

// NewUXMetricsHandler creates a new UX metrics handler.
func NewUXMetricsHandler(service uxmetrics.Service) *UXMetricsHandler {
    return &UXMetricsHandler{service: service}
}

// GetExecutionMetrics retrieves computed UX metrics for an execution.
func (h *UXMetricsHandler) GetExecutionMetrics(w http.ResponseWriter, r *http.Request) {
    // Check entitlement (Pro tier required)
    ent := entitlement.FromContext(r.Context())
    if !ent.Tier.AtLeast(entitlement.TierPro) {
        httpjson.Error(w, http.StatusForbidden, "UX metrics requires Pro plan or higher")
        return
    }

    executionID, err := extractUUID(r, "id")
    if err != nil {
        httpjson.Error(w, http.StatusBadRequest, "invalid execution id")
        return
    }

    metrics, err := h.service.GetExecutionMetrics(r.Context(), executionID)
    if err != nil {
        httpjson.Error(w, http.StatusInternalServerError, err.Error())
        return
    }
    if metrics == nil {
        httpjson.Error(w, http.StatusNotFound, "metrics not found")
        return
    }

    httpjson.OK(w, metrics)
}

// GetStepMetrics retrieves UX metrics for a specific step.
func (h *UXMetricsHandler) GetStepMetrics(w http.ResponseWriter, r *http.Request) {
    ent := entitlement.FromContext(r.Context())
    if !ent.Tier.AtLeast(entitlement.TierPro) {
        httpjson.Error(w, http.StatusForbidden, "UX metrics requires Pro plan or higher")
        return
    }

    executionID, err := extractUUID(r, "id")
    if err != nil {
        httpjson.Error(w, http.StatusBadRequest, "invalid execution id")
        return
    }

    stepIndexStr := chi.URLParam(r, "stepIndex")
    stepIndex, err := strconv.Atoi(stepIndexStr)
    if err != nil {
        httpjson.Error(w, http.StatusBadRequest, "invalid step index")
        return
    }

    metrics, err := h.service.Analyzer().AnalyzeStep(r.Context(), executionID, stepIndex)
    if err != nil {
        httpjson.Error(w, http.StatusInternalServerError, err.Error())
        return
    }
    if metrics == nil {
        httpjson.Error(w, http.StatusNotFound, "step metrics not found")
        return
    }

    httpjson.OK(w, metrics)
}

// ComputeMetrics triggers metric computation for a completed execution.
func (h *UXMetricsHandler) ComputeMetrics(w http.ResponseWriter, r *http.Request) {
    ent := entitlement.FromContext(r.Context())
    if !ent.Tier.AtLeast(entitlement.TierPro) {
        httpjson.Error(w, http.StatusForbidden, "UX metrics requires Pro plan or higher")
        return
    }

    executionID, err := extractUUID(r, "id")
    if err != nil {
        httpjson.Error(w, http.StatusBadRequest, "invalid execution id")
        return
    }

    metrics, err := h.service.ComputeAndSaveMetrics(r.Context(), executionID)
    if err != nil {
        httpjson.Error(w, http.StatusInternalServerError, err.Error())
        return
    }

    httpjson.OK(w, metrics)
}

func extractUUID(r *http.Request, param string) (uuid.UUID, error) {
    idStr := chi.URLParam(r, param)
    return uuid.Parse(idStr)
}
```

### 4.3 WebSocket Events

New event kinds for real-time UX metrics (add to existing events.go):

```go
// In /api/automation/contracts/events.go - add these constants
const (
    // ... existing event kinds
    EventKindUXMetricsUpdate EventKind = "ux.metrics.update"  // Per-step metrics
    EventKindUXFrictionAlert EventKind = "ux.friction.alert"  // High friction detected
)

// UXMetricsUpdatePayload is the payload for ux.metrics.update events.
type UXMetricsUpdatePayload struct {
    StepIndex     int              `json:"step_index"`
    FrictionScore float64          `json:"friction_score"`
    Signals       []FrictionSignal `json:"signals,omitempty"`
}

// UXFrictionAlertPayload is the payload for ux.friction.alert events.
type UXFrictionAlertPayload struct {
    StepIndex   int            `json:"step_index"`
    Signal      FrictionSignal `json:"signal"`
    Recommended string         `json:"recommended_action,omitempty"`
}
```

---

## 5. UI Consumption

### 5.1 Store Design

**File: `/ui/src/stores/uxMetricsStore.ts`**

```typescript
import { create } from 'zustand';
import { getConfig } from '../config';

export interface Point {
  x: number;
  y: number;
}

export interface TimedPoint extends Point {
  timestamp: string;
}

export interface CursorPath {
  stepIndex: number;
  points: TimedPoint[];
  totalDistancePx: number;
  durationMs: number;
  directDistance: number;
  directness: number;
  zigzagScore: number;
  averageSpeed: number;
  maxSpeed: number;
  hesitations: number;
}

export interface FrictionSignal {
  type: string;
  stepIndex: number;
  severity: 'low' | 'medium' | 'high';
  score: number;
  description: string;
  evidence?: Record<string, unknown>;
}

export interface StepMetrics {
  stepIndex: number;
  nodeId: string;
  stepType: string;
  timeToActionMs: number;
  actionDurationMs: number;
  totalDurationMs: number;
  cursorPath?: CursorPath;
  retryCount: number;
  frictionSignals: FrictionSignal[];
  frictionScore: number;
}

export interface MetricsSummary {
  highFrictionSteps: number[];
  slowestSteps: number[];
  topFrictionTypes: string[];
  recommendedActions: string[];
}

export interface ExecutionMetrics {
  executionId: string;
  workflowId: string;
  computedAt: string;
  totalDurationMs: number;
  stepCount: number;
  successfulSteps: number;
  failedSteps: number;
  totalRetries: number;
  avgStepDurationMs: number;
  totalCursorDistance: number;
  overallFriction: number;
  frictionSignals: FrictionSignal[];
  stepMetrics: StepMetrics[];
  summary?: MetricsSummary;
}

interface UXMetricsState {
  // Per-execution metrics
  executionMetrics: Map<string, ExecutionMetrics>;

  // Real-time step metrics (updated via WebSocket)
  currentStepMetrics: StepMetrics | null;

  // Loading states
  isLoading: boolean;
  error: string | null;

  // Actions
  fetchExecutionMetrics: (executionId: string) => Promise<void>;
  computeMetrics: (executionId: string) => Promise<ExecutionMetrics | null>;
  handleMetricsUpdate: (payload: { stepIndex: number; frictionScore: number; signals?: FrictionSignal[] }) => void;
  handleFrictionAlert: (payload: { stepIndex: number; signal: FrictionSignal }) => void;
  clearMetrics: (executionId: string) => void;
  reset: () => void;
}

export const useUXMetricsStore = create<UXMetricsState>((set, get) => ({
  executionMetrics: new Map(),
  currentStepMetrics: null,
  isLoading: false,
  error: null,

  fetchExecutionMetrics: async (executionId) => {
    set({ isLoading: true, error: null });
    try {
      const config = getConfig();
      const response = await fetch(`${config.apiBase}/api/v1/executions/${executionId}/ux-metrics`);

      if (response.status === 403) {
        set({ error: 'UX metrics requires Pro plan or higher', isLoading: false });
        return;
      }

      if (response.status === 404) {
        // Metrics not yet computed
        set({ isLoading: false });
        return;
      }

      if (!response.ok) {
        throw new Error('Failed to fetch metrics');
      }

      const metrics: ExecutionMetrics = await response.json();

      set((state) => ({
        executionMetrics: new Map(state.executionMetrics).set(executionId, metrics),
        isLoading: false,
      }));
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  computeMetrics: async (executionId) => {
    set({ isLoading: true, error: null });
    try {
      const config = getConfig();
      const response = await fetch(
        `${config.apiBase}/api/v1/executions/${executionId}/ux-metrics/compute`,
        { method: 'POST' }
      );

      if (response.status === 403) {
        set({ error: 'UX metrics requires Pro plan or higher', isLoading: false });
        return null;
      }

      if (!response.ok) {
        throw new Error('Failed to compute metrics');
      }

      const metrics: ExecutionMetrics = await response.json();

      set((state) => ({
        executionMetrics: new Map(state.executionMetrics).set(executionId, metrics),
        isLoading: false,
      }));

      return metrics;
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
      return null;
    }
  },

  handleMetricsUpdate: (payload) => {
    set({
      currentStepMetrics: {
        stepIndex: payload.stepIndex,
        nodeId: '',
        stepType: '',
        timeToActionMs: 0,
        actionDurationMs: 0,
        totalDurationMs: 0,
        retryCount: 0,
        frictionSignals: payload.signals || [],
        frictionScore: payload.frictionScore,
      },
    });
  },

  handleFrictionAlert: (payload) => {
    // Could trigger toast notification or highlight in UI
    console.warn('Friction alert:', payload.signal.description);
  },

  clearMetrics: (executionId) => {
    set((state) => {
      const newMap = new Map(state.executionMetrics);
      newMap.delete(executionId);
      return { executionMetrics: newMap };
    });
  },

  reset: () => {
    set({
      executionMetrics: new Map(),
      currentStepMetrics: null,
      isLoading: false,
      error: null,
    });
  },
}));
```

### 5.2 Component Structure

```
/ui/src/features/execution/ux-metrics/
├── UXMetricsPanel.tsx           # Main panel (tab content)
├── FrictionScoreCard.tsx        # Overall friction score display
├── FrictionSignalsList.tsx      # List of friction signals
├── StepMetricsTimeline.tsx      # Visual timeline with per-step metrics
├── CursorPathVisualizer.tsx     # SVG visualization of cursor paths
├── MetricsSummaryCard.tsx       # Recommendations and insights
└── index.ts                     # Barrel export
```

**Example Component: `/ui/src/features/execution/ux-metrics/UXMetricsPanel.tsx`**

```tsx
import { useEffect } from 'react';
import { useUXMetricsStore } from '@stores/uxMetricsStore';
import { Spinner } from '@shared/components/Spinner';
import { EmptyState } from '@shared/components/EmptyState';
import { FrictionScoreCard } from './FrictionScoreCard';
import { FrictionSignalsList } from './FrictionSignalsList';
import { StepMetricsTimeline } from './StepMetricsTimeline';
import { MetricsSummaryCard } from './MetricsSummaryCard';

interface UXMetricsPanelProps {
  executionId: string;
}

export function UXMetricsPanel({ executionId }: UXMetricsPanelProps) {
  const { executionMetrics, fetchExecutionMetrics, computeMetrics, isLoading, error } = useUXMetricsStore();
  const metrics = executionMetrics.get(executionId);

  useEffect(() => {
    fetchExecutionMetrics(executionId);
  }, [executionId, fetchExecutionMetrics]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error) {
    return (
      <EmptyState
        icon="AlertTriangle"
        title="Unable to load UX metrics"
        description={error}
      />
    );
  }

  if (!metrics) {
    return (
      <EmptyState
        icon="BarChart2"
        title="No UX metrics available"
        description="UX metrics have not been computed for this execution yet."
        action={{
          label: 'Compute Metrics',
          onClick: () => computeMetrics(executionId),
        }}
      />
    );
  }

  return (
    <div className="space-y-6 p-4">
      {/* Overview Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <FrictionScoreCard score={metrics.overallFriction} />
        <div className="bg-slate-900/50 rounded-xl p-4 border border-white/5">
          <div className="text-xs uppercase tracking-wider text-slate-400 mb-1">Duration</div>
          <div className="text-2xl font-semibold text-white">
            {(metrics.totalDurationMs / 1000).toFixed(1)}s
          </div>
          <div className="text-xs text-slate-500 mt-1">
            {metrics.stepCount} steps • {metrics.avgStepDurationMs.toFixed(0)}ms avg
          </div>
        </div>
        <div className="bg-slate-900/50 rounded-xl p-4 border border-white/5">
          <div className="text-xs uppercase tracking-wider text-slate-400 mb-1">Cursor Distance</div>
          <div className="text-2xl font-semibold text-white">
            {metrics.totalCursorDistance.toFixed(0)}px
          </div>
          <div className="text-xs text-slate-500 mt-1">
            {metrics.totalRetries} retries
          </div>
        </div>
      </div>

      {/* Friction Signals */}
      {metrics.frictionSignals.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-white mb-3">Friction Signals</h3>
          <FrictionSignalsList signals={metrics.frictionSignals} />
        </div>
      )}

      {/* Step Timeline */}
      <div>
        <h3 className="text-sm font-medium text-white mb-3">Step Analysis</h3>
        <StepMetricsTimeline steps={metrics.stepMetrics} />
      </div>

      {/* Summary & Recommendations */}
      {metrics.summary && (
        <MetricsSummaryCard summary={metrics.summary} />
      )}
    </div>
  );
}
```

---

## 6. Testing Seams

### 6.1 Interface-Driven Testing

Every component is designed for testability through interfaces:

```go
// Collector can be tested with mock sink and repository
func TestCollector_OnStepCompleted(t *testing.T) {
    mockSink := &MockEventSink{}
    mockRepo := &MockUXMetricsRepository{}

    collector := NewCollector(mockSink, mockRepo)

    event := contracts.EventEnvelope{
        Kind:        contracts.EventKindStepCompleted,
        ExecutionID: uuid.New(),
        Payload: &contracts.StepOutcome{
            StepIndex: 0,
            StepType:  "click",
            Success:   true,
            CursorTrail: []contracts.CursorPosition{
                {Point: contracts.Point{X: 100, Y: 100}},
                {Point: contracts.Point{X: 200, Y: 200}},
            },
        },
    }

    err := collector.Publish(context.Background(), event)

    assert.NoError(t, err)
    assert.Equal(t, 1, len(mockRepo.SavedTraces))
    assert.Equal(t, 1, len(mockRepo.SavedCursorPaths))
    assert.Equal(t, 1, mockSink.PublishCalls) // Delegated
}

// Analyzer can be tested with mock repository
func TestAnalyzer_DetectZigZag(t *testing.T) {
    execID := uuid.New()
    mockRepo := &MockUXMetricsRepository{
        CursorPaths: map[string]*contracts.CursorPath{
            cursorPathKey(execID, 0): {
                StepIndex:  0,
                Directness: 0.3,
                ZigZagScore: 0.7,
            },
        },
        Traces: []contracts.InteractionTrace{
            {ExecutionID: execID, StepIndex: 0, ActionType: contracts.ActionClick},
        },
    }

    analyzer := NewAnalyzer(mockRepo, nil)
    metrics, err := analyzer.AnalyzeStep(context.Background(), execID, 0)

    assert.NoError(t, err)
    assert.NotNil(t, metrics)
    assert.True(t, hasFrictionType(metrics.FrictionSignals, contracts.FrictionZigZagPath))
}
```

### 6.2 Mock Implementations

**File: `/api/services/uxmetrics/repository/mock.go`**

```go
package repository

import (
    "context"
    "sync"

    "github.com/google/uuid"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics"
    "github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// MockRepository is a test double for uxmetrics.Repository.
type MockRepository struct {
    mu sync.Mutex

    SavedTraces      []contracts.InteractionTrace
    SavedCursorPaths map[string]*contracts.CursorPath
    SavedMetrics     map[uuid.UUID]*contracts.ExecutionMetrics

    // Pre-loaded data for queries
    Traces      []contracts.InteractionTrace
    CursorPaths map[string]*contracts.CursorPath
    Metrics     map[uuid.UUID]*contracts.ExecutionMetrics

    // Control behavior
    SaveError error
    GetError  error
}

// NewMockRepository creates an empty mock repository.
func NewMockRepository() *MockRepository {
    return &MockRepository{
        SavedCursorPaths: make(map[string]*contracts.CursorPath),
        SavedMetrics:     make(map[uuid.UUID]*contracts.ExecutionMetrics),
        CursorPaths:      make(map[string]*contracts.CursorPath),
        Metrics:          make(map[uuid.UUID]*contracts.ExecutionMetrics),
    }
}

func (m *MockRepository) SaveInteractionTrace(ctx context.Context, trace *contracts.InteractionTrace) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.SaveError != nil {
        return m.SaveError
    }
    m.SavedTraces = append(m.SavedTraces, *trace)
    return nil
}

func (m *MockRepository) SaveCursorPath(ctx context.Context, executionID uuid.UUID, path *contracts.CursorPath) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.SaveError != nil {
        return m.SaveError
    }
    key := cursorPathKey(executionID, path.StepIndex)
    m.SavedCursorPaths[key] = path
    return nil
}

func (m *MockRepository) SaveExecutionMetrics(ctx context.Context, metrics *contracts.ExecutionMetrics) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.SaveError != nil {
        return m.SaveError
    }
    m.SavedMetrics[metrics.ExecutionID] = metrics
    return nil
}

func (m *MockRepository) GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.GetError != nil {
        return nil, m.GetError
    }
    return m.Metrics[executionID], nil
}

func (m *MockRepository) GetStepMetrics(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error) {
    metrics, err := m.GetExecutionMetrics(ctx, executionID)
    if err != nil || metrics == nil {
        return nil, err
    }
    for _, sm := range metrics.StepMetrics {
        if sm.StepIndex == stepIndex {
            return &sm, nil
        }
    }
    return nil, nil
}

func (m *MockRepository) ListInteractionTraces(ctx context.Context, executionID uuid.UUID) ([]contracts.InteractionTrace, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.GetError != nil {
        return nil, m.GetError
    }

    result := make([]contracts.InteractionTrace, 0)
    for _, t := range m.Traces {
        if t.ExecutionID == executionID {
            result = append(result, t)
        }
    }
    return result, nil
}

func (m *MockRepository) GetCursorPath(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.CursorPath, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.GetError != nil {
        return nil, m.GetError
    }

    key := cursorPathKey(executionID, stepIndex)
    return m.CursorPaths[key], nil
}

func (m *MockRepository) GetWorkflowMetricsAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*uxmetrics.WorkflowMetricsAggregate, error) {
    return &uxmetrics.WorkflowMetricsAggregate{
        WorkflowID:     workflowID,
        ExecutionCount: 0,
        TrendDirection: "stable",
    }, nil
}

func cursorPathKey(executionID uuid.UUID, stepIndex int) string {
    return executionID.String() + ":" + string(rune(stepIndex))
}

// Compile-time interface check
var _ uxmetrics.Repository = (*MockRepository)(nil)
```

---

## 7. Integration Points

### 7.1 Wiring into Existing System

**In `/api/main.go` (or service initialization):**

```go
// Initialize UX metrics subsystem
uxMetricsRepo := uxrepository.NewPostgresRepository(db)
uxCollector := uxcollector.NewCollector(wsHubSink, uxMetricsRepo)
uxAnalyzer := uxanalyzer.NewAnalyzer(uxMetricsRepo, nil)
uxService := uxmetrics.NewService(uxCollector, uxAnalyzer, uxMetricsRepo)

// Replace the event sink with the collecting wrapper
// Before: eventSinkFactory returns wsHubSink
// After:  eventSinkFactory returns uxCollector (which wraps wsHubSink)
workflowService := workflow.NewService(
    repo, aiClient, executor, engineFactory, recorder, planCompiler,
    func() autoevents.Sink { return uxCollector }, // <-- Changed
    log,
)

// Register UX metrics handlers
uxHandler := handlers.NewUXMetricsHandler(uxService)
router.GET("/api/v1/executions/:id/ux-metrics", uxHandler.GetExecutionMetrics)
router.GET("/api/v1/executions/:id/ux-metrics/steps/:stepIndex", uxHandler.GetStepMetrics)
router.POST("/api/v1/executions/:id/ux-metrics/compute", uxHandler.ComputeMetrics)
```

### 7.2 Entitlement Gating

UX metrics are gated by subscription tier (Pro and above):

```go
// In handler (shown in handler code above)
ent := entitlement.FromContext(r.Context())
if !ent.Tier.AtLeast(entitlement.TierPro) {
    httpjson.Error(w, http.StatusForbidden, "UX metrics requires Pro plan or higher")
    return
}
```

---

## 8. Migration Path

### Phase 1: Foundation (This proposal)
1. Create `/api/services/uxmetrics/` package structure
2. Define contracts and interfaces
3. Implement Collector (EventSink decorator)
4. Implement PostgreSQL repository
5. Add database migrations
6. Wire into existing event pipeline

### Phase 2: Analysis Engine
1. Implement friction detection algorithms
2. Implement cursor path analysis
3. Add per-execution metric computation
4. Add REST API endpoints

### Phase 3: Real-Time & UI
1. Add WebSocket event broadcasting
2. Implement UI store
3. Build UX Metrics panel component
4. Add cursor path visualization

### Phase 4: Dashboard & Aggregates
1. Add workflow-level aggregation queries
2. Build dashboard widgets
3. Add historical trend analysis

---

## 9. File Structure

```
/api/services/uxmetrics/
├── contracts/
│   └── types.go              # Domain types (InteractionTrace, CursorPath, etc.)
├── collector/
│   ├── collector.go          # EventSink decorator implementation
│   └── collector_test.go
├── analyzer/
│   ├── analyzer.go           # Main analyzer orchestration
│   ├── friction.go           # Friction detection algorithms
│   ├── cursor.go             # Cursor path analysis (optional, can be in analyzer.go)
│   └── analyzer_test.go
├── repository/
│   ├── repository.go         # Interface definition (can be in interfaces.go)
│   ├── postgres.go           # PostgreSQL implementation
│   ├── mock.go               # Test mock
│   └── postgres_test.go
├── interfaces.go             # Public interfaces
├── service.go                # Facade service
└── service_test.go

/api/handlers/
└── uxmetrics.go              # HTTP handlers

/ui/src/
├── stores/
│   └── uxMetricsStore.ts     # Zustand store
└── features/execution/
    └── ux-metrics/
        ├── UXMetricsPanel.tsx
        ├── FrictionScoreCard.tsx
        ├── FrictionSignalsList.tsx
        ├── StepMetricsTimeline.tsx
        ├── CursorPathVisualizer.tsx
        ├── MetricsSummaryCard.tsx
        └── index.ts
```

---

## 10. Screaming Architecture Audit Checklist

| Criterion | Status | Notes |
|-----------|--------|-------|
| **Intent is obvious** | ✅ | Package names scream purpose: `uxmetrics`, `collector`, `analyzer`, `friction` |
| **Boundaries are clear** | ✅ | Contracts in separate package, interfaces define seams |
| **Dependencies point inward** | ✅ | Collector depends on contracts, not concrete types |
| **Testing is trivial** | ✅ | Every component has interface for mocking |
| **Changes are localized** | ✅ | Adding new friction type = one file change |
| **No framework coupling** | ✅ | Domain logic has zero HTTP/DB imports |
| **Decorator pattern** | ✅ | Collector wraps existing sink, zero changes to executor |
| **Single responsibility** | ✅ | Collector collects, Analyzer analyzes, Repository persists |
| **Open for extension** | ✅ | New friction types, new metrics, new visualizations |
| **Closed for modification** | ✅ | Core interfaces stable, implementations swappable |

---

## Appendix: Data Already Available

The existing codebase already captures rich data that UX metrics can leverage:

### From `StepOutcome` (contracts/contracts.go)
- `ClickPosition *Point` - Actual click coordinates
- `CursorTrail []CursorPosition` - Full cursor path with timestamps
- `DurationMs int` - Step execution time
- `FocusedElement *ElementFocus` - Target element info
- `HighlightRegions []HighlightRegion` - Visual focus areas
- `ZoomFactor float64` - Applied zoom level

### From `TimelineFrame` (export/types_timeline.go)
- `RetryAttempt int` - Current retry number
- `RetryHistory []RetryHistoryEntry` - Full retry history
- `ConsoleLogCount int` - Error/warning indicators
- `NetworkEventCount int` - Request activity

### From Recording (recording/types.go)
- `recordingCursor.Path [][]float64` - Raw cursor coordinates
- `recordingFrame.DurationMs int` - Per-frame timing
- `recordingFrame.ClickPosition *recordingPoint` - Click location

This data is already flowing through the system. The UX metrics layer simply needs to:
1. Intercept it via the EventSink decorator
2. Persist it in dedicated tables
3. Compute derived metrics (directness, friction scores)
4. Expose it through new API endpoints

---

## Summary

This architecture provides a **solid foundation** for UX metrics that:

1. **Integrates cleanly** with existing event pipeline via decorator pattern
2. **Has clear testing seams** at every boundary via interfaces
3. **Can be extended** without modifying existing code
4. **Follows established patterns** in the codebase (services, repository, handlers)
5. **Gates features appropriately** by subscription tier (Pro+)
6. **Supports both batch and real-time** metric computation
7. **Enables rich UI visualization** with cursor paths and friction signals

The implementation can proceed incrementally, starting with data collection (Phase 1) and progressively adding analysis, real-time features, and dashboard capabilities.
