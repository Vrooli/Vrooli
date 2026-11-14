package corpusgen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// ModelPolicyRoles is the current Ollama policy role inventory. Corpus
// adequacy is evaluated only on independently reviewed, real caller cases.
var modelPolicyRoles = []string{
	"chat.default", "chat.small", "chat.uncensored", "classify.routing",
	"code.local", "embedding.default", "image.quality.judge",
	"rerank.llm_fallback", "summarize.default", "vision.default", "write.default",
}

func ModelPolicyRoles() []string {
	return append([]string(nil), modelPolicyRoles...)
}

func isModelPolicyRole(role string) bool {
	for _, allowed := range modelPolicyRoles {
		if role == allowed {
			return true
		}
	}
	return false
}

// ValidateModelRoleCoverage counts only reviewed Agent Manager caller cases
// carrying a role assignment and independent label review. Generated fixtures
// and pending candidates never satisfy the per-role floor.
func ValidateModelRoleCoverage(cases []*evalv1.EvalCase, minimumPerRole int) (map[string]int, error) {
	if minimumPerRole <= 0 {
		return nil, fmt.Errorf("minimum cases per role must be positive")
	}
	known := make(map[string]bool, len(modelPolicyRoles))
	counts := make(map[string]int, len(modelPolicyRoles))
	for _, role := range modelPolicyRoles {
		known[role] = true
		counts[role] = 0
	}
	seen := make(map[string]bool)
	for _, c := range cases {
		if c == nil || c.GetStatus() != "reviewed" || !hasCaseTag(c, "caller-audit") || !hasCaseTag(c, "agent-manager") || !hasCaseTag(c, "independently-reviewed") || hasCaseTag(c, "generated") || !hasReviewedCallerProvenance(c) {
			continue
		}
		role := ""
		for _, tag := range c.GetTags() {
			if strings.HasPrefix(tag, "role:") {
				role = strings.TrimPrefix(tag, "role:")
				break
			}
		}
		if !known[role] || role == "" || c.GetCaseId() == "" || len(c.GetExpectIds()) == 0 || seen[c.GetCaseId()] {
			continue
		}
		seen[c.GetCaseId()] = true
		counts[role]++
	}
	var deficient []string
	for _, role := range modelPolicyRoles {
		if counts[role] < minimumPerRole {
			deficient = append(deficient, fmt.Sprintf("%s=%d", role, counts[role]))
		}
	}
	if len(deficient) > 0 {
		sort.Strings(deficient)
		return counts, fmt.Errorf("reviewed caller corpus below %d cases per policy role: %s", minimumPerRole, strings.Join(deficient, ", "))
	}
	return counts, nil
}

// ValidateGatewayTaskRoleCoverage independently measures the task-role
// denominator declared by AI Gateway. Task roles and resource model-policy
// roles are separate contracts, even when their names overlap.
func ValidateGatewayTaskRoleCoverage(cases []*evalv1.EvalCase, taskRoles []string, minimumPerRole int) (map[string]int, error) {
	if minimumPerRole <= 0 {
		return nil, fmt.Errorf("minimum cases per task role must be positive")
	}
	known := make(map[string]bool, len(taskRoles))
	counts := make(map[string]int, len(taskRoles))
	for _, role := range taskRoles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		known[role] = true
		counts[role] = 0
	}
	seen := make(map[string]map[string]bool, len(known))
	for role := range known {
		seen[role] = map[string]bool{}
	}
	for _, c := range cases {
		if c == nil || c.GetStatus() != "reviewed" || !hasCaseTag(c, "caller-audit") || !hasCaseTag(c, "agent-manager") || !hasCaseTag(c, "independently-reviewed") || hasCaseTag(c, "generated") || !hasReviewedCallerProvenance(c) {
			continue
		}
		role := taskRoleForCase(c)
		if !known[role] || role == "" || c.GetCaseId() == "" || len(c.GetExpectIds()) == 0 || seen[role][c.GetCaseId()] {
			continue
		}
		seen[role][c.GetCaseId()] = true
		counts[role]++
	}
	var deficient []string
	for role, count := range counts {
		if count < minimumPerRole {
			deficient = append(deficient, fmt.Sprintf("%s=%d", role, count))
		}
	}
	if len(deficient) > 0 {
		sort.Strings(deficient)
		return counts, fmt.Errorf("reviewed caller corpus below %d cases per AI Gateway task role: %s", minimumPerRole, strings.Join(deficient, ", "))
	}
	return counts, nil
}

// hasReviewedCallerProvenance keeps structural or generated fixtures from
// satisfying the acceptance floor merely by copying caller-audit tags. Cases
// must retain the source, route, capacity, and independent-label references
// emitted by AuditedCallerCase.
func hasReviewedCallerProvenance(c *evalv1.EvalCase) bool {
	note := c.GetNote()
	role := ""
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "role:") {
			role = strings.TrimPrefix(tag, "role:")
			break
		}
	}
	if role == "" {
		return false
	}
	fields := make(map[string]string)
	for _, field := range strings.Fields(note) {
		key, value, ok := strings.Cut(field, "=")
		if ok {
			fields[key] = value
		}
	}
	source := strings.TrimPrefix(fields["source"], "agent-manager://runs/")
	sourceParts := strings.Split(source, "/events/")
	if len(sourceParts) != 2 || sourceParts[0] == "" || sourceParts[1] == "" {
		return false
	}
	_, err := strconv.ParseUint(fields["seq"], 10, 64)
	if err != nil {
		return false
	}
	gatewayEventID := fields["gateway_event"]
	if gatewayEventID == "" || fields["gateway_request"] == "" || fields["role"] != role || fields["model"] == "" ||
		fields["role_evidence"] != "gateway-ledger://events/"+gatewayEventID ||
		!strings.HasPrefix(fields["capacity_estimate"], "capacity-footprint://") ||
		fields["label_review"] == "" || fields["reviewer"] == "" {
		return false
	}
	if fields["task_role"] != taskRoleForCase(c) {
		return false
	}
	if taskRoleForCase(c) == "" {
		return false
	}
	return c.GetCaseId() == callerCaseID(sourceParts[0], sourceParts[1], role, gatewayEventID)
}

func taskRoleForCase(c *evalv1.EvalCase) string {
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "task-role:") {
			return strings.TrimSpace(strings.TrimPrefix(tag, "task-role:"))
		}
	}
	return ""
}

func hasCaseTag(c *evalv1.EvalCase, want string) bool {
	for _, tag := range c.GetTags() {
		if tag == want {
			return true
		}
	}
	return false
}
