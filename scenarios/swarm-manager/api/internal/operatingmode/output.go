package operatingmode

import (
	"encoding/json"
	"regexp"
	"strings"
)

const resultEnvelopeKey = "operating_mode_result"

// PhaseResult is the typed structured output a phase round produces. The
// resolution ladder (resolution.go) extracts it from — or reconstructs it from —
// the agent's output and validates it against the phase's declared output schema.
type PhaseResult struct {
	Artifacts    []ArtifactResult `json:"artifacts,omitempty"`
	Handoff      *Handoff         `json:"handoff,omitempty"`
	Handoffs     []Handoff        `json:"handoffs,omitempty"`
	Readiness    *ReadinessReport `json:"readiness,omitempty"`
	Progress     *ProgressState   `json:"progress,omitempty"`
	Verdict      string           `json:"verdict,omitempty"`
	ReplanNeeded bool             `json:"replan_needed,omitempty"`
	BacklogSync  *BacklogSyncPlan `json:"backlog_sync,omitempty"`
}

type ArtifactResult struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	ContentType string `json:"content_type,omitempty"`
}

type BacklogSyncPlan struct {
	CompletedItems []string        `json:"completed_items,omitempty"`
	CreatedItems   []string        `json:"created_items,omitempty"`
	UpdatedItems   []string        `json:"updated_items,omitempty"`
	Proposal       json.RawMessage `json:"proposal,omitempty"`
	Rationale      string          `json:"rationale,omitempty"`
}

// fencedJSONBlockRE matches a ```json … ``` (or bare ```) fenced block wrapping a
// JSON object, used by the resolution ladder to pull an envelope out of a
// message that surrounds it with prose.
var fencedJSONBlockRE = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

// hasPhaseResultContent reports whether a decoded result carries any meaningful
// field, distinguishing a present-but-empty envelope from a real result.
func hasPhaseResultContent(result PhaseResult) bool {
	return len(result.Artifacts) > 0 || result.Handoff != nil || len(result.Handoffs) > 0 ||
		result.Readiness != nil || result.Progress != nil || strings.TrimSpace(result.Verdict) != "" ||
		result.ReplanNeeded || result.BacklogSync != nil
}
