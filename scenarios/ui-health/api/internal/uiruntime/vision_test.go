package uiruntime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type fakeVisionRunClient struct {
	request  *inferencev1.RunRequest
	response *inferencev1.RunResponse
	err      error
	calls    int
}

func (f *fakeVisionRunClient) Run(_ context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.calls++
	f.request = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.response), nil
}

func TestVisionRunCircuitSkipsRemainingPagesAfterProviderFailure(t *testing.T) {
	client := &fakeVisionRunClient{err: errors.New("offline")}
	judge := newGatewayVisionJudge(func(context.Context) (visionRunClient, error) { return client, nil })
	ctx := withVisionRunCircuit(context.Background())
	first := judge.Check(ctx, []byte{1}, 100, 100, "http://sample/first", "")
	second := judge.Check(ctx, []byte{1}, 100, 100, "http://sample/second", "")
	if len(first) != 1 || !strings.Contains(first[0].Message, "remaining page requests were skipped") {
		t.Fatalf("first failure should explain run-level skip: %+v", first)
	}
	if len(second) != 0 || client.calls != 1 {
		t.Fatalf("open circuit should skip follow-up requests; findings=%+v calls=%d", second, client.calls)
	}
	// A later invocation has a fresh circuit and can retry after capacity changes.
	judge.Check(withVisionRunCircuit(context.Background()), []byte{1}, 100, 100, "http://sample/next-run", "")
	if client.calls != 2 {
		t.Fatalf("new invocation should retry provider; calls=%d", client.calls)
	}
}

func TestVisionJudgeUsesLocalImageRouteAndReturnsConfidentFinding(t *testing.T) {
	client := &fakeVisionRunClient{response: &inferencev1.RunResponse{
		ValueJson: `{"findings":[{"category":"visual polish","description":"The page is uneven.","confidence":0.71,"severity":"warning"},{"category":"touch target","description":"The main action is visibly too small for comfortable tapping.","confidence":0.91,"severity":"warning"}]}`,
		Validated: true,
	}}
	judge := newGatewayVisionJudge(func(context.Context) (visionRunClient, error) { return client, nil })
	findings := judge.Check(context.Background(), []byte{1, 2, 3}, 390, 844, "http://sample/", "http://bas/screenshots/1.png")
	if len(findings) != 2 || findings[0].Code != "vision_ux_touch_target" || findings[1].Code != "vision_ux_visual_polish" {
		t.Fatalf("unexpected findings: %+v", findings)
	}
	if client.request.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY || client.request.GetRole() != "ux.visual-critique" {
		t.Fatalf("vision must use local-only gateway role; request=%+v", client.request)
	}
	if client.request.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL {
		t.Fatalf("wrong privacy class: %s", client.request.GetPrivacyClass())
	}
	if len(client.request.GetAttachments()) != 1 || len(client.request.GetAttachments()[0].GetInlineBytes()) != 3 {
		t.Fatalf("screenshot bytes were not attached: %+v", client.request.GetAttachments())
	}
	if findings[0].Location == "http://sample/" || findings[0].Severity != "warning" {
		t.Fatalf("finding must carry served screenshot evidence and warning severity: %+v", findings[0])
	}
}

func TestVisionJudgeAbstainsForLowConfidenceAndDegradesWhenUnavailable(t *testing.T) {
	client := &fakeVisionRunClient{response: &inferencev1.RunResponse{
		ValueJson: `{"findings":[{"category":"spacing","description":"Maybe something is off.","confidence":0.4,"severity":"error"}]}`,
		Validated: true,
	}}
	judge := newGatewayVisionJudge(func(context.Context) (visionRunClient, error) { return client, nil })
	if findings := judge.Check(context.Background(), []byte{1}, 100, 100, "http://sample/", ""); len(findings) != 0 {
		t.Fatalf("low confidence should abstain, got %+v", findings)
	}
	judge = newGatewayVisionJudge(func(context.Context) (visionRunClient, error) { return nil, errors.New("offline") })
	findings := judge.Check(context.Background(), []byte{1}, 100, 100, "http://sample/", "")
	if len(findings) != 1 || findings[0].Code != "vision_ux_judge_skipped" {
		t.Fatalf("unavailable local judge should report a skip, got %+v", findings)
	}
}

func TestNormalizeUXCategoryUsesCorpusTaxonomy(t *testing.T) {
	for input, want := range map[string]string{
		"Crowded navigation": "navigation_crowding",
		"Button too small":   "touch_target",
		"Low contrast":       "color_contrast",
		"Unsafe status bar":  "safe_area_chrome",
		"Odd visual balance": "visual_polish",
	} {
		if got := normalizeUXCategory(input); got != want {
			t.Errorf("normalizeUXCategory(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSummarizeGatewayVisionErrorDoesNotExposeRawProviderDetails(t *testing.T) {
	if got := summarizeGatewayVisionError(`inference role "ux.visual-critique" is not declared`); got != "gateway role is not declared" {
		t.Fatalf("role failure summary = %q", got)
	}
	if got := summarizeGatewayVisionError("ollama upstream provider unavailable at http://localhost:11434/token"); got != "local provider is unavailable" {
		t.Fatalf("provider failure summary = %q", got)
	}
}
