// Package credentialinventory builds the non-secret address inventory used by
// control-plane recovery. Manifest declarations and live managed instances are
// the sources of truth; values are resolved only later by the credential
// authority during encryption.
package credentialinventory

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/resources/catalog"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
)

type Result struct {
	Entries        []credentialauthority.RecoveryEntry
	RequiredAbsent []string
}

// Collect returns configured credential addresses and the required addresses
// that are declared but absent. It never returns a credential value.
func Collect(root string) (Result, error) {
	if strings.TrimSpace(root) == "" {
		return Result{}, nil
	}
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return Result{}, err
	}
	if err := authority.Availability(); err != nil {
		return Result{}, err
	}
	entries := map[string]credentialauthority.RecoveryEntry{}
	absent := map[string]struct{}{}
	add := func(owner string, declaration credentialspec.Declaration) error {
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
			key := string(identity) + ":" + descriptor.ResolvedField()
			if gap, missing := gapByKey[key]; missing {
				if descriptor.Required {
					absent[key] = struct{}{}
				}
				_ = gap
				continue
			}
			entries[key] = credentialauthority.RecoveryEntry{Identity: identity, Field: descriptor.ResolvedField()}
		}
		return nil
	}

	for _, entry := range resources.LiveVaultUnsealKeyEntries() {
		if err := add(entry.LogicalID, credentialspec.Declaration{Descriptors: []credentialspec.Descriptor{{LogicalID: entry.LogicalID, Field: entry.Field, Required: true}}}); err != nil {
			return Result{}, err
		}
	}
	for _, entry := range resources.LiveKopiaRepositoryEntries() {
		if err := add(entry.LogicalID, credentialspec.Declaration{Descriptors: []credentialspec.Descriptor{{LogicalID: entry.LogicalID, Field: entry.Field, Required: true}}}); err != nil {
			return Result{}, err
		}
	}

	resourceNames, err := catalog.New(root).ManifestNames()
	if err != nil {
		return Result{}, fmt.Errorf("discover resource manifests: %w", err)
	}
	sort.Strings(resourceNames)
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

	result := Result{Entries: make([]credentialauthority.RecoveryEntry, 0, len(entries)), RequiredAbsent: make([]string, 0, len(absent))}
	for _, entry := range entries {
		result.Entries = append(result.Entries, entry)
	}
	for key := range absent {
		result.RequiredAbsent = append(result.RequiredAbsent, key)
	}
	sort.Slice(result.Entries, func(i, j int) bool {
		left := string(result.Entries[i].Identity) + ":" + result.Entries[i].Field
		right := string(result.Entries[j].Identity) + ":" + result.Entries[j].Field
		return left < right
	})
	sort.Strings(result.RequiredAbsent)
	return result, nil
}
