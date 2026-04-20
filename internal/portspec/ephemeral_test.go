package portspec

import (
	"context"
	"errors"
	"testing"
)

type fakeReader struct {
	r   EphemeralRange
	err error
}

func (f fakeReader) Read(_ context.Context) (EphemeralRange, error) {
	return f.r, f.err
}

func TestParseLinuxEphemeral(t *testing.T) {
	r, err := parseLinuxEphemeral("32768\t60999\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Min != 32768 || r.Max != 60999 {
		t.Errorf("min/max = %d/%d, want 32768/60999", r.Min, r.Max)
	}
	if r.Source != "linux-proc" {
		t.Errorf("source = %q, want linux-proc", r.Source)
	}
}

func TestParseLinuxEphemeral_Errors(t *testing.T) {
	cases := map[string]string{
		"empty":        "",
		"single field": "32768\n",
		"non-numeric":  "abc def\n",
		"inverted":     "60000 50000\n",
		"negative":     "-1 80\n",
	}
	for name, input := range cases {
		if _, err := parseLinuxEphemeral(input); err == nil {
			t.Errorf("%s: expected error for %q", name, input)
		}
	}
}

func TestParseDarwinEphemeral(t *testing.T) {
	r, err := parseDarwinEphemeral("49152\n65535\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Min != 49152 || r.Max != 65535 {
		t.Errorf("min/max = %d/%d, want 49152/65535", r.Min, r.Max)
	}
	if r.Source != "darwin-sysctl" {
		t.Errorf("source = %q, want darwin-sysctl", r.Source)
	}
}

func TestParseDarwinEphemeral_Errors(t *testing.T) {
	cases := map[string]string{
		"single line": "49152\n",
		"non-numeric": "first\nlast\n",
		"inverted":    "65535\n49152\n",
	}
	for name, input := range cases {
		if _, err := parseDarwinEphemeral(input); err == nil {
			t.Errorf("%s: expected error for %q", name, input)
		}
	}
}

func TestParseWindowsEphemeral(t *testing.T) {
	raw := `
Protocol tcp Dynamic Port Range
---------------------------------
Start Port      : 49152
Number of Ports : 16384
`
	r, err := parseWindowsEphemeral(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Min != 49152 || r.Max != 65535 {
		t.Errorf("min/max = %d/%d, want 49152/65535", r.Min, r.Max)
	}
	if r.Source != "windows-netsh" {
		t.Errorf("source = %q, want windows-netsh", r.Source)
	}
}

func TestParseWindowsEphemeral_LocalizedVariants(t *testing.T) {
	// Some locales and newer builds surface "Port count" instead of the
	// canonical "Number of Ports". Verify the permissive match still parses.
	raw := "Start port : 30000\nPort count : 1001\n"
	r, err := parseWindowsEphemeral(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Min != 30000 || r.Max != 31000 {
		t.Errorf("min/max = %d/%d, want 30000/31000", r.Min, r.Max)
	}
}

func TestParseWindowsEphemeral_Errors(t *testing.T) {
	cases := map[string]string{
		"empty":             "",
		"no start":          "Number of Ports : 100\n",
		"no count":          "Start Port : 1024\n",
		"non-numeric start": "Start Port : oops\nNumber of Ports : 100\n",
		"non-numeric count": "Start Port : 1024\nNumber of Ports : oops\n",
		"start zero":        "Start Port : 0\nNumber of Ports : 100\n",
		"count zero":        "Start Port : 1024\nNumber of Ports : 0\n",
	}
	for name, input := range cases {
		if _, err := parseWindowsEphemeral(input); err == nil {
			t.Errorf("%s: expected error for %q", name, input)
		}
	}
}

func TestOSEphemeralRange_FallbackOnError(t *testing.T) {
	restore := SetEphemeralReader(fakeReader{err: errors.New("nope")})
	defer restore()

	r := OSEphemeralRange(context.Background())
	if !r.Fallback {
		t.Error("expected fallback to be true")
	}
	if r.Min != ianaDynamicStart || r.Max != ianaDynamicEnd {
		t.Errorf("fallback range = %d..%d, want %d..%d", r.Min, r.Max, ianaDynamicStart, ianaDynamicEnd)
	}
	if r.Source != "fallback-iana" {
		t.Errorf("source = %q, want fallback-iana", r.Source)
	}
	if r.DetectErr == nil {
		t.Error("expected DetectErr to be populated")
	}
}

func TestOSEphemeralRange_UsesReader(t *testing.T) {
	want := EphemeralRange{Min: 10000, Max: 20000, Source: "test"}
	restore := SetEphemeralReader(fakeReader{r: want})
	defer restore()

	r := OSEphemeralRange(context.Background())
	if r.Min != want.Min || r.Max != want.Max || r.Source != want.Source || r.Fallback {
		t.Errorf("got %+v, want %+v (Fallback=false)", r, want)
	}
}

func TestEphemeralRange_ContainsAndOverlaps(t *testing.T) {
	r := EphemeralRange{Min: 32768, Max: 60999}

	if r.Contains(32767) || !r.Contains(32768) || !r.Contains(60999) || r.Contains(61000) {
		t.Error("Contains boundary check failed")
	}

	if !r.Overlaps(35000, 39999) {
		t.Error("expected overlap for 35000-39999")
	}
	if r.Overlaps(15000, 19999) {
		t.Error("did not expect overlap for 15000-19999")
	}
	if !r.Overlaps(32768, 32768) {
		t.Error("expected overlap for exact min")
	}
	// Inverted input should normalize and detect overlap.
	if !r.Overlaps(60000, 32768) {
		t.Error("expected inverted (60000,32768) to normalize and overlap")
	}
	// Inverted input that does not overlap.
	if r.Overlaps(19999, 15000) {
		t.Error("did not expect (19999,15000) to overlap")
	}
}
