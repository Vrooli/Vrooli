package credentialclient

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/resources/catalog"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

// DiscoverDescriptors returns the repository-declared credential inventory.
// Declarations, rather than the authority, are the source of truth because a
// secure store intentionally does not enumerate secret values or identities.
func DiscoverDescriptors(root string) ([]CredentialRef, error) {
	if strings.TrimSpace(root) == "" {
		return []CredentialRef{}, nil
	}
	refs := make([]CredentialRef, 0)
	seen := make(map[string]bool)
	add := func(resource string, declaration credentialspec.Declaration) {
		for _, descriptor := range declaration.All() {
			key := strings.TrimSpace(descriptor.LogicalID) + ":" + descriptor.ResolvedField()
			if seen[key] {
				continue
			}
			seen[key] = true
			refs = append(refs, CredentialRef{Resource: resource, Env: descriptor.Env, LogicalID: descriptor.LogicalID, Field: descriptor.ResolvedField(), Label: firstNonEmpty(descriptor.Label, descriptor.Description), Required: descriptor.Required})
		}
	}
	names, err := catalog.New(root).ManifestNames()
	if err != nil {
		return nil, fmt.Errorf("discover resource manifests: %w", err)
	}
	for _, name := range names {
		resourceManifest, loadErr := manifestpkg.Load(manifestpkg.DefaultPath(root, name))
		if loadErr == nil {
			add(resourceManifest.Name, resourceManifest.Credentials)
		}
	}
	found, err := scenario.Discover(root, scenario.SandboxEnvFromEnv())
	if err != nil {
		return nil, fmt.Errorf("discover scenario manifests: %w", err)
	}
	for _, item := range found {
		add(item.Slug, item.Manifest.Credentials)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].LogicalID == refs[j].LogicalID {
			return refs[i].Field < refs[j].Field
		}
		return refs[i].LogicalID < refs[j].LogicalID
	})
	return refs, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
