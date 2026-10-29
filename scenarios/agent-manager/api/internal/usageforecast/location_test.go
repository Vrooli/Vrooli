package usageforecast

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDigestLocationPrefersConfiguredZoneThenTheHostZone(t *testing.T) {
	link := filepath.Join(t.TempDir(), "localtime")
	if err := os.Symlink("/usr/share/zoneinfo/Asia/Tokyo", link); err != nil {
		t.Fatal(err)
	}
	env := func(values map[string]string) func(string) string {
		return func(k string) string { return values[k] }
	}
	for name, c := range map[string]struct {
		env  map[string]string
		want string
	}{
		"setting wins":          {map[string]string{TimezoneEnv: "America/New_York", "TZ": "UTC"}, "America/New_York"},
		"TZ next":               {map[string]string{"TZ": "Europe/Paris"}, "Europe/Paris"},
		"host zone from link":   {nil, "Asia/Tokyo"},
		"invalid setting skips": {map[string]string{TimezoneEnv: "Not/AZone"}, "Asia/Tokyo"},
	} {
		if got := digestLocation(env(c.env), link).String(); got != c.want {
			t.Errorf("%s: %s, want %s", name, got, c.want)
		}
	}
	if got := digestLocation(env(nil), filepath.Join(t.TempDir(), "missing")); got.String() != "Local" {
		t.Errorf("no zone known: %s, want the process zone", got)
	}
}
