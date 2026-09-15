package threads

import "time"

type Budget struct {
	HourlyLimit, Used         int
	SpendCapCents, SpentCents int64
	Window                    time.Time
	OwnerNotified             bool
}

func (b *Budget) Allow(now time.Time, cost int64) bool {
	if b.Window.IsZero() || now.Sub(b.Window) >= time.Hour {
		b.Window = now
		b.Used = 0
		b.OwnerNotified = false
	}
	if b.HourlyLimit > 0 && b.Used >= b.HourlyLimit {
		return false
	}
	if b.SpendCapCents > 0 && b.SpentCents+cost > b.SpendCapCents {
		return false
	}
	b.Used++
	b.SpentCents += cost
	return true
}
