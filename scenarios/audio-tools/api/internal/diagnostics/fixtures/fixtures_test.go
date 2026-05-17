package fixtures

import "testing"

func TestSmokeWAV(t *testing.T) {
	if len(SmokeWAV()) == 0 {
		t.Fatal("SmokeWAV must embed a non-empty payload")
	}
}

func TestSmokeText(t *testing.T) {
	if SmokeText() == "" {
		t.Fatal("SmokeText must be non-empty")
	}
}
