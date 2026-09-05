package config

import (
	"testing"
	"time"
)

func TestLoadUsesDeclaredDefaults(t *testing.T) {
	t.Setenv("COMPUTE_MANAGER_RECONCILE_INTERVAL", "")
	t.Setenv("COMPUTE_MANAGER_EXPIRY_INTERVAL", "")
	t.Setenv("COMPUTE_MANAGER_MINIMUM_BILLABLE_UNIT", "")
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.ReconcileInterval != 15*time.Minute || got.ExpiryInterval != time.Minute || got.MinimumBillableUnit != time.Hour {
		t.Fatalf("defaults = %+v", got)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("COMPUTE_MANAGER_EXPIRY_INTERVAL", "later")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid duration error")
	}
}
