package main

import (
	"net/http"

	remoteprofilehttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
)

func remoteProfileSessionDependencies(store administration.IncomingRemoteProfileSessionStore) remoteprofilehttp.RemoteProfileSessionDependencies {
	return remoteprofilehttp.RemoteProfileSessionDependencies{Service: administration.NewIncomingRemoteProfileSessionRepository(store), Path: getPathParam, WriteData: writeJSONSuccessData, WriteSimple: writeJSONSuccessSimple, WriteError: writeJSONError}
}

var _ http.Handler = http.HandlerFunc(nil)
