package workflowcatalog_test

import (
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/workflowcatalog"
)

func TestParseRejectsDuplicateParallelBranchTargetsAsDistinctQuorumMembers(t *testing.T) {
	definition := `{
		"schemaVersion":"agent-workflow/v1",
		"owner":"example",
		"key":"example/parallel-quorum",
		"version":"1.0.0",
		"inputSchema":{"type":"object","additionalProperties":false},
		"outputSchema":{"type":"object","additionalProperties":false},
		"entryNode":"fork",
		"nodes":[
			{"id":"fork","kind":"branch","branch":{"parallel":true}},
			{"id":"only","kind":"run","run":{"roleRef":"code.default","promptTemplate":"Review {{.input}}","bindings":[{"name":"input","source":"workflow_input","limit":1,"maxBytes":4096,"renderAs":"json","missingPolicy":"error"}]}},
			{"id":"join","kind":"join","join":{"strategy":"quorum","quorum":2}},
			{"id":"done","kind":"end","end":{"status":"succeeded"}}
		],
		"edges":[
			{"from":"fork","to":"only"},
			{"from":"fork","to":"only"},
			{"from":"only","to":"join"},
			{"from":"join","to":"done"}
		],
		"budgets":{"wallTimeSeconds":60,"maxTurns":4,"maxTokens":1000,"maxChargeMicroUsd":1,"maxNodeAttempts":3,"maxChildren":2,"maxConcurrency":2,"maxRecursion":2,"maxRetries":2,"maxWaitSeconds":30},
		"metadata":{"fixture":"true"}
	}`

	result, err := workflowcatalog.Parse([]byte(definition), nil)
	if err != nil {
		t.Fatalf("Parse returned boundary error: %v", err)
	}
	if !domain.HasBlockingDiagnostic(result.Diagnostics) {
		t.Fatalf("duplicate target satisfied quorum with only one unique member: diagnostics=%+v digest=%q", result.Diagnostics, result.Digest)
	}
	if result.Digest != "" {
		t.Fatalf("blocking parallel-member diagnostic must withhold digest, got %q", result.Digest)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "parallel_members" && strings.Contains(diagnostic.Path, "fork") {
			return
		}
	}
	t.Fatalf("missing blocking parallel_members diagnostic for duplicate targets: %+v", result.Diagnostics)
}
