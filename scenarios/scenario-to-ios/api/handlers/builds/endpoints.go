package builds

import "scenario-to-ios/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "ios_generate", Path: "/api/v1/ios/generate", Method: "POST", Summary: "Generate a deterministic Capacitor iOS project", Category: "builds", RESTException: module.OpsJSONRESTException("Operator generation surface accepts a JSON source reference.")},
	{ID: "ios_build", Path: "/api/v1/ios/build", Method: "POST", Summary: "Build an iOS artifact on an Apple host", Category: "builds", RESTException: module.OpsJSONRESTException("Operator build surface returns unavailable when no Apple host exists.")},
}
