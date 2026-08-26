package credentialclient

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/resources/catalog"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

// ProjectScopeOwner is the owner label carried by a descriptor declared in the
// repository-root manifest. It matches the label the control-plane credential
// doctor already prints, so one address reads the same way on every surface.
const ProjectScopeOwner = "project"

// Scope names the manifest sources a caller wants. The zero value asks for
// every discovered scenario and resource and no project scope; callers state
// the project scope explicitly so a new consumer cannot silently inherit a
// narrower population than the control plane uses.
//
// A nil Scenarios or Resources slice means every discovered member. An empty
// non-nil slice means none, which is how a caller asks for the project scope
// alone.
type Scope struct {
	IncludeProject bool
	Scenarios      []string
	Resources      []string
}

// DescriptorsForScope returns the manifest-declared credential inventory for
// the requested scope. Declarations, rather than the authority, are the source
// of truth because a secure store intentionally does not enumerate secret
// values or identities.
//
// The repository root is itself a service manifest, and it is the authoritative
// owner of host-owned declarations that have no scenario directory. Project
// scope is therefore resolved through the same manifest parser as a scenario,
// and it is merged first so a project declaration owns its address.
func DescriptorsForScope(root string, scope Scope) ([]CredentialRef, error) {
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
			refs = append(refs, CredentialRef{Resource: resource, Env: descriptor.Env, LogicalID: descriptor.LogicalID, Field: descriptor.ResolvedField(), Label: firstNonEmpty(descriptor.Label, descriptor.Description), Description: descriptor.Description, Required: descriptor.Required})
		}
	}
	if scope.IncludeProject {
		projectManifestPath := filepath.Join(root, ".vrooli", "service.json")
		projectManifest, err := scenario.ReadService(projectManifestPath)
		switch {
		case err == nil:
			add(ProjectScopeOwner, projectManifest.Credentials)
		case os.IsNotExist(err):
			// A bundle catalog has no repository-root manifest by design.
		default:
			return nil, fmt.Errorf("read project service manifest: %w", err)
		}
	}
	if scope.Resources == nil || len(scope.Resources) > 0 {
		wanted := nameFilter(scope.Resources)
		names, err := catalog.New(root).ManifestNames()
		if err != nil {
			return nil, fmt.Errorf("discover resource manifests: %w", err)
		}
		for _, name := range names {
			if wanted != nil && !wanted[name] {
				continue
			}
			resourceManifest, loadErr := manifestpkg.Load(manifestpkg.DefaultPath(root, name))
			if loadErr == nil {
				add(resourceManifest.Name, resourceManifest.Credentials)
			}
		}
	}
	if scope.Scenarios == nil || len(scope.Scenarios) > 0 {
		wanted := nameFilter(scope.Scenarios)
		found, err := scenario.Discover(root, scenario.SandboxEnvFromEnv())
		if err != nil {
			return nil, fmt.Errorf("discover scenario manifests: %w", err)
		}
		for _, item := range found {
			if wanted != nil && !wanted[item.Slug] {
				continue
			}
			add(item.Slug, item.Manifest.Credentials)
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].LogicalID == refs[j].LogicalID {
			return refs[i].Field < refs[j].Field
		}
		return refs[i].LogicalID < refs[j].LogicalID
	})
	return refs, nil
}

// DiscoverDescriptors returns the whole repository-declared credential
// inventory, project scope included, which is the population the control-plane
// recovery inventory counts.
func DiscoverDescriptors(root string) ([]CredentialRef, error) {
	return DescriptorsForScope(root, Scope{IncludeProject: true})
}

// nameFilter returns nil for "every discovered member" and a lookup set
// otherwise. An empty non-nil selection never reaches here: the caller skips
// the whole discovery pass for it.
func nameFilter(names []string) map[string]bool {
	if names == nil {
		return nil
	}
	filter := make(map[string]bool, len(names))
	for _, name := range names {
		filter[strings.TrimSpace(name)] = true
	}
	return filter
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
