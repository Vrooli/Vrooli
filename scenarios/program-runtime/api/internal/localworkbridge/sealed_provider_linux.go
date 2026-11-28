//go:build linux

package localworkbridge

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// IsolationBinding is prepared by the trusted lifecycle launcher AFTER it has
// verified the approved image digest and entered its isolated root/PID/UID
// namespace. Device/inode identity is local to that launch, not a portable
// artifact signature. The provider never mounts, changes UID, or grants access.
type IsolationBinding struct {
	ImageSHA256             string
	UID                     uint32
	RootDevice, RootInode   uint64
	StateDevice, StateInode uint64
}

// SealedCredential binds an already-approved authority version to one purpose.
// The trusted launcher supplies a sealed anonymous memfd, never a pathname,
// environment value, generic authority client, or recoverable secret string.
// On successful construction ownership transfers and originals are closed.
// Failure leaves originals caller-owned. Inputs must already be close-on-exec.
// ScopeSHA256 names the reviewed server-enforced principal/operation/target scope.
type SealedCredential struct {
	File                           *os.File
	Identity, Version, ScopeSHA256 string
}

type SealedProviderOptions struct {
	Enabled     bool
	Binding     ApprovedBinding
	Isolation   IsolationBinding
	Credentials map[string]SealedCredential
}

// SealedProvider is an inert-by-default concrete Linux BindingProvider. It can
// only consume already-delivered, kernel-sealed capabilities. It cannot discover,
// mint, refresh, or retrieve credentials from the general user authority.
// Close must follow joined Host.Stop; expiry also refuses further delivery.
type SealedProvider struct {
	mu              sync.Mutex
	enabled, closed bool
	binding         ApprovedBinding
	isolation       IsolationBinding
	files           map[string]*os.File
	now             func() time.Time
	verify          func(IsolationBinding, ApprovedBinding) bool
}

func NewSealedProvider(options SealedProviderOptions) (*SealedProvider, error) {
	p := &SealedProvider{enabled: options.Enabled, now: time.Now, verify: verifyIsolation}
	if !options.Enabled {
		return p, nil
	}
	p.binding = cloneBinding(options.Binding)
	p.isolation = options.Isolation
	if !validProviderBinding(p.binding, p.isolation) {
		return nil, ErrProcess
	}
	var config struct {
		Export struct {
			Enabled bool `json:"enabled"`
		} `json:"export"`
	}
	if json.Unmarshal(p.binding.Config, &config) != nil {
		return nil, ErrProcess
	}
	expected := map[string]bool{"native_webhook_verify": true, "events_reader": true, "runtime_collector": true}
	if config.Export.Enabled {
		expected["flywheel_submit"] = true
	}
	if len(options.Credentials) != len(expected) {
		return nil, ErrProcess
	}
	p.files = make(map[string]*os.File)
	seen := make(map[[2]uint64]bool)
	fail := func() (*SealedProvider, error) { p.Close(); return nil, ErrProcess }
	for purpose, ref := range options.Credentials {
		if !expected[purpose] || ref.File == nil || !safeLabel(ref.Identity) || !safeLabel(ref.Version) || !hexDigest(ref.ScopeSHA256) {
			return fail()
		}
		flags, err := unix.FcntlInt(ref.File.Fd(), unix.F_GETFD, 0)
		if err != nil || flags&unix.FD_CLOEXEC == 0 {
			return fail()
		}
		// Duplicate first, then consume originals only after ALL references pass.
		// No caller descriptor can remain accidentally inheritable by a child.
		fd, err := unix.FcntlInt(ref.File.Fd(), unix.F_DUPFD_CLOEXEC, 5)
		if err != nil {
			return fail()
		}
		owned := os.NewFile(uintptr(fd), "local-work-capability")
		p.files[purpose] = owned
		stat, ok := sealedStat(owned)
		if !ok {
			return fail()
		}
		key := [2]uint64{uint64(stat.Dev), stat.Ino}
		if seen[key] {
			return fail()
		}
		seen[key] = true
	}
	for _, ref := range options.Credentials {
		ref.File.Close()
	}
	return p, nil
}

func cloneBinding(b ApprovedBinding) ApprovedBinding {
	b.Config = append(json.RawMessage(nil), b.Config...)
	copied := make(map[string]string, len(b.ModuleSHA256))
	for k, v := range b.ModuleSHA256 {
		copied[k] = v
	}
	b.ModuleSHA256 = copied
	return b
}

func hexDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func safeLabel(s string) bool {
	if len(s) < 1 || len(s) > 512 {
		return false
	}
	for _, c := range s {
		if c <= 32 || c >= 127 {
			return false
		}
	}
	return true
}

func validProviderBinding(b ApprovedBinding, isolation IsolationBinding) bool {
	// Fixed mount layout prevents caller-supplied paths from reaching a second
	// writable tree. Storage is deliberately writable, separately identity-bound.
	if isolation.UID == 0 || isolation.RootInode == 0 || isolation.StateInode == 0 || !hexDigest(isolation.ImageSHA256) {
		return false
	}
	if b.Runtime.VenvDir != "/runtime" || b.Runtime.LockFile != "/runtime/requirements.lock" || b.Runtime.BasePython != "3.12" || b.Runtime.UV != "" || b.EntryPath != "/app/local_work/ipc_entry.py" {
		return false
	}
	if !hexDigest(b.LockSHA256) || !hexDigest(b.EntrySHA256) || !digestMatches(b.Config, b.ConfigSHA256) || len(b.Config) > 64<<10 {
		return false
	}
	if len(b.ModuleSHA256) != len(moduleNames) {
		return false
	}
	for _, name := range moduleNames {
		if !hexDigest(b.ModuleSHA256[name]) {
			return false
		}
	}
	if b.ModuleSHA256["ipc_entry.py"] != b.EntrySHA256 {
		return false
	}
	var config struct {
		Enabled bool   `json:"enabled"`
		Storage string `json:"storage_root"`
	}
	if json.Unmarshal(b.Config, &config) != nil || !config.Enabled || config.Storage != "/state" {
		return false
	}
	return b.NotBefore.Before(b.ExpiresAt) && b.ExpiresAt.Sub(b.NotBefore) <= 7*24*time.Hour
}

func (p *SealedProvider) active(ctx context.Context) error {
	if !p.enabled {
		return ErrDisabled
	}
	if p.closed || ctx == nil || ctx.Err() != nil {
		return ErrProcess
	}
	now := p.now()
	if now.Before(p.binding.NotBefore) || !now.Before(p.binding.ExpiresAt) {
		return ErrProcess
	}
	if !p.verify(p.isolation, p.binding) {
		return ErrProcess
	}
	if ctx.Err() != nil {
		return ErrProcess
	}
	return nil
}

func (p *SealedProvider) Resolve(ctx context.Context) (ApprovedBinding, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.active(ctx); err != nil {
		return ApprovedBinding{}, err
	}
	return cloneBinding(p.binding), nil
}

func (p *SealedProvider) Credential(ctx context.Context, purpose string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.active(ctx); err != nil {
		return nil, err
	}
	file := p.files[purpose]
	stat, ok := sealedStat(file)
	if !ok {
		return nil, ErrProcess
	}
	value := make([]byte, int(stat.Size))
	n, err := file.ReadAt(value, 0)
	if err != nil && err != io.EOF || n != len(value) || ctx.Err() != nil || !p.now().Before(p.binding.ExpiresAt) || !validCredentialValue(purpose, value) {
		clear(value)
		return nil, ErrProcess
	}
	return value, nil
}

func validCredentialValue(purpose string, value []byte) bool {
	if purpose == "native_webhook_verify" || purpose == "flywheel_submit" {
		return len(value) == 32
	}
	if purpose != "events_reader" && purpose != "runtime_collector" {
		return false
	}
	if len(value) <= len("Bearer ") || !strings.HasPrefix(string(value), "Bearer ") {
		return false
	}
	for _, c := range value[len("Bearer "):] {
		if c <= 32 || c >= 127 {
			return false
		}
	}
	return true
}

func sealedStat(file *os.File) (unix.Stat_t, bool) {
	var st unix.Stat_t
	if file == nil {
		return st, false
	}
	fd := file.Fd()
	if unix.Fstat(int(fd), &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Size < 1 || st.Size > 8192 {
		return st, false
	}
	seals, err := unix.FcntlInt(fd, unix.F_GET_SEALS, 0)
	required := unix.F_SEAL_SEAL | unix.F_SEAL_SHRINK | unix.F_SEAL_GROW | unix.F_SEAL_WRITE
	if err != nil || seals&required != required {
		return st, false
	}
	// Sealed anonymous tmpfs excludes arbitrary file/provider reads and later
	// byte substitution. Kernel I/O and swapped pages are not forcibly preemptible;
	// this is not a hard real-time or malicious-kernel cancellation guarantee.
	var fs unix.Statfs_t
	if unix.Fstatfs(int(fd), &fs) != nil || fs.Type != unix.TMPFS_MAGIC || st.Nlink != 0 {
		return st, false
	}
	return st, true
}

func (p *SealedProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	for purpose, file := range p.files {
		file.Close()
		delete(p.files, purpose)
	}
	return nil
}

// verifyIsolation refuses the normal desktop/shared-checkout execution context.
// Only an already-created, launcher-approved read-only SquashFS/EROFS root with
// exactly /proc and a private /state mount is accepted. This is a check, not a
// sandbox creator. PID/UID namespace provisioning remains the launcher's duty.
func verifyIsolation(expected IsolationBinding, binding ApprovedBinding) bool {
	if uint32(os.Geteuid()) != expected.UID || os.Getuid() != os.Geteuid() {
		return false
	}
	// CLOEXEC does not stop same-UID children reopening a parent's /proc FDs.
	// The lifecycle owner must establish non-dumpability in this running Go
	// process after exec and before secret delivery; exec can reset the flag.
	// This checks the setting; it never changes process security policy.
	dumpable, _, errno := unix.Syscall6(unix.SYS_PRCTL, unix.PR_GET_DUMPABLE, 0, 0, 0, 0, 0)
	if errno != 0 || dumpable != 0 {
		return false
	}

	// Inspect only fixed, bounded process metadata. No environment, status of
	// other processes, credentials, or user data is read.
	status, err := boundedMetadata("/proc/self/status", 64<<10)
	if err != nil || !confinedStatus(string(status)) {
		return false
	}
	mounts, err := boundedMetadata("/proc/self/mountinfo", 256<<10)
	if err != nil || !confinedMounts(string(mounts)) {
		return false
	}
	var root, state unix.Stat_t
	var fs unix.Statfs_t
	if unix.Stat("/", &root) != nil || uint64(root.Dev) != expected.RootDevice || root.Ino != expected.RootInode || unix.Statfs("/", &fs) != nil || fs.Flags&unix.ST_RDONLY == 0 || fs.Type != unix.SQUASHFS_MAGIC && fs.Type != 0xe0f5e1e2 /* EROFS_SUPER_MAGIC_V1 */ {
		return false
	}
	if unix.Lstat("/state", &state) != nil || uint64(state.Dev) != expected.StateDevice || state.Ino != expected.StateInode || state.Mode&unix.S_IFMT != unix.S_IFDIR || state.Uid != expected.UID || state.Mode&0777 != 0700 {
		return false
	}
	// Resolving interpreter symlinks must stay within the approved immutable root.
	python, err := filepath.EvalSymlinks("/runtime/bin/python")
	return err == nil && strings.HasPrefix(python, "/runtime/") && binding.EntryPath == "/app/local_work/ipc_entry.py"
}

func boundedMetadata(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	value, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(value)) > limit {
		return nil, ErrProcess
	}
	return value, nil
}

func confinedStatus(raw string) bool {
	wanted := map[string]string{"NoNewPrivs": "1", "CapInh": "0", "CapPrm": "0", "CapEff": "0", "CapBnd": "0", "CapAmb": "0"}
	for _, line := range strings.Split(raw, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		expect, relevant := wanted[key]
		if !relevant {
			continue
		}
		value = strings.TrimSpace(value)
		if expect == "0" {
			number, err := strconv.ParseUint(value, 16, 64)
			if err != nil || number != 0 {
				return false
			}
		} else if value != expect {
			return false
		}
		delete(wanted, key)
	}
	return len(wanted) == 0
}

func confinedMounts(raw string) bool {
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			return false
		}
		separator := -1
		for i, field := range fields {
			if field == "-" {
				separator = i
				break
			}
		}
		// A private tree has no shared/master/propagate_from optional tags.
		// Reject unknown optional tags too rather than assuming future safety.
		if separator != 6 || separator+3 >= len(fields) {
			return false
		}
		mount, options, kind := fields[4], ","+fields[5]+",", fields[separator+1]
		if seen[mount] {
			return false
		}
		seen[mount] = true
		switch mount {
		case "/":
			if kind != "squashfs" && kind != "erofs" || !strings.Contains(options, ",ro,") {
				return false
			}
		case "/proc":
			if kind != "proc" || !strings.Contains(options, ",ro,") || !strings.Contains(options, ",nosuid,") || !strings.Contains(options, ",nodev,") || !strings.Contains(options, ",noexec,") {
				return false
			}
		case "/state":
			// Never touch an arbitrary network/FUSE-backed state mount. Validate
			// this kernel mount table before path metadata or credential delivery.
			if kind != "ext4" && kind != "xfs" && kind != "btrfs" && kind != "tmpfs" {
				return false
			}
			if !strings.Contains(options, ",rw,") || !strings.Contains(options, ",nosuid,") || !strings.Contains(options, ",nodev,") || !strings.Contains(options, ",noexec,") {
				return false
			}
		default:
			return false
		}
	}
	return len(seen) == 3 && seen["/"] && seen["/proc"] && seen["/state"]
}
