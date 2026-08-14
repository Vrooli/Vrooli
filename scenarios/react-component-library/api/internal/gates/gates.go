// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
)

type Finding struct{ Code, AssetID, Message string }

// Result makes runner coverage observable. A gate that reports no findings
// after inspecting zero inputs is not a passing gate; it is a broken runner.
type Result struct {
	Findings  []Finding
	Inspected int
}

var (
	cssVarRefGateRE  = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
	cssVarDeclGateRE = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
)

// ValidateTokenVocabulary rejects the retired app-prefixed CSS vocabulary in
// active library source. The consumer-side token-map vocabulary is separate
// and is intentionally not inspected here.
func ValidateTokenVocabulary(root string) (Result, error) {
	sources, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		if strings.Contains(string(raw), "--app-") {
			result.Findings = append(result.Findings, Finding{Code: "catalog.token_vocabulary", AssetID: implementationName(path), Message: fmt.Sprintf("%s still references retired --app-* vocabulary", path)})
		}
	}
	return nonEmpty(result, "token-vocabulary"), nil
}

// ValidateTokenRampComplete verifies that every external literal custom
// property used by active library source is published by the canonical RCL
// ramp. Self-defined --rcl-* properties and dynamic families are excluded.
func ValidateTokenRampComplete(root string) (Result, error) {
	sources, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	rampRaw, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "ui", "src", "design-tokens.css"))
	if err != nil {
		return Result{}, err
	}
	ramp := map[string]struct{}{}
	for _, match := range cssVarDeclGateRE.FindAllStringSubmatch(string(rampRaw), -1) {
		ramp[match[1]] = struct{}{}
	}
	result := Result{Inspected: len(sources)}
	for _, path := range sources {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(raw)
		declared := map[string]struct{}{}
		for _, match := range cssVarDeclGateRE.FindAllStringSubmatch(text, -1) {
			declared[match[1]] = struct{}{}
		}
		for _, match := range cssVarRefGateRE.FindAllStringSubmatch(text, -1) {
			property := match[1]
			if _, local := declared[property]; local || strings.HasPrefix(property, "--rcl-") || strings.HasSuffix(property, "-") {
				continue
			}
			if _, published := ramp[property]; !published {
				result.Findings = append(result.Findings, Finding{Code: "catalog.token_ramp_complete", AssetID: implementationName(path), Message: fmt.Sprintf("%s requires %s but the canonical ramp does not publish it", path, property)})
			}
		}
	}
	return nonEmpty(result, "token-ramp-complete"), nil
}

// ValidateReleasedVersionImmutable compares every indexed released version
// with its current on-disk entry and companion files. It is intentionally a
// corpus gate, independent of the indexer's write path, so direct filesystem
// edits remain observable.
func ValidateReleasedVersionImmutable(root string) (Result, error) {
	dbPath := filepath.Join(root, "scenarios", "react-component-library", "data", "react-component-library.db")
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)", dbPath),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return Result{}, fmt.Errorf("open component index: %w", err)
	}
	defer db.Close()
	rows, err := db.QueryContext(context.Background(), `SELECT v.status, v.source_path, v.content_sha256 FROM component_versions v WHERE v.status = 'released'`)
	if err != nil {
		return Result{}, fmt.Errorf("read released version hashes: %w", err)
	}
	defer rows.Close()
	result := Result{}
	for rows.Next() {
		var status, sourcePath, recorded string
		if err := rows.Scan(&status, &sourcePath, &recorded); err != nil {
			return Result{}, err
		}
		if status != "released" {
			continue
		}
		result.Inspected++
		raw, err := os.ReadFile(filepath.Join(root, "scenarios", "react-component-library", "library", sourcePath))
		if err != nil {
			result.Findings = append(result.Findings, Finding{Code: "catalog.released_version_immutable", AssetID: sourcePath, Message: fmt.Sprintf("released source %s cannot be read: %v", sourcePath, err)})
			continue
		}
		sum := sha256.Sum256(raw)
		current := hex.EncodeToString(sum[:])
		if recorded != "" && recorded != current {
			result.Findings = append(result.Findings, Finding{Code: "catalog.released_version_immutable", AssetID: sourcePath, Message: fmt.Sprintf("released source %s changed: recorded %s, current %s", sourcePath, recorded, current)})
		}
	}
	if err := rows.Err(); err != nil {
		return Result{}, err
	}
	return nonEmpty(result, "released-version-immutable"), nil
}

type assetDoc struct {
	Asset struct {
		ID, Kind, Name string
		Target         struct {
			Maturity string `json:"maturity"`
		} `json:"target"`
	} `json:"asset"`
	API *struct {
		Variants map[string][]string `json:"variants"`
		Modes    []string            `json:"modes"`
		Parts    []json.RawMessage   `json:"parts"`
	} `json:"api"`
	Fixture *struct {
		DataShapes []string `json:"dataShapes"`
		Satisfies  *struct {
			Capability    string   `json:"capability"`
			TypeArguments []string `json:"typeArguments"`
		} `json:"satisfies"`
	} `json:"fixture"`
}

func loadAssets(root string) ([]assetDoc, error) {
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []assetDoc
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc assetDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		out = append(out, doc)
	}
	return out, nil
}

// ValidateAPI checks declared API vocabulary against the implementation
// source selected by catalogId. Missing implementations are not failures of
// this runner; coverage keeps those assets at missing/scaffolded.
func ValidateAPI(root string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.API == nil {
			continue
		}
		manifest, source, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if !ok {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		text := string(data)
		for group, values := range asset.API.Variants {
			for _, value := range values {
				if !strings.Contains(text, value) {
					result.Findings = append(result.Findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared %s variant %q is absent from %s", group, value, manifest)})
				}
			}
		}
		for _, value := range asset.API.Modes {
			if !strings.Contains(text, value) {
				result.Findings = append(result.Findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared mode %q is absent from %s", value, manifest)})
			}
		}
		for _, rawPart := range asset.API.Parts {
			partID := ""
			if json.Unmarshal(rawPart, &partID) != nil {
				var part struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(rawPart, &part)
				partID = part.ID
			}
			if partID != "" && !strings.Contains(text, partID) {
				result.Findings = append(result.Findings, Finding{"catalog.api_mismatch", asset.Asset.ID, fmt.Sprintf("declared part %q is absent from %s", partID, manifest)})
			}
		}
	}
	return nonEmpty(result, "api"), nil
}

// ValidateTypes runs the same catalog conformance command declared by the
// catalog registry. Types are intentionally not inferred from the presence of
// source files: a released asset only earns this gate after the real
// TypeScript/ESLint boundary has executed successfully.
func ValidateTypes(root string) (Result, error) {
	uiDir := filepath.Join(root, "scenarios", "react-component-library", "ui")
	if _, err := os.Stat(filepath.Join(uiDir, "package.json")); err != nil {
		if os.IsNotExist(err) {
			return Result{Findings: []Finding{{
				Code:    "catalog.types_zero_inputs",
				AssetID: "catalog.runner",
				Message: "catalog UI package is missing; the declared types runner could not execute",
			}}}, nil
		}
		return Result{}, err
	}

	result := Result{Inspected: countCatalogSources(root)}
	if result.Inspected == 0 {
		return nonEmpty(result, "types"), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "pnpm", "run", "catalog:check")
	command.Dir = uiDir
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		result.Findings = append(result.Findings, Finding{
			Code:    "catalog.types_timeout",
			AssetID: "catalog.runner",
			Message: "catalog conformance timed out before the declared types gate completed",
		})
		return result, nil
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 4000 {
			message = message[len(message)-4000:]
		}
		result.Findings = append(result.Findings, Finding{
			Code:    "catalog.types_failed",
			AssetID: "catalog.runner",
			Message: "catalog conformance failed: " + message,
		})
	}
	return result, nil
}

func countCatalogSources(root string) int {
	sources, _ := activeLibrarySources(root)
	return len(sources)
}

// activeLibrarySources returns the files represented by each manifest's
// latest and draft pointers. Historical versions remain available to callers
// that pin them explicitly, but corpus-wide quality gates should measure the
// active catalog surface consistently with indexing, coverage, and the type
// gate rather than double-counting retired implementations.
func activeLibrarySources(root string) ([]string, error) {
	var sources []string
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		manifests, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		for _, manifest := range manifests {
			data, err := os.ReadFile(manifest)
			if err != nil {
				return nil, err
			}
			var doc struct {
				Latest string `json:"latest"`
				Draft  string `json:"draft"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return nil, err
			}
			versions := []string{doc.Latest}
			if doc.Draft != "" && doc.Draft != doc.Latest {
				versions = append(versions, doc.Draft)
			}
			for _, version := range versions {
				if version == "" {
					continue
				}
				for _, extension := range []string{"*.ts", "*.tsx"} {
					matches, err := filepath.Glob(filepath.Join(filepath.Dir(manifest), "versions", version, extension))
					if err != nil {
						return nil, err
					}
					sources = append(sources, matches...)
				}
			}
		}
	}
	if len(sources) > 0 {
		sort.Strings(sources)
		return sources, nil
	}

	// Keep the unit-level gate contract useful for isolated fixtures that do
	// not need a full component manifest. Real repositories always take the
	// manifest-backed path above.
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		for _, extension := range []string{"*.ts", "*.tsx"} {
			matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "versions", "*", extension))
			if err != nil {
				return nil, err
			}
			sources = append(sources, matches...)
		}
	}
	sort.Strings(sources)
	return sources, nil
}

func implementationSource(root, catalogID string) (string, string, bool, error) {
	paths := []string{}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		matches, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return "", "", false, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	for _, manifest := range paths {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return "", "", false, err
		}
		var doc struct {
			CatalogID string `json:"catalogId"`
			Latest    string `json:"latest"`
		}
		if json.Unmarshal(data, &doc) != nil || doc.CatalogID != catalogID {
			continue
		}
		if doc.Latest == "" {
			return manifest, "", false, nil
		}
		rootDir := filepath.Dir(manifest)
		versionDir := filepath.Join(rootDir, "versions", doc.Latest)
		source := filepath.Join(versionDir, filepath.Base(rootDir)+".tsx")
		if _, err := os.Stat(source); err != nil {
			matches := versionSources(versionDir)
			if len(matches) == 0 {
				versionDir = filepath.Join(rootDir, doc.Latest)
				matches = versionSources(versionDir)
			}
			if len(matches) == 0 {
				return manifest, "", false, nil
			}
			source = matches[0]
		}
		return manifest, source, true, nil
	}
	return "", "", false, nil
}

func versionSources(versionDir string) []string {
	var matches []string
	for _, extension := range []string{"*.tsx", "*.ts"} {
		found, _ := filepath.Glob(filepath.Join(versionDir, extension))
		matches = append(matches, found...)
	}
	sort.Strings(matches)
	return matches
}

var (
	pxValue          = regexp.MustCompile(`--space-[a-z0-9-]+\s*:\s*([0-9.]+)px`)
	literalDimension = regexp.MustCompile(`(?:\b(?:p|px|py|pt|pr|pb|pl|m|mx|my|mt|mr|mb|ml|gap|w|h)-[0-9]+(?:\.[0-9]+)?\b|\[[0-9.]+px\])`)
)

// ValidateTokens checks the shared ramp contract in every design kit and
// rejects non-grid spacing declarations.
func ValidateTokens(root string) (Result, error) {
	paths, err := filepath.Glob(filepath.Join(root, "templates", "design", "*", "adapters", "react-vite-tailwind", "tokens.css"))
	if err != nil {
		return Result{}, err
	}
	shared := []string{"space-3xs", "space-2xs", "space-xs", "space-sm", "space-md", "space-lg", "space-xl", "space-2xl", "text-display", "text-title", "text-heading", "text-body", "elev-flat", "elev-raised", "layer-base", "layer-modal", "dur-instant"}
	result := Result{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(data)
		kit := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
		for _, token := range shared {
			if !strings.Contains(text, "--"+token) {
				result.Findings = append(result.Findings, Finding{"catalog.tokens_missing", "foundations.tokens", fmt.Sprintf("%s does not declare shared token --%s", kit, token)})
			}
		}
		for _, match := range pxValue.FindAllStringSubmatch(text, -1) {
			value, _ := strconv.ParseFloat(match[1], 64)
			if int(value)%4 != 0 {
				result.Findings = append(result.Findings, Finding{"catalog.tokens_grid", "foundations.tokens", fmt.Sprintf("%s spacing token is not on the 4px grid: %spx", kit, match[1])})
			}
		}
	}
	sources, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		for _, match := range literalDimension.FindAllString(string(data), -1) {
			result.Findings = append(result.Findings, Finding{"catalog.tokens_literal", implementationName(path), fmt.Sprintf("implementation contains literal dimension %q; use a declared semantic token", match)})
		}
	}
	return nonEmpty(result, "tokens"), nil
}

// ValidateLifecycle performs conservative static checks over hook/service/
// adapter/generator sources. It deliberately prefers a finding over a green
// result when cleanup evidence is absent.
func ValidateLifecycle(root string) (Result, error) {
	result := Result{}
	paths, err := activeLibrarySources(root)
	if err != nil {
		return Result{}, err
	}
	for _, path := range paths {
		// Stories are browser-only specimens, not released runtime. Including
		// them here makes the lifecycle gate report demo timers and AbortSignal
		// listeners as component defects.
		if isStorySource(path) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		text := string(data)
		if strings.Contains(text, "addEventListener") && !strings.Contains(text, "removeEventListener") {
			result.Findings = append(result.Findings, Finding{"catalog.lifecycle_cleanup", implementationName(path), "adds an event listener without a matching removal"})
		}
		if strings.Contains(text, "new MutationObserver") && !strings.Contains(text, ".disconnect(") {
			result.Findings = append(result.Findings, Finding{"catalog.lifecycle_cleanup", implementationName(path), "creates an observer without disconnect cleanup"})
		}
		if hasBrowserAccessOutsideEffects(text) {
			result.Findings = append(result.Findings, Finding{"catalog.lifecycle_ssr", implementationName(path), "accesses a browser global without an SSR guard"})
		}
	}
	return nonEmpty(result, "lifecycle"), nil
}

func isStorySource(path string) bool {
	base := filepath.Base(path)
	return base == "story.ts" || base == "story.tsx"
}

// hasBrowserAccessOutsideEffects keeps the static SSR check conservative while
// understanding the one React lifecycle boundary that is guaranteed not to
// execute during server rendering. Browser access in render, module scope, or
// an arbitrary exported callback still requires an explicit guard.
func hasBrowserAccessOutsideEffects(text string) bool {
	remaining := []byte(text)
	for _, start := range effectCallbackRanges(text) {
		for index := start[0]; index < start[1] && index < len(remaining); index++ {
			remaining[index] = ' '
		}
	}
	textWithoutEffects := string(remaining)
	return (strings.Contains(textWithoutEffects, "window.") && !strings.Contains(textWithoutEffects, "typeof window")) ||
		(strings.Contains(textWithoutEffects, "document.") && !strings.Contains(textWithoutEffects, "typeof document"))
}

func effectCallbackRanges(text string) [][2]int {
	var ranges [][2]int
	for offset := 0; offset < len(text); {
		match := strings.Index(text[offset:], "useEffect")
		if match < 0 {
			break
		}
		start := offset + match
		after := start + len("useEffect")
		if after < len(text) && isIdentifierPart(text[after]) {
			offset = after
			continue
		}
		for after < len(text) && (text[after] == ' ' || text[after] == '\n' || text[after] == '\r' || text[after] == '\t') {
			after++
		}
		if after >= len(text) || text[after] != '(' {
			offset = after
			continue
		}
		arrow := strings.Index(text[after:], "=>")
		if arrow < 0 {
			break
		}
		arrow += after + 2
		body := arrow
		for body < len(text) && (text[body] == ' ' || text[body] == '\n' || text[body] == '\r' || text[body] == '\t') {
			body++
		}
		if body >= len(text) || text[body] != '{' {
			offset = arrow
			continue
		}
		end, ok := matchingBrace(text, body)
		if !ok {
			break
		}
		ranges = append(ranges, [2]int{body, end + 1})
		offset = end + 1
	}
	return ranges
}

func matchingBrace(text string, open int) (int, bool) {
	depth := 0
	for index := open; index < len(text); index++ {
		switch text[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index, true
			}
		}
	}
	return 0, false
}

func isIdentifierPart(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '_'
}

func implementationName(path string) string {
	return filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
}

func ValidateFixtures(root string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.Asset.Kind != "fixture" || asset.Fixture == nil {
			continue
		}
		result.Inspected++
		if len(asset.Fixture.DataShapes) == 0 {
			result.Findings = append(result.Findings, Finding{"catalog.fixture_adversarial", asset.Asset.ID, "fixture declares no adversarial data shapes"})
		}
		if !contains(asset.Fixture.DataShapes, "failure") && !contains(asset.Fixture.DataShapes, "overflow") {
			result.Findings = append(result.Findings, Finding{"catalog.fixture_adversarial", asset.Asset.ID, "fixture must include failure or overflow data"})
		}
		if asset.Fixture.Satisfies != nil && asset.Fixture.Satisfies.Capability == "data-source" && len(asset.Fixture.Satisfies.TypeArguments) == 0 {
			result.Findings = append(result.Findings, Finding{"catalog.fixture_data_source", asset.Asset.ID, "data-source fixture must declare a type argument"})
		}
	}
	return nonEmpty(result, "fixture_adversarial"), nil
}

// ValidateExamples checks that renderable assets have a public story contract
// beside their released source. Enum completeness is validated by the story
// contract parser in the registry; this gate owns the filesystem-level
// requirement so coverage never promotes a primitive with no specimen.
func ValidateExamples(root string) (Result, error) {
	result := Result{}
	for _, kind := range []string{"components", "primitives"} {
		manifests, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return Result{}, err
		}
		sort.Strings(manifests)
		for _, manifestPath := range manifests {
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return Result{}, err
			}
			var manifest struct {
				Latest string `json:"latest"`
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return Result{}, err
			}
			result.Inspected++
			storyPath := filepath.Join(filepath.Dir(manifestPath), "versions", manifest.Latest, "story.json")
			if _, err := os.Stat(storyPath); err != nil {
				if os.IsNotExist(err) {
					result.Findings = append(result.Findings, Finding{"catalog.examples_missing", filepath.Base(filepath.Dir(manifestPath)), "released renderable asset has no story.json specimen"})
					continue
				}
				return Result{}, err
			}
		}
	}
	return nonEmpty(result, "examples"), nil
}

// ValidateReducedMotion checks the source contract rather than inferring
// support from a component's existence. Motion-bearing implementations must
// declare their reduced-motion behavior; components without motion need no
// special override and pass this gate after inspection.
func ValidateReducedMotion(root string) (Result, error) {
	motionDeclaration := regexp.MustCompile(`(?m)(?:^|[;{\s])(?:transition|animation|transform)\s*:`)
	return validateActiveSources(root, "reduced-motion", func(asset assetDoc, source string) string {
		if !motionDeclaration.MatchString(source) {
			return ""
		}
		if strings.Contains(source, "prefers-reduced-motion") || strings.Contains(source, "useReducedMotion") || strings.Contains(source, "reducedMotion") {
			return ""
		}
		return "motion-bearing source has no reduced-motion branch"
	})
}

// ValidateRTL rejects physical horizontal CSS declarations in active source.
// Logical properties are the shared library's direction-safe contract.
func ValidateRTL(root string) (Result, error) {
	physical := regexp.MustCompile(`(?i)(?:margin|padding|inset|border)-(?:left|right)\s*:|(?:margin|padding)(?:Left|Right)\s*:`)
	return validateActiveSources(root, "rtl", func(asset assetDoc, source string) string {
		if physical.MatchString(source) {
			return "active source contains a physical left/right declaration"
		}
		return ""
	})
}

// ValidateStress requires every active renderable implementation to have an
// indexed story contract. The story contract is the stress fixture boundary:
// it is where long, empty, disabled, and large-value specimens are declared
// and version-pinned for the browser runner.
func ValidateStress(root string) (Result, error) {
	return validateActiveSources(root, "stress", func(asset assetDoc, source string) string {
		_ = source
		manifest, _, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil || !ok {
			return "active implementation is not available to the stress runner"
		}
		data, readErr := os.ReadFile(manifest)
		if readErr != nil {
			return "implementation manifest could not be read"
		}
		var doc struct {
			Latest string `json:"latest"`
		}
		if json.Unmarshal(data, &doc) != nil || doc.Latest == "" {
			return "implementation manifest has no released version"
		}
		storyPath := filepath.Join(filepath.Dir(manifest), "versions", doc.Latest, "story.json")
		story, readErr := os.ReadFile(storyPath)
		if readErr != nil || len(bytes.TrimSpace(story)) == 0 {
			return "released implementation has no non-empty story contract"
		}
		return ""
	})
}

// ValidatePerformance executes the same production build boundary used by
// the scenario lifecycle. It is intentionally corpus-level: a successful
// immutable build proves the selected source can be bundled by the target,
// while bundle-budget policy remains a separate diagnostic.
func ValidatePerformance(root string) (Result, error) {
	uiDir := filepath.Join(root, "scenarios", "react-component-library", "ui")
	result := Result{Inspected: countCatalogSources(root)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "pnpm", "run", "build")
	command.Dir = uiDir
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		result.Findings = append(result.Findings, Finding{Code: "catalog.performance_timeout", AssetID: "catalog.runner", Message: "production build timed out before the performance gate completed"})
	} else if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 4000 {
			message = message[len(message)-4000:]
		}
		result.Findings = append(result.Findings, Finding{Code: "catalog.performance_failed", AssetID: "catalog.runner", Message: "production build failed: " + message})
	}
	return nonEmpty(result, "performance"), nil
}

// ValidateIntegration checks the source-level integration boundary shared by
// every released renderable asset. The actual manager/browser integration is
// recorded by component-test and Experience Manager evidence; this runner
// prevents a source-only asset from receiving an integration pass.
func ValidateIntegration(root string) (Result, error) {
	return validateActiveSources(root, "integration", func(asset assetDoc, source string) string {
		if strings.TrimSpace(source) == "" {
			return "released integration source is empty"
		}
		// The active component manifest supplies the exact released version;
		// source identity may use the library marker or the established
		// adoption-facade marker. Both are valid integration boundaries, while
		// an unowned source is not.
		if !strings.Contains(source, "@libraryId") && !strings.Contains(source, "@vrooliComponentSource") {
			return "released source has no library or adoption identity metadata"
		}
		return ""
	})
}

func validateActiveSources(root, gate string, check func(asset assetDoc, source string) string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, asset := range assets {
		if asset.Asset.Kind != "component" && asset.Asset.Kind != "navigation" && asset.Asset.Kind != "primitive" && asset.Asset.Kind != "pattern" && asset.Asset.Kind != "page-template" {
			continue
		}
		_, source, ok, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if !ok {
			continue
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		if message := check(asset, string(data)); message != "" {
			result.Findings = append(result.Findings, Finding{Code: "catalog." + gate, AssetID: asset.Asset.ID, Message: message})
		}
	}
	return nonEmpty(result, gate), nil
}

func nonEmpty(result Result, gate string) Result {
	if result.Inspected == 0 {
		result.Findings = append(result.Findings, Finding{
			Code:    "catalog." + gate + "_zero_inspected",
			AssetID: "catalog.runner",
			Message: "gate inspected zero inputs; runner configuration is stale or broken",
		})
	}
	return result
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
