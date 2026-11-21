package testfixtures

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
)

func Enabled(development, forceTest bool) bool { return development || forceTest }

const (
	ReferenceScaleV1Path = "/api/v1/dev/test-fixtures/performance/reference-scale-v1"
	ReferenceScaleV1ID   = "p1-reference-scale-v1"
	leaseHeader          = "X-Vrooli-Test-Lease-ID"
)

type referenceScaleRequest struct {
	LocalDate string `json:"local_date"`
}

type referenceScaleCounts struct {
	WorkItems      int `json:"work_items"`
	CalendarEvents int `json:"calendar_events"`
	Allocations    int `json:"allocations"`
	ManualActuals  int `json:"manual_actuals"`
	Goals          int `json:"goals"`
}

type referenceScaleResponse struct {
	Fixture             string               `json:"fixture"`
	LocalDate           string               `json:"local_date"`
	Counts              referenceScaleCounts `json:"counts"`
	AuthoredFingerprint string               `json:"authored_fingerprint"`
	LeaseID             string               `json:"lease_id"`
}

// ReferenceScaleHandler replaces only its own fixed namespace in the live
// routed test pool. It accepts no caller-selected SQL, table, or row counts.
func ReferenceScaleHandler(routed *database.RoutedDB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != ReferenceScaleV1Path {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !database.IsTestMode(r.Context()) {
			http.Error(w, "test-mode routing is required", http.StatusForbidden)
			return
		}
		if routed == nil {
			http.Error(w, "routed test database is unavailable", http.StatusServiceUnavailable)
			return
		}
		leaseID := routed.ActiveLeaseID()
		if leaseID == "" || strings.TrimSpace(r.Header.Get(leaseHeader)) == "" || r.Header.Get(leaseHeader) != leaseID {
			http.Error(w, "request does not match the active test lease", http.StatusConflict)
			return
		}
		if _, err := routed.PoolForContext(r.Context()); err != nil {
			http.Error(w, "active test lease is unavailable", http.StatusServiceUnavailable)
			return
		}
		if routed.ActiveLeaseID() != leaseID {
			http.Error(w, "test lease changed during request setup", http.StatusConflict)
			return
		}
		var request referenceScaleRequest
		decoder := json.NewDecoder(io.LimitReader(r.Body, 256))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "local_date is required", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			http.Error(w, "request must contain one JSON object", http.StatusBadRequest)
			return
		}
		anchor, err := time.Parse("2006-01-02", request.LocalDate)
		if err != nil || anchor.Format("2006-01-02") != request.LocalDate {
			http.Error(w, "local_date must use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		tx, err := routed.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, "active test lease is unavailable", http.StatusServiceUnavailable)
			return
		}
		if routed.ActiveLeaseID() != leaseID {
			_ = tx.Rollback()
			http.Error(w, "test lease changed during transaction setup", http.StatusConflict)
			return
		}
		counts, fingerprint, err := seedReferenceScale(r.Context(), tx, anchor)
		if err != nil {
			_ = tx.Rollback()
			http.Error(w, "reference fixture setup failed", http.StatusInternalServerError)
			return
		}
		if _, err := routed.PoolForContext(r.Context()); err != nil {
			_ = tx.Rollback()
			http.Error(w, "active test lease changed during fixture setup", http.StatusServiceUnavailable)
			return
		}
		if routed.ActiveLeaseID() != leaseID {
			_ = tx.Rollback()
			http.Error(w, "test lease changed during fixture setup", http.StatusConflict)
			return
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, "reference fixture setup failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(referenceScaleResponse{
			Fixture: ReferenceScaleV1ID, LocalDate: request.LocalDate,
			Counts: counts, AuthoredFingerprint: fingerprint, LeaseID: leaseID,
		})
	})
}

func seedReferenceScale(ctx context.Context, tx *sql.Tx, anchor time.Time) (referenceScaleCounts, string, error) {
	const (
		workCount       = 1000
		eventCount      = 10000
		allocationCount = 10000
		actualCount     = 10000
		prefix          = "p1-reference-scale-v1-"
	)
	var counts referenceScaleCounts
	for _, statement := range []string{
		`DELETE FROM manual_actuals WHERE id LIKE 'p1-reference-scale-v1-actual-%'`,
		`DELETE FROM calendar_allocations WHERE id LIKE 'p1-reference-scale-v1-allocation-%'`,
		`DELETE FROM calendar_events WHERE id LIKE 'p1-reference-scale-v1-event-%'`,
		`DELETE FROM goals WHERE id = 'p1-reference-scale-v1-goal'`,
		`DELETE FROM work_items WHERE id LIKE 'p1-reference-scale-v1-work-%'`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return counts, "", err
		}
	}
	h := sha256.New()
	if err := seedWork(ctx, tx, h, anchor); err != nil {
		return counts, "", err
	}
	if err := seedEvents(ctx, tx, h, anchor); err != nil {
		return counts, "", err
	}
	if err := seedAllocationsAndActuals(ctx, tx, h, anchor); err != nil {
		return counts, "", err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO goals (id,title,purpose,status,progress_method,progress_basis_points,target_basis_points,target_date,completed_date,created_at,updated_at,revision) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, prefix+"goal", "P1 Reference outcome", "Fixed test fixture", "active", "manual", 0, 10000, nil, nil, anchor.Unix(), anchor.Unix(), 1); err != nil {
		return counts, "", err
	}
	writeFingerprint(h, "goal", prefix+"goal", "P1 Reference outcome", "Fixed test fixture", "active", "manual", "0", "10000", anchor.Format("2006-01-02"))
	var err error
	counts, err = countReferenceScaleRows(ctx, tx)
	if err != nil {
		return counts, "", err
	}
	want := referenceScaleCounts{WorkItems: workCount, CalendarEvents: eventCount, Allocations: allocationCount, ManualActuals: actualCount, Goals: 1}
	if counts != want {
		return referenceScaleCounts{}, "", fmt.Errorf("fixture row counts = %+v, want %+v", counts, want)
	}
	return counts, hex.EncodeToString(h.Sum(nil)), nil
}

func countReferenceScaleRows(ctx context.Context, tx *sql.Tx) (referenceScaleCounts, error) {
	var counts referenceScaleCounts
	for _, item := range []struct {
		query   string
		pattern string
		dest    *int
	}{
		{`SELECT count(*) FROM work_items WHERE id LIKE ?`, "p1-reference-scale-v1-work-%", &counts.WorkItems},
		{`SELECT count(*) FROM calendar_events WHERE id LIKE ?`, "p1-reference-scale-v1-event-%", &counts.CalendarEvents},
		{`SELECT count(*) FROM calendar_allocations WHERE id LIKE ?`, "p1-reference-scale-v1-allocation-%", &counts.Allocations},
		{`SELECT count(*) FROM manual_actuals WHERE id LIKE ?`, "p1-reference-scale-v1-actual-%", &counts.ManualActuals},
		{`SELECT count(*) FROM goals WHERE id = ?`, "p1-reference-scale-v1-goal", &counts.Goals},
	} {
		if err := tx.QueryRowContext(ctx, item.query, item.pattern).Scan(item.dest); err != nil {
			return referenceScaleCounts{}, err
		}
	}
	return counts, nil
}

func seedWork(ctx context.Context, tx *sql.Tx, h hash.Hash, anchor time.Time) error {
	statement, err := tx.PrepareContext(ctx, `INSERT INTO work_items (id,title,description,remaining_minutes,original_estimate_minutes,status,completed_at,snoozed_until,source_label,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("p1-reference-scale-v1-work-%04d", i)
		title := fmt.Sprintf("P1 Reference work %04d", i+1)
		remaining := 30 + (i % 8 * 15)
		if _, err := statement.ExecContext(ctx, id, title, "Deterministic reference-scale performance fixture", remaining, remaining, "open", "", "", ReferenceScaleV1ID, anchor.UTC().Format(time.RFC3339), anchor.UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		writeFingerprint(h, "work", id, title, "Deterministic reference-scale performance fixture", fmt.Sprint(remaining), fmt.Sprint(remaining), "open", ReferenceScaleV1ID, anchor.UTC().Format(time.RFC3339))
	}
	return nil
}

func seedEvents(ctx context.Context, tx *sql.Tx, h hash.Hash, anchor time.Time) error {
	statement, err := tx.PrepareContext(ctx, `INSERT INTO calendar_events (id,title,subject,notes,availability,timezone,all_day,start_date,end_date_exclusive,start_at,end_at,range_start,range_end,provider,provider_calendar_id,provider_event_id,occurrence_id,revision,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for i := 0; i < 10000; i++ {
		day := anchor.AddDate(0, 0, i%731-365).Format("2006-01-02")
		next := anchor.AddDate(0, 0, i%731-364).Format("2006-01-02")
		id := fmt.Sprintf("p1-reference-scale-v1-event-%05d", i)
		title := fmt.Sprintf("P1 Reference event %05d", i+1)
		if _, err := statement.ExecContext(ctx, id, title, "", "", "busy", "UTC", 1, day, next, "", "", day, next, "", "", "", "", 1, anchor.UTC().Format(time.RFC3339), anchor.UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		writeFingerprint(h, "event", id, title, "busy", "UTC", "all-day", day, next, "revision-1")
	}
	return nil
}

func seedAllocationsAndActuals(ctx context.Context, tx *sql.Tx, h hash.Hash, anchor time.Time) error {
	allocation, err := tx.PrepareContext(ctx, `INSERT INTO calendar_allocations (id,work_item_id,local_date,start_minutes,duration_minutes,state,created_at) VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer allocation.Close()
	actual, err := tx.PrepareContext(ctx, `INSERT INTO manual_actuals (id,work_item_id,title,local_date,reported_minutes,certainty,note,allocation_id,created_at,revision) VALUES (?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer actual.Close()
	for i := 0; i < 10000; i++ {
		day := anchor.AddDate(0, 0, i%365-182).Format("2006-01-02")
		workID := fmt.Sprintf("p1-reference-scale-v1-work-%04d", i%1000)
		allocationID := fmt.Sprintf("p1-reference-scale-v1-allocation-%05d", i)
		actualID := fmt.Sprintf("p1-reference-scale-v1-actual-%05d", i)
		start := 480 + (i%10)*50
		if _, err := allocation.ExecContext(ctx, allocationID, workID, day, start, 40, "accepted", anchor.UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		title := fmt.Sprintf("P1 Reference actual %05d", i+1)
		if _, err := actual.ExecContext(ctx, actualID, workID, title, day, 40, "high", "", allocationID, anchor.Unix(), 1); err != nil {
			return err
		}
		writeFingerprint(h, "allocation", allocationID, workID, day, fmt.Sprint(start), "40", "accepted")
		writeFingerprint(h, "actual", actualID, workID, title, day, "40", "high", allocationID, fmt.Sprint(anchor.Unix()), "revision-1")
	}
	return nil
}

func writeFingerprint(h hash.Hash, fields ...string) {
	for _, field := range fields {
		_, _ = io.WriteString(h, field)
		_, _ = io.WriteString(h, "\x00")
	}
	_, _ = io.WriteString(h, "\n")
}
