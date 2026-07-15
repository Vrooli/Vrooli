package repository

import (
	"sort"
	"time"
)

type processTimelineAgg struct {
	owner       string
	comm        string
	pid         int
	aggregated  bool
	cpuSum      float64
	cpuMax      float64
	rssMax      int64
	gpuVRAMMax  float64
	sampleCount int64
	firstSeen   time.Time
	lastSeen    time.Time
}

// ProcessTimelineAccumulator merges raw and rolled-up process observations into
// ranked timeline entries.
type ProcessTimelineAccumulator struct {
	merged map[string]*processTimelineAgg
}

func NewProcessTimelineAccumulator() *ProcessTimelineAccumulator {
	return &ProcessTimelineAccumulator{merged: map[string]*processTimelineAgg{}}
}

func (a *ProcessTimelineAccumulator) AddRaw(owner, comm string, pid int, cpu float64, rss int64, gpuVRAMMB float64, ts time.Time) {
	row := a.entry(owner, comm)
	if row.sampleCount == 0 {
		row.pid = pid
	}
	row.addCPU(cpu, 1)
	row.addRSS(rss)
	row.addGPU(gpuVRAMMB)
	row.addWindow(ts, ts)
}

func (a *ProcessTimelineAccumulator) AddRollup(owner, comm string, avgCPU, maxCPU float64, maxRSS, count int64, minute time.Time) {
	row := a.entry(owner, comm)
	row.aggregated = true
	row.cpuSum += avgCPU * float64(count)
	if maxCPU > row.cpuMax {
		row.cpuMax = maxCPU
	}
	row.addRSS(maxRSS)
	row.sampleCount += count
	row.addWindow(minute, minute.Add(time.Minute))
}

func (a *ProcessTimelineAccumulator) Entries(top int, rank string) []ProcessTimelineEntry {
	entries := make([]ProcessTimelineEntry, 0, len(a.merged))
	for _, row := range a.merged {
		entries = append(entries, row.entry())
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if rank == "gpu" && entries[i].GPUVRAMMB != entries[j].GPUVRAMMB {
			return entries[i].GPUVRAMMB > entries[j].GPUVRAMMB
		}
		if rank == "rss" && entries[i].RSSKB != entries[j].RSSKB {
			return entries[i].RSSKB > entries[j].RSSKB
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

func (a *processTimelineAgg) addRSS(rss int64) {
	if rss > a.rssMax {
		a.rssMax = rss
	}
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

func (a *processTimelineAgg) entry() ProcessTimelineEntry {
	avg := 0.0
	if a.sampleCount > 0 {
		avg = a.cpuSum / float64(a.sampleCount)
	}
	return ProcessTimelineEntry{
		Owner:       a.owner,
		Comm:        a.comm,
		PID:         a.pid,
		Aggregated:  a.aggregated,
		CPUPct:      avg,
		RSSKB:       a.rssMax,
		GPUVRAMMB:   a.gpuVRAMMax,
		SampleCount: a.sampleCount,
		FirstSeen:   a.firstSeen,
		LastSeen:    a.lastSeen,
	}
}
