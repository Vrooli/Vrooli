package atspi

import (
	"context"
	"encoding/hex"
	"github.com/godbus/dbus/v5"
	"time"
)

// DialBound verifies the daemon's immutable bus identity against protected
// bootstrap. A daemon replacement at the same path requires explicit rebinding.
func DialBound(ctx context.Context, path, busID string, peerGuard func(context.Context, uint32, uint32) error) (*dbus.Conn, error) {
	id, err := hex.DecodeString(busID)
	if err != nil || len(id) != 16 {
		return nil, ErrRefused
	}
	conn, err := Dial(ctx, path, peerGuard)
	if err != nil {
		return nil, err
	}
	check, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var actual string
	if conn.BusObject().CallWithContext(check, "org.freedesktop.DBus.GetId", dbus.FlagNoAutoStart).Store(&actual) != nil || actual != busID {
		conn.Close()
		return nil, ErrRefused
	}
	return conn, nil
}
