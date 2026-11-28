package services

import "time"

type fillSample struct {
	at    time.Time
	bytes int64
}

// fillRateWindow estimates growth using a bounded least-squares window. It is
// deliberately independent of filesystem and transport code so mount and
// governed-root samplers can share the same rate semantics.
type fillRateWindow struct {
	limit   int
	samples []fillSample
}

func newFillRateWindow(limit int) *fillRateWindow {
	if limit < 2 {
		limit = 2
	}
	return &fillRateWindow{limit: limit}
}

func (w *fillRateWindow) Add(at time.Time, bytes int64) (int64, time.Duration, bool) {
	if len(w.samples) > 0 && bytes < w.samples[len(w.samples)-1].bytes {
		w.samples = nil
	}
	w.samples = append(w.samples, fillSample{at: at, bytes: bytes})
	if len(w.samples) > w.limit {
		w.samples = w.samples[len(w.samples)-w.limit:]
	}
	if len(w.samples) < 2 {
		return 0, 0, false
	}
	first := w.samples[0].at
	last := w.samples[len(w.samples)-1].at
	window := last.Sub(first)
	if window <= 0 {
		return 0, window, false
	}
	origin := first.UnixNano()
	var sumX, sumY, sumXX, sumXY float64
	for _, sample := range w.samples {
		x := float64(sample.at.UnixNano()-origin) / float64(time.Second)
		y := float64(sample.bytes)
		sumX += x
		sumY += y
		sumXX += x * x
		sumXY += x * y
	}
	n := float64(len(w.samples))
	denominator := n*sumXX - sumX*sumX
	if denominator <= 0 {
		return 0, window, false
	}
	perSecond := (n*sumXY - sumX*sumY) / denominator
	if perSecond <= 0 {
		return 0, window, true
	}
	return int64(perSecond * 3600), window, true
}
