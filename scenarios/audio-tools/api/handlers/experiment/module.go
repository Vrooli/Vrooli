// Package experiment hosts the ExperimentService Connect-RPC handler: the
// persisted, async STT experiment lifecycle surface for Dictation Studio.
package experiment

import (
	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
)

func Module(d Deps) modulekit.Module {
	if d.Logger == nil {
		panic("experiment.Module requires Deps.Logger")
	}
	connectPath, h := experimentconnect.NewExperimentServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "experiment",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "experiment.start", Path: experimentconnect.ExperimentServiceStartExperimentProcedure, Method: "POST", Category: "experiment"},
	{ID: "experiment.get", Path: experimentconnect.ExperimentServiceGetExperimentProcedure, Method: "POST", Category: "experiment"},
	{ID: "experiment.wait", Path: experimentconnect.ExperimentServiceWaitExperimentProcedure, Method: "POST", Category: "experiment"},
	{ID: "experiment.list", Path: experimentconnect.ExperimentServiceListExperimentsProcedure, Method: "POST", Category: "experiment"},
	{ID: "experiment.cancel", Path: experimentconnect.ExperimentServiceCancelExperimentProcedure, Method: "POST", Category: "experiment"},
	{ID: "experiment.delete", Path: experimentconnect.ExperimentServiceDeleteExperimentProcedure, Method: "POST", Category: "experiment"},
	{ID: "experiment.stream", Path: experimentconnect.ExperimentServiceStreamExperimentEventsProcedure, Method: "POST", Category: "experiment"},
	{ID: "experiment.report", Path: experimentconnect.ExperimentServiceGetExperimentReportProcedure, Method: "POST", Category: "experiment"},
	{ID: "experiment.compare", Path: experimentconnect.ExperimentServiceCompareExperimentsProcedure, Method: "POST", Category: "experiment"},
}
