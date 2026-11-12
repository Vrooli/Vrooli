// Package health provides the /health endpoint on api-core/health's
// standard envelope. The only check is whether the research home is
// readable; it is optional, so a missing home reports degraded rather than
// unhealthy (the desk still serves its manual paths and says why).
package healthapi

import (
	"context"
	"net/http"
	"time"

	apihealth "github.com/vrooli/api-core/health"

	"research-desk/internal/wire"
)

// Deps wires the health handler. ResearchHome reports whether the research
// home can be read; nil means it can.
type Deps struct {
	ResearchHome func(context.Context) error
	// WebConsoleAccess says how automatic launch reaches Web Console;
	// nil omits it.
	WebConsoleAccess func(context.Context) wire.Access
	Service          string
	Version          string
}

// accessTimeout bounds the Web Console lookup a health probe makes.
const accessTimeout = 2 * time.Second

// NewHandler returns the /health handler.
func NewHandler(d Deps) http.HandlerFunc {
	b := apihealth.New(d.Service).
		Version(d.Version).
		Check(apihealth.Func("research-home", d.ResearchHome), apihealth.Optional)
	if d.WebConsoleAccess != nil {
		b = b.Metric("web_console_access", func(time.Time) any {
			ctx, cancel := context.WithTimeout(context.Background(), accessTimeout)
			defer cancel()
			a := d.WebConsoleAccess(ctx)
			return map[string]string{"kind": a.Kind.String(), "said": a.Said}
		})
	}
	return b.Handler()
}
