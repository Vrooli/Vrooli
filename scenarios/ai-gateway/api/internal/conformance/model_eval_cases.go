package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"github.com/vrooli/vrooli/packages/proto/searcheval"
)

func toModelEvalSearchCase(c *evalv1.EvalCase) aisearch.TestCase {
	return aisearch.TestCase{
		ID: c.GetCaseId(), Query: c.GetQuery(), Scope: c.GetScope(), Status: c.GetStatus(),
		Tags: append([]string(nil), c.GetTags()...), ExpectIDs: append([]string(nil), c.GetExpectIds()...),
		ExpectWithinTopK: int(c.GetExpectWithinTopK()), ExpectMinScore: c.GetExpectMinScore(),
		ExpectMaxScore: c.GetExpectMaxScore(), ExpectNoStrongHit: c.GetExpectNoStrongHit(), Note: c.GetNote(),
	}
}

func acceptedCallerCase(c *evalv1.EvalCase) bool {
	if !callerCaseReviewed(c) {
		return false
	}
	role := roleForCase(c)
	consumer := consumerForCase(c)
	if role == "" || taskRoleForCase(c) == "" || consumer == "" || c.GetCaseId() != evalCallerCaseID(c.GetNote(), role) {
		return false
	}
	return acceptedCallerEvidence(c, role, consumer)
}

func callerCaseReviewed(c *evalv1.EvalCase) bool {
	return c != nil && c.GetStatus() == "reviewed" && len(c.GetExpectIds()) > 0 &&
		!hasEvalTag(c, "generated") && hasEvalTag(c, "caller-audit") &&
		hasEvalTag(c, "agent-manager") && hasEvalTag(c, "independently-reviewed")
}

func acceptedCallerEvidence(c *evalv1.EvalCase, role, consumer string) bool {
	fields := parseEvalNote(c.GetNote())
	source := strings.TrimPrefix(fields["source"], "agent-manager://runs/")
	parts := strings.Split(source, "/events/")
	sequence, err := strconv.ParseInt(fields["seq"], 10, 64)
	if err != nil || sequence < 0 || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	expectedRequestID, err := searcheval.AgentManagerReplayRequestID(parts[0], parts[1], sequence, taskRoleForCase(c), consumer)
	return err == nil && fields["gateway_request"] == expectedRequestID && callerCaseLedgerMatches(fields, c, role, consumer)
}

func callerCaseLedgerMatches(fields map[string]string, c *evalv1.EvalCase, role, consumer string) bool {
	eventID := fields["gateway_event"]
	return fields["gateway_request"] != "" && fields["selected_resource_role"] == role && fields["model"] != "" &&
		fields["task_role"] == taskRoleForCase(c) &&
		fields["consumer"] == consumer &&
		fields["role_evidence"] == "gateway-ledger://events/"+eventID &&
		strings.HasPrefix(fields["capacity_estimate"], "capacity-footprint://") &&
		fields["label_review"] != "" && fields["reviewer"] != ""
}

func parseEvalNote(note string) map[string]string {
	fields := make(map[string]string)
	for _, raw := range strings.Fields(note) {
		key, value, ok := strings.Cut(raw, "=")
		if ok {
			fields[key] = value
		}
	}
	return fields
}

func taskRoleForCase(c *evalv1.EvalCase) string {
	if c == nil {
		return ""
	}
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "task-role:") {
			return strings.TrimPrefix(tag, "task-role:")
		}
	}
	return ""
}

func consumerForCase(c *evalv1.EvalCase) string {
	if c == nil {
		return ""
	}
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "consumer:") {
			return strings.TrimPrefix(tag, "consumer:")
		}
	}
	return ""
}

func evalCallerCaseID(note, role string) string {
	fields := parseEvalNote(note)
	source := strings.TrimPrefix(fields["source"], "agent-manager://runs/")
	parts := strings.Split(source, "/events/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || fields["gateway_event"] == "" {
		return ""
	}
	material := parts[0] + "/" + parts[1] + "/" + role + "/" + fields["gateway_event"]
	sum := sha256.Sum256([]byte(material))
	return "caller-" + hex.EncodeToString(sum[:12])
}

func roleForCase(c *evalv1.EvalCase) string {
	if c == nil {
		return ""
	}
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "resource-role:") {
			return strings.TrimPrefix(tag, "resource-role:")
		}
	}
	return ""
}

func hasEvalTag(c *evalv1.EvalCase, tag string) bool {
	for _, candidate := range c.GetTags() {
		if candidate == tag {
			return true
		}
	}
	return false
}
