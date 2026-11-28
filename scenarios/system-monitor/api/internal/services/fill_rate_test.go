package services

import (
	"testing"
	"time"
)

func TestFillRateWindowUsesBoundedLeastSquaresGrowth(t *testing.T) {
	w := newFillRateWindow(6)
	start := time.Unix(100, 0)
	for i := int64(0); i < 6; i++ {
		w.Add(start.Add(time.Duration(i)*time.Hour), 1000+i*2000)
	}
	rate, window, ok := w.Add(start.Add(6*time.Hour), 13_000)
	if !ok || window != 5*time.Hour || rate != 2_000 {
		t.Fatalf("rate=%d window=%s ok=%t, want 2000/5h/true", rate, window, ok)
	}
}

func TestFillRateWindowResetsAfterCounterDecrease(t *testing.T) {
	w := newFillRateWindow(6)
	start := time.Unix(100, 0)
	w.Add(start, 100)
	w.Add(start.Add(time.Hour), 200)
	rate, _, ok := w.Add(start.Add(2*time.Hour), 50)
	if ok || rate != 0 {
		t.Fatalf("after decrease rate=%d ok=%t, want zero measured rate", rate, ok)
	}
}
