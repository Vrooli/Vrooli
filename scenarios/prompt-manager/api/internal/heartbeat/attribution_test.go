package heartbeat

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"prompt-manager/internal/store"
)

// encodeAttribution produces a header value from a fully-specified
// AttributionInfo. Test helper; production code uses the symmetric
// CLI helper at cli/internal/attribution.
func encodeAttribution(t *testing.T, info store.AttributionInfo) string {
	t.Helper()
	payload, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return base64.StdEncoding.EncodeToString(payload)
}

func TestParseAttributionHeader_Empty(t *testing.T) {
	_, err := parseAttributionHeader("")
	if err == nil {
		t.Fatal("expected error for empty header")
	}
	if err != errAttributionRequired {
		t.Errorf("expected errAttributionRequired sentinel, got %v", err)
	}

	// Whitespace-only is treated as empty.
	if _, err := parseAttributionHeader("   \t  "); err != errAttributionRequired {
		t.Errorf("whitespace must be treated as missing, got %v", err)
	}
}

func TestParseAttributionHeader_InvalidBase64(t *testing.T) {
	_, err := parseAttributionHeader("!!!not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if !strings.Contains(err.Error(), "invalid base64") {
		t.Errorf("expected 'invalid base64' in error, got %v", err)
	}
}

func TestParseAttributionHeader_InvalidJSON(t *testing.T) {
	bad := base64.StdEncoding.EncodeToString([]byte("not-json"))
	_, err := parseAttributionHeader(bad)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in error, got %v", err)
	}
}

func TestParseAttributionHeader_ValidAgentMember(t *testing.T) {
	encoded := encodeAttribution(t, store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		MemberID:    ptr("researcher"),
		TeamID:      ptr("marketing-crew"),
		RunID:       ptr("run-abc"),
		SpawnOrigin: store.SpawnOriginHeartbeat,
	})
	info, err := parseAttributionHeader(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Kind != store.KnowledgeKindAgentMember {
		t.Errorf("Kind = %q", info.Kind)
	}
	if info.MemberID == nil || *info.MemberID != "researcher" {
		t.Errorf("MemberID round-trip lost: %v", info.MemberID)
	}
}

func TestValidateAttribution_UnknownKind(t *testing.T) {
	info := store.AttributionInfo{Kind: "robot", SpawnOrigin: store.SpawnOriginOperatorCLI}
	err := validateAttribution(info, "team-x")
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
	if !strings.Contains(err.Error(), "unknown kind") {
		t.Errorf("err = %v", err)
	}
}

func TestValidateAttribution_LegacyRejectedAtWrite(t *testing.T) {
	info := store.AttributionInfo{
		Kind:        store.KnowledgeKindLegacy,
		SpawnOrigin: store.SpawnOriginLegacy,
	}
	err := validateAttribution(info, "team-x")
	if err == nil {
		t.Fatal("expected error rejecting legacy kind at write time")
	}
	if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("err = %v", err)
	}
}

func TestValidateAttribution_UnknownSpawnOrigin(t *testing.T) {
	info := store.AttributionInfo{
		Kind:        store.KnowledgeKindOperatorDirect,
		SpawnOrigin: "unknown-origin",
	}
	err := validateAttribution(info, "team-x")
	if err == nil {
		t.Fatal("expected error for unknown spawn_origin")
	}
	if !strings.Contains(err.Error(), "unknown spawn_origin") {
		t.Errorf("err = %v", err)
	}
}

func TestValidateAttribution_AgentMemberMissingFields(t *testing.T) {
	cases := []struct {
		name string
		info store.AttributionInfo
		want string
	}{
		{
			name: "missing member_id",
			info: store.AttributionInfo{
				Kind:        store.KnowledgeKindAgentMember,
				TeamID:      ptr("team-x"),
				RunID:       ptr("run"),
				SpawnOrigin: store.SpawnOriginHeartbeat,
			},
			want: "member_id",
		},
		{
			name: "missing team_id",
			info: store.AttributionInfo{
				Kind:        store.KnowledgeKindAgentMember,
				MemberID:    ptr("m"),
				RunID:       ptr("run"),
				SpawnOrigin: store.SpawnOriginHeartbeat,
			},
			want: "team_id",
		},
		{
			// run_id is strict for non-heartbeat origins; the rule is
			// relaxed only for spawn_origin=heartbeat (see
			// docs/agent-system/RUNTIME_ATTRIBUTION.md § Env-var bridge).
			name: "missing run_id with non-heartbeat origin",
			info: store.AttributionInfo{
				Kind:        store.KnowledgeKindAgentMember,
				MemberID:    ptr("m"),
				TeamID:      ptr("team-x"),
				SpawnOrigin: store.SpawnOriginSwarmTask,
			},
			want: "run_id",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAttribution(tc.info, "team-x")
			if err == nil {
				t.Fatal("expected validation failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want mention of %q", err, tc.want)
			}
		})
	}
}

// TestValidateAttribution_AgentMemberHeartbeatNullRunIDAccepted asserts the
// heartbeat-spawn relaxation: agent-member kind with spawn_origin=heartbeat
// is permitted to omit run_id (it cannot be known at CreateRunRequest
// construction time; see RUNTIME_ATTRIBUTION.md § Env-var bridge step 1).
// Other spawn origins remain strict — exercised by
// TestValidateAttribution_AgentMemberMissingFields above.
func TestValidateAttribution_AgentMemberHeartbeatNullRunIDAccepted(t *testing.T) {
	info := store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		MemberID:    ptr("researcher"),
		TeamID:      ptr("marketing-crew"),
		SpawnOrigin: store.SpawnOriginHeartbeat,
		// RunID intentionally nil — agent-manager assigns the UUID after
		// CreateRun returns, so it is unavailable at attribution-construction
		// time.
	}
	if err := validateAttribution(info, "marketing-crew"); err != nil {
		t.Errorf("agent-member with spawn_origin=heartbeat must accept null run_id, got %v", err)
	}

	// Empty-string run_id is treated equivalently to nil (defensive against
	// callers that send an explicit empty string instead of the JSON null).
	empty := ""
	info.RunID = &empty
	if err := validateAttribution(info, "marketing-crew"); err != nil {
		t.Errorf("agent-member with spawn_origin=heartbeat must accept empty-string run_id, got %v", err)
	}
}

func TestValidateAttribution_WriterSkillMissingFields(t *testing.T) {
	t.Run("missing source_skill_id", func(t *testing.T) {
		info := store.AttributionInfo{
			Kind:        store.KnowledgeKindWriterSkill,
			TeamID:      ptr("team-x"),
			SpawnOrigin: store.SpawnOriginHeartbeat,
		}
		err := validateAttribution(info, "team-x")
		if err == nil || !strings.Contains(err.Error(), "source_skill_id") {
			t.Errorf("expected source_skill_id error, got %v", err)
		}
	})
	t.Run("missing team_id", func(t *testing.T) {
		info := store.AttributionInfo{
			Kind:          store.KnowledgeKindWriterSkill,
			SourceSkillID: ptr("report-bug"),
			SpawnOrigin:   store.SpawnOriginHeartbeat,
		}
		err := validateAttribution(info, "team-x")
		if err == nil || !strings.Contains(err.Error(), "team_id") {
			t.Errorf("expected team_id error, got %v", err)
		}
	})
}

func TestValidateAttribution_InvestigationRequiresRunID(t *testing.T) {
	info := store.AttributionInfo{
		Kind:        store.KnowledgeKindInvestigation,
		SpawnOrigin: store.SpawnOriginInvestigation,
	}
	err := validateAttribution(info, "team-x")
	if err == nil || !strings.Contains(err.Error(), "run_id") {
		t.Errorf("expected run_id error, got %v", err)
	}
}

func TestValidateAttribution_OperatorDirectAccepted(t *testing.T) {
	info := store.AttributionInfo{
		Kind:        store.KnowledgeKindOperatorDirect,
		SpawnOrigin: store.SpawnOriginOperatorCLI,
	}
	if err := validateAttribution(info, "team-x"); err != nil {
		t.Errorf("operator-direct must validate cleanly with no extra fields, got %v", err)
	}
}

func TestValidateAttribution_TeamMismatch(t *testing.T) {
	info := store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		MemberID:    ptr("m"),
		TeamID:      ptr("marketing-crew"),
		RunID:       ptr("run"),
		SpawnOrigin: store.SpawnOriginHeartbeat,
	}
	err := validateAttribution(info, "monetization")
	if err == nil {
		t.Fatal("expected team_mismatch error")
	}
	if !strings.Contains(err.Error(), "team_mismatch") {
		t.Errorf("err = %v, want team_mismatch", err)
	}
	if !strings.Contains(err.Error(), "marketing-crew") || !strings.Contains(err.Error(), "monetization") {
		t.Errorf("err must name both team_ids, got %v", err)
	}
}

func TestValidateAttribution_TeamMatch(t *testing.T) {
	info := store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		MemberID:    ptr("m"),
		TeamID:      ptr("marketing-crew"),
		RunID:       ptr("run"),
		SpawnOrigin: store.SpawnOriginHeartbeat,
	}
	if err := validateAttribution(info, "marketing-crew"); err != nil {
		t.Errorf("matching team_ids must validate, got %v", err)
	}
}

func TestDeriveCaller(t *testing.T) {
	cases := []struct {
		name      string
		info      store.AttributionInfo
		urlTeamID string
		want      string
	}{
		{
			name: "agent-member",
			info: store.AttributionInfo{
				Kind:        store.KnowledgeKindAgentMember,
				MemberID:    ptr("researcher"),
				TeamID:      ptr("marketing-crew"),
				RunID:       ptr("run-abc"),
				SpawnOrigin: store.SpawnOriginHeartbeat,
			},
			urlTeamID: "marketing-crew",
			want:      "marketing-crew/researcher",
		},
		{
			name: "writer-skill",
			info: store.AttributionInfo{
				Kind:          store.KnowledgeKindWriterSkill,
				TeamID:        ptr("scenario-qa"),
				SourceSkillID: ptr("report-bug"),
				SpawnOrigin:   store.SpawnOriginHeartbeat,
			},
			urlTeamID: "scenario-qa",
			want:      "skill:report-bug",
		},
		{
			name:      "operator-direct",
			info:      store.AttributionInfo{Kind: store.KnowledgeKindOperatorDirect, SpawnOrigin: store.SpawnOriginOperatorCLI},
			urlTeamID: "marketing-crew",
			want:      "operator",
		},
		{
			name:      "external",
			info:      store.AttributionInfo{Kind: store.KnowledgeKindExternal, SpawnOrigin: store.SpawnOriginUnknown},
			urlTeamID: "any",
			want:      "external",
		},
		{
			name: "investigation truncates run id",
			info: store.AttributionInfo{
				Kind:        store.KnowledgeKindInvestigation,
				RunID:       ptr("01234567-89ab-cdef-0123"),
				SpawnOrigin: store.SpawnOriginInvestigation,
			},
			urlTeamID: "scenario-qa",
			want:      "investigation:01234567",
		},
		{
			name: "agent-member falls back to URL team when team_id nil",
			info: store.AttributionInfo{
				Kind:        store.KnowledgeKindAgentMember,
				MemberID:    ptr("researcher"),
				RunID:       ptr("run"),
				SpawnOrigin: store.SpawnOriginHeartbeat,
			},
			urlTeamID: "marketing-crew",
			want:      "marketing-crew/researcher",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveCaller(tc.info, tc.urlTeamID)
			if got != tc.want {
				t.Errorf("deriveCaller = %q, want %q", got, tc.want)
			}
		})
	}
}
