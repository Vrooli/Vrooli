package remotesessionprotection

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// UnitPath comes from the running SYSTEM manager, not the caller's ambient
// SYSTEMD_UNIT_PATH. Directory reads and admitted configuration are bounded.
var reservationReadDir = func(path string) ([]os.DirEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(257)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return entries, err
}

type reservationManager struct {
	values   reservation
	fragment string
	dropins  []string
}
type reservationLayer struct {
	path, data        string
	specificity, rank int
	persistent        bool
}

func reservationProperties(args ...string) (map[string]string, error) {
	raw, err := hostreqkit.CombinedOutputFn("systemctl", args...)
	if err != nil {
		return nil, fmt.Errorf("query system reservation definitions: %w", err)
	}
	if len(raw) > 64<<10 {
		return nil, fmt.Errorf("system reservation metadata exceeds bound")
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid system reservation metadata")
		}
		if _, duplicate := out[key]; duplicate {
			return nil, fmt.Errorf("duplicate system reservation metadata %s", key)
		}
		out[key] = value
	}
	return out, nil
}

func reservationUnitPaths() ([]string, error) {
	props, err := reservationProperties("show", "--property=UnitPath")
	if err != nil {
		return nil, err
	}
	paths := strings.Fields(props["UnitPath"])
	if len(paths) == 0 || len(paths) > 32 {
		return nil, fmt.Errorf("system manager UnitPath is unavailable or exceeds bound")
	}
	seen := map[string]bool{}
	for _, p := range paths {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, "\\\x00") || seen[p] {
			return nil, fmt.Errorf("unqualified system manager unit path %q", p)
		}
		seen[p] = true
	}
	return paths, nil
}

func reservationAmount(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "infinity" {
		return math.MaxUint64, nil
	}
	if value == "" {
		return 0, nil
	} // systemd's empty assignment resets the property.
	factor := uint64(1)
	if last := value[len(value)-1]; last < '0' || last > '9' {
		powers := map[byte]int{'K': 1, 'M': 2, 'G': 3, 'T': 4, 'P': 5, 'E': 6}
		power, ok := powers[last]
		if !ok {
			return 0, fmt.Errorf("unknown memory reservation syntax %q", value)
		}
		for i := 0; i < power; i++ {
			factor *= 1024
		}
		value = value[:len(value)-1]
	}
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil || n > math.MaxUint64/factor {
		return 0, fmt.Errorf("unknown or overflowing memory reservation value %q", value)
	}
	return n * factor, nil
}

func readReservationManager(unit, cgroup string) (reservationManager, error) {
	p, err := reservationProperties("show", unit, "--property=Id,Names,LoadState,FragmentPath,DropInPaths,MemoryMin,MemoryLow,ControlGroup,NeedDaemonReload")
	if err != nil {
		return reservationManager{}, err
	}
	for _, key := range []string{"Id", "Names", "LoadState", "FragmentPath", "DropInPaths", "MemoryMin", "MemoryLow", "ControlGroup", "NeedDaemonReload"} {
		if _, ok := p[key]; !ok {
			return reservationManager{}, fmt.Errorf("missing system manager property %s", key)
		}
	}
	if p["Id"] != unit || p["Names"] != unit || p["LoadState"] != "loaded" || p["ControlGroup"] != strings.TrimPrefix(cgroup, "/sys/fs/cgroup") {
		return reservationManager{}, fmt.Errorf("system reservation unit identity/aliases are unqualified for %s", unit)
	}
	if p["NeedDaemonReload"] != "yes" && p["NeedDaemonReload"] != "no" {
		return reservationManager{}, fmt.Errorf("system unit reload state unknown")
	}
	if strings.TrimSpace(p["MemoryMin"]) == "" || strings.TrimSpace(p["MemoryLow"]) == "" {
		return reservationManager{}, fmt.Errorf("system unit memory properties are unknown")
	}
	min, err := reservationAmount(p["MemoryMin"])
	if err != nil {
		return reservationManager{}, err
	}
	low, err := reservationAmount(p["MemoryLow"])
	if err != nil {
		return reservationManager{}, err
	}
	dropins := strings.Fields(p["DropInPaths"])
	if len(dropins) > 256 {
		return reservationManager{}, fmt.Errorf("loaded system unit drop-in inventory exceeds bound")
	}
	return reservationManager{values: reservation{min, low}, fragment: p["FragmentPath"], dropins: dropins}, nil
}

func reservationDropinNames(unit string) []string {
	names := []string{"slice.d"}
	stem := strings.TrimSuffix(unit, ".slice")
	for i, c := range stem {
		if c == '-' {
			names = append(names, stem[:i+1]+".slice.d")
		}
	}
	return append(names, unit+".d")
}

func reservationFile(path string) (string, error) {
	info, err := reservationLstat(path)
	if err != nil {
		return "", err
	}
	if info == nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("unit definition alias/mask/nonregular file at %s is unqualified", path)
	}
	data, err := hostreqkit.ReadFileFn(path)
	if err != nil {
		return "", err
	}
	if len(data) > 64<<10 {
		return "", fmt.Errorf("unit definition exceeds bound at %s", path)
	}
	return string(data), nil
}

func reservationMain(paths []string, unit string, persistentOnly bool) (*reservationLayer, error) {
	for rank, root := range paths {
		persistent := !strings.HasPrefix(root, "/run/")
		if persistentOnly && !persistent {
			continue
		}
		path := filepath.Join(root, unit)
		data, err := reservationFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return &reservationLayer{path: path, data: data, rank: rank, persistent: persistent}, nil
	}
	return nil, nil // Implicit slice units need not have a main fragment.
}

func reservationLayers(paths []string, unit, owned, planned string) ([]reservationLayer, map[string]bool, error) {
	layers := []reservationLayer{}
	observed := map[string]bool{}
	ownedFound := false
	for rank, root := range paths {
		for specificity, name := range reservationDropinNames(unit) {
			dir := filepath.Join(root, name)
			info, statErr := reservationLstat(dir)
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				return nil, nil, statErr
			}
			if statErr == nil && (info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
				return nil, nil, fmt.Errorf("unit drop-in directory alias/non-directory at %s is unqualified", dir)
			}
			entries, err := reservationReadDir(dir)
			if errors.Is(err, os.ErrNotExist) {
				entries = nil
			} else if err != nil {
				return nil, nil, fmt.Errorf("read unit drop-in directory %s: %w", dir, err)
			}
			if len(entries) > 256 {
				return nil, nil, fmt.Errorf("unit drop-in directory exceeds bound at %s", dir)
			}
			for _, entry := range entries {
				n := entry.Name()
				if !strings.HasSuffix(n, ".conf") {
					continue
				}
				if filepath.Base(n) != n || strings.ContainsAny(n, "\x00\r\n") {
					return nil, nil, fmt.Errorf("unqualified unit drop-in name")
				}
				path := filepath.Join(dir, n)
				data, err := reservationFile(path)
				if err != nil {
					return nil, nil, fmt.Errorf("read unit drop-in %s: %w", path, err)
				}
				observed[path] = true
				if path == owned {
					data = planned
					ownedFound = true
				}
				layers = append(layers, reservationLayer{path: path, data: data, specificity: specificity, rank: rank, persistent: !strings.HasPrefix(root, "/run/")})
				if len(layers) > 256 {
					return nil, nil, fmt.Errorf("unit definition inventory exceeds bound")
				}
			}
			if filepath.Dir(owned) == dir && !ownedFound {
				layers = append(layers, reservationLayer{path: owned, data: planned, specificity: specificity, rank: rank, persistent: !strings.HasPrefix(root, "/run/")})
				if len(layers) > 256 {
					return nil, nil, fmt.Errorf("unit definition inventory exceeds bound")
				}
				ownedFound = true
			}
		}
	}
	if !ownedFound {
		return nil, nil, fmt.Errorf("owned reservation directory is outside the system manager unit paths")
	}
	return layers, observed, nil
}

// This admits a deliberately conservative memory-assignment subset, not a
// second general systemd parser. Unsupported memory intent is never erased.
func applyReservationLayer(layer reservationLayer, values *reservation, maxima *reservation) error {
	section := ""
	for _, raw := range strings.Split(layer.data, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, ".include") {
			return fmt.Errorf("included unit definitions are unqualified at %s", layer.path)
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if section != "Slice" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if key == "DefaultMemoryMin" || key == "DefaultMemoryLow" {
			return fmt.Errorf("inherited default memory intent is unqualified at %s", layer.path)
		}
		if key != "MemoryMin" && key != "MemoryLow" {
			continue
		}
		if !ok {
			return fmt.Errorf("invalid memory assignment at %s", layer.path)
		}
		n, err := reservationAmount(value)
		if err != nil {
			return fmt.Errorf("%s: %w", layer.path, err)
		}
		if key == "MemoryMin" {
			values.min = n
			if n > maxima.min {
				maxima.min = n
			}
		} else {
			values.low = n
			if n > maxima.low {
				maxima.low = n
			}
		}
	}
	return nil
}

func effectiveReservation(main *reservationLayer, layers []reservationLayer, persistentOnly bool) (reservation, error) {
	selected := map[string]reservationLayer{}
	for _, f := range layers {
		if persistentOnly && !f.persistent {
			continue
		}
		name := filepath.Base(f.path)
		old, ok := selected[name]
		if !ok || f.specificity > old.specificity || (f.specificity == old.specificity && f.rank < old.rank) {
			selected[name] = f
		}
	}
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	sort.Strings(names)
	value, maxima := reservation{}, reservation{}
	if main != nil {
		if err := applyReservationLayer(*main, &value, &maxima); err != nil {
			return reservation{}, err
		}
	}
	for _, n := range names {
		if err := applyReservationLayer(selected[n], &value, &maxima); err != nil {
			return reservation{}, err
		}
	}
	// Do not shadow stronger known intent with a weaker runtime property or
	// owned file. Conflicting resets/order must be resolved by its actual owner.
	if !reservationSufficient(value, maxima) {
		return reservation{}, fmt.Errorf("conflicting memory definitions would shadow stronger intent")
	}
	return value, nil
}

func qualifyReservationDefinitions(paths []string, t reservationTarget) (reservation, reservation, error) {
	manager, err := readReservationManager(t.unit, t.cgroup)
	if err != nil {
		return reservation{}, reservation{}, err
	}
	main, err := reservationMain(paths, t.unit, false)
	if err != nil {
		return reservation{}, reservation{}, err
	}
	persistentMain, err := reservationMain(paths, t.unit, true)
	if err != nil {
		return reservation{}, reservation{}, err
	}
	if manager.fragment != "" && (main == nil || manager.fragment != main.path) {
		return reservation{}, reservation{}, fmt.Errorf("loaded system unit fragment differs from disk selection for %s", t.unit)
	}
	layers, observed, err := reservationLayers(paths, t.unit, t.path, t.content)
	if err != nil {
		return reservation{}, reservation{}, err
	}
	for _, path := range manager.dropins {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || (!observed[path] && path != t.path) {
			return reservation{}, reservation{}, fmt.Errorf("loaded drop-in inventory is unqualified at %s", path)
		}
	}
	persistent, err := effectiveReservation(persistentMain, layers, true)
	if err != nil {
		return reservation{}, reservation{}, err
	}
	all, err := effectiveReservation(main, layers, false)
	if err != nil {
		return reservation{}, reservation{}, err
	}
	if !reservationSufficient(persistent, t.managed) || !reservationSufficient(all, t.managed) {
		return reservation{}, reservation{}, fmt.Errorf("effective persistent/runtime definitions conflict with required reservation for %s", t.unit)
	}
	want := t.managed
	for _, v := range []reservation{persistent, all, manager.values} {
		if v.min > want.min {
			want.min = v.min
		}
		if v.low > want.low {
			want.low = v.low
		}
	}
	return want, manager.values, nil
}
