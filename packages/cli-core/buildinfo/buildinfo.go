package buildinfo

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fileEntry struct {
	rel  string
	size int64
	hash [32]byte
}

var skipDirs = []string{
	".git",
	".vscode",
	".idea",
	"coverage",
	"dist",
	"build",
	"tmp",
	"data",
	"node_modules",
}

var skipFiles = []string{
	"build.meta",
}

// ComputeFingerprint walks the provided root directory and returns a deterministic
// fingerprint derived from each file's relative path, size, and contents.
//
// Compiled binaries (ELF, Mach-O, PE, WebAssembly) are excluded — they're
// always build artifacts and including them would rewrite the fingerprint on
// every rebuild. Skip patterns match by path component, so passing
// "<binary-name>" as an extraSkipFile excludes the binary at any depth in
// the source tree (e.g., both `<binary-name>` at the root and
// `cli/<binary-name>` left over from a `go build` in the cli dir).
func ComputeFingerprint(root string, extraSkipFiles ...string) (string, error) {
	var entries []fileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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

		if skipDir(rel) && d.IsDir() {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		if skipFile(rel, extraSkipFiles) {
			return nil
		}

		content, readErr := fs.ReadFile(fs.FS(os.DirFS(root)), rel)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", rel, readErr)
		}

		if IsCompiledBinary(content) {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}

		entries = append(entries, fileEntry{
			rel:  filepath.ToSlash(rel),
			size: info.Size(),
			hash: sha256.Sum256(content),
		})

		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rel < entries[j].rel
	})

	hasher := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(hasher, "%s|%d|%x\n", entry.rel, entry.size, entry.hash)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func skipDir(path string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range skipDirs {
		if PathHasComponent(path, skip) {
			return true
		}
	}
	return false
}

// skipFile reports whether a relative path should be excluded. A skip
// pattern matches when it equals any path component — so "<binary-name>"
// excludes both `<binary-name>` (binary at the context root) AND
// `cli/<binary-name>` (stray binary inside the freshness glob from a
// `go build` in the cli dir). Without component-level matching, leftover
// binaries inside the source tree would rewrite the fingerprint on every
// rebuild and trip the rebuild-loop guard.
func skipFile(path string, extra []string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range skipFiles {
		if PathHasComponent(path, skip) {
			return true
		}
	}
	for _, skip := range extra {
		if PathHasComponent(path, skip) {
			return true
		}
	}
	return false
}

// pathHasComponent reports whether the slash-separated path matches the
// skip pattern want. Single-component patterns (no `/`) match if any path
// segment is exactly equal to want — so `swarm-manager` excludes both
// `swarm-manager` (binary at the root) AND `cli/swarm-manager` (stray
// binary that landed inside a `cli/**` glob via `go build` in cli/).
// Multi-segment patterns (e.g., `custom/cache`) match by prefix —
// `custom/cache/index.json` is excluded but `other/custom/cache.json` is
// not. Empty want never matches.
func PathHasComponent(path, want string) bool {
	if want == "" {
		return false
	}
	if strings.Contains(want, "/") {
		return path == want || strings.HasPrefix(path, want+"/")
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == want {
			return true
		}
	}
	return false
}

// isCompiledBinary reports whether content begins with a recognised
// executable magic number (ELF, Mach-O, PE, WebAssembly). The freshness
// fingerprint excludes these because stray build artifacts would otherwise
// rewrite the fingerprint on every rebuild and trip the rebuild-loop guard.
//
// Java .class files share the CA FE BA BE prefix with Mach-O fat binaries,
// but those are also compiled artifacts and equally inappropriate as
// freshness inputs.
func IsCompiledBinary(content []byte) bool {
	if len(content) < 4 {
		return false
	}
	prefix4 := [4]byte{content[0], content[1], content[2], content[3]}
	switch prefix4 {
	case [4]byte{0x7F, 0x45, 0x4C, 0x46}, // ELF
		[4]byte{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O 32
		[4]byte{0xFE, 0xED, 0xFA, 0xCF}, // Mach-O 64
		[4]byte{0xCE, 0xFA, 0xED, 0xFE}, // Mach-O 32 reverse
		[4]byte{0xCF, 0xFA, 0xED, 0xFE}, // Mach-O 64 reverse
		[4]byte{0xCA, 0xFE, 0xBA, 0xBE}, // Mach-O fat / Java class
		[4]byte{0x00, 0x61, 0x73, 0x6D}: // WebAssembly
		return true
	}
	if content[0] == 'M' && content[1] == 'Z' {
		return true
	}
	return false
}
