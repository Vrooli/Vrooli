//go:build linux

package localworkbridge

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Anonymous RAM-only public fixtures, not authority-issued credentials.
func sealedFixture(t *testing.T, raw []byte, seal bool) *os.File {
	t.Helper()
	fd, err := unix.MemfdCreate("public-bridge-test-fixture", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		t.Fatal(err)
	}
	file := os.NewFile(uintptr(fd), "fixture")
	t.Cleanup(func() { file.Close() })
	if _, err = file.Write(raw); err != nil {
		t.Fatal(err)
	}
	if seal {
		if _, err = unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, unix.F_SEAL_SEAL|unix.F_SEAL_SHRINK|unix.F_SEAL_GROW|unix.F_SEAL_WRITE); err != nil {
			t.Fatal(err)
		}
	}
	return file
}
func providerOptions(t *testing.T) SealedProviderOptions {
	t.Helper()
	b := processFixture(t, "idle").binding
	b.Runtime.VenvDir = "/runtime"
	b.Runtime.LockFile = "/runtime/requirements.lock"
	b.EntryPath = "/app/local_work/ipc_entry.py"
	b.Config = json.RawMessage(`{"enabled":true,"storage_root":"/state","export":{"enabled":false}}`)
	b.ConfigSHA256 = sum(b.Config)
	o := SealedProviderOptions{Enabled: true, Binding: b, Isolation: IsolationBinding{ImageSHA256: strings.Repeat("a", 64), UID: 12345, RootDevice: 1, RootInode: 2, StateDevice: 3, StateInode: 4}, Credentials: map[string]SealedCredential{}}
	for _, purpose := range []string{"native_webhook_verify", "events_reader", "runtime_collector"} {
		raw := []byte("Bearer public-test-fixture")
		if purpose == "native_webhook_verify" {
			raw = []byte(strings.Repeat("x", 32))
		}
		o.Credentials[purpose] = SealedCredential{File: sealedFixture(t, raw, true), Identity: "vrooli/local-work-bridge/" + purpose, Version: "fixture-version", ScopeSHA256: strings.Repeat("b", 64)}
	}
	return o
}
func allowFixtureIsolation(p *SealedProvider) {
	p.verify = func(IsolationBinding, ApprovedBinding) bool { return true }
}
func TestSealedProviderDisabledNoIO(t *testing.T) {
	p, err := NewSealedProvider(SealedProviderOptions{Credentials: map[string]SealedCredential{"invalid": {}}})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	p.verify = func(IsolationBinding, ApprovedBinding) bool { t.Fatal("verifier called"); return false }
	if _, err = p.Resolve(context.Background()); err != ErrDisabled {
		t.Fatal(err)
	}
	if _, err = p.Credential(context.Background(), "anything"); err != ErrDisabled {
		t.Fatal(err)
	}
}
func TestSealedProviderOwnedDescriptorsAndCopies(t *testing.T) {
	o := providerOptions(t)
	p, err := NewSealedProvider(o)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	allowFixtureIsolation(p)
	for _, ref := range o.Credentials {
		if _, err := ref.File.Stat(); err == nil {
			t.Fatal("original retained after ownership transfer")
		}
	}
	o.Binding.Config[0] = '!'
	o.Binding.ModuleSHA256["adapter.py"] = "changed"
	b, err := p.Resolve(context.Background())
	if err != nil || b.Config[0] != '{' || b.ModuleSHA256["adapter.py"] == "changed" {
		t.Fatal("input alias", err)
	}
	b.Config[0] = '!'
	b.ModuleSHA256["adapter.py"] = "changed"
	b, err = p.Resolve(context.Background())
	if err != nil || b.Config[0] != '{' || b.ModuleSHA256["adapter.py"] == "changed" {
		t.Fatal("output alias", err)
	}
	value, err := p.Credential(context.Background(), "native_webhook_verify")
	if err != nil || string(value) != strings.Repeat("x", 32) {
		t.Fatal("fixture delivery", err)
	}
	clear(value)
	value, err = p.Credential(context.Background(), "native_webhook_verify")
	if err != nil || string(value) != strings.Repeat("x", 32) {
		t.Fatal("capability alias", err)
	}
	if _, err = p.Credential(context.Background(), "flywheel_submit"); err != ErrProcess {
		t.Fatal("export allowed", err)
	}
}
func TestSealedProviderRejectsDescriptorShapes(t *testing.T) {
	for _, mode := range []string{"inheritable", "unsealed", "empty", "oversize", "disk", "pipe", "duplicate", "missing", "extra", "scope", "version"} {
		t.Run(mode, func(t *testing.T) {
			o := providerOptions(t)
			r := o.Credentials["events_reader"]
			switch mode {
			case "inheritable":
				_, err := unix.FcntlInt(r.File.Fd(), unix.F_SETFD, 0)
				if err != nil {
					t.Fatal(err)
				}
			case "unsealed":
				r.File = sealedFixture(t, []byte("Bearer fixture"), false)
			case "empty":
				r.File = sealedFixture(t, nil, true)
			case "oversize":
				r.File = sealedFixture(t, make([]byte, 8193), true)
			case "disk":
				var err error
				r.File, err = os.CreateTemp(t.TempDir(), "fixture")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { r.File.Close() })
				r.File.Write([]byte("fixture"))
			case "pipe":
				var err error
				var w *os.File
				r.File, w, err = os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { r.File.Close(); w.Close() })
			case "duplicate":
				r.File = o.Credentials["runtime_collector"].File
			case "missing":
				delete(o.Credentials, "native_webhook_verify")
			case "extra":
				o.Credentials["ambient"] = o.Credentials["native_webhook_verify"]
			case "scope":
				r.ScopeSHA256 = "unknown"
			case "version":
				r.Version = ""
			}
			o.Credentials["events_reader"] = r
			p, err := NewSealedProvider(o)
			if err != ErrProcess || p != nil {
				if p != nil {
					p.Close()
				}
				t.Fatal("accepted", err)
			}
		})
	}
}
func TestSealedProviderRejectsBindingShapes(t *testing.T) {
	for _, mode := range []string{"root", "image", "runtime", "lock", "module", "entry", "config", "state", "window"} {
		t.Run(mode, func(t *testing.T) {
			o := providerOptions(t)
			switch mode {
			case "root":
				o.Isolation.UID = 0
			case "image":
				o.Isolation.ImageSHA256 = "unreviewed"
			case "runtime":
				o.Binding.Runtime.VenvDir = "/home/shared/python"
			case "lock":
				o.Binding.LockSHA256 = "unknown"
			case "module":
				delete(o.Binding.ModuleSHA256, "owners.py")
			case "entry":
				o.Binding.EntryPath = "/state/ipc_entry.py"
			case "config":
				o.Binding.ConfigSHA256 = "changed"
			case "state":
				o.Binding.Config = json.RawMessage(`{"enabled":true,"storage_root":"/other","export":{"enabled":false}}`)
				o.Binding.ConfigSHA256 = sum(o.Binding.Config)
			case "window":
				o.Binding.ExpiresAt = o.Binding.NotBefore.Add(8 * 24 * time.Hour)
			}
			p, err := NewSealedProvider(o)
			if err != ErrProcess || p != nil {
				if p != nil {
					p.Close()
				}
				t.Fatal("accepted", err)
			}
		})
	}
}
func TestSealedProviderExpiryRevocationIsolation(t *testing.T) {
	for _, mode := range []string{"cancelled", "future", "expired", "isolation", "closed"} {
		t.Run(mode, func(t *testing.T) {
			o := providerOptions(t)
			p, err := NewSealedProvider(o)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			allowFixtureIsolation(p)
			ctx := context.Background()
			switch mode {
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "future":
				p.now = func() time.Time { return p.binding.NotBefore.Add(-time.Second) }
			case "expired":
				p.now = func() time.Time { return p.binding.ExpiresAt }
			case "isolation":
				p.verify = func(IsolationBinding, ApprovedBinding) bool { return false }
			case "closed":
				p.Close()
			}
			if _, err = p.Resolve(ctx); err != ErrProcess {
				t.Fatal("resolve accepted", err)
			}
			if _, err = p.Credential(ctx, "native_webhook_verify"); err != ErrProcess {
				t.Fatal("credential accepted", err)
			}
		})
	}
}
func TestSealedProviderReadsMaterialLazily(t *testing.T) {
	o := providerOptions(t)
	r := o.Credentials["events_reader"]
	r.File = sealedFixture(t, []byte("Bearer invalid\nheader"), true)
	o.Credentials["events_reader"] = r
	p, err := NewSealedProvider(o)
	if err != nil {
		t.Fatal("constructor read material", err)
	}
	defer p.Close()
	allowFixtureIsolation(p)
	if _, err = p.Resolve(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Credential(context.Background(), "events_reader"); err != ErrProcess {
		t.Fatal("invalid bearer accepted", err)
	}
}
func TestSealedProviderExplicitExport(t *testing.T) {
	o := providerOptions(t)
	o.Binding.Config = json.RawMessage(`{"enabled":true,"storage_root":"/state","export":{"enabled":true}}`)
	o.Binding.ConfigSHA256 = sum(o.Binding.Config)
	if p, err := NewSealedProvider(o); err != ErrProcess {
		if p != nil {
			p.Close()
		}
		t.Fatal(err)
	}
	o.Credentials["flywheel_submit"] = SealedCredential{File: sealedFixture(t, []byte(strings.Repeat("z", 32)), true), Identity: "vrooli/local-work-bridge/flywheel_submit", Version: "fixture", ScopeSHA256: strings.Repeat("c", 64)}
	p, err := NewSealedProvider(o)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	allowFixtureIsolation(p)
	if value, err := p.Credential(context.Background(), "flywheel_submit"); err != nil || len(value) != 32 {
		t.Fatal(err)
	}
}

const safeMountFixture = "20 1 7:0 / / ro,relatime - squashfs /dev/loop0 ro\n21 20 0:1 / /proc ro,nosuid,nodev,noexec - proc proc ro\n22 20 8:1 /bridge /state rw,nosuid,nodev,noexec - ext4 /dev/state rw\n"

func TestIsolationMountClosure(t *testing.T) {
	if !confinedMounts(safeMountFixture) {
		t.Fatal("valid refused")
	}
	for _, bad := range []string{strings.Replace(safeMountFixture, " / ro,", " / rw,", 1), strings.Replace(safeMountFixture, "squashfs", "ext4", 1), strings.Replace(safeMountFixture, " - squashfs", " shared:7 - squashfs", 1), strings.Replace(safeMountFixture, " - squashfs", " master:7 - squashfs", 1), strings.Replace(safeMountFixture, "- ext4 /dev/state", "- fuse.untrusted /dev/state", 1), strings.Replace(safeMountFixture, "- ext4 /dev/state", "- nfs /dev/state", 1), strings.Replace(safeMountFixture, "ro,nosuid,nodev,noexec", "ro,nosuid,nodev", 1), safeMountFixture + "23 20 8:1 /tmp /app/local_work rw - ext4 /dev/x rw\n", strings.Replace(safeMountFixture, "/state", "/state\\040other", 1), "", safeMountFixture + safeMountFixture} {
		if confinedMounts(bad) {
			t.Fatal("unsafe accepted")
		}
	}
}
func TestIsolationProcessConfinement(t *testing.T) {
	safe := "Name:\tfixture\nNoNewPrivs:\t1\nCapInh:\t0000000000000000\nCapPrm:\t0000000000000000\nCapEff:\t0000000000000000\nCapBnd:\t0000000000000000\nCapAmb:\t0000000000000000\n"
	if !confinedStatus(safe) {
		t.Fatal("safe refused")
	}
	for _, bad := range []string{strings.Replace(safe, "NoNewPrivs:\t1", "NoNewPrivs:\t0", 1), strings.Replace(safe, "CapEff:\t0000000000000000", "CapEff:\t0000000000000001", 1), strings.Replace(safe, "CapBnd:\t0000000000000000\n", "", 1), ""} {
		if confinedStatus(bad) {
			t.Fatal("unsafe accepted")
		}
	}
	if verifyIsolation(IsolationBinding{UID: 0xffffffff}, ApprovedBinding{}) {
		t.Fatal("impossible UID accepted")
	}
}

// Exercise the actual exec descriptor boundary, not only FD flag inspection.
func TestSealedProviderChildDescriptorIsolation(t *testing.T) {
	if os.Getenv("BRIDGE_PUBLIC_FD_PROBE") == "1" {
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			path, _ := os.Readlink("/proc/self/fd/" + entry.Name())
			if strings.Contains(path, "public-bridge-test-fixture") {
				t.Fatal("public fixture capability inherited")
			}
		}
		return
	}
	options := providerOptions(t)
	p, err := NewSealedProvider(options)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestSealedProviderChildDescriptorIsolation$")
	cmd.Env = []string{"BRIDGE_PUBLIC_FD_PROBE=1"}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("descriptor probe failed: %v %s", err, output)
	}
}
