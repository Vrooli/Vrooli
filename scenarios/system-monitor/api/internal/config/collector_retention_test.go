package config

import (
	"testing"
	"time"
)

func TestParseCollectorRetentionDefaultsBoundLargeCollectors(t *testing.T) {
	got := ParseCollectorRetention("")
	if got["device_graph"] != 48*time.Hour {
		t.Errorf("device_graph default = %v, want 48h", got["device_graph"])
	}
	if got["network"] != 7*24*time.Hour || got["pressure"] != 7*24*time.Hour {
		t.Errorf("network/pressure defaults = %v/%v, want 168h", got["network"], got["pressure"])
	}
}

func TestParseCollectorRetentionOverridesAndSkipsInvalid(t *testing.T) {
	got := ParseCollectorRetention(" device_graph=12h , bogus , cpu=-1h, gpu=abc, =3h ")
	if len(got) != 1 || got["device_graph"] != 12*time.Hour {
		t.Errorf("ParseCollectorRetention = %v, want only device_graph=12h", got)
	}
}
