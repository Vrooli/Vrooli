package main

// history_store.go: Bounded output-history ring, monotonic byte counter,
// boundary snapping for trimmed ANSI sequences, and chunk constants for
// history replay.
//
// This file owns:
//   - sgrReset (SGR reset sequence prepended on full-history replay)
//   - historyChunkSize (maximum bytes per replay chunk)
//   - (*Session).appendHistory, .historyStart
//   - snapToCleanBoundary + looksLikeMidSequence (trim-boundary repair)
//
// The file intentionally names the history/resume concern front-and-
// center. Any change to byte-accounting, ring trimming, or resume
// offset arithmetic lands here, not in broadcast.go or session.go.
//
// See docs/plans/terminal-session-refactor-implementation-plan.md §7.1
// and §8.3 for the snapshotFrom contract.

import (
	"bytes"
	"log"
)

// sgrReset is an ANSI SGR reset sequence that clears all text attributes
// (color, bold, underline, etc.). Prepended to replayed history so that
// any dangling color state from a trimmed buffer doesn't bleed into the
// reconnecting client's terminal.
var sgrReset = []byte("\x1b[0m")

// historyChunkSize is the maximum bytes sent per channel message when
// replaying output history to a reconnecting client. Smaller chunks
// prevent browser UI freezes during large history replays.
// DOC: docs/concepts/ARCHITECTURE.md#history-replay-limitations
const historyChunkSize = 64 * 1024 // 64 KB

// historyStart returns the byte offset of the first byte in the current
// outputHistory buffer. Bytes before this offset have been trimmed.
// Must be called with s.mu held.
func (s *Session) historyStart() int64 {
	return s.totalOutputBytes - int64(len(s.outputHistory))
}

// appendHistory grows (and when necessary trims) the bounded output-history
// ring. The ring is sized by offlineBufferMax; trims are snapped to clean
// ANSI/newline boundaries so resumed replays never start mid-escape.
// Must be called with s.mu held.
func (s *Session) appendHistory(data []byte) {
	if len(data) == 0 {
		return
	}
	s.totalOutputBytes += int64(len(data))
	if s.offlineBufferMax <= 0 {
		return
	}

	if len(data) >= s.offlineBufferMax {
		trimmed := data[len(data)-s.offlineBufferMax:]
		s.outputHistory = append([]byte(nil), snapToCleanBoundary(trimmed)...)
		if !s.historyTrimmed {
			s.historyTrimmed = true
			log.Printf("session %s: output history trimmed to %d bytes", s.ID, s.offlineBufferMax)
		}
		return
	}

	combinedLen := len(s.outputHistory) + len(data)
	if combinedLen <= s.offlineBufferMax {
		s.outputHistory = append(s.outputHistory, data...)
		return
	}

	trim := combinedLen - s.offlineBufferMax
	remainder := append(append([]byte(nil), s.outputHistory[trim:]...), data...)
	s.outputHistory = snapToCleanBoundary(remainder)
	if !s.historyTrimmed {
		s.historyTrimmed = true
		log.Printf("session %s: output history trimmed to %d bytes", s.ID, s.offlineBufferMax)
	}
}

// snapToCleanBoundary advances past any partial ANSI escape sequence at the
// start of buf and, when possible, snaps forward to the first newline so
// replayed history starts on a line boundary. This prevents reconnecting
// clients from seeing garbage bytes from a mid-sequence trim.
func snapToCleanBoundary(buf []byte) []byte {
	if len(buf) == 0 {
		return buf
	}

	// If the first byte is ESC, the sequence is intact (starts fresh).
	// If it's NOT ESC but looks like mid-CSI-sequence parameter/intermediate/
	// final bytes, we're inside a truncated sequence.
	start := 0
	if buf[0] != 0x1b && looksLikeMidSequence(buf) {
		// Skip the CSI introducer '[' if present (it follows the truncated ESC).
		if buf[0] == '[' {
			start = 1
		}
		// Scan past parameter bytes (0x30-0x3F) and intermediate bytes (0x20-0x2F)
		// until we hit the final byte (0x40-0x7E) that terminates the sequence.
		for start < len(buf) {
			b := buf[start]
			start++
			if b >= 0x40 && b <= 0x7E {
				break
			}
		}
	}

	// Try to advance to the first newline for a clean line boundary,
	// but only if the newline is within the first 256 bytes to avoid
	// discarding too much history.
	const maxNewlineScan = 256
	scanLimit := start + maxNewlineScan
	if scanLimit > len(buf) {
		scanLimit = len(buf)
	}
	if nlIdx := bytes.IndexByte(buf[start:scanLimit], '\n'); nlIdx >= 0 {
		start += nlIdx + 1
	}

	if start >= len(buf) {
		return nil
	}
	return buf[start:]
}

// looksLikeMidSequence heuristically detects whether buf starts inside a
// truncated ANSI CSI escape sequence. It requires evidence of a real CSI
// sequence (a final byte 0x40-0x7E within a short window) to avoid false
// positives on normal text starting with digits, spaces, or punctuation.
func looksLikeMidSequence(buf []byte) bool {
	if len(buf) == 0 {
		return false
	}
	// '[' following a trimmed ESC is the clear CSI indicator.
	if buf[0] == '[' {
		return true
	}
	// Parameter bytes (0x30-0x3F: digits, semicolons) are only mid-sequence
	// if followed by a CSI final byte (0x40-0x7E) within a short window.
	// This prevents false positives on lines starting with numbers or punctuation.
	if buf[0] >= 0x30 && buf[0] <= 0x3F {
		limit := 8
		if limit > len(buf) {
			limit = len(buf)
		}
		for i := 0; i < limit; i++ {
			b := buf[i]
			if b >= 0x40 && b <= 0x7E {
				return true // Found CSI final byte — this is mid-sequence
			}
			if b < 0x20 || b > 0x3F {
				return false // Non-parameter byte before final — not mid-sequence
			}
		}
		return false // No final byte found in window — not mid-sequence
	}
	// Space (0x20) and intermediate bytes (0x20-0x2F) alone are NOT
	// treated as mid-sequence — they are too common as regular text.
	return false
}
