package main

import (
	"net/http"

	management "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/logx"
)

func userManagementDependencies(service management.UserManagementService) management.UserManagementDependencies {
	return management.UserManagementDependencies{Service: service, Path: getPathParam, WriteJSON: writeJSONSuccessData, WriteError: func(w http.ResponseWriter, status int, message string) { http.Error(w, message, status) }, Log: logx.Info, LogError: logx.Error}
}
