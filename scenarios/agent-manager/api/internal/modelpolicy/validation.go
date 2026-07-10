package modelpolicy

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

func (c *Catalog) Validate() error {
	if c == nil {
		return invalid("modelPolicyCatalog", "field is required")
	}
	if c.SchemaVersion != CurrentSchemaVersion {
		return invalid("modelPolicyCatalog.schemaVersion", fmt.Sprintf("must equal %d", CurrentSchemaVersion))
	}
	if err := validateMetadata(c.Metadata); err != nil {
		return err
	}
	if len(c.Runners) == 0 {
		return invalid("modelPolicyCatalog.runners", "must define at least one runner")
	}

	runnerKeys := make([]string, 0, len(c.Runners))
	for runnerType := range c.Runners {
		runnerKeys = append(runnerKeys, string(runnerType))
	}
	sort.Strings(runnerKeys)
	for _, key := range runnerKeys {
		runnerType := domain.RunnerType(key)
		if !runnerType.IsValid() {
			return invalid("modelPolicyCatalog.runners."+key, "unknown runner")
		}
		if err := validateInventory(runnerType, c.Runners[runnerType]); err != nil {
			return err
		}
	}

	if len(c.Policies) == 0 {
		return invalid("modelPolicyCatalog.policies", "must define at least one policy")
	}
	policyKeys := make([]string, 0, len(c.Policies))
	for name := range c.Policies {
		policyKeys = append(policyKeys, name)
	}
	sort.Strings(policyKeys)
	for _, name := range policyKeys {
		if strings.TrimSpace(name) != name || name == "" {
			return invalid("modelPolicyCatalog.policies", fmt.Sprintf("invalid policy name %q", name))
		}
		if err := c.validatePolicy(name, c.Policies[name]); err != nil {
			return err
		}
	}
	if strings.TrimSpace(c.DefaultPolicy) == "" {
		return invalid("modelPolicyCatalog.defaultPolicy", "field is required")
	}
	if _, ok := c.Policies[c.DefaultPolicy]; !ok {
		return invalid("modelPolicyCatalog.defaultPolicy", fmt.Sprintf("references unknown policy %q", c.DefaultPolicy))
	}
	return nil
}

func validateMetadata(metadata Metadata) error {
	if strings.TrimSpace(metadata.CatalogID) == "" {
		return invalid("modelPolicyCatalog.metadata.catalogId", "field is required")
	}
	if _, err := time.Parse(time.DateOnly, metadata.UpdatedAt); err != nil {
		return invalid("modelPolicyCatalog.metadata.updatedAt", "must use YYYY-MM-DD")
	}
	if len(metadata.Sources) == 0 {
		return invalid("modelPolicyCatalog.metadata.sources", "must record at least one freshness source")
	}
	for index, source := range metadata.Sources {
		prefix := fmt.Sprintf("modelPolicyCatalog.metadata.sources[%d]", index)
		if strings.TrimSpace(source.Name) == "" {
			return invalid(prefix+".name", "field is required")
		}
		if strings.TrimSpace(source.Reference) == "" {
			return invalid(prefix+".reference", "field is required")
		}
		if _, err := time.Parse(time.DateOnly, source.VerifiedAt); err != nil {
			return invalid(prefix+".verifiedAt", "must use YYYY-MM-DD")
		}
	}
	return nil
}

func validateInventory(runnerType domain.RunnerType, inventory Inventory) error {
	prefix := "modelPolicyCatalog.runners." + string(runnerType)
	if len(inventory.Models) == 0 {
		return invalid(prefix+".models", "must declare at least one static model")
	}
	prefixes := make(map[string]struct{}, len(inventory.DynamicModelPrefixes))
	for index, rawPrefix := range inventory.DynamicModelPrefixes {
		modelPrefix := strings.TrimSpace(rawPrefix)
		if modelPrefix == "" || modelPrefix != rawPrefix {
			return invalid(fmt.Sprintf("%s.dynamicModelPrefixes[%d]", prefix, index), "must be non-empty and whitespace-trimmed")
		}
		if _, exists := prefixes[modelPrefix]; exists {
			return invalid(prefix+".dynamicModelPrefixes", fmt.Sprintf("duplicate prefix %q", modelPrefix))
		}
		prefixes[modelPrefix] = struct{}{}
	}
	seen := make(map[string]struct{}, len(inventory.Models))
	for index, model := range inventory.Models {
		field := fmt.Sprintf("%s.models[%d]", prefix, index)
		id := strings.TrimSpace(model.ID)
		if id == "" || id != model.ID {
			return invalid(field+".id", "must be non-empty and whitespace-trimmed")
		}
		if strings.TrimSpace(model.Description) == "" {
			return invalid(field+".description", "field is required")
		}
		if _, exists := seen[id]; exists {
			return invalid(prefix+".models", fmt.Sprintf("duplicate model id %q", id))
		}
		for dynamicPrefix := range prefixes {
			if strings.HasPrefix(id, dynamicPrefix) {
				return invalid(field+".id", fmt.Sprintf("static model uses dynamic inventory prefix %q", dynamicPrefix))
			}
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (c *Catalog) validatePolicy(name string, policy Policy) error {
	prefix := "modelPolicyCatalog.policies." + name
	if !policy.Intent.IsValid() {
		return invalid(prefix+".intent", fmt.Sprintf("unknown intent %q", policy.Intent))
	}
	if len(policy.Candidates) == 0 {
		return invalid(prefix+".candidates", "must contain at least one candidate")
	}
	seen := make(map[string]struct{}, len(policy.Candidates))
	runnerDefaultSeen := make(map[domain.RunnerType]bool)
	for index, candidate := range policy.Candidates {
		field := fmt.Sprintf("%s.candidates[%d]", prefix, index)
		inventory, ok := c.Runners[candidate.Runner]
		if !ok {
			return invalid(field+".runner", fmt.Sprintf("references unknown runner %q", candidate.Runner))
		}
		if runnerDefaultSeen[candidate.Runner] {
			return invalid(field, fmt.Sprintf("unreachable candidate after runner_default for %q", candidate.Runner))
		}

		var identity string
		switch candidate.Selection.Type {
		case SelectionTypeModel:
			modelID := strings.TrimSpace(candidate.Selection.Model)
			if modelID == "" || modelID != candidate.Selection.Model {
				return invalid(field+".selection.model", "must be non-empty and whitespace-trimmed for model selection")
			}
			if !inventoryHasModel(inventory, modelID) {
				return invalid(field+".selection.model", fmt.Sprintf("references undeclared model %q for runner %q", modelID, candidate.Runner))
			}
			identity = string(candidate.Runner) + "|model|" + modelID
		case SelectionTypeRunnerDefault:
			if candidate.Selection.Model != "" {
				return invalid(field+".selection.model", "must be omitted for runner_default selection")
			}
			if !inventory.SupportsRunnerDefault {
				return invalid(field+".selection.type", fmt.Sprintf("runner %q does not support runner_default", candidate.Runner))
			}
			if policy.Intent == PolicyIntentCheap {
				return invalid(field+".selection.type", "cheap policies cannot select runner_default because its cost is unknown")
			}
			runnerDefaultSeen[candidate.Runner] = true
			identity = string(candidate.Runner) + "|runner_default"
		default:
			return invalid(field+".selection.type", fmt.Sprintf("unknown selection type %q", candidate.Selection.Type))
		}
		if _, exists := seen[identity]; exists {
			return invalid(field, fmt.Sprintf("duplicate candidate %q", identity))
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func inventoryHasModel(inventory Inventory, modelID string) bool {
	for _, model := range inventory.Models {
		if model.ID == modelID {
			return true
		}
	}
	return false
}

func invalid(field, message string) error {
	return domain.NewValidationError(field, message)
}
