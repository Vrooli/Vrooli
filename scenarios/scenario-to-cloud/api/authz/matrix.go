package authz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// MatrixSchemaVersion is carried on every export so consumers can refuse a
// shape they do not understand.
const MatrixSchemaVersion = "1"

// Matrix is the exported authorization matrix.
type Matrix struct {
	SchemaVersion string   `json:"schema_version"`
	Scenario      string   `json:"scenario"`
	Scopes        []string `json:"scopes"`
	Routes        []Route  `json:"routes"`
}

// ExportMatrix returns the table in export order.
func ExportMatrix() Matrix {
	return Matrix{SchemaVersion: MatrixSchemaVersion, Scenario: Scenario, Scopes: Scopes(), Routes: Sorted()}
}

// MatrixHandler serves GET /api/v1/authz/matrix.
func MatrixHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ExportMatrix())
	}
}

// RenderMarkdown renders the matrix document checked into
// docs/reference/authorization-matrix.md. A test asserts the file matches.
func RenderMarkdown() string {
	var b strings.Builder
	b.WriteString("# Authorization Matrix\n\n")
	b.WriteString("> Generated from `api/authz/routes.go` (`authz.RenderMarkdown`). Do not edit by hand; `go test ./authz/ -run TestMatrixDocMatchesTable` fails when this file drifts.\n\n")
	b.WriteString("Every management route of the scenario-to-cloud API is classified once. The enforcement middleware (`api/authz/middleware.go`) denies by default: a route absent from this table is refused, a public route must be on the explicit allowlist, and every other route needs a verified principal holding the listed scope.\n\n")
	b.WriteString("Scopes are the shared coarse vocabulary (`" + strings.Join(Scopes(), "`, `") + "`). Effect classes decide the extra boundaries:\n\n")
	b.WriteString("| Effect | Meaning | Extra boundary |\n|---|---|---|\n")
	b.WriteString("| `public` | Liveness metadata only | None (allowlist: `/health`, `/api/v1/health`) |\n")
	b.WriteString("| `read` | Observe local or remote state | Scope only; service principals allowed where marked |\n")
	b.WriteString("| `workload_mutation` | Change deployment records or run lifecycle steps | Human only, origin check, per-principal concurrency, effect recheck |\n")
	b.WriteString("| `secret` | Credential material or its metadata | Human only, `Cache-Control: no-store`, effect recheck |\n")
	b.WriteString("| `host` | Repair or reconfigure a host | Human only, origin check, effect recheck |\n")
	b.WriteString("| `interactive` | Long-lived session on a target | Human only, browser Origin required, socket bound to credential expiry |\n\n")
	b.WriteString("Columns: **target** = `id` (deployment resolved and policy-checked before the handler), `body` (target named in the request body; only unrestricted principals), or `-`; **reach** = the route opens SSH or another reach to a target; **service** = service principals and verified agents may call it; **no-store** = response is never cacheable.\n\n")
	b.WriteString("| Method | Path | Effect | Scope | Target | Reach | Service | No-store | Description |\n|---|---|---|---|---|---|---|---|---|\n")
	for _, route := range Sorted() {
		target := "-"
		switch {
		case route.TargetBound:
			target = "id"
		case route.BodyTarget:
			target = "body"
		}
		path := route.Path
		if route.Prefix {
			path += "*"
		}
		scope := route.Scope
		if scope == "" {
			scope = "-"
		}
		fmt.Fprintf(&b, "| %s | `%s` | %s | `%s` | %s | %s | %s | %s | %s |\n",
			route.Method, path, route.Effect, scope, target, yesNo(route.RemoteReach), yesNo(route.ServiceAllowed), yesNo(route.NoStore), route.Description)
	}
	b.WriteString("\nThe same table is served at `GET /api/v1/authz/matrix` (read scope) for UI and CLI parity.\n")
	return b.String()
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
