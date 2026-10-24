package distribution

import "scenario-to-ios/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "ios_distribution", Path: "/api/v1/ios/distribution", Method: "GET", Summary: "Show independent iOS distribution channels", Category: "distribution", RESTException: module.OpsRESTException("Operator report surface.")},
}
