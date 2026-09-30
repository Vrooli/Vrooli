// Package evidence resolves managed evidence output and checks declared producers.
package evidence

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// backtickedPath finds the `path` spans EVIDENCE.md names artifacts with.
var backtickedPath = regexp.MustCompile("`([^`]+)`")

// ArtifactTableHeading is the section whose table is the declaration. Only that
// table counts.
//
// Reading the whole document instead would be wrong twice over. The rule states
// itself in prose as "every artifact under `evidence/` is produced by a
// command written here", and a parser that took that span as an entry would
// admit the entire tree — the rule's own statement would repeal it, which is
// how the first version of this check passed against fourteen unreproducible
// files. The "Deleted rather than kept" section then names the paths of
// artifacts removed *because* they had no command, and admitting those would
// invite back exactly the files the rule threw out.
const ArtifactTableHeading = "## Artifacts and their producing commands"

// Coverage is one declared entry from EVIDENCE.md and what it admits.
type Coverage struct {
	// Pattern is the entry exactly as EVIDENCE.md writes it.
	Pattern string
	// Directory is true when the entry names a directory rather than a file,
	// in which case it covers everything beneath it. EVIDENCE.md uses that form
	// where the producing command is stated in the directory's own README.
	Directory bool
}

// DeclaredCoverage extracts every artifact path declared by the artifact table
// in EVIDENCE.md. A path named anywhere else in the document is prose, not a
// declaration, and does not admit a file.
func DeclaredCoverage(evidenceDoc string) []Coverage {
	var out []Coverage
	seen := map[string]bool{}
	inTable := false
	for _, line := range strings.Split(evidenceDoc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inTable = trimmed == ArtifactTableHeading
			continue
		}
		if !inTable || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		// The artifact is the first cell; the command and the rationale follow.
		// Reading only that cell keeps a path mentioned inside a producing
		// command from declaring itself.
		cells := strings.Split(strings.Trim(trimmed, "|"), "|")
		if len(cells) == 0 {
			continue
		}
		// EVERY backticked path in that cell, not just the first. A row
		// legitimately declares two artifacts — a directory of images and the
		// README that indexes them — and reading only the first silently left
		// the second uncovered, which this rule then reported as undeclared.
		// The narrowing that matters is the cell, not the count.
		for _, match := range backtickedPath.FindAllStringSubmatch(cells[0], -1) {
			candidate := strings.TrimSpace(match[1])
			if candidate == "evidence/" || !strings.HasPrefix(candidate, "evidence/") || seen[candidate] {
				continue
			}
			seen[candidate] = true
			out = append(out, Coverage{Pattern: candidate, Directory: strings.HasSuffix(candidate, "/")})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pattern < out[j].Pattern })
	return out
}

// covers reports whether one declared entry admits a repo-relative artifact.
func (c Coverage) covers(artifact string) bool {
	if c.Directory {
		return strings.HasPrefix(artifact, c.Pattern)
	}
	if strings.ContainsAny(c.Pattern, "*?[") {
		// Match the pattern against the whole path, and separately against the
		// base name inside the pattern's own directory, so `.../sheet-*.png`
		// admits `.../sheet-treatment-family.png` without admitting a file of
		// that name in some other directory.
		if ok, err := path.Match(c.Pattern, artifact); err == nil && ok {
			return true
		}
		return path.Dir(c.Pattern) == path.Dir(artifact) &&
			func() bool { ok, err := path.Match(path.Base(c.Pattern), path.Base(artifact)); return err == nil && ok }()
	}
	return c.Pattern == artifact
}

// Unreferenced walks the evidence tree and returns every artifact EVIDENCE.md
// does not name, as repo-relative paths.
//
// evidenceRoot is the filesystem path of `evidence`; evidenceDoc is the
// text of `docs/internal/EVIDENCE.md`.
func Unreferenced(evidenceRoot, evidenceDoc string) ([]string, error) {
	coverage := DeclaredCoverage(evidenceDoc)
	var orphans []string
	err := filepath.WalkDir(evidenceRoot, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, relErr := filepath.Rel(evidenceRoot, current)
		if relErr != nil {
			return relErr
		}
		artifact := path.Join("evidence", filepath.ToSlash(relative))
		for _, c := range coverage {
			if c.covers(artifact) {
				return nil
			}
		}
		orphans = append(orphans, artifact)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("evidence: walk %s: %w", evidenceRoot, err)
	}
	sort.Strings(orphans)
	return orphans, nil
}

// Load reads the evidence tree root and the EVIDENCE.md text, given the
// scenario root. It exists so the test and any future caller agree on where
// both live rather than each spelling the relative path itself.
func Load(scenarioRoot string) (evidenceRoot string, evidenceDoc string, err error) {
	evidenceRoot, err = OutputPath()
	if err != nil {
		return "", "", err
	}
	raw, err := os.ReadFile(filepath.Join(scenarioRoot, "docs", "internal", "EVIDENCE.md"))
	if err != nil {
		return "", "", fmt.Errorf("evidence: read EVIDENCE.md: %w", err)
	}
	return evidenceRoot, string(raw), nil
}

// OutputPath resolves logical evidence names through the shared storage authority.
// Tests may inject an absolute output root; callers never fall back to source paths.
func OutputPath(parts ...string) (string, error) {
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `/\\`) {
			return "", fmt.Errorf("evidence: invalid logical path segment %q", part)
		}
	}
	root := os.Getenv("BACKDROP_STUDIO_EVIDENCE_DIR")
	if root != "" {
		if !filepath.IsAbs(root) {
			return "", fmt.Errorf("evidence: output root must be absolute")
		}
	} else {
		resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
		if err != nil {
			return "", err
		}
		namespace, err := storage.ScenarioNamespace("backdrop-studio")
		if err != nil {
			return "", err
		}
		root, err = resolver.Path(storage.Options{ScenarioID: namespace}, storage.ClassTestRuns, "evidence")
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(append([]string{root}, parts...)...), nil
}
