package audioports

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// RemoteSpeechTextProcessor implements SpeechTextProcessor by calling
// audio-tools' TTSService NormalizeForSpeech / SplitParagraphs RPCs.
//
// These are pure functions; the cross-scenario hop adds latency. We cache
// results in-process to keep the per-character cost bounded for the same
// input within a session. Cache is small + bounded.
type RemoteSpeechTextProcessor struct {
	Client *audiotools.Client

	mu         sync.Mutex
	normCache  map[string]cachedString
	splitCache map[string]cachedSplit
}

type cachedString struct {
	value     string
	expiresAt time.Time
}

type cachedSplit struct {
	value     []string
	expiresAt time.Time
}

var _ SpeechTextProcessor = (*RemoteSpeechTextProcessor)(nil)

const remoteProcessorCacheTTL = 5 * time.Minute

func (r *RemoteSpeechTextProcessor) NormalizeForSpeech(text string) string {
	if r == nil || r.Client == nil {
		return text
	}
	r.mu.Lock()
	if r.normCache == nil {
		r.normCache = make(map[string]cachedString)
	}
	if hit, ok := r.normCache[text]; ok && time.Now().Before(hit.expiresAt) {
		r.mu.Unlock()
		return hit.value
	}
	r.mu.Unlock()

	if err := r.Client.Ensure(); err != nil {
		return text
	}
	resp, err := r.Client.TTS.NormalizeForSpeech(
		context.Background(),
		connect.NewRequest(&ttsv1.NormalizeForSpeechRequest{Text: text}),
	)
	if err != nil || resp == nil || resp.Msg == nil {
		return text
	}
	out := resp.Msg.Text
	r.mu.Lock()
	r.normCache[text] = cachedString{value: out, expiresAt: time.Now().Add(remoteProcessorCacheTTL)}
	r.mu.Unlock()
	return out
}

func (r *RemoteSpeechTextProcessor) SplitIntoParagraphs(text string) []string {
	if r == nil || r.Client == nil {
		return []string{text}
	}
	r.mu.Lock()
	if r.splitCache == nil {
		r.splitCache = make(map[string]cachedSplit)
	}
	if hit, ok := r.splitCache[text]; ok && time.Now().Before(hit.expiresAt) {
		r.mu.Unlock()
		return hit.value
	}
	r.mu.Unlock()

	if err := r.Client.Ensure(); err != nil {
		return []string{text}
	}
	resp, err := r.Client.TTS.SplitParagraphs(
		context.Background(),
		connect.NewRequest(&ttsv1.SplitParagraphsRequest{Text: text}),
	)
	if err != nil || resp == nil || resp.Msg == nil {
		return []string{text}
	}
	out := resp.Msg.Paragraphs
	r.mu.Lock()
	r.splitCache[text] = cachedSplit{value: out, expiresAt: time.Now().Add(remoteProcessorCacheTTL)}
	r.mu.Unlock()
	return out
}
