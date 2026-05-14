package main

import (
	"log"
	"time"

	"web-console/internal/backend"
	"web-console/internal/pty"
)

// tmux re-attach retry parameters. When the attach process dies but the tmux
// session itself survives, we retry with exponential backoff before declaring
// the session dead. A single transient failure (brief tmux unresponsiveness,
// resource pressure) should not permanently destroy a session.
const (
	tmuxReattachMaxRetries = 3
	tmuxReattachBaseDelay  = 500 * time.Millisecond
)

// readLoop continuously reads PTY output and broadcasts to subscribers.
// Each read is split at UTF-8 codepoint boundaries so that partial multi-byte
// sequences are buffered and prepended to the next read, preventing JSON
// encoding from replacing incomplete sequences with U+FFFD.
//
// On PTY read error (including normal process exit), it:
//  1. Flushes any buffered UTF-8 remainder
//  2. Marks the session as exited
//  3. Closes all client channels (triggering "exit" messages in WS handlers)
//  4. Signals exitCh so the SessionManager can clean up
//
// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
func (s *Session) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("session %s: readLoop panic (recovered): %v", s.ID, r)
			// Ensure exit signaling even after a panic so waiters don't hang.
			s.mu.Lock()
			if !s.processExited {
				s.processExited = true
				s.processExitCode = -1
				for ch := range s.clients {
					close(ch)
					delete(s.clients, ch)
				}
			}
			s.mu.Unlock()
			select {
			case <-s.exitCh:
			default:
				close(s.exitCh)
			}
		}
	}()
	buf := make([]byte, s.ptyReadBuffer)
	for {
		s.mu.Lock()
		p := s.pty
		s.mu.Unlock()
		n, err := p.Read(buf)
		if n > 0 {
			data := buf[:n]
			// Answer terminal capability queries server-side (DA1/DA3/DECRQM)
			// so TUI programs like Claude Code don't stall waiting for a
			// response xterm.js will not send. Write the reply straight to
			// the PTY master — it arrives at the foreground process as
			// stdin, just as a real terminal emulator's reply would. Only
			// the standard backend needs this: tmux answers queries for its
			// own panes.
			//
			// Phase 3 migrates this inline call to a registered Observer.
			if s.Backend != backend.Persistent {
				if reply := generateAnsiResponses(data); len(reply) > 0 {
					// Server-origin reply; deliver as keystroke. For the
					// standard backend this is just a pipe write.
					if werr := p.WriteInput(reply, pty.KindKeystroke); werr != nil {
						log.Printf("session %s: ansi-responder write failed: %v", s.ID, werr)
					}
				}
			}
			// Prepend any incomplete UTF-8 bytes from the previous read.
			if len(s.utf8Buf) > 0 {
				data = append(s.utf8Buf, data...)
				s.utf8Buf = nil
			}
			complete, remainder := splitCompleteUTF8(data)
			if len(complete) > 0 {
				s.broadcast(complete)
				s.dispatchObservers(ObserverFrame{Decoded: complete})
			}
			// Copy remainder — it may alias the read buffer which is reused.
			if len(remainder) > 0 {
				s.utf8Buf = append([]byte(nil), remainder...)
			}
		}
		if err != nil {
			// For persistent (tmux) sessions, the attach process can die while
			// the underlying tmux session survives. Retry re-attach with
			// exponential backoff before declaring the session dead. A single
			// transient failure should not permanently destroy a session.
			if s.Backend == backend.Persistent {
				s.mu.Lock()
				isClosing := s.closing
				s.mu.Unlock()
				if !isClosing {
					sessionName := s.sessionPrefix + s.ID
					reattached := false
					for attempt := 0; attempt < tmuxReattachMaxRetries; attempt++ {
						delay := tmuxReattachBaseDelay << attempt // 500ms, 1s, 2s
						time.Sleep(delay)
						// Re-check closing in case Shutdown() was called during backoff.
						s.mu.Lock()
						isClosing = s.closing
						s.mu.Unlock()
						if isClosing {
							log.Printf("session %s: shutdown detected during re-attach backoff, stopping retries", s.ID)
							break
						}
						if s.metrics != nil {
							s.metrics.ReattachAttempts.Add(1)
						}
						newPTY, reattachErr := s.reattachFunc(sessionName)
						if reattachErr == nil {
							log.Printf("session %s: tmux attach process died, re-attached successfully (attempt %d)", s.ID, attempt+1)
							if s.metrics != nil {
								s.metrics.ReattachSuccesses.Add(1)
							}
							s.mu.Lock()
							oldPTY := s.pty
							s.pty = newPTY
							s.mu.Unlock()
							// Close the old PTY fd to prevent file descriptor leaks.
							// The old attach process has already exited (that's why
							// we're here), but its PTY master fd is still open.
							_ = oldPTY.Close()
							_ = newPTY.SetSize(s.Cols, s.Rows)
							reattached = true
							break
						}
						log.Printf("session %s: tmux re-attach attempt %d/%d failed (read err: %v, attach err: %v)",
							s.ID, attempt+1, tmuxReattachMaxRetries, err, reattachErr)
					}
					if reattached {
						continue
					}
					if s.metrics != nil {
						s.metrics.ReattachFailures.Add(1)
					}
					log.Printf("session %s: all re-attach attempts exhausted, declaring session dead", s.ID)
				}
			}

			// Flush any remaining incomplete UTF-8 bytes — there is no more
			// data coming, so send them as-is (the terminal will handle them).
			if len(s.utf8Buf) > 0 {
				s.broadcast(s.utf8Buf)
				s.dispatchObservers(ObserverFrame{Decoded: s.utf8Buf})
				s.utf8Buf = nil
			}
			exitCode := s.pty.ExitCode()
			s.mu.Lock()
			s.processExited = true
			s.processExitCode = exitCode
			for ch := range s.clients {
				close(ch)
				delete(s.clients, ch)
			}
			s.mu.Unlock()
			close(s.exitCh)
			return
		}
	}
}

// splitCompleteUTF8 splits data at the last complete UTF-8 codepoint boundary.
// Returns (complete, remainder) where remainder contains any trailing incomplete
// multi-byte sequence that should be buffered for the next read.
//
// UTF-8 encoding rules used:
//   - 0xxxxxxx: 1-byte (ASCII)
//   - 110xxxxx: 2-byte leading byte
//   - 1110xxxx: 3-byte leading byte
//   - 11110xxx: 4-byte leading byte
//   - 10xxxxxx: continuation byte
//
// If the trailing bytes are orphaned continuation bytes with no leading byte,
// they are treated as complete (passed through as-is) since they represent
// pre-existing corruption, not a split boundary.
func splitCompleteUTF8(data []byte) (complete, remainder []byte) {
	if len(data) == 0 {
		return nil, nil
	}

	// Walk backward from the end to find the start of a potential
	// incomplete multi-byte sequence.
	i := len(data) - 1

	// Skip continuation bytes (10xxxxxx).
	contCount := 0
	for i >= 0 && data[i]&0xC0 == 0x80 {
		contCount++
		i--
	}

	// If we consumed the entire slice as continuation bytes, there's no
	// leading byte — treat as pre-existing corruption, pass through.
	if i < 0 {
		return data, nil
	}

	b := data[i]
	var expectedLen int
	switch {
	case b&0x80 == 0:
		// ASCII byte — not a multi-byte sequence leader.
		return data, nil
	case b&0xE0 == 0xC0:
		expectedLen = 2
	case b&0xF0 == 0xE0:
		expectedLen = 3
	case b&0xF8 == 0xF0:
		expectedLen = 4
	default:
		// Invalid leading byte — pass through.
		return data, nil
	}

	actualLen := contCount + 1 // leading byte + continuation bytes
	if actualLen >= expectedLen {
		// Sequence is complete.
		return data, nil
	}

	// Incomplete sequence: split before the leading byte.
	if i == 0 {
		return nil, data
	}
	return data[:i], data[i:]
}
