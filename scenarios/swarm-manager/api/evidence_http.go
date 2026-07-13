package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"

	"swarm-manager/internal/evidence"
	"swarm-manager/internal/identity"

	"github.com/vrooli/cli-core/cliutil"
)

// evidenceInvocationMiddleware records a successful CLI invocation as an
// observed fact. The opaque identity token was already verified by the
// identity middleware; command metadata is deliberately only a channel
// observation, not proof that a particular binary executed it.
func (s *Server) evidenceInvocationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &evidenceResponseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if recorder.status >= http.StatusBadRequest || s.evidenceSvc == nil {
			return
		}
		provenance := identity.FromContext(r.Context())
		if !provenance.IsAgent() || strings.TrimSpace(provenance.RunID) == "" {
			return
		}
		invocationID := strings.TrimSpace(r.Header.Get(cliutil.HeaderInvocationID))
		command := strings.TrimSpace(r.Header.Get(cliutil.HeaderInvocationCommand))
		scenario := strings.TrimSpace(r.Header.Get(cliutil.HeaderInvocationScenario))
		if invocationID == "" || command == "" || scenario != "swarm-manager" {
			return
		}
		_, _ = s.evidenceSvc.Ingest(r.Context(), evidence.Observation{
			SourceSystem:  "swarm-manager.cli",
			SourceEventID: invocationID,
			RunID:         provenance.RunID,
			Subject:       evidence.Subject{Kind: "scenario", ID: "swarm-manager"},
			Action:        "cli." + command,
			Confidence:    evidence.ConfidenceObserved,
			Verification:  evidence.VerificationVerified,
			Metadata: map[string]string{
				"command":         command,
				"http_method":     r.Method,
				"response_status": http.StatusText(recorder.status),
			},
		})
	})
}

type evidenceResponseRecorder struct {
	http.ResponseWriter
	status int
}

func (w *evidenceResponseRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *evidenceResponseRecorder) Write(body []byte) (int, error) {
	return w.ResponseWriter.Write(body)
}

func (w *evidenceResponseRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *evidenceResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (w *evidenceResponseRecorder) ReadFrom(reader io.Reader) (int64, error) {
	if from, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return from.ReadFrom(reader)
	}
	return io.Copy(w.ResponseWriter, reader)
}

func (w *evidenceResponseRecorder) Push(target string, options *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, options)
	}
	return http.ErrNotSupported
}
