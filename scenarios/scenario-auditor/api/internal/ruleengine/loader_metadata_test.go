package ruleengine

import (
	"testing"

	rulespkg "scenario-auditor/rules"
)

func TestParseCommentBlockEnabledFalse(t *testing.T) {
	info := Info{Rule: rulespkg.Rule{Enabled: true}}
	lines := []string{"Enabled: false"}

	updated := parseCommentBlock(lines, info)
	if updated.Rule.Enabled {
		t.Fatalf("expected rule to be disabled when Enabled: false is provided")
	}
}

func TestParseCommentBlockEnabledTrue(t *testing.T) {
	info := Info{Rule: rulespkg.Rule{Enabled: false}}
	lines := []string{"Enabled: true"}

	updated := parseCommentBlock(lines, info)
	if !updated.Rule.Enabled {
		t.Fatalf("expected rule to remain enabled when Enabled: true is provided")
	}
}

func TestParseCommentBlockIgnoresUnknownEnabledValue(t *testing.T) {
	info := Info{Rule: rulespkg.Rule{Enabled: true}}
	lines := []string{"Enabled: maybe"}

	updated := parseCommentBlock(lines, info)
	if !updated.Rule.Enabled {
		t.Fatalf("expected rule enabled state to remain unchanged for unknown Enabled value")
	}
}
