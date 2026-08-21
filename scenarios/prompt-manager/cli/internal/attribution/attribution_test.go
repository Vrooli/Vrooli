package attribution

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"prompt-manager/internal/store"
)

func TestHeaderValue_OperatorDirectWhenEnvUnset(t *testing.T) {
	t.Setenv(EnvVar, "")

	got := HeaderValue()
	if got == "" {
		t.Fatal("HeaderValue must never be empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("base64 decode header: %v", err)
	}
	var info Info
	if err := json.Unmarshal(decoded, &info); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if info.Kind != KindOperatorDirect {
		t.Errorf("Kind = %q, want %q", info.Kind, KindOperatorDirect)
	}
	if info.SpawnOrigin != SpawnOriginOperatorCLI {
		t.Errorf("SpawnOrigin = %q, want %q", info.SpawnOrigin, SpawnOriginOperatorCLI)
	}
	if info.MemberID != nil || info.TeamID != nil || info.RunID != nil || info.SourceSkillID != nil {
		t.Errorf("operator-direct must have nil pointer fields, got %+v", info)
	}
}

func TestHeaderValue_PassthroughEnvVerbatim(t *testing.T) {
	// Per the canon, the env var IS the header value. The CLI must not
	// decode-and-re-encode — a future payload-shape change would
	// otherwise require a CLI update. The value is returned verbatim.
	verbatim := "any-base64-string-even-malformed=="
	t.Setenv(EnvVar, verbatim)

	if got := HeaderValue(); got != verbatim {
		t.Errorf("HeaderValue = %q, want verbatim %q", got, verbatim)
	}
}

func TestHeaderValue_EnvWhitespaceTreatedAsUnset(t *testing.T) {
	t.Setenv(EnvVar, "   \t\n  ")

	got := HeaderValue()
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("expected operator-direct fallback to encode cleanly, got %v", err)
	}
	if !strings.Contains(string(decoded), KindOperatorDirect) {
		t.Errorf("expected operator-direct fallback, got %s", decoded)
	}
}

func TestEncode_RoundTrip(t *testing.T) {
	memberID := "researcher"
	teamID := "marketing-crew"
	runID := "5f9c1b2a-0000-0000-0000-000000000c00"
	info := Info{
		Kind:          KindAgentMember,
		MemberID:      &memberID,
		TeamID:        &teamID,
		RunID:         &runID,
		SpawnOrigin:   SpawnOriginHeartbeat,
		SourceSkillID: nil,
	}

	encoded := Encode(info)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var got Info
	if err := json.Unmarshal(decoded, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(info, got) {
		t.Errorf("round-trip mismatch:\n want %+v\n  got %+v", info, got)
	}
}

func TestEncode_NilPointersRenderAsNull(t *testing.T) {
	// The canon (RUNTIME_ATTRIBUTION.md § structured-attribution
	// payload) preserves nulls over omission so every payload has the
	// same field set regardless of kind. Tests pin the JSON shape.
	encoded := Encode(OperatorDirect())
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	var generic map[string]any
	if err := json.Unmarshal(decoded, &generic); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, field := range []string{"member_id", "team_id", "run_id", "source_skill_id"} {
		raw, ok := generic[field]
		if !ok {
			t.Errorf("field %q missing — must marshal as JSON null, not omit", field)
			continue
		}
		if raw != nil {
			t.Errorf("field %q = %v, want nil", field, raw)
		}
	}
}

func TestHeaderMap_ContainsHeaderName(t *testing.T) {
	t.Setenv(EnvVar, "")

	m := HeaderMap()
	v, ok := m[HeaderName]
	if !ok {
		t.Fatalf("HeaderMap missing %q", HeaderName)
	}
	if v == "" {
		t.Fatalf("HeaderMap value for %q is empty", HeaderName)
	}
}

// TestKindConstants_MirrorAPI pins the CLI-side kind constants against
// the canonical API-side ones in store. Drift surfaces here.
func TestKindConstants_MirrorAPI(t *testing.T) {
	pairs := []struct {
		cli, api string
	}{
		{KindAgentMember, store.KnowledgeKindAgentMember},
		{KindWriterSkill, store.KnowledgeKindWriterSkill},
		{KindOperatorDirect, store.KnowledgeKindOperatorDirect},
		{KindExternal, store.KnowledgeKindExternal},
		{KindLegacy, store.KnowledgeKindLegacy},
		{KindInvestigation, store.KnowledgeKindInvestigation},
	}
	for _, p := range pairs {
		if p.cli != p.api {
			t.Errorf("CLI kind %q != API kind %q", p.cli, p.api)
		}
	}
}

// TestSpawnOriginConstants_MirrorAPI pins the CLI-side spawn-origin
// constants against the canonical API-side ones.
func TestSpawnOriginConstants_MirrorAPI(t *testing.T) {
	pairs := []struct {
		cli, api string
	}{
		{SpawnOriginHeartbeat, store.SpawnOriginHeartbeat},
		{SpawnOriginOperatorCLI, store.SpawnOriginOperatorCLI},
		{SpawnOriginSwarmTask, store.SpawnOriginSwarmTask},
		{SpawnOriginVisionWalk, store.SpawnOriginVisionWalk},
		{SpawnOriginInvestigation, store.SpawnOriginInvestigation},
		{SpawnOriginLegacy, store.SpawnOriginLegacy},
		{SpawnOriginUnknown, store.SpawnOriginUnknown},
	}
	for _, p := range pairs {
		if p.cli != p.api {
			t.Errorf("CLI origin %q != API origin %q", p.cli, p.api)
		}
	}
}
