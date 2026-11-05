package testutil

import (
	"net/http"

	"connectrpc.com/connect"
)

// ConnectClient is test-only because transport clients must remain outside
// deployment-manager's internal domain packages.
func ConnectClient() connect.HTTPClient { return http.DefaultClient }
