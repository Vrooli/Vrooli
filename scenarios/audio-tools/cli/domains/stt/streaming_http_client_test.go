package stt

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func TestStreamingHTTPClientHasNoFixedWallClockTimeout(t *testing.T) {
	if got := newStreamingHTTPClient().Timeout; got != 0 {
		t.Fatalf("streaming client timeout = %s, want no fixed timeout", got)
	}
}

type captureRoundTripper struct {
	url  *url.URL
	host string
}

func (r *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	u := *req.URL
	r.url = &u
	r.host = req.Host
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody, Request: req}, nil
}

func TestStreamingHTTPClientUsesInvocationAPIBase(t *testing.T) {
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:           "audio-tools",
		DefaultAPIBase: "http://127.0.0.1:19630",
		AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var global cliapp.GlobalOptions
	if _, err := cliapp.ParseGlobalFlags([]string{"--api-base", "http://127.0.0.1:15373", "--instance", "lab"}, &global, &core.APIOverride); err != nil {
		t.Fatal(err)
	}
	cliutil.SetInstanceOverride("audio-tools", global.Instance)
	defer cliutil.SetInstanceOverride("audio-tools", "")
	rt := &captureRoundTripper{}
	client := &http.Client{Transport: rt}
	wrapped := streamingHTTPClient{core: core, client: client, baseURL: "http://127.0.0.1:19630"}
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:19630/api/v1/audio-tools.v1.stt.STTService/TranscribeStream", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "127.0.0.1:19630" // a previously selected API must not leak as Host
	if _, err := wrapped.Do(req); err != nil {
		t.Fatal(err)
	}
	if rt.url == nil {
		t.Fatal("transport was not called")
	}
	if got, want := rt.url.Host, "127.0.0.1:15373"; got != want {
		t.Fatalf("request host = %q, want %q", got, want)
	}
	if got, want := rt.url.Path, "/api/v1/audio-tools.v1.stt.STTService/TranscribeStream"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
	if rt.host != "" {
		t.Fatalf("request Host header override = %q, want empty so transport follows selected URL", rt.host)
	}
}
