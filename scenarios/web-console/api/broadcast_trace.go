package main

// broadcast_trace.go: opt-in broadcast fingerprint logging for diagnosing
// client-visible output duplication without browser devtools.
//
// Enable by setting env WC_BROADCAST_TRACE=1 before starting the API.
// When off, all trace calls are no-ops (single env lookup, cached).
//
// Each trace line is a single log.Printf call tagged "bctrace" so it's
// greppable in `vrooli scenario logs web-console`.

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	broadcastTraceOnce    sync.Once
	broadcastTraceEnabled bool
	broadcastTraceSeq     atomic.Uint64
)

func bctraceOn() bool {
	broadcastTraceOnce.Do(func() {
		broadcastTraceEnabled = os.Getenv("WC_BROADCAST_TRACE") == "1"
		if broadcastTraceEnabled {
			log.Printf("bctrace enabled (WC_BROADCAST_TRACE=1)")
		}
	})
	return broadcastTraceEnabled
}

// fingerprint returns "<len>|<escaped-bytes>" where the escaped bytes
// cover the FULL payload up to maxFingerprintBytes (default 8 KiB so a
// typical TUI redraw is fully captured for diagnosis). Override with
// WC_BROADCAST_TRACE_FULL=1 to log every byte regardless of size.
func fingerprint(data []byte) string {
	const cap = 8 * 1024
	esc := func(b []byte) string {
		var sb strings.Builder
		for _, c := range b {
			if c >= 0x20 && c < 0x7f {
				sb.WriteByte(c)
			} else {
				fmt.Fprintf(&sb, "\\x%02x", c)
			}
		}
		return sb.String()
	}
	full := os.Getenv("WC_BROADCAST_TRACE_FULL") == "1"
	if full || len(data) <= cap {
		return fmt.Sprintf("%d|%s", len(data), esc(data))
	}
	return fmt.Sprintf("%d|%s|…(+%d truncated)…|%s",
		len(data), esc(data[:cap/2]), len(data)-cap, esc(data[len(data)-cap/2:]))
}

// bctrace emits one trace line when enabled. Fields are space-separated
// key=value so a mobile user can read them line-wrapped and still match
// identical fingerprints across lines.
func bctrace(tag, sessionID, extra string, data []byte) {
	if !bctraceOn() {
		return
	}
	seq := broadcastTraceSeq.Add(1)
	if data == nil {
		log.Printf("bctrace seq=%d tag=%s sess=%s %s", seq, tag, sessionID, extra)
		return
	}
	log.Printf("bctrace seq=%d tag=%s sess=%s %s fp=%s", seq, tag, sessionID, extra, fingerprint(data))
}
