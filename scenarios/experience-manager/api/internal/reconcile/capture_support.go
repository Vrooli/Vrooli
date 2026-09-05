package reconcile

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"experience-manager/internal/spec"
)

func hasMachineClaim(page spec.PageDocument) bool {
	for _, claim := range page.Claims {
		if claim.Tier != "machine" || len(claim.Locales) > 0 || len(claim.Extensions) > 0 {
			continue
		}
		return true
	}
	return false
}

func pageHasMachineClaimForTarget(page spec.PageDocument, target CaptureTarget) bool {
	for _, claim := range page.Claims {
		if applies, _ := claimAppliesToTarget(claim, target); claim.Tier == "machine" && applies {
			return true
		}
	}
	return false
}

func activeMachineElementIDs(page spec.PageDocument, target CaptureTarget) map[string]bool {
	out := map[string]bool{}
	for _, claim := range page.Claims {
		if applies, _ := claimAppliesToTarget(claim, target); claim.Tier != "machine" || !applies {
			continue
		}
		// An element-absent claim is a negative assertion. Its expected
		// evidence is that no matching node exists, so treating it as an
		// active binding would incorrectly produce bindings_unjoined before
		// the claim evaluator can prove the absence.
		if claim.Type == "element-absent" {
			continue
		}
		for _, elementID := range claim.Elements {
			out[elementID] = true
		}
	}
	return out
}

func claimAppliesToTarget(claim spec.Claim, target CaptureTarget) (bool, string) {
	if len(claim.Locales) > 0 {
		return false, "locale-scoped claims require locale capture support"
	}
	if len(claim.Extensions) > 0 {
		return false, "extension-scoped claims require a deterministic extension checker"
	}
	if !claimTargetsState(claim, target.StateID) {
		return false, fmt.Sprintf("claim states %v are outside captured state %q", claim.States, target.StateID)
	}
	if !claimTargetsViewport(claim, target.ViewportID, target.ViewportAliases) {
		return false, fmt.Sprintf("claim viewports %v are outside captured viewport %q", claim.Viewports, target.ViewportID)
	}
	return true, ""
}

func claimTargetsState(claim spec.Claim, stateID string) bool {
	if stateID == "" {
		stateID = "default"
	}
	if len(claim.States) == 0 {
		return true
	}
	if stateID != "default" {
		for _, state := range claim.States {
			if state == stateID {
				return true
			}
		}
		return false
	}
	for _, state := range claim.States {
		if state == "" || state == "default" {
			return true
		}
	}
	return false
}

func claimTargetsViewport(claim spec.Claim, viewportID string, aliases []string) bool {
	if len(claim.Viewports) == 0 {
		return true
	}
	for _, viewport := range claim.Viewports {
		if viewport == viewportID {
			return true
		}
		for _, alias := range aliases {
			if viewport == alias {
				return true
			}
		}
	}
	return false
}

func unverifiableOutOfMatrixFindings(loc string, page spec.PageDocument, profiles []CaptureProfile) []spec.Finding {
	var findings []spec.Finding
	captured := map[string]bool{}
	for _, profile := range profiles {
		captured[profile.ID] = true
		for _, alias := range profile.Aliases {
			captured[alias] = true
		}
	}
	for _, claim := range page.Claims {
		if claim.Tier != "machine" || len(claim.Viewports) == 0 || isBaselineFloorType(claim.Type) {
			continue
		}
		var missing []string
		for _, viewport := range claim.Viewports {
			if !captured[viewport] {
				missing = append(missing, viewport)
			}
		}
		if len(missing) == 0 {
			continue
		}
		findings = append(findings, spec.Finding{
			Code:       spec.CodeClaimUnverifiable,
			Severity:   spec.SeverityWarning,
			Message:    fmt.Sprintf("machine claim %q targets uncaptured viewports %v", claim.ID, missing),
			Locations:  []string{loc},
			Suggestion: "Add the viewport to the capture matrix, retier the claim, or remove the viewport scope if it should apply everywhere.",
		})
	}
	return findings
}

func unverifiableStateSetupFindings(loc string, page spec.PageDocument) []spec.Finding {
	setups := map[string]bool{"default": true}
	for _, state := range page.States {
		if state.ID != "" && hasStateSetup(state) {
			setups[state.ID] = true
		}
	}
	var findings []spec.Finding
	for _, claim := range page.Claims {
		if claim.Tier != "machine" || len(claim.States) == 0 || isBaselineFloorType(claim.Type) {
			continue
		}
		if claimTargetsState(claim, "default") {
			continue
		}
		var missing []string
		for _, state := range claim.States {
			stateID := state
			if stateID == "" {
				stateID = "default"
			}
			if stateID != "default" && !setups[stateID] {
				missing = append(missing, stateID)
			}
		}
		if len(missing) == 0 {
			continue
		}
		findings = append(findings, spec.Finding{
			Code:       spec.CodeClaimUnverifiable,
			Severity:   spec.SeverityWarning,
			Message:    fmt.Sprintf("machine claim %q targets states without deterministic setup %v", claim.ID, missing),
			Locations:  []string{loc},
			Suggestion: "Add states[].setup for those states, retier the claim, or keep the claim on captured states only.",
		})
	}
	return findings
}

func pageStatuses(refs []spec.DocumentRef) map[string]string {
	out := map[string]string{}
	for _, ref := range refs {
		out[ref.ID] = ref.Status
	}
	return out
}

func firstRoute(routes []string) string {
	if len(routes) == 0 || strings.TrimSpace(routes[0]) == "" {
		return "/"
	}
	return routes[0]
}

func componentHarnessRoute(scenario string, component spec.ComponentDocument, version, example string) string {
	catalogID := componentCatalogID(scenario, component)
	route := "/preview/" + url.PathEscape(catalogID) + "/harness.html"
	query := url.Values{}
	if strings.TrimSpace(version) != "" {
		query.Set("version", version)
	}
	if strings.TrimSpace(example) != "" {
		// RCL's harness contract calls the selected story "story". Keep the
		// provider-generated route identical to the public preview route so
		// readiness and accessibility capture observe the requested specimen.
		query.Set("story", example)
	}
	if encoded := query.Encode(); encoded != "" {
		route += "?" + encoded
	}
	return route
}

func componentCatalogID(scenario string, component spec.ComponentDocument) string {
	refParts := strings.Split(filepath.ToSlash(componentCatalogRef(component)), "/")
	namespace := strings.TrimSpace(scenario)
	for i := 1; i < len(refParts); i++ {
		if refParts[i] != "library" {
			continue
		}
		candidate := strings.TrimSpace(refParts[i-1])
		if candidate != "" && candidate != "." && candidate != ".." {
			namespace = candidate
			break
		}
	}
	for i := 0; i+1 < len(refParts); i++ {
		if refParts[i] == "components" && refParts[i+1] != "" {
			return namespace + ":" + refParts[i+1]
		}
	}
	if strings.TrimSpace(component.Component.Title) != "" {
		return namespace + ":" + strings.TrimSpace(component.Component.Title)
	}
	return namespace + ":" + component.Component.ID
}

func componentVersion(examplesRef string) string {
	parts := strings.Split(filepath.ToSlash(examplesRef), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "versions" {
			return parts[i+1]
		}
	}
	return ""
}

func componentCatalogRef(component spec.ComponentDocument) string {
	if strings.TrimSpace(component.Component.StoryRef) != "" {
		return component.Component.StoryRef
	}
	return component.Component.ExamplesRef
}

func elementRole(page spec.PageDocument, elementID string) string {
	for _, el := range page.Elements {
		if el.ID == elementID {
			return el.Role
		}
	}
	return ""
}

func findBoundIndex(nodes []*AXNode, binding spec.Binding, role string) int {
	for i, node := range nodes {
		if nodeMatches(node, binding, role) {
			return i
		}
	}
	if fallback := findTextSlotNode(nodes, binding, role); fallback != nil {
		for i, node := range nodes {
			if node == fallback {
				return i
			}
		}
	}
	return -1
}

func findBoundNode(nodes []*AXNode, binding spec.Binding, role string) *AXNode {
	for _, node := range nodes {
		if nodeMatches(node, binding, role) {
			return node
		}
	}
	return findTextSlotNode(nodes, binding, role)
}

// BAS accessibility snapshots expose visible inline label text as a
// StaticText node and do not carry the source span's data-testid onto that
// node. A declared x-label binding still has a deterministic structural
// target: the first StaticText descendant of the interactive control in the
// same specimen. Keep this fallback narrow to the conventional label slot so
// arbitrary missing bindings remain honest failures.
func findTextSlotNode(nodes []*AXNode, binding spec.Binding, role string) *AXNode {
	if role != "x-label" || !strings.HasSuffix(strings.TrimSpace(binding.TestID), "-label") {
		return nil
	}
	var textDescendant func(node *AXNode) *AXNode
	textDescendant = func(node *AXNode) *AXNode {
		if node == nil {
			return nil
		}
		if isTextOnlyNode(node) && node.Bounds != nil {
			return node
		}
		for index := range node.Children {
			if found := textDescendant(&node.Children[index]); found != nil {
				return found
			}
		}
		return nil
	}
	for _, node := range nodes {
		if isInteractiveNode(node) {
			if found := textDescendant(node); found != nil {
				return found
			}
		}
	}
	return nil
}

func nodeMatches(node *AXNode, binding spec.Binding, role string) bool {
	if node == nil {
		return false
	}
	if binding.TestID != "" && node.DOM.TestID != binding.TestID {
		return false
	}
	if binding.Selector != "" && !selectorMatches(node, binding.Selector) {
		return false
	}
	if role != "" && !strings.HasPrefix(role, "x-") && node.Role != role {
		return false
	}
	return binding.TestID != "" || binding.Selector != ""
}

func selectorMatches(node *AXNode, selector string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return false
	}
	if strings.Contains(selector, "[role='") || strings.Contains(selector, "[role=\"") {
		return selectorContains(selector, "role", node.Role)
	}
	if strings.Contains(selector, "data-testid^=") {
		prefix := selectorValue(selector)
		return prefix != "" && strings.HasPrefix(node.DOM.TestID, prefix)
	}
	if strings.Contains(selector, "data-testid=") {
		return selectorValue(selector) == node.DOM.TestID
	}
	if strings.Contains(selector, "aria-label=") {
		return ariaLabelValue(selector) == node.Name
	}
	if attribute, value, ok := attributeSelector(selector); ok {
		actual, present := node.DOM.Attributes[attribute]
		if value == "" {
			return present
		}
		return present && actual == value
	}
	if !strings.ContainsAny(selector, "#.[] >:+~") {
		return strings.EqualFold(node.DOM.Tag, selector) || strings.EqualFold(node.Role, selector)
	}
	return false
}

func attributeSelector(selector string) (string, string, bool) {
	start := strings.Index(selector, "[")
	if start < 0 {
		return "", "", false
	}
	end := strings.Index(selector[start+1:], "]")
	if end < 0 {
		return "", "", false
	}
	content := strings.TrimSpace(selector[start+1 : start+1+end])
	parts := strings.SplitN(content, "=", 2)
	if len(parts) == 1 {
		attribute := strings.TrimSpace(parts[0])
		return attribute, "", attribute != ""
	}
	attribute := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
	if attribute == "" || value == "" {
		return "", "", false
	}
	return attribute, value, true
}

func ariaLabelValue(selector string) string {
	idx := strings.Index(selector, "aria-label=")
	if idx < 0 {
		return ""
	}
	value := strings.TrimLeft(selector[idx+len("aria-label="):], `'")`)
	end := strings.IndexAny(value, `'"]`)
	if end >= 0 {
		value = value[:end]
	}
	return value
}

func selectorContains(selector, attr, value string) bool {
	return strings.Contains(selector, attr+"='"+value+"'") || strings.Contains(selector, attr+"=\""+value+"\"")
}

func selectorValue(selector string) string {
	for _, token := range []string{"data-testid^=", "data-testid="} {
		idx := strings.Index(selector, token)
		if idx < 0 {
			continue
		}
		value := strings.TrimLeft(selector[idx+len(token):], `'"`)
		end := strings.IndexAny(value, `'" ]`)
		if end >= 0 {
			value = value[:end]
		}
		return value
	}
	return ""
}

func sortFindings(findings []spec.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return first(findings[i].Locations) < first(findings[j].Locations)
	})
}
