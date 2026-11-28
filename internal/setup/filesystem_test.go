package setup

import (
	"errors"
	"os"
	"testing"
)

func TestPermissionErrorPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		path string
		ok   bool
	}{
		{name: "typed permission", err: &os.PathError{Op: "mkdir", Path: "/home/user/.vrooli", Err: os.ErrPermission}, path: "/home/user/.vrooli", ok: true},
		{name: "permission text in path error", err: &os.PathError{Op: "mkdir", Path: "/home/user/.vrooli", Err: errors.New("permission denied")}, path: "/home/user/.vrooli", ok: true},
		{name: "unrelated path error", err: &os.PathError{Op: "stat", Path: "/tmp/file", Err: os.ErrNotExist}},
		{name: "untyped permission text has no repair path", err: errors.New("permission denied")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, ok := permissionErrorPath(tc.err)
			if path != tc.path || ok != tc.ok {
				t.Fatalf("permissionErrorPath(%v) = (%q, %t), want (%q, %t)", tc.err, path, ok, tc.path, tc.ok)
			}
		})
	}
}
