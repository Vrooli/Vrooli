// Package metrics owns the descriptor for the operational metrics
// scrape endpoint. The handler continues to live in api/metrics.go;
// this package only exposes the canonical metadata so gen-endpoints can
// validate the route under the RESTException rule. Metrics stay REST
// because monitoring tools that scrape this endpoint dictate the wire
// shape and cannot be wrapped in a generated Connect client.
package metrics

import "web-console/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "metrics_snapshot",
		Path:        "/api/v1/metrics",
		Method:      "GET",
		Summary:     "Operational metrics snapshot",
		Description: "Returns a JSON snapshot of all operational metrics for monitoring/scrape tooling.",
		Category:    "system",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonThirdPartyShape,
			Note:   "Metrics are consumed by external monitoring tooling whose scrape format we do not control. Stays REST and never becomes Connect.",
		},
	},
}
