package retention

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Units are mandatory in manifests. A bare integer reads as either bytes or
// gigabytes depending on the reader, the mistake is invisible in review, and a
// 1000x error in a byte ceiling is a disk outage.

// byteUnits maps the accepted binary size suffixes to their multipliers.
// Decimal suffixes (GB, MB) are deliberately absent: mixing 10^9 and 2^30 in the
// same field is exactly the ambiguity a mandatory unit exists to remove.
var byteUnits = map[string]int64{
	"B":   1,
	"KiB": 1 << 10,
	"MiB": 1 << 20,
	"GiB": 1 << 30,
	"TiB": 1 << 40,
}

// byteUnitOrder is the descending order used when rendering a size back out.
var byteUnitOrder = []string{"TiB", "GiB", "MiB", "KiB", "B"}

// ageUnits maps the accepted age suffixes. Retention horizons are measured in
// hours and days; finer units invite a budget that expires data faster than a
// cycle can run.
var ageUnits = map[string]time.Duration{
	"h": time.Hour,
	"d": 24 * time.Hour,
}

// splitUnit divides a value into its leading decimal digits and its trailing
// unit. Both halves must be non-empty, which is what makes a bare integer and a
// bare unit equally invalid.
func splitUnit(s string) (digits, unit string, ok bool) {
	trimmed := strings.TrimSpace(s)
	cut := 0
	for cut < len(trimmed) && trimmed[cut] >= '0' && trimmed[cut] <= '9' {
		cut++
	}
	digits, unit = trimmed[:cut], trimmed[cut:]
	return digits, unit, digits != "" && unit != ""
}

// ParseBytes parses a size string with a mandatory binary unit, such as "2GiB".
func ParseBytes(s string) (int64, error) {
	digits, unit, ok := splitUnit(s)
	if !ok {
		return 0, fmt.Errorf("%w: %q must be digits followed by one of B, KiB, MiB, GiB, TiB", ErrUnknownUnit, s)
	}
	scale, known := byteUnits[unit]
	if !known {
		return 0, fmt.Errorf("%w: %q uses unit %q, want one of B, KiB, MiB, GiB, TiB", ErrUnknownUnit, s, unit)
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse size %q: %w", s, err)
	}
	if n != 0 && n > (1<<62)/scale {
		return 0, fmt.Errorf("parse size %q: value overflows int64 bytes", s)
	}
	return n * scale, nil
}

// ParseAge parses an age string with a mandatory unit, such as "30d" or "72h".
func ParseAge(s string) (time.Duration, error) {
	digits, unit, ok := splitUnit(s)
	if !ok {
		return 0, fmt.Errorf("%w: %q must be digits followed by h or d", ErrUnknownUnit, s)
	}
	scale, known := ageUnits[unit]
	if !known {
		return 0, fmt.Errorf("%w: %q uses unit %q, want h or d", ErrUnknownUnit, s, unit)
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse age %q: %w", s, err)
	}
	if n != 0 && n > int64(time.Duration(1<<62)/scale) {
		return 0, fmt.Errorf("parse age %q: value overflows a duration", s)
	}
	return time.Duration(n) * scale, nil
}

// FormatBytes renders a byte count for a human.
//
// A declared budget round-trips exactly — 2GiB in, "2GiB" out — so a value
// echoed into a log or a finding reads the way it was written. A measured size
// rarely divides evenly, and rendering it exactly produces "477738016KiB", which
// no operator can compare against a 2GiB ceiling at a glance; those fall back to
// one decimal place in the largest unit that fits.
func FormatBytes(n int64) string {
	if n == 0 {
		return "0B"
	}
	if n < 0 {
		return strconv.FormatInt(n, 10) + "B"
	}
	// Pick the largest unit the value reaches, then render exactly if it divides
	// evenly there. Choosing the unit by divisibility instead would render
	// 455 GiB as "477486448KiB", because that number happens to be a whole
	// number of KiB.
	for _, unit := range byteUnitOrder {
		scale := byteUnits[unit]
		if n < scale {
			continue
		}
		if n%scale == 0 {
			return strconv.FormatInt(n/scale, 10) + unit
		}
		return strconv.FormatFloat(float64(n)/float64(scale), 'f', 1, 64) + unit
	}
	return strconv.FormatInt(n, 10) + "B"
}
