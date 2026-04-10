package cliutil

import "net/http"

// DryRunHeader is the canonical header used to signal dry-run mode.
const DryRunHeader = "X-Dry-Run"

// IsDryRun reports whether the request has the dry-run header set.
func IsDryRun(r *http.Request) bool {
	return r.Header.Get(DryRunHeader) == "true"
}
