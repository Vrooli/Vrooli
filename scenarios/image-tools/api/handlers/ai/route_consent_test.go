package ai

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"image-tools/internal/safety"
)

// faceSwapSubmit builds a multipart face_swap submit; parts left empty are omitted.
func faceSwapSubmit(t *testing.T, params string, withImage, withReference bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for part, present := range map[string]bool{"file": withImage, "reference": withReference} {
		if !present {
			continue
		}
		w, err := mw.CreateFormFile(part, part+".png")
		if err != nil {
			t.Fatalf("create %s part: %v", part, err)
		}
		_, _ = w.Write([]byte("synthetic image bytes"))
	}
	if err := mw.WriteField("params", params); err != nil {
		t.Fatalf("write params: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/face_swap", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestFaceSwap_LocalRunNeedsNoConsentAndCarriesTheReference(t *testing.T) {
	r, sub := newGatedServer(t, safety.TierLocal)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, faceSwapSubmit(t, `{}`, true, true))
	if w.Code != http.StatusAccepted {
		t.Fatalf("local face swap status = %d, want 202 without any consent; body=%s", w.Code, w.Body.String())
	}
	if !contains(string(sub.last.Payload), `"reference_key"`) {
		t.Errorf("job payload must carry the reference image key; got %s", sub.last.Payload)
	}
}

func TestFaceSwap_RequiresTheReferenceImage(t *testing.T) {
	r, _ := newGatedServer(t, safety.TierLocal)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, faceSwapSubmit(t, `{}`, true, false))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("face swap without a reference = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), "reference") {
		t.Errorf("the refusal must name the missing reference; got %s", w.Body.String())
	}
}

func TestFaceSwap_PublicTierStillAsksForConsentOnALocalRoute(t *testing.T) {
	r, _ := newGatedServer(t, safety.TierPublic)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, faceSwapSubmit(t, `{}`, true, true))
	if w.Code != http.StatusForbidden || !contains(w.Body.String(), "consent") {
		t.Fatalf("public face swap without consent = %d %s, want a 403 naming consent", w.Code, w.Body.String())
	}
	affirmed := httptest.NewRecorder()
	r.ServeHTTP(affirmed, faceSwapSubmit(t, `{"consent_affirmed":true}`, true, true))
	if affirmed.Code != http.StatusAccepted {
		t.Fatalf("affirmed public face swap = %d, want 202; body=%s", affirmed.Code, affirmed.Body.String())
	}
}
