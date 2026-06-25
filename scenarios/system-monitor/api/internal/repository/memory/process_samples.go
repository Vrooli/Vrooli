package memory

import (
	"context"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// SaveProcessSamples appends one cycle's rows to the in-memory store.
func (r *MemoryRepository) SaveProcessSamples(_ context.Context, samples []repository.ProcessSample) error {
	if len(samples) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processSamples = append(r.processSamples, samples...)
	return nil
}

// QueryProcessTimeline ranks consumers over [Start, End) across raw rows and
// per-minute rollups, mirroring the SQLite implementation.
func (r *MemoryRepository) QueryProcessTimeline(_ context.Context, q repository.ProcessTimelineQuery) ([]repository.ProcessTimelineEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	acc := repository.NewProcessTimelineAccumulator()
	for _, s := range r.processSamples {
		if !processSampleMatchesQuery(s, q) {
			continue
		}
		acc.AddRaw(s.Owner, s.Comm, s.PID, s.CPUPct, s.RSSKB, s.Timestamp)
	}

	for _, ru := range r.processRollups {
		if !processRollupMatchesQuery(ru, q) {
			continue
		}
		acc.AddRollup(ru.Owner, ru.Comm, ru.AvgCPUPct, ru.MaxCPUPct, ru.MaxRSSKB, ru.SampleCount, ru.Minute)
	}

	return acc.Entries(q.Top), nil
}

func processSampleMatchesQuery(s repository.ProcessSample, q repository.ProcessTimelineQuery) bool {
	return !s.Timestamp.Before(q.Start) && s.Timestamp.Before(q.End) && (q.Owner == "" || s.Owner == q.Owner)
}

func processRollupMatchesQuery(ru processRollup, q repository.ProcessTimelineQuery) bool {
	return !ru.Minute.Before(q.Start) && ru.Minute.Before(q.End) && (q.Owner == "" || ru.Owner == q.Owner)
}

// PruneProcessSamplesBefore drops raw rows older than cutoff.
func (r *MemoryRepository) PruneProcessSamplesBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.processSamples[:0]
	var deleted int64
	for _, s := range r.processSamples {
		if s.Timestamp.Before(cutoff) {
			deleted++
			continue
		}
		kept = append(kept, s)
	}
	r.processSamples = kept
	return deleted, nil
}

// PruneProcessRollupsBefore drops rollup rows older than cutoff.
func (r *MemoryRepository) PruneProcessRollupsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.processRollups[:0]
	var deleted int64
	for _, ru := range r.processRollups {
		if ru.Minute.Before(cutoff) {
			deleted++
			continue
		}
		kept = append(kept, ru)
	}
	r.processRollups = kept
	return deleted, nil
}

// RollupProcessSamples downsamples raw rows in [from, to) into per-owner/minute
// rollups and removes the consumed raw rows.
func (r *MemoryRepository) RollupProcessSamples(_ context.Context, from, to time.Time) (repository.RollupResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := repository.RollupResult{From: from, To: to}

	type bucket struct {
		minute time.Time
		owner  string
		comm   string
		cpuSum float64
		cpuMax float64
		rssSum int64
		rssMax int64
		count  int64
	}
	buckets := map[string]*bucket{}
	kept := r.processSamples[:0]
	for _, s := range r.processSamples {
		if s.Timestamp.Before(from) || !s.Timestamp.Before(to) {
			kept = append(kept, s)
			continue
		}
		minute := s.Timestamp.UTC().Truncate(time.Minute)
		k := minute.Format(time.RFC3339) + "\x00" + s.Owner + "\x00" + s.Comm
		b := buckets[k]
		if b == nil {
			b = &bucket{minute: minute, owner: s.Owner, comm: s.Comm}
			buckets[k] = b
		}
		b.cpuSum += s.CPUPct
		if s.CPUPct > b.cpuMax {
			b.cpuMax = s.CPUPct
		}
		b.rssSum += s.RSSKB
		if s.RSSKB > b.rssMax {
			b.rssMax = s.RSSKB
		}
		b.count++
		result.RawRowsConsumed++
	}
	if result.RawRowsConsumed == 0 {
		return result, nil
	}
	r.processSamples = kept

	for _, b := range buckets {
		r.mergeRollup(processRollup{
			Minute:      b.minute,
			Owner:       b.owner,
			Comm:        b.comm,
			AvgCPUPct:   b.cpuSum / float64(b.count),
			MaxCPUPct:   b.cpuMax,
			AvgRSSKB:    b.rssSum / b.count,
			MaxRSSKB:    b.rssMax,
			SampleCount: b.count,
		})
		result.RollupRows++
	}
	return result, nil
}

// mergeRollup upserts a rollup, merging into an existing (minute,owner,comm) row
// so overlapping rollup windows stay correct rather than double-counting.
func (r *MemoryRepository) mergeRollup(in processRollup) {
	for i := range r.processRollups {
		ex := &r.processRollups[i]
		if ex.Minute.Equal(in.Minute) && ex.Owner == in.Owner && ex.Comm == in.Comm {
			total := ex.SampleCount + in.SampleCount
			ex.AvgCPUPct = (ex.AvgCPUPct*float64(ex.SampleCount) + in.AvgCPUPct*float64(in.SampleCount)) / float64(total)
			ex.AvgRSSKB = (ex.AvgRSSKB*ex.SampleCount + in.AvgRSSKB*in.SampleCount) / total
			if in.MaxCPUPct > ex.MaxCPUPct {
				ex.MaxCPUPct = in.MaxCPUPct
			}
			if in.MaxRSSKB > ex.MaxRSSKB {
				ex.MaxRSSKB = in.MaxRSSKB
			}
			ex.SampleCount = total
			return
		}
	}
	r.processRollups = append(r.processRollups, in)
}
