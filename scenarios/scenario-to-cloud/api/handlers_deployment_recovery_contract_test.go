package main

import (
	"net/http/httptest"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/evidence"
)

func TestRollbackBindingRejectsStaleLastGoodDigest(t *testing.T) {
	publication := &evidence.Publication{PredecessorReleaseDigest: "sha256:last-good-current"}
	if got, err := bindRollbackReleaseDigest("stale-predecessor", publication); err == nil || got != "" {
		t.Fatalf("stale predecessor accepted: digest=%q err=%v", got, err)
	}
	if got, err := bindRollbackReleaseDigest("last-good-current", publication); err != nil || got != "last-good-current" {
		t.Fatalf("exact predecessor refused: digest=%q err=%v", got, err)
	}
}

func TestRecoveryPreviewIdentityBindsTargetReleaseReviewRouteAndKey(t *testing.T) {
	base := recoveryPreviewRef("dep-1", "rollback", "current-bundle", "current-release", "last-good-bundle", "review-1", domain.RecoveryRouteCloudTargetVerb, "dm-recovery-1")
	cases := []struct {
		name string
		args []string
	}{
		{"deployment", []string{"dep-2", "rollback", "current-bundle", "current-release", "last-good-bundle", "review-1", domain.RecoveryRouteCloudTargetVerb, "dm-recovery-1"}},
		{"current bundle", []string{"dep-1", "rollback", "other-bundle", "current-release", "last-good-bundle", "review-1", domain.RecoveryRouteCloudTargetVerb, "dm-recovery-1"}},
		{"current release", []string{"dep-1", "rollback", "current-bundle", "other-current-release", "last-good-bundle", "review-1", domain.RecoveryRouteCloudTargetVerb, "dm-recovery-1"}},
		{"last-good bundle", []string{"dep-1", "rollback", "current-bundle", "current-release", "other-last-good", "review-1", domain.RecoveryRouteCloudTargetVerb, "dm-recovery-1"}},
		{"review", []string{"dep-1", "rollback", "current-bundle", "current-release", "last-good-bundle", "review-2", domain.RecoveryRouteCloudTargetVerb, "dm-recovery-1"}},
		{"route", []string{"dep-1", "rollback", "current-bundle", "current-release", "last-good-bundle", "review-1", domain.RecoveryRouteBundlePipeline, "dm-recovery-1"}},
		{"idempotency key", []string{"dep-1", "rollback", "current-bundle", "current-release", "last-good-bundle", "review-1", domain.RecoveryRouteCloudTargetVerb, "dm-recovery-2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := recoveryPreviewRef(tc.args[0], tc.args[1], tc.args[2], tc.args[3], tc.args[4], tc.args[5], tc.args[6], tc.args[7]); got == base {
				t.Fatalf("changed %s retained the same preview reference", tc.name)
			}
		})
	}
}

func TestRecoveryStartRefusesPreviewFromAnotherIdentity(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	requestBody := cloudRecoveryHTTPRequest{PreviewRef: "preview-from-another-deployment"}
	binding := &domain.RecoveryBinding{PreviewRef: "preview-for-this-exact-effect"}
	if server.cloudRecoveryCanStart(recorder, requestBody, &domain.Deployment{}, binding) {
		t.Fatal("start accepted a preview ref from another recovery identity")
	}
	if recorder.Code != 412 {
		t.Fatalf("invalid preview status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
