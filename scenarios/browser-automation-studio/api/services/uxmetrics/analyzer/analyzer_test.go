package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/repository"
)

func TestAnalyzer_AnalyzeExecution_Empty(t *testing.T) {
	repo := repository.NewMockRepository()
	analyzer := NewAnalyzer(repo, nil)

	execID := uuid.New()
	metrics, err := analyzer.AnalyzeExecution(context.Background(), execID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics == nil {
		t.Fatal("expected metrics, got nil")
	}
	if metrics.StepCount != 0 {
		t.Errorf("expected 0 steps, got %d", metrics.StepCount)
	}
	if len(metrics.FrictionSignals) != 0 {
		t.Errorf("expected 0 friction signals, got %d", len(metrics.FrictionSignals))
	}
}

func TestAnalyzer_AnalyzeExecution_WithTraces(t *testing.T) {
	repo := repository.NewMockRepository()
	analyzer := NewAnalyzer(repo, nil)

	execID := uuid.New()

	// Add some traces
	now := time.Now()
	repo.Traces = []contracts.InteractionTrace{
		{
			ID:          uuid.New(),
			ExecutionID: execID,
			StepIndex:   0,
			ActionType:  contracts.ActionClick,
			Timestamp:   now,
			DurationMs:  500,
			Success:     true,
		},
		{
			ID:          uuid.New(),
			ExecutionID: execID,
			StepIndex:   1,
			ActionType:  contracts.ActionType_,
			Timestamp:   now.Add(time.Second),
			DurationMs:  1000,
			Success:     true,
		},
	}

	metrics, err := analyzer.AnalyzeExecution(context.Background(), execID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.StepCount != 2 {
		t.Errorf("expected 2 steps, got %d", metrics.StepCount)
	}
	if metrics.SuccessfulSteps != 2 {
		t.Errorf("expected 2 successful steps, got %d", metrics.SuccessfulSteps)
	}
	if metrics.TotalDurationMs != 1500 {
		t.Errorf("expected 1500ms total duration, got %d", metrics.TotalDurationMs)
	}
}

func TestAnalyzer_DetectExcessiveTime(t *testing.T) {
	repo := repository.NewMockRepository()
	config := &Config{
		ExcessiveStepDurationMs: 5000, // 5s threshold
	}
	analyzer := NewAnalyzer(repo, config)

	// Test step that doesn't exceed threshold
	signals := analyzer.detectExcessiveTime(0, 3000)
	if len(signals) != 0 {
		t.Errorf("expected no signals for 3s step, got %d", len(signals))
	}

	// Test step that exceeds threshold
	signals = analyzer.detectExcessiveTime(0, 10000)
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal for 10s step, got %d", len(signals))
	}
	if signals[0].Type != contracts.FrictionExcessiveTime {
		t.Errorf("expected excessive_time friction type, got %s", signals[0].Type)
	}
}

func TestAnalyzer_DetectZigZag(t *testing.T) {
	repo := repository.NewMockRepository()
	config := &Config{
		ZigZagThreshold: 0.3, // directness < 0.7 triggers signal
	}
	analyzer := NewAnalyzer(repo, config)

	// Test straight path (no zigzag)
	straightPath := &contracts.CursorPath{
		StepIndex:  0,
		Directness: 0.95,
	}
	signals := analyzer.detectZigZag(0, straightPath)
	if len(signals) != 0 {
		t.Errorf("expected no signals for straight path, got %d", len(signals))
	}

	// Test zigzag path
	zigzagPath := &contracts.CursorPath{
		StepIndex:       0,
		Directness:      0.4,
		TotalDistancePx: 500,
		DirectDistance:  200,
		Points: []contracts.TimedPoint{ // Need at least 2 points
			{X: 0, Y: 0},
			{X: 200, Y: 0},
		},
	}
	signals = analyzer.detectZigZag(0, zigzagPath)
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal for zigzag path, got %d", len(signals))
	}
	if signals[0].Type != contracts.FrictionZigZagPath {
		t.Errorf("expected zigzag_path friction type, got %s", signals[0].Type)
	}
}

func TestAnalyzer_DetectHesitations(t *testing.T) {
	repo := repository.NewMockRepository()
	analyzer := NewAnalyzer(repo, nil)

	// Test no hesitations
	noHesitationPath := &contracts.CursorPath{
		StepIndex:   0,
		Hesitations: 0,
	}
	signals := analyzer.detectHesitations(0, noHesitationPath)
	if len(signals) != 0 {
		t.Errorf("expected no signals for path without hesitations, got %d", len(signals))
	}

	// Test with hesitations
	hesitationPath := &contracts.CursorPath{
		StepIndex:   0,
		Hesitations: 3,
	}
	signals = analyzer.detectHesitations(0, hesitationPath)
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal for path with hesitations, got %d", len(signals))
	}
	if signals[0].Type != contracts.FrictionLongHesitation {
		t.Errorf("expected long_hesitation friction type, got %s", signals[0].Type)
	}
	if signals[0].Severity != contracts.SeverityHigh {
		t.Errorf("expected high severity for 3 hesitations, got %s", signals[0].Severity)
	}
}

func TestAnalyzer_DetectRapidClicks(t *testing.T) {
	repo := repository.NewMockRepository()
	config := &Config{
		RapidClickWindowMs:       500,
		RapidClickCountThreshold: 3,
	}
	analyzer := NewAnalyzer(repo, config)

	// Test no rapid clicks (only 2 clicks)
	now := time.Now()
	fewClicks := []contracts.InteractionTrace{
		{ActionType: contracts.ActionClick, Timestamp: now},
		{ActionType: contracts.ActionClick, Timestamp: now.Add(100 * time.Millisecond)},
	}
	signals := analyzer.detectRapidClicks(0, fewClicks)
	if len(signals) != 0 {
		t.Errorf("expected no signals for 2 clicks, got %d", len(signals))
	}

	// Test rapid clicks
	rapidClicks := []contracts.InteractionTrace{
		{ActionType: contracts.ActionClick, Timestamp: now},
		{ActionType: contracts.ActionClick, Timestamp: now.Add(100 * time.Millisecond)},
		{ActionType: contracts.ActionClick, Timestamp: now.Add(200 * time.Millisecond)},
		{ActionType: contracts.ActionClick, Timestamp: now.Add(300 * time.Millisecond)},
	}
	signals = analyzer.detectRapidClicks(0, rapidClicks)
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal for rapid clicks, got %d", len(signals))
	}
	if signals[0].Type != contracts.FrictionRapidClicks {
		t.Errorf("expected rapid_clicks friction type, got %s", signals[0].Type)
	}
}

func TestAnalyzer_DetectMultipleRetries(t *testing.T) {
	repo := repository.NewMockRepository()
	analyzer := NewAnalyzer(repo, nil)

	// Test no retries
	signals := analyzer.detectMultipleRetries(0, 0)
	if len(signals) != 0 {
		t.Errorf("expected no signals for 0 retries, got %d", len(signals))
	}

	// Test with retries
	signals = analyzer.detectMultipleRetries(0, 2)
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal for 2 retries, got %d", len(signals))
	}
	if signals[0].Type != contracts.FrictionMultipleRetries {
		t.Errorf("expected multiple_retries friction type, got %s", signals[0].Type)
	}
}

func TestAnalyzer_CalculateFrictionScore(t *testing.T) {
	repo := repository.NewMockRepository()
	analyzer := NewAnalyzer(repo, nil)

	// Test no signals
	score := analyzer.calculateFrictionScore(nil)
	if score != 0 {
		t.Errorf("expected 0 score for no signals, got %f", score)
	}

	// Test with signals
	signals := []contracts.FrictionSignal{
		{Severity: contracts.SeverityLow, Score: 20},
		{Severity: contracts.SeverityMedium, Score: 50},
		{Severity: contracts.SeverityHigh, Score: 80},
	}
	score = analyzer.calculateFrictionScore(signals)

	// Expected: (20*0.5 + 50*1.0 + 80*2.0) / (0.5 + 1.0 + 2.0) = (10 + 50 + 160) / 3.5 = 62.86
	if score < 62 || score > 63 {
		t.Errorf("expected ~62.86 score, got %f", score)
	}
}

func TestAnalyzer_GenerateSummary(t *testing.T) {
	repo := repository.NewMockRepository()
	analyzer := NewAnalyzer(repo, nil)

	stepMetrics := []contracts.StepMetrics{
		{StepIndex: 0, FrictionScore: 30, TotalDurationMs: 5000},
		{StepIndex: 1, FrictionScore: 60, TotalDurationMs: 2000}, // High friction
		{StepIndex: 2, FrictionScore: 70, TotalDurationMs: 10000}, // High friction, slowest
	}

	signals := []contracts.FrictionSignal{
		{Type: contracts.FrictionZigZagPath},
		{Type: contracts.FrictionZigZagPath},
		{Type: contracts.FrictionExcessiveTime},
	}

	summary := analyzer.generateSummary(stepMetrics, signals)

	if len(summary.HighFrictionSteps) != 2 {
		t.Errorf("expected 2 high friction steps, got %d", len(summary.HighFrictionSteps))
	}
	if len(summary.TopFrictionTypes) == 0 {
		t.Error("expected at least one top friction type")
	}
	if len(summary.RecommendedActions) == 0 {
		t.Error("expected at least one recommended action")
	}
}

func TestAnalyzer_AnalyzeStep(t *testing.T) {
	repo := repository.NewMockRepository()
	analyzer := NewAnalyzer(repo, nil)

	execID := uuid.New()
	now := time.Now()

	// Add traces for step 0
	repo.Traces = []contracts.InteractionTrace{
		{
			ID:          uuid.New(),
			ExecutionID: execID,
			StepIndex:   0,
			ActionType:  contracts.ActionClick,
			Timestamp:   now,
			DurationMs:  500,
			Success:     true,
		},
	}

	metrics, err := analyzer.AnalyzeStep(context.Background(), execID, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics == nil {
		t.Fatal("expected metrics, got nil")
	}
	if metrics.StepIndex != 0 {
		t.Errorf("expected step index 0, got %d", metrics.StepIndex)
	}
	if metrics.TotalDurationMs != 500 {
		t.Errorf("expected 500ms duration, got %d", metrics.TotalDurationMs)
	}
}
