package privsep

import (
	"context"
	"path/filepath"
	"strings"
)

// The helper runs as root so that the runner cannot forge or widen a
// provisioning request. The work itself belongs to the account that owns the
// checkout: git, `vrooli setup`, and scenario restarts run as that account, so
// a provision never leaves root-owned files in a user's checkout or runtime
// home (2026-09-15: a root-run break-glass step left ~/.vrooli/identity owned
// by root and the node's own Bridge API could no longer read its key).

// principalRunner is implemented by step runners that can run a step as the
// checkout owner. Test runners that do not implement it run steps directly.
type principalRunner interface {
	RunAs(ctx context.Context, argv []string, dir string, p *principal, onLog func(string)) (int, error)
}

// ownerEnv replaces the identity and search path in base with the owner's, so
// a step run on the owner's behalf finds the owner's toolchains (the helper's
// own service environment has a minimal PATH).
func ownerEnv(home, name string, base []string) []string {
	out := make([]string, 0, len(base)+4)
	for _, kv := range base {
		key, _, _ := strings.Cut(kv, "=")
		switch key {
		case "HOME", "USER", "LOGNAME", "PATH":
			continue
		}
		out = append(out, kv)
	}
	path := []string{
		filepath.Join(home, ".vrooli", "bin"),
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "go", "bin"),
		"/usr/local/go/bin", "/opt/homebrew/bin", "/usr/local/bin",
		"/usr/bin", "/bin", "/usr/sbin", "/sbin",
	}
	return append(out, "HOME="+home, "USER="+name, "LOGNAME="+name, "PATH="+strings.Join(path, ":"))
}
