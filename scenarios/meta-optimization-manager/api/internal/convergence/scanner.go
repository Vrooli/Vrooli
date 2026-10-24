package convergence

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// FitnessScanner computes the four-lens fitness of each template. Production
// walks the template filesystem and derives honest proxy counts; tests fake it.
// The precise reference-pattern-fitness mechanization (the add/delete
// coordinated-edit walkthrough, drift-surface enumeration via cartographer) is a
// documented refinement seam — the proxies here are real, non-fabricated
// filesystem signals, never invented numbers.
type FitnessScanner interface {
	Scan(ctx context.Context, template string) ([]TemplateFitness, error)
}

// fsFitnessScanner is the production scanner: it reads templates/scenarios/*.
type fsFitnessScanner struct {
	root string // repo root; resolved lazily when empty
}

// NewFitnessScanner returns the production FitnessScanner.
func NewFitnessScanner() FitnessScanner { return &fsFitnessScanner{} }

// NewFitnessScannerWithRoot returns a scanner rooted at the given repo path
// (tests point it at a fixture tree).
func NewFitnessScannerWithRoot(root string) FitnessScanner { return &fsFitnessScanner{root: root} }

var _ FitnessScanner = (*fsFitnessScanner)(nil)

// contractPhrases are the comment-only-contract markers (lens 3): a contract
// expressed in prose that does not survive copy-paste-and-modify.
var contractPhrases = []string{
	"callers must", "caller must", "must not", "must leave", "caller-zero",
	"do not call", "callers should", "invariant:", "precondition:",
}

// driftMarkers are explicit "N copies must agree but only convention enforces it"
// markers (lens 2), plus hand-rolled fakes of shared interfaces (a drift surface
// per REFERENCE_PATTERN_FITNESS.md).
var driftMarkers = []string{"keep in sync", "must match", "must stay in sync", "drift"}

// centralWiringFiles are the files an add-domain / delete-domain edit must touch
// together (lens 4 proxy): the coordinated-edit surface.
var centralWiringFiles = []string{
	"registry.go", "main.go", "domains.go", "manifest.json", "app.go",
}

// sourceExts bounds per-replica-cost (lens 1) and the comment scans to real
// source files (skip generated, vendored, lockfiles).
var sourceExts = map[string]bool{".go": true, ".ts": true, ".tsx": true, ".sh": true}

func (s *fsFitnessScanner) Scan(ctx context.Context, template string) ([]TemplateFitness, error) {
	root := s.root
	if root == "" {
		r, err := repocontract.FindRepoRootFromEnvOrCWD()
		if err != nil {
			return nil, err
		}
		root = r
	}
	templatesDir := filepath.Join(root, "templates", "scenarios")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, err
	}
	var out []TemplateFitness
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if template != "" && name != template {
			continue
		}
		tf, err := scanTemplate(ctx, filepath.Join(templatesDir, name), name)
		if err != nil {
			continue // degrade: skip an unreadable template rather than failing the scan
		}
		out = append(out, tf)
	}
	return out, nil
}

// scanTemplate walks one template tree and derives the four proxy counts.
func scanTemplate(ctx context.Context, dir, name string) (TemplateFitness, error) {
	tf := TemplateFitness{Template: name}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		base := d.Name()
		// Lens 4: coordinated-edit surface — count central wiring files present.
		for _, w := range centralWiringFiles {
			if base == w {
				tf.CoordinatedEditCount++
				break
			}
		}
		// Lens 2: hand-rolled fakes/mocks are drift surfaces.
		low := strings.ToLower(base)
		if strings.Contains(low, "fake") || strings.Contains(low, "mock") {
			tf.DriftSurfaceCount++
		}
		ext := filepath.Ext(base)
		if !sourceExts[ext] {
			return nil
		}
		loc, contracts, drifts := scanFile(path)
		tf.PerReplicaCost += loc
		tf.CommentOnlyContractCount += contracts
		tf.DriftSurfaceCount += drifts
		return nil
	})
	if err != nil {
		return TemplateFitness{}, err
	}
	tf.Tier = deriveTier(tf)
	return tf, nil
}

// scanFile counts non-blank lines (per-replica cost), comment-only contract
// phrases, and in-comment drift markers in one source file.
func scanFile(path string) (loc, contracts, drifts int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		loc++
		if !isCommentLine(trimmed) {
			continue
		}
		low := strings.ToLower(trimmed)
		for _, p := range contractPhrases {
			if strings.Contains(low, p) {
				contracts++
				break
			}
		}
		for _, m := range driftMarkers {
			if strings.Contains(low, m) {
				drifts++
				break
			}
		}
	}
	return loc, contracts, drifts
}

// isCommentLine reports whether a trimmed source line is a comment (Go // /* or
// shell #). Conservative: only counts contract/drift phrases that live in
// comments (the lens is about contracts in prose, not code).
func isCommentLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "#")
}

func skipDir(name string) bool {
	switch name {
	case "node_modules", ".git", "gen", "dist", "build", "coverage", "vendor", ".vrooli":
		return true
	}
	return false
}
