package atspi

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// Application identifies one accessible root on the explicitly bound bus.
// Names are presentation only: callers must retain Ref rather than resolve a
// later selection by name or PID, either of which can be reused.
type Application struct {
	Ref  Ref
	Name string
}

// Applications reads root names, never field contents. Non-accessible bus
// clients and other users are omitted. A bounded scan fails rather than
// presenting an apparently complete list after cancellation or overflow.
func (c *Client) Applications(ctx context.Context, busID string) ([]Application, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.guard == nil || len(busID) != 32 || ctx.Err() != nil {
		return nil, ErrRefused
	}
	body, err := c.wire.call(ctx, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.GetId")
	var actualID string
	if err != nil || dbus.Store(body, &actualID) != nil || actualID != busID {
		return nil, ErrRefused
	}
	body, err = c.wire.call(ctx, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.ListNames")
	var names []string
	if err != nil || dbus.Store(body, &names) != nil || len(names) > 512 {
		return nil, ErrRefused
	}
	result := make([]Application, 0)
	seen := make(map[string]bool)
	bytes := 0
	for _, owner := range names {
		if ctx.Err() != nil {
			return nil, ErrRefused
		}
		if !strings.HasPrefix(owner, ":") || seen[owner] {
			continue
		}
		seen[owner] = true
		body, err := c.wire.call(ctx, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.GetConnectionUnixProcessID", owner)
		var pid uint32
		if err != nil || dbus.Store(body, &pid) != nil || pid == 0 {
			continue
		}
		ref := Ref{BusID: busID, Owner: owner, Path: "/org/a11y/atspi/accessible/root", PID: pid}
		if c.guard(ctx, ref) != nil {
			return nil, ErrRefused
		}
		name, err := c.Name(ctx, ref)
		if err != nil {
			continue
		}
		body, err = c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetRole")
		var role uint32
		// ATSPI_ROLE_APPLICATION. The registry's desktop root is not an app.
		if err != nil || dbus.Store(body, &role) != nil || role != 75 {
			continue
		}
		if c.identity(ctx, ref) != nil {
			return nil, ErrRefused
		}
		bytes += len(name)
		if len(result) >= 128 || bytes > 64*1024 {
			return nil, ErrRefused
		}
		result = append(result, Application{Ref: ref, Name: name})
	}
	if ctx.Err() != nil {
		return nil, ErrRefused
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].Ref.Owner < result[j].Ref.Owner
	})
	return result, nil
}

func (c *Client) ProcessRoot(ctx context.Context, busID string, pid uint32) (Ref, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	body, err := c.wire.call(ctx, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.ListNames")
	var names []string
	if pid == 0 || err != nil || dbus.Store(body, &names) != nil || len(names) > 512 {
		return Ref{}, ErrRefused
	}
	var found Ref
	for _, name := range names {
		if !strings.HasPrefix(name, ":") {
			continue
		}
		body, err := c.wire.call(ctx, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus.GetConnectionUnixProcessID", name)
		var candidate uint32
		if err != nil || dbus.Store(body, &candidate) != nil || candidate != pid {
			continue
		}
		ref := Ref{BusID: busID, Owner: name, Path: "/org/a11y/atspi/accessible/root", PID: pid}
		if _, err := c.Name(ctx, ref); err != nil {
			continue
		}
		if found.Owner != "" {
			return Ref{}, ErrRefused
		}
		found = ref
	}
	if found.Owner == "" {
		return Ref{}, ErrRefused
	}
	return found, nil
}

func (c *Client) Editable(ctx context.Context, ref Ref) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return false, ErrRefused
	}
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetRole")
	var role uint32
	if err != nil || dbus.Store(body, &role) != nil {
		return false, ErrRefused
	}
	// ATSPI_ROLE_PASSWORD_TEXT. Never cache or expose password contents.
	if role == 40 {
		return false, nil
	}
	body, err = c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Accessible.GetInterfaces")
	var interfaces []string
	if err != nil || dbus.Store(body, &interfaces) != nil || len(interfaces) > 64 {
		return false, ErrRefused
	}
	for _, name := range interfaces {
		if name == "org.a11y.atspi.EditableText" {
			return true, nil
		}
	}
	return false, nil
}
