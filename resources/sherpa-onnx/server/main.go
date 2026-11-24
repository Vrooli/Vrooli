package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	var engine TTS
	if envBool("SHERPA_ONNX_TTS_ENABLED", true) {
		loaded, err := newEngineFromEnv()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		engine = loaded
		defer engine.Close()
	} else {
		fmt.Fprintln(os.Stderr, "TTS disabled by SHERPA_ONNX_TTS_ENABLED=false")
	}
	streaming, streamingErr := newStreamingEngineFromEnv()
	if streamingErr != nil {
		fmt.Fprintln(os.Stderr, "streaming STT disabled:", streamingErr)
	} else {
		streaming = limitStreaming(streaming, streamingSessionLimit())
		defer streaming.Close()
	}
	var speaker SpeakerRuntime
	if envBool("SHERPA_ONNX_SPEAKER_ENABLED", true) {
		loaded, err := newSpeakerRuntimeFromEnv()
		if err != nil {
			fmt.Fprintln(os.Stderr, "speaker runtime disabled:", err)
		} else {
			speaker = loaded
			defer speaker.Close()
		}
	} else {
		fmt.Fprintln(os.Stderr, "speaker runtime disabled by SHERPA_ONNX_SPEAKER_ENABLED=false")
	}
	addr := envOr("RESOURCE_LISTEN_ADDR", "127.0.0.1:8881")
	if err := http.ListenAndServe(addr, newHandlerWithEncoderStreamingAndSpeaker(engine, newFFmpegEncoder(), streaming, speaker)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func streamingSessionLimit() int {
	const fallback = 4
	value := strings.TrimSpace(os.Getenv("SHERPA_ONNX_STREAMING_MAX_SESSIONS"))
	if value == "" {
		return fallback
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > 64 {
		fmt.Fprintf(os.Stderr, "invalid SHERPA_ONNX_STREAMING_MAX_SESSIONS %q; using %d\n", value, fallback)
		return fallback
	}
	return limit
}
