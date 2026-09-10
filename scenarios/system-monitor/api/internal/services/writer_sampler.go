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

// Defaults for WriterSamplerConfig. The total budget is the cost ceiling of one
// sample across every root: before it existed each expanded child carried its
// own 2 s budget, so a go-work directory with 16 children could spend 32 s of
// every 60 s tick walking the filesystem.
const (
	defaultWriterSampleInterval    = 60 * time.Second
	defaultWriterSampleTotalBudget = 2 * time.Second
	defaultWriterChildrenPerSample = 4
	defaultWriterRootBudget        = 2 * time.Second
)

// WriterSamplerConfig bounds a sampler's cost. Zero values take the defaults.
type WriterSamplerConfig struct {
	// Interval is the minimum time between two samples (the sensor tick that
	// .vrooli/repo-contract.json storage.recovery.sample_interval_seconds
	// declares).
	Interval time.Duration
	// TotalBudget is the wall-clock ceiling for one Sample call across all
	// roots. A root's own MeasureBudget is further clipped to whatever of the
	// total remains, and roots reached after it is spent are not walked at
	// all rather than being recorded as zero bytes.
	TotalBudget time.Duration
	// ChildrenPerSample is how many children of an ExpandChildren root are
	// measured per sample. Children are taken round-robin across samples, so a
	// child is measured every ceil(children / ChildrenPerSample) intervals;
	// each snapshot's ObservedAt and DeltaHours carry that real spacing.
	ChildrenPerSample int
}

func (c WriterSamplerConfig) withDefaults() WriterSamplerConfig {
	if c.Interval <= 0 {
		c.Interval = defaultWriterSampleInterval
	}
	if c.TotalBudget <= 0 {
		c.TotalBudget = defaultWriterSampleTotalBudget
	}
	if c.ChildrenPerSample <= 0 {
		c.ChildrenPerSample = defaultWriterChildrenPerSample
	}
	return c
}

// WriterSampler measures governed roots and delegates rate calculation to the
// same bounded estimator used by mount pressure. A partial walk is evidence,
// not a false complete measurement.
type WriterSampler struct {
	roots        []GovernedRoot
	config       WriterSamplerConfig
	state        map[string]*fillRateWindow
	last         map[string]int64
	lastAt       map[string]time.Time
	childCursor  map[string]int
	lastSampleAt time.Time
	// measure walks one root; tests substitute it to observe budget handling.
	measure func(ctx context.Context, root string) (int64, bool)
	// clock supplies the wall clock the budget is charged against. It is
	// separate from the `now` passed to Sample, which is the sensor's time.
	clock func() time.Time
}

func NewWriterSampler(roots []GovernedRoot) *WriterSampler {
	return NewWriterSamplerWithConfig(roots, WriterSamplerConfig{})
}

func NewWriterSamplerWithInterval(roots []GovernedRoot, interval time.Duration) *WriterSampler {
	return NewWriterSamplerWithConfig(roots, WriterSamplerConfig{Interval: interval})
}

func NewWriterSamplerWithConfig(roots []GovernedRoot, config WriterSamplerConfig) *WriterSampler {
	return &WriterSampler{
		roots:       append([]GovernedRoot(nil), roots...),
		config:      config.withDefaults(),
		state:       map[string]*fillRateWindow{},
		last:        map[string]int64{},
		lastAt:      map[string]time.Time{},
		childCursor: map[string]int{},
		measure:     directoryBytes,
		clock:       time.Now,
	}
}

// Sample measures the governed roots due this tick. It returns nil inside the
// sample interval. Roots that the total budget could not reach are omitted
// from the result instead of being reported with a fabricated size.
func (s *WriterSampler) Sample(ctx context.Context, now time.Time) []WriterSnapshot {
	if !s.lastSampleAt.IsZero() && now.Sub(s.lastSampleAt) < s.config.Interval {
		return nil
	}
	s.lastSampleAt = now
	roots := s.dueRoots()
	deadline := s.clock().Add(s.config.TotalBudget)
	out := make([]WriterSnapshot, 0, len(roots))
	for _, root := range roots {
		remaining := deadline.Sub(s.clock())
		if remaining <= 0 {
			break
		}
		budget := root.MeasureBudget
		if budget <= 0 {
			budget = defaultWriterRootBudget
		}
		if budget > remaining {
			budget = remaining
		}
		measureCtx, cancel := context.WithTimeout(ctx, budget)
		bytes, partial := s.measure(measureCtx, root.Root)
		cancel()
		out = append(out, s.record(root, now, bytes, partial))
	}
	return out
}

// dueRoots expands ExpandChildren roots into this tick's share of their
// children and returns the flat list of roots to measure.
func (s *WriterSampler) dueRoots() []GovernedRoot {
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
		if len(children) == 0 {
			continue
		}
		sort.Strings(children)
		take := s.config.ChildrenPerSample
		if take > len(children) {
			take = len(children)
		}
		start := s.childCursor[root.ID] % len(children)
		s.childCursor[root.ID] = (start + take) % len(children)
		for i := 0; i < take; i++ {
			child := children[(start+i)%len(children)]
			childRoot := root
			childRoot.ID = root.ID + "/" + child
			childRoot.Root = filepath.Join(root.Root, child)
			childRoot.ExpandChildren = false
			roots = append(roots, childRoot)
		}
	}
	return roots
}

func (s *WriterSampler) record(root GovernedRoot, now time.Time, bytes int64, partial bool) WriterSnapshot {
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
	return WriterSnapshot{Root: root.Root, RootID: root.ID, Mount: root.Mount, Bytes: bytes, DeltaBytes: delta, DeltaHours: now.Sub(previousAt).Hours(), BytesPerHour: rate, Partial: partial, Hot: hot, ObservedAt: now}
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
