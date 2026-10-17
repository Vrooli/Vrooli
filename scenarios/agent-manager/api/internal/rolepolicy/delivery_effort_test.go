package rolepolicy

import (
	"agent-manager/internal/domain"
	"context"
	"strings"
	"testing"
)

func TestResourceDeliveryEffortEvidenceStrictAndRetained(t *testing.T) {
	for _, value := range []string{"medium", "high"} {
		raw := strings.Replace(string(validResponse(`"runner":"codex"`)), `"role":"code.default"`, `"role":"code.delivery","effort":"`+value+`"`, 1)
		executor := &fakeCommandExecutor{output: []byte(raw)}
		got, e := NewResourceRoleResolver(executor).Resolve(context.Background(), domain.RunnerTypeCodex, "code.delivery")
		if e != nil || string(got.Effort) != value {
			t.Fatal("resource effort lost", e)
		}
		snapshot := (&Resolution{Candidates: []ResolvedCandidate{{Runner: got.Runner, ResourceRole: got.Role, Model: got.Model, Effort: got.Effort, Available: true}}}).Snapshot()
		if snapshot.Candidates[0].DeclaredEffort != got.Effort {
			t.Fatal("snapshot lost effort")
		}
	}
	for _, field := range []string{"", `,"effort":""`, `,"effort":" HIGH "`, `,"effort":"future"`} {
		raw := strings.Replace(string(validResponse(`"runner":"codex"`)), `"role":"code.default"`, `"role":"code.delivery"`+field, 1)
		if _, e := NewResourceRoleResolver(&fakeCommandExecutor{output: []byte(raw)}).Resolve(context.Background(), domain.RunnerTypeCodex, "code.delivery"); e == nil {
			t.Fatal("delivery missing/invalid effort accepted")
		}
	}
	if _, e := NewResourceRoleResolver(&fakeCommandExecutor{output: validResponse(`"runner":"codex"`)}).Resolve(context.Background(), domain.RunnerTypeCodex, "code.default"); e != nil {
		t.Fatal("legacy resource role changed", e)
	}
}

func TestResourceEffortDuplicateEscapedAndCaseAliasRefuse(t *testing.T) {
	for _, field := range []string{`,"effort":"medium","effort":"high"`, `,"Effort":"medium"`, `,"effort":"medium","eff\u006frt":"medium"`} {
		raw := strings.Replace(string(validResponse(`"runner":"codex"`)), `"role":"code.default"`, `"role":"code.delivery"`+field, 1)
		if _, err := NewResourceRoleResolver(&fakeCommandExecutor{output: []byte(raw)}).Resolve(context.Background(), domain.RunnerTypeCodex, "code.delivery"); err == nil {
			t.Fatal("ambiguous effort evidence accepted")
		}
	}
}
