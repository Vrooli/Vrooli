//go:build linux

package hostwireless

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func recoveryFixture(t *testing.T) (*recoveryObserver, RecoveryProofRequest, func(string, string)) {
	t.Helper()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	req := RecoveryProofRequest{TargetABI: "7.0.0-30-generic", RecoveryRelease: "7.0.0-29-generic", MenuID: "advanced>stock", RequiredEthernetModules: []string{"atlantic"}, RequiredNVIDIAModules: []string{"nvidia", "nvidia_drm"}}
	write := func(name, data string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte(data), 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, p := range []string{"boot/vmlinuz-" + req.RecoveryRelease, "boot/initrd.img-" + req.RecoveryRelease, "boot/grub/grubenv", "etc/default/grub", "lib/modules/" + req.RecoveryRelease + "/modules.dep", "lib/modules/" + req.RecoveryRelease + "/modules.alias"} {
		write(p, "stock "+p)
	}
	write("boot/grub/grub.cfg", "submenu 'Advanced' --id 'advanced' {\nmenuentry 'stock' --id 'stock' {\nlinux /boot/vmlinuz-"+req.RecoveryRelease+" root=x\ninitrd /boot/initrd.img-"+req.RecoveryRelease+"\n}\n}\n")
	write("etc/machine-id", strings.Repeat("a", 32))
	write("proc/sys/kernel/random/boot_id", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	write("proc/cmdline", "BOOT_IMAGE=/boot/vmlinuz-"+req.RecoveryRelease+" ro")
	write("proc/net/route", "Iface Destination Gateway Flags\nenp1s0 00000000 01010101 0003\n")
	write("sys/class/net/enp1s0/operstate", "up")
	write("sys/class/net/enp1s0/carrier", "1")
	if e := os.MkdirAll(filepath.Join(dir, "sys/class/net/enp1s0/device/driver"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink("../../../../module/atlantic", filepath.Join(dir, "sys/class/net/enp1s0/device/driver/module")); e != nil {
		t.Fatal(e)
	}
	for _, m := range []string{"atlantic", "nvidia", "nvidia_drm"} {
		if e := os.MkdirAll(filepath.Join(dir, "sys/module", m), 0700); e != nil {
			t.Fatal(e)
		}
		write("lib/modules/"+req.RecoveryRelease+"/kernel/"+m+".ko", "stock "+m)
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { root.Close() })
	o := &recoveryObserver{root: root, uid: uint32(os.Getuid()), release: func() (string, error) { return req.RecoveryRelease, nil }, console: func() (string, error) { return "tty2", nil }, command: func(_ context.Context, cmd string, args ...string) ([]byte, error) {
		if cmd != "/usr/sbin/modinfo" || len(args) != 4 || args[0] != "-k" || args[1] != req.RecoveryRelease || args[2] != "-n" {
			t.Fatal("unexpected native command", cmd, args)
		}
		return []byte("/lib/modules/" + req.RecoveryRelease + "/kernel/" + args[3] + ".ko\n"), nil
	}}
	return o, req, write
}
func TestRecoveryProofIndependentlyObservedAndReopened(t *testing.T) {
	o, r, write := recoveryFixture(t)
	p, e := o.record(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	if p.Console != "tty2" || len(p.Files) != 10 {
		t.Fatalf("%+v", p)
	}
	if _, e = o.record(context.Background(), r); e != nil {
		t.Fatal("independently reobserved proof refresh failed", e)
	}
	o.release = func() (string, error) { return r.TargetABI, nil }
	if _, e = o.verify(context.Background(), r); e == nil {
		t.Fatal("same boot accepted")
	}
	write("proc/sys/kernel/random/boot_id", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	if _, e = o.verify(context.Background(), r); e != nil {
		t.Fatal(e)
	}
	write("boot/initrd.img-"+r.RecoveryRelease, "foreign")
	if _, e = o.verify(context.Background(), r); e == nil {
		t.Fatal("changed fallback artifact accepted")
	}
}
func TestRecoveryCannotUseAssertionsOrForeignRecord(t *testing.T) {
	for _, mode := range []string{"console", "ethernet", "module", "image", "menu", "overlay", "artifact-parent", "machine", "symlink", "permissions", "unknown-json"} {
		t.Run(mode, func(t *testing.T) {
			o, r, write := recoveryFixture(t)
			switch mode {
			case "console":
				o.console = func() (string, error) { return "", errors.New("SSH terminal") }
			case "ethernet":
				write("sys/class/net/enp1s0/carrier", "0")
			case "module":
				if e := o.root.Remove("sys/module/nvidia"); e != nil {
					t.Fatal(e)
				}
			case "image":
				write("proc/cmdline", "BOOT_IMAGE=/vmlinuz-foreign")
			case "menu":
				r.MenuID = "missing"
			case "overlay":
				write("lib/modules/"+r.RecoveryRelease+"/updates/vrooli-mt7927/module.ko", "candidate")
			case "artifact-parent":
				if e := o.root.Chmod("boot", 0777); e != nil {
					t.Fatal(e)
				}
			}
			if mode == "console" || mode == "ethernet" || mode == "module" || mode == "image" || mode == "menu" || mode == "overlay" || mode == "artifact-parent" {
				if _, e := o.record(context.Background(), r); e == nil {
					t.Fatal("accepted incomplete independent proof")
				}
				return
			}
			p, e := o.record(context.Background(), r)
			if e != nil {
				t.Fatal(e)
			}
			o.release = func() (string, error) { return r.TargetABI, nil }
			write("proc/sys/kernel/random/boot_id", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
			name := recoveryRecordDir + "/" + recoveryRecordName(r)
			switch mode {
			case "machine":
				write("etc/machine-id", strings.Repeat("b", 32))
			case "symlink":
				if e := o.root.Remove(name); e != nil {
					t.Fatal(e)
				}
				if e := o.root.Symlink("/etc/passwd", name); e != nil {
					t.Fatal(e)
				}
			case "permissions":
				if e := o.root.Chmod(name, 0666); e != nil {
					t.Fatal(e)
				}
			case "unknown-json":
				data, _ := json.Marshal(p)
				write(name, strings.TrimSuffix(string(data), "}")+`,"assertion":true}`)
			}
			if _, e = o.verify(context.Background(), r); e == nil {
				t.Fatal("accepted untrusted recovery proof")
			}
		})
	}
}
