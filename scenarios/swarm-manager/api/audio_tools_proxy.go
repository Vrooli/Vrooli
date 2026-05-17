package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"swarm-manager/integrations/audiotools"

	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
	summarizeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

var audioToolsConnectPrefixes = []string{
	"/" + sttconnect.STTServiceName + "/",
	"/" + sttconnect.STTAdminServiceName + "/",
	"/" + ttsconnect.TTSServiceName + "/",
	"/" + summarizeconnect.SummarizeServiceName + "/",
}

type audioToolsProxy struct {
	resolver audiotools.URLResolver
}

func newAudioToolsProxy(resolver audiotools.URLResolver) *audioToolsProxy {
	return &audioToolsProxy{resolver: resolver}
}

func (p *audioToolsProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || p.resolver == nil {
		http.Error(w, "audio-tools proxy not configured", http.StatusServiceUnavailable)
		return
	}
	base, err := p.resolver.Resolve()
	if err != nil {
		log.Printf("audio_tools_proxy: resolve audio-tools: %v", err)
		http.Error(w, "audio-tools unavailable", http.StatusServiceUnavailable)
		return
	}
	target, err := url.Parse(base)
	if err != nil {
		log.Printf("audio_tools_proxy: parse upstream %q: %v", base, err)
		http.Error(w, "invalid audio-tools upstream URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = singleJoiningSlash(target.Path, r.URL.Path)
		req.URL.RawPath = ""
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		log.Printf("audio_tools_proxy: upstream %s: %v", req.URL.Path, proxyErr)
		http.Error(rw, "failed to reach audio-tools", http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}
