package metrics

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/vrooli/api-core/schedule"
)

func revenueQueryMatcher(expected, actual string) error {
	if expected == "revenue-subscriptions" {
		for _, fragment := range []string{"sub.status = 'active'", "sub.status = 'trialing'", "billing_interval = 'year'", "intro_enabled", "discount_percent"} {
			if !strings.Contains(actual, fragment) {
				return fmt.Errorf("subscription rollup query missing %q", fragment)
			}
		}
		if strings.Contains(actual, "past_due") {
			return fmt.Errorf("subscription rollup query must exclude past_due")
		}
		return nil
	}
	return sqlmock.QueryMatcherRegexp.Match(expected, actual)
}

type fixedMetricsClock struct{ now time.Time }

func (c fixedMetricsClock) Now() time.Time                        { return c.now }
func (fixedMetricsClock) NewTimer(time.Duration) schedule.Timer   { return nil }
func (fixedMetricsClock) NewTicker(time.Duration) schedule.Ticker { return nil }
func (fixedMetricsClock) Sleep(time.Duration)                     {}

func expectRevenueSummaryQueries(mock sqlmock.Sqlmock, mrr, today, window float64, active, trials, currencies, churned, creditBalance, creditBurned, usage int64, currency string) {
	mock.ExpectQuery("revenue-subscriptions").WillReturnRows(
		sqlmock.NewRows([]string{"mrr", "active", "trials", "currency", "currencies"}).AddRow(mrr, active, trials, currency, currencies),
	)
	mock.ExpectQuery("(?s)SELECT COALESCE\\(SUM\\(amount_cents\\)").WillReturnRows(
		sqlmock.NewRows([]string{"sum"}).AddRow(today),
	)
	mock.ExpectQuery("(?s)SELECT COALESCE\\(SUM\\(amount_cents\\)").WillReturnRows(
		sqlmock.NewRows([]string{"sum"}).AddRow(window),
	)
	mock.ExpectQuery("(?s)SELECT COUNT\\(\\*\\) FILTER").WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(churned),
	)
	mock.ExpectQuery("(?s)SELECT COALESCE\\(SUM\\(balance_credits").WillReturnRows(
		sqlmock.NewRows([]string{"sum"}).AddRow(creditBalance),
	)
	mock.ExpectQuery("(?s)SELECT COALESCE\\(SUM\\(ABS\\(amount_credits\\)").WillReturnRows(
		sqlmock.NewRows([]string{"sum"}).AddRow(creditBurned),
	)
	mock.ExpectQuery("(?s)SELECT COUNT\\(\\*\\) FROM usage_records").WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(usage),
	)
}

func TestRevenueSummaryAppliesDocumentedRollupSemantics(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(revenueQueryMatcher)))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	expectRevenueSummaryQueries(mock, 2400.50, 1299.0, 9999.0, 2, 1, 2, 1, 500, 200, 8, "usd")
	observedAt := time.Date(2026, 9, 3, 20, 0, 0, 0, time.UTC)
	summary, err := NewServiceWithClock(db, fixedMetricsClock{now: observedAt}).GetRevenueSummary()
	if err != nil {
		t.Fatalf("GetRevenueSummary() error = %v", err)
	}
	if summary.MRRMinor != 2401 || summary.SampleSize != 2 || summary.TrialsWithoutPaymentMethod != 1 {
		t.Fatalf("MRR fields = %+v, want rounded MRR 2401, sample 2, excluded trials 1", summary)
	}
	if summary.Currency != "usd" || summary.CurrencyExcludedCount != 1 {
		t.Fatalf("currency fields = %q/%d, want usd/1", summary.Currency, summary.CurrencyExcludedCount)
	}
	if summary.RevenueTodayMinor != 1299 || summary.RevenueWindowMinor != 9999 || summary.SubscriptionsChurnedWindow != 1 || summary.ChurnRatePercent != 100.0/3.0 {
		t.Fatalf("revenue/churn fields = %+v", summary)
	}
	if summary.CreditBalanceTotal != 500 || summary.CreditBurnedWindow != 200 || summary.UsageRecordsWindow != 8 {
		t.Fatalf("usage fields = %+v", summary)
	}
	if summary.MRRUnit != "minor_currency" || summary.RevenueTodayUnit != "minor_currency" || summary.RevenueWindowUnit != "minor_currency" || summary.CreditUnit != "credits" {
		t.Fatalf("units = %+v", summary)
	}
	if summary.ObservedAt == nil || !summary.ObservedAt.Equal(observedAt) {
		t.Fatalf("observed_at = %v, want %v", summary.ObservedAt, observedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestRevenueSummaryReturnsObservedZeroesForEmptyTenant(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(revenueQueryMatcher)))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	expectRevenueSummaryQueries(mock, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "usd")
	summary, err := NewServiceWithClock(db, fixedMetricsClock{now: time.Date(2026, 9, 3, 20, 0, 0, 0, time.UTC)}).GetRevenueSummary()
	if err != nil {
		t.Fatalf("GetRevenueSummary() error = %v", err)
	}
	if summary.MRRMinor != 0 || summary.SampleSize != 0 || summary.ChurnRatePercent != 0 || summary.ObservedAt == nil {
		t.Fatalf("empty summary = %+v, want observed zero-valued summary", summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}
