package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	engine, err := newEngineFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer engine.Close()
	streaming, streamingErr := newStreamingEngineFromEnv()
	if streamingErr != nil {
		fmt.Fprintln(os.Stderr, "streaming STT disabled:", streamingErr)
	} else {
		defer streaming.Close()
	}
	speaker, speakerErr := newSpeakerRuntimeFromEnv()
	if speakerErr != nil {
		fmt.Fprintln(os.Stderr, "speaker runtime disabled:", speakerErr)
	} else {
		defer speaker.Close()
	}
	addr := envOr("RESOURCE_LISTEN_ADDR", "127.0.0.1:8881")
	if err := http.ListenAndServe(addr, newHandlerWithEncoderStreamingAndSpeaker(engine, newFFmpegEncoder(), streaming, speaker)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
