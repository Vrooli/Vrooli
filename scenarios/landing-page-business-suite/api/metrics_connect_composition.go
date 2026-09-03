package main

import (
	metricshttp "landing-page-business-suite-api/handlers/metrics"
	domainmetrics "landing-page-business-suite-api/internal/metrics"
)

// metricsConnectDependencies wires the metrics domain to its generated
// transport. Request parsing and serialization are owned by Connect.
func metricsConnectDependencies(service *domainmetrics.Service) metricshttp.ConnectDependencies {
	return metricshttp.ConnectDependencies{Tracker: service, Reader: service, Revenue: service}
}
