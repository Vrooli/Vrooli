package androidjourney

import (
	"context"
	"os"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestHTTPBASClientExecutesLiveAndroidFlow(t *testing.T) {
	if os.Getenv("ANDROID_BAS_LIVE") != "1" {
		t.Skip("set ANDROID_BAS_LIVE=1 with BAS_URL, ANDROID_CDP_ENDPOINT, ANDROID_RENDERER_ID, and ANDROID_LEASE_ID for live Android WebView validation")
	}
	client := HTTPBASClient{BaseURL: os.Getenv("BAS_URL"), FlowRoot: os.Getenv("VROOLI_REPO_ROOT")}
	result, err := client.Execute(context.Background(), BASRequest{
		TargetID: os.Getenv("ANDROID_TARGET_ID"), Scenario: "hello-mobile", StepID: "hello-mobile-smoke",
		RunID: "android-live-bas", IsolationLeaseID: os.Getenv("ANDROID_LEASE_ID"), CDPEndpoint: os.Getenv("ANDROID_CDP_ENDPOINT"), RendererID: os.Getenv("ANDROID_RENDERER_ID"),
		Artifact: deliveryramp.Artifact{ImmutableRef: "android-debug:live"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Completed {
		t.Fatal("live BAS flow did not complete")
	}
}
