//go:build linux

package atspi

import (
	"bufio"
	"context"
	"github.com/godbus/dbus/v5"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrivateBusPeerAuthentication(t *testing.T) {
	executable, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("D-Bus fixture unavailable")
	}
	path := filepath.Join(t.TempDir(), "bus")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "--session", "--nofork", "--address=unix:path="+path, "--print-address=1")
	read, write, err := os.Pipe()
	require.NoError(t, err)
	cmd.Stdout = write
	require.NoError(t, cmd.Start())
	write.Close()
	t.Cleanup(func() { cancel(); _ = cmd.Wait(); read.Close() })
	require.NoError(t, read.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, err = bufio.NewReader(read).ReadString('\n')
	require.NoError(t, err)
	called := false
	conn, err := Dial(ctx, path, func(_ context.Context, pid, uid uint32) error {
		called = true
		require.EqualValues(t, cmd.Process.Pid, pid)
		require.EqualValues(t, os.Getuid(), uid)
		return nil
	})
	require.NoError(t, err)
	defer conn.Close()
	require.True(t, called)
	var busID string
	require.NoError(t, conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetId", 0).Store(&busID))
	require.Len(t, busID, 32)
	bound, err := DialBound(ctx, path, busID, func(context.Context, uint32, uint32) error { return nil })
	require.NoError(t, err)
	bound.Close()
	_, err = DialBound(ctx, path, strings.Repeat("0", 32), func(context.Context, uint32, uint32) error { return nil })
	require.ErrorIs(t, err, ErrRefused)

	fixture := &editableFixture{value: "é|tail"}
	objectPath := dbus.ObjectPath("/org/a11y/atspi/accessible/1")
	require.NoError(t, conn.Export(fixture, objectPath, "org.a11y.atspi.Text"))
	require.NoError(t, conn.Export(fixture, objectPath, "org.a11y.atspi.EditableText"))
	client, err := New(conn, func(context.Context, Ref) error { return nil })
	require.NoError(t, err)
	ref := Ref{BusID: busID, Owner: conn.Names()[0], Path: objectPath, PID: uint32(os.Getpid())}
	observed, err := client.ReadText(ctx, ref)
	require.NoError(t, err)
	require.NoError(t, client.InsertText(ctx, ref, observed, 2, "日本語 العربية 🧪"))
	result, err := client.ReadText(ctx, ref)
	require.NoError(t, err)
	require.Equal(t, "é|日本語 العربية 🧪tail", result)
	_, err = Dial(ctx, path, func(context.Context, uint32, uint32) error { return ErrRefused })
	require.ErrorIs(t, err, ErrRefused)
}

func TestBusHandshakeTimeoutAndNoAmbientFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "silent")
	listener, err := net.Listen("unix", path)
	require.NoError(t, err)
	defer listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = Dial(ctx, path, func(context.Context, uint32, uint32) error { return nil })
	require.ErrorIs(t, err, ErrRefused)
	require.Less(t, time.Since(start), time.Second)
	for _, invalid := range []string{"", "relative", "unix:path=/tmp/bus", "/tmp/../bus"} {
		_, err := Dial(context.Background(), invalid, func(context.Context, uint32, uint32) error { t.Fatal("invalid path reached peer guard"); return nil })
		require.ErrorIs(t, err, ErrRefused)
	}
}

// This exported protocol fixture exercises real D-Bus marshaling and replies;
// the separate GTK spike remains the application-level mechanism evidence.
type editableFixture struct {
	mu    sync.Mutex
	value string
}

func (f *editableFixture) GetText(start, end int32) (string, *dbus.Error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if start != 0 || end != -1 {
		return "", dbus.NewError("org.a11y.Error", nil)
	}
	return f.value, nil
}
func (f *editableFixture) InsertText(position int32, text string, length int32) (bool, *dbus.Error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	runes := []rune(f.value)
	if position < 0 || int(position) > len(runes) || int(length) != len(text) {
		return false, nil
	}
	f.value = string(runes[:position]) + text + string(runes[position:])
	return true, nil
}
