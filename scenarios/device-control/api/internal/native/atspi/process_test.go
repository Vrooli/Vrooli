package atspi

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/require"
)

type applicationBus struct {
	names    []string
	foreign  string
	vanished string
	name     string
	reads    int
	cancel   context.CancelFunc
	desktop  string
}

func (b *applicationBus) call(ctx context.Context, owner string, path dbus.ObjectPath, method string, args ...any) ([]any, error) {
	switch method {
	case "org.a11y.atspi.Accessible.GetRole":
		if owner == b.desktop {
			return []any{uint32(14)}, nil
		}
		return []any{uint32(75)}, nil
	case "org.freedesktop.DBus.GetId":
		return []any{strings.Repeat("a", 32)}, nil
	case "org.freedesktop.DBus.ListNames":
		return []any{b.names}, nil
	case "org.freedesktop.DBus.GetConnectionUnixProcessID":
		return []any{uint32(42)}, nil
	case "org.freedesktop.DBus.GetConnectionUnixUser":
		if args[0] == b.foreign {
			return []any{uint32(1001)}, nil
		}
		return []any{uint32(1000)}, nil
	case "org.freedesktop.DBus.Properties.Get":
		b.reads++
		if b.cancel != nil {
			b.cancel()
		}
		if owner == b.vanished {
			return nil, ErrRefused
		}
		if path != "/org/a11y/atspi/accessible/root" || args[1] != "Name" {
			return nil, ErrRefused
		}
		return []any{dbus.MakeVariant(b.name)}, nil
	}
	return nil, fmt.Errorf("unexpected discovery call %s", method)
}

func TestApplicationsPreserveDistinctOwnersAndExcludeForeignUsers(t *testing.T) {
	b := &applicationBus{names: []string{"org.a11y.Bus", ":1.3", ":1.2", ":1.2", ":1.4", ":1.5", ":1.6"}, foreign: ":1.4", vanished: ":1.5", desktop: ":1.6", name: "Editor"}
	c := &Client{wire: b, uid: 1000, guard: func(context.Context, Ref) error { return nil }}
	apps, err := c.Applications(context.Background(), strings.Repeat("a", 32))
	require.NoError(t, err)
	require.Len(t, apps, 2, "same names and PID do not merge different bus owners")
	require.Equal(t, ":1.2", apps[0].Ref.Owner)
	require.Equal(t, ":1.3", apps[1].Ref.Owner)
	require.Equal(t, 4, b.reads, "foreign UID is refused before reading its name")
}

func TestApplicationsRefuseIncompleteOrUntrustedScan(t *testing.T) {
	for _, scenario := range []string{"wrong bus", "cancelled", "too many names", "too many applications", "too many bytes", "guard refusal"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			b := &applicationBus{names: []string{":1.2"}, name: "Editor"}
			c := &Client{wire: b, uid: 1000, guard: func(ctx context.Context, _ Ref) error { return ctx.Err() }}
			busID := strings.Repeat("a", 32)
			switch scenario {
			case "wrong bus":
				busID = strings.Repeat("b", 32)
			case "cancelled":
				b.cancel = cancel
			case "too many names":
				b.names = make([]string, 513)
			case "too many applications":
				for i := 0; i < 129; i++ {
					b.names = append(b.names, fmt.Sprintf(":2.%d", i))
				}
			case "too many bytes":
				b.name = strings.Repeat("x", 4096)
				for i := 0; i < 17; i++ {
					b.names = append(b.names, fmt.Sprintf(":2.%d", i))
				}
			case "guard refusal":
				c.guard = func(context.Context, Ref) error { return ErrRefused }
			}
			apps, err := c.Applications(ctx, busID)
			require.ErrorIs(t, err, ErrRefused)
			require.Nil(t, apps)
		})
	}
}
