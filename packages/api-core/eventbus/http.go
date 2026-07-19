package eventbus

import (
	"bytes"
	"net/http"
	"time"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

// VerifiedCorrelation extracts the only run attribution that a generic API
// surface may trust. Invocation headers are intentionally excluded: they are
// annotations, not proof that an Agent Manager run made the request.
func VerifiedCorrelation(r *http.Request) Correlation {
	p := provenance.FromContext(r.Context())
	if !p.IsVerifiedAgent() {
		return Correlation{}
	}
	return Correlation{RequestID: p.Invocation.InvocationID, RunID: p.RunID, TaskID: p.TaskID, WorkflowExecutionID: p.WorkflowExecutionID, WorkflowNodeID: p.WorkflowNodeID, Attempt: p.Attempt}
}

// VerifiedIdentityToken returns the inbound opaque token only when the server
// provenance middleware already verified it. This permits the post-response
// receipt hop to be independently verified without treating a forged header as
// evidence.
func VerifiedIdentityToken(r *http.Request) string {
	if r == nil || !provenance.FromContext(r.Context()).IsVerifiedAgent() {
		return ""
	}
	return r.Header.Get(cliutil.HeaderAgentIdentityToken)
}

// Projection returns a pre-sanitised typed response projection. ok=false is
// the safe default for raw, streaming, large, or sensitive responses.
type Projection func(*http.Request, int, []byte) (fields map[string]any, ok bool)

// ReceiptProjectionPolicy is supplied by the refreshed local policy cache.
// A nil policy means receipt emission is disabled rather than self-authorized.
type ReceiptProjectionPolicy interface {
	ProjectReceipt(source, target, operation string, candidate map[string]any) (projection map[string]any, policyVersion string, ok bool)
}

// MiddlewareConfig describes one standard typed server surface. It contains no
// remote dependency and is therefore safe when Vrooli Events is unavailable.
type MiddlewareConfig struct {
	Source, Target    string
	SourceFromRequest func(*http.Request) string
	Operation         func(*http.Request) string
	Reporter          Client
	Projection        Projection
	ReceiptPolicy     ReceiptProjectionPolicy
	Correlation       func(*http.Request) Correlation
}

// Middleware applies local traffic policy before the handler and launches
// receipt delivery only after the handler has produced its HTTP status.
func Middleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Platform receipts are audit evidence, not anonymous telemetry. A
			// request without verified Agent Manager provenance must not create an
			// unverified receipt that could later be mistaken for agent work.
			if !provenance.FromContext(r.Context()).IsVerifiedAgent() {
				next.ServeHTTP(w, r)
				return
			}
			source := cfg.Source
			if cfg.SourceFromRequest != nil {
				source = cfg.SourceFromRequest(r)
			}
			op := r.Method + " " + r.URL.Path
			if cfg.Operation != nil {
				op = cfg.Operation(r)
			}
			started := time.Now()
			recorder := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			if cfg.Projection == nil {
				return
			}
			projection, ok := cfg.Projection(r, recorder.status, recorder.body.Bytes())
			if !ok {
				return
			}
			if cfg.ReceiptPolicy == nil {
				return
			}
			policyVersion := ""
			projection, policyVersion, ok = cfg.ReceiptPolicy.ProjectReceipt(source, cfg.Target, op, projection)
			if !ok {
				return
			}
			correlation := Correlation{}
			if cfg.Correlation != nil {
				correlation = cfg.Correlation(r)
			}
			// A receipt without verified run correlation is not an Agent Manager
			// receipt. Other producers may supply their own correlation contract.
			cfg.Reporter.PublishAsync(Receipt{Source: source, Target: cfg.Target, Operation: op,
				Outcome: outcome(recorder.status), StatusCode: recorder.status, Duration: time.Since(started),
				PolicyVer: policyVersion, Projection: projection, Correlation: correlation, SubjectID: VerifiedSubjectID(r),
				ActorKind: actorKind(r), IdentityToken: VerifiedIdentityToken(r)})
		})
	}
}

func actorKind(r *http.Request) string {
	if provenance.FromContext(r.Context()).IsVerifiedAgent() {
		return "agent"
	}
	return "system"
}

func VerifiedSubjectID(r *http.Request) string {
	if r == nil {
		return ""
	}
	p := provenance.FromContext(r.Context())
	if !p.IsVerifiedAgent() {
		return ""
	}
	return p.ProfileKey
}

type statusWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusWriter) Write(body []byte) (int, error) {
	if w.body.Len() < 256*1024 {
		_, _ = w.body.Write(body[:min(len(body), 256*1024-w.body.Len())])
	}
	return w.ResponseWriter.Write(body)
}
func outcome(status int) string {
	if status >= 200 && status < 400 {
		return "success"
	}
	return "error"
}
