package models

import "music-tools/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "models_list", Path: "/api/v1/models", Method: "GET", Summary: "List available music models and licence lanes.", Category: "models", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "REST projection of the model registry."}},
	{ID: "models_get", Path: "/api/v1/models/{id}", Method: "GET", Summary: "Get one music model.", Category: "models", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "REST projection of the model registry."}},
}
