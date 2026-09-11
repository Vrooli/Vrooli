package codecs

import (
	"reflect"
	"testing"
)

type spawnCapabilityReceipt struct {
	runnerType    string
	executionMode string
	sandboxModes  []string
	nativeGoal    bool
	receiptPath   string
	harness       string
}

func TestSpawnCapabilitiesConformance(t *testing.T) {
	receipts := []spawnCapabilityReceipt{
		{runnerType: "claude-code", executionMode: "interactive", sandboxModes: []string{"tracking", "off"}, nativeGoal: true, receiptPath: "evidence/04-native-objective/claude-smoke", harness: "claude 2.1.268"},
		{runnerType: "codex", executionMode: "codec_pipe", sandboxModes: []string{"protected", "tracking", "off"}, receiptPath: "evidence/00-baseline/codex-codec-pipe", harness: "codex-cli 0.153.4"},
		{runnerType: "codex", executionMode: "interactive", sandboxModes: []string{"tracking", "off"}, nativeGoal: true, receiptPath: "evidence/04-native-objective/codex-smoke", harness: "codex-cli 0.153.4"},
	}
	codecs := []Codec{NewClaudeForTest(), NewCodexForTest(), NewGrokForTest(), NewOpenCodeForTest(), NewAntigravityForTest()}
	for _, codec := range codecs {
		for _, capability := range codec.Capabilities().SpawnCapabilities {
			var matched *spawnCapabilityReceipt
			for i := range receipts {
				candidate := &receipts[i]
				if candidate.runnerType == string(codec.Type()) && candidate.executionMode == capability.ExecutionMode && candidate.nativeGoal == capability.NativeObjective && reflect.DeepEqual(candidate.sandboxModes, capability.SandboxModes) {
					matched = candidate
					break
				}
			}
			if matched == nil {
				t.Fatalf("%s declares spawn capability without a matching live-smoke receipt: %+v", codec.Type(), capability)
			}
			if matched.receiptPath == "" || matched.harness == "" {
				t.Fatalf("%s receipt metadata is incomplete: %+v", codec.Type(), *matched)
			}
		}
	}
}
