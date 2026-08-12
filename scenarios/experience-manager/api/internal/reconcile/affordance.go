package reconcile

import (
	"fmt"
	"strings"

	"experience-manager/internal/spec"
)

func evaluateAffordancePresentClaim(page spec.PageDocument, claim spec.Claim, _ CaptureTarget, nodes []*AXNode) claimEvaluation {
	affordances := paramStringSlice(claim.Params, "affordances")
	if len(affordances) == 0 {
		affordance := paramString(claim.Params, "affordance")
		if affordance != "" {
			affordances = []string{affordance}
		}
	}
	if len(affordances) == 0 {
		return claimEvaluation{Unverifiable: "affordance-present requires params.affordances"}
	}
	targetRole := paramString(claim.Params, "targetRole")
	if targetRole == "" {
		targetRole = paramString(claim.Params, "role")
	}
	if targetRole == "" && len(claim.Elements) > 0 {
		targetRole = elementRole(page, claim.Elements[0])
	}
	if targetRole != "" && !hasTargetRole(page, claim, nodes, targetRole) {
		return claimEvaluation{Pass: false, Failure: fmt.Sprintf("target component role %q was not found in the accessibility snapshot", targetRole)}
	}
	var missing []string
	var axNodeJSON string
	for _, affordance := range affordances {
		affordance = strings.ToLower(strings.TrimSpace(affordance))
		if affordance == "" {
			continue
		}
		node := firstAffordanceNode(nodes, affordance)
		if node == nil {
			missing = append(missing, affordance)
			continue
		}
		axNodeJSON = encodeAXNode(node)
	}
	if len(missing) > 0 {
		return claimEvaluation{Pass: false, AXNodeJSON: axNodeJSON, Failure: "missing affordance controls: " + strings.Join(missing, ", ")}
	}
	return claimEvaluation{Pass: true, AXNodeJSON: axNodeJSON}
}

func hasTargetRole(page spec.PageDocument, claim spec.Claim, nodes []*AXNode, targetRole string) bool {
	targetRole = strings.ToLower(strings.TrimSpace(targetRole))
	if targetRole == "" {
		return true
	}
	for _, elementID := range claim.Elements {
		node := findBoundNode(nodes, page.Bindings.Elements[elementID], elementRole(page, elementID))
		if node != nil && strings.EqualFold(node.Role, targetRole) {
			return true
		}
	}
	for _, node := range nodes {
		if node != nil && strings.EqualFold(node.Role, targetRole) {
			return true
		}
	}
	return false
}

func firstAffordanceNode(nodes []*AXNode, affordance string) *AXNode {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if nodeMatchesAffordance(node, affordance) {
			return node
		}
	}
	return nil
}

var affordanceMatchers = map[string]func(*AXNode, string, string) bool{
	"search":        matchSearchAffordance,
	"sort":          matchSortAffordance,
	"filter":        matchFilterAffordance,
	"validate":      matchValidationAffordance,
	"validation":    matchValidationAffordance,
	"confirm":       matchConfirmationAffordance,
	"confirmation":  matchConfirmationAffordance,
	"retry":         matchRetryAffordance,
	"progress":      matchProgressAffordance,
	"refresh":       matchRefreshAffordance,
	"stale-refresh": matchRefreshAffordance,
	"action":        matchActionAffordance,
}

func nodeMatchesAffordance(node *AXNode, affordance string) bool {
	name := strings.ToLower(strings.TrimSpace(node.Name + " " + node.Description + " " + node.DOM.TestID))
	role := strings.ToLower(strings.TrimSpace(node.Role))
	if matcher := affordanceMatchers[affordance]; matcher != nil {
		return matcher(node, name, role)
	}
	return strings.Contains(name, affordance)
}

func matchSearchAffordance(_ *AXNode, name, role string) bool {
	return role == "searchbox" || strings.Contains(name, "search")
}

func matchSortAffordance(node *AXNode, name, role string) bool {
	if strings.Contains(name, "sort") || role == "columnheader" && containsState(node, "sortable") {
		return true
	}
	// Sort labels are localized, so their English wording cannot be the
	// contract. A table header carrying aria-sort is the platform semantic for
	// a sortable column and remains stable across translations.
	return role == "columnheader" && hasAttribute(node, "aria-sort")
}

func matchFilterAffordance(node *AXNode, name, role string) bool {
	return strings.Contains(name, "filter") ||
		(role == "combobox" || role == "listbox") && containsAny(name, "status", "category", "type", "severity", "owner") ||
		(role == "group" || role == "toolbar") && hasDescendant(node, hasPressedControl)
}

func hasAttribute(node *AXNode, key string) bool {
	if node == nil || node.DOM.Attributes == nil {
		return false
	}
	_, present := node.DOM.Attributes[key]
	return present
}

func hasPressedControl(node *AXNode) bool {
	if node == nil || strings.ToLower(strings.TrimSpace(node.Role)) != "button" {
		return false
	}
	if hasAttribute(node, "aria-pressed") {
		return true
	}
	return containsState(node, "pressed")
}

func hasDescendant(node *AXNode, predicate func(*AXNode) bool) bool {
	if node == nil {
		return false
	}
	for index := range node.Children {
		child := &node.Children[index]
		if predicate(child) || hasDescendant(child, predicate) {
			return true
		}
	}
	return false
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func matchValidationAffordance(_ *AXNode, name, role string) bool {
	return strings.Contains(name, "valid") || role == "alert"
}

func matchConfirmationAffordance(_ *AXNode, name, _ string) bool {
	return strings.Contains(name, "confirm") || strings.Contains(name, "cancel")
}

func matchRetryAffordance(_ *AXNode, name, _ string) bool {
	return strings.Contains(name, "retry") || strings.Contains(name, "try again")
}

func matchProgressAffordance(_ *AXNode, name, role string) bool {
	return role == "progressbar" || role == "meter" || strings.Contains(name, "progress")
}

func matchRefreshAffordance(_ *AXNode, name, _ string) bool {
	return strings.Contains(name, "refresh") || strings.Contains(name, "stale")
}

func matchActionAffordance(_ *AXNode, name, role string) bool {
	return (role == "button" || role == "link") && name != ""
}

func paramString(params map[string]any, key string) string {
	if len(params) == 0 {
		return ""
	}
	value, ok := params[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func paramStringSlice(params map[string]any, key string) []string {
	if len(params) == 0 {
		return nil
	}
	value, ok := params[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if item = strings.TrimSpace(item); item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if ok {
				text = strings.TrimSpace(text)
			}
			if ok && text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
