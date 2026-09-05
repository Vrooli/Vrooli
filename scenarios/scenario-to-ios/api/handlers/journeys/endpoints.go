package journeys

import "scenario-to-ios/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "ios_conformance_plan", Path: "/api/v1/ios/conformance-plan", Method: "GET", Summary: "Show the twelve iOS conformance chapters", Category: "journeys", RESTException: module.OpsRESTException("Operator report surface.")},
}
