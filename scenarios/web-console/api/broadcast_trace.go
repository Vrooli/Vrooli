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

// fingerprint returns "<len>|<head>|<tail>" where head/tail are up to
// 40 bytes of the payload, with non-printable bytes escaped. Good enough
// to eyeball-match two frames in log output.
func fingerprint(data []byte) string {
	const edge = 40
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
	if len(data) <= edge*2 {
		return fmt.Sprintf("%d|%s", len(data), esc(data))
	}
	return fmt.Sprintf("%d|%s|…|%s", len(data), esc(data[:edge]), esc(data[len(data)-edge:]))
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
