package deployments

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/receiptsigning"
)

func TestCloudDeploymentReceiptVerifiesProducerSignatureAndRejectsTampering(t *testing.T) {
	signer := receiptsigning.NewDevelopmentSigner()
	receipt := CloudDeploymentReceipt{
		SchemaVersion: 1, DeploymentID: "dep-1", ScenarioID: "demo", ScenarioVersion: "1.0.0",
		TargetKind: "vps", DestinationID: "target-1", DestinationHost: "vps.example",
		DestinationWorkdir: "/srv/demo", DestinationDomain: "demo.example",
		BundleSHA256: strings.Repeat("a", 64), ReleaseDigest: strings.Repeat("b", 64),
		TargetKey: "host:vps.example", Outcome: "deployed", Health: "healthy",
		ExternalReceipt: "scenario-to-cloud:dep-1", ObservedAt: time.Unix(10, 0).UTC(),
		ProducerRef: cloudReceiptProducerRef,
	}
	canonical, err := receipt.CanonicalJSON()
	if err != nil {
		t.Fatalf("CanonicalJSON() error = %v", err)
	}
	envelope, err := signer.Sign(context.Background(), receiptsigning.PurposeCloudEvidenceReceipt, canonical)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	receipt.Signature, err = json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	if err := receipt.VerifySignature(context.Background(), signer); err != nil {
		t.Fatalf("VerifySignature() error = %v", err)
	}

	receipt.BundleSHA256 = strings.Repeat("c", 64)
	if err := receipt.VerifySignature(context.Background(), signer); err == nil {
		t.Fatal("VerifySignature() accepted a tampered receipt")
	}
}

func TestCloudDeploymentReceiptRejectsUnsignedAndWrongPurpose(t *testing.T) {
	signer := receiptsigning.NewDevelopmentSigner()
	receipt := CloudDeploymentReceipt{}
	if err := receipt.VerifySignature(context.Background(), signer); err == nil {
		t.Fatal("VerifySignature() accepted an unsigned receipt")
	}

	canonical, err := receipt.CanonicalJSON()
	if err != nil {
		t.Fatalf("CanonicalJSON() error = %v", err)
	}
	envelope, err := signer.Sign(context.Background(), receiptsigning.PurposeExperimentAuditReceipt, canonical)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	receipt.Signature, err = json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal signature: %v", err)
	}
	if err := receipt.VerifySignature(context.Background(), signer); err == nil {
		t.Fatal("VerifySignature() accepted a signature for another purpose")
	}
}
