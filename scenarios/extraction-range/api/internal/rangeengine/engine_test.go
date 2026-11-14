package rangeengine

import "testing"

// [REQ:EXTRACTI-P0-001] [REQ:EXTRACTI-P0-002] The complete catalog covers all
// ten families with known answers, while deterministic scoring rejects extra
// duplicates and retains each tool's best result.
func TestCommissionedLevelCoverageAndScoring(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(engine.List()); got < 40 {
		t.Fatalf("expected at least 40 levels, got %d", got)
	}
	families := map[string]int{}
	hardByFamily := map[string]int{}
	hardCount := 0
	veryHardCount := 0
	for _, level := range engine.List() {
		families[level.Family]++
		if level.Difficulty >= 3 {
			hardCount++
			hardByFamily[level.Family]++
			if !contains(level.Tags, "hard") {
				t.Errorf("level %q has difficulty %d but is missing the hard tag", level.ID, level.Difficulty)
			}
		} else if contains(level.Tags, "hard") {
			t.Errorf("level %q is tagged hard below difficulty 3", level.ID)
		}
		if level.Difficulty >= 4 {
			veryHardCount++
			if !contains(level.Tags, "very-hard") {
				t.Errorf("level %q has difficulty %d but is missing the very-hard tag", level.ID, level.Difficulty)
			}
		}
	}
	if len(families) != 10 {
		t.Fatalf("expected all 10 families, got %d", len(families))
	}
	for family, count := range families {
		if count < 4 {
			t.Errorf("family %q has %d levels; want at least four", family, count)
		}
		if hardByFamily[family] < 2 {
			t.Errorf("family %q has %d hard levels; want at least two", family, hardByFamily[family])
		}
	}
	if hardCount < 20 || veryHardCount < 10 {
		t.Fatalf("catalog has %d hard and %d very-hard levels; want at least 20 hard and 10 very-hard", hardCount, veryHardCount)
	}
	level, ok := engine.Get("plain-tables-01")
	if !ok {
		t.Fatal("plain table level is absent")
	}
	result, err := engine.Score("test-tool", level.ID, level.Answer)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.F1 != 1 || result.Expected != len(level.Answer) {
		t.Fatalf("answer key did not pass: %+v", result)
	}
	if got := len(engine.Scoreboard()); got != 1 {
		t.Fatalf("scoreboard has %d entries, want 1", got)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func TestScoreRejectsDuplicateInflationAndTracksBest(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	level, _ := engine.Get("pagination-deep-links-01")
	result, err := engine.Score("crawler-x", level.ID, []Record{level.Answer[0], level.Answer[0]})
	if err != nil {
		t.Fatal(err)
	}
	if result.Correct != 1 || result.Passed {
		t.Fatalf("duplicate inflated score: %+v", result)
	}
	result, err = engine.Score("crawler-x", level.ID, level.Answer)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || len(engine.Scoreboard()) != 1 {
		t.Fatalf("better result was not retained: %+v", result)
	}
}
