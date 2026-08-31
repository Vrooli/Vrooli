// Package preview is the application-layer home for the live-preview
// pipeline. It is stateless — no SQL or schema writes — so it ships only
// a Service interface plus an esbuild-backed Bundler implementation.
//
// The service depends on the components Service for source resolution
// (id → on-disk TSX content), which is also where the path-traversal
// guard and registry NotFound semantics live. Everything preview adds
// is the transpile step.
package preview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/deps"
)

// Bundle is the wire-shape-equivalent output of GetBundle.
type Bundle struct {
	// JS is the ES-module text after esbuild has transformed the TSX
	// source. Imports react / react-dom / react-dom/client are kept
	// as bare specifiers; the harness HTML resolves them via importmap.
	JS string
	// HarnessJS is an optional separately bundled story.tsx module. Keeping it
	// separate preserves the component entry's export selection and lets a
	// harness import the version-local component normally.
	HarnessJS            string
	CompositionHarnessJS string
	// FrameJS is the optional catalog frame module used to render a subject in
	// context. It is kept separate from JS so the subject's adoption closure
	// never acquires the frame asset.
	FrameJS         string
	FrameAsset      string
	FrameRegion     string
	FixtureJSON     string
	FrameSourcePath string

	// SourcePath is echoed from the components service so callers can
	// render a heading without a second lookup.
	SourcePath string

	// SHA256 is the hex digest of JS. The host UI uses it as a cache-
	// buster on the iframe src so saves trigger a reload.
	SHA256 string

	// Warnings carries non-fatal diagnostics from the bundler. Fatal
	// errors surface as the typed bundler error from BuildBundle.
	Warnings []string

	Dependencies []deps.Declaration
}

// Bundler is the seam over esbuild. Production wires Esbuilder; tests
// inject a fake that returns deterministic strings.
type Bundler interface {
	BuildBundle(ctx context.Context, tsx string, sourcePath string) (js string, warnings []string, err error)
}

// Service is the application surface. GetBundle resolves the
// component, reads its content, then bundles.
type Service interface {
	GetBundle(ctx context.Context, id string) (Bundle, error)
	// GetBundleVersion compiles an immutable catalog version. An empty version
	// preserves GetBundle's editable-current-content behavior.
	GetBundleVersion(ctx context.Context, id, version string) (Bundle, error)
	GetBundleVersionWithFrame(ctx context.Context, id, version string, frame *components.StoryFrame) (Bundle, error)
}

type service struct {
	components components.Service
	bundler    Bundler
	deps       deps.Service
	repoRoot   string
}

// NewService wires the components service (for resolution + content)
// and the bundler. Both are required — there is no degraded mode.
func NewService(comp components.Service, bundler Bundler) Service {
	return &service{components: comp, bundler: bundler}
}

func NewServiceWithDeps(comp components.Service, bundler Bundler, depsSvc deps.Service) Service {
	return &service{components: comp, bundler: bundler, deps: depsSvc}
}

func NewServiceWithDepsAtRoot(comp components.Service, bundler Bundler, depsSvc deps.Service, repoRoot string) Service {
	return &service{components: comp, bundler: bundler, deps: depsSvc, repoRoot: strings.TrimSpace(repoRoot)}
}

func (s *service) GetBundle(ctx context.Context, id string) (Bundle, error) {
	return s.GetBundleVersion(ctx, id, "")
}

func (s *service) GetBundleVersionWithFrame(ctx context.Context, id, version string, frame *components.StoryFrame) (Bundle, error) {
	return s.GetBundleVersionWithFrameAndHarness(ctx, id, version, frame, nil)
}

func (s *service) GetBundleVersionWithFrameAndHarness(ctx context.Context, id, version string, frame *components.StoryFrame, harness *components.StoryHarnessRef) (Bundle, error) {
	bundle, err := s.GetBundleVersion(ctx, id, version)
	if err != nil {
		return bundle, err
	}
	var frameDeps []deps.Declaration
	if frame != nil {
		frameJS, fixtureJSON, sourcePath, resolvedDeps, frameErr := s.bundleFrame(ctx, frame)
		if frameErr != nil {
			return Bundle{}, frameErr
		}
		bundle.FrameJS = frameJS
		bundle.FrameAsset = frame.Asset
		bundle.FrameRegion = frame.Region
		bundle.FixtureJSON = fixtureJSON
		bundle.FrameSourcePath = sourcePath
		frameDeps = resolvedDeps
	}
	if harness != nil {
		compositionHarness, compositionErr := s.bundleCompositionHarness(ctx, harness)
		if compositionErr != nil {
			return Bundle{}, compositionErr
		}
		bundle.CompositionHarnessJS = compositionHarness
	}
	bundle.Dependencies = appendUniqueDeclarations(bundle.Dependencies, frameDeps)
	// Include the complete declarative composition in the digest. A capability
	// change, harness config change, or export change must produce a distinct
	// preview artifact even when the generated JavaScript happens to be the
	// same. This is the immutable composition boundary used by evidence and
	// screenshot manifests.
	composition, _ := json.Marshal(struct {
		Frame   *components.StoryFrame      `json:"frame,omitempty"`
		Harness *components.StoryHarnessRef `json:"harness,omitempty"`
	}{Frame: frame, Harness: harness})
	bundle.SHA256 = digest(bundle.JS + bundle.HarnessJS + bundle.CompositionHarnessJS + bundle.FrameJS + bundle.FixtureJSON + string(composition))
	return bundle, nil
}

type compositionHarnessFamily struct {
	ID         string   `json:"id"`
	Version    string   `json:"version"`
	Export     string   `json:"export"`
	ConfigKeys []string `json:"configKeys"`
}

type compositionHarnessRegistry struct {
	Families []compositionHarnessFamily `json:"families"`
}

func (s *service) bundleCompositionHarness(ctx context.Context, harness *components.StoryHarnessRef) (string, error) {
	if harness == nil || strings.TrimSpace(s.repoRoot) == "" {
		return "", frameBundleError("composition.harness", "composition harness repository root is not configured")
	}
	parts := strings.Split(strings.TrimSpace(harness.Asset), ".")
	if len(parts) != 2 || parts[0] != "preview" || parts[1] == "" || strings.ContainsAny(harness.Asset+harness.Version+harness.Export, `/\\`) || strings.TrimSpace(harness.Version) == "" || strings.TrimSpace(harness.Export) == "" {
		return "", frameBundleError(harness.Asset, "invalid composition harness reference")
	}
	manifestPath := filepath.Join(s.repoRoot, "scenarios", "react-component-library", "harnesses", "manifest.json")
	manifestBytes, manifestErr := os.ReadFile(manifestPath)
	if manifestErr != nil {
		return "", frameBundleError(harness.Asset, "composition harness registry was not found")
	}
	var manifest compositionHarnessRegistry
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return "", frameBundleError(harness.Asset, "composition harness registry is invalid")
	}
	var registered *compositionHarnessFamily
	for index := range manifest.Families {
		family := &manifest.Families[index]
		if family.ID == parts[1] && family.Version == harness.Version && family.Export == harness.Export {
			registered = family
			break
		}
	}
	if registered == nil {
		return "", frameBundleError(harness.Asset, "composition harness is not registered at the requested version/export")
	}
	if err := validateCompositionHarnessConfig(harness.Config, registered.ConfigKeys); err != nil {
		return "", frameBundleError(harness.Asset, err.Error())
	}
	path := filepath.Join(s.repoRoot, "scenarios", "react-component-library", "harnesses", parts[1], "versions", harness.Version, harness.Export+".tsx")
	root := filepath.Join(s.repoRoot, "scenarios", "react-component-library", "harnesses")
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil || (cleanPath != cleanRoot && !strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator))) {
		return "", frameBundleError(harness.Asset, "composition harness path escapes configured root")
	}
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", frameBundleError(harness.Asset, "composition harness implementation was not found")
	}
	if strings.Contains(string(content), "library/components/") || strings.Contains(string(content), "library/primitives/") || strings.Contains(string(content), "library/hooks/") {
		return "", frameBundleError(harness.Asset, "composition harness must not import a component-specific library asset")
	}
	// Use the validated absolute path as the bundler source so relative imports
	// between generic Preview foundations resolve from the harness directory.
	// The path remains inside the registry root checked above.
	js, _, err := s.bundler.BuildBundle(ctx, string(content), cleanPath)
	if err != nil {
		return "", err
	}
	return js, nil
}

func validateCompositionHarnessConfig(raw json.RawMessage, allowed []string) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("composition harness config must be a JSON object")
	}
	keys := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		keys[key] = struct{}{}
	}
	for key := range config {
		if _, ok := keys[key]; !ok {
			return fmt.Errorf("composition harness config key %q is not declared by the registry", key)
		}
	}
	return nil
}

func appendUniqueDeclarations(base, extra []deps.Declaration) []deps.Declaration {
	seen := make(map[string]struct{}, len(base))
	for _, declaration := range base {
		seen[declaration.DepName+"\x00"+declaration.VersionRange] = struct{}{}
	}
	for _, declaration := range extra {
		key := declaration.DepName + "\x00" + declaration.VersionRange
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		base = append(base, declaration)
	}
	return base
}

func (s *service) GetBundleVersion(ctx context.Context, id, version string) (Bundle, error) {
	version = strings.TrimSpace(version)
	asset, err := s.components.Get(ctx, id)
	if err != nil {
		return Bundle{}, err
	}
	// Resolve the complete pinned asset graph before bundling. Relative imports
	// remain bundled from the catalog source tree, while this explicit check
	// prevents previewing a component whose declared hook/component closure is
	// missing or cyclic.
	var closure []components.ResolvedAsset
	if len(asset.Dependencies) > 0 {
		var closureErr error
		closure, closureErr = components.ResolveDependencyClosure(ctx, s.components, id, version)
		if closureErr != nil {
			return Bundle{}, closureErr
		}
	}
	var (
		content    components.Content
		contentErr error
	)
	if version == "" {
		content, contentErr = s.components.GetContent(ctx, id)
	} else {
		content, contentErr = s.components.GetVersionContent(ctx, id, version)
	}
	if contentErr != nil {
		return Bundle{}, contentErr
	}
	var declarations []deps.Declaration
	if s.deps != nil {
		dependencyVersion := version
		if dependencyVersion == "" {
			dependencyVersion = asset.LatestVersion
		}
		declarations, err = s.deps.ListForComponentVersion(ctx, id, dependencyVersion)
		if err != nil {
			return Bundle{}, err
		}
		versionRecord, versionErr := s.components.GetVersion(ctx, id, dependencyVersion)
		if versionErr != nil {
			return Bundle{}, versionErr
		}
		versionFiles := []components.ComponentVersionFile{}
		versionFiles = append(versionFiles, versionRecord.Files...)
		for _, resolved := range closure {
			versionFiles = append(versionFiles, resolved.Version.Files...)
		}
		for _, file := range versionFiles {
			fields, parseErr := deps.ParseSourceDeclarations(file.Content)
			if parseErr != nil {
				return Bundle{}, fmt.Errorf("parse preview dependencies in %s: %w", file.Path, parseErr)
			}
			for _, field := range fields {
				declarations = appendUniqueDeclarations(declarations, []deps.Declaration{{
					ComponentID:  id,
					Version:      dependencyVersion,
					DepName:      field.DepName,
					VersionRange: field.VersionRange,
					Kind:         field.Kind,
				}})
			}
		}
		importedFields, scanErr := s.scanImportedSourceDeclarations(versionFiles, content.SourcePath)
		if scanErr != nil {
			return Bundle{}, scanErr
		}
		for _, field := range importedFields {
			declarations = appendUniqueDeclarations(declarations, []deps.Declaration{{
				ComponentID:  id,
				Version:      dependencyVersion,
				DepName:      field.DepName,
				VersionRange: field.VersionRange,
				Kind:         field.Kind,
			}})
		}
	}
	stampVersion := version
	if stampVersion == "" {
		stampVersion = asset.LatestVersion
	}
	stampedSource := stampPreviewSource(content.Body, content.SourcePath, asset.CatalogID, stampVersion)
	js, warnings, err := s.bundler.BuildBundle(ctx, stampedSource, content.SourcePath)
	if err != nil {
		return Bundle{}, err
	}
	var harnessJS string
	if storyReader, ok := s.components.(interface {
		GetVersionContentAt(context.Context, string, string, string) (components.Content, error)
	}); ok && version != "" {
		storyContent, storyErr := storyReader.GetVersionContentAt(ctx, id, version, "story.tsx")
		if storyErr == nil {
			if err := validateSpecimenSource(storyContent.Body); err != nil {
				return Bundle{}, err
			}
			var storyWarnings []string
			harnessJS, storyWarnings, storyErr = s.bundler.BuildBundle(ctx, storyContent.Body, storyContent.SourcePath)
			warnings = append(warnings, storyWarnings...)
			if storyErr != nil {
				return Bundle{}, storyErr
			}
		} else if !isMissingStoryArtifact(storyErr) {
			return Bundle{}, storyErr
		}
	}
	return Bundle{
		JS:           js,
		HarnessJS:    harnessJS,
		SourcePath:   content.SourcePath,
		SHA256:       digest(js + harnessJS),
		Warnings:     warnings,
		Dependencies: declarations,
	}, nil
}

// validateSpecimenSource keeps Preview specimens deterministic and local. A
// story may use browser primitives for presentation, but it may not turn the
// preview route into a production-data client or mutate browser persistence.
// The check is intentionally conservative and runs before esbuild so the
// failure is reported as a specimen diagnostic rather than a confusing blank
// harness.
func validateSpecimenSource(source string) error {
	patterns := []struct {
		needle string
		detail string
	}{
		{"fetch(", "network fetch is not allowed in Preview specimens; use a deterministic fixture"},
		{"XMLHttpRequest", "XMLHttpRequest is not allowed in Preview specimens"},
		{"WebSocket", "WebSocket is not allowed in Preview specimens"},
		{"localStorage", "localStorage is not allowed in Preview specimens"},
		{"sessionStorage", "sessionStorage is not allowed in Preview specimens"},
		{"indexedDB", "indexedDB is not allowed in Preview specimens"},
		{"document.cookie", "document.cookie is not allowed in Preview specimens"},
		{"child_process", "process access is not allowed in Preview specimens"},
		{"node:fs", "filesystem access is not allowed in Preview specimens"},
		{"node:http", "network access is not allowed in Preview specimens"},
		{"process.", "process access is not allowed in Preview specimens"},
		{"Deno.", "Deno access is not allowed in Preview specimens"},
	}
	for _, pattern := range patterns {
		if strings.Contains(source, pattern.needle) {
			return ErrBundle{SourcePath: "story.tsx", Messages: []string{pattern.detail}}
		}
	}
	return nil
}

func isMissingStoryArtifact(err error) bool {
	return errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "file does not exist")
}

// ErrBundle wraps an esbuild failure with the diagnostics esbuild
// emitted. Mapped to InvalidArgument by the handler — the file on
// disk has a syntax error or unresolvable import the caller can fix.
type ErrBundle struct {
	SourcePath string
	Messages   []string
}

func (e ErrBundle) Error() string {
	return fmt.Sprintf("bundle %q failed: %v", e.SourcePath, e.Messages)
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
