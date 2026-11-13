package experimentarms

import (
	"net/http"

	internal "search-hub/internal/experimentarms"
	"search-hub/internal/module"

	"github.com/gorilla/mux"
)

const basePath = "/v1/embedding-experiments/arms"

// Module exposes durable embedding-arm identity and CAS state over Search
// Hub's local API. Generation staging is lifecycle-injected and fails closed
// when its verified runtime or capacity admission owner is unavailable.
func Module(store *internal.Registry, stages ...*internal.FleetGenerationStage) module.Module {
	var stage *internal.FleetGenerationStage
	if len(stages) > 0 {
		stage = stages[0]
	}
	h := handler{registry: store, stage: stage}
	return module.Module{
		Name: "experiment-arms",
		Mount: func(r *mux.Router) {
			r.HandleFunc(basePath, h.create).Methods(http.MethodPost)
			r.HandleFunc(basePath+"/{armID}", h.get).Methods(http.MethodGet)
			r.HandleFunc(basePath+"/{armID}/transitions", h.transition).Methods(http.MethodPost)
			r.HandleFunc(basePath+"/{armID}/generation", h.stageGeneration).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

// Endpoints is the machine-readable description used by Search Hub's
// generated endpoint manifest and contract checks.
var Endpoints = []module.EndpointDescriptor{
	{ID: "embedding_arms_create", Path: basePath, Method: "POST", Summary: "Create or read an immutable embedding experiment arm", Description: "Requires verified Search Hub write capability and an approved embedding_experiment.manage operation for the exact arm target. Persists exact recipe, caller/build identities and source evidence references. Owner identity is injected from the trusted Search Hub lifecycle namespace. Repeating an identical arm ID is idempotent; conflicting immutable identity is rejected.", Category: "embedding-experiments", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Search Hub's local lifecycle control API records immutable arm operations before provider work is scheduled."}, Request: &module.Schema{Type: "object", Properties: map[string]string{"experiment_id": "string", "arm_id": "string", "provider_id": "string", "caller_id": "string", "build_id": "string", "recipe": "EmbeddingRecipeInput", "evidence": "array<EvidenceRef>"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"arm": "Durable embedding experiment arm"}}, Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Arm, recipe or evidence identity is invalid"}, {Status: 401, Code: "unauthenticated", Description: "Verified Search Hub identity is required"}, {Status: 403, Code: "permission_denied", Description: "The caller lacks approved arm operation authority"}, {Status: 409, Code: "conflict", Description: "Arm ID already names a different immutable identity"}}},
	{ID: "embedding_arms_get", Path: basePath + "/{armID}", Method: "GET", Summary: "Read a durable embedding experiment arm", Description: "Returns the persisted immutable recipe and current CAS version/state.", Category: "embedding-experiments", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Search Hub lifecycle control reads the registry without requiring a generated client."}, Request: &module.Schema{Type: "object", Properties: map[string]string{"arm_id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"arm": "Durable embedding experiment arm"}}, Errors: []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Arm does not exist"}}},
	{ID: "embedding_arms_transition", Path: basePath + "/{armID}/transitions", Method: "POST", Summary: "CAS transition an embedding experiment arm", Description: "Requires verified Search Hub write capability and an approved embedding_experiment.manage operation for the exact arm target. Promotion, rollback, and deletion decisions additionally require an independent verified human. Enforces the arm state machine, expected version and idempotency key. State recording never invokes physical provider operations.", Category: "embedding-experiments", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Search Hub's local lifecycle control API records arm transitions before provider operations are run."}, Request: &module.Schema{Type: "object", Properties: map[string]string{"expected_state": "string", "expected_version": "integer", "to": "string", "idempotency_key": "string", "receipt": "EvidenceRef (optional)"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"arm": "Transition result at the committed version"}}, Errors: []module.ErrorDesc{{Status: 409, Code: "conflict", Description: "Stale version or idempotency-key conflict"}, {Status: 422, Code: "illegal_transition", Description: "Transition is not allowed by the state machine"}}},
	{ID: "embedding_arm_generation_stage", Path: basePath + "/{armID}/generation", Method: "POST", Summary: "Stage the Search Hub descriptor generation for an immutable arm", Description: "Requires verified Search Hub write capability and an approved embedding_experiment.manage operation for the exact arm target. Uses the arm's persisted recipe and Search Hub's own provider registry source. The runtime encoder and trusted peak estimator are lifecycle dependencies; the Search Hub owner gate binds each Qdrant mutation to a shared-capacity disk job. The endpoint returns 503 while a dependency is absent. Request bodies cannot select recipe, source, Qdrant target, or peak bytes.", Category: "embedding-experiments", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Generation stage remains disabled until Search Hub lifecycle injects a verified encoder and trusted peak estimator; Qdrant writes require a shared-capacity disk-job receipt."}, Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"generation": "Durable staged generation receipt"}}, Errors: []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Verified Search Hub identity is required"}, {Status: 403, Code: "permission_denied", Description: "The caller lacks approved arm operation authority"}, {Status: 404, Code: "not_found", Description: "Arm does not exist"}, {Status: 503, Code: "unavailable", Description: "Verified runtime, trusted peak estimator or capacity admission is unavailable"}}},
}

// Schema exposes the arm registry's additive schema to Search Hub's central
// database schema registry.
func Schema() string { return internal.Schema() }
