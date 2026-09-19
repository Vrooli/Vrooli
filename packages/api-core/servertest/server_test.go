package servertest

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type testHandlerProvider struct{ handler http.Handler }

func (p testHandlerProvider) Handler() http.Handler { return p.handler }

func TestNewLiveServerUsesNarrowHandlerProvider(t *testing.T) {
	live := NewLiveServer(t, testHandlerProvider{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok:"+r.URL.Path)
	})})

	resp, body := live.Do(t, http.MethodGet, "health", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := string(body); got != "ok:/health" {
		t.Fatalf("body = %q, want %q", got, "ok:/health")
	}
}

type recordingT struct {
	fatal bool
}

func (r *recordingT) Helper()           {}
func (r *recordingT) Cleanup(func())    {}
func (r *recordingT) Fatal(args ...any) { r.fatal = true }

func TestNewLiveServerRejectsNilProvider(t *testing.T) {
	recorder := &recordingT{}
	if got := newLiveServer(recorder, nil); got != nil {
		t.Fatalf("nil provider returned %#v", got)
	}
	if !recorder.fatal {
		t.Fatal("nil provider should report a fatal test error")
	}
}

func TestLiveServerDoNormalizesPaths(t *testing.T) {
	live := NewLiveServer(t, testHandlerProvider{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	})})
	_, body := live.Do(t, http.MethodGet, "without-leading-slash", nil)
	if !strings.HasPrefix(string(body), "/") {
		t.Fatalf("normalized path body = %q", body)
	}
}
