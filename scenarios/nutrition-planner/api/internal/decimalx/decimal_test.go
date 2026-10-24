package decimalx

import "testing"

func TestCanonicalAndUnknown(t *testing.T) {
	for input, want := range map[string]string{"1.00": "1", "0.50": "0.5", "-2.25": "-2.25"} {
		d, err := Parse(input)
		if err != nil || d.String() != want {
			t.Fatalf("%s => %s, %v", input, d.String(), err)
		}
	}
	z := KnownInt(0)
	if z.IsUnknown() || !z.IsZero() || Unknown.String() != "unknown" {
		t.Fatal("unknown and zero must remain distinct")
	}
	got, err := Add(Unknown, z)
	if err != nil || !got.IsUnknown() {
		t.Fatal("unknown arithmetic must remain unknown")
	}
}
