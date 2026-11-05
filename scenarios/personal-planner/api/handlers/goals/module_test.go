package goals

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestTargetDateRoutesMapValidationConflictAndHistoryRead(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO goals (id,title,purpose,status,progress_method,progress_basis_points,target_basis_points,target_date,created_at,updated_at,revision) VALUES ('g-1','Ship','', 'active','manual',0,10000,'2026-11-01',1,1,3)`); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(database.NewFromPrimary(sqlDB), schedule.System(), log.New(io.Discard, "", 0)).Mount(router)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
		return recorder
	}
	invalid := request(http.MethodPost, "/api/v1/goals/g-1/target-date", `{"target_date":"2026-12-01","expected_revision":3,"reason":" "}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid request status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	conflict := request(http.MethodPost, "/api/v1/goals/g-1/target-date", `{"target_date":"2026-12-01","expected_revision":2,"reason":"Reschedule"}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("stale request status=%d body=%s", conflict.Code, conflict.Body.String())
	}
	if _, err := sqlDB.Exec(`INSERT INTO goal_target_date_history (goal_id,previous_target_date,target_date,reason,resulting_revision,changed_at) VALUES ('g-1','2026-10-01','2026-11-01','Original plan',3,'2026-10-20T15:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	read := request(http.MethodGet, "/api/v1/goals/g-1/target-date/history", "")
	if read.Code != http.StatusOK {
		t.Fatalf("history status=%d body=%s", read.Code, read.Body.String())
	}
	var body struct {
		History []struct {
			PreviousTargetDate string `json:"previous_target_date"`
			TargetDate         string `json:"target_date"`
			Reason             string `json:"reason"`
			Revision           int64  `json:"revision"`
		} `json:"history"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.History) != 1 || body.History[0].PreviousTargetDate != "2026-10-01" || body.History[0].TargetDate != "2026-11-01" || body.History[0].Reason != "Original plan" || body.History[0].Revision != 3 {
		t.Fatalf("history response=%s", read.Body.String())
	}
}

func TestMeasuredGoalRESTCreateReadRoundTripAndValidation(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(database.NewFromPrimary(sqlDB), schedule.System(), log.New(io.Discard, "", 0)).Mount(router)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
		return recorder
	}
	createCases := []struct{ title, date, recurrence, period string }{
		{"First $1,000 after costs before tax", "2026-11-30", "one_time", "once"},
		{"Recurring $1,000 per month", "2026-12-31", "recurring", "month"},
	}
	for _, tc := range createCases {
		body, _ := json.Marshal(map[string]any{"title": tc.title, "purpose": "Synthetic acceptance fixture", "progress_method": "manual", "target_basis_points": 10000, "target_date": tc.date, "measure": map[string]any{"unit": "currency", "target_amount_minor": 100000, "currency_code": "USD", "basis": "after_costs_before_tax", "recurrence": tc.recurrence, "period": tc.period}})
		created := request(http.MethodPost, "/api/v1/goals", string(body))
		if created.Code != http.StatusCreated {
			t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
		}
		var result struct {
			Goal goalResponse `json:"goal"`
		}
		if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Goal.Measure == nil || result.Goal.Measure.TargetAmountMinor != 100000 || result.Goal.Measure.CurrencyCode != "USD" || result.Goal.Measure.Basis != "after_costs_before_tax" || result.Goal.Measure.Recurrence != tc.recurrence || result.Goal.Measure.Period != tc.period || result.Goal.TargetDate != tc.date || result.Goal.ProgressState != "unknown" || result.Goal.ProgressBasisPoints != nil {
			t.Fatalf("create response=%s", created.Body.String())
		}
	}
	generic := request(http.MethodPost, "/api/v1/goals", `{"title":"Generic goal","progress_method":"manual","target_basis_points":10000}`)
	if generic.Code != http.StatusCreated {
		t.Fatalf("generic goal status=%d body=%s", generic.Code, generic.Body.String())
	}
	invalid := request(http.MethodPost, "/api/v1/goals", `{"title":"Contradictory","target_basis_points":10000,"target_date":"2026-12-31","measure":{"unit":"currency","target_amount_minor":100000,"currency_code":"USD","basis":"after_costs_before_tax","recurrence":"one_time","period":"month"}}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	read := request(http.MethodGet, "/api/v1/goals", "")
	if read.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", read.Code, read.Body.String())
	}
	var listed struct {
		Goals []goalResponse `json:"goals"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Goals) != 3 {
		t.Fatalf("list response=%s", read.Body.String())
	}
	measuredCount, genericCount := 0, 0
	for _, goal := range listed.Goals {
		if goal.Measure == nil {
			genericCount++
			if goal.ProgressState != "" || goal.ProgressBasisPoints == nil || *goal.ProgressBasisPoints != 0 {
				t.Fatalf("generic readback=%#v", goal)
			}
			continue
		}
		measuredCount++
		if goal.ProgressState != "unknown" || goal.ProgressBasisPoints != nil {
			t.Fatalf("measured readback=%#v", goal)
		}
	}
	if measuredCount != 2 || genericCount != 1 {
		t.Fatalf("measured=%d generic=%d", measuredCount, genericCount)
	}
}
