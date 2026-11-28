package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/types/known/structpb"
)

// TerminalAdmissionScope is supplied by the authenticated owner boundary, not
// the request. The seed must match the current installation and source epoch.
type TerminalAdmissionScope struct{ SourceInstance, SourceEpoch string }

// TerminalAdmissionBoundary authenticates the caller and authorizes this exact
// local owner operation using the existing authority owner. It MUST serialize
// revocation/replacement through the entire synchronous callback (exactly once),
// supply current source scope and honor cancellation. Merely reading cached
// claims, request headers or a request-supplied epoch is not an implementation.
// Authority covers admission through COMMIT. The acknowledgment proves that
// durable acceptance, not continuing permission; every replay reauthorizes.
// Success is emitted only after the boundary returns without error.
// No production boundary, credential or authorization grant is installed here.
type TerminalAdmissionBoundary interface {
	WithTerminalAdmission(context.Context, *http.Request, func(context.Context, TerminalAdmissionScope) error) error
}

var errTerminalAdmissionBoundary = errors.New("terminal admission boundary rejected")

func (s *Server) registerTerminalAdmission(mux *http.ServeMux) {
	if s.terminalAdmissionBoundary != nil {
		mux.HandleFunc("POST "+eventbus.TerminalAdmissionPath, s.handleTerminalAdmission)
	}
}

func (s *Server) handleTerminalAdmission(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.terminalAdmissionBoundary == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost || r.URL.Path != eventbus.TerminalAdmissionPath || r.URL.RawQuery != "" || r.URL.ForceQuery {
		http.Error(w, "invalid terminal admission request", http.StatusBadRequest)
		return
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || r.Header.Get("Content-Encoding") != "" {
		http.Error(w, "invalid terminal admission type", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, eventbus.TerminalAdmissionMaxRequestBytes))
	if err != nil {
		http.Error(w, "invalid terminal admission body", http.StatusBadRequest)
		return
	}
	request, err := eventbus.DecodeTerminalAdmissionRequest(body)
	if err != nil {
		http.Error(w, "invalid terminal admission intent", http.StatusBadRequest)
		return
	}
	env, _, _ := eventbus.RetainedDomainEventEnvelope(request.IntentJSON, request.EventID)
	var facts structpb.Struct
	if env.EventType != terminalIntentEventType || env.GetSource().GetScenario() != "agent-manager" || env.Data == nil || env.Data.UnmarshalTo(&facts) != nil || validateTerminalIntentFacts(&facts) != nil {
		http.Error(w, "invalid terminal facts", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	// The request context remains effective even if an incorrectly implemented
	// boundary passes a detached context. Closing the callback prevents late use.
	var mu sync.Mutex
	var returned atomic.Bool
	calls := 0
	var ack eventbus.DomainEventAcknowledgment
	var callbackErr error
	var authorityContext context.Context
	boundaryErr := s.terminalAdmissionBoundary.WithTerminalAdmission(ctx, r, func(inner context.Context, scope TerminalAdmissionScope) error {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if returned.Load() || calls != 1 || inner == nil || inner.Err() != nil || ctx.Err() != nil || !terminalScopeToken.MatchString(scope.SourceInstance) || !terminalScopeToken.MatchString(scope.SourceEpoch) || scope.SourceInstance != facts.Fields["source_instance"].GetStringValue() || scope.SourceEpoch != facts.Fields["source_epoch"].GetStringValue() {
			callbackErr = errTerminalAdmissionBoundary
			return callbackErr
		}
		authorityContext = inner
		// Preserve request routing and pinned storage values. The boundary may
		// add cancellation/deadlines, but must not replace the request context.
		deadline, _ := ctx.Deadline()
		if innerDeadline, ok := inner.Deadline(); ok && innerDeadline.Before(deadline) {
			deadline = innerDeadline
		}
		joined, stop := context.WithDeadline(ctx, deadline)
		stopBoundary := context.AfterFunc(inner, stop)
		defer stopBoundary()
		defer stop()
		if inner.Err() != nil {
			stop()
		}
		ack, callbackErr = s.acceptTerminalEventIntent(joined, request.IntentJSON, request.EventID)
		if joined.Err() != nil || inner.Err() != nil || returned.Load() {
			callbackErr = errTerminalAdmissionBoundary
		}
		return callbackErr
	})
	returned.Store(true)
	requestErr := ctx.Err()
	mu.Lock()
	defer mu.Unlock()
	if boundaryErr != nil || callbackErr != nil || calls != 1 || requestErr != nil || r.Context().Err() != nil || authorityContext == nil || authorityContext.Err() != nil {
		status := http.StatusForbidden
		if errors.Is(callbackErr, store.ErrTerminalAdmissionConflict) || errors.Is(callbackErr, store.ErrDuplicateEvent) {
			status = http.StatusConflict
		}
		http.Error(w, "terminal admission not acknowledged", status)
		return
	}
	if eventbus.ValidateDomainEventAcknowledgment(request.IntentJSON, request.EventID, ack) != nil {
		http.Error(w, "terminal admission not acknowledged", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(ack)
}
