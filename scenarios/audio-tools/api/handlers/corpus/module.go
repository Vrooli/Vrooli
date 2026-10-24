// Package corpus hosts the CorpusService Connect-RPC handler — the
// operator-facing CRUD surface over the speech-eval clip store
// (internal/corpus). It re-exports the domain Schema() for the
// modules.AllSchemas registry.
package corpus

import (
	intcorpus "audio-tools/internal/corpus"
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	corpusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus/corpus_v1connect"
)

// Deps is the corpus handler's dependency bundle.
type Deps struct {
	Logger  logx.Logger
	Clock   schedule.Clock
	Service *intcorpus.Service
}

// Module wires the CorpusService Connect handler.
func Module(d Deps) modulekit.Module {
	if d.Logger == nil {
		panic("corpus.Module requires Deps.Logger")
	}
	if d.Clock == nil {
		panic("corpus.Module requires Deps.Clock")
	}
	connectPath, h := corpusconnect.NewCorpusServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "corpus",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the internal corpus domain schema for EnsureSchemas.
func Schema() string { return intcorpus.Schema() }

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "corpus.create_clip", Path: "/vrooli.audio_tools.v1.corpus.CorpusService/CreateClip", Method: "POST", Category: "corpus"},
	{ID: "corpus.list_clips", Path: "/vrooli.audio_tools.v1.corpus.CorpusService/ListClips", Method: "POST", Category: "corpus"},
	{ID: "corpus.get_clip", Path: "/vrooli.audio_tools.v1.corpus.CorpusService/GetClip", Method: "POST", Category: "corpus"},
	{ID: "corpus.get_clip_audio", Path: "/vrooli.audio_tools.v1.corpus.CorpusService/GetClipAudio", Method: "POST", Category: "corpus"},
	{ID: "corpus.delete_clip", Path: "/vrooli.audio_tools.v1.corpus.CorpusService/DeleteClip", Method: "POST", Category: "corpus"},
}
