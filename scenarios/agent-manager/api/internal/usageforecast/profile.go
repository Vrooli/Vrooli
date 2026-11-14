package usageforecast

import "time"

const (
	hoursPerWeek = 7 * 24
	// profileFloor is the share of the weight spread evenly over the week, so
	// an hour never seen active still carries some burn.
	profileFloor = 0.2
	// profileHistory is the history a profile needs before it is trusted in
	// full; with less, it is mixed toward uniform in proportion, so one
	// window of history cannot make a spiky profile.
	profileHistory = 4 * 7 * 24 * time.Hour
	// projectionLimit bounds how far ahead a run-out time is searched.
	projectionLimit = 60 * 24 * time.Hour
)

// Profile is the operator's burn by hour of the week (UTC), learned from the
// scope's own used-percent increases. Each hour of the week has few samples,
// so its share is shrunk halfway toward the share of its hour of the day.
// Weights average 1, so a profile with no history is uniform and projects
// exactly the linear burn.
type Profile struct {
	weight [hoursPerWeek]float64
}

// burnProfile attributes every used-percent increase observed at or before
// asOf to the hour of the week it was first reported in.
func burnProfile(windows []*Window, asOf time.Time) Profile {
	var p Profile
	var total float64
	first := asOf
	for _, w := range windows {
		if o := w.origin(); o.Before(first) {
			first = o
		}
		previous := w.valueAt(w.origin())
		for _, point := range w.points {
			if point.At.After(asOf) {
				break
			}
			if delta := point.Percent - previous; delta > 0 {
				p.weight[hourOfWeek(point.At)] += delta
				total += delta
			}
			previous = point.Percent
		}
	}
	trust := (1 - profileFloor) * min(1, float64(asOf.Sub(first))/float64(profileHistory))
	var day [24]float64
	for h, burn := range p.weight {
		day[h%24] += burn / 7
	}
	for h := range p.weight {
		share := 1.0 / hoursPerWeek
		if total > 0 {
			share = (p.weight[h] + day[h%24]) / 2 / total
		}
		p.weight[h] = trust*share*hoursPerWeek + 1 - trust
	}
	return p
}

func hourOfWeek(t time.Time) int {
	t = t.UTC()
	return int(t.Weekday())*24 + t.Hour()
}

// Mass is the profile weight between from and to in weighted hours.
func (p Profile) Mass(from, to time.Time) float64 {
	var mass float64
	for t := from; t.Before(to); {
		next := t.Truncate(time.Hour).Add(time.Hour)
		if next.After(to) {
			next = to
		}
		mass += p.weight[hourOfWeek(t)] * next.Sub(t).Hours()
		t = next
	}
	return mass
}

// until is when Mass(from, t) reaches mass, or nil beyond projectionLimit.
func (p Profile) until(from time.Time, mass float64) *time.Time {
	for t := from; t.Before(from.Add(projectionLimit)); {
		next := t.Truncate(time.Hour).Add(time.Hour)
		step := p.weight[hourOfWeek(t)] * next.Sub(t).Hours()
		if step >= mass {
			at := t.Add(time.Duration(mass / p.weight[hourOfWeek(t)] * float64(time.Hour)))
			return &at
		}
		mass -= step
		t = next
	}
	return nil
}
