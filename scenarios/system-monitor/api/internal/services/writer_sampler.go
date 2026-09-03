package services

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// GovernedRoot is a declarative root supplied by storage policy. The sampler
// does not own deletion and therefore cannot turn an observation into a side
// effect.
type GovernedRoot struct {
	ID                 string
	Root               string
	Mount              string
	HotWriterBytesHour int64
	MeasureBudget      time.Duration
	ExpandChildren     bool
}

type WriterSnapshot struct {
	// Root is the absolute governed path consumed by storage-manager. RootID is
	// the stable policy identity used for durable attribution.
	Root, RootID, Mount string
	Bytes               int64
	DeltaBytes          int64
	DeltaHours          float64
	BytesPerHour        int64
	Partial, Hot        bool
	ObservedAt          time.Time
}

// WriterSampler measures governed roots and delegates rate calculation to the
// same bounded estimator used by mount pressure. A partial walk is evidence,
// not a false complete measurement.
type WriterSampler struct {
	roots        []GovernedRoot
	state        map[string]*fillRateWindow
	last         map[string]int64
	lastAt       map[string]time.Time
	sampleEvery  time.Duration
	lastSampleAt time.Time
}

func NewWriterSampler(roots []GovernedRoot) *WriterSampler {
	return NewWriterSamplerWithInterval(roots, 60*time.Second)
}

func NewWriterSamplerWithInterval(roots []GovernedRoot, interval time.Duration) *WriterSampler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &WriterSampler{roots: append([]GovernedRoot(nil), roots...), state: map[string]*fillRateWindow{}, last: map[string]int64{}, lastAt: map[string]time.Time{}, sampleEvery: interval}
}

func (s *WriterSampler) Sample(ctx context.Context, now time.Time) []WriterSnapshot {
	if !s.lastSampleAt.IsZero() && now.Sub(s.lastSampleAt) < s.sampleEvery {
		return nil
	}
	s.lastSampleAt = now
	roots := make([]GovernedRoot, 0, len(s.roots))
	for _, root := range s.roots {
		if !root.ExpandChildren {
			roots = append(roots, root)
			continue
		}
		entries, err := os.ReadDir(root.Root)
		if err != nil {
			continue
		}
		children := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				children = append(children, entry.Name())
			}
		}
		sort.Strings(children)
		for _, child := range children {
			childRoot := root
			childRoot.ID = root.ID + "/" + child
			childRoot.Root = filepath.Join(root.Root, child)
			childRoot.ExpandChildren = false
			roots = append(roots, childRoot)
		}
	}
	out := make([]WriterSnapshot, 0, len(roots))
	for _, root := range roots {
		budget := root.MeasureBudget
		if budget <= 0 {
			budget = 2 * time.Second
		}
		measureCtx, cancel := context.WithTimeout(ctx, budget)
		bytes, partial := directoryBytes(measureCtx, root.Root)
		cancel()
		window := s.state[root.ID]
		if window == nil {
			window = newFillRateWindow(6)
			s.state[root.ID] = window
		}
		rate, duration, ok := window.Add(now, bytes)
		previous, previousAt := s.last[root.ID], s.lastAt[root.ID]
		delta := bytes - previous
		if delta < 0 {
			delta = 0
		}
		s.last[root.ID], s.lastAt[root.ID] = bytes, now
		limit := root.HotWriterBytesHour
		hot := ok && limit > 0 && rate > limit && duration > 0
		out = append(out, WriterSnapshot{Root: root.Root, RootID: root.ID, Mount: root.Mount, Bytes: bytes, DeltaBytes: delta, DeltaHours: now.Sub(previousAt).Hours(), BytesPerHour: rate, Partial: partial, Hot: hot, ObservedAt: now})
	}
	return out
}

func directoryBytes(ctx context.Context, root string) (int64, bool) {
	var total int64
	partial := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			partial = true
			return ctx.Err()
		}
		if err != nil {
			partial = true
			return nil
		}
		if entry.Type().IsRegular() {
			info, statErr := entry.Info()
			if statErr != nil {
				partial = true
				return nil
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil && ctx.Err() == nil {
		partial = true
	}
	return total, partial
}
