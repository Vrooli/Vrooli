// Package voice is the HTTP-handler home for the voice domain. It exposes
// the generated Connect-RPC VoiceService (proto schema:
// packages/proto/schemas/web-console/v1/voice).
//
// All domain types and service logic live in web-console/internal/voice.
// This package is transport only: it imports the internal domain, wires the
// Connect handler, and maps domain errors to Connect codes.
package voice

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	voiceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice/voice_v1connect"

	"web-console/internal/module"
	intvoice "web-console/internal/voice"
)

// Re-exports so transport-layer code (connect_handler.go, tests, package main)
// can keep using familiar names without importing internal/voice everywhere.
type (
	Service             = intvoice.HandlerService
	Backend             = intvoice.Backend
	Adapter             = intvoice.Adapter
	TranscribeInput     = intvoice.TranscribeInput
	StreamConfig        = intvoice.Config
	StreamConfigPatch   = intvoice.ConfigPatch
	WakeWordConfig      = intvoice.WakeWordConfig
	SpeakerConfig       = intvoice.SpeakerConfig
	SpeakerConfigPatch  = intvoice.SpeakerConfigPatch
	SpeakerProfile      = intvoice.SpeakerProfile
	SpeakerResourceInfo = intvoice.SpeakerResourceInfo
	SpeakerStatus       = intvoice.SpeakerStatus
	SpeakerEnrollment   = intvoice.SpeakerEnrollment
	EnrollInput         = intvoice.EnrollInput
	SpeakerDecision     = intvoice.SpeakerDecision
)

// Sentinel errors re-exported from internal/voice so the connect handler's
// classify() can keep matching on them via errors.Is.
var (
	ErrInvalidArgument = intvoice.ErrInvalidArgument
	ErrUnavailable     = intvoice.ErrUnavailable
	ErrInternal        = intvoice.ErrInternal
)

// Module wires the voice domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := voiceconnect.NewVoiceServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "voice",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
