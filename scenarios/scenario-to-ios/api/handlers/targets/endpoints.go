package targets

import "scenario-to-ios/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "ios_targets", Path: "/api/v1/ios/targets", Method: "GET", Summary: "Probe Apple targets and bridge readiness", Category: "targets", RESTException: module.OpsRESTException("Operator probe surface returns report-shaped JSON.")},
}
