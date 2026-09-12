// Package settings hosts the SettingsService Connect-RPC handler.
package settings

type connectHandler struct{ deps Deps }

// NewConnectHandler returns the live Connect handler. Deps.Logger is
// required; a nil value panics.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("settings.NewConnectHandler requires Deps.Logger")
	}
	return &connectHandler{deps: d}
}
