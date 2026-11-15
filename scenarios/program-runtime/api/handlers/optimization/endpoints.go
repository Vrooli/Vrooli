package optimization

import (
	optimizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/optimization/optimization_v1connect"
	"program-runtime/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "optimization_submit", Method: "POST", Path: optimizationconnect.OptimizationServiceSubmitProcedure, Summary: "Submit or resume a durable governed optimization run.", Category: "optimization"},
	{ID: "optimization_wait", Method: "POST", Path: optimizationconnect.OptimizationServiceWaitProcedure, Summary: "Wait once for an optimizer stage or terminal receipt.", Category: "optimization"},
}
