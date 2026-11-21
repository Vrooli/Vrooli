package repository

import (
	"sort"
	"time"
)

// MaxProcessTimelineGroups is the largest bounded owner/command projection
// used by reports and alert rules. A saturated result cannot prove that the
// omitted tail is clear.
const MaxProcessTimelineGroups = 5000

// CompleteProcessRollupRange returns the minute-aligned subrange whose
// rollups are fully contained in q's half-open interval. A rollup summarizes
// its whole minute, so including a partial leading or trailing minute would
// leak observations outside the requested interval.
func CompleteProcessRollupRange(q ProcessTimelineQuery) (time.Time, time.Time) {
	start := q.Start.UTC().Truncate(time.Minute)
	if !start.Equal(q.Start.UTC()) {
		start = start.Add(time.Minute)
	}
	return start, q.End.UTC().Truncate(time.Minute)
}

type processTimelineAgg struct {
	owner               string
	comm                string
	pid                 int
	aggregated          bool
	cpuSum              float64
	cpuMax              float64
	cpuSeconds          float64
	rssSum              float64
	swapSum             float64
	gpuVRAMMax          float64
	sampleCount         int64
	firstSeen           time.Time
	lastSeen            time.Time
	swapMeasuredSamples int64
	swapUnknownSamples  int64
	swapFirstSeen       time.Time
	swapLastSeen        time.Time
}

// ProcessTimelineAccumulator merges raw and rolled-up process observations into
// ranked timeline entries.
type ProcessTimelineAccumulator struct {
	merged map[string]*processTimelineAgg
}

func NewProcessTimelineAccumulator() *ProcessTimelineAccumulator {
	return &ProcessTimelineAccumulator{merged: map[string]*processTimelineAgg{}}
}

func (a *ProcessTimelineAccumulator) AddRaw(owner, comm string, pid int, cpu, cpuSeconds float64, rss, swap int64, swapMeasured bool, gpuVRAMMB float64, ts time.Time) {
	row := a.entry(owner, comm)
	if row.sampleCount == 0 {
		row.pid = pid
	}
	row.addCPU(cpu, 1)
	row.cpuSeconds += cpuSeconds
	row.addRSS(rss, 1)
	if swapMeasured {
		row.swapSum += float64(swap)
		row.swapMeasuredSamples++
		row.addSwapWindow(ts, ts)
	} else {
		row.swapUnknownSamples++
	}
	row.addGPU(gpuVRAMMB)
	row.addWindow(ts, ts)
}

func (a *ProcessTimelineAccumulator) AddRollup(owner, comm string, avgCPU, maxCPU, cpuSeconds float64, avgRSS, avgSwap, count, swapMeasured, swapUnknown int64, maxGPUVRAMMB float64, minute time.Time) {
	row := a.entry(owner, comm)
	row.aggregated = true
	row.cpuSum += avgCPU * float64(count)
	row.cpuSeconds += cpuSeconds
	if maxCPU > row.cpuMax {
		row.cpuMax = maxCPU
	}
	row.addRSS(avgRSS, count)
	row.swapSum += float64(avgSwap) * float64(swapMeasured)
	row.swapMeasuredSamples += swapMeasured
	row.swapUnknownSamples += swapUnknown
	if swapMeasured > 0 {
		row.addSwapWindow(minute, minute.Add(time.Minute))
	}
	row.addGPU(maxGPUVRAMMB)
	row.sampleCount += count
	row.addWindow(minute, minute.Add(time.Minute))
}

func (a *ProcessTimelineAccumulator) Entries(top int, rank string) []ProcessTimelineEntry {
	entries := make([]ProcessTimelineEntry, 0, len(a.merged))
	for _, row := range a.merged {
		entries = append(entries, row.entry())
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if rank == "swap" && entries[i].SwapKB != entries[j].SwapKB {
			return entries[i].SwapKB > entries[j].SwapKB
		}
		if rank == "gpu" && entries[i].GPUVRAMMB != entries[j].GPUVRAMMB {
			return entries[i].GPUVRAMMB > entries[j].GPUVRAMMB
		}
		if rank == "rss" && entries[i].RSSKB != entries[j].RSSKB {
			return entries[i].RSSKB > entries[j].RSSKB
		}
		if rank == "cpu_seconds" && entries[i].CPUSeconds != entries[j].CPUSeconds {
			return entries[i].CPUSeconds > entries[j].CPUSeconds
		}
		if entries[i].CPUPct != entries[j].CPUPct {
			return entries[i].CPUPct > entries[j].CPUPct
		}
		return entries[i].RSSKB > entries[j].RSSKB
	})

	if top <= 0 {
		top = 20
	}
	if len(entries) > top {
		return entries[:top]
	}
	return entries
}

func (a *ProcessTimelineAccumulator) entry(owner, comm string) *processTimelineAgg {
	key := owner + "\x00" + comm
	row := a.merged[key]
	if row == nil {
		row = &processTimelineAgg{owner: owner, comm: comm}
		a.merged[key] = row
	}
	return row
}

func (a *processTimelineAgg) addCPU(cpu float64, count int64) {
	a.cpuSum += cpu
	if cpu > a.cpuMax {
		a.cpuMax = cpu
	}
	a.sampleCount += count
}

func (a *processTimelineAgg) addRSS(rss int64, count int64) {
	a.rssSum += float64(rss) * float64(count)
}

func (a *processTimelineAgg) addGPU(vram float64) {
	if vram > a.gpuVRAMMax {
		a.gpuVRAMMax = vram
	}
}

func (a *processTimelineAgg) addWindow(start, end time.Time) {
	if a.firstSeen.IsZero() || start.Before(a.firstSeen) {
		a.firstSeen = start
	}
	if end.After(a.lastSeen) {
		a.lastSeen = end
	}
}

func (a *processTimelineAgg) addSwapWindow(start, end time.Time) {
	if a.swapFirstSeen.IsZero() || start.Before(a.swapFirstSeen) {
		a.swapFirstSeen = start
	}
	if end.After(a.swapLastSeen) {
		a.swapLastSeen = end
	}
}

func (a *processTimelineAgg) entry() ProcessTimelineEntry {
	avg := 0.0
	if a.sampleCount > 0 {
		avg = a.cpuSum / float64(a.sampleCount)
	}
	avgRSS := int64(0)
	avgSwap := int64(0)
	if a.sampleCount > 0 {
		avgRSS = int64(a.rssSum / float64(a.sampleCount))
	}
	if a.swapMeasuredSamples > 0 {
		avgSwap = int64(a.swapSum / float64(a.swapMeasuredSamples))
	}
	return ProcessTimelineEntry{
		Owner:               a.owner,
		Comm:                a.comm,
		PID:                 a.pid,
		Aggregated:          a.aggregated,
		CPUPct:              avg,
		MaxCPUPct:           a.cpuMax,
		CPUSeconds:          a.cpuSeconds,
		RSSKB:               avgRSS,
		SwapKB:              avgSwap,
		SwapMeasuredSamples: a.swapMeasuredSamples,
		SwapUnknownSamples:  a.swapUnknownSamples,
		SwapCoverageStart:   a.swapFirstSeen,
		SwapCoverageEnd:     a.swapLastSeen,
		GPUVRAMMB:           a.gpuVRAMMax,
		SampleCount:         a.sampleCount,
		FirstSeen:           a.firstSeen,
		LastSeen:            a.lastSeen,
	}
}
