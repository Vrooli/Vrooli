package nativeisolation

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"strings"
	"testing"
	"time"
)

const hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func fixture() (Manifest, Request) {
	now := time.Now()
	m := Manifest{Enabled: true, Backend: "linux-systemd-service-v1", SystemdRun: Executable{"/usr/bin/systemd-run", hash}, Systemctl: Executable{"/usr/bin/systemctl", hash}, RootFS: "/var/lib/vrooli/finite-auth02/image", WorkspaceSource: "/var/lib/vrooli/finite-auth02/worktree", RootFSDigest: hash, NativeUID: 1234, NativeGID: 1234, Isolation: &effortauthority.NativeUIDIsolation{NativeUID: 1234, ProtectedPaths: []string{"/var/lib/issuer/ledger", "/var/lib/issuer/key"}}, Bindings: map[string]Binding{"canary": {hash, hash, "/srv/work/repository", now.Add(time.Hour)}}, EnvKeys: []string{"PATH", "HOME", "VROOLI_AGENT_IDENTITY_TOKEN"}, NativeBinaries: map[string]string{"/usr/bin/codex": hash}, NetworkAllow: []string{"203.0.113.1/32"}}
	r := Request{RunID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", PolicyID: "canary", PolicyDigest: hash, ProfileDigest: hash, Repository: "/srv/work/repository", Deadline: m.Bindings["canary"].Deadline, Command: "env", Args: []string{"CODEX_AGENT_TAG=exact-run", "/usr/bin/codex", "--some-native-flag"}, Env: []string{"PATH=/usr/bin:/bin", "HOME=/srv/work/repository/.native-home", "DBUS_SESSION_BUS_ADDRESS=secret-host-bus", "OTHER_SECRET=do-not-inherit", "VROOLI_AGENT_IDENTITY_TOKEN=exact-parent-proof"}, WorkingDir: "/srv/work/repository", Timeout: time.Minute}
	return m, r
}
func TestFinitePlanPreservesNativeRouteAndEnforcesDescendants(t *testing.T) {
	m, r := fixture()
	p, e := m.Plan(context.Background(), r, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	joined := strings.Join(p.Args, "\n")
	for _, property := range []string{"NoNewPrivileges=yes", "CapabilityBoundingSet=", "AmbientCapabilities=", "RestrictNamespaces=yes", "Delegate=no", "KillMode=control-group", "IPAddressDeny=any", "IPAddressAllow=203.0.113.1/32", "ProtectProc=invisible", "RootDirectory=" + m.RootFS, "ReadWritePaths=" + r.Repository, "BindPaths=" + m.WorkspaceSource + ":" + r.Repository, "RemainAfterExit=yes", "RuntimeMaxSec=60"} {
		if !strings.Contains(joined, "--property="+property+"\n") {
			t.Errorf("missing enforcement %s", property)
		}
	}
	if strings.Contains(joined, "OTHER_SECRET") || strings.Contains(joined, "secret-host-bus") || strings.Contains(joined, "exact-parent-proof") {
		t.Fatal("host or peer identity leaked into manager environment")
	}
	if !strings.Contains(joined, "--\n/usr/bin/env\nCODEX_AGENT_TAG=exact-run\n/usr/bin/codex") {
		t.Fatal("native env/tag/binary route changed")
	}
}
func TestFinitePlanRefusesWithoutManagerEffects(t *testing.T) {
	cases := map[string]func(*Manifest, *Request){"disabled": func(m *Manifest, r *Request) { m.Enabled = false }, "rootUID": func(m *Manifest, r *Request) { m.NativeUID = 0 }, "wrongUID": func(m *Manifest, r *Request) { m.Isolation.NativeUID++ }, "unknownBackend": func(m *Manifest, r *Request) { m.Backend = "host" }, "loopbackNetwork": func(m *Manifest, r *Request) { m.NetworkAllow = []string{"127.0.0.1/32"} }, "privateNetwork": func(m *Manifest, r *Request) { m.NetworkAllow = []string{"10.0.0.1/32"} }, "wildcardNetwork": func(m *Manifest, r *Request) { m.NetworkAllow = []string{"0.0.0.0/0"} }, "rootOverlap": func(m *Manifest, r *Request) { m.RootFS = r.Repository + "/image" }, "protectedOverlap": func(m *Manifest, r *Request) { m.Isolation.ProtectedPaths = []string{r.Repository + "/key"} }, "policyDrift": func(m *Manifest, r *Request) { r.PolicyDigest = strings.Repeat("b", 64) }, "profileDrift": func(m *Manifest, r *Request) { r.ProfileDigest = strings.Repeat("b", 64) }, "scopeEscape": func(m *Manifest, r *Request) { r.WorkingDir = "/srv/work/repository-neighbor" }, "untrustedBinary": func(m *Manifest, r *Request) { r.Args[1] = "/usr/bin/sh" }, "envShimInjection": func(m *Manifest, r *Request) { r.Args[0] = "--property=User=0" }, "duplicateEnv": func(m *Manifest, r *Request) { r.Env = append(r.Env, "HOME=/root") }, "deadlineDrift": func(m *Manifest, r *Request) { r.Deadline = r.Deadline.Add(time.Hour) }, "expired": func(m *Manifest, r *Request) {
		b := m.Bindings["canary"]
		b.Deadline = time.Now().Add(-time.Second)
		m.Bindings["canary"] = b
		r.Deadline = b.Deadline
	}, "proxyAllowed": func(m *Manifest, r *Request) { m.EnvKeys = append(m.EnvKeys, "HTTPS_PROXY") }, "rootCWD": func(m *Manifest, r *Request) { r.WorkingDir = "/" }, "binaryFlagInjection": func(m *Manifest, r *Request) { r.Args[1] = "--uid=0" }}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			m, r := fixture()
			change(&m, &r)
			if _, e := m.Plan(context.Background(), r, time.Now()); e == nil {
				t.Fatal("unsafe request produced a launch plan")
			}
		})
	}
}
func TestRuntimeManifestIsImmutableAndDisabled(t *testing.T) {
	m, r := fixture()
	m.Enabled = false
	rt, e := NewRuntime(m)
	if e != nil {
		t.Fatal(e)
	}
	m.Enabled = true
	m.Bindings["canary"] = Binding{}
	if _, e := rt.Plan(context.Background(), r); e == nil {
		t.Fatal("caller enabled/amended installed manifest")
	}
}
func TestTerminalRequiresNativeDescendantEvidence(t *testing.T) {
	good := map[string]string{"LoadState": "loaded", "SubState": "exited", "MainPID": "0", "TasksCurrent": "0", "ExecMainCode": "1", "Result": "success"}
	if !terminalState(good) {
		t.Fatal("positive terminal evidence refused")
	}
	for _, k := range []string{"LoadState", "SubState", "MainPID", "TasksCurrent", "ExecMainCode", "Result"} {
		copy := map[string]string{}
		for key, value := range good {
			copy[key] = value
		}
		copy[k] = "unknown"
		if terminalState(copy) {
			t.Fatal("wrapper/unknown fact released occupancy:", k)
		}
	}
	good["TasksCurrent"] = "1"
	if terminalState(good) {
		t.Fatal("live descendant released occupancy")
	}
}
