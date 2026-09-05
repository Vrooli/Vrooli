package codecs

import "testing"

func TestSpawnCapabilitiesConformance(t *testing.T) {
	codex := NewCodexForTest().Capabilities().SpawnCapabilities
	if len(codex) != 2 {
		t.Fatalf("Codex spawn capability count = %d, want 2", len(codex))
	}
	if codex[0].ExecutionMode != "codec_pipe" || codex[0].NativeObjective {
		t.Fatalf("codec_pipe capability = %+v", codex[0])
	}
	if codex[1].ExecutionMode != "interactive" || !codex[1].NativeObjective {
		t.Fatalf("interactive capability = %+v", codex[1])
	}
	for _, codec := range []Codec{NewClaudeForTest(), NewGrokForTest(), NewOpenCodeForTest(), NewAntigravityForTest()} {
		if got := codec.Capabilities().SpawnCapabilities; len(got) != 0 {
			t.Fatalf("%s declares unverified spawn capabilities: %+v", codec.Type(), got)
		}
	}
}
