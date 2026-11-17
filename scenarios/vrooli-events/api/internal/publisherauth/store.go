package publisherauth

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

const schema = `
CREATE TABLE IF NOT EXISTS publisher_observations (
  claimed_source TEXT NOT NULL,
  family TEXT NOT NULL,
  class TEXT NOT NULL,
  authenticated_source TEXT NOT NULL DEFAULT '',
  count INTEGER NOT NULL DEFAULT 0,
  first_seen TEXT NOT NULL,
  last_seen TEXT NOT NULL,
  sample_event_id TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (claimed_source, family, class, authenticated_source)
);
`

// Schema returns the observation schema for the routed database registry.
func Schema() string { return schema }

// FlushInterval is how often counts reach the database. A report flushes first,
// so it is always current; only a crash loses up to this much counting.
const FlushInterval = 30 * time.Second

// Flush writes the counted events to the database.
func (o *Observer) Flush(ctx context.Context) error {
	o.mu.Lock()
	pending := o.pending
	o.pending = map[key]*tally{}
	o.mu.Unlock()
	var firstErr error
	for k, t := range pending {
		_, err := o.db.ExecContext(ctx, `
INSERT INTO publisher_observations (claimed_source, family, class, authenticated_source, count, first_seen, last_seen, sample_event_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(claimed_source, family, class, authenticated_source) DO UPDATE SET
  count = count + excluded.count, last_seen = excluded.last_seen, sample_event_id = excluded.sample_event_id`,
			k.claimed, string(k.family), string(k.class), k.authenticated, t.count, stamp(t.first), stamp(t.last), t.sample)
		if err != nil {
			// Keep what could not be written for the next flush.
			o.restore(k, t)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (o *Observer) restore(k key, t *tally) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if existing := o.pending[k]; existing != nil {
		existing.count += t.count
		if t.first.Before(existing.first) {
			existing.first = t.first
		}
		return
	}
	o.pending[k] = t
}

// Run flushes every FlushInterval until ctx ends, then once more.
func (o *Observer) Run(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := o.Flush(flushCtx); err != nil && onError != nil {
				onError(err)
			}
			cancel()
			return
		case <-ticker.C:
			if err := o.Flush(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// Row is one counted combination.
type Row struct {
	ClaimedSource string `json:"claimed_source"`
	Family        Family `json:"family"`
	Class         Class  `json:"class"`
	// AuthenticatedSource is the principal's own source, set for authenticated and
	// mismatch rows.
	AuthenticatedSource string `json:"authenticated_source,omitempty"`
	Count               int64  `json:"count"`
	FirstSeen           string `json:"first_seen"`
	LastSeen            string `json:"last_seen"`
	// SampleEventID is the latest event counted in the row, to find it in the store.
	SampleEventID string `json:"sample_event_id,omitempty"`
}

// Report is what the observation endpoint serves.
type Report struct {
	// Mode is "observe": every event is accepted whatever it presents.
	Mode string `json:"mode"`
	// Since is when the first event was counted.
	Since string `json:"since,omitempty"`
	// Totals counts accepted events by class.
	Totals map[Class]int64 `json:"totals"`
	// ByFamily counts them by event family and class, the high-consequence
	// families first, so the operator sees what enforcing a family would refuse.
	ByFamily []FamilyTotals `json:"by_family"`
	Rows     []Row          `json:"rows"`
}

// FamilyTotals is the per-class count of one family.
type FamilyTotals struct {
	Family Family          `json:"family"`
	Totals map[Class]int64 `json:"totals"`
}

var familyOrder = []Family{Answer, Decision, Notification, Other}

// Report flushes and reads every counted combination.
func (o *Observer) Report(ctx context.Context) (Report, error) {
	if err := o.Flush(ctx); err != nil {
		return Report{}, err
	}
	rows, err := o.db.QueryContext(ctx, `SELECT claimed_source, family, class, authenticated_source, count, first_seen, last_seen, sample_event_id FROM publisher_observations`)
	if err != nil {
		return Report{}, err
	}
	defer rows.Close()
	report := Report{Mode: "observe", Totals: map[Class]int64{}}
	byFamily := map[Family]map[Class]int64{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ClaimedSource, &r.Family, &r.Class, &r.AuthenticatedSource, &r.Count, &r.FirstSeen, &r.LastSeen, &r.SampleEventID); err != nil {
			return Report{}, err
		}
		report.Rows = append(report.Rows, r)
		report.Totals[r.Class] += r.Count
		if byFamily[r.Family] == nil {
			byFamily[r.Family] = map[Class]int64{}
		}
		byFamily[r.Family][r.Class] += r.Count
		if report.Since == "" || r.FirstSeen < report.Since {
			report.Since = r.FirstSeen
		}
	}
	if err := rows.Err(); err != nil {
		return Report{}, err
	}
	for _, class := range Classes {
		if _, ok := report.Totals[class]; !ok {
			report.Totals[class] = 0
		}
	}
	for _, family := range familyOrder {
		totals := map[Class]int64{}
		for _, class := range Classes {
			totals[class] = byFamily[family][class]
		}
		report.ByFamily = append(report.ByFamily, FamilyTotals{Family: family, Totals: totals})
	}
	sort.Slice(report.Rows, func(i, j int) bool {
		a, b := report.Rows[i], report.Rows[j]
		if fa, fb := rank(a.Family), rank(b.Family); fa != fb {
			return fa < fb
		}
		if a.ClaimedSource != b.ClaimedSource {
			return a.ClaimedSource < b.ClaimedSource
		}
		return a.Class < b.Class
	})
	return report, nil
}

func rank(f Family) int {
	for i, family := range familyOrder {
		if family == f {
			return i
		}
	}
	return len(familyOrder)
}

// Handler serves the report as JSON. It is read-only.
func (o *Observer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o == nil {
			http.Error(w, "publisher observation is not enabled", http.StatusServiceUnavailable)
			return
		}
		report, err := o.Report(r.Context())
		if err != nil {
			http.Error(w, "publisher observation could not be read", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	})
}
