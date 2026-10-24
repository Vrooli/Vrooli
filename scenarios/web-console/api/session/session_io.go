package session

import (
	"log"
	"time"
	"unicode/utf8"

	"web-console/internal/backend"
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
//  4. Signals exitCh so the Manager can clean up
//
// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
func (s *Session) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("session %s: readLoop panic (recovered): %v", s.ID, r)
			// Ensure exit signaling even after a panic so waiters don't hang.
			s.clientsMu.Lock()
			s.emuMu.Lock()
			if !s.processExited {
				s.stopInputWriter()
				s.processExited = true
				s.processExitCode = -1
				for ch := range s.clients {
					if info := s.clients[ch]; info != nil && info.FrameCh != nil {
						close(info.FrameCh)
					}
					close(ch)
					delete(s.clients, ch)
				}
			}
			s.emuMu.Unlock()
			s.clientsMu.Unlock()
			select {
			case <-s.exitCh:
			default:
				close(s.exitCh)
			}
		}
	}()
	buf := make([]byte, s.ptyReadBuffer)
	for {
		s.emuMu.Lock()
		p := s.currentPTY()
		s.emuMu.Unlock()
		n, err := p.Read(buf)
		if n > 0 {
			data := buf[:n]
			// ANSI terminal-query replies (DA1/DA3/XTVERSION) are handled
			// server-side; xterm owns synchronized-output DECRQM state.
			// are produced by the responder goroutine started in
			// startAnsiResponder() — it subscribes to the emulator's
			// ControlEvent channel and writes replies through SendInput.
			// No inline scan here.
			//
			// Prepend any incomplete UTF-8 bytes from the previous read.
			if len(s.utf8Buf) > 0 {
				data = append(s.utf8Buf, data...)
				s.utf8Buf = nil
			}
			complete, remainder := splitCompleteUTF8(data)
			if len(complete) > 0 {
				s.broadcast(complete)
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
				s.emuMu.Lock()
				isClosing := s.closing
				s.emuMu.Unlock()
				if !isClosing {
					sessionName := s.sessionPrefix + s.ID
					reattached := false
					for attempt := 0; attempt < tmuxReattachMaxRetries; attempt++ {
						delay := tmuxReattachBaseDelay << attempt // 500ms, 1s, 2s
						time.Sleep(delay)
						// Re-check closing in case Shutdown() was called during backoff.
						s.emuMu.Lock()
						isClosing = s.closing
						s.emuMu.Unlock()
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
							oldPTY := s.replacePTY(newPTY)
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
				s.utf8Buf = nil
			}
			exitCode := s.currentPTY().ExitCode()
			s.stopInputWriter()
			s.clientsMu.Lock()
			s.emuMu.Lock()
			s.processExited = true
			s.processExitCode = exitCode
			for ch := range s.clients {
				if info := s.clients[ch]; info != nil && info.FrameCh != nil {
					close(info.FrameCh)
				}
				close(ch)
				delete(s.clients, ch)
			}
			s.emuMu.Unlock()
			s.clientsMu.Unlock()
			close(s.exitCh)
			return
		}
	}
}

// splitCompleteUTF8 splits data at the last complete UTF-8 codepoint boundary.
// Returns (complete, remainder) where remainder contains any trailing incomplete
// multi-byte sequence that should be buffered for the next read.
//
// A trailing run of orphaned continuation bytes is treated as complete and
// passed through. Only a rune start whose suffix is not yet a complete UTF-8
// sequence is buffered for the next read.
func splitCompleteUTF8(data []byte) (complete, remainder []byte) {
	if len(data) == 0 {
		return nil, nil
	}
	start := max(0, len(data)-4)
	for i := len(data) - 1; i >= start; i-- {
		if utf8.RuneStart(data[i]) && !utf8.FullRune(data[i:]) {
			return data[:i], data[i:]
		}
	}
	return data, nil
}
