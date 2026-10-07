package main

import (
	"context"
	"testing"
)

func TestProtectedFiniteOwnerSelectionDefaultAndInvalidRefuseWithoutAcquisition(t *testing.T) {
	for _, phase := range []string{"", "off"} {
		t.Setenv("VROOLI_FINITE_OWNER_PHASE", phase)
		owner, err := loadProtectedFiniteOwnerStartup(context.Background())
		if err != nil || owner != nil {
			t.Fatal("inert selection changed startup")
		}
	}
	t.Setenv("VROOLI_FINITE_OWNER_PHASE", "anonymous")
	if owner, err := loadProtectedFiniteOwnerStartup(context.Background()); err == nil || owner != nil {
		t.Fatal("invalid selection admitted owner")
	}
}

func TestProtectedFiniteOwnerSelectionChecksFlagBeforeAcquisition(t *testing.T) {
	for _, test := range []struct{ phase, enabled string }{{"live", "invalid"}, {"live", "false"}, {"commission", "true"}} {
		t.Setenv("VROOLI_FINITE_OWNER_PHASE", test.phase)
		t.Setenv("VROOLI_FINITE_ENABLED", test.enabled)
		if owner, err := loadProtectedFiniteOwnerStartup(context.Background()); err == nil || owner != nil {
			t.Fatal("incompatible flags admitted owner")
		}
	}
}
