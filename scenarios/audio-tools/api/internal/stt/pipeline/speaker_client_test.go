package pipeline

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSpeakerClientReturnsTypedModelMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"speaker_model_mismatch","error":"profile requires re-enrollment"}`))
	}))
	defer server.Close()

	client := &SpeakerClient{BaseURL: server.URL, Doer: http.DefaultClient}
	_, err := client.Enroll(t.Context(), []byte("pcm"), "profile", "", "", "", "clip.wav")
	var mismatch *SpeakerModelMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("error = %v, want SpeakerModelMismatchError", err)
	}
}
