package toolfacts_test

import (
	"strings"
	"testing"

	"agent-manager/internal/toolfacts"
)

func TestDefaultRulesAreValid(t *testing.T) {
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	if rules.UnchangedWindow < 1 || len(rules.PathKeys) == 0 {
		t.Fatalf("default rules are incomplete: %+v", rules)
	}
}

func TestParseRulesRejectsWhatItCannotTrust(t *testing.T) {
	const head = `"unchanged_window": 8, "chunk_min_events": 4, "chunk_max_events": 16, "max_line_bytes": 4096, "result_head_bytes": 64, "result_tail_bytes": 16`
	cases := map[string]string{
		"an unknown field":       `{` + head + `, "windw": 3}`,
		"a bad pattern":          `{` + head + `, "normalize": [{"id": "x", "applies": "args", "pattern": "(", "replace": ""}]}`,
		"a bad target":           `{` + head + `, "normalize": [{"id": "x", "applies": "never", "pattern": "a", "replace": ""}]}`,
		"a duplicate id":         `{` + head + `, "normalize": [{"id": "x", "applies": "args", "pattern": "a", "replace": ""}, {"id": "x", "applies": "both", "pattern": "b", "replace": ""}]}`,
		"a bad wait pattern":     `{` + head + `, "wait_commands": [{"id": "w", "pattern": "("}]}`,
		"a wait rule no id":      `{` + head + `, "wait_commands": [{"pattern": "a"}]}`,
		"a bad park key":         `{` + head + `, "park_commands": [{"id": "p", "pattern": "a", "key": "["}]}`,
		"an empty tail window":   `{` + head + `, "tail_calls": [64, 0]}`,
		"no window":              `{"chunk_min_events": 4, "chunk_max_events": 16, "max_line_bytes": 4096, "result_head_bytes": 64}`,
		"a chunk bound inverted": `{"unchanged_window": 8, "chunk_min_events": 40, "chunk_max_events": 16, "max_line_bytes": 4096, "result_head_bytes": 64}`,
	}
	for name, raw := range cases {
		if _, err := toolfacts.ParseRules([]byte(raw)); err == nil {
			t.Errorf("%s was accepted", name)
		} else if !strings.Contains(err.Error(), "toolfacts rules") {
			t.Errorf("%s: error %q does not name the rules", name, err)
		}
	}
}
