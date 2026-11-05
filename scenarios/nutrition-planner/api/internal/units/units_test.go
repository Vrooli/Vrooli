package units

import "testing"

func TestMassRoundTrip(t *testing.T) {
	g, err := Convert(2, "kg", "g")
	if err != nil || g != 2000 {
		t.Fatalf("%d %v", g, err)
	}
	kg, err := Convert(g, "g", "kg")
	if err != nil || kg != 2 {
		t.Fatalf("%d %v", kg, err)
	}
	if _, err := Convert(1, "g", "ml"); err == nil {
		t.Fatal("mass-volume conversion accepted")
	}
}
