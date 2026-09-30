// Package atspi implements exact-object accessibility operations. The caller
// owns protected bus connection provisioning and desktop lease admission.
package atspi

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/godbus/dbus/v5"
)

var ErrRefused = errors.New("accessibility operation refused")
var ErrOutcomeUnknown = errors.New("accessibility mutation outcome unknown")

type Ref struct {
	BusID string
	Owner string
	Path  dbus.ObjectPath
	PID   uint32
}

// Guard revalidates the bound desktop session and its lease before each
// operation, including immediately before a mutation. It must not be nil.
type Guard func(context.Context, Ref) error
type caller interface {
	call(context.Context, string, dbus.ObjectPath, string, ...any) ([]any, error)
}
type Client struct {
	wire  caller
	guard Guard
	uid   uint32
}
type busCaller struct{ conn *dbus.Conn }

func (b busCaller) call(ctx context.Context, owner string, path dbus.ObjectPath, method string, args ...any) ([]any, error) {
	c := b.conn.Object(owner, path).CallWithContext(ctx, method, dbus.FlagNoAutoStart, args...)
	return c.Body, c.Err
}

// New uses an already-authenticated private accessibility bus connection.
// It does not discover or autostart a bus from ambient environment variables.
func New(conn *dbus.Conn, guard Guard) (*Client, error) {
	if conn == nil || guard == nil {
		return nil, ErrRefused
	}
	return &Client{wire: busCaller{conn}, guard: guard, uid: uint32(os.Getuid())}, nil
}
func (c *Client) identity(ctx context.Context, ref Ref) error {
	if c.guard == nil || len(ref.BusID) != 32 || !strings.HasPrefix(ref.Owner, ":") || len(ref.Owner) > 255 || !ref.Path.IsValid() || !strings.HasPrefix(string(ref.Path), "/org/a11y/atspi/accessible/") || ref.PID == 0 {
		return ErrRefused
	}
	if c.guard(ctx, ref) != nil {
		return ErrRefused
	}
	var busID string
	var pid, uid uint32
	for _, query := range []struct {
		method string
		args   []any
		result any
	}{
		{"GetId", nil, &busID}, {"GetConnectionUnixProcessID", []any{ref.Owner}, &pid}, {"GetConnectionUnixUser", []any{ref.Owner}, &uid},
	} {
		body, err := c.wire.call(ctx, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus."+query.method, query.args...)
		if err != nil || dbus.Store(body, query.result) != nil {
			return ErrRefused
		}
	}
	if busID != ref.BusID || pid != ref.PID || uid != c.uid {
		return ErrRefused
	}
	return nil
}
func (c *Client) text(ctx context.Context, ref Ref) (string, error) {
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.Text.GetText", int32(0), int32(-1))
	var value string
	if err != nil || dbus.Store(body, &value) != nil || !validText(value) {
		return "", ErrRefused
	}
	return value, nil
}
func validText(text string) bool {
	return len(text) <= 16*1024 && utf8.ValidString(text) && !strings.ContainsRune(text, 0)
}
func (c *Client) ReadText(ctx context.Context, ref Ref) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if c.identity(ctx, ref) != nil {
		return "", ErrRefused
	}
	return c.text(ctx, ref)
}

// InsertText inserts at a Unicode character offset, preserving surrounding
// observed text. It never retries a mutation, including after timeout.
func (c *Client) InsertText(ctx context.Context, ref Ref, observed string, position int32, text string) error {
	return c.InsertTextChecked(ctx, ref, observed, position, text, func(context.Context) error { return nil })
}

// InsertTextChecked adds caller authority at the final boundary after reads.
func (c *Client) InsertTextChecked(ctx context.Context, ref Ref, observed string, position int32, text string, check func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if check == nil || text == "" || !validText(text) || !validText(observed) || len(text)+len(observed) > 16*1024 || position < 0 {
		return ErrRefused
	}
	runes := []rune(observed)
	if int64(position) > int64(len(runes)) || c.identity(ctx, ref) != nil {
		return ErrRefused
	}
	current, err := c.text(ctx, ref)
	if err != nil || current != observed || c.guard(ctx, ref) != nil {
		return ErrRefused
	}
	if check(ctx) != nil {
		return ErrRefused
	}
	var accepted bool
	body, err := c.wire.call(ctx, ref.Owner, ref.Path, "org.a11y.atspi.EditableText.InsertText", position, text, int32(len(text)))
	if err != nil || dbus.Store(body, &accepted) != nil || !accepted {
		return ErrOutcomeUnknown
	}
	current, err = c.text(ctx, ref)
	want := string(runes[:position]) + text + string(runes[position:])
	if err != nil || current != want {
		return ErrOutcomeUnknown
	}
	return nil
}
