package home

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Bounds of the walk that finds a run folder's newest file (rulings F3). One
// run folder on the real home holds 2.8 million files; the walk stops at
// whichever bound it meets first and says so.
const (
	walkEntries = 10000
	walkTime    = 250 * time.Millisecond
	walkTTL     = 2 * time.Minute
)

// newestWrite is what one walk found.
type newestWrite struct {
	// All is the newest file in the folder.
	All Time
	// Before is the newest file at or before the cutoff the walk was given
	// (the owning commission's interval end); equal to All without one.
	Before Time
	// After is the newest file after the cutoff, unknown when none.
	After Time
}

type walkKey struct {
	path, skip string
	cutoff     time.Time
}

type walkResult struct {
	at     time.Time
	result newestWrite
}

// walkCache keeps walk results for walkTTL so a page of reads walks each
// folder once.
type walkCache struct {
	mu      sync.Mutex
	entries map[walkKey]walkResult
}

// newest walks dir, skipping the folder skip (empty: none).
func (c *walkCache) newest(dir, skip string, cutoff, now time.Time) newestWrite {
	key := walkKey{dir, skip, cutoff}
	c.mu.Lock()
	if hit, ok := c.entries[key]; ok && now.Sub(hit.at) < walkTTL && !now.Before(hit.at) {
		c.mu.Unlock()
		return hit.result
	}
	c.mu.Unlock()
	result := walkNewest(dir, skip, cutoff, walkEntries, time.Now().Add(walkTime))
	c.mu.Lock()
	if c.entries == nil {
		c.entries = map[walkKey]walkResult{}
	}
	for k, v := range c.entries {
		if now.Sub(v.at) >= walkTTL {
			delete(c.entries, k)
		}
	}
	c.entries[key] = walkResult{at: now, result: result}
	c.mu.Unlock()
	return result
}

var errWalkCut = errors.New("walk cut short")

// walkNewest finds the newest regular file under dir (hidden entries and the
// folder skip excluded). Folder times are not writes. A zero cutoff splits
// nothing. When the walk stops at a bound the values found so far are
// reported as "at least".
func walkNewest(dir, skip string, cutoff time.Time, maxEntries int, deadline time.Time) newestWrite {
	none := Time{Evidence: Unknown("no files in " + dir)}
	out := newestWrite{All: none, Before: none, After: Time{Evidence: Unknown("nothing written after the end")}}
	visited := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == dir {
				out.All.Reason, out.Before.Reason = "cannot read "+dir+": "+err.Error(), "cannot read "+dir+": "+err.Error()
			}
			return nil
		}
		if visited++; visited > maxEntries || (visited%256 == 0 && time.Now().After(deadline)) {
			return errWalkCut
		}
		if path != dir && (strings.HasPrefix(d.Name(), ".") || path == skip) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		t := Time{Value: info.ModTime(), Evidence: From(path, info.ModTime())}
		out.All = later(out.All, t)
		if cutoff.IsZero() || !t.Value.After(cutoff) {
			out.Before = later(out.Before, t)
		} else {
			out.After = later(out.After, t)
		}
		return nil
	})
	if errors.Is(err, errWalkCut) {
		for _, t := range []*Time{&out.All, &out.Before, &out.After} {
			if t.Known {
				t.Reason = fmt.Sprintf("at least: the walk stopped after %d entries or %s; newest file seen %s", min(visited, maxEntries), walkTime, t.Source)
				t.Source = dir + " (walk cut short)"
			}
		}
	}
	return out
}
