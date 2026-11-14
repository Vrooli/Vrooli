package corpusgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestModelPolicyRoleCoverageMatchesAuthoritativeInventory(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../../../.."))
	policyPath := filepath.Join(repoRoot, "resources", "ollama", "model-policy.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("read authoritative Ollama model policy: %v", err)
	}
	var policy struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatalf("decode authoritative Ollama model policy: %v", err)
	}
	want := make([]string, 0, len(policy.Roles))
	for role := range policy.Roles {
		want = append(want, role)
	}
	sort.Strings(want)
	got := ModelPolicyRoles()
	sort.Strings(got)
	if len(policy.Roles) == 0 || len(got) != len(want) {
		t.Fatalf("policy role denominator size mismatch: corpus=%v policy=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("policy role denominator differs from model-policy.json: corpus=%v policy=%v", got, want)
		}
	}
}

func TestGatewayTaskRoleCoverageMatchesAuthoritativeCatalog(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../../../.."))
	catalogPath := filepath.Join(repoRoot, "scenarios", "ai-gateway", "config", "inference-role-catalog.json")
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("read authoritative AI Gateway task-role catalog: %v", err)
	}
	var catalog struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatalf("decode authoritative AI Gateway task-role catalog: %v", err)
	}
	want := make([]string, 0, len(catalog.Roles))
	for role := range catalog.Roles {
		want = append(want, role)
	}
	sort.Strings(want)
	counts, err := ValidateGatewayTaskRoleCoverage(nil, want, 20)
	if err == nil || len(catalog.Roles) == 0 || len(counts) != len(want) {
		t.Fatalf("task-role coverage result = %v, %v; want all catalog roles represented as deficient", counts, err)
	}
	for _, role := range want {
		if counts[role] != 0 {
			t.Fatalf("empty acceptance corpus unexpectedly counted task role %s", role)
		}
	}
}

func TestModelRoleCoverageRequiresReviewedRealCasesForEveryRole(t *testing.T) {
	roles := ModelPolicyRoles()
	cases := make([]*evalv1.EvalCase, 0, len(roles)*2+1)
	for i, role := range roles {
		for n := 0; n < 2; n++ {
			event := callerEventFixture()
			event.EventID = string(rune('a'+i)) + "-event-" + string(rune('0'+n))
			event.EvidenceRef = "agent-manager://runs/" + event.RunID + "/events/" + event.EventID
			route := gatewayRouteFixture()
			route.SourceEventID = event.EventID
			route.GatewayEventID = string(rune('a'+i)) + "-route-" + string(rune('0'+n))
			route.GatewayRoleEvidenceURI = "gateway-ledger://events/" + route.GatewayEventID
			route.Role = role
			got, err := AuditedCallerCase(event, route, CallerLabelReview{
				Reviewer: "independent reviewer", EvidenceURI: "audit://expected-docs/" + role,
				ExpectedIDs: []string{"doc/" + role}, IndependentlyValidated: true,
			})
			if err != nil {
				t.Fatalf("build reviewed case for %s: %v", role, err)
			}
			cases = append(cases, got)
		}
	}
	cases = append(cases, &evalv1.EvalCase{CaseId: "synthetic", Status: "reviewed", ExpectIds: []string{"doc/x"}, Tags: []string{"independently-reviewed", "role:chat.default"}})
	counts, err := ValidateModelRoleCoverage(cases, 2)
	if err != nil {
		t.Fatalf("adequate real caller corpus rejected: %v", err)
	}
	if counts["chat.default"] != 2 {
		t.Fatalf("synthetic fixture entered caller denominator: %d", counts["chat.default"])
	}
}

func TestModelRoleCoverageRejectsCallerTagsWithoutLinkedEvidence(t *testing.T) {
	caseWithTagsOnly := &evalv1.EvalCase{
		CaseId: "pretend", Query: "caller query", Status: "reviewed", ExpectIds: []string{"doc/expected"},
		Tags: []string{"caller-audit", "independently-reviewed", "role:chat.default"},
	}
	counts, err := ValidateModelRoleCoverage([]*evalv1.EvalCase{caseWithTagsOnly}, 1)
	if err == nil || counts["chat.default"] != 0 {
		t.Fatalf("tag-only case counted as real caller evidence: counts=%v err=%v", counts, err)
	}
}

func TestModelRoleCoverageRejectsCaseIDThatDoesNotMatchLinkedRoute(t *testing.T) {
	caller, err := AuditedCallerCase(callerEventFixture(), gatewayRouteFixture(), CallerLabelReview{
		Reviewer: "independent reviewer", EvidenceURI: "audit://expected-docs/chat.default",
		ExpectedIDs: []string{"doc/chat.default"}, IndependentlyValidated: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	caller.CaseId = "caller-unrelated"
	counts, err := ValidateModelRoleCoverage([]*evalv1.EvalCase{caller}, 1)
	if err == nil || counts["chat.default"] != 0 {
		t.Fatalf("case without deterministic source/route identity was counted: counts=%v err=%v", counts, err)
	}
}

func TestModelRoleCoverageReportsAllUnderfilledRoles(t *testing.T) {
	_, err := ValidateModelRoleCoverage(nil, 20)
	if err == nil || len(err.Error()) < len(ModelPolicyRoles()) {
		t.Fatalf("expected all role gaps to be reported, got %v", err)
	}
}
