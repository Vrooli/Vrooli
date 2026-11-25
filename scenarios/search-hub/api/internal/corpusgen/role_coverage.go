package corpusgen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// ValidateModelRoleCoverage counts only reviewed Agent Manager caller cases
// carrying a role assignment and independent label review. Generated fixtures
// and pending candidates never satisfy the per-role floor. policyRoles must
// come from the authoritative model-policy owner; Search Hub does not maintain
// a second role inventory.
func ValidateModelRoleCoverage(cases []*evalv1.EvalCase, policyRoles []string, minimumPerRole int) (map[string]int, error) {
	if minimumPerRole <= 0 {
		return nil, fmt.Errorf("minimum cases per role must be positive")
	}
	roles, err := normalizeRoleInventory(policyRoles)
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(roles))
	counts := make(map[string]int, len(roles))
	for _, role := range roles {
		known[role] = true
		counts[role] = 0
	}
	seen := make(map[string]bool)
	for _, c := range cases {
		if c == nil || c.GetStatus() != "reviewed" || !hasCaseTag(c, "caller-audit") || !hasCaseTag(c, "agent-manager") || !hasCaseTag(c, "independently-reviewed") || hasCaseTag(c, "generated") || !hasReviewedCallerProvenance(c) {
			continue
		}
		role := resourceRoleForCase(c)
		if !known[role] || role == "" || c.GetCaseId() == "" || len(c.GetExpectIds()) == 0 || seen[c.GetCaseId()] {
			continue
		}
		seen[c.GetCaseId()] = true
		counts[role]++
	}
	var deficient []string
	for _, role := range roles {
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

func normalizeRoleInventory(input []string) ([]string, error) {
	roles := make([]string, 0, len(input))
	seen := make(map[string]bool, len(input))
	for _, raw := range input {
		role := strings.TrimSpace(raw)
		if role == "" || seen[role] {
			return nil, fmt.Errorf("authoritative model-policy role inventory contains an empty or duplicate role")
		}
		seen[role] = true
		roles = append(roles, role)
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("authoritative model-policy role inventory is empty")
	}
	sort.Strings(roles)
	return roles, nil
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
	role := resourceRoleForCase(c)
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
	consumer := consumerForCase(c)
	if gatewayEventID == "" || fields["gateway_request"] == "" || fields["selected_resource_role"] != role || fields["model"] == "" || consumer == "" || fields["consumer"] != consumer ||
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

func resourceRoleForCase(c *evalv1.EvalCase) string {
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "resource-role:") {
			return strings.TrimSpace(strings.TrimPrefix(tag, "resource-role:"))
		}
	}
	return ""
}

func taskRoleForCase(c *evalv1.EvalCase) string {
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "task-role:") {
			return strings.TrimSpace(strings.TrimPrefix(tag, "task-role:"))
		}
	}
	return ""
}

func consumerForCase(c *evalv1.EvalCase) string {
	for _, tag := range c.GetTags() {
		if strings.HasPrefix(tag, "consumer:") {
			return strings.TrimSpace(strings.TrimPrefix(tag, "consumer:"))
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
