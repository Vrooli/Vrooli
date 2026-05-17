// Package settings hosts the SettingsService Connect-RPC handler.
package settings

import (
	"log"
)

type connectHandler struct{ deps Deps }

// NewConnectHandler returns the live Connect handler. Caller is
// responsible for wiring the dependencies.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}
