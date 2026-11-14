package goals

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRESTGoalCLITransportRoundTripsMeasureMetadata(t *testing.T) {
	measures := []goalMeasureDTO{
		{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: "one_time", Period: "once"},
		{Unit: "currency", TargetAmountMinor: 100000, CurrencyCode: "USD", Basis: "after_costs_before_tax", Recurrence: "recurring", Period: "month"},
	}
	createdCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/goals" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			var body struct {
				Title             string         `json:"title"`
				TargetDate        string         `json:"target_date"`
				Measure           goalMeasureDTO `json:"measure"`
				TargetBasisPoints int64          `json:"target_basis_points"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			createdCount++
			date, title := "2026-11-30", "First $1,000 after costs before tax"
			if createdCount == 2 {
				date, title = "2026-12-31", "Recurring $1,000/month"
			}
			if body.Title != title || body.TargetDate != date || body.Measure != measures[createdCount-1] || body.TargetBasisPoints != 10000 {
				t.Fatalf("create body=%#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"goal": map[string]any{"id": fmt.Sprintf("goal-%d", createdCount), "title": body.Title, "status": "active", "progress_method": "manual", "progress_state": "unknown", "target_basis_points": 10000, "revision": 1, "target_date": body.TargetDate, "measure": body.Measure}})
		case http.MethodGet:
			items := []any{}
			for index, measure := range measures {
				title, date := "First $1,000 after costs before tax", "2026-11-30"
				if index == 1 {
					title, date = "Recurring $1,000/month", "2026-12-31"
				}
				items = append(items, map[string]any{"id": fmt.Sprintf("goal-%d", index+1), "title": title, "status": "active", "progress_method": "manual", "progress_state": "unknown", "target_basis_points": 10000, "revision": 1, "target_date": date, "measure": measure})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"goals": items})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	h := &handlers{httpClient: server.Client(), baseURL: server.URL}
	for index, measure := range measures {
		title, date := "First $1,000 after costs before tax", "2026-11-30"
		if index == 1 {
			title, date = "Recurring $1,000/month", "2026-12-31"
		}
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "title"}, {Name: "purpose"}, {Name: "progress-method"}, {Name: "target-percent"}, {Name: "target-date"}, {Name: "measure-unit"}, {Name: "target-amount-minor"}, {Name: "currency-code"}, {Name: "measure-basis"}, {Name: "recurrence"}, {Name: "period"}}},
			Flags:  map[string]string{"title": title, "purpose": "Synthetic goal", "progress-method": "manual", "target-percent": "100", "target-date": date, "measure-unit": measure.Unit, "target-amount-minor": "100000", "currency-code": measure.CurrencyCode, "measure-basis": measure.Basis, "recurrence": measure.Recurrence, "period": measure.Period},
		})
		created, err := h.createCall(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if created.Measure == nil || *created.Measure != measure || created.ProgressState != "unknown" || created.ProgressBasisPoints != nil || created.TargetDate != date {
			t.Fatalf("created=%#v", created)
		}
	}
	listed, err := h.listCall(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Goals) != 2 || listed.Goals[0].Measure == nil || listed.Goals[1].Measure == nil || listed.Goals[0].ProgressState != "unknown" || listed.Goals[1].ProgressState != "unknown" || listed.Goals[0].ProgressBasisPoints != nil || listed.Goals[1].ProgressBasisPoints != nil {
		t.Fatalf("listed=%#v", listed.Goals)
	}
	if got := h.listReport(nil, listed); len(got.Changes) != 2 || !strings.Contains(got.Changes[0], "progress unknown") {
		t.Fatalf("CLI read output=%#v", got)
	}
}
