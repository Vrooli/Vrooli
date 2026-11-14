package ai

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai/ai_v1connect"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
	"image-tools/internal/safety"
)

func TestStoredAdmissionSharesRESTIdentityAndRecoversWithoutModel(t *testing.T) {
	router, jobs, store := newTestServerWithStore(t, true)
	for key, data := range map[string]string{"input/a": "same bytes", "input/b": "same bytes", "input/changed": "changed bytes"} {
		if err := store.Put(context.Background(), key, bytes.NewBufferString(data), "image/png"); err != nil {
			t.Fatal(err)
		}
	}
	id := uuid.NewString()
	first := newMultipartSubmit(t, "text_to_image", `{"prompt":"a landscape","fallbackPolicy":"local_only"}`)
	first.Header.Set("Idempotency-Key", id)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, first)
	if w.Code != 202 {
		t.Fatalf("REST admission: %d %s", w.Code, w.Body)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	client := aiconnect.NewAIServiceClient(server.Client(), server.URL)
	request := &aiv1.SubmitStoredAIRequest{Operation: "text_to_image", OperationId: id, Params: &aiv1.AIParams{Prompt: "a landscape", FallbackPolicy: "local_only"}}
	response, err := client.SubmitStoredAI(context.Background(), connect.NewRequest(request))
	if err != nil || response.Msg.JobId != "job-1" || jobs.calls != 1 {
		t.Fatalf("cross-transport retry: %v %+v calls=%d", err, response, jobs.calls)
	}
	// A source-bearing admission uses content identity, not a storage key.
	request = &aiv1.SubmitStoredAIRequest{Operation: "segment", InputRef: "input/a", OperationId: uuid.NewString(), Params: &aiv1.AIParams{ModelOverride: "mobilesam", Segmentation: &aiv1.SegmentationPrompt{Points: []*selectionv1.Point{{X: .5, Y: .4}}}}}
	if _, err = client.SubmitStoredAI(context.Background(), connect.NewRequest(request)); err != nil {
		t.Fatal(err)
	}
	request.InputRef = "input/b"
	if _, err = client.SubmitStoredAI(context.Background(), connect.NewRequest(request)); err != nil || jobs.calls != 2 {
		t.Fatalf("content retry: %v calls=%d", err, jobs.calls)
	}
	request.InputRef = "input/changed"
	if _, err = client.SubmitStoredAI(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed source accepted: %v", err)
	}
	offline, retained, offlineStore := newTestServerWithStore(t, false)
	retained.last = jobs.last
	if err := offlineStore.Put(context.Background(), "input/a", bytes.NewBufferString("same bytes"), "image/png"); err != nil {
		t.Fatal(err)
	}
	offlineServer := httptest.NewServer(offline)
	defer offlineServer.Close()
	request.InputRef = "input/a"
	result, err := aiconnect.NewAIServiceClient(offlineServer.Client(), offlineServer.URL).SubmitStoredAI(context.Background(), connect.NewRequest(request))
	if err != nil || result.Msg.ModelId != "mobilesam" || retained.calls != 0 {
		t.Fatalf("recovery selected model again: %v %+v calls=%d", err, result, retained.calls)
	}
}

func TestStoredAdmissionCannotBypassValidationOrDeploymentGate(t *testing.T) {
	router, jobs, _ := newTestServerWithStore(t, true)
	server := httptest.NewServer(router)
	defer server.Close()
	client := aiconnect.NewAIServiceClient(server.Client(), server.URL)
	for _, tc := range []struct {
		request *aiv1.SubmitStoredAIRequest
		code    connect.Code
	}{
		{&aiv1.SubmitStoredAIRequest{Operation: "text_to_image"}, connect.CodeInvalidArgument},
		{&aiv1.SubmitStoredAIRequest{Operation: "text_to_image", OperationId: "not-a-uuid"}, connect.CodeInvalidArgument},
		{&aiv1.SubmitStoredAIRequest{Operation: "segment", OperationId: uuid.NewString()}, connect.CodeInvalidArgument},
		{&aiv1.SubmitStoredAIRequest{Operation: "edit_instruct", OperationId: uuid.NewString(), InputRef: "/etc/passwd"}, connect.CodeNotFound},
	} {
		_, err := client.SubmitStoredAI(context.Background(), connect.NewRequest(tc.request))
		if connect.CodeOf(err) != tc.code {
			t.Errorf("request=%+v got=%v want=%v", tc.request, err, tc.code)
		}
	}
	if jobs.calls != 0 {
		t.Fatal("invalid input queued inference")
	}
	gated, sub := newGatedServer(t, safety.TierPublic)
	public := httptest.NewServer(gated)
	defer public.Close()
	_, err := aiconnect.NewAIServiceClient(public.Client(), public.URL).SubmitStoredAI(context.Background(), connect.NewRequest(&aiv1.SubmitStoredAIRequest{Operation: "inpaint", OperationId: uuid.NewString(), InputRef: "input/unknown", MaskRef: "mask/unknown"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied || sub.calls != 0 {
		t.Fatalf("deployment gate bypassed: %v calls=%d", err, sub.calls)
	}
}
