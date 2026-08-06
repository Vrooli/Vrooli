package preview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

// Esbuilder is the production Bundler. It uses esbuild's Build API so
// relative imports are followed and folded into the component module.
// Bare package imports stay external; the harness import map is the
// per-preview isolation boundary for React and declared dependencies.
type Esbuilder struct {
	root string
}

// NewEsbuilder constructs the production Bundler.
func NewEsbuilder() *Esbuilder { return &Esbuilder{} }

// NewEsbuilderWithRoot constructs a Bundler pinned to a component
// source root. Tests use this to exercise relative import resolution
// without depending on the repository layout.
func NewEsbuilderWithRoot(root string) *Esbuilder {
	return &Esbuilder{root: root}
}

// BuildBundle bundles TSX → ES module text. sourcePath is used as the
// stdin Sourcefile and to derive ResolveDir, so diagnostics reference
// the component file and relative imports resolve beside it.
func (b Esbuilder) BuildBundle(_ context.Context, tsx string, sourcePath string) (string, []string, error) {
	resolveDir, err := b.resolveDir(sourcePath)
	if err != nil {
		return "", nil, ErrBundle{SourcePath: sourcePath, Messages: []string{err.Error()}}
	}
	result := esbuild.Build(esbuild.BuildOptions{
		Stdin: &esbuild.StdinOptions{
			Contents:   tsx,
			Sourcefile: sourcePath,
			ResolveDir: resolveDir,
			Loader:     esbuild.LoaderTSX,
		},
		Bundle:   true,
		Format:   esbuild.FormatESModule,
		Target:   esbuild.ES2020,
		JSX:      esbuild.JSXAutomatic,
		Platform: esbuild.PlatformBrowser,
		Loader: map[string]esbuild.Loader{
			".js":  esbuild.LoaderJS,
			".jsx": esbuild.LoaderJSX,
			".ts":  esbuild.LoaderTS,
			".tsx": esbuild.LoaderTSX,
		},
		Define: map[string]string{
			"process.env.NODE_ENV": `"development"`,
		},
		Write:    false,
		LogLevel: esbuild.LogLevelSilent,
		Plugins: []esbuild.Plugin{{
			Name: "preview-bare-imports",
			Setup: func(build esbuild.PluginBuild) {
				build.OnResolve(esbuild.OnResolveOptions{Filter: ".*"}, func(args esbuild.OnResolveArgs) (esbuild.OnResolveResult, error) {
					if localPath, ok := resolveLocalVrooliPackage(args.Path, resolveDir); ok {
						return esbuild.OnResolveResult{Path: localPath}, nil
					}
					if isBareImport(args.Path) {
						return esbuild.OnResolveResult{Path: args.Path, External: true}, nil
					}
					return esbuild.OnResolveResult{}, nil
				})
			},
		}},
	})
	if len(result.Errors) > 0 {
		msgs := make([]string, 0, len(result.Errors))
		for _, m := range result.Errors {
			msgs = append(msgs, m.Text)
		}
		return "", nil, ErrBundle{SourcePath: sourcePath, Messages: msgs}
	}
	warns := make([]string, 0, len(result.Warnings))
	for _, m := range result.Warnings {
		warns = append(warns, m.Text)
	}
	if len(result.OutputFiles) == 0 {
		return "", warns, ErrBundle{SourcePath: sourcePath, Messages: []string{"bundle produced no output"}}
	}
	return string(result.OutputFiles[0].Contents), warns, nil
}

// resolveLocalVrooliPackage keeps preview artifacts self-contained for
// governed repo packages such as @vrooli/audio-capture-browser. These are
// source packages, not third-party runtime-store entries: bundling their
// source preserves the same preview contract as a relative companion import
// while leaving their React peer external for the harness import map.
func resolveLocalVrooliPackage(importPath, startDir string) (string, bool) {
	const prefix = "@vrooli/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", false
	}
	packageName := strings.TrimPrefix(importPath, prefix)
	if packageName == "" || strings.Contains(packageName, "/") || strings.Contains(packageName, "..") {
		return "", false
	}
	for dir := startDir; dir != ""; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "packages", packageName, "src", "index.ts")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", false
}

func (b Esbuilder) resolveDir(sourcePath string) (string, error) {
	if filepath.IsAbs(sourcePath) {
		return filepath.Dir(sourcePath), nil
	}
	root := strings.TrimSpace(b.root)
	if root == "" {
		root = discoverComponentSourceRoot(sourcePath)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve component source root: %w", err)
	}
	return filepath.Join(absRoot, filepath.Dir(filepath.Clean(sourcePath))), nil
}

func discoverComponentSourceRoot(sourcePath string) string {
	candidates := []string{
		".",
		"library",
		"../library",
		"../../library",
		"scenarios/react-component-library/library",
	}
	for _, candidate := range candidates {
		full := filepath.Join(candidate, filepath.Clean(sourcePath))
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return "."
}

func isBareImport(path string) bool {
	return path != "" && !strings.HasPrefix(path, ".") && !strings.HasPrefix(path, "/")
}
