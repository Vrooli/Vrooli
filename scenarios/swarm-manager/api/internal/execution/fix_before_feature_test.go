package execution

import (
	"strings"
	"testing"
)

func TestEvaluateFixBeforeFeature(t *testing.T) {
	openOnAlpha := []openRemediationItem{
		{kind: "fix", name: "alpha-bug", scenarios: []string{"alpha"}},
		{kind: "chore", name: "alpha-tidy", scenarios: []string{"alpha"}},
	}

	tests := []struct {
		name          string
		itemKind      string
		featureScenes []string
		openItems     []openRemediationItem
		mode          string
		wantBlock     bool
		wantAdvisory  bool
		wantContains  string
	}{
		{
			name:          "block mode with open fix on target scenario blocks",
			itemKind:      "execute",
			featureScenes: []string{"alpha"},
			openItems:     openOnAlpha,
			mode:          FixBeforeFeatureBlock,
			wantBlock:     true,
			wantContains:  "alpha has 2 open remediation item(s)",
		},
		{
			name:          "suggest mode with open fix returns advisory only",
			itemKind:      "execute",
			featureScenes: []string{"alpha"},
			openItems:     openOnAlpha,
			mode:          FixBeforeFeatureSuggest,
			wantAdvisory:  true,
			wantContains:  "alpha has 2 open remediation item(s)",
		},
		{
			name:          "off mode never fires",
			itemKind:      "execute",
			featureScenes: []string{"alpha"},
			openItems:     openOnAlpha,
			mode:          FixBeforeFeatureOff,
		},
		{
			name:          "empty mode never fires",
			itemKind:      "execute",
			featureScenes: []string{"alpha"},
			openItems:     openOnAlpha,
			mode:          "",
		},
		{
			name:          "non-execute kind never gates (fix item)",
			itemKind:      "fix",
			featureScenes: []string{"alpha"},
			openItems:     openOnAlpha,
			mode:          FixBeforeFeatureBlock,
		},
		{
			name:          "no open items on target scenario does not gate",
			itemKind:      "execute",
			featureScenes: []string{"beta"},
			openItems:     openOnAlpha,
			mode:          FixBeforeFeatureBlock,
		},
		{
			name:          "no target scenarios does not gate",
			itemKind:      "execute",
			featureScenes: nil,
			openItems:     openOnAlpha,
			mode:          FixBeforeFeatureBlock,
		},
		{
			name:          "multi-scenario feature gates on any overlapping scenario",
			itemKind:      "execute",
			featureScenes: []string{"alpha", "gamma"},
			openItems:     []openRemediationItem{{kind: "fix", name: "g-bug", scenarios: []string{"gamma"}}},
			mode:          FixBeforeFeatureBlock,
			wantBlock:     true,
			wantContains:  "gamma has 1 open remediation item(s) (fix/g-bug)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateFixBeforeFeature(tt.itemKind, tt.featureScenes, tt.openItems, tt.mode)
			gotBlock := got.blockingReason != ""
			gotAdvisory := got.advisory != ""
			if gotBlock != tt.wantBlock {
				t.Errorf("blockingReason presence = %v, want %v (got %q)", gotBlock, tt.wantBlock, got.blockingReason)
			}
			if gotAdvisory != tt.wantAdvisory {
				t.Errorf("advisory presence = %v, want %v (got %q)", gotAdvisory, tt.wantAdvisory, got.advisory)
			}
			if tt.wantContains != "" {
				combined := got.blockingReason + got.advisory
				if !strings.Contains(combined, tt.wantContains) {
					t.Errorf("output %q does not contain %q", combined, tt.wantContains)
				}
			}
		})
	}
}

func TestIsOpenRemediationStatus(t *testing.T) {
	open := []string{"backlog", "researching", "ready", "queued", "in_progress", "in_review", "review_pending", "needs_followup"}
	for _, s := range open {
		if !isOpenRemediationStatus(s) {
			t.Errorf("status %q should be open", s)
		}
	}
	closed := []string{"completed", "failed"}
	for _, s := range closed {
		if isOpenRemediationStatus(s) {
			t.Errorf("status %q should be closed", s)
		}
	}
	// Case/space-insensitive.
	if !isOpenRemediationStatus("  Ready ") {
		t.Errorf("trimmed/cased status should be open")
	}
	if isOpenRemediationStatus("COMPLETED") {
		t.Errorf("cased completed should be closed")
	}
}
