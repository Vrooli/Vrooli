package cliutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/vrooli/envkit-go"
)

// withEmbeddedFreshnessInputs conservatively resolves production embed directives
// among the declared sources. Literal paths preserve assets whose bytes or names
// resemble build products. Inactive platform files may add harmless extra inputs.
func withEmbeddedFreshnessInputs(spec FreshnessSpec) (FreshnessSpec, error) {
	root := strings.TrimSpace(spec.ContextRoot)
	if root == "" {
		root = spec.SourceRoot
	}
	var sources []string
	err := walkFreshnessInputs(spec, func(_, abs string, _ fs.FileInfo, _ func() bool) error {
		if strings.HasSuffix(abs, ".go") && !strings.HasSuffix(abs, "_test.go") {
			sources = append(sources, abs)
		}
		return nil
	})
	if err != nil {
		return spec, err
	}
	var assets []string
	activePackages := make(map[string]map[string]bool)
	explicit := explicitFreshnessFiles(spec)
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return spec, err
		}
		if !bytes.Contains(data, []byte("//go:embed")) {
			continue
		}
		var scan scanner.Scanner
		scan.Init(token.NewFileSet().AddFile(source, -1, len(data)), data, nil, scanner.ScanComments)
		for {
			_, kind, comment := scan.Scan()
			if kind == token.EOF {
				break
			}
			if kind != token.COMMENT || !strings.HasPrefix(comment, "//go:embed ") && !strings.HasPrefix(comment, "//go:embed\t") {
				continue
			}
			rest := strings.TrimSpace(strings.TrimPrefix(comment, "//go:embed"))
			for rest != "" {
				var pattern string
				if rest[0] == '"' || rest[0] == '`' {
					quoted, err := strconv.QuotedPrefix(rest)
					if err != nil {
						return spec, fmt.Errorf("embed directive in %s: %w", source, err)
					}
					pattern, err = strconv.Unquote(quoted)
					if err != nil {
						return spec, err
					}
					rest = rest[len(quoted):]
				} else {
					n := strings.IndexFunc(rest, unicode.IsSpace)
					if n < 0 {
						n = len(rest)
					}
					pattern, rest = rest[:n], rest[n:]
				}
				rest = strings.TrimSpace(rest)
				includeHidden := strings.HasPrefix(pattern, "all:")
				pattern = strings.TrimPrefix(pattern, "all:")
				if !fs.ValidPath(pattern) || pattern == "." {
					return spec, fmt.Errorf("invalid embed pattern %q in %s", pattern, source)
				}
				globPattern := filepath.Join(filepath.Dir(source), filepath.FromSlash(pattern))
				matches, err := filepath.Glob(globPattern)
				if err != nil {
					return spec, err
				}
				spec.observer.globbed(globPattern)
				for _, match := range matches {
					err = filepath.WalkDir(match, func(file string, entry fs.DirEntry, err error) error {
						if err != nil {
							return err
						}
						if entry.Type()&os.ModeSymlink != 0 {
							return nil
						}
						name := entry.Name()
						if file != match && (!includeHidden && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) || name == ".git" || name == ".hg" || name == ".svn" || name == ".bzr") {
							if entry.IsDir() {
								return filepath.SkipDir
							}
							return nil
						}
						if entry.IsDir() {
							// The listing (including a nested go.mod) decides what is embedded.
							spec.observer.listed(file)
							if file != match {
								if _, err := os.Stat(filepath.Join(file, "go.mod")); err == nil {
									return filepath.SkipDir
								}
							}
							return nil
						}
						info, err := entry.Info()
						if err != nil {
							return err
						}
						if !info.Mode().IsRegular() {
							return nil
						}
						spec.observer.read(file, info)
						rel, err := filepath.Rel(root, file)
						if err != nil {
							return err
						}
						if explicitlySkippedFile(filepath.ToSlash(rel), spec.SkipFiles) || hasSkippedSuffix(filepath.ToSlash(rel), spec.SkipSuffixes) {
							active := explicit[filepath.Clean(source)]
							if !active {
								// The Go toolchain's build constraints decide this; a
								// watch cannot see them.
								spec.observer.markUnwatchable()
								var err error
								active, err = embedSourceActive(source, activePackages)
								if err != nil {
									return err
								}
							}
							if !active {
								return nil
							}
							return fmt.Errorf("embedded input %s conflicts with freshness exclusions", rel)
						}
						assets = append(assets, filepath.ToSlash(rel))
						return nil
					})
					if err != nil {
						return spec, err
					}
				}
			}
		}
	}
	if len(assets) > 0 {
		if len(spec.Inputs) == 0 {
			rel, err := filepath.Rel(root, spec.SourceRoot)
			if err != nil {
				return spec, err
			}
			spec.Inputs = []string{filepath.ToSlash(rel)}
		}
		spec.literalInputs = assets
	}
	return spec, nil
}

// Resolve a conflicting source with the same Go toolchain environment used by
// cli-installer. Uncertain toolchain results fail closed instead of silently
// excluding a potentially active embedded input.
func embedSourceActive(source string, cache map[string]map[string]bool) (bool, error) {
	dir := filepath.Dir(source)
	files, ok := cache[dir]
	if !ok {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "list", "-e", "-json", ".")
		cmd.Dir = dir
		cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
		output, err := cmd.Output()
		if err != nil {
			return false, fmt.Errorf("resolve Go build inputs for %s: %w", dir, err)
		}
		var pkg struct{ GoFiles, CgoFiles, IgnoredGoFiles []string }
		if err := json.Unmarshal(output, &pkg); err != nil {
			return false, fmt.Errorf("resolve Go build inputs for %s: %w", dir, err)
		}
		files = make(map[string]bool)
		for _, name := range pkg.IgnoredGoFiles {
			files[name] = false
		}
		for _, name := range append(pkg.GoFiles, pkg.CgoFiles...) {
			files[name] = true
		}
		cache[dir] = files
	}
	active, known := files[filepath.Base(source)]
	if !known {
		return false, fmt.Errorf("Go build inputs for %s do not classify %s", dir, filepath.Base(source))
	}
	return active, nil
}
