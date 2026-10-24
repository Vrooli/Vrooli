package runtimeapp

import "io"

func (app *Service) resolveRoot() (string, error) {
	if app.ResolveRootFn == nil {
		return "", io.ErrClosedPipe
	}
	return app.ResolveRootFn()
}
