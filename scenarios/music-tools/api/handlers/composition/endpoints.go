package composition

import (
	compositionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/composition/composition_v1connect"
	"music-tools/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "composition-submit", Path: "/vrooli.music_tools.v1.composition.CompositionService/Submit", Method: "POST", Summary: "Submit a composition batch", Category: "composition"},
	{ID: "composition-list-takes", Path: "/vrooli.music_tools.v1.composition.CompositionService/ListTakes", Method: "POST", Summary: "List peer takes", Category: "composition"},
	{ID: "transformation-derivation-receipt", Path: "/api/v1/jobs/{id}/derivation-receipt", Method: "GET", Summary: "Verify a remix or sample job receipt", Description: "Return immutable source hashes, excerpt ranges, output hash, operation/tool provenance and rights scope only for a successful Music Tools transformation job.", Category: "transformation", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "This owner receipt is a service-to-service proof projection from the durable job record; callers cannot submit or alter it."}},
	{ID: "candidate-refill-request", Path: compositionconnect.CandidateInventoryServiceRequestRefillProcedure, Method: "POST", Summary: "Record a durable candidate refill request", Description: "Idempotently records a Music Library consumption event in the unreviewed candidate queue. Requests remain held until explicitly dispatched after research and capacity gates.", Category: "composition"},
	{ID: "candidate-refill-cancel", Path: compositionconnect.CandidateInventoryServiceCancelRefillProcedure, Method: "POST", Summary: "Cancel a candidate refill request", Description: "Cancels a held refill request using its stable source reservation identity.", Category: "composition"},
}
