// Package preview is the application-layer home for the live-preview
// pipeline. It is stateless — no SQL, no migrations — so it ships only
// a Service interface plus an esbuild-backed Bundler implementation.
//
// The service depends on the components Service for source resolution
// (id → on-disk TSX content), which is also where the path-traversal
// guard and registry NotFound semantics live. Everything preview adds
// is the transpile step.
package preview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
}

type service struct {
	components components.Service
	bundler    Bundler
	deps       deps.Service
}

// NewService wires the components service (for resolution + content)
// and the bundler. Both are required — there is no degraded mode.
func NewService(comp components.Service, bundler Bundler) Service {
	return &service{components: comp, bundler: bundler}
}

func NewServiceWithDeps(comp components.Service, bundler Bundler, depsSvc deps.Service) Service {
	return &service{components: comp, bundler: bundler, deps: depsSvc}
}

func (s *service) GetBundle(ctx context.Context, id string) (Bundle, error) {
	return s.GetBundleVersion(ctx, id, "")
}

func (s *service) GetBundleVersion(ctx context.Context, id, version string) (Bundle, error) {
	version = strings.TrimSpace(version)
	asset, err := s.components.Get(ctx, id)
	if err != nil {
		return Bundle{}, err
	}
	if asset.AssetKind == components.AssetKindHook {
		return Bundle{}, ErrNotRenderable{AssetID: asset.ID, LibraryID: asset.LibraryID}
	}
	// Resolve the complete pinned asset graph before bundling. Relative imports
	// remain bundled from the catalog source tree, while this explicit check
	// prevents previewing a component whose declared hook/component closure is
	// missing or cyclic.
	if len(asset.Dependencies) > 0 {
		if _, err := components.ResolveDependencyClosure(ctx, s.components, id, version); err != nil {
			return Bundle{}, err
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
	}
	js, warnings, err := s.bundler.BuildBundle(ctx, content.Body, content.SourcePath)
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{
		JS:           js,
		SourcePath:   content.SourcePath,
		SHA256:       digest(js),
		Warnings:     warnings,
		Dependencies: declarations,
	}, nil
}

// ErrNotRenderable makes hook preview rejection explicit. Hooks are
// first-class assets but do not have a React render surface; callers should
// inspect their versioned source and adopt them through a component closure.
type ErrNotRenderable struct {
	AssetID   string
	LibraryID string
}

func (e ErrNotRenderable) Error() string {
	return fmt.Sprintf("asset %q is a hook and cannot be previewed; inspect its source or select a consuming component", e.LibraryID)
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
