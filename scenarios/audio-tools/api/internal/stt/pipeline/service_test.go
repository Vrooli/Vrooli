package pipeline

import (
	"sync/atomic"
	"testing"
)

func newServiceForTest() *Service {
	return NewService(
		Config{FlushIntervalMs: 500, MinDeltaBytes: 4096, OverlapBytes: 2048, SegmentSilenceMs: 1500, WakeWordThreshold: 0.65},
		"",
		nil, "",
		SpeakerConfig{},
		"",
		nil,
		nil,
		&atomic.Int64{},
		"http://whisper.test",
		nil,
		nil,
	)
}

func TestServiceConfigGetSetSnapshot(t *testing.T) {
	s := newServiceForTest()
	got := s.GetConfig()
	if got.FlushIntervalMs != 500 {
		t.Fatalf("expected initial config preserved, got %+v", got)
	}
	got.FlushIntervalMs = 750
	s.SetConfig(got)
	if s.GetConfig().FlushIntervalMs != 750 {
		t.Fatalf("SetConfig didn't persist")
	}
}

func TestServiceSpeakerConfigSnapshot(t *testing.T) {
	s := newServiceForTest()
	if s.SpeakerConfigSnapshot().Enabled {
		t.Fatalf("expected disabled by default")
	}
	s.SetSpeakerConfig(SpeakerConfig{Enabled: true, Mode: "enforce", Threshold: 0.7})
	if !s.SpeakerConfigSnapshot().Enabled {
		t.Fatalf("SetSpeakerConfig didn't persist")
	}
}

func TestSpeakerConfigSnapshotIsDetached(t *testing.T) {
	s := newServiceForTest()
	input := SpeakerConfig{Enabled: true, ProfileIDs: []string{"p1"}}
	s.SetSpeakerConfig(input)
	input.ProfileIDs[0] = "mutated-after-set"

	got := s.SpeakerConfigSnapshot()
	if got.ProfileIDs[0] != "p1" {
		t.Fatalf("SetSpeakerConfig retained caller slice: %v", got.ProfileIDs)
	}
	got.ProfileIDs[0] = "mutated-snapshot"
	if again := s.SpeakerConfigSnapshot(); again.ProfileIDs[0] != "p1" {
		t.Fatalf("snapshot shares backing array with service state: %v", again.ProfileIDs)
	}
}

func TestWakeWordTemplateSnapshotIsDetached(t *testing.T) {
	s := newServiceForTest()
	input := &WakeWordTemplate{Samples: []AudioFeatures{{Kind: "original"}}}
	s.SetWakeWordTemplate(input)
	input.Samples[0].Kind = "mutated-after-set"

	got := s.GetWakeWordTemplate()
	if got == nil || got.Samples[0].Kind != "original" {
		t.Fatalf("SetWakeWordTemplate retained caller slice: %+v", got)
	}
	got.Samples[0].Kind = "mutated-snapshot"
	if again := s.GetWakeWordTemplate(); again.Samples[0].Kind != "original" {
		t.Fatalf("snapshot shares backing array with service state: %+v", again)
	}
}

func TestServicePathSetters(t *testing.T) {
	s := newServiceForTest()
	s.SetConfigPath("/tmp/cfg")
	s.SetSpeakerConfigPath("/tmp/spk")
	s.SetWakeWordPath("/tmp/ww")
	if s.ConfigPath() != "/tmp/cfg" || s.SpeakerConfigPath() != "/tmp/spk" || s.WakeWordPath() != "/tmp/ww" {
		t.Fatalf("path setters not threaded through")
	}
}

func TestServiceWhisperURLSetter(t *testing.T) {
	s := newServiceForTest()
	s.SetWhisperURL("http://other.test")
	// No direct getter; just exercise the setter for coverage.
	_ = s
}

func TestServiceGetWakeWordTemplate(t *testing.T) {
	s := newServiceForTest()
	if s.GetWakeWordTemplate() != nil {
		t.Fatalf("expected nil template initially")
	}
	tmpl := validTemplate()
	s.SetWakeWordTemplate(tmpl)
	got := s.GetWakeWordTemplate()
	if got == nil || got.Label != tmpl.Label {
		t.Fatalf("SetWakeWordTemplate not visible to Get: %+v", got)
	}
}

func TestServiceGetWakeWordTransport(t *testing.T) {
	s := newServiceForTest()
	c := s.GetWakeWord()
	if c.Configured {
		t.Fatalf("expected unconfigured")
	}
	s.SetWakeWordTemplate(validTemplate())
	c = s.GetWakeWord()
	if !c.Configured || c.TemplateJSON == "" {
		t.Fatalf("expected configured + JSON, got %+v", c)
	}
}

func TestServiceSpeakerClientAccessors(t *testing.T) {
	s := newServiceForTest()
	if s.SpeakerClient() != nil {
		t.Fatalf("expected nil")
	}
	sc := &SpeakerClient{}
	s.SetSpeakerClient(sc)
	if s.SpeakerClient() != sc {
		t.Fatalf("SetSpeakerClient round-trip failed")
	}
}
