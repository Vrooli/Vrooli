package stt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/byok/envelope"
	sttpipeline "audio-tools/internal/stt/pipeline"
)

// transcribeTimeout bounds a single buffered transcription. It matches the
// documented Whisper /asr API timeout (resources/whisper/docs/API.md, 300s)
// so a hung sidecar fails the request instead of pinning the connection open
// indefinitely. Shared by every buffered entrypoint (multipart + Connect
// unary Transcribe).
const transcribeTimeout = 300 * time.Second

// audioExceedsLimit reports whether n audio bytes exceeds the documented
// MaxAudioSize ceiling. Enforced on every buffered transcription entrypoint
// (multipart upload + Connect unary Transcribe) so oversize payloads are
// rejected before they reach the Whisper sidecar.
func audioExceedsLimit(n int) bool { return n > sttpipeline.MaxAudioSize }

// MultipartTranscribeHandler exposes a thin multipart endpoint that
// shares the Connect Transcribe codepath. Required because the UI's
// AudioWorklet recordings post `audio/webm` chunks that don't encode
// well as inline proto-JSON bytes.
func MultipartTranscribeHandler(d Deps) http.Handler {
	h := &connectHandler{deps: d}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.deps.Chain == nil {
			http.Error(w, "stt chain not configured", http.StatusServiceUnavailable)
			return
		}
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f, _, err := r.FormFile("audio")
		if err != nil {
			http.Error(w, "audio file required", http.StatusBadRequest)
			return
		}
		defer f.Close()
		audio, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if audioExceedsLimit(len(audio)) {
			http.Error(w, fmt.Sprintf("audio exceeds maximum size of %d bytes", sttpipeline.MaxAudioSize), http.StatusRequestEntityTooLarge)
			return
		}

		env := envelope.FromHTTP(r.Header)
		env.Provider, env.Key = h.applyDefaultCredential(r.Context(), "stt", env.Provider, env.Key)
		format := r.FormValue("format")
		if format == "" {
			format = "wav"
		}
		cfg := h.resolveStreamPipelineConfig(r.Context())
		req := sttchain.Request{
			Audio:         audio,
			Format:        format,
			Language:      r.FormValue("language"),
			InitialPrompt: r.FormValue("initial_prompt"),
			VADFilter:     cfg.VADFilterEnabled,
			BYOKProvider:  env.Provider,
			BYOKKey:       env.Key,
			LPBSToken:     env.LPBSToken,
			UserIdentity:  env.UserIdentity,
		}
		// Derive from the request context so a client disconnect cancels the
		// in-flight transcription, and cap it at transcribeTimeout so a hung
		// Whisper sidecar can't pin this handler open forever (the original
		// context.Background() had neither cancellation nor a deadline).
		ctx, cancel := context.WithTimeout(r.Context(), transcribeTimeout)
		defer cancel()
		res, err := h.deps.Chain.Execute(ctx, req)
		if err != nil {
			httpFromChainErr(w, err)
			return
		}
		out := h.responseFromResult(ctx, res, audio, cfg)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}

func httpFromChainErr(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	switch connect.CodeOf(mapChainError(err)) {
	case connect.CodeInvalidArgument:
		status = http.StatusBadRequest
	case connect.CodeResourceExhausted:
		status = http.StatusPaymentRequired
	case connect.CodeUnavailable:
		status = http.StatusServiceUnavailable
	}
	http.Error(w, fmt.Sprintf("transcribe failed: %v", err), status)
}
