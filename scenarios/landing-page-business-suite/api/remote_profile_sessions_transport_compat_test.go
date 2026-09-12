package main

import (
	"net/http"

	remoteprofilehttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
)

type (
	IncomingRemoteProfileSessionResponse = administration.IncomingRemoteProfileSession
	RemoteProfileSessionStore            = administration.IncomingRemoteProfileSessionStore
)

func handleAdminListIncomingRemoteProfileSessions(store RemoteProfileSessionStore) http.HandlerFunc {
	return remoteprofilehttp.ListIncomingRemoteProfileSessions(remoteProfileSessionDependencies(store))
}

func handleAdminRevokeIncomingRemoteProfileSession(store RemoteProfileSessionStore) http.HandlerFunc {
	return remoteprofilehttp.RevokeIncomingRemoteProfileSession(remoteProfileSessionDependencies(store))
}
