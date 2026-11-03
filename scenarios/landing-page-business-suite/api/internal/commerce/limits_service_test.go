package commerce

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func d8LimitDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE subscription_tier_limits(id TEXT PRIMARY KEY DEFAULT(lower(hex(randomblob(16)))),tier_id TEXT,limit_type TEXT,limit_key TEXT,limit_value BIGINT,cost_multiplier BIGINT,app_bundle_key TEXT,reset_period TEXT DEFAULT 'monthly',created_at DATETIME DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestLimitLookupRejectsAmbiguousSameScope(t *testing.T) {
	db := d8LimitDB(t)
	_, err := db.Exec(`INSERT INTO subscription_tier_limits(tier_id,limit_type,limit_key,limit_value,cost_multiplier,app_bundle_key) VALUES('pro','cost_based','ai_credits',100,1000000,'app'),('pro','app_specific','ai_credits',90,1000000,'app')`)
	if err != nil {
		t.Fatal(err)
	}
	app := "app"
	if _, err := NewLimitsService(db, "sqlite", nil).GetLimit(context.Background(), "pro", "ai_credits", &app); err == nil || !strings.Contains(err.Error(), "multiple meter") {
		t.Fatalf("two applicable scope definitions must be refused as ambiguous; error=%v", err)
	}
}

func TestLegacyAICapHasExplicitReviewedCostKind(t *testing.T) {
	db := d8LimitDB(t)
	_, err := db.Exec(`INSERT INTO subscription_tier_limits(tier_id,limit_type,limit_key,limit_value,cost_multiplier,app_bundle_key) VALUES('pro','app_specific','ai_credits',100,1000000,'app')`)
	if err != nil {
		t.Fatal(err)
	}
	app := "app"
	limit, err := NewLimitsService(db, "sqlite", nil).GetLimit(context.Background(), "pro", "ai_credits", &app)
	if err != nil {
		t.Fatal(err)
	}
	if limit.LimitType != "cost_based" {
		t.Fatalf("legacy AI cap kind=%s", limit.LimitType)
	}
	var stored string
	if err := db.QueryRow(`SELECT limit_type FROM subscription_tier_limits`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "app_specific" {
		t.Fatal("reading the reviewed mapping must not migrate configuration")
	}
}

func TestCountLimitRejectsDollarUpdate(t *testing.T) {
	db := d8LimitDB(t)
	_, err := db.Exec(`INSERT INTO subscription_tier_limits(tier_id,limit_type,limit_key,limit_value,cost_multiplier) VALUES('pro','count_based','compute_minutes',60,1)`)
	if err != nil {
		t.Fatal(err)
	}
	dollars := 5.0
	if _, err := NewLimitsService(db, "sqlite", nil).UpdateLimit(context.Background(), "pro", "compute_minutes", nil, TierLimitUpdate{DisplayDollars: &dollars}); err == nil || !strings.Contains(err.Error(), "dollar") {
		t.Fatalf("count quota must reject dollar units; error=%v", err)
	}
	var value int64
	if err := db.QueryRow(`SELECT limit_value FROM subscription_tier_limits`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != 60 {
		t.Fatalf("count quota changed to %d", value)
	}
}

func TestCreateCountMeterKindIsExplicit(t *testing.T) {
	db := d8LimitDB(t)
	app := "app"
	limit, err := NewLimitsService(db, "sqlite", nil).CreateLimit(context.Background(), TierLimit{TierID: "pro", LimitType: "count_based", LimitKey: "exports", LimitValue: 3, AppBundleKey: &app})
	if err != nil {
		t.Fatal(err)
	}
	if limit.LimitType != "count_based" || limit.LimitValue != 3 || limit.DisplayDollars != nil {
		t.Fatalf("count definition=%+v", limit)
	}
}

type d8MeterDefinitions []TierLimit

func (d d8MeterDefinitions) GetTierLimits(context.Context, string) ([]TierLimit, error) {
	return d, nil
}
func (d d8MeterDefinitions) GetLimit(context.Context, string, string, *string) (*TierLimit, error) {
	return nil, nil
}

func TestMeterResolutionPreservesBothExactCapsAndRefusesAmbiguity(t *testing.T) {
	app, other, empty := "app", "other", ""
	global := TierLimit{TierID: "pro", LimitType: "cost_based", LimitKey: "ai_credits", LimitValue: 100, CostMultiplier: 1000000}
	cap := global
	cap.AppBundleKey = &app
	cap.LimitValue = 40
	wrongKind := cap
	wrongKind.LimitType = "count_based"
	wrongConversion := cap
	wrongConversion.CostMultiplier = 10
	ambiguous := cap
	ambiguous.LimitType = "app_specific"
	ambiguous.LimitKey = "exports"
	for _, tc := range []struct {
		name, key string
		rows      d8MeterDefinitions
		wantError bool
	}{
		{"global and app", "ai_credits", d8MeterDefinitions{global, cap}, false},
		{"duplicate global", "ai_credits", d8MeterDefinitions{global, global, cap}, true},
		{"duplicate app", "ai_credits", d8MeterDefinitions{global, cap, cap}, true},
		{"contradictory kind", "ai_credits", d8MeterDefinitions{global, wrongKind}, true},
		{"contradictory conversion", "ai_credits", d8MeterDefinitions{global, wrongConversion}, true},
		{"ambiguous legacy quota", "exports", d8MeterDefinitions{ambiguous}, true},
		{"missing", "ai_credits", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := ResolveMeterLimits(context.Background(), tc.rows, "pro", tc.key, app)
			if (err != nil) != tc.wantError {
				t.Fatalf("resolved=%+v error=%v wantError=%v", resolved, err, tc.wantError)
			}
			if !tc.wantError && (resolved.Kind != "cost_based" || resolved.Global == nil || resolved.App == nil || resolved.Global.LimitValue != 100 || resolved.App.LimitValue != 40) {
				t.Fatalf("app cap erased or substituted: %+v", resolved)
			}
		})
	}
	foreign := cap
	foreign.AppBundleKey = &other
	if _, err := ResolveMeterLimits(context.Background(), d8MeterDefinitions{foreign}, "pro", "ai_credits", app); err == nil {
		t.Fatal("a different app's cap cannot supply this app's authority")
	}
	invalid := global
	invalid.AppBundleKey = &empty
	if _, err := ResolveMeterLimits(context.Background(), d8MeterDefinitions{invalid}, "pro", "ai_credits", app); err == nil {
		t.Fatal("empty stored app scope must not be treated as global")
	}
}

func TestMeterResolutionKeepsCountUnitsAndExplicitLegacyAIMapping(t *testing.T) {
	app := "app"
	for _, tc := range []struct {
		key, stored, expected string
		multiplier            int64
	}{
		{"compute_minutes", "count_based", "count_based", 0},
		{"ai_credits", "app_specific", "cost_based", 1000000},
	} {
		resolved, err := ResolveMeterLimits(context.Background(), d8MeterDefinitions{{TierID: "pro", LimitType: tc.stored, LimitKey: tc.key, LimitValue: 60, CostMultiplier: tc.multiplier, AppBundleKey: &app}}, "pro", tc.key, app)
		if err != nil || resolved.Kind != tc.expected || resolved.App == nil || resolved.App.LimitValue != 60 {
			t.Fatalf("key %s resolution=%+v error=%v", tc.key, resolved, err)
		}
	}
}
