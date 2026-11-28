//go:build linux

package hostreqkit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"golang.org/x/sys/unix"
)

const openRGBPermissionStateDir = "/var/lib/vrooli/openrgb-device-access"

type openRGBNodeSnapshot struct {
	Selector       OpenRGBLinuxDevice
	UID, GID, Mode uint32
	ACL            []byte
}
type openRGBPermissionReceipt struct {
	Plan         OpenRGBPermissionPlan
	Phase        string
	PreviousRule []byte
	PreviousID   string
	Snapshots    []openRGBNodeSnapshot
	Baselines    []openRGBNodeSnapshot
	Removed      []openRGBNodeSnapshot
}

type openRGBPermissionExecutor struct {
	rule, state, boundary string
	rootUID               uint32
	isRoot                func() bool
	operatorUID           func() (uint32, error)
	now                   func() time.Time
	machine               func() (string, error)
	resolve               func(context.Context, []OpenRGBLinuxDevice) ([]OpenRGBLinuxNode, error)
	resolvePresent        func(context.Context, []OpenRGBLinuxDevice) ([]OpenRGBLinuxNode, error)
	run                   func(context.Context, ...string) error
	snapshot              func(OpenRGBLinuxNode) (openRGBNodeSnapshot, error)
	restore               func(context.Context, OpenRGBLinuxNode, openRGBNodeSnapshot) error
}

func nativeOpenRGBPermissionExecutor() openRGBPermissionExecutor {
	return openRGBPermissionExecutor{rule: OpenRGBLinuxRulePath, state: openRGBPermissionStateDir, boundary: "/", isRoot: func() bool { return os.Geteuid() == 0 }, operatorUID: func() (uint32, error) {
		if os.Geteuid() != 0 {
			return uint32(os.Geteuid()), nil
		}
		if uid, _, ok := InvokingUserIDs(); ok {
			return uint32(uid), nil
		}
		return 0, errors.New("permission apply requires the invoking unprivileged operator identity")
	}, now: time.Now, machine: func() (string, error) { return readOpenRGBText("/etc/machine-id") }, resolve: ResolveLinuxOpenRGBDevices, resolvePresent: resolvePresentOpenRGBNodes, snapshot: snapshotOpenRGBNode, restore: restoreOpenRGBNode, run: func(ctx context.Context, args ...string) error {
		_, err := CombinedOutputContextFn(ctx, "udevadm", args...)
		return err
	}}
}

func PlanLinuxOpenRGBDeviceAccess(ctx context.Context, request OpenRGBPermissionRequest) (OpenRGBPermissionPlan, error) {
	if !linuxOpenRGBPermissionArchitectureSupported(runtime.GOARCH) {
		return OpenRGBPermissionPlan{}, ErrUnsupportedPlatform
	}
	return nativeOpenRGBPermissionExecutor().plan(ctx, request)
}

func ApplyLinuxOpenRGBDeviceAccess(ctx context.Context, plan OpenRGBPermissionPlan) (OpenRGBPermissionResult, error) {
	if !linuxOpenRGBPermissionArchitectureSupported(runtime.GOARCH) {
		return OpenRGBPermissionResult{}, ErrUnsupportedPlatform
	}
	return nativeOpenRGBPermissionExecutor().apply(ctx, plan)
}

func RollbackLinuxOpenRGBDeviceAccess(ctx context.Context, planID string) (OpenRGBPermissionResult, error) {
	if !linuxOpenRGBPermissionArchitectureSupported(runtime.GOARCH) {
		return OpenRGBPermissionResult{}, ErrUnsupportedPlatform
	}
	return nativeOpenRGBPermissionExecutor().rollback(ctx, planID)
}

func (e openRGBPermissionExecutor) plan(ctx context.Context, r OpenRGBPermissionRequest) (OpenRGBPermissionPlan, error) {
	var plan OpenRGBPermissionPlan
	if err := ctx.Err(); err != nil {
		return plan, err
	}
	uid, err := e.operatorUID()
	if err != nil || uid == 0 || uid != r.OwnerUID {
		return plan, errors.Join(errors.New("selected permission owner must be the invoking unprivileged operator"), err)
	}
	machine, err := e.machine()
	if err != nil {
		return plan, err
	}
	p := OpenRGBLinuxPolicy{MachineID: machine, ArtifactSHA256: r.ArtifactSHA256, DetectorSHA256: r.DetectorSHA256, OwnerUID: r.OwnerUID, Devices: slices.Clone(r.Devices)}
	rules, err := openRGBPermissionRules(p)
	if err != nil {
		return plan, err
	}
	p.RuleSHA256 = openRGBPermissionDigest([]byte(rules))
	if _, err = p.ID(); err != nil {
		return plan, err
	}
	nodes, err := e.resolve(ctx, p.Devices)
	if err != nil {
		return plan, err
	}
	previous, err := e.readRule()
	if err != nil {
		return plan, err
	}
	if previous != nil && !strings.HasPrefix(string(previous), openRGBPermissionHeader) {
		return plan, errors.New("existing udev rule is foreign; preserved")
	}

	now := e.now().UTC()
	plan = OpenRGBPermissionPlan{Request: r, Policy: p, RulePath: e.rule, RuleContent: rules, PreviousRuleSHA256: openRGBPreviousRuleDigest(previous), CreatedAt: now, ExpiresAt: now.Add(openRGBPermissionLifetime), Nodes: nodes}
	plan.ID, err = plan.identity()
	return plan, err
}

func openRGBPreviousRuleDigest(data []byte) string {
	if data == nil {
		return ""
	}
	return openRGBPermissionDigest(data)
}

func openRGBPermissionDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func (e openRGBPermissionExecutor) validatePlan(ctx context.Context, p OpenRGBPermissionPlan) error {
	id, err := p.identity()
	if err != nil || id != p.ID || !openRGBHex(p.ID, 64) || p.RulePath != e.rule {
		return errors.New("permission plan identity or fixed destination differs")
	}
	uid, err := e.operatorUID()
	if err != nil || uid == 0 || uid != p.Request.OwnerUID || uid != p.Policy.OwnerUID {
		return errors.Join(errors.New("permission plan belongs to another operator"), err)
	}
	machine, err := e.machine()
	if err != nil || machine != p.Policy.MachineID {
		return errors.Join(errors.New("permission plan belongs to another machine"), err)
	}
	if p.Request.ArtifactSHA256 != p.Policy.ArtifactSHA256 || p.Request.DetectorSHA256 != p.Policy.DetectorSHA256 || !slices.Equal(p.Request.Devices, p.Policy.Devices) {
		return errors.New("permission plan request differs from policy")
	}
	rules, err := openRGBPermissionRules(p.Policy)
	if err != nil || rules != p.RuleContent || openRGBPermissionDigest([]byte(rules)) != p.Policy.RuleSHA256 {
		return errors.Join(errors.New("permission plan differs from qualified static recipe"), err)
	}
	if _, err = p.Policy.ID(); err != nil {
		return err
	}
	if !p.ExpiresAt.Equal(p.CreatedAt.Add(openRGBPermissionLifetime)) || p.CreatedAt.After(e.now().Add(time.Second)) {
		return errors.New("permission plan has invalid lifetime")
	}
	return ctx.Err()
}

func (e openRGBPermissionExecutor) apply(ctx context.Context, p OpenRGBPermissionPlan) (OpenRGBPermissionResult, error) {
	result := OpenRGBPermissionResult{PlanID: p.ID, Policy: p.Policy}
	if !e.isRoot() {
		return result, ErrElevationRequired
	}
	if err := e.validatePlan(ctx, p); err != nil {
		return result, err
	}
	release, err := e.lock(ctx)
	if err != nil {
		return result, err
	}
	defer release()
	receipt, err := e.readReceipt(p.ID)
	if err == nil {
		if receipt.Plan.ID != p.ID {
			return result, errors.New("retained permission receipt differs")
		}
		if receipt.Phase == "rolled_back" {
			return result, errors.New("permission plan was rolled back; review a new plan")
		}
		if receipt.Phase == "rolling_back" {
			return result, errors.New("permission rollback is pending; finish rollback before applying a new plan")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, err
	} else {
		if !e.now().Before(p.ExpiresAt) {
			return result, errors.New("permission plan expired; review a fresh plan")
		}
		previous, err := e.readRule()
		if err != nil {
			return result, err
		}
		if openRGBPreviousRuleDigest(previous) != p.PreviousRuleSHA256 {
			return result, errors.New("udev rule changed after planning; preserved")
		}
		receipt = openRGBPermissionReceipt{Plan: p, Phase: "applying", PreviousRule: previous}
		var prior openRGBPermissionReceipt
		if previous != nil {
			prior, err = e.currentReceipt(previous)
			if err != nil {
				return result, err
			}
			receipt.PreviousID = prior.Plan.ID
			for _, d := range prior.Plan.Policy.Devices {
				if !slices.Contains(p.Policy.Devices, d) {
					i := slices.IndexFunc(prior.Baselines, func(s openRGBNodeSnapshot) bool { return s.Selector == d })
					if i < 0 {
						return result, errors.New("previous permission grant lacks original node recovery metadata")
					}
					receipt.Removed = append(receipt.Removed, prior.Baselines[i])
				}
			}
		}
		nodes, err := e.resolve(ctx, p.Policy.Devices)
		if err != nil {
			return result, err
		}
		extra, err := e.resolvePresent(ctx, selectorsForSnapshots(receipt.Removed))
		if err != nil {
			return result, err
		}
		nodes = append(nodes, extra...)
		for _, n := range nodes {
			snapshot, err := e.snapshot(n)
			if err != nil {
				return result, err
			}
			receipt.Snapshots = append(receipt.Snapshots, snapshot)
		}
		for _, d := range p.Policy.Devices {
			baselineIndex := slices.IndexFunc(prior.Baselines, func(s openRGBNodeSnapshot) bool { return s.Selector == d })
			if baselineIndex >= 0 {
				receipt.Baselines = append(receipt.Baselines, prior.Baselines[baselineIndex])
				continue
			}
			i := slices.IndexFunc(receipt.Snapshots, func(s openRGBNodeSnapshot) bool { return s.Selector == d })
			if i < 0 {
				return result, errors.New("selected permission node has no rollback snapshot")
			}
			receipt.Baselines = append(receipt.Baselines, receipt.Snapshots[i])
		}

		result.Pending = true
		if err = e.writeReceipt(receipt); err != nil {
			return result, err
		}
	}
	result.Pending = true
	if err = e.checkLineage(receipt); err != nil {
		return result, err
	}
	current, err := e.readRule()
	if err != nil {
		return result, err
	}
	if string(current) != p.RuleContent && openRGBPreviousRuleDigest(current) != p.PreviousRuleSHA256 {
		return result, errors.New("udev rule changed during permission transaction; preserved")
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = e.writeFile(e.rule, []byte(p.RuleContent), 0o644); err != nil {
		return result, err
	}
	affected := append(slices.Clone(p.Policy.Devices), selectorsForSnapshots(receipt.Removed)...)
	if err = e.activatePresent(ctx, affected); err != nil {
		return result, fmt.Errorf("permission rule retained pending targeted activation: %w", err)
	}
	removedNodes, resolveErr := e.resolvePresent(ctx, selectorsForSnapshots(receipt.Removed))
	if resolveErr != nil {
		return result, resolveErr
	}
	for _, n := range removedNodes {
		idx := slices.IndexFunc(receipt.Removed, func(s openRGBNodeSnapshot) bool { return s.Selector == n.Selector })
		if err = e.restore(ctx, n, receipt.Removed[idx]); err != nil {
			return result, err
		}
	}
	if err = e.verifyGranted(ctx, p.Policy, receipt.Snapshots); err != nil {
		return result, err
	}
	if err = e.writeFile(filepath.Join(e.state, "current"), []byte(p.ID+"\n"), 0o600); err != nil {
		return result, err
	}
	receipt.Phase = "applied"
	if err = e.writeReceipt(receipt); err != nil {
		return result, err
	}
	result.ActivePlanID = p.ID
	active := p.Policy
	result.ActivePolicy = &active
	result.Applied, result.Pending = true, false
	nodes, err := e.resolve(ctx, p.Policy.Devices)
	if err != nil {
		return result, err
	}
	for _, n := range nodes {
		result.Nodes = append(result.Nodes, n.Node)
	}
	return result, nil
}

func (e openRGBPermissionExecutor) rollback(ctx context.Context, id string) (OpenRGBPermissionResult, error) {
	result := OpenRGBPermissionResult{PlanID: id}
	if !e.isRoot() {
		return result, ErrElevationRequired
	}
	if !openRGBHex(id, 64) {
		return result, errors.New("invalid permission plan ID")
	}
	release, err := e.lock(ctx)
	if err != nil {
		return result, err
	}
	defer release()
	r, err := e.readReceipt(id)
	if err != nil {
		return result, err
	}
	result.Policy = r.Plan.Policy
	if err = e.validatePlan(ctx, r.Plan); err != nil {
		return result, err
	}
	if r.Phase == "rolled_back" {
		result.RolledBack = true
		err = e.fillActive(&result)
		return result, err
	}
	if err = e.checkLineage(r); err != nil {
		return result, err
	}
	current, err := e.readRule()
	if err != nil {
		return result, err
	}
	if string(current) != r.Plan.RuleContent && string(current) != string(r.PreviousRule) {
		return result, errors.New("foreign udev rule replaced permission transaction; preserved")
	}
	result.Pending = true
	r.Phase = "rolling_back"
	if err = e.writeReceipt(r); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if len(r.PreviousRule) == 0 {
		err = e.removeFile(e.rule)
	} else {
		err = e.writeFile(e.rule, r.PreviousRule, 0o644)
	}
	if err != nil {
		return result, err
	}
	affected := append(slices.Clone(r.Plan.Policy.Devices), selectorsForSnapshots(r.Removed)...)
	if err = e.activatePresent(ctx, affected); err != nil {
		return result, fmt.Errorf("permission rollback retained pending targeted activation: %w", err)
	}
	nodes, err := e.resolvePresent(ctx, affected)
	if err != nil {
		return result, err
	}
	for _, n := range nodes {
		idx := slices.IndexFunc(r.Snapshots, func(s openRGBNodeSnapshot) bool { return s.Selector == n.Selector })
		if idx < 0 {
			continue
		} // A previously absent removed route is restored by the prior guarded rule.
		if err = e.restore(ctx, n, r.Snapshots[idx]); err != nil {
			return result, err
		}
	}
	if r.PreviousID == "" {
		err = e.removeFile(filepath.Join(e.state, "current"))
	} else {
		err = e.writeFile(filepath.Join(e.state, "current"), []byte(r.PreviousID+"\n"), 0o600)
	}
	if err != nil {
		return result, err
	}
	r.Phase = "rolled_back"
	if err = e.writeReceipt(r); err != nil {
		return result, err
	}
	result.RolledBack, result.Pending = true, false
	err = e.fillActive(&result)
	return result, err
}

func (e openRGBPermissionExecutor) verifyGranted(ctx context.Context, p OpenRGBLinuxPolicy, prior []openRGBNodeSnapshot) error {
	data, err := e.readRule()
	if err != nil || openRGBPermissionDigest(data) != p.RuleSHA256 {
		return errors.Join(errors.New("installed root-owned udev rule readback differs"), err)
	}
	nodes, err := e.resolve(ctx, p.Devices)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		s, err := e.snapshot(n)
		if err != nil {
			return err
		}
		var original *openRGBNodeSnapshot
		for i := range prior {
			if prior[i].Selector == n.Selector {
				if original != nil {
					return errors.New("selected node has ambiguous trusted permission snapshots")
				}
				original = &prior[i]
			}
		}
		if original == nil {
			return errors.New("selected node has no trusted permission snapshot")
		}
		if err := verifyOpenRGBGrantedNode(s, *original, p.OwnerUID); err != nil {
			return fmt.Errorf("selected node %s permission readback: %w", n.Node, err)
		}
	}
	return nil
}

func (e openRGBPermissionExecutor) qualify(path string) error {
	if err := ValidateManagedFilePath(path, nil); err != nil {
		return err
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		_, err := os.Lstat(dir)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			var st unix.Stat_t
			if err = unix.Lstat(dir, &st); err != nil {
				return err
			}
			if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != e.rootUID || st.Mode&0o022 != 0 {
				return fmt.Errorf("permission destination ancestor %s is not protected root-owned directory", dir)
			}
		}
		if dir == e.boundary || filepath.Dir(dir) == dir {
			break
		}
	}
	return nil
}

func (e openRGBPermissionExecutor) readFile(path string) ([]byte, error) {
	if err := e.qualify(path); err != nil {
		return nil, err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil {
		return nil, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != e.rootUID || st.Mode&0o022 != 0 {
		return nil, errors.New("permission managed file lacks root ownership/write protection")
	}
	b, err := io.ReadAll(io.LimitReader(f, 256<<10))
	if err != nil {
		return nil, err
	}
	if len(b) >= 256<<10 {
		return nil, errors.New("permission state exceeds size bound")
	}
	return b, nil
}

func (e openRGBPermissionExecutor) readRule() ([]byte, error) {
	b, err := e.readFile(e.rule)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

func (e openRGBPermissionExecutor) currentReceipt(rule []byte) (openRGBPermissionReceipt, error) {
	b, err := e.readFile(filepath.Join(e.state, "current"))
	if err != nil {
		return openRGBPermissionReceipt{}, errors.Join(errors.New("existing udev rule has no trusted owner receipt; preserved"), err)
	}
	r, err := e.readReceipt(strings.TrimSpace(string(b)))
	if err != nil {
		return r, err
	}
	if r.Phase != "applied" || string(rule) != r.Plan.RuleContent {
		return r, errors.New("existing udev rule differs from trusted owner receipt; preserved")
	}
	return r, nil
}

func (e openRGBPermissionExecutor) readReceipt(id string) (openRGBPermissionReceipt, error) {
	var r openRGBPermissionReceipt
	if !openRGBHex(id, 64) {
		return r, errors.New("invalid receipt ID")
	}
	b, err := e.readFile(filepath.Join(e.state, id+".json"))
	if err != nil {
		return r, err
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err = d.Decode(&r); err != nil {
		return r, err
	}
	if d.Decode(new(any)) != io.EOF {
		return r, errors.New("trailing permission receipt data")
	}
	if r.Plan.ID != id {
		return r, errors.New("permission receipt identity differs")
	}
	return r, nil
}

func (e openRGBPermissionExecutor) writeReceipt(r openRGBPermissionReceipt) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(b)+1 >= 256<<10 {
		return errors.New("permission receipt exceeds recovery bound")
	}
	return e.writeFile(filepath.Join(e.state, r.Plan.ID+".json"), append(b, '\n'), 0o600)
}

func (e openRGBPermissionExecutor) writeFile(path string, data []byte, mode os.FileMode) error {
	if err := e.qualify(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := e.qualify(path); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".vrooli-openrgb-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = e.qualify(path); err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	return syncOpenRGBPermissionDir(filepath.Dir(path))
}

func (e openRGBPermissionExecutor) removeFile(path string) error {
	if err := e.qualify(path); err != nil {
		return err
	}
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return syncOpenRGBPermissionDir(filepath.Dir(path))
}

func syncOpenRGBPermissionDir(path string) error {
	for {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if err = errors.Join(f.Sync(), f.Close()); err != nil {
			return err
		}
		if filepath.Dir(path) == path {
			return nil
		}
		path = filepath.Dir(path)
	}
}

func (e openRGBPermissionExecutor) lock(ctx context.Context) (func(), error) {
	path := filepath.Join(e.state, "operation.lock")
	if err := e.qualify(path); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(e.state, 0o700); err != nil {
		return nil, err
	}
	if err := e.qualify(path); err != nil {
		return nil, err
	}
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil || st.Uid != e.rootUID || st.Mode&0o022 != 0 || st.Mode&unix.S_IFMT != unix.S_IFREG {
		f.Close()
		return nil, errors.Join(errors.New("untrusted permission operation lock"), err)
	}
	release, err := platform.LockFileContext(ctx, f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return func() { release(); f.Close() }, nil
}

func snapshotOpenRGBNode(n OpenRGBLinuxNode) (openRGBNodeSnapshot, error) {
	s := openRGBNodeSnapshot{Selector: n.Selector}
	var st unix.Stat_t
	if err := unix.Lstat(n.Node, &st); err != nil {
		return s, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFCHR || fmt.Sprintf("%d:%d", unix.Major(st.Rdev), unix.Minor(st.Rdev)) != n.MajorMinor {
		return s, errors.New("selected node metadata changed")
	}
	s.UID, s.GID, s.Mode = st.Uid, st.Gid, st.Mode&0o7777
	size, err := unix.Lgetxattr(n.Node, "system.posix_acl_access", nil)
	if !errors.Is(err, unix.ENODATA) && !errors.Is(err, unix.EOPNOTSUPP) {
		if err != nil {
			return s, err
		}
		if size > 64<<10 {
			return s, errors.New("selected node ACL exceeds bound")
		}
		s.ACL = make([]byte, size)
		if _, err = unix.Lgetxattr(n.Node, "system.posix_acl_access", s.ACL); err != nil {
			return s, err
		}
	}

	var after unix.Stat_t
	if err = unix.Lstat(n.Node, &after); err != nil {
		return s, err
	}
	if st.Dev != after.Dev || st.Ino != after.Ino || st.Rdev != after.Rdev || st.Mode != after.Mode || st.Uid != after.Uid || st.Gid != after.Gid || st.Ctim != after.Ctim {
		return s, errors.New("selected node changed during permission snapshot")
	}
	return s, nil
}

func restoreOpenRGBNode(ctx context.Context, n OpenRGBLinuxNode, s openRGBNodeSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fd, err := unix.Open(n.Node, unix.O_PATH|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFCHR || fmt.Sprintf("%d:%d", unix.Major(st.Rdev), unix.Minor(st.Rdev)) != n.MajorMinor {
		return errors.New("rollback selected node identity changed")
	}
	resolved, resolveErr := ResolveLinuxOpenRGBDevices(ctx, []OpenRGBLinuxDevice{n.Selector})
	if resolveErr != nil || len(resolved) != 1 || resolved[0].Node != n.Node || resolved[0].SysPath != n.SysPath || resolved[0].MajorMinor != n.MajorMinor {
		return errors.Join(errors.New("rollback physical route changed after pinning"), resolveErr)
	}
	var current unix.Stat_t
	if err = unix.Lstat(n.Node, &current); err != nil {
		return err
	}
	if current.Dev != st.Dev || current.Ino != st.Ino || current.Rdev != st.Rdev {
		return errors.New("rollback node was replaced after pinning")
	}
	if err = unix.Fchownat(fd, "", int(s.UID), int(s.GID), unix.AT_EMPTY_PATH); err != nil {
		return err
	}
	if err = unix.Fchmodat(fd, "", s.Mode, unix.AT_EMPTY_PATH); err != nil {
		return err
	}
	pinned := fmt.Sprintf("/proc/self/fd/%d", fd)
	if len(s.ACL) > 0 {
		err = unix.Setxattr(pinned, "system.posix_acl_access", s.ACL, 0)
	} else {
		err = unix.Removexattr(pinned, "system.posix_acl_access")
		if errors.Is(err, unix.ENODATA) || errors.Is(err, unix.EOPNOTSUPP) {
			err = nil
		}
	}
	if err != nil {
		return err
	}
	got, err := snapshotOpenRGBNode(n)
	if err != nil {
		return err
	}
	if got.UID != s.UID || got.GID != s.GID || got.Mode != s.Mode || !slices.Equal(got.ACL, s.ACL) {
		return errors.New("selected node rollback readback differs")
	}
	return nil
}

// OpenRGBPermissionPersistentTypes exposes the exact durable codec roots to the
// existing host resource contract inventory without creating another owner.
func OpenRGBPermissionPersistentTypes() []reflect.Type {
	return []reflect.Type{reflect.TypeFor[openRGBPermissionReceipt]()}
}

func resolvePresentOpenRGBNodes(ctx context.Context, devices []OpenRGBLinuxDevice) ([]OpenRGBLinuxNode, error) {
	all, err := DiscoverLinuxOpenRGBDevices(ctx)
	if err != nil {
		return nil, err
	}
	var nodes []OpenRGBLinuxNode
	for _, d := range devices {
		var found bool
		for _, n := range all {
			if n.Selector == d {
				if found {
					return nil, errors.New("selected route is ambiguous during permission activation")
				}
				nodes = append(nodes, n)
				found = true
			}
		}
	}
	return nodes, nil
}

func (e openRGBPermissionExecutor) activatePresent(ctx context.Context, ds []OpenRGBLinuxDevice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := e.run(ctx, "control", "--reload-rules"); err != nil {
		return err
	}
	for _, d := range ds {
		nodes, err := e.resolvePresent(ctx, []OpenRGBLinuxDevice{d})
		if err != nil {
			return err
		}
		for _, n := range nodes {
			if err = e.run(ctx, "trigger", "--action=change", "--settle", n.SysPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e openRGBPermissionExecutor) currentID() (string, error) {
	b, err := e.readFile(filepath.Join(e.state, "current"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(b))
	if !openRGBHex(id, 64) {
		return "", errors.New("invalid current permission transaction")
	}
	return id, nil
}

func (e openRGBPermissionExecutor) checkLineage(r openRGBPermissionReceipt) error {
	current, err := e.currentID()
	if err != nil {
		return err
	}
	if current == r.Plan.ID {
		return nil
	}
	if (r.Phase == "applying" || r.Phase == "rolling_back") && current == r.PreviousID {
		return nil
	}
	return errors.New("permission transaction was superseded; current grant preserved")
}

func (e openRGBPermissionExecutor) fillActive(result *OpenRGBPermissionResult) error {
	id, err := e.currentID()
	if err != nil {
		return err
	}
	result.ActivePlanID = id
	if id != "" {
		r, err := e.readReceipt(id)
		if err != nil {
			return err
		}
		if r.Phase != "applied" {
			return errors.New("current permission transaction is pending; preserve existing enrollment until recovery")
		}
		p := r.Plan.Policy
		result.ActivePolicy = &p
	}
	return nil
}

func selectorsForSnapshots(snapshots []openRGBNodeSnapshot) []OpenRGBLinuxDevice {
	out := make([]OpenRGBLinuxDevice, 0, len(snapshots))
	for _, s := range snapshots {
		out = append(out, s.Selector)
	}
	return out
}

// The permission recipe uses Linux sysfs, stat/xattr and udev interfaces on
// both qualified release ABIs. This does not attest a particular host/device.
func linuxOpenRGBPermissionArchitectureSupported(arch string) bool {
	return arch == "amd64" || arch == "arm64"
}
