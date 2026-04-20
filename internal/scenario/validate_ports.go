package scenario

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/portspec"
)

// PortValidationMode controls how the ephemeral-overlap validator reacts to
// findings. The default (empty string) behaves as ModeFatal.
type PortValidationMode string

const (
	ModeFatal PortValidationMode = ""     // reject manifest load
	ModeWarn  PortValidationMode = "warn" // print but do not reject (not plumbed yet; reserved)
	ModeOff   PortValidationMode = "off"  // skip validation entirely
)

// envPortValidation names the env-var escape hatch. "off" skips validation;
// any other value is treated as fatal. This is intentionally not a general
// toggle — it exists so an operator can unblock a scenario start while they
// prepare a migration, not as a permanent bypass.
const envPortValidation = "VROOLI_PORT_VALIDATION"

// portValidationMode returns the active mode from the environment.
var portValidationMode = func() PortValidationMode {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(envPortValidation)))
	switch v {
	case "off":
		return ModeOff
	case "warn":
		return ModeWarn
	default:
		return ModeFatal
	}
}

// portEphemeral is the ephemeral-range probe. Tests override it via
// setPortEphemeralForTest.
var portEphemeral = func(ctx context.Context) portspec.EphemeralRange {
	return portspec.OSEphemeralRange(ctx)
}

// validateManifestPorts walks every declared port and rejects fixed ports or
// ranges that overlap the host OS's ephemeral window or exceed the canonical
// max. Called from ReadService after structural validation.
func validateManifestPorts(path string, ports map[string]Port) error {
	if portValidationMode() == ModeOff || len(ports) == 0 {
		return nil
	}

	eph := portEphemeral(context.Background())

	// Sort keys so error messages are stable.
	names := make([]string, 0, len(ports))
	for name := range ports {
		names = append(names, name)
	}
	sort.Strings(names)

	var issues []string
	for _, name := range names {
		port := ports[name]
		if msg := checkFixedPort(name, port, eph); msg != "" {
			issues = append(issues, msg)
		}
		if msg := checkPortRange(name, port, eph); msg != "" {
			issues = append(issues, msg)
		}
	}

	if len(issues) == 0 {
		return nil
	}

	msg := "validate ports in " + path + ":\n  " + strings.Join(issues, "\n  ") +
		"\n\nSee docs/reference/port-allocation.md for the canonical bands " +
		"(API " + rangeLabel(portspec.APIRangeStart, portspec.APIRangeEnd) + ", " +
		"UI " + rangeLabel(portspec.UIRangeStart, portspec.UIRangeEnd) + ", " +
		"WS " + rangeLabel(portspec.WSRangeStart, portspec.WSRangeEnd) +
		"). Run `go run ./cmd/vrooli-ports-migrate` to apply the shift."
	return errors.New(msg)
}

func checkFixedPort(name string, port Port, eph portspec.EphemeralRange) string {
	if port.Port == nil {
		return ""
	}
	p := *port.Port
	if p <= 0 || p > 65535 {
		return fmt.Sprintf("port %q: invalid fixed port %d (must be 1..65535)", name, p)
	}
	// Report ephemeral overlap first — that is the real failure mode the user
	// experiences as "port already in use". Plain above-canonical-max is a
	// weaker finding that only applies when the host OS happens to use a
	// higher ephemeral floor (e.g. macOS 49152+).
	if eph.Contains(p) {
		return fmt.Sprintf(
			"port %q: fixed port %d overlaps OS ephemeral range %d-%d (source=%s); move to a canonical band below %d",
			name, p, eph.Min, eph.Max, eph.Source, portspec.CanonicalMax+1,
		)
	}
	if p > portspec.CanonicalMax {
		return fmt.Sprintf("port %q: fixed port %d exceeds canonical max %d", name, p, portspec.CanonicalMax)
	}
	return ""
}

func checkPortRange(name string, port Port, eph portspec.EphemeralRange) string {
	raw := strings.TrimSpace(port.Range)
	if raw == "" {
		return ""
	}
	lo, hi, err := parsePortRange(raw)
	if err != nil {
		return fmt.Sprintf("port %q: invalid range %q: %v", name, raw, err)
	}
	if lo <= 0 || hi > 65535 {
		return fmt.Sprintf("port %q: range %q outside 1..65535", name, raw)
	}
	// Ephemeral overlap is the operational failure; canonical-max is the
	// weaker structural concern. Report the operational one first.
	if eph.Overlaps(lo, hi) {
		return fmt.Sprintf(
			"port %q: range %s overlaps OS ephemeral range %d-%d (source=%s); move into a canonical band below %d",
			name, raw, eph.Min, eph.Max, eph.Source, portspec.CanonicalMax+1,
		)
	}
	if hi > portspec.CanonicalMax {
		return fmt.Sprintf(
			"port %q: range %s extends past canonical max %d; clamp at %d or lower",
			name, raw, portspec.CanonicalMax, portspec.CanonicalMax,
		)
	}
	return ""
}

func parsePortRange(raw string) (int, int, error) {
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected start-end")
	}
	lo, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("parse start: %w", err)
	}
	hi, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("parse end: %w", err)
	}
	if hi < lo {
		return 0, 0, fmt.Errorf("end %d < start %d", hi, lo)
	}
	return lo, hi, nil
}

func rangeLabel(lo, hi int) string {
	return strconv.Itoa(lo) + "-" + strconv.Itoa(hi)
}
