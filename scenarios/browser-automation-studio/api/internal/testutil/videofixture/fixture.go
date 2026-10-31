// Package videofixture retains tiny real encoded media for deterministic export tests.
// Generated from 0.12 seconds of black 16x16 video with one encoder thread.
package videofixture

import _ "embed"

//go:embed tiny.mp4
var mp4 []byte

//go:embed tiny.webm
var webm []byte

func MP4() []byte  { return append([]byte(nil), mp4...) }
func WebM() []byte { return append([]byte(nil), webm...) }
