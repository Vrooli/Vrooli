package remotesessionprotection

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const (
	ancestorSlicePath = "/etc/systemd/system/user.slice.d/90-vrooli-remote-session-protection.conf"
	userCgroupPath    = "/sys/fs/cgroup/user.slice"
)

var reservationLstat = os.Lstat

type reservation struct{ min, low uint64 }
type reservationTarget struct {
	unit, path, cgroup, content    string
	want, managed, before, manager reservation
	write                          bool
}

func ancestorSliceContent(minMB, lowMB int) string {
	return fmt.Sprintf("[Slice]\n# Vrooli Remote Session Protection (ancestor reservation)\nMemoryMin=%dM\nMemoryLow=%dM\n", minMB, lowMB)
}

// Only this owner's exact templates may be rewritten. Other operator files,
// including stronger settings in an unfamiliar representation, are preserved
// and reported for review instead of being silently replaced.
func reservationDefinition(path string, alloc memoryAllocation) (string, reservation, bool, error) {
	if err := hostreqkit.ValidateManagedFilePath(path, reservationLstat); err != nil {
		return "", reservation{}, false, err
	}
	render := func(min, low int) string {
		if path == ancestorSlicePath {
			return ancestorSliceContent(min, low)
		}
		return desktopSliceContent(memoryAllocation{desktopMinMB: min, desktopLowMB: low})
	}
	desired := render(alloc.desktopMinMB, alloc.desktopLowMB)
	info, statErr := reservationLstat(path)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return "", reservation{}, false, fmt.Errorf("inspect reservation definition %s: %w", path, statErr)
	}
	if statErr == nil && !info.Mode().IsRegular() {
		return "", reservation{}, false, fmt.Errorf("non-regular reservation definition at %s; preserved", path)
	}
	data, err := hostreqkit.ReadFileFn(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", reservation{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	if errors.Is(statErr, os.ErrNotExist) != errors.Is(err, os.ErrNotExist) {
		return "", reservation{}, false, fmt.Errorf("reservation definition appeared or disappeared while reading %s", path)
	}
	min, low := alloc.desktopMinMB, alloc.desktopLowMB
	if err == nil {
		pattern := regexp.QuoteMeta(render(0, 0))
		pattern = strings.Replace(pattern, "MemoryMin=0M", `MemoryMin=([0-9]+)M`, 1)
		pattern = strings.Replace(pattern, "MemoryLow=0M", `MemoryLow=([0-9]+)M`, 1)
		matches := regexp.MustCompile("^" + pattern + "$").FindStringSubmatch(string(data))
		if matches == nil {
			return "", reservation{}, false, fmt.Errorf("unrecognized reservation definition at %s; preserved", path)
		}
		values := []*int{&min, &low}
		for i, value := range values {
			n, e := strconv.ParseUint(matches[i+1], 10, 63)
			if e != nil || n > uint64(math.MaxInt/bytesPerMiB) {
				return "", reservation{}, false, fmt.Errorf("invalid managed reservation at %s", path)
			}
			if int(n) > *value {
				*value = int(n)
			}
		}
		desired = render(min, low)
	}
	return desired, reservation{uint64(min) * bytesPerMiB, uint64(low) * bytesPerMiB}, string(data) != desired, nil
}

func readReservation(cgroup string) (reservation, error) {
	values := []uint64{0, 0}
	for i, name := range []string{"memory.min", "memory.low"} {
		path := filepath.Join(cgroup, name)
		data, err := hostreqkit.ReadFileFn(path)
		if err != nil {
			return reservation{}, fmt.Errorf("read effective %s: %w", path, err)
		}
		value := strings.TrimSpace(string(data))
		if value == "max" {
			values[i] = math.MaxUint64
			continue
		}
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return reservation{}, fmt.Errorf("invalid effective %s: %w", path, err)
		}
		values[i] = n
	}
	return reservation{values[0], values[1]}, nil
}

func desktopReservationPlan(alloc memoryAllocation) ([]reservationTarget, error) {
	targets := []reservationTarget{{unit: "user.slice", path: ancestorSlicePath, cgroup: userCgroupPath}, {unit: fmt.Sprintf("user-%d.slice", desktopUID), path: desktopSlicePath, cgroup: fmt.Sprintf("%s/user-%d.slice", userCgroupPath, desktopUID)}}
	paths, err := reservationUnitPaths()
	if err != nil {
		return nil, err
	}
	// Qualify every definition and live node before allowing the first write.
	for i := range targets {
		t := &targets[i]
		var err error
		t.content, t.managed, t.write, err = reservationDefinition(t.path, alloc)
		if err != nil {
			return nil, err
		}
		t.want, t.manager, err = qualifyReservationDefinitions(paths, *t)
		if err != nil {
			return nil, err
		}
		t.before, err = readReservation(t.cgroup)
		if err != nil {
			return nil, err
		}
	}
	return targets, nil
}

func reservationSufficient(got, want reservation) bool {
	return got.min >= want.min && got.low >= want.low
}

func inspectDesktopReservation(alloc memoryAllocation) []string {
	targets, err := desktopReservationPlan(alloc)
	if err != nil {
		return []string{err.Error()}
	}
	var pending []string
	for _, t := range targets {
		if t.write {

			pending = append(pending, t.path)
		}
		if !reservationSufficient(t.before, t.want) || !reservationSufficient(t.manager, t.want) {
			pending = append(pending, "effective reservation at "+t.cgroup)
		}
	}
	return pending
}

func ensureDesktopReservation(alloc memoryAllocation, opts hostreqkit.EnsureOptions) error {
	targets, err := desktopReservationPlan(alloc)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if t.write {
			// Requalify immediately before writing; preserve a newly appeared or
			// strengthened operator definition instead of replaying a stale plan.
			current, _, _, err := reservationDefinition(t.path, alloc)
			if err != nil || current != t.content {
				return fmt.Errorf("reservation definition changed before write at %s: %w", t.path, errors.Join(err, errors.New("reservation plan is stale")))
			}
			if err := hostreqkit.EnsureManagedDir(filepath.Dir(t.path), opts.SudoMode, opts); err != nil {
				return fmt.Errorf("create reservation directory: %w", err)
			}
			if err := hostreqkit.ValidateManagedFilePath(t.path, reservationLstat); err != nil {
				return err
			}
			writeErr := hostreqkit.InstallManagedContent(t.path, t.content, opts.SudoMode, opts)
			// Resolve lost acknowledgements by exact owner bytes, not by replay.
			if !hostreqkit.FileContentMatches(t.path, t.content) {
				return fmt.Errorf("reservation definition not verified at %s: %w", t.path, errors.Join(writeErr, errors.New("persistent readback mismatch")))
			}
		}
		// Preserve stronger live allocations from other operator definitions.
		want := t.want
		if t.before.min > want.min {
			want.min = t.before.min
		}
		if t.before.low > want.low {
			want.low = t.before.low
		}
		var commandErr error
		if !reservationSufficient(t.before, t.want) || !reservationSufficient(t.manager, want) {
			format := func(n uint64) string {
				if n == math.MaxUint64 {
					return "infinity"
				}
				return strconv.FormatUint(n, 10)
			}
			commandErr = hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"set-property", "--runtime", t.unit, "MemoryMin=" + format(want.min), "MemoryLow=" + format(want.low)}, opts)
		}
		// set-property applies immediately and needs no daemon-reload. A repair of
		// these two properties must not restart Docker/GPU or change login policy.
		got, readErr := readReservation(t.cgroup)
		if readErr != nil || !reservationSufficient(got, want) {
			return fmt.Errorf("effective reservation not verified at %s: %w", t.cgroup, errors.Join(commandErr, readErr, errors.New("kernel reservation postcondition failed")))
		}
	}
	for _, t := range targets {
		content, _, needsWrite, err := reservationDefinition(t.path, alloc)
		if err != nil || needsWrite || content != t.content {
			return fmt.Errorf("persistent reservation not verified after convergence at %s: %w", t.path, errors.Join(err, errors.New("persistent postcondition failed")))
		}
	}
	verified, err := desktopReservationPlan(alloc)
	if err != nil {
		return fmt.Errorf("reservation definition/manager readback: %w", err)
	}
	for _, t := range verified {
		if t.write || !reservationSufficient(t.before, t.want) || !reservationSufficient(t.manager, t.want) {
			return fmt.Errorf("reservation definitions or manager/kernel floor changed during convergence for %s", t.unit)
		}
	}
	return nil
}
