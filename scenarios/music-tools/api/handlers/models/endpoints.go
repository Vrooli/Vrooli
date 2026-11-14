package models

import "music-tools/internal/module"

func registryPayload(response string) *module.RESTException {
	return &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "REST projection of the model registry; canonical namespaced IDs are accepted.", ProtoPayloads: &module.RESTProtoPayloads{
		Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
		Response: module.RESTPayload{ProtoFullName: response, Transport: "json", Conformance: "protojson"},
		Error:    module.RESTPayload{ProtoFullName: "vrooli.music_tools.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
	}}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "models_list", Path: "/api/v1/models", Method: "GET", Summary: "List declared music models, capabilities and licence lanes.", Category: "models", RESTException: registryPayload("vrooli.music_tools.v1.models.ListModelsResponse")},
	{ID: "models_get", Path: "/api/v1/models/{id}", Method: "GET", Summary: "Get one namespaced music model.", Category: "models", RESTException: registryPayload("vrooli.music_tools.v1.models.Model")},
}
