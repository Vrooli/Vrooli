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
)

// Deps wires the health handler. ResearchHome reports whether the research
// home can be read; nil means it can.
type Deps struct {
	ResearchHome func(context.Context) error
	// WebConsoleAccess says how automatic launch reaches Web Console
	// (rulings F14); nil omits it.
	WebConsoleAccess func(context.Context) (bool, string)
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
			_, said := d.WebConsoleAccess(ctx)
			return said
		})
	}
	return b.Handler()
}
