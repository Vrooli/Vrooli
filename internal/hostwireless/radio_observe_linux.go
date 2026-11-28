//go:build linux

package hostwireless

import (
	"bytes"
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

var (
	radioPCIAddress   = regexp.MustCompile(`^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]$`)
	radioUSBAddress   = regexp.MustCompile(`^[1-9][0-9]*-[0-9]+(?:\.[0-9]+)*$`)
	radioUSBInterface = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
)

func selectedUSBInterfaceLink(link, address string) bool {
	for _, component := range strings.Split(link, "/") {
		device, function, ok := strings.Cut(component, ":")
		if ok && device == address && radioUSBInterface.MatchString(function) {
			return true
		}
	}
	return false
}

type wirelessModuleSnapshot struct {
	Name       string            `json:"name"`
	Loaded     bool              `json:"loaded"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// Test injection is private to the owner. Normal commands always use the
// native kernel executor and independently observed sysfs module identities.
type wirelessRadioSystem struct {
	command  func(context.Context, string, ...string) ([]byte, error)
	unload   func(context.Context, string) error
	identity func(context.Context, *durableWireless, string) (bool, bool, bool, error)
	aliases  func(context.Context, *durableWireless) ([]string, error)
}

func radioModuleName(name string) string {
	return strings.ReplaceAll(strings.TrimSuffix(name, ".ko"), "-", "_")
}

func radioModuleNames(profile string) []string {
	var names []string
	for _, name := range preparationProfiles[profile].modules {
		names = append(names, radioModuleName(name))
	}
	return names
}

func radioLeaf(profile string) string {
	if profile == NobleBluetoothProfile {
		return "btusb"
	}
	return "mt7925e"
}

func radioRead(root *os.Root, name string, limit int64) ([]byte, error) {
	f, err := root.OpenFile(name, os.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("radio observation is not a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(data)) > limit {
		return nil, errors.New("radio observation exceeds bound")
	}
	return data, err
}

func moduleSnapshots(root *os.Root, profile string) ([]wirelessModuleSnapshot, error) {
	var snapshots []wirelessModuleSnapshot
	for _, name := range radioModuleNames(profile) {
		snapshot := wirelessModuleSnapshot{Name: name}
		if _, err := root.Stat("sys/module/" + name); errors.Is(err, fs.ErrNotExist) {
			snapshots = append(snapshots, snapshot)
			continue
		} else if err != nil {
			return nil, err
		}
		snapshot.Loaded = true
		snapshot.Parameters = map[string]string{}
		entries, err := fs.ReadDir(root.FS(), "sys/module/"+name+"/parameters")
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if len(entries) > 64 {
			return nil, errors.New("radio parameter count exceeds bound")
		}
		for _, entry := range entries {
			if !nativeName.MatchString(entry.Name()) {
				return nil, errors.New("invalid radio parameter")
			}
			data, err := radioRead(root, "sys/module/"+name+"/parameters/"+entry.Name(), 512)
			if err != nil {
				return nil, err
			}
			value := strings.TrimSpace(string(data))
			if strings.ContainsAny(value, "\x00\r\n") {
				return nil, errors.New("unsupported radio parameter value")
			}
			snapshot.Parameters[entry.Name()] = value
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func loadedModuleNames(root *os.Root) ([]string, error) {
	entries, err := fs.ReadDir(root.FS(), "sys/module")
	if err != nil {
		return nil, err
	}
	if len(entries) > 4096 {
		return nil, errors.New("loaded module inventory exceeds bound")
	}
	var names []string
	for _, entry := range entries {
		if nativeName.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

func (t *durableWireless) radioSystem() *wirelessRadioSystem {
	if t.radio != nil {
		return t.radio
	}
	return nativeWirelessRadioSystem()
}

func candidateModuleBytes(ctx context.Context, t *durableWireless, module string) ([]byte, error) {
	for i, file := range t.intent.Files {
		if strings.HasPrefix(file.Path, t.intent.Plan.OverlayDirectory+"/") && radioModuleName(path.Base(file.Path)) == module {
			return t.readState(t.statePath(fmt.Sprintf("new-%d", i)), maxSignedModuleBytes)
		}
	}
	return nil, errors.New("candidate module not in reviewed journal")
}

func moduleELF(ctx context.Context, data []byte, filename string) (*elf.File, error) {
	suffix := ""
	if strings.HasSuffix(filename, ".zst") {
		suffix = ".zst"
	} else if strings.HasSuffix(filename, ".xz") {
		suffix = ".initrd.xz"
	}
	decoded, err := firmwarePolicyDecode(ctx, data, suffix, maxArtifactBytes)
	if err != nil {
		return nil, err
	}
	return elf.NewFile(bytes.NewReader(decoded))
}

func moduleLoadedMatches(root *os.Root, module string, file *elf.File) (bool, error) {
	defer file.Close()
	if section := file.Section(".note.gnu.build-id"); section != nil {
		expected, err := section.Data()
		if err != nil {
			return false, err
		}
		actual, err := radioRead(root, "sys/module/"+module+"/notes/.note.gnu.build-id", 4096)
		if err == nil {
			return len(expected) > 0 && bytes.Equal(actual, expected), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
	}
	section := file.Section(".modinfo")
	if section == nil {
		return false, errors.New("module lacks measurable build identity")
	}
	data, err := section.Data()
	if err != nil {
		return false, err
	}
	var expected string
	for _, item := range strings.Split(string(data), "\x00") {
		if value, ok := strings.CutPrefix(item, "srcversion="); ok {
			expected = value
		}
	}
	if expected == "" {
		return false, errors.New("module build identity unavailable")
	}
	actual, err := radioRead(root, "sys/module/"+module+"/srcversion", 256)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(actual)) == expected, nil
}

func nativeRadioModuleIdentity(ctx context.Context, t *durableWireless, module string) (loaded, candidate, stock bool, err error) {
	if _, e := t.root.Stat("sys/module/" + module); errors.Is(e, fs.ErrNotExist) {
		return false, false, false, nil
	} else if e != nil {
		return false, false, false, e
	}
	loaded = true
	data, err := candidateModuleBytes(ctx, t, module)
	if err != nil {
		return loaded, false, false, err
	}
	file, err := moduleELF(ctx, data, module+".ko")
	if err != nil {
		return loaded, false, false, err
	}
	candidate, err = moduleLoadedMatches(t.root, module, file)
	if err != nil {
		return loaded, false, false, err
	}
	for i, name := range preparationProfiles[t.intent.Plan.Profile].modules {
		if radioModuleName(name) != module {
			continue
		}
		original := strings.TrimPrefix(t.intent.Plan.Originals[i].Path, "/")
		data, err = t.read(original, maxPreservedBytes)
		if err != nil {
			return loaded, candidate, false, err
		}
		file, err = moduleELF(ctx, data, original)
		if err != nil {
			return loaded, candidate, false, err
		}
		stock, err = moduleLoadedMatches(t.root, module, file)
		return loaded, candidate, stock, err
	}
	return loaded, candidate, false, errors.New("stock module identity is unavailable")
}

func nativeRadioAliases(ctx context.Context, t *durableWireless) ([]string, error) {
	data, err := candidateModuleBytes(ctx, t, radioLeaf(t.intent.Plan.Profile))
	if err != nil {
		return nil, err
	}
	file, err := moduleELF(ctx, data, "candidate.ko")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	section := file.Section(".modinfo")
	if section == nil {
		return nil, errors.New("candidate lacks modalias metadata")
	}
	data, err = section.Data()
	if err != nil {
		return nil, err
	}
	var aliases []string
	for _, entry := range strings.Split(string(data), "\x00") {
		if alias, ok := strings.CutPrefix(entry, "alias="); ok {
			if len(alias) > 512 {
				return nil, errors.New("candidate alias exceeds bound")
			}
			aliases = append(aliases, alias)
		}
	}
	if len(aliases) == 0 || len(aliases) > 2048 {
		return nil, errors.New("candidate alias inventory unavailable or oversized")
	}
	return aliases, nil
}

func radioDevice(profile, id string) (bus, address string, err error) {
	if profile == NobleBluetoothProfile {
		address = strings.TrimPrefix(id, "usb:")
		if id != "usb:"+address || !radioUSBAddress.MatchString(address) {
			return "", "", errors.New("invalid selected USB radio identity")
		}
		return "usb", address, nil
	}
	address = strings.TrimPrefix(id, "pci:")
	if id != "pci:"+address || !radioPCIAddress.MatchString(address) {
		return "", "", errors.New("invalid selected PCI radio identity")
	}
	return "pci", address, nil
}

func assertWirelessConsumers(ctx context.Context, t *durableWireless) error {
	if t.intent.Context == nil {
		return errors.New("radio admission context missing")
	}
	bus, address, err := radioDevice(t.intent.Plan.Profile, t.intent.Context.Device)
	if err != nil {
		return err
	}
	cohort := radioModuleNames(t.intent.Plan.Profile)
	for _, module := range cohort {
		entries, e := fs.ReadDir(t.root.FS(), "sys/module/"+module+"/holders")
		if errors.Is(e, fs.ErrNotExist) {
			if _, statErr := t.root.Stat("sys/module/" + module); errors.Is(statErr, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("radio module %s holder inventory unavailable", module)
		}
		if e != nil {
			return e
		}
		for _, entry := range entries {
			if !slices.Contains(cohort, entry.Name()) {
				return fmt.Errorf("radio module %s has unrelated holder %s", module, entry.Name())
			}
		}
	}
	aliases, err := t.radioSystem().aliases(ctx, t)
	if err != nil {
		return err
	}
	entries, err := fs.ReadDir(t.root.FS(), "sys/bus/"+bus+"/devices")
	if err != nil {
		return err
	}
	if len(entries) > 2048 {
		return errors.New("bus inventory exceeds radio scope bound")
	}
	selected := false
	for _, entry := range entries {
		name := entry.Name()
		if bus == "pci" && !radioPCIAddress.MatchString(name) {
			continue
		}
		if bus == "usb" && !strings.Contains(name, ":") {
			continue
		}
		data, e := radioRead(t.root, "sys/bus/"+bus+"/devices/"+name+"/modalias", 1024)
		if e != nil {
			if _, statErr := t.root.Stat("sys/bus/" + bus + "/devices/" + name); errors.Is(statErr, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("cannot qualify present radio bus function %s: %w", name, e)
		}
		matches := false
		for _, alias := range aliases {
			match, e := path.Match(strings.ToUpper(alias), strings.ToUpper(strings.TrimSpace(string(data))))
			if e != nil {
				return e
			}
			if match {
				matches = true
			}
		}
		if !matches {
			continue
		}
		own := name == address || (bus == "usb" && strings.HasPrefix(name, address+":"))
		if own {
			selected = true
			continue
		}
		link, e := t.root.Readlink("sys/bus/" + bus + "/devices/" + name + "/driver")
		if e != nil || path.Base(link) == radioLeaf(t.intent.Plan.Profile) {
			return fmt.Errorf("candidate would affect unrelated matching function %s", name)
		}
	}
	if !selected {
		return errors.New("exact selected radio function or modalias is absent")
	}
	return nil
}

func observeWirelessRadio(ctx context.Context, t *durableWireless) (WirelessRadioObservation, error) {
	r := WirelessRadioObservation{}
	if t.intent.Context == nil {
		return r, errors.New("radio admission context missing")
	}
	r.Device = t.intent.Context.Device
	bus, address, err := radioDevice(t.intent.Plan.Profile, r.Device)
	if err != nil {
		return r, err
	}
	all := true
	for _, module := range radioModuleNames(t.intent.Plan.Profile) {
		loaded, candidate, stock, e := t.radioSystem().identity(ctx, t, module)
		if e != nil {
			return r, e
		}
		if loaded {
			r.LoadedModules = append(r.LoadedModules, module)
		}
		if candidate {
			r.CandidateModulesLoaded = append(r.CandidateModulesLoaded, module)
		}
		if loaded && !candidate && !stock {
			return r, fmt.Errorf("loaded radio module %s has an unreviewed identity", module)
		}
		if !loaded || !candidate {
			all = false
		}
	}
	r.CandidateModulesVerified = all
	if bus == "pci" {
		link, e := t.root.Readlink("sys/bus/pci/devices/" + address + "/driver")
		if e == nil {
			r.Driver = path.Base(link)
		}
		entries, e := fs.ReadDir(t.root.FS(), "sys/bus/pci/devices/"+address+"/net")
		if e == nil {
			for _, entry := range entries {
				if nativeName.MatchString(entry.Name()) {
					r.Interfaces = append(r.Interfaces, entry.Name())
				}
			}
		}
	} else {
		link, e := t.root.Readlink("sys/bus/usb/devices/" + address + ":1.0/driver")
		if e == nil {
			r.Driver = path.Base(link)
		}
		entries, e := fs.ReadDir(t.root.FS(), "sys/class/bluetooth")
		if e == nil {
			for _, entry := range entries {
				link, e := t.root.Readlink("sys/class/bluetooth/" + entry.Name() + "/device")
				if e == nil && selectedUSBInterfaceLink(link, address) {
					r.Controllers = append(r.Controllers, entry.Name())
				}
			}
		}
	}
	r.BindingObserved = r.Driver == radioLeaf(t.intent.Plan.Profile)
	observeWirelessFunction(ctx, t, &r)
	r.Details = append(r.Details, "Binding and module build identity are observations; firmware initialization, association/traffic or peer pairing and resume require separate functional evidence")
	return r, assertWirelessConsumers(ctx, t)
}

// Passive function evidence sends no scan, connection, pairing or network
// probe. It becomes true only after the operator's actual association/traffic
// or paired live peer is observable through the selected device.
func observeWirelessFunction(ctx context.Context, t *durableWireless, r *WirelessRadioObservation) {
	if !r.CandidateModulesVerified || !r.BindingObserved {
		return
	}
	if t.intent.Plan.Profile == NobleProfile {
		for _, iface := range r.Interfaces {
			out, err := t.radioSystem().command(ctx, "/usr/sbin/iw", "dev", iface, "link")
			if err != nil {
				r.Details = append(r.Details, "Wi-Fi association observation unavailable: "+err.Error())
				continue
			}
			if !strings.HasPrefix(strings.TrimSpace(string(out)), "Connected to ") {
				continue
			}
			carrier, err := radioRead(t.root, "sys/class/net/"+iface+"/carrier", 32)
			if err != nil || strings.TrimSpace(string(carrier)) != "1" {
				continue
			}
			both := true
			for _, counter := range []string{"rx_packets", "tx_packets"} {
				data, err := radioRead(t.root, "sys/class/net/"+iface+"/statistics/"+counter, 64)
				if err != nil {
					both = false
					break
				}
				n, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
				if err != nil || n == 0 {
					both = false
				}
			}
			if both {
				r.FunctionVerified = true
				r.Details = append(r.Details, "Selected Wi-Fi interface is associated and has both RX/TX packet evidence; endpoint reachability and resume were not probed")
			}
		}
		return
	}
	observeBlueZPeer(ctx, t, r)
}
