//go:build linux

package hostwireless

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit/grub"
	"golang.org/x/sys/unix"
)

const recoveryRecordDir = "var/lib/vrooli/host-wireless/recovery"

var recoveryReleaseName = regexp.MustCompile(`^[0-9][A-Za-z0-9_.-]{0,63}$`)
var recoveryModuleName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
var recoveryUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type recoveryObserver struct {
	root    *os.Root
	uid     uint32
	release func() (string, error)
	console func() (string, error)
	command func(context.Context, string, ...string) ([]byte, error)
}

func nativeRecoveryObserver() (*recoveryObserver, error) {
	if os.Geteuid() != 0 {
		return nil, errors.New("recovery proof requires the normal root host command")
	}
	root, e := os.OpenRoot("/")
	if e != nil {
		return nil, e
	}
	return &recoveryObserver{root: root, uid: 0, release: func() (string, error) {
		var u unix.Utsname
		e := unix.Uname(&u)
		return unix.ByteSliceToString(u.Release[:]), e
	}, console: activePhysicalVT, command: func(ctx context.Context, command string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
		cmd.WaitDelay = time.Second
		var b mokStatusOutput
		cmd.Stdout = &b
		cmd.Stderr = &b
		e := cmd.Run()
		if b.exceeded {
			return nil, errors.New("native provider output exceeds bound")
		}
		return b.Bytes(), e
	}}, nil
}
func recordNativeRecoveryProof(ctx context.Context, req RecoveryProofRequest) (RecoveryProof, error) {
	o, e := nativeRecoveryObserver()
	if e != nil {
		return RecoveryProof{}, e
	}
	defer o.root.Close()
	return o.record(ctx, req)
}
func verifyNativeRecoveryProof(ctx context.Context, req RecoveryProofRequest) (RecoveryProof, error) {
	o, e := nativeRecoveryObserver()
	if e != nil {
		return RecoveryProof{}, e
	}
	defer o.root.Close()
	return o.verify(ctx, req)
}
func validateRecoveryRequest(r RecoveryProofRequest) error {
	if !recoveryReleaseName.MatchString(r.TargetABI) || !recoveryReleaseName.MatchString(r.RecoveryRelease) || r.TargetABI == r.RecoveryRelease || r.MenuID == "" || len(r.MenuID) > 512 {
		return errors.New("exact distinct target/fallback releases and GRUB menu ID required")
	}
	if len(r.RequiredEthernetModules) == 0 || len(r.RequiredEthernetModules) > 16 || len(r.RequiredNVIDIAModules) == 0 || len(r.RequiredNVIDIAModules) > 16 {
		return errors.New("required target Ethernet and NVIDIA provider sets are empty")
	}
	all := append(slices.Clone(r.RequiredEthernetModules), r.RequiredNVIDIAModules...)
	seen := map[string]bool{}
	for _, m := range all {
		if !recoveryModuleName.MatchString(m) || seen[m] {
			return errors.New("invalid or duplicate recovery provider")
		}
		seen[m] = true
	}
	for _, m := range r.RequiredNVIDIAModules {
		if !strings.HasPrefix(m, "nvidia") {
			return errors.New("non-NVIDIA recovery provider")
		}
	}
	return nil
}
func (o *recoveryObserver) text(name string) (string, error) {
	f, e := o.root.Open(name)
	if e != nil {
		return "", e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 65537))
	if len(b) > 65536 {
		return "", errors.New("observation exceeds bound")
	}
	return strings.TrimSpace(string(b)), e
}
func (o *recoveryObserver) identity(ctx context.Context, name string) (FileIdentity, error) {
	if e := ctx.Err(); e != nil {
		return FileIdentity{}, e
	}
	// Every artifact parent must remain outside unprivileged write control.
	for ancestor := path.Dir(name); ancestor != "."; ancestor = path.Dir(ancestor) {
		st, e := o.root.Stat(ancestor)
		if e != nil {
			return FileIdentity{}, e
		}
		u, ok := st.Sys().(*syscall.Stat_t)
		if !ok || u.Uid != o.uid || st.Mode().Perm()&0022 != 0 || !st.IsDir() {
			return FileIdentity{}, fmt.Errorf("unprotected recovery artifact ancestor %s", ancestor)
		}
	}
	leaf, e := o.root.Lstat(name)
	if e != nil || !leaf.Mode().IsRegular() {
		return FileIdentity{}, fmt.Errorf("recovery artifact must be a regular non-symlink file: %s", name)
	}
	f, e := o.root.Open(name)
	if e != nil {
		return FileIdentity{}, e
	}
	defer f.Close()
	var before, after unix.Stat_t
	if e = unix.Fstat(int(f.Fd()), &before); e != nil {
		return FileIdentity{}, e
	}
	if before.Mode&unix.S_IFMT != unix.S_IFREG || before.Uid != o.uid || before.Mode&0022 != 0 || before.Size > 512<<20 {
		return FileIdentity{}, fmt.Errorf("unprotected recovery artifact %s", name)
	}
	hash := sha256.New()
	n, e := io.Copy(hash, io.LimitReader(f, 512<<20+1))
	if e != nil {
		return FileIdentity{}, e
	}
	if n > 512<<20 {
		return FileIdentity{}, errors.New("recovery artifact exceeds bound")
	}
	if e = unix.Fstat(int(f.Fd()), &after); e != nil {
		return FileIdentity{}, e
	}
	before.Atim = after.Atim // Reading may update access time; content/ownership must stay fixed.
	if before != after {
		return FileIdentity{}, errors.New("recovery artifact changed during observation")
	}
	if e = ctx.Err(); e != nil {
		return FileIdentity{}, e
	}
	return FileIdentity{Path: "/" + name, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}
func (o *recoveryObserver) artifacts(ctx context.Context, r RecoveryProofRequest) ([]FileIdentity, error) {
	cfg, e := o.text("boot/grub/grub.cfg")
	if e != nil {
		return nil, e
	}
	entries, e := grub.BootEntries([]byte(cfg))
	if e != nil {
		return nil, e
	}
	found := false
	for _, entry := range entries {
		if entry.Selector != r.MenuID {
			continue
		}
		if found {
			return nil, errors.New("ambiguous fallback menu")
		}
		found = true
		if entry.Kernel != "/boot/vmlinuz-"+r.RecoveryRelease && entry.Kernel != "/vmlinuz-"+r.RecoveryRelease {
			return nil, errors.New("fallback menu kernel mismatch")
		}
		if entry.Initrd != "/boot/initrd.img-"+r.RecoveryRelease && entry.Initrd != "/initrd.img-"+r.RecoveryRelease {
			return nil, errors.New("fallback menu initrd mismatch")
		}
	}
	if !found {
		return nil, errors.New("explicit fallback GRUB menu ID not found")
	}
	names := []string{"boot/vmlinuz-" + r.RecoveryRelease, "boot/initrd.img-" + r.RecoveryRelease, "boot/grub/grub.cfg", "boot/grub/grubenv", "etc/default/grub", "lib/modules/" + r.RecoveryRelease + "/modules.dep", "lib/modules/" + r.RecoveryRelease + "/modules.alias"}
	updates, e := o.root.Open("lib/modules/" + r.RecoveryRelease + "/updates")
	if e == nil {
		entries, err := updates.ReadDir(-1)
		updates.Close()
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "vrooli-") {
				return nil, errors.New("fallback contains a Vrooli wireless overlay")
			}
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
	var files []FileIdentity
	for _, name := range names {
		f, e := o.identity(ctx, name)
		if e != nil {
			return nil, e
		}
		files = append(files, f)
	}
	for _, module := range append(slices.Clone(r.RequiredEthernetModules), r.RequiredNVIDIAModules...) {
		output, e := o.command(ctx, "/usr/sbin/modinfo", "-k", r.RecoveryRelease, "-n", module)
		if e != nil {
			return nil, fmt.Errorf("fallback %s provider: %w", module, e)
		}
		p := strings.TrimSpace(string(output))
		prefix := "/lib/modules/" + r.RecoveryRelease + "/"
		if strings.HasPrefix(p, "/usr/lib/modules/") {
			p = strings.TrimPrefix(p, "/usr")
		}
		if !strings.HasPrefix(p, prefix) || path.Clean(p) != p || strings.Contains(p, "vrooli-") {
			return nil, errors.New("provider is not a retained fallback module")
		}
		f, e := o.identity(ctx, strings.TrimPrefix(p, "/"))
		if e != nil {
			return nil, e
		}
		files = append(files, f)
	}
	return files, nil
}
func (o *recoveryObserver) machineBoot() (string, string, error) {
	m, e := o.text("etc/machine-id")
	if e != nil || len(m) != 32 {
		return "", "", errors.New("machine identity unavailable")
	}
	for _, c := range m {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return "", "", errors.New("invalid machine identity")
		}
	}
	b, e := o.text("proc/sys/kernel/random/boot_id")
	if e != nil || !recoveryUUID.MatchString(b) {
		return "", "", errors.New("boot identity unavailable")
	}
	return m, b, nil
}
func (o *recoveryObserver) ethernet(r RecoveryProofRequest) ([]string, error) {
	routes, e := o.text("proc/net/route")
	if e != nil {
		return nil, e
	}
	var out []string
	found := map[string]bool{}
	for _, line := range strings.Split(routes, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || f[1] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(f[3], 16, 32)
		if err != nil || flags&1 == 0 || flags&0x200 != 0 {
			continue
		}
		name := f[0]
		if !nativeName.MatchString(name) {
			continue
		}
		if _, e := o.root.Stat("sys/class/net/" + name + "/wireless"); e == nil {
			continue
		}
		state, e := o.text("sys/class/net/" + name + "/operstate")
		if e != nil || state != "up" {
			continue
		}
		carrier, e := o.text("sys/class/net/" + name + "/carrier")
		if e != nil || carrier != "1" {
			continue
		}
		link, e := o.root.Readlink("sys/class/net/" + name + "/device/driver/module")
		if e != nil {
			continue
		}
		module := path.Base(link)
		if !slices.Contains(r.RequiredEthernetModules, module) {
			continue
		}
		found[module] = true
		out = append(out, name)
	}
	for _, m := range r.RequiredEthernetModules {
		if !found[m] {
			return nil, fmt.Errorf("fallback has no usable default Ethernet route through %s", m)
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}
func (o *recoveryObserver) record(ctx context.Context, r RecoveryProofRequest) (RecoveryProof, error) {
	var proof RecoveryProof
	if e := validateRecoveryRequest(r); e != nil {
		return proof, e
	}
	release, e := o.release()
	if e != nil || release != r.RecoveryRelease {
		return proof, errors.New("record proof while actually running the selected stock fallback")
	}
	console, e := o.console()
	if e != nil {
		return proof, e
	}
	machine, boot, e := o.machineBoot()
	if e != nil {
		return proof, e
	}
	cmdline, e := o.text("proc/cmdline")
	if e != nil {
		return proof, e
	}
	image := false
	for _, f := range strings.Fields(cmdline) {
		if f == "BOOT_IMAGE=/boot/vmlinuz-"+r.RecoveryRelease || f == "BOOT_IMAGE=/vmlinuz-"+r.RecoveryRelease {
			image = true
		}
	}
	if !image {
		return proof, errors.New("running boot image does not match selected fallback")
	}
	for _, m := range append(slices.Clone(r.RequiredEthernetModules), r.RequiredNVIDIAModules...) {
		if _, e := o.root.Stat("sys/module/" + m); e != nil {
			return proof, fmt.Errorf("required fallback module %s is not loaded", m)
		}
	}
	wired, e := o.ethernet(r)
	if e != nil {
		return proof, e
	}
	files, e := o.artifacts(ctx, r)
	if e != nil {
		return proof, e
	}
	proof = RecoveryProof{SchemaVersion: 1, Request: r, MachineID: machine, BootID: boot, Console: console, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), Files: files, EthernetInterfaces: wired}
	if _, b, e := o.machineBoot(); e != nil || b != boot {
		return RecoveryProof{}, errors.New("boot changed during recovery observation")
	}
	data, e := json.Marshal(proof)
	if e != nil {
		return RecoveryProof{}, e
	}
	if e = o.save(r, data); e != nil {
		return RecoveryProof{}, e
	}
	return proof, nil
}
func (o *recoveryObserver) verify(ctx context.Context, r RecoveryProofRequest) (RecoveryProof, error) {
	return o.verifyRelease(ctx, r, r.TargetABI)
}
func (o *recoveryObserver) verifyRelease(ctx context.Context, r RecoveryProofRequest, expectedRelease string) (RecoveryProof, error) {
	var p RecoveryProof
	if e := validateRecoveryRequest(r); e != nil {
		return p, e
	}
	release, e := o.release()
	if e != nil || release != expectedRelease {
		return p, errors.New("recovery verification requires the exact running target ABI")
	}
	f, e := o.protectedRecord(r)
	if e != nil {
		return p, e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, 65537))
	if e != nil || len(data) > 65536 {
		return p, errors.New("invalid recovery record size")
	}
	if e = strictWirelessJSON(data, &p); e != nil {
		return p, e
	}
	machine, boot, e := o.machineBoot()
	if e != nil {
		return p, e
	}
	if p.SchemaVersion != 1 || !reflect.DeepEqual(p.Request, r) || p.MachineID != machine || p.BootID == boot || !recoveryUUID.MatchString(p.BootID) || !regexp.MustCompile(`^tty[1-9][0-9]*$`).MatchString(p.Console) || len(p.EthernetInterfaces) == 0 {
		return p, errors.New("recovery record is not independent stock boot proof for this machine")
	}
	if _, e = time.Parse(time.RFC3339Nano, p.ObservedAt); e != nil {
		return p, errors.New("invalid recovery observation time")
	}
	files, e := o.artifacts(ctx, r)
	if e != nil {
		return p, e
	}
	if !reflect.DeepEqual(files, p.Files) {
		return p, errors.New("recovery boot artifacts or provider identities changed")
	}
	return p, nil
}

func verifyIndependentWirelessFallback(ctx context.Context, r RecoveryProofRequest) (RecoveryProof, error) {
	o, err := nativeRecoveryObserver()
	if err != nil {
		return RecoveryProof{}, err
	}
	defer o.root.Close()
	return o.verifyFallback(ctx, r)
}

func (o *recoveryObserver) verifyFallback(ctx context.Context, r RecoveryProofRequest) (RecoveryProof, error) {
	proof, err := o.verifyRelease(ctx, r, r.RecoveryRelease)
	if err != nil {
		return proof, err
	}
	cmdline, err := o.text("proc/cmdline")
	if err != nil {
		return proof, err
	}
	imageMatches := false
	for _, field := range strings.Fields(cmdline) {
		if field == "BOOT_IMAGE=/boot/vmlinuz-"+r.RecoveryRelease || field == "BOOT_IMAGE=/vmlinuz-"+r.RecoveryRelease {
			imageMatches = true
		}
	}
	if !imageMatches {
		return proof, errors.New("running fallback boot image does not match the independent recovery proof")
	}
	if _, err = o.ethernet(r); err != nil {
		return proof, err
	}
	for _, module := range r.RequiredNVIDIAModules {
		if _, err = o.root.Stat("sys/module/" + module); err != nil {
			return proof, fmt.Errorf("fallback NVIDIA provider %s is not loaded", module)
		}
	}
	return proof, nil
}
