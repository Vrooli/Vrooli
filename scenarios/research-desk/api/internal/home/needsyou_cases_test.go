package home

import (
	"cmp"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The needs-you grammar is one set of cases, kept beside the agents' file check
// (needs_you.py) and run by both parsers, so the desk and the check never drift.
const grammarCases = "../../../skills/research-coordination/scripts/needs_you_cases.json"

// grammarView is an item in the shape the cases and needs_you.py's parse use.
func grammarView(it NeedsYouItem) map[string]any {
	v := map[string]any{"id": it.ID, "title": it.Title, "options": nil, "recommended": nil, "explainer": nil, "packet": nil}
	if it.optionsLine {
		keys := []string{}
		for _, o := range it.Options {
			keys = append(keys, o.Key)
		}
		v["options"] = keys
	}
	if it.Recommended != nil {
		v["recommended"] = it.Recommended.Key
	}
	if it.Explainer != "" {
		v["explainer"] = it.Explainer
	}
	if it.Packet != "" {
		v["packet"] = it.Packet
	}
	return v
}

func TestTheGrammarCasesTheFileCheckRuns(t *testing.T) {
	raw, err := os.ReadFile(grammarCases)
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Cases []struct {
			Name, Section, Text string
			Items               []map[string]any
			Problems            []string
		}
	}
	if err := json.Unmarshal(raw, &file); err != nil || len(file.Cases) == 0 {
		t.Fatalf("no cases: %v", err)
	}
	for _, c := range file.Cases {
		t.Run(c.Name, func(t *testing.T) {
			items, legacy := ParseNeedsYou(c.Text, cmp.Or(c.Section, StateNeedsSection), "")
			var problems []string
			for _, l := range legacy {
				for _, line := range strings.Split(l.Text, "\n") {
					if strings.HasPrefix(line, "### ") {
						problems = append(problems, "malformed item heading: "+line)
					}
				}
			}
			views := []map[string]any{}
			for _, it := range items {
				views = append(views, grammarView(it))
				problems = append(problems, it.Problems...)
			}
			got, _ := json.Marshal(views)
			want, _ := json.Marshal(c.Items)
			if string(got) != string(want) {
				t.Errorf("items\n%s\nwant\n%s", got, want)
			}
			if len(problems) != len(c.Problems) {
				t.Fatalf("problems %q, want ones containing %q", problems, c.Problems)
			}
			for i, want := range c.Problems {
				if !strings.Contains(problems[i], want) {
					t.Errorf("problem %q does not say %q", problems[i], want)
				}
			}
		})
	}
}
