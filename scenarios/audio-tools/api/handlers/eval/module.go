// Package eval hosts the EvalService Connect-RPC handler — the STT
// strategy comparison harness surface. It replays the corpus
// (internal/corpus) through the strategies via the offline harness
// (internal/eval) and returns the WER/compute/latency report. Kept
// separate from CorpusService so clip CRUD and evaluation stay distinct
// (plan §8).
package eval

import (
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/clock"
	intcorpus "audio-tools/internal/corpus"
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval/eval_v1connect"
)

// Deps is the eval handler's dependency bundle.
type Deps struct {
	Logger logx.Logger
	Clock  clock.Clock
	// Corpus loads clip audio + references to replay. Required for RunEval.
	Corpus *intcorpus.Service
	// NewProvider returns a fresh STT provider per replay (the handler wraps
	// it in a MeteredProvider). Production wires sttchain.NewLocalProvider
	// over the live Whisper service; nil disables RunEval (returns
	// FailedPrecondition).
	NewProvider func() sttchain.Provider
	// Defaults supplies the overlap/vad config used when an EvalStrategy
	// leaves a knob unset.
	Defaults stt.StreamConfig
	// SpeakerConfig is an optional per-run experiment speaker snapshot. Nil
	// keeps eval speaker stages off, which preserves the public EvalService's
	// historical behavior and avoids reading the live speaker config cell.
	SpeakerConfig *sttpipeline.SpeakerConfig
	// SpeakerExtractionEnabled and SpeakerVerificationEnabled select which
	// speaker stages this eval run binds. They let experiments run attribution
	// ablations such as extraction-on / verification-off with the same
	// underlying adapter implementations.
	SpeakerExtractionEnabled   bool
	SpeakerVerificationEnabled bool
	// SpeakerResource is the speaker-verification resource client used only
	// when SpeakerConfig enables extraction and/or verification.
	SpeakerResource *sttpipeline.SpeakerClient
}

// Module wires the EvalService Connect handler.
func Module(d Deps) modulekit.Module {
	if d.Logger == nil {
		panic("eval.Module requires Deps.Logger")
	}
	if d.Clock == nil {
		panic("eval.Module requires Deps.Clock")
	}
	connectPath, h := evalconnect.NewEvalServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "eval",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the eval domain's schema contribution — none; eval owns no
// tables (it reads the corpus domain).
func Schema() string { return "" }

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "eval.run_eval", Path: "/vrooli.audio_tools.v1.eval.EvalService/RunEval", Method: "POST", Category: "eval"},
}
