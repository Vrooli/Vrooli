// Package learning exposes agent learning over the shared journal.
package learning

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning/learningv1connect"
	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "learning_record", Path: rpc.LearningServiceRecordAttemptProcedure, Method: "POST", Summary: "Record an outcome-linked learning attempt", Category: "learning", Request: &module.Schema{Type: "RecordAttemptRequest"}, Response: &module.Schema{Type: "RecordAttemptResponse"}},
	{ID: "learning_observation", Path: rpc.LearningServiceRecordObservationProcedure, Method: "POST", Summary: "Record append-only feedback for an attempt", Category: "learning", Request: &module.Schema{Type: "RecordObservationRequest"}, Response: &module.Schema{Type: "RecordObservationResponse"}},
	{ID: "learning_measure", Path: rpc.LearningServiceMeasureLearningProcedure, Method: "POST", Summary: "Measure learning outcomes by comparable context", Category: "learning", Request: &module.Schema{Type: "MeasureLearningRequest"}, Response: &module.Schema{Type: "MeasureLearningResponse"}},
}
var ProtoFile = pb.File_vrooli_memory_v1_learning_learning_proto

func Module(client *ledgerclient.Client) module.Module {
	path, handler := rpc.NewLearningServiceHandler(NewHandler(client))
	return module.Module{Name: "learning", Endpoints: Endpoints, Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }}
}
