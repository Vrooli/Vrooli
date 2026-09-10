package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
)

// applyPlanInput is the single input to planning. The review endpoint and the
// executor both consume the result of buildApplyPlan; neither surface derives
// its own list of host mutations.
type applyPlanInput struct {
	Closure      closureResult
	Requirements hostRequirementsResponse
	State        OperatorState
	// Observed maps an apply item ID to what this host was measured to be in
	// before the plan is shown. The operator asked which items were already
	// applied; without this the plan is a desired-state list that reads as if
	// every entry were a pending change. buildApplyPlan stays pure: the caller
	// does the measuring and passes the result in.
	Observed map[string]string
}

func observedState(observed map[string]string, id string) string {
	if state, ok := observed[id]; ok && state != "" {
		return state
	}
	return applyStateUnknown
}

func plannedRequirementState(status string, observed map[string]string, id string) string {
	if status == "not_applicable" {
		return "not_applicable"
	}
	return observedState(observed, id)
}

// observeApplyStates measures the host for the item kinds that can be checked
// without side effects. Tools resolve through PATH; safeguards resolve through
// their declared verification files, or, when they are handler-owned, through
// the control plane's read-only inspection boundary. This is the same
// inspection the readiness endpoint performs.
//
// Resources and scenarios are reported as unknown. That is a cost verdict, not
// a capability one: sampling them means a control-plane round trip per item
// (`vrooli resource status` alone takes tens of seconds), which is too slow to
// run before every plan render. They are the only remaining unknowns by design.
func observeApplyStates(root string, requirements hostRequirementsResponse) map[string]string {
	observed := map[string]string{}
	for _, tool := range requirements.Tools {
		observed["tool:"+tool.Name] = applyStateFromReadiness(inspectToolReadiness(tool).Status)
	}
	for _, safeguard := range requirements.Safeguards {
		observed["safeguard:"+safeguard.Name] = applyStateFromReadiness(inspectSafeguardReadiness(root, safeguard).Status)
	}
	return observed
}

func applyStateFromReadiness(status string) string {
	switch status {
	case "ready":
		return applyStateSatisfied
	case "missing":
		return applyStatePending
	case "degraded":
		// Configured but not yet in effect (a safeguard awaiting a reboot).
		// "Already in place" would overstate it: the operator still has
		// something to do before the host is actually protected.
		return applyStatePending
	default:
		// "deferred" and "unsupported" both mean this process did not reach a
		// verdict about applying, so neither may be presented as a fact.
		return applyStateUnknown
	}
}

func buildApplyPlan(input applyPlanInput) []applyItem {
	items := make([]applyItem, 0, len(input.Requirements.Tools)+len(input.Requirements.Safeguards)+len(input.Closure.Resources)+len(input.Closure.Scenarios))
	for _, item := range input.Requirements.Tools {
		if item.Status != "required" && item.Status != "opted_in" && item.Status != "not_applicable" {
			continue
		}
		items = append(items, applyItem{ID: "tool:" + item.Name, Kind: "tool", Name: item.Name, Required: item.Required, Privileged: item.Privilege == "elevated", State: plannedRequirementState(item.Status, input.Observed, "tool:"+item.Name)})
	}
	for _, item := range input.Requirements.Safeguards {
		if item.Status != "required" && item.Status != "opted_in" && item.Status != "not_applicable" {
			continue
		}
		items = append(items, applyItem{ID: "safeguard:" + item.Name, Kind: "safeguard", Name: item.Name, Required: item.Required, Privileged: item.Privilege == "elevated", State: plannedRequirementState(item.Status, input.Observed, "safeguard:"+item.Name)})
	}
	for _, member := range input.Closure.Resources {
		if choice, ok := input.State.Resources[member.Name]; ok && choice.Enabled != nil && !*choice.Enabled && !member.Required {
			continue
		}
		items = append(items, applyItem{ID: "resource:" + member.Name, Kind: "resource", Name: member.Name, Required: member.Required, State: observedState(input.Observed, "resource:"+member.Name)})
	}
	for _, member := range input.Closure.Scenarios {
		dependencies := make([]string, 0)
		for _, resource := range input.Closure.Resources {
			for _, provenance := range resource.Provenance {
				if provenance.From == member.Name {
					dependencies = append(dependencies, "resource:"+resource.Name)
					break
				}
			}
		}
		items = append(items, applyItem{ID: "scenario:" + member.Name, Kind: "scenario", Name: member.Name, Dependencies: dependencies, Required: member.Required || member.Direct, State: observedState(input.Observed, "scenario:"+member.Name)})
	}
	return items
}

func (s *Server) buildCurrentApplyPlan(ctx context.Context, target string) (applydomain.Plan, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		target = "local"
	}
	root, err := manifestRoot()
	if err != nil {
		return applydomain.Plan{}, err
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		return applydomain.Plan{}, err
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return applydomain.Plan{}, err
	}
	closure, err := resolveClosureForState(root, models, state)
	if err != nil {
		return applydomain.Plan{}, fmt.Errorf("resolve apply closure: %w", err)
	}
	hostModels, err := hostRequirementScenarioModels(root, models, state)
	if err != nil {
		return applydomain.Plan{}, fmt.Errorf("resolve host requirements: %w", err)
	}
	requirements, err := deriveV2HostRequirements(root, state, hostModels)
	if err != nil {
		return applydomain.Plan{}, err
	}
	items := buildApplyPlan(applyPlanInput{Closure: closure, Requirements: requirements, State: state, Observed: observeApplyStates(root, requirements)})
	catalogDigest, err := catalogIdentity(root, models)
	if err != nil {
		return applydomain.Plan{}, fmt.Errorf("identify apply catalog: %w", err)
	}
	digest := planDigest(target, state, items, catalogDigest)
	revision := strings.TrimSpace(state.UpdatedAt)
	if revision == "" {
		revision = strings.TrimSpace(state.Version)
	}
	plan := applydomain.Plan{
		Target: target, PlanID: "plan-" + digest[:16], Digest: digest, Revision: revision,
		ExpiresAt: operatorStateNow().Add(15 * time.Minute).UTC().Format(time.RFC3339),
		Items:     make([]applydomain.Item, 0, len(items)),
	}
	for _, item := range items {
		plan.Items = append(plan.Items, applydomain.Item{ID: item.ID, Kind: item.Kind, Name: item.Name, Dependencies: item.Dependencies, Required: item.Required, Privileged: item.Privileged, ObservedState: item.State})
	}
	return plan, nil
}

func (s *Server) getApplyPlan(ctx context.Context, target string) (applydomain.Plan, error) {
	return s.buildCurrentApplyPlan(ctx, target)
}

// planDigest binds every non-secret choice that can change an apply effect.
// Volatile observations and completion/session pointers are intentionally
// excluded; the plan carries those as separate, fresh metadata.
func planDigest(target string, state OperatorState, items []applyItem, catalogDigest ...string) string {
	// Schema is a storage annotation, not an operator choice. Older greenfield
	// state files may omit it while the authority adds it on the first write;
	// treating that migration as a changed apply effect would invalidate a
	// review without changing anything the host will do.
	state.Schema = ""
	state.UpdatedAt = ""
	state.Completion = nil
	state.Session = nil
	stateJSON, _ := json.Marshal(state)
	var semantic map[string]json.RawMessage
	_ = json.Unmarshal(stateJSON, &semantic)
	for key, value := range state.RawFields {
		semantic[key] = value
	}
	semanticItems := make([]applyItem, len(items))
	copy(semanticItems, items)
	for index := range semanticItems {
		// Host observations are disclosure metadata. They are intentionally
		// excluded from consent identity so a machine changing between review
		// and admission cannot invalidate or silently replace the reviewed
		// effects.
		semanticItems[index].State = ""
	}
	canonical, _ := json.Marshal(struct {
		Target        string                     `json:"target"`
		CatalogDigest string                     `json:"catalog_digest,omitempty"`
		State         map[string]json.RawMessage `json:"state"`
		Items         []applyItem                `json:"items"`
	}{Target: strings.TrimSpace(target), CatalogDigest: firstString(catalogDigest), State: semantic, Items: semanticItems})
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:])
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// catalogIdentity binds the exact catalog inputs used to derive the plan. It
// deliberately excludes the repository's executable source tree: a code edit
// unrelated to onboarding must not invalidate an operator's reviewed effects.
func catalogIdentity(root string, models []ScenarioReadModel) (string, error) {
	paths := []string{filepath.Join(root, ".vrooli", "service.json")}
	for _, model := range models {
		paths = append(paths, filepath.Join(root, "scenarios", model.Name, ".vrooli", "service.json"))
		for _, resource := range model.Resources {
			paths = append(paths, filepath.Join(root, "resources", resource, "resource.json"))
		}
	}
	for _, kind := range []string{"tools", "safeguards"} {
		entries, err := os.ReadDir(filepath.Join(root, "internal", kind))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				paths = append(paths, filepath.Join(root, "internal", kind, entry.Name(), strings.TrimSuffix(kind, "s")+".json"))
			}
		}
	}
	sort.Strings(paths)
	hash := sha256.New()
	seen := map[string]struct{}{}
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, _ = fmt.Fprintf(hash, "%s\x00", filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))))
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
