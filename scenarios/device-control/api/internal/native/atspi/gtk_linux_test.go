//go:build linux

package atspi

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/require"
)

func TestGoClientInsertsUnicodeIntoGTK(t *testing.T) {
	for _, binary := range []string{"dbus-run-session", "xvfb-run", "/usr/bin/python3"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skip("GTK fixture tool unavailable: " + binary)
		}
	}
	probe := exec.Command("/usr/bin/python3", "-c", "import gi;gi.require_version('Gtk','3.0');from gi.repository import Gtk")
	if probe.Run() != nil {
		t.Skip("GTK Python fixture unavailable")
	}
	directory, err := os.MkdirTemp("", "a11y-")
	require.NoError(t, err)
	defer os.RemoveAll(directory)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resultPath := filepath.Join(directory, "result")
	command := exec.Command("dbus-run-session", "--", "xvfb-run", "-a", "/usr/bin/python3", "testdata/gtk_fixture.py", resultPath)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "XDG_RUNTIME_DIR=") && !strings.HasPrefix(value, "GSETTINGS_BACKEND=") && !strings.HasPrefix(value, "GIO_USE_VFS=") && !strings.HasPrefix(value, "NO_AT_BRIDGE=") {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "XDG_RUNTIME_DIR="+directory, "GSETTINGS_BACKEND=memory", "GIO_USE_VFS=local")
	read, write, err := os.Pipe()
	require.NoError(t, err)
	command.Stdout = write
	require.NoError(t, command.Start())
	write.Close()
	stop := context.AfterFunc(ctx, func() { _ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL) })
	defer func() {
		stop()
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		_ = command.Wait()
		read.Close()
	}()
	require.NoError(t, read.SetReadDeadline(time.Now().Add(5*time.Second)))
	var ready struct {
		Address string
		PID     uint32
	}
	scanner := bufio.NewScanner(read)
	for scanner.Scan() {
		if json.Unmarshal(scanner.Bytes(), &ready) == nil && ready.Address != "" {
			break
		}
	}
	require.NotEmpty(t, ready.Address)
	address := strings.Split(ready.Address, ",")[0]
	require.True(t, strings.HasPrefix(address, "unix:path="))
	path := strings.TrimPrefix(address, "unix:path=")
	require.True(t, strings.HasPrefix(path, directory+"/"))
	conn, err := Dial(ctx, path, func(_ context.Context, pid, uid uint32) error {
		if pid == 0 || uid != uint32(os.Getuid()) {
			return ErrRefused
		}
		return nil
	})
	require.NoError(t, err)
	defer conn.Close()
	var busID string
	require.NoError(t, conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetId", 0).Store(&busID))
	var names []string
	require.NoError(t, conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names))
	owner := ""
	for _, name := range names {
		if !strings.HasPrefix(name, ":") {
			continue
		}
		var pid uint32
		if conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetConnectionUnixProcessID", 0, name).Store(&pid) == nil && pid == ready.PID {
			owner = name
			break
		}
	}
	require.NotEmpty(t, owner)
	client, err := New(conn, func(_ context.Context, ref Ref) error {
		if ref.BusID != busID {
			return ErrRefused
		}
		return nil
	})
	require.NoError(t, err)
	queue := []Ref{{BusID: busID, Owner: owner, Path: dbus.ObjectPath("/org/a11y/atspi/accessible/root"), PID: ready.PID}}
	var target Ref
	for visited := 0; len(queue) > 0 && visited < 32; visited++ {
		ref := queue[0]
		queue = queue[1:]
		name, err := client.Name(ctx, ref)
		require.NoError(t, err)
		if name == "go-unicode-entry" {
			target = ref
			break
		}
		children, err := client.Children(ctx, ref)
		require.NoError(t, err)
		queue = append(queue, children...)
	}
	require.NotEmpty(t, target.Path)
	root, err := client.ProcessRoot(ctx, busID, ready.PID)
	require.NoError(t, err)
	require.Equal(t, owner, root.Owner)
	applications, err := client.Applications(ctx, busID)
	require.NoError(t, err)
	require.Len(t, applications, 1)
	require.Equal(t, root, applications[0].Ref)
	require.Equal(t, "gtk_fixture.py", applications[0].Name)
	editable, err := client.Editable(ctx, target)
	require.NoError(t, err)
	require.True(t, editable)
	original, err := client.ReadText(ctx, target)
	require.NoError(t, err)
	require.Equal(t, "é|tail", original)
	text := "日本語 العربية café e\u0301 🧪"
	require.NoError(t, client.InsertText(ctx, target, original, 2, text))
	actual, err := os.ReadFile(resultPath)
	require.NoError(t, err)
	require.Equal(t, "é|"+text+"tail", string(actual))
}
