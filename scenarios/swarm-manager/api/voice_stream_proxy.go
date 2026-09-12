package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"swarm-manager/integrations/audiotools"
)

type voiceStreamProxy struct {
	resolver audiotools.URLResolver
}

func newVoiceStreamProxy(resolver audiotools.URLResolver) *voiceStreamProxy {
	return &voiceStreamProxy{resolver: resolver}
}

var voiceStreamUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (p *voiceStreamProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || p.resolver == nil {
		http.Error(w, "voice stream proxy not configured", http.StatusServiceUnavailable)
		return
	}
	base, err := p.resolver.Resolve()
	if err != nil {
		log.Printf("voice_stream_proxy: resolve audio-tools: %v", err)
		http.Error(w, "audio-tools unavailable", http.StatusServiceUnavailable)
		return
	}
	upstreamURL, err := buildUpstreamWS(base, r.URL.RawQuery)
	if err != nil {
		log.Printf("voice_stream_proxy: build upstream URL: %v", err)
		http.Error(w, "invalid upstream URL", http.StatusInternalServerError)
		return
	}

	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second
	upstream, _, err := dialer.DialContext(r.Context(), upstreamURL, nil)
	if err != nil {
		log.Printf("voice_stream_proxy: dial upstream %s: %v", upstreamURL, err)
		http.Error(w, "failed to reach audio-tools voice stream", http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	client, err := voiceStreamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("voice_stream_proxy: upgrade client: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(2)
	go pipeWS(ctx, cancel, &wg, client, upstream, "client->upstream")
	go pipeWS(ctx, cancel, &wg, upstream, client, "upstream->client")
	wg.Wait()
}

func pipeWS(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, src, dst *websocket.Conn, dir string) {
	defer wg.Done()
	defer cancel()
	for {
		if ctx.Err() != nil {
			return
		}
		mt, data, err := src.ReadMessage()
		if err != nil {
			if !isExpectedWSClose(err) {
				log.Printf("voice_stream_proxy: %s read: %v", dir, err)
			}
			return
		}
		if err := dst.WriteMessage(mt, data); err != nil {
			if err != io.EOF {
				log.Printf("voice_stream_proxy: %s write: %v", dir, err)
			}
			return
		}
	}
}

func isExpectedWSClose(err error) bool {
	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
		websocket.CloseAbnormalClosure,
	)
}

func buildUpstreamWS(base, rawQuery string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/voice/stream"
	u.RawQuery = rawQuery
	return u.String(), nil
}
