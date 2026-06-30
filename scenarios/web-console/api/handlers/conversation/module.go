// Package conversation is the HTTP-handler home for the conversation domain.
// It exposes the generated Connect-RPC ConversationService (proto schema:
// packages/proto/schemas/web-console/v1/conversation).
package conversation

import (
	"context"
	"log"
	"web-console/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation/conversation_v1connect"
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main (adapts ConversationStore and the TTS
// summarizer to satisfy this interface).
type Service interface {
	Get(sessionID string, sinceSequence int64) (SessionState, error)
	UpdateCursor(sessionID string, patch CursorPatch) (Cursor, error)
	SummarizeEvent(ctx context.Context, sessionID, eventID string) (SummarizeResult, error)
}

// Event mirrors the legacy JSON shape of one stored conversation entry.
type Event struct {
	ID                       string
	SessionID                string
	Source                   string
	Role                     string
	Text                     string
	SpeechParagraphs         []string
	OriginalSpeechParagraphs []string
	Summarized               bool
	CreatedAt                string
	Sequence                 int64
	DeliveryState            string
	TTSState                 string
	ConsumptionState         string
}

// Cursor is the transport-neutral cursor shape.
type Cursor struct {
	LastSeenSequence     int64
	LastListenedSequence int64
}

// SessionState bundles a session's events and current cursor.
type SessionState struct {
	SessionID string
	Events    []Event
	Cursor    Cursor
}

// CursorPatch carries cursor field overrides; each Has* flag indicates whether
// the paired field should be applied.
type CursorPatch struct {
	LastSeenSequence        int64
	HasLastSeenSequence     bool
	LastListenedSequence    int64
	HasLastListenedSequence bool
}

// SummarizeResult mirrors the legacy soft-failure response: when Summarized is
// false, Error carries the user-visible reason (model offline, empty output).
type SummarizeResult struct {
	Summarized       bool
	SpeechParagraphs []string
	Error            string
}

// Module wires the conversation domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := conversationconnect.NewConversationServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "conversation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
