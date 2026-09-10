package projection

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// The same lock protects startup and explicit refresh. Receipts survive process
// restart; they are written before replacement so an interrupted apply is safe
// to retry. This is a cooperative local writer, not an adversarial sandbox.
var refreshMu sync.Mutex

type RefreshRequest struct {
	Runtime        string   `json:"runtime"`
	Skills         []string `json:"skills"`
	Apply          bool     `json:"apply"`
	ExpectedDigest string   `json:"expectedDigest"`
	AdoptLegacy    bool     `json:"adoptLegacy"`
}

type RefreshRow struct {
	Runtime       string `json:"runtime"`
	Skill         string `json:"skill"`
	Status        string `json:"status"`
	SourceHash    string `json:"sourceHash,omitempty"`
	InstalledHash string `json:"installedHash,omitempty"`
	BaselineHash  string `json:"baselineHash,omitempty"`
	ReceiptHash   string `json:"receiptHash,omitempty"`
	Error         string `json:"error,omitempty"`
	BackupPath    string `json:"backupPath,omitempty"`
	Applied       bool   `json:"applied"`
	body          []byte
	dir           string
}

type RefreshResult struct {
	Digest string       `json:"digest"`
	Rows   []RefreshRow `json:"rows"`
}

type receipt struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

// Service accepts only declared runtime names and base-pack members: clients
// cannot turn refresh into an arbitrary filesystem write or full-corpus install.
type Service struct {
	SourceRoot string
	Targets    []Target
	LoadPack   func() (BasePack, error)
	Resolve    func(string) (string, error)
}

func hash(data []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }

func safeID(id string) bool {
	return id != "" && id != "." && id != ".." && !strings.ContainsAny(id, `/\`+"\x00") && !strings.HasPrefix(id, ".")
}

// Reject symlinks at every existing component, including target roots and
// receipts. Missing suffixes are allowed for first installation.
func safePath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for p := abs; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not a projection target: %s", p)
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return nil
}

func readOptional(path string) ([]byte, error) {
	if err := safePath(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

func atomicWrite(path string, data []byte) error {
	if err := safePath(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".projection-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := safePath(path); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func overlaps(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	return a == b || strings.HasPrefix(a, b+string(filepath.Separator)) || strings.HasPrefix(b, a+string(filepath.Separator))
}

func (s *Service) Refresh(req RefreshRequest) (RefreshResult, error) {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	pack, err := s.LoadPack()
	if err != nil {
		return RefreshResult{}, err
	}
	if pack.MaxSkills > 0 && len(pack.Skills) > pack.MaxSkills {
		return RefreshResult{}, fmt.Errorf("base pack exceeds skill ceiling")
	}
	allowed := map[string]bool{}
	for _, id := range pack.Skills {
		if !safeID(id) || allowed[id] {
			return RefreshResult{}, fmt.Errorf("invalid or duplicate base skill %q", id)
		}
		allowed[id] = true
	}
	ids := append([]string(nil), req.Skills...)
	if len(ids) == 0 {
		ids = append(ids, pack.Skills...)
	}
	sort.Strings(ids)
	for i, id := range ids {
		if !allowed[id] || (i > 0 && id == ids[i-1]) {
			return RefreshResult{}, fmt.Errorf("skill %q is not a unique base-pack selection", id)
		}
	}
	var targets []Target
	for _, target := range s.Targets {
		if req.Runtime == "" || req.Runtime == target.Runtime {
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		return RefreshResult{}, fmt.Errorf("no declared projection target for runtime %q", req.Runtime)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Runtime < targets[j].Runtime })
	result := RefreshResult{Rows: []RefreshRow{}}
	// Enforce residency on the whole governed pack, even for a scoped refresh.
	bodies, dirs := map[string][]byte{}, map[string]string{}
	cost := 0
	for _, id := range pack.Skills {
		resolve := s.Resolve
		if resolve == nil {
			resolve = func(id string) (string, error) { return resolveSkillDir(s.SourceRoot, id) }
		}
		dir, err := resolve(id)
		if err != nil {
			return result, err
		}
		if err := validateProjectionEligibility(dir); err != nil {
			return result, err
		}
		body, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			return result, err
		}
		if !hasFrontmatter(string(body)) {
			return result, fmt.Errorf("skill %s has no valid frontmatter", id)
		}
		bodies[id], dirs[id] = []byte(ensureMarker(string(body))), dir
		cost += estimateResidentTokens(string(body))
	}
	if pack.MaxTokens > 0 && cost > pack.MaxTokens {
		return result, fmt.Errorf("base pack costs %d tokens, ceiling is %d", cost, pack.MaxTokens)
	}
	for _, target := range targets {
		for _, id := range ids {
			row := RefreshRow{Runtime: target.Runtime, Skill: id, SourceHash: hash(bodies[id]), body: bodies[id], dir: target.Path}
			if overlaps(target.Path, s.SourceRoot) || overlaps(target.Path, dirs[id]) {
				row.Status, row.Error = "conflict", "projection target overlaps canonical source"
			} else {
				inspect(&row)
			}
			result.Rows = append(result.Rows, row)
		}
	}
	// Bind review to scope, target paths, policy, and all observed identities.
	encoded, _ := json.Marshal(struct {
		Rows    []RefreshRow
		Targets []Target
		Adopt   bool
	}{result.Rows, targets, req.AdoptLegacy})
	result.Digest = hash(encoded)
	if !req.Apply {
		return result, nil
	}
	if req.ExpectedDigest == "" || req.ExpectedDigest != result.Digest {
		return result, fmt.Errorf("preview changed or missing; review a new preview before apply")
	}
	for i := range result.Rows {
		row := &result.Rows[i]
		switch row.Status {
		case "missing", "current", "outdated", "recoverable":
		case "legacy":
			if !req.AdoptLegacy {
				row.Error = "review legacy differences and preview with adoptLegacy before replacement"
				continue
			}
		default:
			continue
		}
		if err := applyRow(row); err != nil {
			row.Error = err.Error()
			continue
		}
		row.Applied = true
	}
	return result, nil
}

func inspect(row *RefreshRow) {
	body, err := readOptional(filepath.Join(row.dir, row.Skill, "SKILL.md"))
	if err != nil {
		row.Status, row.Error = "conflict", err.Error()
		return
	}
	if body != nil {
		row.InstalledHash = hash(body)
	}
	data, err := readOptional(filepath.Join(row.dir, ".prompt-manager", row.Skill+".json"))
	if err != nil {
		row.Status, row.Error = "conflict", err.Error()
		return
	}
	var baseline receipt
	if data != nil {
		row.ReceiptHash = hash(data)
		if err := json.Unmarshal(data, &baseline); err != nil || baseline.After == "" {
			row.Status, row.Error = "conflict", "invalid projection receipt"
			return
		}
		row.BaselineHash = baseline.After
	}
	switch {
	case body == nil:
		row.Status = "missing"
	case row.InstalledHash == row.SourceHash:
		row.Status = "current"
	case baseline.After != "" && row.InstalledHash == baseline.After:
		row.Status = "outdated"
	case baseline.Before != "" && row.InstalledHash == baseline.Before:
		row.Status = "recoverable"
	case data != nil:
		row.Status, row.Error = "modified", "installed content differs from recorded projection; preserve and resolve local edits"
	case strings.Contains(string(body), generatedMarker):
		row.Status = "legacy"
	default:
		row.Status, row.Error = "unmanaged", "existing file is not a managed projection"
	}
}

func applyRow(row *RefreshRow) error {
	path := filepath.Join(row.dir, row.Skill, "SKILL.md")
	before, err := readOptional(path)
	if err != nil {
		return err
	}
	observed := ""
	if before != nil {
		observed = hash(before)
	}
	if observed != row.InstalledHash {
		return fmt.Errorf("installed content changed during apply")
	}
	if before != nil && row.InstalledHash != row.SourceHash {
		row.BackupPath = filepath.Join(row.dir, ".prompt-manager", "backups", row.Skill, strings.TrimPrefix(row.InstalledHash, "sha256:")+".md")
		if err := atomicWrite(row.BackupPath, before); err != nil {
			return err
		}
	}
	data, _ := json.Marshal(receipt{Before: observed, After: row.SourceHash})
	if err := atomicWrite(filepath.Join(row.dir, ".prompt-manager", row.Skill+".json"), data); err != nil {
		return err
	}
	if row.InstalledHash != row.SourceHash {
		if err := atomicWrite(path, row.body); err != nil {
			return err
		}
	}
	return nil
}
