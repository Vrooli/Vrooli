package main

import (
	"strings"
	"testing"
)

func TestRankDuplicationOpportunities_GroupsAndRanksLineDebt(t *testing.T) {
	makeFinding := func(class string, debt int, files ...string) TidinessFinding {
		locations := make([]DuplicateLocation, 0, len(files))
		for _, file := range files {
			locations = append(locations, DuplicateLocation{Path: file, StartLine: 10, EndLine: 20})
		}
		return TidinessFinding{RuleID: "DUPLICATED_CODE", Evidence: map[string]any{"duplication_class": class, "duplication_line_debt": debt, "locations": locations}}
	}
	opportunities := rankDuplicationOpportunities([]TidinessFinding{
		makeFinding(string(DuplicationClassOpportunity), 30, "api/provider/a.go", "api/provider/b.go"),
		makeFinding(string(DuplicationClassOpportunity), 40, "api/provider/c.go", "api/provider/d.go"),
		makeFinding(string(DuplicationClassHighLeverage), 70, "api/stt/a.go", "api/tts/b.go"),
		makeFinding(string(DuplicationClassStructural), 100, "api/routes/a.go", "api/routes/b.go"),
	})
	if len(opportunities) != 2 {
		t.Fatalf("opportunities = %#v, want two actionable clusters", opportunities)
	}
	if opportunities[0].Class != string(DuplicationClassHighLeverage) || opportunities[0].LineDebt != 70 {
		t.Fatalf("first opportunity = %#v, want high-leverage 70 debt", opportunities[0])
	}
	if opportunities[1].Key != "api/provider" || opportunities[1].LineDebt != 70 || opportunities[1].MemberGroups != 2 {
		t.Fatalf("provider opportunity = %#v, want grouped 70 debt", opportunities[1])
	}
	action := formatDuplicationOpportunityAction(opportunities)
	if !strings.Contains(action, "70 line-debt") || !strings.Contains(action, "high-leverage") {
		t.Fatalf("presentation action = %q, want ranked weighted opportunities", action)
	}
}
