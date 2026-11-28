package domain

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/receiptsigning"
)

func TestCloudDeploymentReceiptSignatureBindsCanonicalPayload(t *testing.T) {
	receipt := CloudDeploymentReceipt{
		SchemaVersion: 1,
		DeploymentID:  "deploy-1",
		ScenarioID:    "scenario-1",
		Outcome:       "deployed",
		Health:        "healthy",
	}
	before, err := receipt.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	signer := receiptsigning.NewDevelopmentSigner()
	if err := receipt.Sign(context.Background(), signer); err != nil {
		t.Fatal(err)
	}
	after, err := receipt.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("canonical payload changed when signature was attached")
	}
	if err := receipt.VerifySignature(context.Background(), signer); err != nil {
		t.Fatalf("fresh receipt signature did not verify: %v", err)
	}

	receipt.DestinationHost = "attacker.example"
	if err := receipt.VerifySignature(context.Background(), signer); err == nil {
		t.Fatal("mutated deployment receipt still verified")
	}
}

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
