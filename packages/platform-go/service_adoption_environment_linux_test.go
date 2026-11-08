//go:build linux

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the real busctl argv/environment boundary without contacting a
// manager or moving any process. A bounded native worker deliberately has no
// login-session environment; a stale bus from another uid is equally invalid.
func TestServiceAdoptionUsesOwnUserManagerEnvironment(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(fmt.Sprint(stale), func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("PATH", dir)
			t.Setenv("XDG_RUNTIME_DIR", "")
			t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
			if stale {
				t.Setenv("XDG_RUNTIME_DIR", "/run/user/not-this-uid")
				t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/run/user/not-this-uid/bus")
			}
			t.Setenv("VROOLI_AGENT_SESSION_ID", "must-remain-bound")
			want := map[string]string{}
			for _, e := range userManagerEnv(nil) {
				k, v, _ := strings.Cut(e, "=")
				want[k] = v
			}
			script := fmt.Sprintf("#!/bin/sh\n[ \"$XDG_RUNTIME_DIR\" = '%s' ] && [ \"$DBUS_SESSION_BUS_ADDRESS\" = '%s' ] || { echo wrong-manager-environment >&2; exit 23; }\n[ \"$VROOLI_AGENT_SESSION_ID\" = must-remain-bound ] || exit 24\n[ \"$1\" = --user ] && [ \"$6\" = StartTransientUnit ] || exit 25\n", want["XDG_RUNTIME_DIR"], want["DBUS_SESSION_BUS_ADDRESS"])
			if err := os.WriteFile(filepath.Join(dir, "busctl"), []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "systemctl"), []byte("#!/bin/sh\nprintf '/fixture/vrooli-services.slice/fixture-service.scope\\n'\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			ref, err := adoptThroughManager(AdoptSpec{PID: 123, Scope: "fixture-service", Slice: ServicesSlice, Containment: Containment{ManagedOOMAvoid: true}}, "fixture-service.scope")
			if err != nil {
				t.Fatalf("service handoff failed: %v", err)
			}
			if ref.Path != "/fixture/vrooli-services.slice/fixture-service.scope" {
				t.Fatalf("unexpected scope: %+v", ref)
			}
			if os.Getenv("XDG_RUNTIME_DIR") != map[bool]string{true: "/run/user/not-this-uid", false: ""}[stale] {
				t.Fatal("manager client changed caller environment")
			}
		})
	}
}

func TestServiceAdoptionRetainsManagerAndFallbackFailures(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	for name, body := range map[string]string{"busctl": "echo manager-denied >&2; exit 17", "systemctl": "echo /fixture"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	prior := cgroupMount
	cgroupMount = t.TempDir()
	t.Cleanup(func() { cgroupMount = prior })
	_, _, err := AdoptIntoScope(AdoptSpec{PID: 123, Scope: "fixture-service", Slice: ServicesSlice, Containment: Containment{ManagedOOMAvoid: true}})
	if err == nil || !strings.Contains(err.Error(), "manager-denied") || !strings.Contains(err.Error(), "managed OOM preference requires") {
		t.Fatalf("lost adoption failure: %v", err)
	}
}
