package atspi

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/require"
)

type fixtureBus struct {
	text            string
	mutations       int
	invocations     int
	voidAction      bool
	failAfterEffect bool
	pid             uint32
}

func (b *fixtureBus) call(_ context.Context, owner string, path dbus.ObjectPath, method string, args ...any) ([]any, error) {
	switch method {
	case "org.freedesktop.DBus.GetId":
		return []any{strings.Repeat("a", 32)}, nil
	case "org.freedesktop.DBus.GetConnectionUnixProcessID":
		return []any{b.pid}, nil
	case "org.freedesktop.DBus.GetConnectionUnixUser":
		return []any{uint32(1000)}, nil
	}
	if owner != ":1.42" || path != "/org/a11y/atspi/accessible/1" {
		return nil, errors.New("wrong object")
	}
	switch method {
	case "org.a11y.atspi.Text.GetText":
		return []any{b.text}, nil
	case "org.a11y.atspi.EditableText.InsertText":
		pos, text, length := args[0].(int32), args[1].(string), args[2].(int32)
		if int(length) != len(text) {
			return nil, errors.New("wrong UTF-8 byte count")
		}
		runes := []rune(b.text)
		b.text = string(runes[:pos]) + text + string(runes[pos:])
		b.mutations++
		if b.failAfterEffect {
			return nil, context.DeadlineExceeded
		}
		return []any{true}, nil
	case "org.a11y.atspi.Action.GetNActions":
		return []any{int32(1)}, nil
	case "org.a11y.atspi.Action.DoAction":
		b.invocations++
		if b.voidAction {
			return nil, nil
		}
		return []any{true}, nil
	}
	return nil, errors.New("unexpected call")
}

func TestInvokeUsesExactRefAndReportsAcceptedAction(t *testing.T) {
	ref := Ref{BusID: strings.Repeat("a", 32), Owner: ":1.42", Path: "/org/a11y/atspi/accessible/1", PID: 42}
	b := &fixtureBus{pid: 42}
	c := &Client{wire: b, uid: 1000, guard: func(context.Context, Ref) error { return nil }}
	require.NoError(t, c.Invoke(context.Background(), ref))
	require.Equal(t, 1, b.invocations)
}

func TestInvokeAcceptsSuccessfulVoidActionReply(t *testing.T) {
	ref := Ref{BusID: strings.Repeat("a", 32), Owner: ":1.42", Path: "/org/a11y/atspi/accessible/1", PID: 42}
	b := &fixtureBus{pid: 42, voidAction: true}
	c := &Client{wire: b, uid: 1000, guard: func(context.Context, Ref) error { return nil }}
	require.NoError(t, c.Invoke(context.Background(), ref))
	require.Equal(t, 1, b.invocations)
}
func TestUnicodeInsertionExactRefAndObservedText(t *testing.T) {
	ref := Ref{BusID: strings.Repeat("a", 32), Owner: ":1.42", Path: "/org/a11y/atspi/accessible/1", PID: 42}
	b := &fixtureBus{text: "é|suffix", pid: 42}
	c := &Client{wire: b, uid: 1000, guard: func(context.Context, Ref) error { return nil }}
	text := "日本語 العربية e\u0301 🧪"
	require.NoError(t, c.InsertText(context.Background(), ref, b.text, 2, text))
	require.Equal(t, "é|"+text+"suffix", b.text)
	require.Equal(t, 1, b.mutations)
	require.ErrorIs(t, c.InsertText(context.Background(), ref, "old", 0, "x"), ErrRefused)
	b.pid = 43
	require.ErrorIs(t, c.InsertText(context.Background(), ref, b.text, 0, "x"), ErrRefused)
	require.Equal(t, 1, b.mutations)
}
func TestMutationFailureNeverRetriesAndGuardRechecks(t *testing.T) {
	ref := Ref{BusID: strings.Repeat("a", 32), Owner: ":1.42", Path: "/org/a11y/atspi/accessible/1", PID: 42}
	b := &fixtureBus{pid: 42, failAfterEffect: true}
	calls := 0
	c := &Client{wire: b, uid: 1000, guard: func(context.Context, Ref) error { calls++; return nil }}
	require.ErrorIs(t, c.InsertText(context.Background(), ref, "", 0, "🧪"), ErrOutcomeUnknown)
	require.Equal(t, 1, b.mutations)
	require.Equal(t, "🧪", b.text)
	require.Equal(t, 2, calls)
	c.guard = func(context.Context, Ref) error {
		calls++
		if calls%2 == 0 {
			return ErrRefused
		}
		return nil
	}
	require.ErrorIs(t, c.InsertText(context.Background(), ref, b.text, 0, "x"), ErrRefused)
	require.Equal(t, 1, b.mutations)
}

func TestInsertionChecksCallerAuthorityAfterReads(t *testing.T) {
	ref := Ref{BusID: strings.Repeat("a", 32), Owner: ":1.42", Path: "/org/a11y/atspi/accessible/1", PID: 42}
	bus := &fixtureBus{pid: 42, text: "before"}
	guards := 0
	client := &Client{wire: bus, uid: 1000, guard: func(context.Context, Ref) error { guards++; return nil }}
	err := client.InsertTextChecked(context.Background(), ref, "before", 0, "🧪", func(context.Context) error {
		require.Equal(t, 2, guards, "identity and final session checks precede caller authority")
		return ErrRefused
	})
	require.ErrorIs(t, err, ErrRefused)
	require.Zero(t, bus.mutations)
	require.Equal(t, "before", bus.text)
}
