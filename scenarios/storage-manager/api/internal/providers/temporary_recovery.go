package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"storage-manager/internal/cleanup"
)

type recoveryWalkState struct {
	entries   []cleanup.FileInfo
	index     int
	walk      cleanup.FileWalk
	aggregate entryAggregate
	protected bool
	ready     map[string]cleanup.PreviewItem
}

// A recovery observation keeps measuring until it holds about one controller
// batch of complete units. Publishing a single unit per observation made
// throughput depend on entry count rather than bytes: a host with ~7,900 small
// top-level temp entries ahead of one 20 GB tree needed thousands of passes to
// reach it. These mirror the orchestrator's batch cap and item limit; the
// observation budget still bounds the wall clock of every pass.
const (
	temporaryRecoveryReadyBytes = int64(2 * 1024 * 1024 * 1024)
	temporaryRecoveryReadyItems = 200
)

// elevationBackoff is how long an entry that this user cannot fully read stays
// out of recovery. Such an entry (a fixture whose subtree belongs to root, a
// directory left mode 000) cannot be measured or removed without elevation, so
// re-walking it every run only repeats the same openat failure; and because it
// was reported as the provider's blocked reason, one such entry made the whole
// /tmp provider read as skipped while thousands of removable entries waited.
const elevationBackoff = 24 * time.Hour

type elevationRecord struct {
	identity string
	until    time.Time
	reported bool
}

// rememberNeedsElevation records that an entry needs elevation. It reports
// true only the first time an entry identity is classified, so the operator
// is told once rather than on every run.
// Callers hold recoveryMu; the entry never reached the ready set.
func (p *FileProvider) rememberNeedsElevation(path, identity string) bool {
	p.failedMu.Lock()
	defer p.failedMu.Unlock()
	now := p.failureNow()
	if p.elevation == nil {
		p.elevation = make(map[string]elevationRecord)
	}
	key := filepath.Clean(path)
	previous, known := p.elevation[key]
	firstReport := !known || !previous.reported || previous.identity != identity
	p.elevation[key] = elevationRecord{identity: identity, until: now.Add(elevationBackoff), reported: true}
	if len(p.elevation) > 4096 {
		for candidate, record := range p.elevation {
			if !now.Before(record.until) {
				delete(p.elevation, candidate)
			}
		}
	}
	return firstReport
}

// needsElevation reports whether path was classified as needing elevation and
// is still the same entry inside its back-off.
func (p *FileProvider) needsElevation(path, identity string) bool {
	p.failedMu.Lock()
	defer p.failedMu.Unlock()
	record, ok := p.elevation[filepath.Clean(path)]
	if !ok || !p.failureNow().Before(record.until) {
		return false
	}
	return record.identity == "" || identity == "" || record.identity == identity
}

func (p *FileProvider) forgetRecoveryCandidate(path string) {
	p.recoveryMu.Lock()
	defer p.recoveryMu.Unlock()
	for _, state := range p.recoveryWalks {
		delete(state.ready, path)
	}
}

// CloseRecoveryWalks releases paused directory streams at API shutdown.
func (p *FileProvider) CloseRecoveryWalks() {
	p.recoveryMu.Lock()
	defer p.recoveryMu.Unlock()
	for _, state := range p.recoveryWalks {
		if state.walk != nil {
			_ = state.walk.Close()
		}
	}
	p.recoveryWalks = nil
}

// Temporary recovery resumes the same strict traversal after its observation
// budget. Only complete trees become candidates; ownership and age are checked
// again by Apply. Cursor state is acceleration, never deletion authority.
func (p *FileProvider) measureTemporaryRecovery(ctx context.Context, fsys cleanup.ResumableFileSystem, scope cleanup.ObservationScope, policy cleanup.ProviderPolicy) (cleanup.Preview, error) {
	p.recoveryMu.Lock()
	defer p.recoveryMu.Unlock()
	out := cleanup.Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version, MinAge: policy.MinAge, AllowSingleOvershoot: true}
	if p.recoveryWalks == nil {
		p.recoveryWalks = make(map[string]*recoveryWalkState)
	}
	roots := p.roots
	if len(scope.RootPaths) > 0 {
		roots = intersectRoots(roots, cleanRoots(scope.RootPaths))
	}
	measuring, cancel := context.WithTimeout(ctx, p.budget())
	defer cancel()
	deadline := p.observedNow(scope.Now).Add(p.budget())
	cutoff := p.now(scope).Add(-policy.MinAge)
	var elevationNotes []string
	for _, root := range roots {
		key := fmt.Sprintf("%s|%s", root, policy.MinAge)
		state := p.recoveryWalks[key]
		if state == nil {
			// Keep namespace-scoped rate reports from retaining unbounded handles.
			if len(p.recoveryWalks) >= 8 {
				out.Warnings = append(out.Warnings, "temporary measurement capacity reached; existing walks are retained")
				continue
			}
			entries, err := p.files.ReadDir(measuring, root)
			if err == nil {
				entries, err = p.expandGovernedUnits(measuring, entries)
			}
			if err != nil {
				out.Warnings = append(out.Warnings, cleanup.Redact(err.Error()))
				continue
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].ModTime.Equal(entries[j].ModTime) {
					return entries[i].Path < entries[j].Path
				}
				return entries[i].ModTime.Before(entries[j].ModTime)
			})
			state = &recoveryWalkState{entries: entries, ready: make(map[string]cleanup.PreviewItem)}
			p.recoveryWalks[key] = state
		}
		var active func(string) bool
		if p.leaseCheck == "temporary" {
			observed, err := p.temporaryActivityFilter(measuring, root)
			if err != nil {
				out.Warnings = append(out.Warnings, cleanup.Redact(err.Error()))
				out.BlockedReason = cleanup.Redact(err.Error())
				continue
			}
			active = observed
		} else {
			active = p.activePredicate(root)
		}
		for path, item := range state.ready {
			info, err := p.files.Stat(measuring, path)
			if err != nil || (item.Identity != "" && info.Identity != item.Identity) || !info.ModTime.Equal(item.ModTime) || active(path) || p.recentlyFailed(path, info.Identity) {
				delete(state.ready, path)
				continue
			}
			out.Items = append(out.Items, item)
		}
		backedOff := 0
		var needElevation []string
		for state.index < len(state.entries) {
			if sumPreviewBytes(out.Items) >= temporaryRecoveryReadyBytes || len(out.Items) >= temporaryRecoveryReadyItems {
				break
			}
			if measuring.Err() != nil || !p.observedNow(scope.Now).Before(deadline) {
				break
			}
			entry := state.entries[state.index]
			if state.walk == nil {
				if !p.withinConfiguredRoot(entry.Path) || p.isProtectedPath(entry.Path) || active(entry.Path) || (policy.MinAge > 0 && entry.ModTime.After(cutoff)) {
					state.index++
					continue
				}
				if p.recentlyFailed(entry.Path, entry.Identity) {
					backedOff++
					state.index++
					continue
				}
				if p.needsElevation(entry.Path, entry.Identity) {
					state.index++
					continue
				}
				walk, err := fsys.OpenWalk(measuring, entry.Path)
				if err != nil {
					if isPermissionError(err) {
						if p.rememberNeedsElevation(entry.Path, entry.Identity) {
							needElevation = append(needElevation, entry.Path)
						}
					} else {
						out.Warnings = append(out.Warnings, cleanup.Redact(err.Error()))
					}
					state.index++
					continue
				}
				state.walk, state.aggregate, state.protected = walk, entryAggregate{}, false
			}
			info, err := state.walk.Next(measuring)
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				break
			}
			if errors.Is(err, io.EOF) {
				_ = state.walk.Close()
				state.walk = nil
				state.index++
				current, statErr := p.files.Stat(ctx, entry.Path)
				if statErr != nil || (entry.Identity != "" && current.Identity != entry.Identity) || !current.ModTime.Equal(entry.ModTime) {
					continue
				}
				if state.protected || (policy.MinAge > 0 && state.aggregate.newest.After(cutoff)) {
					continue
				}
				// Recovery uses positive bytes to decide whether it can advance.
				// An empty file must not become the cached first candidate and
				// hide every reclaimable tree behind it on subsequent runs.
				if state.aggregate.bytes <= 0 {
					continue
				}
				if policy.MaxBytes > 0 && state.aggregate.bytes > policy.MaxBytes {
					out.Warnings = append(out.Warnings, "max reclaim limit reached")
					continue
				}
				item := p.previewItem(entry.Path, state.aggregate.bytes)
				item.Identity, item.ModTime = entry.Identity, entry.ModTime
				state.ready[entry.Path] = item
				out.Items = append(out.Items, item)
				// Keep measuring until about one batch is ready; the loop guard
				// publishes promptly once an oversized unit alone fills it.
				continue
			}
			if err != nil {
				// An unreadable descendant (another user's subtree, a mode-000
				// directory) makes the unit unmeasurable and unremovable for
				// this user. It is classified once, not retried every run.
				if isPermissionError(err) {
					if p.rememberNeedsElevation(entry.Path, entry.Identity) {
						needElevation = append(needElevation, entry.Path)
					}
				} else {
					out.Warnings = append(out.Warnings, cleanup.Redact(err.Error()))
				}
				_ = state.walk.Close()
				state.walk = nil
				state.index++
				continue
			}
			if !pathContains(entry.Path, info.Path) {
				state.protected = true
			}
			state.aggregate.bytes += info.Size
			if info.ModTime.After(state.aggregate.newest) {
				state.aggregate.newest = info.ModTime
			}
			if p.isProtectedPath(info.Path) || p.temporaryDescendantActive(info.Path) || info.Mode&(fs.ModeSocket|fs.ModeNamedPipe|fs.ModeDevice|fs.ModeCharDevice) != 0 {
				state.protected = true
			}
		}
		if state.index == len(state.entries) && len(state.ready) == 0 {
			delete(p.recoveryWalks, key)
		}
		if state.walk != nil || state.index < len(state.entries) {
			out.Warnings = append(out.Warnings, "temporary measurement pending; traversal position retained")
		}
		// After the pending warning, so a paused walk still reads as pending.
		if backedOff > 0 {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%d temporary entries skipped after a recent removal failure", backedOff))
		}
		if len(needElevation) > 0 {
			elevationNotes = append(elevationNotes, elevationNote(root, needElevation))
		}
	}
	// Largest complete unit first. The controller admits one oversized unit
	// only at the head of a batch, so byte-heavy trees must not queue behind
	// small ones that happened to be measured first.
	sort.Slice(out.Items, func(i, j int) bool {
		if out.Items[i].Bytes != out.Items[j].Bytes {
			return out.Items[i].Bytes > out.Items[j].Bytes
		}
		return out.Items[i].Path < out.Items[j].Path
	})
	if len(out.Items) == 0 && len(out.Warnings) > 0 {
		out.BlockedReason = out.Warnings[0]
	}
	// Entries this user can never remove are not a reason the provider could
	// not make progress, so they are notices, never the blocked reason.
	out.Notices = elevationNotes
	return out, nil
}

// elevationNote names newly classified entries once. Only base names are
// listed, bounded, so the note stays a single readable audit line.
func elevationNote(root string, paths []string) string {
	names := make([]string, 0, min(len(paths), 5))
	for _, path := range paths[:min(len(paths), 5)] {
		names = append(names, filepath.Base(path))
	}
	more := ""
	if len(paths) > len(names) {
		more = fmt.Sprintf(" and %d more", len(paths)-len(names))
	}
	return fmt.Sprintf("%d entries under %s hold content this user cannot read or remove and need elevation (%s%s); skipped for %s", len(paths), cleanup.Redact(root), strings.Join(names, ", "), more, elevationBackoff)
}
