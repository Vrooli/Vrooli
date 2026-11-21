package stt

import (
	"fmt"
	"strings"

	"audio-tools/internal/sttengine"
)

// streamingDegradationNotice reports when a streaming session uses an engine
// that batches while a declared native streaming engine was rejected.
func streamingDegradationNotice(resolution *sttengine.Resolution, seatedID string) string {
	if resolution == nil || seatedID == "" {
		return ""
	}
	var seated *sttengine.Candidate
	rejected := make([]sttengine.Candidate, 0, len(resolution.Candidates))
	for i := range resolution.Candidates {
		candidate := resolution.Candidates[i]
		if candidate.Engine.ID == seatedID {
			seated = &resolution.Candidates[i]
			continue
		}
		if candidate.Engine.Provides.NativeStreaming && candidate.Verdict == "rejected" {
			rejected = append(rejected, candidate)
		}
	}
	if seated == nil || seated.Engine.Provides.NativeStreaming || len(rejected) == 0 {
		return ""
	}
	reasons := make([]string, 0, len(rejected))
	for _, candidate := range rejected {
		reasons = append(reasons, fmt.Sprintf("%s (%s)", candidate.Engine.ID, candidate.Reason))
	}
	return fmt.Sprintf(
		"Live transcription is running on %s, which transcribes in batches rather than streaming. The streaming engine is unavailable: %s",
		seatedID, strings.Join(reasons, "; "))
}
