package deployments

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"deployment-manager/releases"
)

type verificationResumeCloud struct {
	calls  int
	result *CloudHealthResult
}

func (c *verificationResumeCloud) CheckDeploymentHealth(_ context.Context, req DeploymentHealthRequest) (*CloudHealthResult, error) {
	c.calls++
	if c.result == nil {
		return nil, nil
	}
	result := *c.result
	if result.DeploymentID == "" {
		result.DeploymentID = req.DeploymentID
	}
	return &result, nil
}

func TestResumeCloudVerificationPersistsAndReusesExactReadback(t *testing.T) {
	const operationID, releaseID, bundle = "release-op-resume-verification", "rel-verification", "sha256:" + "ab" + "cd"
	receipt, _ := json.Marshal(CloudDeploymentReceipt{DeploymentID: "dep-exact", OperationID: operationID, BundleSHA256: bundle, TargetKey: "target-key-1", DestinationID: "cm-instance-1", DestinationHost: "203.0.113.8"})
	repo := newFakeReleasesRepo()
	repo.cloudSteps[operationID] = releases.CloudStep{OperationID: operationID, RequestKey: "dm-cloud:" + operationID, RequestDigest: "sha256:immutable-request", State: "downstream_started", DeploymentID: "dep-exact", Receipt: receipt}
	cloud := &verificationResumeCloud{result: &CloudHealthResult{Healthy: true, DeploymentID: "dep-exact", ObservedReleaseDigest: bundle, ObservedAt: time.Now().UTC()}}
	o := &Orchestrator{releasesRepo: repo, cloudClient: cloud}
	request := releases.DeployRequest{OperationID: operationID, ReleaseID: releaseID, CloudManifest: json.RawMessage(`{"scenario":{"id":"fixture"},"environment":"staging"}`), CloudBundleSHA256: bundle}

	step, err := o.ResumeCloudVerification(context.Background(), request)
	if err != nil || step == nil || step.VerificationState != "passed" || step.State != "verification_passed" || step.VerificationKey != releases.CloudVerificationIdempotencyKey(operationID) || !strings.HasPrefix(step.VerificationRequestDigest, "sha256:") || step.VerificationAttemptedAt == nil || cloud.calls != 1 {
		t.Fatalf("first verification: step=%+v calls=%d err=%v", step, cloud.calls, err)
	}
	step, err = o.ResumeCloudVerification(context.Background(), request)
	if err != nil || step == nil || step.VerificationState != "passed" || cloud.calls != 1 {
		t.Fatalf("retry should attach to persisted result: step=%+v calls=%d err=%v", step, cloud.calls, err)
	}
}

func TestResumeCloudVerificationRefusesReceiptDriftBeforeHealthRead(t *testing.T) {
	const operationID = "release-op-verification-drift"
	receipt, _ := json.Marshal(CloudDeploymentReceipt{DeploymentID: "dep-exact", OperationID: operationID, BundleSHA256: "sha256:receipt", TargetKey: "target-key-1"})
	repo := newFakeReleasesRepo()
	repo.cloudSteps[operationID] = releases.CloudStep{OperationID: operationID, RequestKey: "dm-cloud:" + operationID, RequestDigest: "sha256:immutable-request", State: "downstream_started", DeploymentID: "dep-exact", Receipt: receipt}
	cloud := &verificationResumeCloud{result: &CloudHealthResult{Healthy: true}}
	o := &Orchestrator{releasesRepo: repo, cloudClient: cloud}
	_, err := o.ResumeCloudVerification(context.Background(), releases.DeployRequest{OperationID: operationID, ReleaseID: "rel-1", CloudManifest: json.RawMessage(`{"scenario":{"id":"fixture"}}`), CloudBundleSHA256: "sha256:other"})
	if err == nil || cloud.calls != 0 {
		t.Fatalf("receipt drift err=%v calls=%d", err, cloud.calls)
	}
}

func TestResumeCloudVerificationRetriesOnlyAnAmbiguousRead(t *testing.T) {
	const operationID, bundle = "release-op-verification-ambiguous", "sha256:112233"
	receipt, _ := json.Marshal(CloudDeploymentReceipt{DeploymentID: "dep-exact", OperationID: operationID, BundleSHA256: bundle, TargetKey: "target-key-1"})
	repo := newFakeReleasesRepo()
	repo.cloudSteps[operationID] = releases.CloudStep{OperationID: operationID, RequestKey: "dm-cloud:" + operationID, RequestDigest: "sha256:immutable-request", State: "downstream_started", DeploymentID: "dep-exact", Receipt: receipt}
	cloud := &verificationResumeCloud{}
	o := &Orchestrator{releasesRepo: repo, cloudClient: cloud}
	request := releases.DeployRequest{OperationID: operationID, ReleaseID: "rel-ambiguous", CloudManifest: json.RawMessage(`{"scenario":{"id":"fixture"}}`), CloudBundleSHA256: bundle}
	step, err := o.ResumeCloudVerification(context.Background(), request)
	if err != nil || step == nil || step.VerificationState != "ambiguous" || step.State != "verification_ambiguous" || cloud.calls != 1 {
		t.Fatalf("ambiguous result: step=%+v calls=%d err=%v", step, cloud.calls, err)
	}
	cloud.result = &CloudHealthResult{Healthy: true, DeploymentID: "dep-exact", ObservedReleaseDigest: bundle, ObservedAt: time.Now().UTC()}
	step, err = o.ResumeCloudVerification(context.Background(), request)
	if err != nil || step == nil || step.VerificationState != "passed" || step.State != "verification_passed" || cloud.calls != 2 {
		t.Fatalf("ambiguous retry: step=%+v calls=%d err=%v", step, cloud.calls, err)
	}
}
