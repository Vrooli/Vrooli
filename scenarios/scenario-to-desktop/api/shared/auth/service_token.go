// Package auth resolves short-lived service credentials used for owner-to-owner
// control-plane calls. The token is never included in logs or transport data
// outside the Authorization header.
package auth

import (
	"os"
	"strings"
)

// DeploymentManagerServiceToken returns the explicitly provisioned
// scenario-to-desktop service credential for Deployment Manager. A token file
// is preferred so the credential does not have to appear in the process
// environment; the direct variable remains useful for managed runtimes that
// inject short-lived credentials through the environment.
func DeploymentManagerServiceToken() string {
	if token := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN")); token != "" {
		return token
	}
	path := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN_FILE"))
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
