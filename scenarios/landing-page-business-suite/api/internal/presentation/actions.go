package presentation

import (
	"bytes"
	"encoding/json"
	"strings"
)

// ActionStatus is a display observation. It is not a checkout or download
// authorization result; the owning service still performs that operation.
type ActionStatus string

const (
	ResolvedActionReady       ActionStatus = "ready"
	ResolvedActionUnavailable ActionStatus = "unavailable"
)

// ResolvedAction is the public, typed projection consumed by the renderer.
// The key is the stable UI action identity, while app/plan references remain
// visible for owner joins without exposing private owner payloads.
type ResolvedAction struct {
	Key     string       `json:"key"`
	Status  ActionStatus `json:"status"`
	Reason  string       `json:"reason,omitempty"`
	Href    string       `json:"href,omitempty"`
	AppKey  string       `json:"app_key,omitempty"`
	PlanRef string       `json:"plan_ref,omitempty"`
}

// ActionOwnerObservation is the small public result of an authoritative owner
// lookup. Owners provide an already-safe application/checkout href; this
// package never derives an installer URL or payment destination.
type ActionOwnerObservation struct {
	Ready  bool
	Href   string
	Reason string
}

type ActionOwnerObservations struct {
	Downloads map[string]ActionOwnerObservation
	Purchases map[string]ActionOwnerObservation
	Opens     map[string]ActionOwnerObservation
}

// ActionKey matches the public UI's JSON.stringify([kind, app, plan, target])
// identity. SetEscapeHTML(false) keeps Go's tuple encoding aligned with the
// browser for Unicode and otherwise-safe configured text.
func ActionKey(action Action) string {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode([]string{string(action.Kind), action.AppKey, action.PlanRef, action.Target}); err != nil {
		return "[\"\",\"\",\"\",\"\"]"
	}
	// Go always escapes these separators; JSON.stringify does not. Walk JSON
	// escape tokens so a literal backslash-u sequence remains untouched.
	value := strings.TrimSuffix(encoded.String(), "\n")
	var result strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			if strings.HasPrefix(value[i:], `\u2028`) || strings.HasPrefix(value[i:], `\u2029`) {
				if value[i+5] == '8' {
					result.WriteRune('\u2028')
				} else {
					result.WriteRune('\u2029')
				}
				i += 5
				continue
			}
			result.WriteByte(value[i])
			i++
		}
		result.WriteByte(value[i])
	}
	return result.String()
}

// ResolveActions projects every public header and block action in page order.
// Local navigation can be ready immediately; commerce and delivery actions
// remain unavailable until landing joins authoritative owner observations.
func ResolveActions(result ResolveResult) []ResolvedAction {
	configuredReason := strings.TrimSpace(result.Page.Display.Shell.UnavailableReason)
	if configuredReason == "" {
		configuredReason = "Unavailable"
	}
	configured := pageActions(result.Page)
	resolved := make([]ResolvedAction, 0, len(configured))
	for _, action := range configured {
		value := ResolvedAction{Key: ActionKey(action), Status: ResolvedActionUnavailable, Reason: configuredReason, AppKey: action.AppKey, PlanRef: action.PlanRef}
		switch action.Kind {
		case ActionAnchor, ActionRequestAccess:
			value.Status, value.Href, value.Reason = ResolvedActionReady, action.Target, ""
		case ActionOpen:
			if action.AppKey == "" {
				value.Status, value.Href, value.Reason = ResolvedActionReady, action.Target, ""
			}
		case ActionAppDetail:
			if href := CanonicalAppHref(result, action.AppKey); href != "" {
				value.Status, value.Href, value.Reason = ResolvedActionReady, href, ""
			}
		case ActionUnavailable:
			if strings.TrimSpace(action.Reason) != "" {
				value.Reason = action.Reason
			}
		}
		resolved = append(resolved, value)
	}
	return resolved
}

// JoinActionOwners replaces only owner-dependent action observations. A
// missing row, incomplete row, or owner outage leaves the page intact and
// produces a visible configured unavailable reason for that action.
func JoinActionOwners(result *ResolveResult, observations ActionOwnerObservations) {
	if result == nil {
		return
	}
	configured := pageActions(result.Page)
	result.Actions = ResolveActions(*result)
	reason := strings.TrimSpace(result.Page.Display.Shell.UnavailableReason)
	if reason == "" {
		reason = "Unavailable"
	}
	for index, action := range configured {
		if index >= len(result.Actions) {
			break
		}
		var observation ActionOwnerObservation
		var found bool
		switch action.Kind {
		case ActionDownload:
			observation, found = observations.Downloads[action.AppKey]
		case ActionPurchase:
			observation, found = observations.Purchases[action.PlanRef]
		case ActionOpen:
			if action.AppKey == "" {
				continue
			}
			observation, found = observations.Opens[action.AppKey]
		default:
			continue
		}
		if !found || !observation.Ready || !isSafeTarget(observation.Href) {
			result.Actions[index].Status = ResolvedActionUnavailable
			result.Actions[index].Href = ""
			if strings.TrimSpace(observation.Reason) != "" {
				result.Actions[index].Reason = observation.Reason
			} else {
				result.Actions[index].Reason = reason
			}
			continue
		}
		result.Actions[index].Status = ResolvedActionReady
		result.Actions[index].Href = observation.Href
		result.Actions[index].Reason = ""
	}
}

func pageActions(page ResolvedPage) []Action {
	result := make([]Action, 0)
	if action := page.Display.Shell.HeaderAction; action != nil {
		result = append(result, *action)
	}
	for _, block := range page.Blocks {
		switch content := block.Content.(type) {
		case ProductHeroContent:
			result = append(result, content.Actions...)
		case BundleHeroContent:
			result = append(result, content.Actions...)
		case PricingContent:
			result = append(result, content.Actions...)
		case ClosingActionContent:
			result = append(result, content.Actions...)
		}
	}
	// Header/hero/footer commonly repeat one CTA. The wire owner map has one
	// observation per stable identity, not one per visual placement.
	seen := make(map[string]bool, len(result))
	unique := make([]Action, 0, len(result))
	for _, action := range result {
		key := ActionKey(action)
		if !seen[key] {
			unique = append(unique, action)
			seen[key] = true
		}
	}
	return unique
}

func CanonicalAppHref(result ResolveResult, appKey string) string {
	for _, spotlight := range result.Spotlights {
		if spotlight.AppKey == appKey && strings.TrimSpace(spotlight.DetailRoute) != "" {
			return spotlight.DetailRoute
		}
	}
	return ""
}
