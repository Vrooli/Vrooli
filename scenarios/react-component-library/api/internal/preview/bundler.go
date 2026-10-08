package preview

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
			Sourcefile: filepath.Join(resolveDir, filepath.Base(sourcePath)),
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
				build.OnLoad(esbuild.OnLoadOptions{Filter: `\.css$`}, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
					css, err := os.ReadFile(args.Path)
					if err != nil {
						return esbuild.OnLoadResult{}, err
					}
					key := fmt.Sprintf("rcl-asset-style-%x", sha256.Sum256([]byte(filepath.Clean(args.Path))))
					module := fmt.Sprintf(`const css = %s;
const key = %q;
if (typeof document !== "undefined" && !document.querySelector("style[data-rcl-asset-style=\"" + key + "\"]")) {
  const style = document.createElement("style");
  style.setAttribute("data-rcl-asset-style", key);
  style.textContent = css;
  document.head.appendChild(style);
}
export default css;
`, stringQuote(string(css)), key)
					return esbuild.OnLoadResult{Contents: &module, Loader: esbuild.LoaderJS}, nil
				})
				build.OnLoad(esbuild.OnLoadOptions{Filter: `\.(?:ts|tsx)$`}, func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
					source, err := os.ReadFile(args.Path)
					if err != nil {
						return esbuild.OnLoadResult{}, err
					}
					content := string(source)
					loader := esbuild.LoaderTSX
					if strings.HasSuffix(args.Path, ".ts") {
						loader = esbuild.LoaderTS
					}
					return esbuild.OnLoadResult{Contents: &content, Loader: loader}, nil
				})
				build.OnResolve(esbuild.OnResolveOptions{Filter: ".*"}, func(args esbuild.OnResolveArgs) (esbuild.OnResolveResult, error) {
					importPath, err := pinCatalogImport(args.Path, args.Importer)
					if err != nil {
						return esbuild.OnResolveResult{}, err
					}
					if localPath, ok := resolveLocalVrooliPackage(importPath, resolveDir); ok {
						return esbuild.OnResolveResult{Path: localPath}, nil
					}
					if strings.HasPrefix(importPath, "@vrooli/react-component-library/") {
						return esbuild.OnResolveResult{}, fmt.Errorf("catalog source unavailable for %s", importPath)
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
	jsOutputs := make([]esbuild.OutputFile, 0, len(result.OutputFiles))
	for _, output := range result.OutputFiles {
		if output.Path == "<stdout>" || strings.HasSuffix(strings.ToLower(output.Path), ".js") {
			jsOutputs = append(jsOutputs, output)
		}
	}
	if len(jsOutputs) != 1 {
		return "", warns, ErrBundle{SourcePath: sourcePath, Messages: []string{fmt.Sprintf("bundle produced %d JavaScript outputs", len(jsOutputs))}}
	}
	return string(jsOutputs[0].Contents), warns, nil
}

func stringQuote(value string) string {
	quoted := fmt.Sprintf("%q", value)
	return quoted
}

// resolveLocalVrooliPackage keeps preview artifacts self-contained for
// governed repo packages such as @vrooli/audio-capture-browser. These are
// source packages, not third-party runtime-store entries: bundling their
// source preserves the same preview contract as a relative companion import
// while leaving their React peer external for the harness import map.
func resolveLocalVrooliPackage(importPath, startDir string) (string, bool) {
	const componentLibraryPrefix = "@vrooli/react-component-library/"
	if strings.HasPrefix(importPath, componentLibraryPrefix) {
		parts := strings.Split(strings.TrimPrefix(importPath, componentLibraryPrefix), "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
			return "", false
		}
		// Preview evidence follows authored catalog bytes. An unrelated package
		// build may be stale even when the selected version ledger is current.

		libraryRoot := ""
		for dir := startDir; dir != ""; dir = filepath.Dir(dir) {
			canonical := filepath.Join(dir, "scenarios", "react-component-library", "library")
			if info, err := os.Stat(canonical); err == nil && info.IsDir() {
				libraryRoot = canonical
				break
			}
			if filepath.Base(dir) == "library" {
				libraryRoot = dir
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
		if libraryRoot == "" {
			return "", false
		}
		if candidate, ok := versionedAssetFile(libraryRoot, parts[0], parts[1], []string{"components", "foundations", "hooks", "primitives", "services"}, []string{".tsx", ".ts", ".jsx", ".js"}); ok {
			return candidate, true
		}
		return "", false
	}

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

func versionedAssetFile(root, name, version string, kinds, extensions []string) (string, bool) {
	for _, kind := range kinds {
		assetRoot := filepath.Join(root, kind, name, "versions", version)
		if candidate, ok := assetFile(assetRoot, name, extensions); ok {
			return candidate, true
		}
	}
	return "", false
}

func assetFile(root, name string, extensions []string) (string, bool) {
	for _, extension := range extensions {
		candidate := filepath.Join(root, name+extension)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
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

// pinCatalogImport resolves a published major import using the importing version's
// ledger. Companions inherit the ledger of their enclosing version directory.
func pinCatalogImport(specifier, importer string) (string, error) {
	const prefix = "@vrooli/react-component-library/"
	if !strings.HasPrefix(specifier, prefix) {
		return specifier, nil
	}
	parts := strings.Split(strings.TrimPrefix(specifier, prefix), "/")
	if len(parts) != 2 {
		return specifier, nil
	}
	major, err := strconv.Atoi(parts[1])
	if err != nil {
		return specifier, nil
	}
	for dir := filepath.Dir(importer); dir != "." && dir != ""; dir = filepath.Dir(dir) {
		if filepath.Base(filepath.Dir(dir)) == "versions" {
			raw, err := os.ReadFile(filepath.Join(dir, "dependencies.json"))
			if os.IsNotExist(err) {
				return "", fmt.Errorf("dependency ledger missing for %s", importer)
			}
			if err != nil {
				return "", err
			}
			var ledger struct {
				Dependencies []struct {
					LibraryID string `json:"libraryId"`
					Major     int    `json:"major"`
					Observed  string `json:"observed"`
				} `json:"dependencies"`
			}
			if err := json.Unmarshal(raw, &ledger); err != nil {
				return "", fmt.Errorf("read catalog dependency ledger: %w", err)
			}
			for _, entry := range ledger.Dependencies {
				if entry.LibraryID == "react-component-library:"+parts[0] && entry.Major == major {
					if !strings.HasPrefix(entry.Observed, strconv.Itoa(major)+".") || !compositionVersion.MatchString(entry.Observed) {
						return "", fmt.Errorf("invalid recorded version for %s", specifier)
					}
					return prefix + parts[0] + "/" + entry.Observed, nil
				}
			}
			return "", fmt.Errorf("dependency ledger for %s does not record %s", importer, specifier)
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	return specifier, nil
}
