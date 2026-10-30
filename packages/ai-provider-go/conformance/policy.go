// Package conformance provides reusable checks for AI provider resources.
package conformance

import (
	"fmt"
	"strings"

	"github.com/vrooli/ai-provider-go/policy"
)

// CheckPolicyFile loads a provider catalog and resolves every declared role and
// model. It is the minimum contract every AI provider resource must satisfy.
func CheckPolicyFile(path string, opts policy.ValidationOptions) error {
	catalog, err := policy.LoadFile(path, opts)
	if err != nil {
		return fmt.Errorf("load policy: %w", err)
	}
	return CheckCatalog(catalog, opts)
}

// CheckCatalog validates a loaded catalog and resolves its full inventory.
func CheckCatalog(catalog policy.Catalog, opts policy.ValidationOptions) error {
	if err := policy.Validate(catalog, opts); err != nil {
		return fmt.Errorf("validate policy: %w", err)
	}
	for _, role := range catalog.RoleNames() {
		resolved, err := policy.ResolveRole(catalog, role)
		if err != nil {
			return fmt.Errorf("resolve role %q: %w", role, err)
		}
		if strings.TrimSpace(resolved.Model) == "" {
			return fmt.Errorf("resolve role %q: empty model", role)
		}
		if len(resolved.Capabilities) == 0 {
			return fmt.Errorf("resolve role %q: model %q has no capabilities", role, resolved.Model)
		}
		if resolved.RoleProvenance == nil {
			return fmt.Errorf("resolve role %q: role provenance missing", role)
		}
	}
	for _, model := range catalog.ModelRefs() {
		if _, err := policy.ResolveModel(catalog, model); err != nil {
			return fmt.Errorf("resolve model %q: %w", model, err)
		}
	}
	return nil
}
