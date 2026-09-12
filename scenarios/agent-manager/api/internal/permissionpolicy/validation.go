package permissionpolicy

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

var ruleID = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

func (c *Catalog) Validate() error {
	if c == nil {
		return invalid("permissionPolicyCatalog", "field is required")
	}
	if c.SchemaVersion != CurrentSchemaVersion {
		return invalid("permissionPolicyCatalog.schemaVersion", fmt.Sprintf("must equal %d", CurrentSchemaVersion))
	}
	if strings.TrimSpace(c.Metadata.CatalogID) == "" {
		return invalid("permissionPolicyCatalog.metadata.catalogId", "field is required")
	}
	if _, err := time.Parse(time.DateOnly, c.Metadata.UpdatedAt); err != nil {
		return invalid("permissionPolicyCatalog.metadata.updatedAt", "must use YYYY-MM-DD")
	}
	if len(c.TargetScopes) == 0 {
		return invalid("permissionPolicyCatalog.targetScopes", "must declare at least one scope")
	}
	targetScopes := make(map[string]struct{}, len(c.TargetScopes))
	for index, scope := range c.TargetScopes {
		field := fmt.Sprintf("permissionPolicyCatalog.targetScopes[%d]", index)
		if !validScope(scope) {
			return invalid(field, "must be user or admin")
		}
		if _, exists := targetScopes[scope]; exists {
			return invalid(field, fmt.Sprintf("duplicates scope %q", scope))
		}
		targetScopes[scope] = struct{}{}
	}

	ids := make(map[string]struct{}, len(c.Rules))
	matchers := make(map[string]struct{}, len(c.Rules))
	for index, rule := range c.Rules {
		prefix := fmt.Sprintf("permissionPolicyCatalog.rules[%d]", index)
		if !ruleID.MatchString(rule.ID) {
			return invalid(prefix+".id", "must be a stable lowercase rule ID")
		}
		if _, exists := ids[rule.ID]; exists {
			return invalid(prefix+".id", fmt.Sprintf("duplicates rule %q", rule.ID))
		}
		ids[rule.ID] = struct{}{}
		if strings.TrimSpace(rule.Rationale) == "" {
			return invalid(prefix+".rationale", "field is required")
		}
		if strings.TrimSpace(rule.Owner) == "" {
			return invalid(prefix+".owner", "field is required")
		}
		if !validScope(rule.TargetScope) {
			return invalid(prefix+".targetScope", "must be user or admin")
		}
		if _, exists := targetScopes[rule.TargetScope]; !exists {
			return invalid(prefix+".targetScope", "must be declared in targetScopes")
		}
		if rule.Action != "allow" && rule.Action != "ask" && rule.Action != "deny" {
			return invalid(prefix+".action", "must be allow, ask, or deny")
		}
		if rule.Matcher.Kind != "bash" {
			return invalid(prefix+".matcher.kind", "only bash is portable")
		}
		if strings.TrimSpace(rule.Matcher.Pattern) == "" {
			return invalid(prefix+".matcher.pattern", "field is required")
		}
		identity := rule.TargetScope + "\x00" + rule.Matcher.Kind + "\x00" + rule.Matcher.Pattern
		if _, exists := matchers[identity]; exists {
			return invalid(prefix+".matcher", "duplicates another matcher in the same target scope")
		}
		matchers[identity] = struct{}{}
	}
	return nil
}

func validScope(scope string) bool {
	return scope == "user" || scope == "admin"
}

func invalid(field, message string) error {
	return domain.NewValidationError(field, message)
}
