package credentialclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialinventory"
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
	// IncludeManaged adds authority-owned references that have no manifest
	// declaration, such as the release-authority key and device-control
	// identities. Callers opt in explicitly so a selected workload does not
	// inherit unrelated host recovery entries by accident.
	IncludeManaged bool
	Scenarios      []string
	Resources      []string
	// Tier narrows the declaration population to descriptors explicitly
	// applicable to one deployment tier. Empty preserves the historical
	// all-tier inventory.
	Tier string
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
	byAddress := make(map[string]int)
	add := func(resource, sourceRef string, declaration credentialspec.Declaration) error {
		for _, descriptor := range declaration.All() {
			if scope.Tier != "" && !tierApplies(descriptor, scope.Tier) {
				continue
			}
			key := strings.TrimSpace(descriptor.LogicalID) + ":" + descriptor.ResolvedField()
			consumerRefs := appendUniqueStrings(append([]string(nil), descriptor.ConsumerRefs...), declaredConsumerNames(declaration.Consumers, descriptor, scope.Tier)...)
			consumerProvenance := declaredConsumerProvenance(declaration.Consumers, descriptor, sourceRef, scope.Tier)
			owner := firstNonEmpty(descriptor.Owner, resource)
			migrationDiagnostics := descriptorMigrationDiagnostics(descriptor)
			provenance := CredentialProvenance{
				Version: descriptor.Version, Owner: owner, SourceRef: sourceRef, Kind: descriptorKind(descriptor), Provider: descriptor.Provider, AppliesWhen: descriptor.AppliesWhen,
				Tiers:            append([]string(nil), descriptor.Tiers...),
				RequirementGroup: descriptor.RequirementGroup, ConsumerRefs: append([]string(nil), consumerRefs...), CompanionSettings: append([]string(nil), descriptor.CompanionSettings...), CompanionCredentials: append([]string(nil), descriptor.CompanionCredentials...),
				AcquisitionRef: descriptor.AcquisitionRef, VerificationRef: descriptor.VerificationRef, RecoveryRef: descriptor.RecoveryRef, HelpRef: descriptor.HelpRef, EvidencePolicy: descriptor.EvidencePolicy, ProviderVersion: descriptor.ProviderVersion,
				Env:   descriptor.Env,
				Label: descriptor.Label, Description: descriptor.Description, ObtainURL: descriptor.ObtainURL,
				Provisioning: descriptor.Provisioning, DerivedFrom: descriptor.DerivedFrom, Required: descriptor.Required,
				Consumers: consumerProvenance,
			}
			if index, ok := byAddress[key]; ok {
				if err := validateSharedCredentialMetadata(refs[index].Provenance[0], provenance); err != nil {
					return fmt.Errorf("credential %s: %w", key, err)
				}
				refs[index].Required = refs[index].Required || descriptor.Required
				refs[index].ConsumerRefs = appendUniqueStrings(refs[index].ConsumerRefs, consumerRefs...)
				refs[index].MigrationDiagnostics = appendUniqueMigrationDiagnostics(refs[index].MigrationDiagnostics, migrationDiagnostics...)
				refs[index].Provenance = append(refs[index].Provenance, provenance)
				continue
			}
			byAddress[key] = len(refs)
			refs = append(refs, CredentialRef{Version: descriptor.Version, Resource: resource, Env: descriptor.Env, LogicalID: descriptor.LogicalID, Field: descriptor.ResolvedField(), Owner: owner, SourceRef: sourceRef, Kind: descriptorKind(descriptor), Provider: descriptor.Provider, AppliesWhen: descriptor.AppliesWhen, Tiers: append([]string(nil), descriptor.Tiers...), RequirementGroup: descriptor.RequirementGroup, ConsumerRefs: consumerRefs, MigrationDiagnostics: migrationDiagnostics, CompanionSettings: append([]string(nil), descriptor.CompanionSettings...), CompanionCredentials: append([]string(nil), descriptor.CompanionCredentials...), AcquisitionRef: descriptor.AcquisitionRef, VerificationRef: descriptor.VerificationRef, RecoveryRef: descriptor.RecoveryRef, HelpRef: descriptor.HelpRef, EvidencePolicy: descriptor.EvidencePolicy, ProviderVersion: descriptor.ProviderVersion, Provenance: []CredentialProvenance{provenance}, Label: firstNonEmpty(descriptor.Label, descriptor.Description), Description: descriptor.Description, ObtainURL: descriptor.ObtainURL, Provisioning: descriptor.Provisioning, DerivedFrom: descriptor.DerivedFrom, Required: descriptor.Required})
		}
		return nil
	}
	if scope.IncludeProject {
		projectManifestPath := filepath.Join(root, ".vrooli", "service.json")
		projectManifest, err := scenario.ReadService(projectManifestPath)
		switch {
		case err == nil:
			if addErr := add(ProjectScopeOwner, projectManifestPath, projectManifest.Credentials); addErr != nil {
				return nil, addErr
			}
		case os.IsNotExist(err):
			// A bundle catalog has no repository-root manifest by design.
		default:
			return nil, fmt.Errorf("read project service manifest: %w", err)
		}
	}
	if scope.Resources == nil || len(scope.Resources) > 0 {
		if scope.Resources != nil {
			for _, name := range scope.Resources {
				path := manifestpkg.DefaultPath(root, name)
				resourceManifest, loadErr := manifestpkg.Load(path)
				if loadErr == nil {
					if addErr := add(resourceManifest.Name, path, resourceManifest.Credentials); addErr != nil {
						return nil, addErr
					}
					continue
				}
				fallback, fallbackErr := readCredentialDeclaration(path)
				if fallbackErr != nil {
					return nil, fmt.Errorf("read selected resource manifest %s: %w", name, loadErr)
				}
				if addErr := add(name, path, fallback); addErr != nil {
					return nil, addErr
				}
			}
		} else {
			names, err := catalog.New(root).ManifestNames()
			if err != nil {
				return nil, fmt.Errorf("discover resource manifests: %w", err)
			}
			for _, name := range names {
				path := manifestpkg.DefaultPath(root, name)
				resourceManifest, loadErr := manifestpkg.Load(path)
				if loadErr == nil {
					if addErr := add(resourceManifest.Name, path, resourceManifest.Credentials); addErr != nil {
						return nil, addErr
					}
				}
			}
		}
	}
	if scope.Scenarios == nil || len(scope.Scenarios) > 0 {
		if scope.Scenarios != nil {
			for _, name := range scope.Scenarios {
				path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
				manifest, readErr := scenario.ReadService(path)
				if readErr != nil {
					return nil, fmt.Errorf("read selected scenario manifest %s: %w", name, readErr)
				}
				owner := strings.TrimSpace(manifest.Service.Name)
				if owner == "" {
					owner = name
				}
				if addErr := add(owner, path, manifest.Credentials); addErr != nil {
					return nil, addErr
				}
			}
		} else {
			found, err := scenario.Discover(root, scenario.SandboxEnvFromEnv())
			if err != nil {
				return nil, fmt.Errorf("discover scenario manifests: %w", err)
			}
			for _, item := range found {
				if addErr := add(item.Slug, item.ServicePath, item.Manifest.Credentials); addErr != nil {
					return nil, addErr
				}
			}
		}
	}
	if scope.IncludeManaged {
		for _, managed := range credentialinventory.ManagedSystemEntries(root) {
			key := strings.TrimSpace(managed.LogicalID) + ":" + strings.TrimSpace(managed.Field)
			if _, exists := byAddress[key]; exists {
				continue
			}
			owner := strings.TrimSpace(managed.Owner)
			if owner == "" {
				owner = "managed-system"
			}
			sourceRef := filepath.Join(root, "internal", "credentialinventory", "inventory.go")
			descriptor := credentialspec.Descriptor{
				LogicalID: managed.LogicalID, Field: managed.Field,
				Kind: credentialspec.KindDelegated, Provisioning: credentialspec.ProvisioningGenerated,
			}
			consumerRefs := declaredConsumerNames(managed.Consumers, descriptor, scope.Tier)
			consumerProvenance := declaredConsumerProvenance(managed.Consumers, descriptor, sourceRef, scope.Tier)
			required := false
			for _, consumer := range managed.Consumers {
				if consumer.Required {
					required = true
					break
				}
			}
			provenance := CredentialProvenance{
				Version: "managed-credential/v1", Owner: owner, SourceRef: sourceRef,
				Kind: "managed", Provisioning: credentialspec.ProvisioningGenerated,
				ConsumerRefs: consumerRefs, Consumers: consumerProvenance, Required: required,
			}
			byAddress[key] = len(refs)
			refs = append(refs, CredentialRef{
				Version: "managed-credential/v1", Resource: owner, LogicalID: managed.LogicalID, Field: managed.Field,
				Owner: owner, SourceRef: sourceRef, Kind: "managed", ConsumerRefs: consumerRefs, Label: "Managed " + owner,
				Description: "Authority-owned identity managed by the control plane.", Provisioning: credentialspec.ProvisioningGenerated,
				Required: required, Provenance: []CredentialProvenance{provenance},
			})
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

// validateSharedCredentialMetadata protects the invariant that one authority
// address has one security/provisioning contract. Consumer-specific labels,
// descriptions, acquisition links, and requiredness are intentionally kept in
// provenance because different consumers may impose different operator
// constraints without changing how the value is delivered. Environment names
// and consumer kinds may legitimately differ here: one owner can broker the
// address directly while another injects the same authority value into a
// managed process, and both facts belong in provenance.
func validateSharedCredentialMetadata(existing, incoming CredentialProvenance) error {
	if existing.Provisioning != "" && incoming.Provisioning != "" && existing.Provisioning != incoming.Provisioning {
		return fmt.Errorf("incompatible provisioning modes between %s (%s) and %s (%s)", existing.SourceRef, existing.Provisioning, incoming.SourceRef, incoming.Provisioning)
	}
	if existing.DerivedFrom != incoming.DerivedFrom {
		return fmt.Errorf("incompatible derivation sources between %s (%s) and %s (%s)", existing.SourceRef, existing.DerivedFrom, incoming.SourceRef, incoming.DerivedFrom)
	}
	for name, values := range map[string][2]string{
		"provider":          {existing.Provider, incoming.Provider},
		"requirement group": {existing.RequirementGroup, incoming.RequirementGroup}, "acquisition action": {existing.AcquisitionRef, incoming.AcquisitionRef},
		"verification action": {existing.VerificationRef, incoming.VerificationRef}, "recovery action": {existing.RecoveryRef, incoming.RecoveryRef},
		"evidence policy": {existing.EvidencePolicy, incoming.EvidencePolicy},
	} {
		left, right := values[0], values[1]
		if left != "" && right != "" && left != right {
			return fmt.Errorf("incompatible %s between %s (%s) and %s (%s)", name, existing.SourceRef, left, incoming.SourceRef, right)
		}
	}
	return nil
}

func readCredentialDeclaration(path string) (credentialspec.Declaration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return credentialspec.Declaration{}, err
	}
	var manifest struct {
		Credentials credentialspec.Declaration `json:"credentials"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return credentialspec.Declaration{}, err
	}
	return manifest.Credentials, nil
}

func declaredConsumerNames(consumers []credentialspec.Consumer, descriptor credentialspec.Descriptor, tier string) []string {
	address := strings.TrimSpace(descriptor.LogicalID) + ":" + descriptor.ResolvedField()
	seen := make(map[string]struct{})
	names := make([]string, 0)
	for _, consumer := range consumers {
		if tier != "" && !tierAppliesToConsumer(consumer, tier) {
			continue
		}
		field := strings.TrimSpace(consumer.Field)
		if field == "" {
			field = credentialspec.DefaultField
		}
		if pattern := strings.TrimSpace(consumer.AddressPattern); pattern != "" {
			if !matchesAddressPattern(pattern, address) {
				continue
			}
		} else {
			if strings.TrimSpace(consumer.LogicalID) == "" || strings.TrimSpace(consumer.LogicalID)+":"+field != address {
				continue
			}
		}
		name := strings.TrimSpace(consumer.Consumer)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func declaredConsumerProvenance(consumers []credentialspec.Consumer, descriptor credentialspec.Descriptor, manifestRef, tier string) []CredentialConsumerProvenance {
	address := strings.TrimSpace(descriptor.LogicalID) + ":" + descriptor.ResolvedField()
	result := make([]CredentialConsumerProvenance, 0)
	for _, consumer := range consumers {
		if tier != "" && !tierAppliesToConsumer(consumer, tier) {
			continue
		}
		field := strings.TrimSpace(consumer.Field)
		if field == "" {
			field = credentialspec.DefaultField
		}
		if pattern := strings.TrimSpace(consumer.AddressPattern); pattern != "" {
			if !matchesAddressPattern(pattern, address) {
				continue
			}
		} else if consumerAddress := strings.TrimSpace(consumer.LogicalID) + ":" + field; consumerAddress != address {
			continue
		}
		result = append(result, CredentialConsumerProvenance{
			LogicalID: consumer.LogicalID, AddressPattern: consumer.AddressPattern, Field: field, Kind: consumer.Kind,
			Consumer: consumer.Consumer, SourceRef: absoluteConsumerSourceRef(manifestRef, consumer.SourceRef),
			Required: consumer.Required, Reason: consumer.Reason, Tiers: append([]string(nil), consumer.Tiers...),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SourceRef == result[j].SourceRef {
			return result[i].Consumer < result[j].Consumer
		}
		return result[i].SourceRef < result[j].SourceRef
	})
	return result
}

func absoluteConsumerSourceRef(manifestRef, sourceRef string) string {
	if strings.TrimSpace(sourceRef) == "" || filepath.IsAbs(sourceRef) {
		return sourceRef
	}
	ownerRoot := filepath.Dir(manifestRef)
	if filepath.Base(ownerRoot) == ".vrooli" {
		ownerRoot = filepath.Dir(ownerRoot)
	}
	return filepath.Clean(filepath.Join(ownerRoot, sourceRef))
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func descriptorMigrationDiagnostics(descriptor credentialspec.Descriptor) []credentialspec.MigrationDiagnostic {
	return descriptorDeclaration(descriptor).MigrationDiagnostics()
}

func descriptorDeclaration(descriptor credentialspec.Descriptor) credentialspec.Declaration {
	return credentialspec.Declaration{Descriptors: []credentialspec.Descriptor{descriptor}}
}

func appendUniqueMigrationDiagnostics(values []credentialspec.MigrationDiagnostic, additions ...credentialspec.MigrationDiagnostic) []credentialspec.MigrationDiagnostic {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value.Address+":"+value.Code] = struct{}{}
	}
	for _, value := range additions {
		key := value.Address + ":" + value.Code
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Address == values[j].Address {
			return values[i].Code < values[j].Code
		}
		return values[i].Address < values[j].Address
	})
	return values
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
