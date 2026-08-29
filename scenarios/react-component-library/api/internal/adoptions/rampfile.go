package adoptions

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/themes"
)

const (
	tokenRampPath  = "ui/src/design-tokens.css" // #nosec G101 -- this is a source path, not a credential.
	tokenRampBegin = "/* rcl:tokens:begin */"   // #nosec G101 -- this is a managed-region marker, not a credential.
	tokenRampEnd   = "/* rcl:tokens:end */"     // #nosec G101 -- this is a managed-region marker, not a credential.
)

type TokenSyncInput struct {
	Scenario string
	DryRun   bool
}

type TokenSyncResult struct {
	Scenario   string
	Added      []string
	Collisions []string
	Changed    bool
}

type TokenPruneInput struct {
	Scenario string
	Apply    bool
}

type TokenPruneResult struct {
	Scenario string
	Removed  []string
	Retained []string
	Changed  bool
}

type rampFile struct {
	prefix, managed, suffix string
}

var rampDeclarationRE = regexp.MustCompile(`(?m)^\s*(--[A-Za-z0-9_-]+)\s*:\s*([^;]+);?\s*$`)

func parseRampFile(raw string) (rampFile, error) {
	begin := strings.Index(raw, tokenRampBegin)
	end := strings.Index(raw, tokenRampEnd)
	switch {
	case begin < 0 && end < 0:
		if prefix, suffix, ok := splitRootBlock(raw); ok {
			return rampFile{prefix: prefix, suffix: suffix}, nil
		}
		return rampFile{prefix: raw}, nil
	case begin < 0 || end < 0:
		return rampFile{}, fmt.Errorf("token ramp must contain both %s and %s", tokenRampBegin, tokenRampEnd)
	case end < begin:
		return rampFile{}, fmt.Errorf("token ramp markers are out of order")
	}
	managedStart := begin + len(tokenRampBegin)
	prefix := raw[:begin]
	managed := raw[managedStart:end]
	suffix := raw[end+len(tokenRampEnd):]
	if strings.Count(prefix, "{") <= strings.Count(prefix, "}") {
		if rootPrefix, rootSuffix, ok := splitRootBlock(prefix + suffix); ok {
			return rampFile{prefix: rootPrefix, managed: managed, suffix: rootSuffix}, nil
		}
	}
	return rampFile{prefix: prefix, managed: managed, suffix: suffix}, nil
}

func splitRootBlock(raw string) (string, string, bool) {
	root := regexp.MustCompile(`:root\s*\{`).FindStringIndex(raw)
	if root == nil {
		return "", "", false
	}
	depth := 0
	for index := root[1] - 1; index < len(raw); index++ {
		switch raw[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return raw[:index], raw[index:], true
			}
		}
	}
	return "", "", false
}

func (r rampFile) render() string {
	return r.prefix + tokenRampBegin + "\n" + strings.Trim(r.managed, "\n") + "\n" + tokenRampEnd + r.suffix
}

func (s *service) SyncScenarioTokens(ctx context.Context, in TokenSyncInput) (TokenSyncResult, error) {
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return TokenSyncResult{}, ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	required, err := s.requiredTokensForScenario(ctx, scenario)
	if err != nil {
		return TokenSyncResult{}, err
	}
	declared, err := s.tokenInventory.DeclaredTokens(ctx, scenario)
	if err != nil {
		return TokenSyncResult{}, err
	}
	declaredSet := stringSet(declared)
	missing := make(map[string]struct{})
	for property := range required {
		if _, ok := declaredSet[property]; !ok {
			missing[property] = struct{}{}
		}
	}
	result := TokenSyncResult{Scenario: scenario, Added: sortedKeys(missing)}
	raw, readErr := s.files.Read(ctx, scenario, tokenRampPath)
	if readErr != nil && !isMissingAdoptedFile(readErr) {
		return TokenSyncResult{}, readErr
	}
	ramp, err := parseRampFile(string(raw))
	if err != nil {
		return TokenSyncResult{}, err
	}
	managed := make(map[string]string)
	for _, match := range rampDeclarationRE.FindAllStringSubmatch(ramp.managed, -1) {
		managed[match[1]] = strings.TrimSpace(match[2])
	}
	contractTierRemoved := false
	if reader, ok := s.tokenInventory.(ScenarioReferenceTokenReader); ok {
		tokens, tokenErr := reader.ReferenceTokens(ctx)
		if tokenErr != nil {
			return TokenSyncResult{}, tokenErr
		}
		for property := range managed {
			if token, exists := tokens[property]; exists && token.Tier == themes.TokenTierContract {
				delete(managed, property)
				contractTierRemoved = true
			}
		}
	}
	runtimeOwned := map[string]struct{}{}
	runtimeCollisionRemoved := false
	if reader, ok := s.tokenInventory.(ScenarioRuntimeTokenInventoryReader); ok {
		properties, runtimeErr := reader.RuntimeWrittenTokens(ctx, scenario)
		if runtimeErr != nil {
			return TokenSyncResult{}, runtimeErr
		}
		for _, property := range properties {
			runtimeOwned[property] = struct{}{}
			if _, exists := managed[property]; exists {
				runtimeCollisionRemoved = true
			}
			delete(managed, property)
		}
	}
	outside := make(map[string]struct{})
	for _, part := range []string{ramp.prefix, ramp.suffix} {
		for _, match := range scenarioTokenDeclarationRE.FindAllStringSubmatch(part, -1) {
			outside[match[1]] = struct{}{}
		}
	}
	values, err := s.referenceRampValues(ctx)
	if err != nil {
		return TokenSyncResult{}, err
	}
	for property := range required {
		if _, exists := managed[property]; exists {
			continue
		}
		if _, exists := outside[property]; exists {
			result.Collisions = append(result.Collisions, property)
		}
	}
	for property := range missing {
		if _, exists := managed[property]; !exists {
			value := strings.TrimSpace(values[property])
			if value == "" {
				value = "initial"
			}
			managed[property] = value
		}
	}
	sort.Strings(result.Collisions)
	ramp.managed = renderRampDeclarations(managed)
	result.Changed = len(result.Added) > 0 || runtimeCollisionRemoved || contractTierRemoved || !strings.Contains(string(raw), tokenRampBegin)
	if result.Changed && !in.DryRun {
		if _, err := s.files.Write(ctx, scenario, tokenRampPath, []byte(ramp.render())); err != nil {
			return TokenSyncResult{}, err
		}
	}
	return result, nil
}

func (s *service) PruneScenarioTokens(ctx context.Context, in TokenPruneInput) (TokenPruneResult, error) {
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return TokenPruneResult{}, ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	required, err := s.requiredTokensForScenario(ctx, scenario)
	if err != nil {
		return TokenPruneResult{}, err
	}
	raw, err := s.files.Read(ctx, scenario, tokenRampPath)
	if err != nil {
		if isMissingAdoptedFile(err) {
			return TokenPruneResult{Scenario: scenario}, nil
		}
		return TokenPruneResult{}, err
	}
	ramp, err := parseRampFile(string(raw))
	if err != nil {
		return TokenPruneResult{}, err
	}
	managed := make(map[string]string)
	for _, match := range rampDeclarationRE.FindAllStringSubmatch(ramp.managed, -1) {
		managed[match[1]] = strings.TrimSpace(match[2])
	}
	result := TokenPruneResult{Scenario: scenario}
	for property, value := range managed {
		if _, needed := required[property]; needed {
			result.Retained = append(result.Retained, property)
		} else {
			result.Removed = append(result.Removed, property)
			delete(managed, property)
			_ = value
		}
	}
	sort.Strings(result.Removed)
	sort.Strings(result.Retained)
	result.Changed = len(result.Removed) > 0
	if result.Changed && in.Apply {
		ramp.managed = renderRampDeclarations(managed)
		if _, err := s.files.Write(ctx, scenario, tokenRampPath, []byte(ramp.render())); err != nil {
			return TokenPruneResult{}, err
		}
	}
	return result, nil
}

func (s *service) requiredTokensForScenario(ctx context.Context, scenario string) (map[string]struct{}, error) {
	rows, err := s.repo.List(ctx, ListQuery{Scenario: scenario, Limit: 100000})
	if err != nil {
		return nil, err
	}
	required := make(map[string]struct{})
	seenVersions := map[string]bool{}
	collect := func(root components.Component, version components.ComponentVersion) error {
		key := root.ID + "@" + version.Version
		if seenVersions[key] {
			return nil
		}
		seenVersions[key] = true
		closure, closureErr := s.resolveAdoptionClosure(ctx, root, version, scenario, nil)
		if closureErr != nil {
			return closureErr
		}
		for _, asset := range closure.Assets {
			for _, property := range asset.Version.RequiredTokens {
				required[property] = struct{}{}
			}
			for _, pattern := range asset.Version.RequiredTokenPatterns {
				prefix := strings.TrimSuffix(pattern, "*")
				for property := range s.referenceRampCache(ctx) {
					if strings.HasPrefix(property, prefix) {
						required[property] = struct{}{}
					}
				}
			}
		}
		return nil
	}
	for _, row := range rows {
		root, err := s.library.Get(ctx, row.ComponentID)
		var missing components.ErrComponentNotFound
		if errors.As(err, &missing) && strings.TrimSpace(row.LibraryID) != "" {
			// Component UUIDs are projection identities and can change when the
			// source catalog is rebuilt. Adoption rows also retain the stable
			// manifest library ID, so resolve through it instead of making a
			// catalog reindex permanently break token synchronization.
			root, err = s.library.Get(ctx, row.LibraryID)
		}
		if err != nil {
			return nil, err
		}
		versionName := row.AdoptedVersion
		// Sync prepares the target for the same version that a default reapply
		// resolves. Using the historical adopted version here can leave
		// latest-only token requirements unsatisfied immediately after sync.
		if strings.TrimSpace(root.LatestVersion) != "" {
			versionName = root.LatestVersion
		}
		version, err := s.library.GetVersion(ctx, root.ID, versionName)
		if err != nil {
			return nil, err
		}
		if err := collect(root, version); err != nil {
			return nil, err
		}
	}
	if reader, ok := s.files.(ScenarioLibraryImportReader); ok {
		specifiers, scanErr := reader.ImportedLibrarySpecifiers(ctx, scenario)
		if scanErr != nil {
			return nil, scanErr
		}
		for _, specifier := range specifiers {
			root, getErr := s.library.Get(ctx, "react-component-library:"+specifier.Name)
			if getErr != nil {
				return nil, getErr
			}
			versions, listErr := s.library.ListVersions(ctx, root.ID, 100000)
			if listErr != nil {
				return nil, listErr
			}
			active := make([]string, 0, len(versions))
			for _, version := range versions {
				if version.Status == components.VersionStatusReleased && version.Presence != "evicted" {
					active = append(active, version.Version)
				}
			}
			selected, found := components.SelectActivePackageVersion(active, specifier.RequestedVersion)
			if !found {
				return nil, fmt.Errorf("imported asset %s has no active release matching %q", specifier.Name, specifier.RequestedVersion)
			}
			version, versionErr := s.library.GetVersion(ctx, root.ID, selected)
			if versionErr != nil {
				return nil, versionErr
			}
			if err := collect(root, version); err != nil {
				return nil, err
			}
		}
	}
	if reader, ok := s.tokenInventory.(ScenarioReferenceTokenReader); ok {
		tokens, tokenErr := reader.ReferenceTokens(ctx)
		if tokenErr != nil {
			return nil, tokenErr
		}
		for property := range required {
			token, exists := tokens[property]
			// Sync owns only the canonical design vocabulary. Asset-local/runtime
			// properties (including --rcl-*) are supplied by the component unit,
			// while undefined vocabulary is diagnosed by the compatibility gate.
			if !exists || token.Tier == themes.TokenTierContract {
				delete(required, property)
			}
		}
	}
	return required, nil
}

func (s *service) referenceRampCache(ctx context.Context) map[string]string {
	values, err := s.referenceRampValues(ctx)
	if err != nil {
		return map[string]string{}
	}
	return values
}

func (s *service) referenceRampValues(ctx context.Context) (map[string]string, error) {
	values := map[string]string{}
	if reader, ok := s.tokenInventory.(ScenarioReferenceTokenReader); ok {
		tokens, err := reader.ReferenceTokens(ctx)
		if err != nil {
			return nil, err
		}
		for name, token := range tokens {
			if token.Tier != themes.TokenTierContract {
				values[name] = token.Value
			}
		}
		return values, nil
	}
	raw, err := s.files.Read(ctx, "react-component-library", tokenRampPath)
	if err == nil {
		for _, match := range rampDeclarationRE.FindAllStringSubmatch(string(raw), -1) {
			values[match[1]] = strings.TrimSpace(match[2])
		}
	}
	return values, nil
}

func renderRampDeclarations(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("  %s: %s;", key, values[key]))
	}
	return strings.Join(lines, "\n")
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func isMissingAdoptedFile(err error) bool {
	var missing ErrAdoptedFileMissing
	return strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "missing") || errors.As(err, &missing)
}
