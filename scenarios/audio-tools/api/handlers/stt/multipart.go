package stt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/byok/envelope"
	"audio-tools/internal/protomap"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// MultipartTranscribeHandler exposes a thin multipart endpoint that
// shares the Connect Transcribe codepath. Required because the UI's
// AudioWorklet recordings post `audio/webm` chunks that don't encode
// well as inline proto-JSON bytes.
func MultipartTranscribeHandler(chain *sttchain.Chain) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if chain == nil {
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

		env := envelope.FromHTTP(r.Header)
		format := r.FormValue("format")
		if format == "" {
			format = "wav"
		}
		req := sttchain.Request{
			Audio:         audio,
			Format:        format,
			Language:      r.FormValue("language"),
			InitialPrompt: r.FormValue("initial_prompt"),
			BYOKProvider:  env.Provider,
			BYOKKey:       env.Key,
			LPBSToken:     env.LPBSToken,
			UserIdentity:  env.UserIdentity,
		}
		res, err := chain.Execute(context.Background(), req)
		if err != nil {
			httpFromChainErr(w, err)
			return
		}
		out := &sttv1.TranscribeResponse{
			Text: res.Text, DetectedLanguage: res.DetectedLanguage,
			DurationSeconds: res.DurationSeconds,
			ProviderTier:    protomap.ProviderTierToProto(string(res.Tier)),
			ProviderId:      res.ProviderID, ModelId: res.ModelID,
			LatencyMs: float64(res.Latency.Milliseconds()),
		}
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
