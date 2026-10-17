package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"

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
			if err != nil || (item.Identity != "" && info.Identity != item.Identity) || !info.ModTime.Equal(item.ModTime) || active(path) {
				delete(state.ready, path)
				continue
			}
			out.Items = append(out.Items, item)
		}
		if len(out.Items) > 0 {
			continue
		}
		for state.index < len(state.entries) {
			if measuring.Err() != nil || !p.observedNow(scope.Now).Before(deadline) {
				break
			}
			entry := state.entries[state.index]
			if state.walk == nil {
				if !p.withinConfiguredRoot(entry.Path) || p.isProtectedPath(entry.Path) || active(entry.Path) || (policy.MinAge > 0 && entry.ModTime.After(cutoff)) {
					state.index++
					continue
				}
				walk, err := fsys.OpenWalk(measuring, entry.Path)
				if err != nil {
					out.Warnings = append(out.Warnings, cleanup.Redact(err.Error()))
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
				if policy.MaxBytes > 0 && state.aggregate.bytes > policy.MaxBytes {
					out.Warnings = append(out.Warnings, "max reclaim limit reached")
					continue
				}
				item := p.previewItem(entry.Path, state.aggregate.bytes)
				item.Identity, item.ModTime = entry.Identity, entry.ModTime
				state.ready[entry.Path] = item
				out.Items = append(out.Items, item)
				// Publish one complete unit promptly, including an oversized unit.
				break
			}
			if err != nil {
				out.Warnings = append(out.Warnings, cleanup.Redact(err.Error()))
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
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Path < out.Items[j].Path })
	if len(out.Items) == 0 && len(out.Warnings) > 0 {
		out.BlockedReason = out.Warnings[0]
	}
	return out, nil
}
