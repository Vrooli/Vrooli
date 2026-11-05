package artifacts

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
)

// Run artifacts are laid out as <root>/<target>/runs/<run-id>/<asset>/ with a
// small redacted latest.json (the checksummed evidence reference consumers
// link) and a timeline.json that carries full BAS step telemetry, dominated by
// per-step DOM snapshots (often 2-20 MB each). Retention removes only
// timeline.json from finished runs that nobody cites; latest.json, the run
// directory, and every cited or pinned run are always kept.

const (
	timelineFile = "timeline.json"
	latestFile   = "latest.json"
	// RetainMarker pins a run (in its run directory) or one workflow (in its
	// asset directory) against timeline retention.
	RetainMarker = ".retain"
)

// RetentionPolicy bounds timeline storage.
type RetentionPolicy struct {
	// MinAge is the review window: timelines younger than this are kept
	// regardless of size.
	MinAge time.Duration
	// KeepPerWorkflow keeps the newest N timelines of each target+workflow so
	// the latest diagnostic stays inspectable however old it is.
	KeepPerWorkflow int
	// MaxBytes, when positive, triggers an oldest-first pass over timelines
	// older than PressureMinAge until the store fits.
	MaxBytes int64
	// PressureMinAge is the floor of the byte pass; it never removes a
	// timeline younger than this.
	PressureMinAge time.Duration
}

// DefaultRetentionPolicy keeps three days of full timelines, the newest two per
// workflow, and under byte pressure trims to six GiB without touching the last
// day.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{MinAge: 72 * time.Hour, KeepPerWorkflow: 2, MaxBytes: 6 << 30, PressureMinAge: 24 * time.Hour}
}

// RetentionInputs are the live protection signals for one cycle.
type RetentionInputs struct {
	Now time.Time
	// ActiveRuns are queued/running validation run IDs; their artifacts are
	// still being written.
	ActiveRuns map[string]bool
	// CitedRuns are run IDs cited by retained evidence (docs, work records).
	CitedRuns map[string]bool
}

// PrunedTimeline records one removed timeline for the receipt.
type PrunedTimeline struct {
	Path    string    `json:"path"`
	RunID   string    `json:"run_id"`
	Bytes   int64     `json:"bytes"`
	ModTime time.Time `json:"mod_time"`
	Reason  string    `json:"reason"`
}

// RetentionResult summarizes one cycle and is persisted as the receipt.
type RetentionResult struct {
	CycleAt        time.Time        `json:"cycle_at"`
	Root           string           `json:"root"`
	Policy         RetentionPolicy  `json:"policy"`
	TimelinesSeen  int              `json:"timelines_seen"`
	BytesBefore    int64            `json:"bytes_before"`
	BytesAfter     int64            `json:"bytes_after"`
	FreedBytes     int64            `json:"freed_bytes"`
	Protected      map[string]int   `json:"protected"`
	Pruned         []PrunedTimeline `json:"pruned"`
	OverBudget     bool             `json:"over_budget"`
	Errors         []string         `json:"errors,omitempty"`
	ReceiptVersion int              `json:"receipt_version"`
}

type timelineEntry struct {
	path    string
	target  string
	runID   string
	asset   string
	bytes   int64
	modTime time.Time
}

// RunArtifactsRoot resolves the owner-scoped run-artifact store.
func RunArtifactsRoot() (string, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return "", fmt.Errorf("create artifact resolver: %w", err)
	}
	return resolver.Path(corestorage.Options{ScenarioID: "workflow-health"}, corestorage.ClassState, "run-artifacts")
}

// PruneTimelines applies policy to the store under root. It deletes only
// timeline.json files and never a file it cannot attribute to the
// <target>/runs/<run>/<asset>/ layout.
func PruneTimelines(root string, policy RetentionPolicy, in RetentionInputs) (RetentionResult, error) {
	result := RetentionResult{CycleAt: in.Now.UTC(), Root: root, Policy: policy, Protected: map[string]int{}, ReceiptVersion: 1}
	entries, totalBytes, err := scanTimelines(root)
	if err != nil {
		return result, err
	}
	result.TimelinesSeen = len(entries)
	result.BytesBefore = totalBytes

	// Newest-first per workflow; the first KeepPerWorkflow are kept.
	groups := map[string][]*timelineEntry{}
	for i := range entries {
		key := entries[i].target + "\x00" + entries[i].asset
		groups[key] = append(groups[key], &entries[i])
	}
	newest := map[*timelineEntry]bool{}
	for _, group := range groups {
		sort.Slice(group, func(i, j int) bool { return group[i].modTime.After(group[j].modTime) })
		for i := 0; i < len(group) && i < policy.KeepPerWorkflow; i++ {
			newest[group[i]] = true
		}
	}

	// Candidates are eligible on everything but age; age decides the pass.
	var candidates []*timelineEntry
	for i := range entries {
		entry := &entries[i]
		if reason := protectionReason(root, entry, newest, in); reason != "" {
			result.Protected[reason]++
			continue
		}
		candidates = append(candidates, entry)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].modTime.Before(candidates[j].modTime) })

	remaining := totalBytes
	remove := func(entry *timelineEntry, reason string) {
		if err := corestorage.RemoveFile(entry.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			result.Errors = append(result.Errors, fmt.Sprintf("remove %s: %v", entry.path, err))
			return
		}
		remaining -= entry.bytes
		result.FreedBytes += entry.bytes
		result.Pruned = append(result.Pruned, PrunedTimeline{Path: entry.path, RunID: entry.runID, Bytes: entry.bytes, ModTime: entry.modTime.UTC(), Reason: reason})
	}
	var young []*timelineEntry
	for _, entry := range candidates {
		if policy.MinAge > 0 && in.Now.Sub(entry.modTime) >= policy.MinAge {
			remove(entry, "age")
			continue
		}
		young = append(young, entry)
	}
	if policy.MaxBytes > 0 {
		for _, entry := range young {
			if remaining <= policy.MaxBytes {
				break
			}
			if in.Now.Sub(entry.modTime) < policy.PressureMinAge {
				result.Protected["pressure_floor"]++
				continue
			}
			remove(entry, "bytes")
		}
	}
	result.BytesAfter = remaining
	result.OverBudget = policy.MaxBytes > 0 && remaining > policy.MaxBytes
	if len(result.Errors) > 0 {
		return result, fmt.Errorf("timeline retention: %d removals failed (first: %s)", len(result.Errors), result.Errors[0])
	}
	return result, nil
}

func protectionReason(root string, entry *timelineEntry, newest map[*timelineEntry]bool, in RetentionInputs) string {
	switch {
	case in.ActiveRuns[entry.runID]:
		return "active_run"
	case in.CitedRuns[entry.runID]:
		return "cited_evidence"
	case newest[entry]:
		return "newest_per_workflow"
	}
	runDir := filepath.Join(root, entry.target, "runs", entry.runID)
	if fileExists(filepath.Join(runDir, RetainMarker)) || fileExists(filepath.Join(runDir, entry.asset, RetainMarker)) {
		return "retain_marker"
	}
	return ""
}

// scanTimelines walks only <root>/<target>/runs/<run>/<asset>/timeline.json.
// It returns the attributed timelines and the total bytes of every regular
// file in the store, so budget math counts what the store actually holds.
func scanTimelines(root string) ([]timelineEntry, int64, error) {
	var (
		entries []timelineEntry
		total   int64
	)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == root {
				return fs.SkipAll
			}
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		if d.Name() != timelineFile {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 5 || parts[1] != "runs" {
			return nil
		}
		entries = append(entries, timelineEntry{path: path, target: parts[0], runID: parts[2], asset: parts[3], bytes: info.Size(), modTime: info.ModTime()})
		return nil
	})
	return entries, total, err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var runIDPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// citationFileLimit skips files too large to be hand-written evidence text.
const citationFileLimit = 32 << 20

// CitedRunIDs collects every UUID mentioned in text files under roots. Any run
// whose ID appears there (evidence notes, work records, readbacks) is treated
// as cited evidence. Over-matching only keeps more timelines; it can never
// cause a deletion. Missing roots are ignored.
func CitedRunIDs(roots ...string) (map[string]bool, error) {
	cited := map[string]bool{}
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
					return nil
				}
				return err
			}
			if d.IsDir() {
				switch d.Name() {
				case "node_modules", ".git":
					return fs.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			if info, err := d.Info(); err != nil || info.Size() > citationFileLimit {
				return nil
			}
			return scanFileForRunIDs(path, cited)
		})
		if err != nil {
			return cited, fmt.Errorf("scan citations under %s: %w", root, err)
		}
	}
	return cited, nil
}

func scanFileForRunIDs(path string, cited map[string]bool) error {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), citationFileLimit)
	for scanner.Scan() {
		for _, id := range runIDPattern.FindAll(scanner.Bytes(), -1) {
			cited[string(id)] = true
		}
	}
	// A line beyond the buffer is binary or generated data, not a citation.
	return nil
}

// WriteRetentionReceipt atomically persists the cycle result so each deletion
// is verifiable after the fact.
func WriteRetentionReceipt(path string, result RetentionResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return corestorage.WriteFileAtomic(path, append(data, '\n'), 0o644)
}
