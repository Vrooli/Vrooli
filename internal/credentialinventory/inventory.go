// Package credentialinventory builds the non-secret address inventory used by
// control-plane recovery. Manifest declarations and live managed instances are
// the sources of truth; values are resolved only later by the credential
// authority during encryption.
package credentialinventory

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/resources/catalog"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/securestore"
)

type Result struct {
	// Basis identifies what the inventory count means. The authoritative
	// recovery inventory counts distinct logical addresses, not declaration
	// sites, so shared credentials are represented once.
	Basis string
	// Entries contains addresses that currently resolve through the authority.
	// It is the value-free input used by recovery export.
	Entries []credentialauthority.RecoveryEntry
	// Declared contains every declared address, including required addresses
	// that are not currently configured. It is kept separate from Entries so
	// inventory consumers cannot mistake an absent value for an undeclared
	// address.
	Declared []credentialauthority.RecoveryEntry
	// DeclarationSiteCount counts descriptor occurrences before address
	// de-duplication. Managed instances are included in both this count and
	// Declared because they are recovery-bearing credential sources.
	DeclarationSiteCount     int
	ManagedInstancesIncluded bool
	RequiredAbsent           []string
	// Rows preserves one row per declaration site for diagnostic consumers.
	// Entries remains de-duplicated for recovery selection; Rows retains the
	// richer owner and descriptor metadata used by `credentials list`.
	Rows []Entry
}

// Entry is one metadata-only credential declaration projected for diagnostic
// output. It deliberately contains no credential value.
type Entry struct {
	Resource     string
	Env          string
	LogicalID    string
	Field        string
	Label        string
	Description  string
	Required     bool
	Provisioning string
	DerivedFrom  string
	Configured   bool
	State        string
	Remediation  string
}

// CollectionOptions supplies live-source seams to CollectWithOptions. The
// defaults are production sources; callers use this only to isolate tests.
type CollectionOptions struct {
	LiveVaultUnsealKeyEntries  func() []resources.VaultUnsealKeyEntry
	LiveKopiaRepositoryEntries func() []resources.KopiaRepositoryEntry
}

func providerGapReason(state credentialauthority.ProviderState) resourceenv.CredentialGapReason {
	if state == credentialauthority.ProviderAbsent {
		return resourceenv.GapProviderAbsent
	}
	return resourceenv.GapProviderUnavailable
}

func providerRemediation(state credentialauthority.ProviderState) string {
	if state == credentialauthority.ProviderAbsent {
		return "this host has no credential backend; run `vrooli credentials doctor` to see what to install"
	}
	return "the credential store is unreachable; run `vrooli credentials doctor` for the host diagnosis"
}

// SystemEntry is a live authority-owned credential that has no resource or
// scenario manifest to declare it. These are still inventory inputs, but their
// ownership is explicit rather than being guessed from a manifest.
type SystemEntry struct {
	Owner     string
	LogicalID string
	Field     string
	Consumers []credentialspec.Consumer
}

// ManagedSystemEntries returns metadata-only system credentials. Dynamic
// device-control references are visible in the encrypted store's cleartext
// index; values are never opened. The release-authority reference is stable
// and is included even on hosts whose backend does not expose an index.
func ManagedSystemEntries(root string) []SystemEntry {
	if _, err := os.Stat(filepath.Join(root, repocontractmeta.ProjectConfigDir)); err != nil {
		return nil
	}
	seen := map[string]SystemEntry{}
	add := func(owner, logicalID, field string, consumers ...credentialspec.Consumer) {
		identity, err := credentialauthority.ParseIdentity(logicalID)
		if err != nil || strings.TrimSpace(field) == "" {
			return
		}
		key := string(identity) + ":" + strings.TrimSpace(field)
		seen[key] = SystemEntry{Owner: owner, LogicalID: string(identity), Field: strings.TrimSpace(field), Consumers: append([]credentialspec.Consumer(nil), consumers...)}
	}
	add("release-authority", "vrooli/release-authority", "rsa-pkcs8-v1", credentialspec.Consumer{
		LogicalID: "vrooli/release-authority", Field: "rsa-pkcs8-v1", Kind: credentialspec.KindDelegated,
		Consumer: "release metadata signer", SourceRef: filepath.Join(root, "internal", "releaseauthority", "authority.go"), Required: true,
	})
	if refs, err := securestore.ListEntryRefs(); err == nil {
		for _, ref := range refs {
			if ref.Service != "vrooli.credentials.v1" || !strings.HasPrefix(ref.Key, "device-control/") {
				continue
			}
			separator := strings.LastIndex(ref.Key, ":")
			if separator <= 0 || separator == len(ref.Key)-1 {
				continue
			}
			add("device-control", ref.Key[:separator], ref.Key[separator+1:], credentialspec.Consumer{
				AddressPattern: "device-control/{name}", Kind: "dynamic",
				Consumer: "device-control credential resolver", SourceRef: filepath.Join(root, "scenarios", "device-control", "api", "internal", "auth", "auth.go"), Required: true,
			})
		}
	}
	entries := make([]SystemEntry, 0, len(seen))
	for _, entry := range seen {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		left := entries[i].LogicalID + ":" + entries[i].Field
		right := entries[j].LogicalID + ":" + entries[j].Field
		return left < right
	})
	return entries
}

// Collect returns configured credential addresses and the required addresses
// that are declared but absent. It never returns a credential value.
//
//nolint:gocyclo // inventory collection merges independent provider, filesystem, and verification outcomes.
func Collect(root string) (Result, error) {
	return CollectWithOptions(root, CollectionOptions{})
}

// CollectWithOptions performs the authoritative credential source walk.
// Options exist solely for deterministic tests that must replace host-backed
// managed-instance discovery.
func CollectWithOptions(root string, options CollectionOptions) (Result, error) {
	if strings.TrimSpace(root) == "" {
		return Result{}, nil
	}
	if options.LiveVaultUnsealKeyEntries == nil {
		options.LiveVaultUnsealKeyEntries = resources.LiveVaultUnsealKeyEntries
	}
	if options.LiveKopiaRepositoryEntries == nil {
		options.LiveKopiaRepositoryEntries = resources.LiveKopiaRepositoryEntries
	}
	entries := map[string]credentialauthority.RecoveryEntry{}
	declared := map[string]credentialauthority.RecoveryEntry{}
	absent := map[string]struct{}{}
	declarationSites := 0
	rows := make([]Entry, 0)
	providerState := credentialauthority.ProviderAvailable
	if authority, authErr := credentialauthority.DefaultAuthority(); authErr != nil {
		providerState = credentialauthority.ProviderStateFor(authErr)
	} else if availabilityErr := authority.Availability(); availabilityErr != nil {
		providerState = credentialauthority.ProviderStateFor(availabilityErr)
	}
	add := func(owner string, declaration credentialspec.Declaration) error {
		resultSiteCount := len(declaration.All())
		// The closure is called once per declaration source. Keep the count in
		// the outer result rather than deriving it from the de-duplicated map.
		declarationSites += resultSiteCount
		gaps, err := resourceenv.ResolveCredentialGaps(manifestpkg.ResourceManifest{Credentials: declaration})
		if err != nil {
			// Scenario and resource resolution differ only in their owner label;
			// use the direct descriptor status path below when a synthetic
			// resource manifest is not appropriate.
			gaps, err = resourceenv.ResolveScenarioCredentialGaps(owner, declaration)
		}
		if err != nil {
			return err
		}
		gapByKey := make(map[string]resourceenv.MissingCredential, len(gaps.Missing))
		for _, gap := range gaps.Missing {
			gapByKey[gap.LogicalID+":"+gap.Field] = gap
		}
		for _, descriptor := range declaration.All() {
			identity, err := credentialauthority.ParseIdentity(descriptor.LogicalID)
			if err != nil {
				return fmt.Errorf("%s credential identity: %w", owner, err)
			}
			field := descriptor.ResolvedField()
			key := string(identity) + ":" + field
			declared[key] = credentialauthority.RecoveryEntry{Identity: identity, Field: field}
			row := Entry{
				Resource: owner, Env: strings.TrimSpace(descriptor.Env),
				LogicalID: string(identity), Field: field,
				Label: strings.TrimSpace(descriptor.Label), Description: strings.TrimSpace(descriptor.Description),
				Required: descriptor.Required, Provisioning: descriptor.Provisioning, DerivedFrom: descriptor.DerivedFrom,
				Configured: true, State: "configured",
			}
			if gap, missing := gapByKey[key]; missing {
				// RequiredAbsent is the operator-facing gap list: it drives what
				// setup and onboarding refuse to complete over. A derived or
				// generated value is written by its declaring component, so an
				// operator cannot clear it and blocking on it would name a
				// problem with no available action.
				if descriptor.Required && descriptor.OperatorSupplied() {
					absent[key] = struct{}{}
				}
				row.Configured = false
				row.State = string(gap.Reason)
				row.Remediation = gap.Remediation
				switch descriptor.Provisioning {
				case credentialspec.ProvisioningDerived:
					row.State = "derived-unconfigured"
					row.Remediation = "this value is derived by the declaring component; provision its source credential first"
				case credentialspec.ProvisioningGenerated:
					row.State = "generated-unconfigured"
					row.Remediation = "this value is generated by the declaring component on first start; there is nothing to provision"
				}
			} else if providerState != credentialauthority.ProviderAvailable {
				row.Configured = false
				row.State = string(providerGapReason(providerState))
				row.Remediation = providerRemediation(providerState)
			}
			rows = append(rows, row)
			if !row.Configured {
				continue
			}
			entries[key] = credentialauthority.RecoveryEntry{Identity: identity, Field: field}
		}
		return nil
	}

	// A broker registration can outlive a deliberately disabled resource. The
	// control-plane resource choice decides whether that runtime instance is
	// part of the authoritative recovery population.
	includeVault := false
	if configEntries, configErr := catalog.New(root).ReadConfigEntries(); configErr == nil {
		if choice, found := configEntries["vault"]; found {
			includeVault = choice.Enabled
		}
	}
	if includeVault {
		for _, entry := range options.LiveVaultUnsealKeyEntries() {
			if err := add(entry.LogicalID, credentialspec.Declaration{Descriptors: []credentialspec.Descriptor{{LogicalID: entry.LogicalID, Field: entry.Field, Label: "Vault unseal key (instance " + entry.InstanceID + ")", Required: true}}}); err != nil {
				return Result{}, err
			}
		}
	}
	for _, entry := range ManagedSystemEntries(root) {
		if err := add(entry.Owner, credentialspec.Declaration{Descriptors: []credentialspec.Descriptor{{LogicalID: entry.LogicalID, Field: entry.Field, Required: true}}}); err != nil {
			return Result{}, err
		}
	}
	for _, entry := range options.LiveKopiaRepositoryEntries() {
		if err := add(entry.LogicalID, credentialspec.Declaration{Descriptors: []credentialspec.Descriptor{{LogicalID: entry.LogicalID, Field: entry.Field, Label: "Kopia repository passphrase (repository " + entry.Repository + ")", Required: true}}}); err != nil {
			return Result{}, err
		}
	}

	// The repository root is itself a service manifest. Host-owned safeguards
	// (notably remote desktop) have no scenario directory to hang a
	// declaration from, so the project manifest is their authoritative owner.
	// Read it through the same manifest parser as scenarios so credential
	// uniqueness and descriptor validation cannot drift between the two paths.
	projectManifestPath := filepath.Join(root, repocontractmeta.ProjectConfigDir, "service.json")
	if projectManifest, projectErr := scenario.ReadService(projectManifestPath); projectErr == nil {
		if len(projectManifest.Credentials.All()) > 0 {
			if err := add("project", projectManifest.Credentials); err != nil {
				return Result{}, err
			}
		}
	} else if !os.IsNotExist(projectErr) {
		return Result{}, fmt.Errorf("read project service manifest: %w", projectErr)
	}

	resourceNames, err := catalog.New(root).ManifestNames()
	if err != nil {
		return Result{}, fmt.Errorf("discover resource manifests: %w", err)
	}
	slices.Sort(resourceNames)
	for _, name := range resourceNames {
		manifest, err := manifestpkg.Load(manifestpkg.DefaultPath(root, name))
		if err != nil || len(manifest.Credentials.All()) == 0 {
			continue
		}
		if err := add(name, manifest.Credentials); err != nil {
			return Result{}, err
		}
	}

	foundScenarios, err := scenario.Discover(root, scenario.SandboxEnvFromEnv())
	if err == nil {
		sort.Slice(foundScenarios, func(i, j int) bool { return foundScenarios[i].Slug < foundScenarios[j].Slug })
		for _, found := range foundScenarios {
			if len(found.Manifest.Credentials.All()) == 0 {
				continue
			}
			if err := add(found.Slug, found.Manifest.Credentials); err != nil {
				return Result{}, err
			}
		}
	}

	result := Result{
		Basis:                    "distinct_addresses",
		Entries:                  make([]credentialauthority.RecoveryEntry, 0, len(entries)),
		Declared:                 make([]credentialauthority.RecoveryEntry, 0, len(declared)),
		DeclarationSiteCount:     declarationSites,
		ManagedInstancesIncluded: true,
		RequiredAbsent:           make([]string, 0, len(absent)),
		Rows:                     rows,
	}
	for _, entry := range entries {
		result.Entries = append(result.Entries, entry)
	}
	for key := range absent {
		result.RequiredAbsent = append(result.RequiredAbsent, key)
	}
	for _, entry := range declared {
		result.Declared = append(result.Declared, entry)
	}
	sort.Slice(result.Entries, func(i, j int) bool {
		left := string(result.Entries[i].Identity) + ":" + result.Entries[i].Field
		right := string(result.Entries[j].Identity) + ":" + result.Entries[j].Field
		return left < right
	})
	slices.Sort(result.RequiredAbsent)
	sort.Slice(result.Declared, func(i, j int) bool {
		left := string(result.Declared[i].Identity) + ":" + result.Declared[i].Field
		right := string(result.Declared[j].Identity) + ":" + result.Declared[j].Field
		return left < right
	})
	return result, nil
}
