package retention

import (
	"errors"
	"testing"
	"time"
)

func TestParseBytes(t *testing.T) {
	accepted := map[string]int64{
		"0B":      0,
		"1B":      1,
		"512KiB":  512 * 1024,
		"2GiB":    2 * 1024 * 1024 * 1024,
		"5GiB":    5 * 1024 * 1024 * 1024,
		"1TiB":    1024 * 1024 * 1024 * 1024,
		" 2GiB ":  2 * 1024 * 1024 * 1024,
		"1024MiB": 1024 * 1024 * 1024,
	}
	for in, want := range accepted {
		got, err := ParseBytes(in)
		if err != nil {
			t.Errorf("ParseBytes(%q) unexpected error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseBytes(%q) = %d, want %d", in, got, want)
		}
	}

	// A unit is mandatory, and decimal units are deliberately unsupported so
	// GB/GiB ambiguity cannot enter a byte ceiling.
	unknownUnit := []string{"", "2000000", "2GB", "2gib", "GiB", "2 GiB", "2Gi", "-1GiB", "2PiB"}
	for _, in := range unknownUnit {
		if _, err := ParseBytes(in); !errors.Is(err, ErrUnknownUnit) {
			t.Errorf("ParseBytes(%q) = %v, want ErrUnknownUnit", in, err)
		}
	}
}

func TestParseAge(t *testing.T) {
	accepted := map[string]time.Duration{
		"0d":  0,
		"30d": 30 * 24 * time.Hour,
		"72h": 72 * time.Hour,
		"1h":  time.Hour,
	}
	for in, want := range accepted {
		got, err := ParseAge(in)
		if err != nil {
			t.Errorf("ParseAge(%q) unexpected error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseAge(%q) = %v, want %v", in, got, want)
		}
	}
	for _, in := range []string{"", "30", "30m", "30s", "d", "thirty days"} {
		if _, err := ParseAge(in); !errors.Is(err, ErrUnknownUnit) {
			t.Errorf("ParseAge(%q) = %v, want ErrUnknownUnit", in, err)
		}
	}
}

func TestFormatBytesRoundTrips(t *testing.T) {
	for _, in := range []string{"0B", "1B", "2GiB", "512KiB", "1TiB", "3MiB"} {
		n, err := ParseBytes(in)
		if err != nil {
			t.Fatalf("ParseBytes(%q): %v", in, err)
		}
		if got := FormatBytes(n); got != in {
			t.Errorf("FormatBytes(ParseBytes(%q)) = %q, want %q", in, got, in)
		}
	}
}

func TestFormatBytesRendersMeasuredSizesReadably(t *testing.T) {
	// A measured size rarely divides evenly. Rendering it exactly gives
	// "477738016KiB", which cannot be compared against a 2GiB ceiling at a
	// glance — and every finding and log line puts these two side by side.
	cases := map[int64]string{
		488946122752: "455.4GiB",
		1610612737:   "1.5GiB",
		1500:         "1.5KiB",
		1023:         "1023B",
	}
	for in, want := range cases {
		if got := FormatBytes(in); got != want {
			t.Errorf("FormatBytes(%d) = %q, want %q", in, got, want)
		}
	}
	// Exact values must still round-trip exactly, so a declared budget reads
	// back the way it was written.
	if got := FormatBytes(2 << 30); got != "2GiB" {
		t.Errorf("FormatBytes(2GiB) = %q, want the exact form", got)
	}
}
