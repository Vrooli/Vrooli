package rolepolicy

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

func (c *Catalog) Validate() error {
	if c == nil {
		return invalid("rolePolicyCatalog", "field is required")
	}
	if c.SchemaVersion != CurrentSchemaVersion {
		return invalid("rolePolicyCatalog.schemaVersion", fmt.Sprintf("must equal %d", CurrentSchemaVersion))
	}
	if strings.TrimSpace(c.Metadata.CatalogID) == "" {
		return invalid("rolePolicyCatalog.metadata.catalogId", "field is required")
	}
	if _, err := time.Parse(time.DateOnly, c.Metadata.UpdatedAt); err != nil {
		return invalid("rolePolicyCatalog.metadata.updatedAt", "must use YYYY-MM-DD")
	}
	if len(c.Roles) == 0 {
		return invalid("rolePolicyCatalog.roles", "must define at least one role")
	}
	keys := make([]string, 0, len(c.Roles))
	for key := range c.Roles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !validRoleKey(key) {
			return invalid("rolePolicyCatalog.roles", fmt.Sprintf("invalid role key %q", key))
		}
		if err := validateRole(key, c.Roles[key]); err != nil {
			return err
		}
	}
	if !validRoleKey(c.DefaultRole) {
		return invalid("rolePolicyCatalog.defaultRole", "must be a non-empty portable role key")
	}
	if _, ok := c.Roles[c.DefaultRole]; !ok {
		return invalid("rolePolicyCatalog.defaultRole", fmt.Sprintf("references unknown role %q", c.DefaultRole))
	}
	return nil
}

func validateRole(key string, role Role) error {
	prefix := "rolePolicyCatalog.roles." + key
	if strings.TrimSpace(role.Description) == "" {
		return invalid(prefix+".description", "field is required")
	}
	if strings.TrimSpace(role.Intent) == "" {
		return invalid(prefix+".intent", "field is required")
	}
	if len(role.Candidates) == 0 {
		return invalid(prefix+".candidates", "must contain at least one candidate")
	}
	seen := make(map[string]struct{}, len(role.Candidates))
	for index, candidate := range role.Candidates {
		field := fmt.Sprintf("%s.candidates[%d]", prefix, index)
		if !candidate.Runner.IsValid() {
			return invalid(field+".runner", fmt.Sprintf("unknown runner %q", candidate.Runner))
		}
		if !validRoleKey(candidate.ResourceRole) {
			return invalid(field+".resourceRole", "must be a non-empty portable role key")
		}
		identity := string(candidate.Runner) + "|" + candidate.ResourceRole
		if _, exists := seen[identity]; exists {
			return invalid(field, fmt.Sprintf("duplicate candidate %q", identity))
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func validRoleKey(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		if part == "" {
			return false
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

func invalid(field, message string) error {
	return domain.NewValidationError(field, message)
}
