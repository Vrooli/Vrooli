package teamattribution

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestWriterHeaderSharesReceiverContractAndPreservesLineage(t *testing.T) {
	member, run := "origin-member", "origin-run"
	info := WriterSkill(Info{MemberID: &member, RunID: &run, SpawnOrigin: SpawnOriginHeartbeat}, "report-bug", "scenario-qa")
	if err := Validate(info, "scenario-qa"); err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(Encode(info))
	if err != nil || *parsed.MemberID != member || *parsed.RunID != run {
		t.Fatalf("lineage lost: %+v %v", parsed, err)
	}
	raw, _ := base64.StdEncoding.DecodeString(Encode(OperatorDirect()))
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil || len(fields) != 6 || fields["run_id"] != nil {
		t.Fatalf("persisted null shape changed: %s %v", raw, err)
	}
}

func TestValidationPreservesLegacyAndSpawnExceptions(t *testing.T) {
	member, team := "member", "team"
	for _, origin := range []string{SpawnOriginConversation, SpawnOriginHeartbeat} {
		if err := Validate(Info{Kind: KindAgentMember, MemberID: &member, TeamID: &team, SpawnOrigin: origin}, team); err != nil {
			t.Fatal(err)
		}
	}
	for _, info := range []Info{
		{Kind: KindLegacy, SpawnOrigin: SpawnOriginLegacy},
		{Kind: KindAgentMember, MemberID: &member, TeamID: &team, SpawnOrigin: SpawnOriginSwarmTask},
		{Kind: KindInvestigation, SpawnOrigin: SpawnOriginInvestigation},
		{Kind: KindWriterSkill, TeamID: &team, SpawnOrigin: SpawnOriginHeartbeat},
		{Kind: KindOperatorDirect, SpawnOrigin: ""},
		{Kind: "unknown-kind", SpawnOrigin: SpawnOriginUnknown},
	} {
		if err := Validate(info, team); err == nil {
			t.Fatalf("malformed live attribution accepted: %+v", info)
		}
	}
	if _, err := Parse(Encode(Info{Kind: KindLegacy, SpawnOrigin: SpawnOriginLegacy})); err != nil {
		t.Fatalf("legacy readback rejected: %v", err)
	}
	if err := Validate(WriterSkill(OperatorDirect(), "report-bug", "other-team"), team); err == nil {
		t.Fatal("team mismatch accepted")
	}
	if _, err := Parse(""); err != ErrRequired {
		t.Fatalf("missing header contract changed: %v", err)
	}
}
