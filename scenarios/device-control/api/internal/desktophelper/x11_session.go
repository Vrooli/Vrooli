package desktophelper

import (
	"context"
	"os"

	"device-control/internal/native/x11"
)

// OpenX11Session binds a trusted bootstrap's exact local display and login
// session using the same private authority-file policy as the desktop helper.
func OpenX11Session(ctx context.Context, display uint16, authorityPath, sessionID string) (*x11.Backend, error) {
	if sessionID == "" {
		return nil, ErrBootstrap
	}
	authority, err := readPrivate(authorityPath, 1024*1024)
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, ErrBootstrap
	}
	cookie, err := xCookie(authority, display, hostname)
	if err != nil {
		return nil, err
	}
	return x11.NewForSession(ctx, display, cookie, sessionID)
}
