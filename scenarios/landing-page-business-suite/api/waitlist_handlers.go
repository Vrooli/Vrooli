package main

import (
	"net/http"

	metricshttp "landing-page-business-suite-api/handlers/metrics"
)

var waitlistHandlerDependencies = metricshttp.Dependencies{
	DecodeJSON:         decodeJSONBody,
	ValidateEmail:      ValidateEmailForHandler,
	PathInt64:          getPathParamInt64,
	WriteSuccess:       writeJSONSuccess,
	WriteSuccessData:   writeJSONSuccessData,
	WriteSuccessSimple: writeJSONSuccessSimple,
	WriteError: func(w http.ResponseWriter, status int, message string) {
		writeJSONError(w, status, message, ApiErrorTypeServerError)
	},
	LogError: logStructuredError,
}

func handleWaitlistCreate(svc WaitlistServicer) http.HandlerFunc {
	return waitlistHandlerDependencies.CreateWaitlist(svc)
}

func handleWaitlistList(svc WaitlistServicer) http.HandlerFunc {
	return waitlistHandlerDependencies.ListWaitlist(svc)
}

func handleWaitlistDelete(svc WaitlistServicer) http.HandlerFunc {
	return waitlistHandlerDependencies.DeleteWaitlist(svc)
}

func handleWaitlistExport(svc WaitlistServicer) http.HandlerFunc {
	return waitlistHandlerDependencies.ExportWaitlist(svc)
}
