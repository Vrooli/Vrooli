package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/tokenparity"
)

// ValidateFallbackParity makes authored fallbacks an honest view of the
// canonical base vocabulary. Properties owned inside the same version and
// host-computed --rcl-* contracts are intentionally outside this comparison.
func ValidateFallbackParity(scope Scope) (Result, error) {
	root := scope.Root
	canonical, err := tokenparity.LoadCanonical(root)
	if err != nil {
		return Result{}, fmt.Errorf("read canonical token vocabulary: %w", err)
	}
	// Backfilled releases preserve immutable historical bytes. Their fallbacks
	// were authored against an earlier canonical ramp and are intentionally not
	// rewritten during a current-corpus parity check; newly published releases
	// remain fully subject to this gate.
	var ledger struct {
		Entries []struct {
			LibraryID  string `json:"libraryId"`
			Version    string `json:"version"`
			Backfilled bool   `json:"backfilled"`
		} `json:"entries"`
	}
	backfilled := map[string]bool{}
	if raw, readErr := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "library", "release-provenance.json")); readErr == nil {
		if json.Unmarshal(raw, &ledger) == nil {
			for _, entry := range ledger.Entries {
				if entry.Backfilled {
					backfilled[entry.LibraryID+"@"+entry.Version] = true
				}
			}
		}
	}
	sources, err := activeLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	byVersion := map[string][]string{}
	for _, source := range sources {
		if !strings.Contains(filepath.ToSlash(source), "/library/components/") {
			continue
		}
		byVersion[filepath.Dir(source)] = append(byVersion[filepath.Dir(source)], source)
	}
	result := Result{}
	versionPathRE := regexp.MustCompile(`/components/([^/]+)/versions/([^/]+)/`)
	for _, versionSources := range byVersion {
		if len(versionSources) > 0 {
			versionPath := filepath.ToSlash(versionSources[0])
			match := versionPathRE.FindStringSubmatch(versionPath)
			if len(match) == 3 && backfilled["react-component-library:"+match[1]+"@"+match[2]] {
				versionSources = nil
			}
		}
		if len(versionSources) == 0 {
			continue
		}
		contents := make(map[string][]byte, len(versionSources))
		for _, source := range versionSources {
			data, readErr := os.ReadFile(source)
			if readErr != nil {
				return Result{}, readErr
			}
			contents[source] = data
		}
		result.Inspected += len(versionSources)
		for _, mismatch := range tokenparity.CheckVersion(canonical, contents) {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.fallback_parity", File: repoRel(root, mismatch.File), Line: lineAt(contents[mismatch.File], mismatch.Offset),
				Message:     fmt.Sprintf("%s fallback %q disagrees with canonical value %q", mismatch.Property, mismatch.Value, mismatch.Canonical),
				Remediation: fmt.Sprintf("Use var(%s, %s), or declare the property inside this component version when it is genuinely component-owned.", mismatch.Property, mismatch.Canonical),
				DocsRef:     "docs/design/TOKEN-DICTIONARY.md",
			})
		}
	}
	return result, nil
}

// ValidateKitCompatibility blocks versions whose token contract cannot be
// satisfied by the registered kit vocabulary.
