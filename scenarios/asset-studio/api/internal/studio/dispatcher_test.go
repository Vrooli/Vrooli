package studio

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageToolsDispatcherUsesSubmitAndWaitWithoutCopyingBytes(t *testing.T) { // [REQ:ASSET-P0-006] [REQ:ASSET-P0-007] [REQ:ASSET-P0-008] [REQ:ASSET-P0-009]
	var submitted, waited bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/ai/text_to_image":
			submitted = true
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			if got := r.FormValue("params"); !strings.Contains(got, "launch image") {
				t.Fatalf("params = %s", got)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"jobId":"job-1","modelId":"local-image","tier":"local-gpu"}`)
		case "/vrooli.image_tools.v1.jobs.JobsService/WaitJob":
			waited = true
			_, _ = io.WriteString(w, `{"job":{"state":"JOB_STATE_SUCCEEDED","resultRef":"outputs/job-1.png"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	got, err := (&ImageToolsDispatcher{BaseURL: server.URL, Client: server.Client()}).Dispatch(context.Background(), RenderDispatchRequest{RenderID: "render-1", Prompt: "launch image", CandidateCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !submitted || !waited || len(got.Outputs) != 1 {
		t.Fatalf("dispatch result = %#v, submitted=%t waited=%t", got, submitted, waited)
	}
	if got.Outputs[0].Reference != "image-tools://outputs/job-1.png" || got.Outputs[0].MediaType != "image/png" {
		t.Fatalf("output = %#v", got.Outputs[0])
	}
}

func TestImageToolsDispatcherRejectsCloudReceiptWithoutActualCost(t *testing.T) { // [REQ:ASSET-P0-008]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"jobId":"job-1","modelId":"cloud-image","tier":"byok-cloud"}`)
	}))
	defer server.Close()
	_, err := (&ImageToolsDispatcher{BaseURL: server.URL, Client: server.Client()}).Dispatch(context.Background(), RenderDispatchRequest{Prompt: "x", CandidateCount: 1})
	if err == nil || !strings.Contains(err.Error(), "actual cost") {
		t.Fatalf("expected truthful cloud-cost error, got %v", err)
	}
}

func TestImageToolsDispatcherRefinementUsesProducerOwnedInputReference(t *testing.T) { // [REQ:ASSET-P1-013]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/ai/edit_instruct":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			if got := r.FormValue("input_ref"); got != "outputs/parent.png" {
				t.Fatalf("input_ref=%q", got)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"jobId":"edit-1","modelId":"local-edit","tier":"local-gpu"}`)
		case "/vrooli.image_tools.v1.jobs.JobsService/WaitJob":
			_, _ = io.WriteString(w, `{"job":{"state":"JOB_STATE_SUCCEEDED","resultRef":"outputs/refined.png"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	got, err := (&ImageToolsDispatcher{BaseURL: server.URL, Client: server.Client()}).Dispatch(context.Background(), RenderDispatchRequest{RenderID: "refine-1", Prompt: "brighten background", CandidateCount: 1, Producer: ProducerRefine, ParentReference: "image-tools://outputs/parent.png"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outputs[0].Reference != "image-tools://outputs/refined.png" {
		t.Fatalf("result=%#v", got)
	}
}

func TestImageToolsDispatcherPassesIdentityAdapterConditioning(t *testing.T) { // [REQ:ASSET-P1-007]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/ai/text_to_image":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			if got := r.FormValue("params"); !strings.Contains(got, `"adapterId":"identity-lora"`) {
				t.Fatalf("params=%s", got)
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"jobId":"conditioned-1","modelId":"local","tier":"local-gpu"}`)
		case "/vrooli.image_tools.v1.jobs.JobsService/WaitJob":
			_, _ = io.WriteString(w, `{"job":{"state":"JOB_STATE_SUCCEEDED","resultRef":"outputs/conditioned.png"}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	_, err := (&ImageToolsDispatcher{BaseURL: server.URL, Client: server.Client()}).Dispatch(context.Background(), RenderDispatchRequest{Prompt: "conditioned identity", CandidateCount: 1, ConditioningReferences: []ConditioningReference{{Kind: "adapter", ID: "identity-lora", Version: "1"}}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGatewayVideoDispatcherUsesOneDurableWait(t *testing.T) { // [REQ:ASSET-P1-002]
	var submitted, waited bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/vrooli.ai_gateway.v1.routing.RoutingService/SubmitMedia":
			submitted = true
			_, _ = io.WriteString(w, `{"execution":{"executionId":"video-1","status":"MEDIA_EXECUTION_STATUS_QUEUED"}}`)
		case "/vrooli.ai_gateway.v1.routing.RoutingService/WaitMediaExecution":
			waited = true
			_, _ = io.WriteString(w, `{"execution":{"executionId":"video-1","status":"MEDIA_EXECUTION_STATUS_SUCCEEDED","actualCostUsd":0.24,"resolvedModel":"resource-video","seed":"99","outputs":[{"reference":"media://video-1.mp4","mediaType":"video/mp4"}]}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	got, err := (&GatewayVideoDispatcher{BaseURL: server.URL, Client: server.Client()}).Dispatch(context.Background(), RenderDispatchRequest{RenderID: "render-video", Prompt: "product orbit", CandidateCount: 1, Producer: ProducerVideo, FrameCount: 24})
	if err != nil {
		t.Fatal(err)
	}
	if !submitted || !waited || got.Backend != "ai-gateway" || got.ActualCost != 0.24 || got.Outputs[0].MediaType != "video/mp4" {
		t.Fatalf("video result=%#v submitted=%t waited=%t", got, submitted, waited)
	}
}

func TestBrowserCaptureDispatcherStoresPortableBASReference(t *testing.T) { // [REQ:ASSET-P1-003]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/browser_automation_studio.v1.capture.CaptureService/Capture" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"executionId":"capture-1","artifacts":[{"type":"CAPTURE_TYPE_SCREENSHOT","reference":"bas-capture://capture-1/screenshot","metadata":{"width":"1440","height":"900"}}]}`)
	}))
	defer server.Close()
	got, err := (&BrowserCaptureDispatcher{BaseURL: server.URL, Client: server.Client()}).Dispatch(context.Background(), RenderDispatchRequest{RenderID: "capture-render", Producer: ProducerCapture, CaptureURL: "scenario=asset-studio,path=/"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outputs[0].Reference != "bas-capture://capture-1/screenshot" || got.Outputs[0].Width != 1440 || got.Outputs[0].Height != 900 {
		t.Fatalf("capture result=%#v", got)
	}
}

func TestImageToolsAdvisoryAnalyzerUsesOwnedReference(t *testing.T) { // [REQ:ASSET-P1-010]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/analysis/quality_assessment" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("input_ref"); got != "outputs/asset.png" {
			t.Fatalf("input_ref=%q", got)
		}
		_, _ = io.WriteString(w, `{"quality":{"overallScore":0.8,"notes":["low contrast"]}}`)
	}))
	defer server.Close()
	got, err := (&ImageToolsAdvisoryAnalyzer{BaseURL: server.URL, Client: server.Client()}).Analyze(context.Background(), "image-tools://outputs/asset.png")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "image-tools/quality_assessment" || got.Score != 0.8 || len(got.Notes) != 1 {
		t.Fatalf("advisory=%#v", got)
	}
}
