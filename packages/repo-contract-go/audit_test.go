package repocontract

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

// TestNoManifestPathLiteralJoins enforces the Phase 1 drift-impossibility
// property: the literal strings "docs/manifest.json" and "cli/manifest.json"
// must not appear anywhere outside .vrooli/repo-contract.json,
// packages/repo-contract-go/ helpers, and the contract invariant checker.
// Consumers must resolve manifest paths through the Scenario*Manifest* helpers
// instead of carrying their own path authority.
func TestNoManifestPathLiteralJoins(t *testing.T) {
	repoRoot := testkitgo.ProjectRoot(t)

	needles := []string{`"docs/manifest.json"`, `"cli/manifest.json"`}

	// Directories pruned during walk (entire subtree skipped).
	prunedDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"data":         true,
		"coverage":     true,
		"dist":         true,
		"build":        true,
		".next":        true,
		".cache":       true,
		".turbo":       true,
		"gen":          true, // generated proto trees
	}

	// Specific paths (repo-root-relative, slash) excluded from the literal check.
	allowlist := map[string]bool{
		".vrooli/repo-contract.json":              true,
		"internal/repocontractcheck/checks.go":    true,
		"packages/repo-contract-go/audit_test.go": true,
		"packages/repo-contract-go/manifests.go":  true,
	}

	// Allowlist directory prefixes (repo-root-relative, slash). The helpers
	// package legitimately mentions the fallback literals.
	allowlistPrefixes := []string{
		"packages/repo-contract-go/",
	}

	// Code extensions only — the audit enforces against code-level path joins.
	// Documentation (markdown) and declarative config (JSON/YAML schemas with
	// `default` values referencing the path) are out of scope; they cannot
	// import the helpers and their literal mentions are content, not drift.
	scanExt := map[string]bool{
		".go":   true,
		".ts":   true,
		".tsx":  true,
		".js":   true,
		".jsx":  true,
		".py":   true,
		".rs":   true,
		".java": true,
		".kt":   true,
		".sh":   true,
		".bash": true,
		".rb":   true,
	}

	type hit struct {
		path string
		line int
		text string
	}
	var hits []hit

	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(repoRoot, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if prunedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if allowlist[rel] {
			return nil
		}
		for _, prefix := range allowlistPrefixes {
			if strings.HasPrefix(rel, prefix) {
				return nil
			}
		}
		if !scanExt[strings.ToLower(filepath.Ext(rel))] {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 4*1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			for _, needle := range needles {
				if strings.Contains(line, needle) {
					hits = append(hits, hit{path: rel, line: lineNo, text: strings.TrimSpace(line)})
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(hits) == 0 {
		return
	}
	t.Errorf("found %d literal manifest-path references; use repo-contract Scenario*Manifest* helpers, or add a narrow audit exception only for contract assertions:", len(hits))
	for _, h := range hits {
		t.Errorf("  %s:%d: %s", h.path, h.line, h.text)
	}
}
