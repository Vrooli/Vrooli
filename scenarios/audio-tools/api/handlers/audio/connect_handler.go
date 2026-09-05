package audio

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. Deps.Logger is required
// (logx.Logger); a nil value panics so a forgotten wire-up surfaces at
// boot, not at request-time.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		panic("audio.NewConnectHandler requires Deps.Logger")
	}
	return &connectHandler{deps: d}
}
