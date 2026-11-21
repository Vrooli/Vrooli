package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Kept with the legacy generic helper so it measures the original retained-log
// cost without retaining a second production skill-read implementation.
func BenchmarkSkillReadTelemetryLegacyAtCapacity(b *testing.B) {
	now := time.Now().UTC()
	type read struct {
		ID         string `json:"id"`
		At         string `json:"at"`
		SkillID    string `json:"skillId"`
		Caller     string `json:"caller"`
		CallerKind string `json:"callerKind"`
		AgentRunID string `json:"agentRunId"`
	}
	payload := `{"id":"benchmark","at":"` + now.Format(time.RFC3339) + `","skillId":"skill","caller":"agent-member:worker","callerKind":"agent-member","agentRunId":"run"}` + "\n"
	path := filepath.Join(b.TempDir(), "legacy.jsonl")
	if err := os.WriteFile(path, []byte(strings.Repeat(payload, 20000)), 0600); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := appendBoundedJSONL(path, read{fmt.Sprint(i), now.Format(time.RFC3339), "skill", "agent-member:worker", "agent-member", "run"}, now, 20000, 30*24*time.Hour, func(r read) string { return r.At }); err != nil {
			b.Fatal(err)
		}
	}
}
