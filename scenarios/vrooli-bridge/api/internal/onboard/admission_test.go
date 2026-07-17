package onboard_test

import (
	"context"
	"testing"

	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/mocks"
)

func TestAdmissionFailureStopsBeforeTransferOrPairing(t *testing.T) {
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{AdmissionResult: onboard.AdmissionResult{Category: onboard.AdmissionControlPlaneUnreachable, Detail: "curl-exit-28"}}
	issuer := &mocks.FakeCodeIssuer{Code: "must-not-issue"}
	svc := onboard.NewService(repo, driver, issuer, &mocks.FakeOnlineConfirmer{Online: true}, clock.System{}, onboard.WithDefaultRevision("abc"))
	decision, err := svc.Start(context.Background(), onboard.StartInput{Host: "node", User: "root", ControlPlaneURL: "http://192.168.1.173:18767"})
	if err != nil {
		t.Fatal(err)
	}
	op, timedOut, err := svc.Wait(context.Background(), decision.OpID, 0)
	if err != nil || timedOut {
		t.Fatalf("Wait err=%v timedOut=%v", err, timedOut)
	}
	if op.FailureReason != onboard.FailureControlPlaneUnreachable {
		t.Fatalf("failure=%q, want %q", op.FailureReason, onboard.FailureControlPlaneUnreachable)
	}
	if op.ControlPlaneURL != "http://192.168.1.173:18767" || op.ReachabilityMode != "lan" {
		t.Fatalf("endpoint evidence was not durable: url=%q mode=%q", op.ControlPlaneURL, op.ReachabilityMode)
	}
	if issuer.Calls != 0 {
		t.Fatal("pairing code was issued after failed admission")
	}
	if len(driver.CapturedArgs) != 0 {
		t.Fatal("bootstrap ran after failed admission")
	}
	_, events, err := svc.GetOp(context.Background(), decision.OpID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.StepID == onboard.StepPushScript {
			t.Fatal("script transfer started after failed admission")
		}
	}
}
