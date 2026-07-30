package main

import (
	"encoding/json"
	"net/http"

	metricshttp "landing-page-business-suite-api/handlers/metrics"
	domainmetrics "landing-page-business-suite-api/internal/metrics"
)

// metricsHTTPDependencies is API composition: shared response, validation,
// routing, and logging policy is injected into the metrics HTTP boundary.
// Domain request handling itself remains in handlers/metrics.
var metricsHTTPDependencies = metricshttp.Dependencies{
	DecodeJSON:         decodeJSONBody,
	ValidateEmail:      ValidateEmailForHandler,
	PathInt64:          getPathParamInt64,
	WriteSuccess:       writeJSONSuccess,
	WriteSuccessData:   writeJSONSuccessData,
	WriteSuccessSimple: writeJSONSuccessSimple,
	WriteError: func(w http.ResponseWriter, status int, message string) {
		writeJSONError(w, status, message, ApiErrorTypeServerError)
	},
	WriteErrorType: func(w http.ResponseWriter, status int, message, errorType string) {
		writeJSONError(w, status, message, errorType)
	},
	WriteJSON: func(w http.ResponseWriter, value interface{}) error {
		return json.NewEncoder(w).Encode(value)
	},
	LogError: logStructuredError,
}

func metricsConnectDependencies(service *domainmetrics.Service) metricshttp.ConnectDependencies {
	return metricshttp.ConnectDependencies{Tracker: service, Reader: service}
}
