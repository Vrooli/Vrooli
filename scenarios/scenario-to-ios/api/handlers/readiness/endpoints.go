package readiness

import "scenario-to-ios/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "ios_readiness", Path: "/api/v1/ios/readiness", Method: "GET", Summary: "Show the probed Apple readiness ladder", Category: "readiness", RESTException: module.OpsRESTException("Operator report surface.")},
}
