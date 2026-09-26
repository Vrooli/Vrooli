package main

import (
	"net/http"

	remoteprofilehttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/logx"
)

// remoteProfileHandlerDependencies is the sole adapter between the root server
// and remote-profile HTTP transport. Domain types stay in administration.
func remoteProfileHandlerDependencies(service remoteprofilehttp.RemoteProfileManager, resolveEmail func(*http.Request) (string, bool)) remoteprofilehttp.RemoteProfileDependencies {
	return remoteprofilehttp.RemoteProfileDependencies{
		Service:       service,
		ResolveEmail:  resolveEmail,
		DecodeJSON:    decodeJSONBody,
		PathInt64:     getPathParamInt64,
		ValidateEmail: ValidateEmailForHandler,
		WriteData:     writeJSONSuccessData,
		WriteSimple:   writeJSONSuccessSimple,
		WriteError:    writeJSONError,
		LogError:      logx.Error,
	}
}
