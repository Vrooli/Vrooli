package reconcile

import (
	"fmt"
	"strings"

	"experience-manager/internal/spec"
)

func evaluateDifferentialClaim(page spec.PageDocument, claim spec.Claim, snapshots map[string]Snapshot) claimEvaluation {
	if strings.TrimSpace(claim.Subject) == "" || strings.TrimSpace(claim.Metric) == "" {
		return claimEvaluation{Unverifiable: "differential requires subject and metric"}
	}
	if claim.Require != "contexts-differ" || len(claim.Contexts) < 2 {
		return claimEvaluation{Unverifiable: "differential requires at least two contexts and require=contexts-differ"}
	}
	subjects := make([]MeasuredSubject, 0, len(claim.Contexts))
	values := make([]string, 0, len(claim.Contexts))
	for _, context := range claim.Contexts {
		snapshot, ok := snapshots[context.ID]
		if !ok && context.Story != "" {
			snapshot, ok = snapshots[context.Story]
		}
		if !ok {
			return claimEvaluation{Pass: false, Measurement: measurement(claim.Metric, "", "contexts-differ", nil, nil, subjects), Failure: fmt.Sprintf("differential context %q was not captured", context.ID)}
		}
		node := findBoundNode(snapshot.Flatten(), page.Bindings.Elements[claim.Subject], elementRole(page, claim.Subject))
		if node == nil {
			return claimEvaluation{Pass: false, Measurement: measurement(claim.Metric, "", "contexts-differ", nil, nil, subjects), Failure: fmt.Sprintf("differential subject %q was not captured in context %q", claim.Subject, context.ID)}
		}
		value, ok := differentialMetricValue(node, claim.Metric)
		if !ok {
			return claimEvaluation{Pass: false, Measurement: measurement(claim.Metric, "", "contexts-differ", nil, nil, subjects), Failure: fmt.Sprintf("differential metric %q was not captured in context %q", claim.Metric, context.ID)}
		}
		subjects = append(subjects, MeasuredSubject{ElementID: claim.Subject, TestID: node.DOM.TestID, Bounds: node.Bounds, ContextID: context.ID, Value: value})
		values = append(values, value)
		if context.Expect != "" && value != context.Expect {
			m := measurement(claim.Metric, "", "contexts-differ", nil, nil, subjects)
			return claimEvaluation{Pass: false, Measurement: m, Failure: fmt.Sprintf("differential context %q observed %q, expected %q", context.ID, value, context.Expect)}
		}
	}
	for index := 1; index < len(values); index++ {
		if values[index] != values[0] {
			m := measurement(claim.Metric, "", "contexts-differ", nil, nil, subjects)
			return claimEvaluation{Pass: true, Measurement: m}
		}
	}
	m := measurement(claim.Metric, "", "contexts-differ", nil, nil, subjects)
	return claimEvaluation{Pass: false, Measurement: m, Failure: "differential contexts-differ failed: captured values were equal"}
}

func differentialMetricValue(node *AXNode, metric string) (string, bool) {
	metric = strings.TrimSpace(metric)
	if node == nil || metric == "" {
		return "", false
	}
	keys := []string{metric, "data-" + metric, "data-rcl-" + metric, "aria-" + metric}
	for _, key := range keys {
		if value, ok := node.DOM.Attributes[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), true
		}
	}
	for _, state := range node.States {
		if strings.HasPrefix(state, metric+"=") {
			return strings.TrimPrefix(state, metric+"="), true
		}
	}
	if value := strings.TrimSpace(computedStyleValue(node, metric)); value != "" {
		return value, true
	}
	return "", false
}
