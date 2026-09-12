//go:build !linux

package atspi

import (
	"context"
	"github.com/godbus/dbus/v5"
)

func Dial(context.Context, string, func(context.Context, uint32, uint32) error) (*dbus.Conn, error) {
	return nil, ErrRefused
}
