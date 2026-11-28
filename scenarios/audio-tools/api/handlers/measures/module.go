// Package measures serves Audio Tools' typed, aggregate-only MeasuresService
// and the shared measures-go registry used by measures-health and search-hub.
package measures

import (
	"net/http"
	"strings"

	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/schedule"
	measures "github.com/vrooli/measures-go"
	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/measures/measures_v1connect"
)

type Deps struct {
	BYOK             BYOKCounter
	Corpus           CorpusCounter
	Experiments      ExperimentCounter
	Speakers         SpeakerCounter
	Wakewords        WakewordCounter
	DuplexAccounting DuplexAccountingCounter
	Logger           Logger
	Clock            schedule.Clock
}

// Module mounts the typed Connect measure calls and measures-go's standard
// declaration/execute surface. Both transports call the same owner counters.
func Module(d Deps) modulekit.Module {
	if d.Clock == nil {
		panic("measures.Module requires Deps.Clock")
	}
	if d.Logger == nil {
		panic("measures.Module requires Deps.Logger")
	}
	registry := measures.NewRegistry(measures.WithClock(d.Clock.Now))
	if err := registerMeasures(registry, d); err != nil {
		panic("measures.Module register declarations: " + err.Error())
	}
	connectPath, connectHandler := measuresconnect.NewMeasuresServiceHandler(NewConnectHandler(d))
	registryHandler := registry.Handler()
	return modulekit.Module{
		Name: "measures",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			if strings.HasPrefix(connectPath, "/vrooli.") {
				legacyConnectPath := "/" + strings.TrimPrefix(connectPath, "/vrooli.")
				r.PathPrefix(legacyConnectPath).Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					canonicalReq := req.Clone(req.Context())
					canonicalURL := *req.URL
					canonicalURL.Path = "/vrooli." + strings.TrimPrefix(req.URL.Path, "/")
					canonicalReq.URL = &canonicalURL
					connectHandler.ServeHTTP(w, canonicalReq)
				}))
			}
			// The shared measures-go handler exposes exactly two JSON operations.
			r.Handle("/measures/declarations", http.StripPrefix("/measures", registryHandler)).Methods(http.MethodGet)
			r.Handle("/measures/execute", http.StripPrefix("/measures", registryHandler)).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID: "measures.registry.declarations", Path: "/measures/declarations", Method: http.MethodGet,
		Summary: "List declarations in the shared measures-go registry", Category: "measure",
		RESTException: &modulekit.RESTException{
			Reason: modulekit.RESTReasonThirdPartyShape,
			Note:   "The shared measures-go library owns this cross-scenario JSON wire contract.",
			ProtoPayloads: &modulekit.RESTProtoPayloads{
				Request:  modulekit.RESTPayload{Transport: "none", Conformance: "none"},
				Response: modulekit.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    modulekit.RESTPayload{Transport: "json", Conformance: "external_shape"},
			},
		},
	},
	{
		ID: "measures.registry.execute", Path: "/measures/execute", Method: http.MethodPost,
		Summary: "Execute a declared measure through the shared measures-go registry", Category: "measure",
		RESTException: &modulekit.RESTException{
			Reason: modulekit.RESTReasonThirdPartyShape,
			Note:   "The shared measures-go library owns this cross-scenario JSON wire contract.",
			ProtoPayloads: &modulekit.RESTProtoPayloads{
				Request:  modulekit.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Response: modulekit.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    modulekit.RESTPayload{Transport: "json", Conformance: "external_shape"},
			},
		},
	},
	{ID: "measures.byok_credentials_created", Path: measuresconnect.MeasuresServiceCountBYOKCredentialsProcedure, Method: http.MethodPost, Category: "measure"},
	{ID: "measures.corpus_clips_created", Path: measuresconnect.MeasuresServiceCountCorpusClipsProcedure, Method: http.MethodPost, Category: "measure"},
	{ID: "measures.experiment_runs_created", Path: measuresconnect.MeasuresServiceCountExperimentRunsProcedure, Method: http.MethodPost, Category: "measure"},
	{ID: "measures.experiments_created", Path: measuresconnect.MeasuresServiceCountExperimentsProcedure, Method: http.MethodPost, Category: "measure"},
	{ID: "measures.speaker_profiles_created", Path: measuresconnect.MeasuresServiceCountSpeakerProfilesProcedure, Method: http.MethodPost, Category: "measure"},
	{ID: "measures.wakeword_templates_created", Path: measuresconnect.MeasuresServiceCountWakewordTemplatesProcedure, Method: http.MethodPost, Category: "measure"},
	{ID: "measures.duplex_accounting_operations_created", Path: measuresconnect.MeasuresServiceCountDuplexAccountingOperationsProcedure, Method: http.MethodPost, Category: "measure"},
}
