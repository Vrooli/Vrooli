//go:build !sherpa_onnx || !cgo

package main

import "testing"

func TestUnavailableBuildFailsClosedAtStartup(t *testing.T) {
	engine, err := newEngineFromEnv()
	if err == nil {
		if engine != nil {
			engine.Close()
		}
		t.Fatal("unqualified build returned a usable TTS engine")
	}
}
