package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-completeness-scoring/pkg/scoring"
)

// [REQ:SCS-HIST-001] Test score history storage
// [REQ:SCS-HIST-002] Test SQLite database
// [REQ:SCS-HIST-003] Test trend detection
// [REQ:SCS-HIST-004] Test history API

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "history-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	db, err := NewDB(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

// TestNewDB verifies database initialization
// [REQ:SCS-HIST-002] SQLite database initialization
func TestNewDB(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("expected database to be created")
	}

	if err := db.Ping(); err != nil {
		t.Errorf("expected ping to succeed, got: %v", err)
	}

	// Verify database file exists
	if _, err := os.Stat(db.Path()); os.IsNotExist(err) {
		t.Error("expected database file to exist")
	}
}

// TestDBMigration verifies schema creation
// [REQ:SCS-HIST-002] SQLite database schema
func TestDBMigration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Verify table exists by inserting a test row
	repo := NewRepository(db)
	breakdown := &scoring.ScoreBreakdown{
		Score:          50,
		Classification: "functional_incomplete",
	}

	snapshot, err := repo.Save("test-scenario", breakdown, nil)
	if err != nil {
		t.Fatalf("expected insert to succeed after migration: %v", err)
	}

	if snapshot.ID <= 0 {
		t.Error("expected snapshot ID to be assigned")
	}
}

// TestRepositorySave verifies snapshot storage
// [REQ:SCS-HIST-001] Store score snapshots
func TestRepositorySave(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	breakdown := &scoring.ScoreBreakdown{
		Score:          75,
		Classification: "mostly_complete",
		Quality: scoring.QualityScore{
			Score: 35,
			Max:   50,
		},
	}
	config := map[string]interface{}{
		"preset": "default",
	}

	snapshot, err := repo.Save("test-scenario", breakdown, config)
	if err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	if snapshot.Scenario != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got %s", snapshot.Scenario)
	}
	if snapshot.Score != 75 {
		t.Errorf("expected score 75, got %d", snapshot.Score)
	}
	if snapshot.Classification != "mostly_complete" {
		t.Errorf("expected classification 'mostly_complete', got %s", snapshot.Classification)
	}
	if snapshot.Breakdown == nil {
		t.Error("expected breakdown to be saved")
	}
	if snapshot.ConfigSnapshot == nil {
		t.Error("expected config snapshot to be saved")
	}
}

// TestRepositoryGetLatest verifies latest snapshot retrieval
func TestRepositoryGetLatest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save multiple snapshots
	for i := 1; i <= 3; i++ {
		breakdown := &scoring.ScoreBreakdown{
			Score:          i * 10,
			Classification: "test",
		}
		_, err := repo.Save("test-scenario", breakdown, nil)
		if err != nil {
			t.Fatalf("failed to save snapshot %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	latest, err := repo.GetLatest("test-scenario")
	if err != nil {
		t.Fatalf("failed to get latest: %v", err)
	}

	if latest == nil {
		t.Fatal("expected latest snapshot")
	}
	if latest.Score != 30 {
		t.Errorf("expected latest score 30, got %d", latest.Score)
	}
}

// TestRepositoryGetHistory verifies history retrieval
// [REQ:SCS-HIST-004] History API endpoint
func TestRepositoryGetHistory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save multiple snapshots
	for i := 1; i <= 5; i++ {
		breakdown := &scoring.ScoreBreakdown{
			Score:          i * 10,
			Classification: "test",
		}
		_, err := repo.Save("test-scenario", breakdown, nil)
		if err != nil {
			t.Fatalf("failed to save snapshot %d: %v", i, err)
		}
	}

	history, err := repo.GetHistory("test-scenario", 3)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(history))
	}

	// Verify order (newest first)
	if history[0].Score != 50 {
		t.Errorf("expected most recent score 50, got %d", history[0].Score)
	}
}

// TestRepositoryCount verifies snapshot counting
func TestRepositoryCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save some snapshots
	for i := 1; i <= 5; i++ {
		breakdown := &scoring.ScoreBreakdown{Score: i * 10}
		repo.Save("scenario-a", breakdown, nil)
	}
	for i := 1; i <= 3; i++ {
		breakdown := &scoring.ScoreBreakdown{Score: i * 10}
		repo.Save("scenario-b", breakdown, nil)
	}

	countA, err := repo.Count("scenario-a")
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if countA != 5 {
		t.Errorf("expected count 5 for scenario-a, got %d", countA)
	}

	countAll, err := repo.CountAll()
	if err != nil {
		t.Fatalf("failed to count all: %v", err)
	}
	if countAll != 8 {
		t.Errorf("expected total count 8, got %d", countAll)
	}
}

// TestRepositoryPrune verifies old snapshot cleanup
func TestRepositoryPrune(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save 10 snapshots
	for i := 1; i <= 10; i++ {
		breakdown := &scoring.ScoreBreakdown{Score: i * 10}
		repo.Save("test-scenario", breakdown, nil)
	}

	// Prune to keep only 3
	deleted, err := repo.Prune("test-scenario", 3)
	if err != nil {
		t.Fatalf("failed to prune: %v", err)
	}
	if deleted != 7 {
		t.Errorf("expected 7 deleted, got %d", deleted)
	}

	count, _ := repo.Count("test-scenario")
	if count != 3 {
		t.Errorf("expected 3 remaining, got %d", count)
	}
}

// TestTrendAnalyzer verifies trend detection
// [REQ:SCS-HIST-003] Trend detection
func TestTrendAnalyzer(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 3)

	// Save improving trend
	scores := []int{50, 55, 60, 65, 70}
	for _, score := range scores {
		breakdown := &scoring.ScoreBreakdown{
			Score:          score,
			Classification: "test",
		}
		repo.Save("improving-scenario", breakdown, nil)
		time.Sleep(10 * time.Millisecond)
	}

	analysis, err := analyzer.Analyze("improving-scenario", 10)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	if analysis.CurrentScore != 70 {
		t.Errorf("expected current score 70, got %d", analysis.CurrentScore)
	}
	if analysis.PreviousScore != 65 {
		t.Errorf("expected previous score 65, got %d", analysis.PreviousScore)
	}
	if analysis.Delta != 5 {
		t.Errorf("expected delta 5, got %d", analysis.Delta)
	}
	if analysis.Direction != TrendImproving {
		t.Errorf("expected improving trend, got %s", analysis.Direction)
	}
}

// TestTrendStalled verifies stall detection
// [REQ:SCS-HIST-003] Detect stalls
func TestTrendStalled(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 3) // Stall after 3 unchanged

	// Save stalled trend (same score repeatedly)
	for i := 0; i < 5; i++ {
		breakdown := &scoring.ScoreBreakdown{
			Score:          50,
			Classification: "test",
		}
		repo.Save("stalled-scenario", breakdown, nil)
		time.Sleep(10 * time.Millisecond)
	}

	analysis, err := analyzer.Analyze("stalled-scenario", 10)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	if !analysis.IsStalled {
		t.Error("expected scenario to be detected as stalled")
	}
	if analysis.Direction != TrendStalled {
		t.Errorf("expected stalled direction, got %s", analysis.Direction)
	}
	if analysis.StallCount < 3 {
		t.Errorf("expected stall count >= 3, got %d", analysis.StallCount)
	}
}

// TestTrendDeclining verifies declining trend detection
func TestTrendDeclining(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 5)

	// Save declining trend
	scores := []int{70, 65, 60, 55, 50}
	for _, score := range scores {
		breakdown := &scoring.ScoreBreakdown{
			Score:          score,
			Classification: "test",
		}
		repo.Save("declining-scenario", breakdown, nil)
		time.Sleep(10 * time.Millisecond)
	}

	analysis, err := analyzer.Analyze("declining-scenario", 10)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	if analysis.Direction != TrendDeclining {
		t.Errorf("expected declining trend, got %s", analysis.Direction)
	}
	if analysis.Delta >= 0 {
		t.Errorf("expected negative delta, got %d", analysis.Delta)
	}
}

// TestTrendStatistics verifies statistical calculations
func TestTrendStatistics(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 5)

	// Save varied scores
	scores := []int{40, 60, 50, 70, 80}
	for _, score := range scores {
		breakdown := &scoring.ScoreBreakdown{Score: score}
		repo.Save("stats-scenario", breakdown, nil)
		time.Sleep(10 * time.Millisecond)
	}

	analysis, err := analyzer.Analyze("stats-scenario", 10)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	if analysis.MinScore != 40 {
		t.Errorf("expected min 40, got %d", analysis.MinScore)
	}
	if analysis.MaxScore != 80 {
		t.Errorf("expected max 80, got %d", analysis.MaxScore)
	}
	expectedAvg := 60.0 // (40+60+50+70+80) / 5
	if analysis.AverageScore != expectedAvg {
		t.Errorf("expected average %f, got %f", expectedAvg, analysis.AverageScore)
	}
	if analysis.Volatility <= 0 {
		t.Error("expected non-zero volatility")
	}
}

// TestTrendSummary verifies aggregate summary
func TestTrendSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 5)

	// Create scenarios with different trends
	// Improving
	for _, s := range []int{50, 60} {
		repo.Save("improving", &scoring.ScoreBreakdown{Score: s}, nil)
		time.Sleep(10 * time.Millisecond)
	}
	// Declining
	for _, s := range []int{60, 50} {
		repo.Save("declining", &scoring.ScoreBreakdown{Score: s}, nil)
		time.Sleep(10 * time.Millisecond)
	}
	// Stable
	for _, s := range []int{50, 50} {
		repo.Save("stable", &scoring.ScoreBreakdown{Score: s}, nil)
		time.Sleep(10 * time.Millisecond)
	}

	summary, err := analyzer.GetTrendSummary(10)
	if err != nil {
		t.Fatalf("failed to get summary: %v", err)
	}

	if summary.TotalScenarios != 3 {
		t.Errorf("expected 3 scenarios, got %d", summary.TotalScenarios)
	}
	if summary.Improving != 1 {
		t.Errorf("expected 1 improving, got %d", summary.Improving)
	}
	if summary.Declining != 1 {
		t.Errorf("expected 1 declining, got %d", summary.Declining)
	}
}

// TestEmptyHistory verifies handling of no history
func TestEmptyHistory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 5)

	analysis, err := analyzer.Analyze("nonexistent", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analysis.Direction != TrendUnknown {
		t.Errorf("expected unknown trend for empty history, got %s", analysis.Direction)
	}
	if analysis.SnapshotsAnalyzed != 0 {
		t.Errorf("expected 0 snapshots, got %d", analysis.SnapshotsAnalyzed)
	}
}

// TestSingleSnapshot verifies handling of single snapshot
func TestSingleSnapshot(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 5)

	repo.Save("single", &scoring.ScoreBreakdown{Score: 50}, nil)

	analysis, err := analyzer.Analyze("single", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analysis.Direction != TrendUnknown {
		t.Errorf("expected unknown trend for single snapshot, got %s", analysis.Direction)
	}
	if analysis.SnapshotsAnalyzed != 1 {
		t.Errorf("expected 1 snapshot, got %d", analysis.SnapshotsAnalyzed)
	}
}

// TestDBPath verifies database path handling
func TestDBPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-path-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := NewDB(tmpDir)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	expectedPath := filepath.Join(tmpDir, "scores.db")
	if db.Path() != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, db.Path())
	}
}

// TestSaveWithSourceAndTags verifies source/tags storage
func TestSaveWithSourceAndTags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	breakdown := &scoring.ScoreBreakdown{
		Score:          75,
		Classification: "mostly_complete",
	}

	opts := SaveOptions{
		Source: "ecosystem-manager",
		Tags:   []string{"task:abc123", "iteration:5", "phase:ux"},
	}

	snapshot, err := repo.SaveWithOptions("test-scenario", breakdown, opts)
	if err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	if snapshot.Source != "ecosystem-manager" {
		t.Errorf("expected source 'ecosystem-manager', got %s", snapshot.Source)
	}
	if len(snapshot.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(snapshot.Tags))
	}
	if snapshot.Tags[0] != "task:abc123" {
		t.Errorf("expected first tag 'task:abc123', got %s", snapshot.Tags[0])
	}
}

// TestGetHistoryWithSourceFilter verifies filtering by source
func TestGetHistoryWithSourceFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save snapshots from different sources
	for i := 1; i <= 3; i++ {
		breakdown := &scoring.ScoreBreakdown{Score: i * 10, Classification: "test"}
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "ecosystem-manager"})
	}
	for i := 1; i <= 2; i++ {
		breakdown := &scoring.ScoreBreakdown{Score: i * 20, Classification: "test"}
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "cli"})
	}

	// Filter by ecosystem-manager
	filter := HistoryFilter{Source: "ecosystem-manager", Limit: 10}
	history, err := repo.GetHistoryWithFilter("test-scenario", filter)
	if err != nil {
		t.Fatalf("failed to get filtered history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("expected 3 snapshots from ecosystem-manager, got %d", len(history))
	}

	// Verify all snapshots have correct source
	for _, snap := range history {
		if snap.Source != "ecosystem-manager" {
			t.Errorf("expected source 'ecosystem-manager', got %s", snap.Source)
		}
	}
}

// TestGetHistoryWithTagFilter verifies filtering by tags
func TestGetHistoryWithTagFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save snapshots with different tags
	breakdown := &scoring.ScoreBreakdown{Score: 50, Classification: "test"}
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Source: "ecosystem-manager",
		Tags:   []string{"task:abc123", "iteration:1"},
	})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Source: "ecosystem-manager",
		Tags:   []string{"task:abc123", "iteration:2"},
	})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Source: "ecosystem-manager",
		Tags:   []string{"task:def456", "iteration:1"},
	})

	// Filter by task:abc123
	filter := HistoryFilter{Tags: []string{"task:abc123"}, Limit: 10}
	history, err := repo.GetHistoryWithFilter("test-scenario", filter)
	if err != nil {
		t.Fatalf("failed to get filtered history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("expected 2 snapshots with task:abc123, got %d", len(history))
	}
}

// TestGetHistoryWithMultipleTagFilters verifies AND logic for multiple tags
func TestGetHistoryWithMultipleTagFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save snapshots with different tag combinations
	breakdown := &scoring.ScoreBreakdown{Score: 50, Classification: "test"}
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Tags: []string{"task:abc123", "phase:ux"},
	})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Tags: []string{"task:abc123", "phase:test"},
	})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Tags: []string{"task:def456", "phase:ux"},
	})

	// Filter by both task:abc123 AND phase:ux (AND logic)
	filter := HistoryFilter{Tags: []string{"task:abc123", "phase:ux"}, Limit: 10}
	history, err := repo.GetHistoryWithFilter("test-scenario", filter)
	if err != nil {
		t.Fatalf("failed to get filtered history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 snapshot matching both tags, got %d", len(history))
	}
}

// TestGetHistoryWithSourceAndTagFilter verifies combined filtering
func TestGetHistoryWithSourceAndTagFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save snapshots with different source/tag combinations
	breakdown := &scoring.ScoreBreakdown{Score: 50, Classification: "test"}
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Source: "ecosystem-manager",
		Tags:   []string{"task:abc123"},
	})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Source: "cli",
		Tags:   []string{"task:abc123"},
	})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
		Source: "ecosystem-manager",
		Tags:   []string{"task:def456"},
	})

	// Filter by ecosystem-manager AND task:abc123
	filter := HistoryFilter{
		Source: "ecosystem-manager",
		Tags:   []string{"task:abc123"},
		Limit:  10,
	}
	history, err := repo.GetHistoryWithFilter("test-scenario", filter)
	if err != nil {
		t.Fatalf("failed to get filtered history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 snapshot matching source and tag, got %d", len(history))
	}
}

// TestCountWithFilter verifies filtered count
func TestCountWithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save snapshots from different sources
	breakdown := &scoring.ScoreBreakdown{Score: 50, Classification: "test"}
	for i := 0; i < 5; i++ {
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "ecosystem-manager"})
	}
	for i := 0; i < 3; i++ {
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "cli"})
	}

	// Count with filter
	filter := HistoryFilter{Source: "ecosystem-manager"}
	count, err := repo.CountWithFilter("test-scenario", filter)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 5 {
		t.Errorf("expected 5 snapshots from ecosystem-manager, got %d", count)
	}
}

// TestGetDistinctSources verifies source enumeration
func TestGetDistinctSources(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save snapshots from different sources
	breakdown := &scoring.ScoreBreakdown{Score: 50, Classification: "test"}
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "ecosystem-manager"})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "cli"})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "ci"})
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{}) // No source

	sources, err := repo.GetDistinctSources()
	if err != nil {
		t.Fatalf("failed to get distinct sources: %v", err)
	}

	if len(sources) != 3 {
		t.Errorf("expected 3 distinct sources, got %d", len(sources))
	}
}

// TestGetLatestWithFilter verifies filtered latest retrieval
func TestGetLatestWithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)

	// Save snapshots from different sources
	for i := 1; i <= 3; i++ {
		breakdown := &scoring.ScoreBreakdown{Score: i * 10, Classification: "test"}
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "ecosystem-manager"})
		time.Sleep(10 * time.Millisecond)
	}
	// Save a newer one from CLI
	breakdown := &scoring.ScoreBreakdown{Score: 100, Classification: "test"}
	repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "cli"})

	// Get latest from ecosystem-manager only
	filter := HistoryFilter{Source: "ecosystem-manager"}
	latest, err := repo.GetLatestWithFilter("test-scenario", filter)
	if err != nil {
		t.Fatalf("failed to get latest: %v", err)
	}

	if latest == nil {
		t.Fatal("expected latest snapshot")
	}
	if latest.Score != 30 {
		t.Errorf("expected score 30 (latest from ecosystem-manager), got %d", latest.Score)
	}
}

// TestAnalyzeWithFilter verifies filtered trend analysis
func TestAnalyzeWithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 3)

	// Save improving trend from ecosystem-manager
	for _, score := range []int{50, 55, 60, 65, 70} {
		breakdown := &scoring.ScoreBreakdown{Score: score, Classification: "test"}
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
			Source: "ecosystem-manager",
			Tags:   []string{"task:abc123"},
		})
		time.Sleep(10 * time.Millisecond)
	}

	// Save some noise from CLI
	for _, score := range []int{10, 20, 30} {
		breakdown := &scoring.ScoreBreakdown{Score: score, Classification: "test"}
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{Source: "cli"})
		time.Sleep(10 * time.Millisecond)
	}

	// Analyze only ecosystem-manager snapshots
	filter := HistoryFilter{Source: "ecosystem-manager", Tags: []string{"task:abc123"}, Limit: 10}
	analysis, err := analyzer.AnalyzeWithFilter("test-scenario", filter)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	if analysis.CurrentScore != 70 {
		t.Errorf("expected current score 70, got %d", analysis.CurrentScore)
	}
	if analysis.Direction != TrendImproving {
		t.Errorf("expected improving trend, got %s", analysis.Direction)
	}
	if analysis.SnapshotsAnalyzed != 5 {
		t.Errorf("expected 5 snapshots analyzed (from ecosystem-manager), got %d", analysis.SnapshotsAnalyzed)
	}
}

// TestStallDetectionWithFilter verifies stall detection scoped to specific task
func TestStallDetectionWithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewRepository(db)
	analyzer := NewTrendAnalyzer(repo, 3) // Stall after 3 unchanged

	// Save stalled snapshots for task:abc123
	for i := 0; i < 5; i++ {
		breakdown := &scoring.ScoreBreakdown{Score: 50, Classification: "test"}
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
			Source: "ecosystem-manager",
			Tags:   []string{"task:abc123"},
		})
		time.Sleep(10 * time.Millisecond)
	}

	// Save improving snapshots for task:def456
	for _, score := range []int{50, 55, 60} {
		breakdown := &scoring.ScoreBreakdown{Score: score, Classification: "test"}
		repo.SaveWithOptions("test-scenario", breakdown, SaveOptions{
			Source: "ecosystem-manager",
			Tags:   []string{"task:def456"},
		})
		time.Sleep(10 * time.Millisecond)
	}

	// Analyze task:abc123 only - should be stalled
	filterABC := HistoryFilter{Tags: []string{"task:abc123"}, Limit: 10}
	analysisABC, _ := analyzer.AnalyzeWithFilter("test-scenario", filterABC)
	if !analysisABC.IsStalled {
		t.Error("expected task:abc123 to be stalled")
	}

	// Analyze task:def456 only - should be improving
	filterDEF := HistoryFilter{Tags: []string{"task:def456"}, Limit: 10}
	analysisDEF, _ := analyzer.AnalyzeWithFilter("test-scenario", filterDEF)
	if analysisDEF.IsStalled {
		t.Error("expected task:def456 to NOT be stalled")
	}
	if analysisDEF.Direction != TrendImproving {
		t.Errorf("expected task:def456 to be improving, got %s", analysisDEF.Direction)
	}
}
