package atspi

import (
	"context"
	"time"

	"github.com/godbus/dbus/v5"
)

// Children stays within one exact application owner. Embedded foreign-process
// objects require separate admission; they are not silently followed.
func (c *Client) Children(ctx context.Context, ref Ref) ([]Ref, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return nil, ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetChildren")
	var children []struct {
		Owner string
		Path  dbus.ObjectPath
	}
	if err != nil || dbus.Store(body, &children) != nil || len(children) > 128 {
		return nil, ErrRefused
	}
	result := make([]Ref, 0, len(children))
	for _, child := range children {
		if child.Owner != ref.Owner || !child.Path.IsValid() {
			return nil, ErrRefused
		}
		next := ref
		next.Path = child.Path
		result = append(result, next)
	}
	return result, nil
}

func (c *Client) Name(ctx context.Context, ref Ref) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return "", ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.freedesktop.DBus.Properties.Get", "org.a11y.atspi.Accessible", "Name")
	var value dbus.Variant
	if err != nil || dbus.Store(body, &value) != nil {
		return "", ErrRefused
	}
	name, ok := value.Value().(string)
	if !ok || len(name) > 4096 {
		return "", ErrRefused
	}
	return name, nil
}

// Role returns the native semantic role after revalidating exact identity.
func (c *Client) Role(ctx context.Context, ref Ref) (uint32, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return 0, ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetRole")
	var role uint32
	if err != nil || dbus.Store(body, &role) != nil {
		return 0, ErrRefused
	}
	return role, nil
}

// Invoke activates the first native action exposed by an exact accessible
// element. The identity and action count are rechecked immediately before the
// mutation; a transport failure is reported as an unknown outcome so callers
// do not blindly retry a possibly delivered activation.
func (c *Client) Invoke(ctx context.Context, ref Ref) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Action.GetNActions")
	var count int32
	if err != nil {
		return ErrRefused
	}
	if dbus.Store(body, &count) != nil {
		var unsigned uint32
		if dbus.Store(body, &unsigned) != nil || unsigned > 64 {
			return ErrRefused
		}
		count = int32(unsigned)
	}
	if count <= 0 || count > 64 {
		return ErrRefused
	}
	if c.guard(ctx, ref) != nil {
		return ErrRefused
	}
	body, err = c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Action.DoAction", int32(0))
	if err != nil {
		return ErrOutcomeUnknown
	}
	// A few AT-SPI bridges expose DoAction as a void method even though the
	// canonical interface returns a boolean. A successful, empty reply still
	// confirms that the request crossed the native boundary.
	if len(body) == 0 {
		return nil
	}
	var accepted bool
	if dbus.Store(body, &accepted) != nil || !accepted {
		return ErrOutcomeUnknown
	}
	return nil
}
