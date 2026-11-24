package main

import (
	"fmt"
	"testing"

	downloadhttp "landing-page-business-suite-api/handlers/delivery"
	"landing-page-business-suite-api/internal/delivery"
)

func TestDownloadAuthorizationErrorContract(t *testing.T) {
	plans := newTestPlanService(t, "delivery-contract")
	rest := downloadAuthorizationDependencies(nil, nil, plans, nil)
	connect := downloadConnectAuthorizationDependencies(nil, nil, plans, nil)
	tests := []struct {
		name string
		err  error
		want downloadhttp.ErrorKind
	}{
		{"asset", delivery.ErrAssetNotFound, downloadhttp.ErrorNotFound},
		{"app", delivery.ErrAppNotFound, downloadhttp.ErrorAppNotFound},
		{"subscription", delivery.ErrRequiresActiveSubscription, downloadhttp.ErrorSubscriptionRequired},
		{"identity", delivery.ErrIdentityRequired, downloadhttp.ErrorIdentityRequired},
		{"platform", delivery.ErrPlatformRequired, downloadhttp.ErrorPlatformRequired},
		{"entitlements", delivery.ErrEntitlementsUnavailable, downloadhttp.ErrorEntitlementsUnavailable},
		{"selector", delivery.ErrAssetSelectorUnavailable, downloadhttp.ErrorAssetSelectorUnavailable},
		{"unknown", fmt.Errorf("unclassified failure"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.err
			if tt.name != "unknown" {
				err = fmt.Errorf("wrapped by owner: %w", err)
			}
			restKind := rest.ClassifyError(err)
			connectKind := connect.ClassifyError(err)
			if restKind != tt.want || connectKind != tt.want {
				t.Fatalf("REST classification=%q, Connect classification=%q, want %q", restKind, connectKind, tt.want)
			}
		})
	}
	if err := rest.RecordEvent(t.Context(), "app", "windows", "authorized"); err != nil {
		t.Fatalf("REST optional event recorder: %v", err)
	}
	if err := connect.RecordEvent(t.Context(), "app", "windows", "authorized"); err != nil {
		t.Fatalf("Connect optional event recorder: %v", err)
	}
}
