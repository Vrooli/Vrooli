package transformation

import "music-tools/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "transformation-remix-submit", Path: "/api/v1/transformations/remix", Method: "POST", Summary: "Queue a hash-bound audio remix", Description: "Queue a CPU transformation of one exact managed source excerpt. The successful job records an immutable source/output hash receipt; no generative model is selected.", Category: "transformation", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "This local job ingress carries managed blob identity, exact hashes and bounded time ranges; its evolving receipt is an owner REST contract."}},
	{ID: "transformation-sample-submit", Path: "/api/v1/transformations/sample", Method: "POST", Summary: "Queue a hash-bound sample arrangement", Description: "Queue a CPU sample arrangement by repeating one exact managed excerpt. The successful job records an immutable source/output hash receipt; no generative model is selected.", Category: "transformation", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "This local job ingress carries managed blob identity, exact hashes and bounded time ranges; its evolving receipt is an owner REST contract."}},
	{ID: "transformation-audio", Path: "/api/v1/jobs/{id}/audio", Method: "GET", Summary: "Stream owner-verified transformation audio", Description: "Stream only the derived WAV whose bytes match the successful Music Tools job receipt; supports private byte-range review.", Category: "transformation", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "The listening room needs byte-range access to the exact derived output, whose content hash is checked before streaming."}},
}
