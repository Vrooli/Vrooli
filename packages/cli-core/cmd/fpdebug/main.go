// fpdebug is a one-shot diagnostic for the CLI freshness-fingerprint
// computation. Prints the fingerprint AND a listing of every file that
// participated, so install-time vs run-time divergence can be diffed.
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type stringListFlag []string

func (f *stringListFlag) String() string     { return strings.Join(*f, ",") }
func (f *stringListFlag) Set(v string) error { *f = append(*f, v); return nil }

func main() {
	sourceRoot := flag.String("source-root", "", "absolute SourceRoot")
	contextRoot := flag.String("context-root", "", "absolute ContextRoot (defaults to source-root)")
	skipFile := flag.String("skip-file", "", "skip-file basename (typically binary name)")
	listFiles := flag.Bool("list-files", false, "list every file that participated in the fingerprint")
	var inputs stringListFlag
	flag.Var(&inputs, "input", "freshness input pattern (repeatable)")
	flag.Parse()

	if *sourceRoot == "" {
		fmt.Fprintln(os.Stderr, "missing --source-root")
		os.Exit(2)
	}
	if *contextRoot == "" {
		*contextRoot = *sourceRoot
	}

	spec := cliutil.FreshnessSpec{
		SourceRoot:  *sourceRoot,
		ContextRoot: *contextRoot,
		SkipFiles:   []string{*skipFile},
		Inputs:      []string(inputs),
	}
	fp, err := cliutil.ComputeFreshnessFingerprint(spec)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	fmt.Println("fingerprint:", fp)

	if *listFiles {
		files := enumerateFiles(*contextRoot, []string(inputs), *skipFile)
		sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
		for _, f := range files {
			fmt.Printf("%s  %d  %s\n", f.hash[:12], f.size, f.rel)
		}
		fmt.Printf("total files: %d\n", len(files))
	}
}

type fileEntry struct {
	rel  string
	size int64
	hash string
}

func enumerateFiles(root string, inputs []string, skipFile string) []fileEntry {
	var out []fileEntry
	for _, input := range inputs {
		var matches []string
		if strings.ContainsAny(input, "*?[") {
			m, _ := filepath.Glob(filepath.Join(root, filepath.FromSlash(input)))
			matches = m
		} else {
			matches = []string{filepath.Join(root, filepath.FromSlash(input))}
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.IsDir() {
				_ = filepath.WalkDir(match, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					rel, relErr := filepath.Rel(root, path)
					if relErr != nil {
						return relErr
					}
					rel = filepath.ToSlash(rel)
					if rel == "." {
						return nil
					}
					if isSkipDir(rel) && d.IsDir() {
						return filepath.SkipDir
					}
					if d.IsDir() || isSkipFile(rel, skipFile) {
						return nil
					}
					content, readErr := os.ReadFile(path)
					if readErr != nil {
						return readErr
					}
					fi, _ := d.Info()
					h := sha256.Sum256(content)
					out = append(out, fileEntry{rel: rel, size: fi.Size(), hash: fmt.Sprintf("%x", h)})
					return nil
				})
				continue
			}
			rel, _ := filepath.Rel(root, match)
			rel = filepath.ToSlash(rel)
			if isSkipFile(rel, skipFile) {
				continue
			}
			content, _ := os.ReadFile(match)
			h := sha256.Sum256(content)
			out = append(out, fileEntry{rel: rel, size: info.Size(), hash: fmt.Sprintf("%x", h)})
		}
	}
	return out
}

func isSkipDir(path string) bool {
	for _, skip := range []string{".git", ".vscode", ".idea", "coverage", "dist", "build", "tmp", "data", "node_modules"} {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}

func isSkipFile(path, skip string) bool {
	if skip != "" {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	if path == "build.meta" || strings.HasPrefix(path, "build.meta/") {
		return true
	}
	return false
}
