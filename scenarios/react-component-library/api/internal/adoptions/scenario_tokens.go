package adoptions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/librarywalk"
	"react-component-library/internal/themes"
)

// TokenNamespace reads the consumer's declared semantic namespace. The
// scenarios intentionally keep isolated Tailwind configs, so this is a
// filesystem fact rather than a library-global default.
func (r *FSScenarioFileReader) TokenNamespace(_ context.Context, scenario string) (string, error) {
	base := filepath.Join(r.root, scenario, "ui")
	raw, err := os.ReadFile(filepath.Join(base, "tailwind.config.ts"))
	if err != nil {
		if os.IsNotExist(err) {
			return "app", nil
		}
		return "", err
	}
	text := string(raw)
	switch {
	case strings.Contains(text, "wc:"):
		return "wc", nil
	case strings.Contains(text, "slate:"):
		return "slate", nil
	default:
		return "app", nil
	}
}

func (r *FSScenarioFileReader) TokenMapping(_ context.Context, scenario string) (TokenMapping, error) {
	path := filepath.Join(r.root, scenario, "ui", "token-map.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TokenMapping{}, fmt.Errorf("adoption token mapping missing for scenario %q at %s", scenario, path)
		}
		return TokenMapping{}, err
	}
	var mapping TokenMapping
	if err := json.Unmarshal(raw, &mapping); err != nil {
		return TokenMapping{}, fmt.Errorf("decode adoption token mapping for %q: %w", scenario, err)
	}
	if strings.TrimSpace(mapping.Namespace) == "" {
		return TokenMapping{}, fmt.Errorf("adoption token mapping for scenario %q is missing namespace", scenario)
	}
	if err := validateTokenMapping(mapping, []string{"app-danger", "app-info", "app-primary", "app-warning"}); err != nil {
		return TokenMapping{}, fmt.Errorf("validate adoption token mapping for %q: %w", scenario, err)
	}
	return mapping, nil
}

// DeclaredTokens returns every custom property declared by the scenario UI.
// It intentionally scans source rather than only the canonical ramp so a
// consumer-owned declaration outside the managed region still satisfies a
// library requirement without being overwritten.
func (r *FSScenarioFileReader) DeclaredTokens(ctx context.Context, scenario string) ([]string, error) {
	base := filepath.Join(r.root, scenario, "ui")
	declared := map[string]struct{}{}
	if _, err := os.Stat(base); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	err := librarywalk.WalkContext(ctx, base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".css" && ext != ".ts" && ext != ".tsx" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range scenarioTokenDeclarationRE.FindAllStringSubmatch(string(raw), -1) {
			if len(match) == 2 {
				declared[match[1]] = struct{}{}
			}
		}
		for _, match := range scenarioRuntimeTokenWriteRE.FindAllStringSubmatch(string(raw), -1) {
			if len(match) == 2 {
				declared[match[1]] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan declared tokens for scenario %q: %w", scenario, err)
	}
	result := make([]string, 0, len(declared))
	for property := range declared {
		result = append(result, property)
	}
	sort.Strings(result)
	return result, nil
}

// RuntimeWrittenTokens reports properties whose value is owned by application
// state rather than by the static design-token ramp. Token sync removes these
// from its managed region so a stale default cannot race the runtime writer.
func (r *FSScenarioFileReader) RuntimeWrittenTokens(ctx context.Context, scenario string) ([]string, error) {
	base := filepath.Join(r.root, scenario, "ui", "src")
	written := map[string]struct{}{}
	if _, err := os.Stat(base); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	err := librarywalk.WalkContext(ctx, base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range scenarioRuntimeTokenWriteRE.FindAllStringSubmatch(string(raw), -1) {
			if len(match) == 2 {
				written[match[1]] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan runtime token writers for scenario %q: %w", scenario, err)
	}
	result := make([]string, 0, len(written))
	for property := range written {
		result = append(result, property)
	}
	sort.Strings(result)
	return result, nil
}

func (r *FSScenarioFileReader) ImportedLibrarySpecifiers(ctx context.Context, scenario string) ([]components.LibraryPackageSpecifier, error) {
	sources, err := r.walkScenarioSources(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[components.LibraryPackageSpecifier]bool{}
	var result []components.LibraryPackageSpecifier
	for _, source := range sources {
		if source.scenario != scenario {
			continue
		}
		for _, specifier := range components.LibraryPackageSpecifiers(string(source.content)) {
			if !seen[specifier] {
				seen[specifier] = true
				result = append(result, specifier)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].RequestedVersion < result[j].RequestedVersion
	})
	return result, nil
}

func (r *FSScenarioFileReader) ReferenceTokens(_ context.Context) (map[string]themes.DesignToken, error) {
	path := filepath.Join(filepath.Dir(r.root), "templates", "design", "_base", "tokens.css")
	tokens, err := themes.ReadTokenFile(path)
	if err != nil {
		return nil, err
	}
	result := make(map[string]themes.DesignToken, len(tokens))
	for _, token := range tokens {
		if token.Tier == "" {
			return nil, fmt.Errorf("canonical token %s has no parseable tier annotation", token.Name)
		}
		result[token.Name] = token
	}
	return result, nil
}

// TokenFallbackExemptions returns the exact host/runtime exceptions declared
// by the canonical RCL catalog. Read errors fail preflight closed.
func (r *FSScenarioFileReader) TokenFallbackExemptions(_ context.Context) (map[string]string, error) {
	return components.ReadTokenFallbackExemptions(filepath.Join(r.root, "react-component-library", "catalog", "config.json"))
}
