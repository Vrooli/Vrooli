package scenes

import "testing"

func TestValidPreparationIDRequiresLowercaseSHA256(t *testing.T) {
	good := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if !validPreparationID(good) {
		t.Fatal("expected valid lowercase SHA-256 digest")
	}
	for _, value := range []string{
		"",
		good[:63],
		good[:63] + "g",
		"ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
	} {
		if validPreparationID(value) {
			t.Errorf("accepted invalid preparation id %q", value)
		}
	}
}

func TestParseAllowLocalRenderIsStrictAndOffByDefault(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"", false}, {"false", false}, {"TRUE", true},
	} {
		got, err := parseAllowLocalRender(tc.value)
		if err != nil || got != tc.want {
			t.Errorf("parseAllowLocalRender(%q) = %v, %v; want %v, nil", tc.value, got, err, tc.want)
		}
	}
	for _, value := range []string{"yes", "1", "on"} {
		if _, err := parseAllowLocalRender(value); err == nil {
			t.Errorf("parseAllowLocalRender(%q) accepted an ambiguous value", value)
		}
	}
}
