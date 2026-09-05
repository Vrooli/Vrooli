package main

import (
	"time"

	"web-console/internal/tailer"
)

// runTranscriptPollLoop is the shared lifecycle engine for file-backed
// transcript adapters. Source-specific scanners and decoders stay in their
// adapters, but timer ownership, stop semantics, and one scan per tick are
// defined once.
func runTranscriptPollLoop(stop <-chan struct{}, interval time.Duration, scan func()) {
	tailer.RunPollLoop(stop, interval, scan)
}
