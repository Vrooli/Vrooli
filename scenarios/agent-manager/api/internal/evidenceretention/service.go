package evidenceretention

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultGrace is how long a terminal effort's evidence stays untouched so
// reviewers and resumed acceptance can still use its intermediates.
const DefaultGrace = 7 * 24 * time.Hour

// Options configures a Service.
type Options struct {
	// Root is the evidence home (~/.vrooli/evidence).
	Root string
	// ReferenceRoots are scanned for citations of evidence paths.
	ReferenceRoots []string
	// ReceiptDir receives one JSON receipt per prune that removed anything,
	// plus latest.json for every cycle.
	ReceiptDir string
	// Efforts lists the current effort enrollments.
	Efforts func(context.Context) ([]Effort, error)
	Grace   time.Duration
	Now     func() time.Time
	// remove is a test seam; os.RemoveAll in production.
	remove func(string) error
}

// Service classifies and prunes regenerable evidence intermediates.
type Service struct {
	opts Options
	mu   sync.Mutex // one scan or prune at a time
}

// New validates options.
func New(opts Options) (*Service, error) {
	if strings.TrimSpace(opts.Root) == "" || !filepath.IsAbs(opts.Root) {
		return nil, errors.New("evidence retention: absolute root is required")
	}
	if strings.TrimSpace(opts.ReceiptDir) == "" {
		return nil, errors.New("evidence retention: receipt directory is required")
	}
	opts.Root = filepath.Clean(opts.Root)
	if opts.Grace <= 0 {
		opts.Grace = DefaultGrace
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Efforts == nil {
		opts.Efforts = func(context.Context) ([]Effort, error) { return nil, nil }
	}
	if opts.remove == nil {
		opts.remove = os.RemoveAll
	}
	return &Service{opts: opts}, nil
}

// ClassBytes is a byte breakdown.
type ClassBytes struct {
	// Regenerable is unprotected intermediate bytes a prune would remove.
	Regenerable int64 `json:"regenerableBytes"`
	// ProtectedRegenerable is intermediate bytes kept because a work record
	// cites them.
	ProtectedRegenerable int64 `json:"protectedRegenerableBytes"`
	// Referenced is retained evidence bytes a work record cites.
	Referenced int64 `json:"referencedBytes"`
	// Unknown is retained evidence nobody cites by path; never removed
	// automatically.
	Unknown int64           `json:"unknownBytes"`
	Total   int64           `json:"totalBytes"`
	ByClass map[Class]int64 `json:"regenerableByClass,omitempty"`
	Counts  map[Class]int   `json:"regenerableCountByClass,omitempty"`
}

func (b *ClassBytes) addEffort(e EffortReport) {
	b.Regenerable += e.Bytes.Regenerable
	b.ProtectedRegenerable += e.Bytes.ProtectedRegenerable
	b.Referenced += e.Bytes.Referenced
	b.Unknown += e.Bytes.Unknown
	b.Total += e.Bytes.Total
	for c, n := range e.Bytes.ByClass {
		if b.ByClass == nil {
			b.ByClass, b.Counts = map[Class]int64{}, map[Class]int{}
		}
		b.ByClass[c] += n
		b.Counts[c] += e.Bytes.Counts[c]
	}
}

// EffortReport classifies one effort directory.
type EffortReport struct {
	Name       string      `json:"name"`
	Lifecycle  Lifecycle   `json:"lifecycle"`
	Bytes      ClassBytes  `json:"bytes"`
	Candidates []Candidate `json:"candidates"`
}

// Report is the dry-run classification of the evidence home.
type Report struct {
	Root        string         `json:"root"`
	GeneratedAt time.Time      `json:"generatedAt"`
	Grace       string         `json:"grace"`
	References  int            `json:"citedPaths"`
	Totals      ClassBytes     `json:"totals"`
	Efforts     []EffortReport `json:"efforts"`
	Warnings    []string       `json:"warnings,omitempty"`
}

// Scan produces the dry-run report. It never changes the filesystem.
func (s *Service) Scan(ctx context.Context) (Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scan(ctx)
}

func (s *Service) scan(ctx context.Context) (Report, error) {
	now := s.opts.Now().UTC()
	report := Report{Root: s.opts.Root, GeneratedAt: now, Grace: s.opts.Grace.String()}
	entries, err := os.ReadDir(s.opts.Root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return report, nil
		}
		return report, err
	}
	refs, err := CollectReferences(ctx, s.opts.ReferenceRoots, s.opts.Root)
	if err != nil {
		return report, fmt.Errorf("collect evidence citations: %w", err)
	}
	report.References = refs.Len()
	efforts, err := s.opts.Efforts(ctx)
	if err != nil {
		// Without enrollments nothing is terminal; the report stays useful.
		report.Warnings = append(report.Warnings, "effort enrollments unavailable: "+err.Error())
		efforts = nil
	}
	sz := newSizer()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return report, err
		}
		scan := scanEffort(s.opts.Root, entry.Name(), refs, sz)
		er := EffortReport{Name: entry.Name(), Lifecycle: ResolveLifecycle(entry.Name(), efforts, s.opts.Grace, now), Candidates: scan.Candidates}
		if er.Candidates == nil {
			er.Candidates = []Candidate{}
		}
		er.Bytes.Total = scan.TotalBytes
		er.Bytes.Referenced = scan.ReferencedBytes
		var candidateBytes int64
		for _, c := range scan.Candidates {
			candidateBytes += c.Bytes
			if c.Protected {
				er.Bytes.ProtectedRegenerable += c.Bytes
				continue
			}
			er.Bytes.Regenerable += c.Bytes
			if er.Bytes.ByClass == nil {
				er.Bytes.ByClass, er.Bytes.Counts = map[Class]int64{}, map[Class]int{}
			}
			er.Bytes.ByClass[c.Class] += c.Bytes
			er.Bytes.Counts[c.Class]++
		}
		er.Bytes.Unknown = scan.TotalBytes - candidateBytes - scan.ReferencedBytes
		report.Efforts = append(report.Efforts, er)
		report.Totals.addEffort(er)
	}
	sort.Slice(report.Efforts, func(i, j int) bool { return report.Efforts[i].Bytes.Total > report.Efforts[j].Bytes.Total })
	return report, nil
}

// ErrInvalidRequest marks a prune request refused before any scan.
var ErrInvalidRequest = errors.New("invalid evidence prune request")

// PruneRequest selects what a prune may remove.
type PruneRequest struct {
	// Efforts names effort directories an operator has judged finished. When
	// empty, only auto-eligible (terminal past grace) efforts are pruned.
	Efforts []string `json:"efforts,omitempty"`
	// Classes limits the removed classes; empty means every class.
	Classes []Class `json:"classes,omitempty"`
	DryRun  bool    `json:"dry_run"`
	Actor   string  `json:"actor,omitempty"`
	Reason  string  `json:"reason,omitempty"`
	// OnlyPaths, when non-nil, further limits removal to these candidate
	// paths. An operator-approved prune sets it to the previewed set so an
	// approval can never cover a candidate the operator did not see. It is
	// never read from the wire.
	OnlyPaths []string `json:"-"`
	// Approval attributes a prune applied on an operator decision.
	Approval *Approval `json:"-"`
}

// Removal records one removed (or, in a dry run, removable) path.
type Removal struct {
	Effort string `json:"effort"`
	Path   string `json:"path"`
	Class  Class  `json:"class"`
	Bytes  int64  `json:"bytes"`
	Files  int    `json:"files"`
	Reason string `json:"reason"`
}

// Skip records an operator-named effort or candidate that was not pruned.
type Skip struct {
	Effort string `json:"effort"`
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
}

// Receipt is the durable record of a prune.
type Receipt struct {
	Mode         string    `json:"mode"`
	Actor        string    `json:"actor,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	DryRun       bool      `json:"dryRun"`
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt"`
	Root         string    `json:"root"`
	RemovedBytes int64     `json:"removedBytes"`
	RemovedFiles int       `json:"removedFiles"`
	Removed      []Removal `json:"removed"`
	Skipped      []Skip    `json:"skipped,omitempty"`
	Errors       []string  `json:"errors,omitempty"`
	ReceiptPath  string    `json:"receiptPath,omitempty"`
	// Approval is set when the prune ran on an approved operator decision.
	Approval *Approval `json:"approval,omitempty"`
}

// Prune removes unprotected regenerable intermediates. With no named
// efforts it is the automatic cycle and touches only auto-eligible efforts;
// named efforts are an operator assertion that the work is finished.
func (s *Service) Prune(ctx context.Context, req PruneRequest) (Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt := Receipt{Mode: "automatic", Actor: req.Actor, Reason: req.Reason, DryRun: req.DryRun, StartedAt: s.opts.Now().UTC(), Root: s.opts.Root, Removed: []Removal{}, Approval: req.Approval}
	named := map[string]bool{}
	for _, name := range req.Efforts {
		name = strings.TrimSpace(name)
		if name == "" || name != filepath.Base(name) || name == "." || name == ".." {
			return receipt, fmt.Errorf("%w: effort directory name %q", ErrInvalidRequest, name)
		}
		named[name] = true
	}
	if len(named) > 0 {
		receipt.Mode = "operator"
	}
	classes := map[Class]bool{}
	for _, c := range req.Classes {
		if !knownClass(c) {
			return receipt, fmt.Errorf("%w: unknown class %q", ErrInvalidRequest, c)
		}
		classes[c] = true
	}
	var only map[string]bool
	if req.OnlyPaths != nil {
		only = map[string]bool{}
		for _, p := range req.OnlyPaths {
			only[p] = true
		}
	}
	report, err := s.scan(ctx)
	if err != nil {
		return receipt, err
	}
	seen := map[string]bool{}
	for _, e := range report.Efforts {
		seen[e.Name] = true
		if len(named) > 0 {
			if !named[e.Name] {
				continue
			}
			if e.Lifecycle.State == StateActive {
				receipt.Skipped = append(receipt.Skipped, Skip{Effort: e.Name, Reason: "a matching effort is still live; withdraw or finalize it first"})
				continue
			}
		} else if !e.Lifecycle.AutoEligible {
			continue
		}
		for _, c := range e.Candidates {
			if len(classes) > 0 && !classes[c.Class] {
				continue
			}
			if only != nil && !only[c.Path] {
				continue
			}
			if c.Protected {
				receipt.Skipped = append(receipt.Skipped, Skip{Effort: e.Name, Path: c.Path, Reason: "cited by " + c.ProtectedBy})
				continue
			}
			if err := ctx.Err(); err != nil {
				receipt.Errors = append(receipt.Errors, err.Error())
				break
			}
			removal := Removal{Effort: e.Name, Path: c.Path, Class: c.Class, Bytes: c.Bytes, Files: c.Files, Reason: c.Reason}
			if !req.DryRun {
				if err := s.removeCandidate(c.Path); err != nil {
					receipt.Errors = append(receipt.Errors, c.Path+": "+err.Error())
					continue
				}
			}
			receipt.Removed = append(receipt.Removed, removal)
			receipt.RemovedBytes += c.Bytes
			receipt.RemovedFiles += c.Files
		}
	}
	for name := range named {
		if !seen[name] {
			receipt.Skipped = append(receipt.Skipped, Skip{Effort: name, Reason: "no such effort directory"})
		}
	}
	receipt.FinishedAt = s.opts.Now().UTC()
	if writeErr := s.writeReceipt(&receipt); writeErr != nil {
		return receipt, writeErr
	}
	if len(receipt.Errors) > 0 {
		return receipt, fmt.Errorf("%d evidence removals failed", len(receipt.Errors))
	}
	return receipt, nil
}

func knownClass(c Class) bool {
	switch c {
	case ClassPlatformRelease, ClassReleaseArchive, ClassDependencyTree, ClassBuildBinary, ClassRenderFrames:
		return true
	}
	return false
}

// removeCandidate re-checks containment immediately before removal: the
// path must resolve inside the root without crossing a symlink, and the
// candidate itself must not be a symlink.
func (s *Service) removeCandidate(rel string) error {
	full := filepath.Join(s.opts.Root, filepath.FromSlash(rel))
	if !strings.HasPrefix(full, s.opts.Root+string(filepath.Separator)) {
		return errors.New("path escapes the evidence root")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(full))
	if err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(s.opts.Root)
	if err != nil {
		return err
	}
	if parent != root && !strings.HasPrefix(parent, root+string(filepath.Separator)) {
		return errors.New("parent resolves outside the evidence root")
	}
	info, err := os.Lstat(full)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("candidate is a symlink")
	}
	return s.opts.remove(full)
}

// writeReceipt always refreshes latest.json and keeps a timestamped receipt
// whenever bytes were removed, so every deletion stays auditable.
func (s *Service) writeReceipt(r *Receipt) error {
	if err := os.MkdirAll(s.opts.ReceiptDir, 0o755); err != nil {
		return err
	}
	if len(r.Removed) > 0 && !r.DryRun {
		r.ReceiptPath = filepath.Join(s.opts.ReceiptDir, "prune-"+r.StartedAt.Format("20060102T150405.000000000Z")+".json")
		if err := writeJSON(r.ReceiptPath, r); err != nil {
			return err
		}
	}
	return writeJSON(filepath.Join(s.opts.ReceiptDir, "latest.json"), r)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
