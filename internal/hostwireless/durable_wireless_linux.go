//go:build linux

package hostwireless

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"golang.org/x/sys/unix"
)

const wirelessJournalDir = "var/lib/vrooli/host-wireless"

type durableFileIdentity struct {
	SHA256 string `json:"sha256,omitempty"`
	Mode   uint32 `json:"mode,omitempty"`
	UID    uint32 `json:"uid,omitempty"`
	GID    uint32 `json:"gid,omitempty"`
	Absent bool   `json:"absent,omitempty"`
}

type durableWirelessFile struct {
	Path   string              `json:"path"`
	Before durableFileIdentity `json:"before"`
	After  durableFileIdentity `json:"after"`
}

type durableWirelessIntent struct {
	Version           int                            `json:"version"`
	ID                string                         `json:"id"`
	Previous          string                         `json:"previous,omitempty"`
	PreviousSHA256    string                         `json:"previous_intent_sha256,omitempty"`
	PreviousPublished bool                           `json:"previous_published,omitempty"`
	RootDevice        uint64                         `json:"root_device"`
	RootInode         uint64                         `json:"root_inode"`
	Plan              ActivationPlan                 `json:"plan"`
	CertificateSHA256 string                         `json:"certificate_sha256"`
	Files             []durableWirelessFile          `json:"files"`
	Firmware          []FileIdentity                 `json:"firmware,omitempty"`
	Directories       []string                       `json:"created_directories"`
	Protected         map[string]durableFileIdentity `json:"protected"`
	PhysicalPaths     map[string]string              `json:"physical_paths,omitempty"`
	Context           *wirelessJournalContext        `json:"context,omitempty"`
}

// durableWireless is the production file-transaction engine. Construction is
// internal and requires an authenticated/indexed generation and a separately
// owned private target. No live-root/elevation adapter or device effect exists.
// Close releases descriptors/lock; journals and undo bytes survive it.
type durableWireless struct {
	root      *os.Root
	fd        *os.File
	release   func()
	intent    durableWirelessIntent
	intentSHA string
	phase     string
	// Fault seam deliberately surrounds durable operations, including the gap
	// after rename and before directory fsync. Errors retain recoverable state.
	fault               func(string) error
	paths               map[string]string
	admission           *wirelessRootAdmission
	context             *wirelessJournalContext
	readOnly            bool
	radio               *wirelessRadioSystem
	deferParent         bool
	independentRecovery bool
}

type durableActive struct{ ID, IntentSHA256 string }

func openDurableWireless(ctx context.Context, root *os.Root) (*durableWireless, error) {
	return openAdmittedDurableWireless(ctx, root, nil)
}
func openAdmittedDurableWireless(ctx context.Context, root *os.Root, admission *wirelessRootAdmission, readOnly ...bool) (*durableWireless, error) {
	if root == nil || (path.Clean(root.Name()) == "/" && admission == nil) {
		return nil, errors.New("wireless transaction requires an owned private root")
	}
	fd, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	t := &durableWireless{root: root, fd: fd, admission: admission}
	t.readOnly = len(readOnly) > 0 && readOnly[0]
	if admission != nil {
		t.paths = maps.Clone(admission.paths)
		t.independentRecovery = admission.independentRecovery
	}
	var rootStat, hostStat unix.Stat_t
	if err = unix.Fstat(int(fd.Fd()), &rootStat); err != nil {
		fd.Close()
		return nil, err
	}
	if err = unix.Stat("/", &hostStat); err != nil {
		fd.Close()
		return nil, err
	}
	if rootStat.Dev == hostStat.Dev && rootStat.Ino == hostStat.Ino && (os.Geteuid() != 0 || !admission.permits(rootStat)) {
		fd.Close()
		return nil, errors.New("live host root has no admitted wireless adapter")
	}
	if rootStat.Uid != uint32(os.Geteuid()) || rootStat.Mode&022 != 0 {
		fd.Close()
		return nil, errors.New("wireless target root is not privately owned")
	}
	var stateErr error
	if t.readOnly {
		stateErr = t.qualifyStateDirs()
	} else {
		stateErr = t.makeStateDirs()
	}
	if stateErr != nil {
		fd.Close()
		return nil, stateErr
	}
	flags, mode := unix.O_RDWR|unix.O_CREAT, uint32(0600)
	if t.readOnly {
		flags, mode = unix.O_RDONLY, 0
	}
	lock, err := t.open(path.Join(wirelessJournalDir, "operation.lock"), flags, mode)
	if err != nil {
		fd.Close()
		return nil, err
	}
	var lockStat unix.Stat_t
	if err = unix.Fstat(int(lock.Fd()), &lockStat); err != nil || lockStat.Uid != uint32(os.Geteuid()) || lockStat.Mode&077 != 0 || lockStat.Nlink != 1 || lockStat.Mode&unix.S_IFMT != unix.S_IFREG {
		lock.Close()
		fd.Close()
		return nil, errors.New("wireless operation lock is not private")
	}
	unlock, err := platform.LockFileContext(ctx, lock)
	if err != nil {
		lock.Close()
		fd.Close()
		return nil, err
	}
	t.release = func() { unlock(); lock.Close(); fd.Close() }
	data, err := t.readState(path.Join(wirelessJournalDir, "active.json"), 4096)
	if errors.Is(err, fs.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		t.close()
		return nil, err
	}
	var active durableActive
	if err = strictWirelessJSON(data, &active); err != nil || !validJournalID(active.ID) || !validSHA(active.IntentSHA256) {
		t.close()
		return nil, errors.New("invalid wireless journal lineage")
	}
	if err = t.qualifyTransactionDir(active.ID); err != nil {
		t.close()
		return nil, err
	}
	data, err = t.readState(path.Join(wirelessJournalDir, active.ID, "intent.json"), 8<<20)
	if err != nil || digest(data) != active.IntentSHA256 {
		t.close()
		return nil, errors.New("wireless journal intent identity differs")
	}
	if err = strictWirelessJSON(data, &t.intent); err != nil {
		t.close()
		return nil, err
	}
	t.intentSHA = active.IntentSHA256
	if t.intent.ID != active.ID {
		t.close()
		return nil, errors.New("wireless active lineage differs")
	}
	data, err = t.readState(t.statePath("phase"), 64)
	if err != nil {
		t.close()
		return nil, err
	}
	t.phase = string(data)
	if !slices.Contains([]string{preparedArtifactState, "publishing", "published", "rolling_back", "rolled_back"}, t.phase) {
		t.close()
		return nil, errors.New("unknown wireless transaction phase")
	}
	if err = t.validate(ctx); err != nil {
		t.close()
		return nil, err
	}
	return t, nil
}

func (t *durableWireless) close() error {
	if t.release != nil {
		t.release()
		t.release = nil
	}
	return nil
}

// begin accepts only a fully authenticated signed/native-indexed generation.
// A new generation may follow only a completely rolled-back predecessor.
func (t *durableWireless) begin(ctx context.Context, indexed *indexedActivation, cert *x509.Certificate, firmware ...FirmwarePreflight) error {
	if t.intent.ID != "" && t.phase != "rolled_back" && t.phase != "published" {
		return errors.New("wireless lineage has an unfinished or published transaction")
	}
	if t.intent.ID != "" {
		if err := t.validate(ctx); err != nil {
			return err
		}
	}
	if indexed == nil || cert == nil || indexed.candidate == nil || digest(cert.Raw) != indexed.certSHA {
		return errors.New("authenticated indexed generation required")
	}
	if t.phase == "published" {
		if indexed.plan.Profile == t.intent.Plan.Profile {
			return errors.New("an exact profile overlay is already published")
		}
		for _, file := range t.intent.Files {
			got, err := t.identity(file.Path)
			if err != nil || got != file.After {
				return errors.New("previous wireless publication is incomplete")
			}
		}
	}
	if err := indexed.verifySelected(ctx); err != nil {
		return err
	}
	firmwarePins, firmwareBytes, err := readDurableFirmware(ctx, indexed.plan.Profile, firmware)
	if err != nil {
		return err
	}
	// The signed files are stored under the exact overlay, not at root.
	if _, err := verifySignedFiles(ctx, os.DirFS(path.Join(indexed.candidate.root.Name(), indexed.plan.OverlayDirectory)), indexed.plan.Candidates, cert); err != nil {
		return err
	}
	beforeTree, err := t.snapshotTarget(ctx, indexed)
	if err != nil {
		return err
	}
	for name, want := range indexed.stockTree {
		if got, ok := beforeTree[name]; !ok || got.SHA256 != want.SHA256 || got.Mode.Type() != want.Mode.Type() {
			return fmt.Errorf("target stock differs: %s", name)
		}
	}
	for name := range beforeTree {
		firmwarePath := false
		for _, pin := range firmwarePins {
			if name == pin.Path || strings.HasPrefix(pin.Path, name+"/") {
				firmwarePath = true
			}
		}
		if t.phase == "published" {
			for _, file := range t.intent.Files {
				if name == file.Path || strings.HasPrefix(file.Path, name+"/") {
					firmwarePath = true
				}
			}
		}
		if _, ok := indexed.stockTree[name]; !ok && !strings.HasPrefix(name, "var/") && name != "var" && !firmwarePath {
			return fmt.Errorf("foreign target entry: %s", name)
		}
	}
	previous := t.intent.ID
	var random [16]byte
	if _, err = rand.Read(random[:]); err != nil {
		return err
	}
	intent := durableWirelessIntent{Version: 1, ID: hex.EncodeToString(random[:]), Previous: previous, PreviousSHA256: t.intentSHA, PreviousPublished: t.phase == "published", Plan: indexed.plan, CertificateSHA256: indexed.certSHA, Protected: map[string]durableFileIdentity{}, Firmware: firmwarePins}
	intent.PhysicalPaths = maps.Clone(t.paths)
	intent.Context = t.context
	var st unix.Stat_t
	if err = unix.Fstat(int(t.fd.Fd()), &st); err != nil {
		return err
	}
	intent.RootDevice, intent.RootInode = uint64(st.Dev), st.Ino
	mutations := map[string]bool{}
	for _, module := range indexed.modules {
		mutations[path.Join(indexed.plan.OverlayDirectory, module.Name)] = true
	}
	for _, name := range depmodIndexes {
		mutations[path.Join(abiTree, name)] = true
	}
	for name := range indexed.stockTree {
		if mutations[name] {
			continue
		}
		info, err := t.root.Lstat(name)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		identity, err := t.identity(name)
		if err != nil {
			return err
		}
		intent.Protected[name] = identity
	}
	if t.phase == "published" {
		for _, file := range t.intent.Files {
			if !mutations[file.Path] {
				intent.Protected[file.Path] = file.After
			}
		}
	}
	dirs := []string{path.Dir(indexed.plan.OverlayDirectory), indexed.plan.OverlayDirectory}
	if len(firmwarePins) > 0 {
		dirs = append(dirs, "lib/firmware", "lib/firmware/mediatek", "lib/firmware/mediatek/mt7927")
	}
	for _, dir := range dirs {
		f, err := t.open(dir, unix.O_RDONLY|unix.O_DIRECTORY, 0)
		if err == nil {
			f.Close()
			if dir == indexed.plan.OverlayDirectory {
				return errors.New("preexisting wireless overlay")
			}
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		intent.Directories = append(intent.Directories, dir)
	}
	// Modules precede indexes. Reverse rollback restores indexes first.
	var targets []string
	for _, module := range indexed.modules {
		targets = append(targets, path.Join(indexed.plan.OverlayDirectory, module.Name))
	}
	for _, image := range firmwarePins {
		targets = append(targets, image.Path)
	}
	for _, name := range depmodIndexes {
		targets = append(targets, path.Join(abiTree, name))
	}
	t.intent = intent
	if err = t.mkdir(path.Join(wirelessJournalDir, intent.ID)); err != nil {
		return err
	}
	for i, name := range targets {
		before, err := t.identity(name)
		if err != nil {
			return err
		}
		data, isFirmware := firmwareBytes[name]
		if !isFirmware {
			data, err = readSmallRegular(indexed.candidate.root.FS(), name, maxPreservedBytes)
			if err != nil {
				return err
			}
		}
		after := t.ownIdentity(0644)
		after.SHA256 = digest(data)
		entry := durableWirelessFile{Path: name, Before: before, After: after}
		t.intent.Files = append(t.intent.Files, entry)
		if !before.Absent {
			data, err := t.read(name, maxPreservedBytes)
			if err != nil {
				return err
			}
			if err = t.writeState(t.statePath(fmt.Sprintf("old-%d", i)), data); err != nil {
				return err
			}
		}
		if err = t.writeState(t.statePath(fmt.Sprintf("new-%d", i)), data); err != nil {
			return err
		}
	}
	if err = t.writeState(t.statePath("certificate.der"), cert.Raw); err != nil {
		return err
	}
	data, err := json.Marshal(t.intent)
	if err != nil {
		return err
	}
	t.intentSHA = digest(data)
	if err = t.writeState(t.statePath("intent.json"), data); err != nil {
		return err
	}
	if err = t.setPhase(preparedArtifactState); err != nil {
		return err
	}
	active, _ := json.Marshal(durableActive{t.intent.ID, t.intentSHA})
	return t.writeState(path.Join(wirelessJournalDir, "active.json"), active)
}

func (t *durableWireless) validate(ctx context.Context) error {
	r := t.intent
	activeBytes, err := t.readState(path.Join(wirelessJournalDir, "active.json"), 4096)
	if err != nil {
		return err
	}
	var active durableActive
	if err = strictWirelessJSON(activeBytes, &active); err != nil || active.ID != r.ID || active.IntentSHA256 != t.intentSHA {
		return errors.New("wireless transaction superseded or lineage changed")
	}
	var st unix.Stat_t
	if err := unix.Fstat(int(t.fd.Fd()), &st); err != nil {
		return err
	}
	if r.Version != 1 || !validJournalID(r.ID) || r.RootDevice != uint64(st.Dev) || r.RootInode != st.Ino || r.Plan.KernelRelease != nobleABI || r.Plan.OverlayDirectory != overlayDirectory(r.Plan.Profile) {
		return errors.New("wireless journal root or profile differs")
	}
	if !maps.Equal(r.PhysicalPaths, t.paths) {
		return errors.New("wireless physical namespace changed")
	}
	if !validSHA(r.Plan.BundleReceiptSHA256) || !bundleID.MatchString(r.Plan.Bundle) || !validSHA(r.Plan.KernelSymbolsSHA256) {
		return errors.New("wireless journal lacks exact preparation identity")
	}
	if r.Previous != "" && !validJournalID(r.Previous) {
		return errors.New("invalid previous wireless lineage")
	}
	if r.Previous == "" && (r.PreviousSHA256 != "" || r.PreviousPublished) {
		return errors.New("wireless previous lineage metadata without predecessor")
	}
	if r.Previous != "" {
		if err = t.qualifyTransactionDir(r.Previous); err != nil {
			return err
		}
		previous, err := t.readState(path.Join(wirelessJournalDir, r.Previous, "intent.json"), 8<<20)
		if err != nil || digest(previous) != r.PreviousSHA256 {
			return errors.New("wireless previous lineage differs")
		}
	}
	spec, ok := preparationProfiles[r.Plan.Profile]
	if !ok || len(r.Files) != len(spec.modules)+len(depmodIndexes)+len(r.Firmware) || len(r.Protected) == 0 || len(r.Protected) > 12000 {
		return errors.New("invalid wireless journal closure")
	}
	if err := verifyPlanModules(r.Plan.Originals, r.Plan.Candidates, spec); err != nil {
		return err
	}
	if r.Plan.KernelSymbolsSHA256 != r.Protected[abiTree+"/build/Module.symvers"].SHA256 {
		return errors.New("wireless retained ABI symbols differ")
	}
	for _, file := range r.Plan.Originals {
		if id, ok := r.Protected[strings.TrimPrefix(file.Path, "/")]; !ok || id.SHA256 != file.SHA256 {
			return errors.New("wireless stock provider protection missing")
		}
	}
	protectedPaths := append(slices.Clone(r.Plan.Originals), r.Plan.Preserved...)
	for i := range protectedPaths {
		protectedPaths[i].Path = strings.TrimPrefix(protectedPaths[i].Path, "/")
	}
	if err := verifyPreservedProviders(r.Plan, protectedPaths); err != nil {
		return err
	}
	if r.Plan.RecoveryRelease == nobleABI || !nativeName.MatchString(r.Plan.RecoveryRelease) || !slices.Contains(r.Plan.RecoveryCandidates, r.Plan.RecoveryRelease) {
		return errors.New("wireless distinct recovery identity missing")
	}
	required := append(bootFiles(nobleABI), "/boot/vmlinuz-"+r.Plan.RecoveryRelease, "/boot/initrd.img-"+r.Plan.RecoveryRelease)
	if len(r.Plan.WiredInterfaces) > 0 {
		required = append(required, "/proc/net/route")
	}
	for _, name := range required {
		if !validSHA(r.Protected[strings.TrimPrefix(name, "/")].SHA256) {
			return errors.New("wireless protected recovery file missing")
		}
	}
	var expected []string
	for _, module := range spec.modules {
		expected = append(expected, path.Join(r.Plan.OverlayDirectory, module+".ko"))
	}
	if len(r.Firmware) != 0 {
		pins := durableFirmwarePins(r.Plan.Profile)
		if !slices.Equal(pins, r.Firmware) {
			return errors.New("retained firmware closure differs")
		}
		for _, image := range pins {
			expected = append(expected, image.Path)
		}
	}
	for _, name := range depmodIndexes {
		expected = append(expected, path.Join(abiTree, name))
	}
	for i, file := range r.Files {
		if file.Path != expected[i] || !validSHA(file.After.SHA256) || file.After.Mode != 0644 || file.After.UID != uint32(os.Geteuid()) || file.After.GID != uint32(os.Getegid()) || (!file.Before.Absent && !validSHA(file.Before.SHA256)) {
			return errors.New("wireless journal target differs")
		}
		for _, pin := range r.Firmware {
			if file.Path == pin.Path && file.After.SHA256 != pin.SHA256 {
				return errors.New("retained firmware digest differs")
			}
		}
		for _, item := range []struct {
			prefix string
			id     durableFileIdentity
		}{{"new", file.After}, {"old", file.Before}} {
			if item.id.Absent {
				continue
			}
			data, err := t.readState(t.statePath(fmt.Sprintf("%s-%d", item.prefix, i)), maxPreservedBytes)
			if err != nil || digest(data) != item.id.SHA256 {
				return errors.New("wireless retained payload differs")
			}
		}
	}
	allowedDirs := []string{path.Dir(r.Plan.OverlayDirectory), r.Plan.OverlayDirectory}
	if len(r.Firmware) > 0 {
		allowedDirs = append(allowedDirs, "lib/firmware", "lib/firmware/mediatek", "lib/firmware/mediatek/mt7927")
	}
	seenDirs := map[string]bool{}
	for _, dir := range r.Directories {
		if !slices.Contains(allowedDirs, dir) || seenDirs[dir] {
			return errors.New("wireless journal directory differs")
		}
		seenDirs[dir] = true
		opened, err := t.open(dir, unix.O_RDONLY|unix.O_DIRECTORY, 0)
		if err == nil {
			var st unix.Stat_t
			statErr := unix.Fstat(int(opened.Fd()), &st)
			opened.Close()
			if statErr != nil || st.Uid != uint32(os.Geteuid()) || st.Gid != uint32(os.Getegid()) || st.Mode&07777 != 0700 {
				return errors.New("wireless created directory metadata differs")
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	certBytes, err := t.readState(t.statePath("certificate.der"), 64<<10)
	if err != nil || digest(certBytes) != r.CertificateSHA256 {
		return errors.New("retained wireless certificate identity differs")
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}
	if _, err = verifySignedFiles(ctx, durableSignedFiles{t}, r.Plan.Candidates, cert); err != nil {
		return err
	}
	if err = t.verifyProtected(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return t.classify(ctx)
}

func (t *durableWireless) verifyProtected(ctx context.Context) error {
	for name, id := range t.intent.Protected {
		if err := ctx.Err(); err != nil {
			return err
		}
		firmware := false
		for _, profile := range []string{NobleProfile, NobleBluetoothProfile} {
			for _, pin := range durableFirmwarePins(profile) {
				if name == pin.Path && id.SHA256 == pin.SHA256 {
					firmware = true
				}
			}
		}
		if (!safeProtectedPath(name, t.intent.Plan.OverlayDirectory) && !firmware) || !validSHA(id.SHA256) {
			return errors.New("invalid wireless protected path")
		}
		got, err := t.identity(name)
		if name == "proc/net/route" && t.intent.Context != nil && err == nil {
			data, e := t.read(name, 64<<10)
			if e != nil {
				return e
			}
			if !t.independentRecovery && (len(t.intent.Context.EthernetRoutes) == 0 || !slices.Equal(ethernetRouteLines(data, t.intent.Plan.WiredInterfaces), t.intent.Context.EthernetRoutes)) {
				return errors.New("preserved Ethernet routes changed")
			}
			got.SHA256 = id.SHA256 // Additional radio routes do not change the wired guard.
		}
		if err != nil || got != id {
			return fmt.Errorf("wireless protected identity drift: %s", name)
		}
	}
	return nil
}

type durableSignedFiles struct{ transaction *durableWireless }

func (f durableSignedFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	if name != "." {
		return nil, fs.ErrNotExist
	}
	var entries []fs.DirEntry
	for i, file := range f.transaction.intent.Files {
		if !strings.HasPrefix(file.Path, f.transaction.intent.Plan.OverlayDirectory+"/") {
			continue
		}
		opened, err := f.transaction.open(f.transaction.statePath(fmt.Sprintf("new-%d", i)), unix.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}
		info, err := opened.Stat()
		opened.Close()
		if err != nil {
			return nil, err
		}
		entries = append(entries, durableNamedEntry{DirEntry: fs.FileInfoToDirEntry(info), name: path.Base(file.Path)})
	}
	return entries, nil
}

type durableNamedEntry struct {
	fs.DirEntry
	name string
}

func (e durableNamedEntry) Name() string { return e.name }
func (f durableSignedFiles) Open(name string) (fs.File, error) {
	for i, file := range f.transaction.intent.Files {
		if strings.HasPrefix(file.Path, f.transaction.intent.Plan.OverlayDirectory+"/") && path.Base(file.Path) == name {
			return f.transaction.open(f.transaction.statePath(fmt.Sprintf("new-%d", i)), unix.O_RDONLY, 0)
		}
	}
	return nil, fs.ErrNotExist
}

func (t *durableWireless) classify(ctx context.Context) error {
	for _, file := range t.intent.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		got, err := t.identity(file.Path)
		if err != nil {
			return err
		}
		if got != file.Before && got != file.After {
			return fmt.Errorf("foreign wireless target preserved: %s", file.Path)
		}
	}
	// An unexpected sibling cannot become part of a successful overlay.
	directory, err := t.open(t.intent.Plan.OverlayDirectory, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err == nil {
		entries, readErr := directory.ReadDir(-1)
		directory.Close()
		if readErr != nil {
			return readErr
		}
		for _, entry := range entries {
			found := false
			for _, file := range t.intent.Files {
				if file.Path == path.Join(t.intent.Plan.OverlayDirectory, entry.Name()) {
					found = true
				}
			}
			if !found || !entry.Type().IsRegular() {
				return errors.New("foreign wireless overlay entry preserved")
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (t *durableWireless) apply(ctx context.Context) (resultErr error) {
	if t.phase == "rolling_back" || t.phase == "rolled_back" {
		return errors.New("wireless rollback lineage cannot be applied")
	}
	if err := t.validate(ctx); err != nil {
		return err
	}
	defer func() {
		if ctx.Err() != nil {
			cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			resultErr = errors.Join(resultErr, t.rollback(cleanup))
		}
	}()
	if err := t.setPhase("publishing"); err != nil {
		return err
	}
	for _, dir := range t.intent.Directories {
		f, err := t.open(dir, unix.O_RDONLY|unix.O_DIRECTORY, 0)
		if err == nil {
			if err = errors.Join(f.Sync(), f.Close(), t.syncParents(dir)); err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err = t.mkdir(dir); err != nil {
			return err
		}
	}
	for i, file := range t.intent.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := t.classify(ctx); err != nil {
			return err
		}
		if err := t.verifyProtected(ctx); err != nil {
			return err
		}
		got, err := t.identity(file.Path)
		if err != nil {
			return err
		}
		if got == file.After {
			if err = t.syncParents(file.Path); err != nil {
				return err
			}
			continue
		}
		data, err := t.readState(t.statePath(fmt.Sprintf("new-%d", i)), maxPreservedBytes)
		if err != nil {
			return err
		}
		if err = t.atomic(file.Path, data, file.After, got); err != nil {
			return err
		}
	}
	if err := t.validate(ctx); err != nil {
		return err
	}
	for _, file := range t.intent.Files {
		got, err := t.identity(file.Path)
		if err != nil || got != file.After {
			return errors.New("wireless publication did not select every candidate exactly")
		}
	}
	return t.setPhase("published")
}

func (t *durableWireless) rollback(ctx context.Context) error {
	if err := t.validate(ctx); err != nil {
		return err
	}
	if err := t.setPhase("rolling_back"); err != nil {
		return err
	}
	for i := len(t.intent.Files) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		file := t.intent.Files[i]
		got, err := t.identity(file.Path)
		if err != nil {
			return err
		}
		if got == file.Before {
			if err = t.syncParents(file.Path); err != nil {
				return err
			}
			continue
		}
		if got != file.After {
			return fmt.Errorf("foreign wireless rollback target preserved: %s", file.Path)
		}
		if file.Before.Absent {
			if err = t.check("before-unlink:" + file.Path); err != nil {
				return err
			}
			current, err := t.identity(file.Path)
			if err != nil || current != got {
				return errors.New("wireless target changed immediately before rollback removal")
			}
			parent, base, err := t.parent(file.Path)
			if err != nil {
				return err
			}
			err = unix.Unlinkat(int(parent.Fd()), base, 0)
			if err == nil {
				err = t.check("after-unlink:" + file.Path)
			}
			if err == nil {
				err = parent.Sync()
			}
			parent.Close()
			if err != nil {
				return err
			}
		} else {
			data, err := t.readState(t.statePath(fmt.Sprintf("old-%d", i)), maxPreservedBytes)
			if err != nil {
				return err
			}
			if err = t.atomic(file.Path, data, file.Before, got); err != nil {
				return err
			}
		}
	}
	for _, file := range t.intent.Files {
		got, err := t.identity(file.Path)
		if err != nil || got != file.Before {
			return errors.New("wireless rollback originals not restored exactly")
		}
	}
	for i := len(t.intent.Directories) - 1; i >= 0; i-- {
		parent, base, err := t.parent(t.intent.Directories[i])
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		err = unix.Unlinkat(int(parent.Fd()), base, unix.AT_REMOVEDIR)
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		if err == nil {
			err = parent.Sync()
		}
		parent.Close()
		if err != nil {
			return err
		}
	}
	if err := t.setPhase("rolled_back"); err != nil {
		return err
	}
	if t.intent.PreviousPublished && !t.deferParent {
		active, _ := json.Marshal(durableActive{t.intent.Previous, t.intent.PreviousSHA256})
		return t.writeState(path.Join(wirelessJournalDir, "active.json"), active)
	}
	return nil
}

func (t *durableWireless) completeParentLineage() error {
	if !t.intent.PreviousPublished {
		return nil
	}
	active, _ := json.Marshal(durableActive{t.intent.Previous, t.intent.PreviousSHA256})
	return t.writeState(path.Join(wirelessJournalDir, "active.json"), active)
}
