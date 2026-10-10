// Package nativeisolation owns the opt-in finite native launch contract.
// It does not install units, identities, keys, permissions or host settings.
package nativeisolation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/effortauthority"
)

var ErrRefused = errors.New("finite native isolation refused")

// ErrUnknown keeps admission occupied when unit termination cannot be proved.
var ErrUnknown = errors.New("finite native unit outcome unknown")
var runID = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
var envKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,79}$`)

type Executable struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Binding struct {
	PolicyDigest  string    `json:"policyDigest"`
	ProfileDigest string    `json:"profileDigest"`
	Repository    string    `json:"repository"`
	Deadline      time.Time `json:"deadline"`
}

// Manifest is owner-installed configuration, never an HTTP/run request body.
// Enabled defaults false. RootFS must be a separately reviewed minimal image,
// with no authority files or host runtime sockets. The host probe must qualify
// systemd's enforcement and descendant/workspace/network behavior first.
type Manifest struct {
	Enabled         bool                                `json:"enabled"`
	Backend         string                              `json:"backend"`
	SystemdRun      Executable                          `json:"systemdRun"`
	Systemctl       Executable                          `json:"systemctl"`
	RootFS          string                              `json:"rootFS"`
	WorkspaceSource string                              `json:"workspaceSource"`
	RootFSDigest    string                              `json:"rootFSDigest"`
	NativeUID       uint32                              `json:"nativeUID"`
	NativeGID       uint32                              `json:"nativeGID"`
	Isolation       *effortauthority.NativeUIDIsolation `json:"isolation"`
	Bindings        map[string]Binding                  `json:"bindings"`
	// ProfileBindings is an optional complete two-profile owner table for the
	// same single policy. Absent preserves the original singleton contract.
	ProfileBindings map[string]string `json:"profileBindings,omitempty"`
	// Only explicit public API/DNS destinations; private/loopback are refused.
	NetworkAllow   []string          `json:"networkAllow"`
	EnvKeys        []string          `json:"envKeys"`
	NativeBinaries map[string]string `json:"nativeBinaries"`
}
type Request struct {
	RunID, PolicyID, PolicyDigest, ProfileDigest, Repository string
	ReservationKey                                           string
	Deadline                                                 time.Time
	Command                                                  string
	Args, Env                                                []string
	WorkingDir                                               string
	Timeout                                                  time.Duration
}

// Plan is immutable construction evidence; only Runtime can execute it.
// Args contain a fixed service definition followed by an explicit separator.
type Plan struct {
	Unit, Command string
	Args          []string
	Env           []string `json:"-"`
	Digest        string
	Request       Request
}

func digest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func validHash(v string) bool {
	b, e := hex.DecodeString(v)
	return e == nil && len(b) == 32 && v == strings.ToLower(v)
}
func cleanPath(p string) bool {
	return filepath.IsAbs(p) && filepath.Clean(p) == p && !strings.ContainsAny(p, "\x00\r\n\t :%\\")
}
func inside(root, p string) bool { return p == root || strings.HasPrefix(p, root+"/") }
func overlap(a, b string) bool   { return inside(a, b) || inside(b, a) }
func validPublicCIDR(s string) bool {
	p, e := netip.ParsePrefix(s)
	if e != nil || p != p.Masked() {
		return false
	}
	// Disallow wildcard, private, local and metadata/service networks. Requiring
	// host CIDRs prevents a prefix from covering a private range at its end.
	a := p.Addr()
	return p.Bits() == a.BitLen() && a.IsGlobalUnicast() && !a.IsPrivate() && !a.IsLoopback() && !a.IsLinkLocalUnicast() && !a.IsUnspecified() && !a.Is4In6()
}
func (m Manifest) Validate() error {
	if m.Backend != "linux-systemd-service-v1" || m.NativeUID == 0 || m.NativeGID == 0 || !cleanPath(m.RootFS) || !cleanPath(m.WorkspaceSource) || m.WorkspaceSource == "/" || overlap(m.RootFS, m.WorkspaceSource) || m.RootFS == "/" || !validHash(m.RootFSDigest) || m.Isolation == nil || m.Isolation.NativeUID != m.NativeUID || len(m.Bindings) != 1 {
		return ErrRefused
	}
	for _, x := range []Executable{m.SystemdRun, m.Systemctl} {
		if !cleanPath(x.Path) || !validHash(x.SHA256) {
			return ErrRefused
		}
	}
	if m.SystemdRun.Path == m.Systemctl.Path {
		return ErrRefused
	}
	for id, b := range m.Bindings {
		if id == "" || !validHash(b.PolicyDigest) || !validHash(b.ProfileDigest) || !cleanPath(b.Repository) || b.Repository == "/" || m.WorkspaceSource == b.Repository || b.Deadline.IsZero() || overlap(m.RootFS, b.Repository) {
			return ErrRefused
		}
		for _, p := range m.Isolation.ProtectedPaths {
			if !cleanPath(p) || overlap(p, b.Repository) || overlap(p, m.RootFS) || overlap(p, m.WorkspaceSource) {
				return ErrRefused
			}
		}
	}
	if len(m.ProfileBindings) != 0 {
		if len(m.ProfileBindings) != 2 {
			return ErrRefused
		}
		seenProfiles := map[string]bool{}
		basePresent := false
		for key, profile := range m.ProfileBindings {
			if key == "" || key != strings.TrimSpace(key) || strings.ContainsAny(key, "\x00\r\n") || !validHash(profile) || seenProfiles[profile] {
				return ErrRefused
			}
			seenProfiles[profile] = true
			for _, b := range m.Bindings {
				basePresent = basePresent || b.ProfileDigest == profile
			}
		}
		if !basePresent {
			return ErrRefused
		}
	}
	seen := map[string]bool{}
	for _, n := range m.NetworkAllow {
		if seen[n] || !validPublicCIDR(n) {
			return ErrRefused
		}
		seen[n] = true
	}
	seen = map[string]bool{}
	for _, k := range m.EnvKeys {
		if !envKey.MatchString(k) || seen[k] || strings.Contains(k, "PROXY") || strings.HasPrefix(k, "LD_") || strings.HasPrefix(k, "DBUS_") || k == "XDG_RUNTIME_DIR" || strings.HasPrefix(k, "SYSTEMD_") || k == "GCONV_PATH" || k == "LOCPATH" || k == "LANG" || k == "LC_ALL" {
			return ErrRefused
		}
		seen[k] = true
	}
	for path, hash := range m.NativeBinaries {
		if !cleanPath(path) || !validHash(hash) {
			return ErrRefused
		}
	}
	if len(m.NativeBinaries) == 0 || !seen["PATH"] || !seen["HOME"] {
		return ErrRefused
	}
	return nil
}

// CheckPolicy compares the complete owner-installed profile table against one
// exact policy. It grants no authority and performs no host or lifecycle reads.
func (m Manifest) CheckPolicy(p effortauthority.Policy) error {
	if m.Validate() != nil || p.Validate() != nil {
		return ErrRefused
	}
	b, ok := m.Bindings[p.ID]
	if !ok || b.PolicyDigest != effortauthority.Digest(p) || b.Repository != p.Repository || !b.Deadline.Equal(p.Deadline) {
		return ErrRefused
	}
	if len(m.ProfileBindings) == 0 {
		if len(p.Profiles) != 1 {
			return ErrRefused
		}
		for _, profile := range p.Profiles {
			if profile != b.ProfileDigest {
				return ErrRefused
			}
		}
		return nil
	}
	if len(p.Profiles) != 2 || len(p.Profiles) != len(m.ProfileBindings) {
		return ErrRefused
	}
	for key, profile := range p.Profiles {
		if m.ProfileBindings[key] != profile {
			return ErrRefused
		}
	}
	return nil
}
func (m Manifest) bindingForProfile(id, profile string) (Binding, bool) {
	b, ok := m.Bindings[id]
	if !ok {
		return Binding{}, false
	}
	if len(m.ProfileBindings) == 0 {
		return b, b.ProfileDigest == profile
	}
	for _, allowed := range m.ProfileBindings {
		if allowed == profile {
			b.ProfileDigest = profile
			return b, true
		}
	}
	return Binding{}, false
}
func (m Manifest) Plan(ctx context.Context, r Request, now time.Time) (Plan, error) {
	if ctx.Err() != nil || !m.Enabled || m.Validate() != nil || !runID.MatchString(r.RunID) || r.Command != "env" || len(r.Args) < 2 || !cleanPath(r.Args[1]) || !cleanPath(r.WorkingDir) || r.Timeout <= 0 || r.Timeout > time.Hour || !validHash(m.NativeBinaries[r.Args[1]]) {
		return Plan{}, ErrRefused
	}
	b, ok := m.bindingForProfile(r.PolicyID, r.ProfileDigest)
	if !ok || r.PolicyDigest != b.PolicyDigest || r.ProfileDigest != b.ProfileDigest || r.Repository != b.Repository || !r.Deadline.Equal(b.Deadline) || !now.Before(b.Deadline) || !inside(b.Repository, r.WorkingDir) {
		return Plan{}, ErrRefused
	}
	// The env shim is limited to the reconciler tag; it cannot introduce flags,
	// nested interpreters or extra environment assignments before the binary.
	tag, value, ok := strings.Cut(r.Args[0], "=")
	if !ok || (tag != "CLAUDE_CODE_AGENT_TAG" && tag != "CODEX_AGENT_TAG" && tag != "OPENCODE_AGENT_TAG") || !envKey.MatchString(tag) || value == "" || strings.ContainsAny(value, "\x00\r\n") {
		return Plan{}, ErrRefused
	}
	for _, arg := range r.Args {
		if strings.ContainsRune(arg, 0) {
			return Plan{}, ErrRefused
		}
	}
	allowed := map[string]bool{}
	for _, k := range m.EnvKeys {
		allowed[k] = true
	}
	seen := map[string]bool{}
	nativeEnv := []string{}
	for _, entry := range r.Env {
		k, v, ok := strings.Cut(entry, "=")
		if !ok || seen[k] || strings.ContainsRune(v, 0) {
			return Plan{}, ErrRefused
		}
		seen[k] = true
		if k == "HOME" && !inside(b.Repository, v) {
			return Plan{}, ErrRefused
		}
		if k == "PATH" && v != "/usr/bin:/bin" {
			return Plan{}, ErrRefused
		}
		if allowed[k] {
			nativeEnv = append(nativeEnv, entry)
		}
	}
	if !seen["HOME"] || !seen["PATH"] {
		return Plan{}, ErrRefused
	}
	sort.Strings(nativeEnv)
	seconds := int64(r.Timeout / time.Second)
	remaining := int64(b.Deadline.Sub(now) / time.Second)
	if remaining < seconds {
		seconds = remaining
	}
	if seconds < 1 {
		return Plan{}, ErrRefused
	}
	unit := "vrooli-finite-" + r.RunID + ".service"
	props := []string{
		"User=" + fmt.Sprint(m.NativeUID), "Group=" + fmt.Sprint(m.NativeGID),
		"RootDirectory=" + m.RootFS, "WorkingDirectory=" + r.WorkingDir,
		"BindPaths=" + m.WorkspaceSource + ":" + b.Repository, "ReadWritePaths=" + b.Repository,
		"RemainAfterExit=yes", "ProtectSystem=strict", "ProtectHome=tmpfs", "PrivateMounts=yes", "PrivateDevices=yes",
		"MountAPIVFS=yes", "ProtectProc=invisible", "ProcSubset=pid",
		"NoNewPrivileges=yes", "CapabilityBoundingSet=", "AmbientCapabilities=",
		"RestrictSUIDSGID=yes", "RestrictNamespaces=yes", "Delegate=no", "KillMode=control-group",
		"ProtectKernelTunables=yes", "ProtectKernelModules=yes", "ProtectKernelLogs=yes", "ProtectControlGroups=yes",
		"RestrictAddressFamilies=AF_INET AF_INET6", "SystemCallFilter=~@mount @privileged @reboot @swap",
		"TemporaryFileSystem=/tmp:rw,nosuid,nodev /run:rw,nosuid,nodev",
		"IPAddressDeny=any", "RuntimeMaxSec=" + fmt.Sprint(seconds), "TimeoutStopSec=5s",
	}
	if len(m.NetworkAllow) > 0 {
		props = append(props, "IPAddressAllow="+strings.Join(m.NetworkAllow, " "))
	}
	args := []string{"--system", "--quiet", "--pipe", "--wait", "--service-type=exec", "--unit=" + unit}
	for _, p := range props {
		args = append(args, "--property="+p)
	}
	for _, e := range nativeEnv {
		key, _, _ := strings.Cut(e, "=")
		args = append(args, "--setenv="+key)
	}
	args = append(args, "--", "/usr/bin/env")
	args = append(args, r.Args...)
	p := Plan{Unit: unit, Command: m.SystemdRun.Path, Args: args, Env: append(nativeEnv, "LANG=C", "LC_ALL=C"), Request: Request{ReservationKey: r.ReservationKey, RunID: r.RunID, PolicyID: r.PolicyID, PolicyDigest: r.PolicyDigest, ProfileDigest: r.ProfileDigest, Repository: r.Repository, Deadline: r.Deadline, Command: r.Command, Args: r.Args, WorkingDir: r.WorkingDir, Timeout: r.Timeout}}
	p.Digest = digest(struct {
		Manifest Manifest
		Request  Request
		Args     []string
	}{m, r, args})
	return p, nil
}
