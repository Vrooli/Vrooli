package domain

import "testing"

func TestCloudRecoveryReceiptValidatesEffectIdentity(t *testing.T) {
	valid := CloudRecoveryReceipt{
		SchemaVersion:   1,
		DeploymentID:    "dep-1",
		OperationID:     "op-1",
		Action:          "forward_repair",
		Outcome:         "forward_repaired",
		Health:          "healthy",
		BundleSHA256:    "repair",
		ExternalReceipt: "scenario-to-cloud:recovery:op-1",
		ObservedAt:      "2026-09-08T21:00:00Z",
	}
	if !valid.Valid() {
		t.Fatal("valid recovery effect was rejected")
	}
	for name, mutate := range map[string]func(*CloudRecoveryReceipt){
		"preview":         func(r *CloudRecoveryReceipt) { r.DryRun = true },
		"failed outcome":  func(r *CloudRecoveryReceipt) { r.Outcome = "failed" },
		"unhealthy":       func(r *CloudRecoveryReceipt) { r.Health = "unhealthy" },
		"missing receipt": func(r *CloudRecoveryReceipt) { r.ExternalReceipt = "" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if candidate.Valid() {
				t.Fatal("invalid recovery effect was accepted")
			}
		})
	}
}
