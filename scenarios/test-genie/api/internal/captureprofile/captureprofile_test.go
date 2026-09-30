package captureprofile

import "testing"

func TestResolveDefaultIsCheapest(t *testing.T) {
	for _, name := range []string{"", "   "} {
		p, ok := Resolve(name)
		if !ok {
			t.Fatalf("Resolve(%q) ok = false, want true", name)
		}
		if p.AllPages || p.Video {
			t.Fatalf("default profile must not enable all-pages/video: %+v", p)
		}
		if p.DiagnosticsPreset() != "" {
			t.Fatalf("default profile must not force a diagnostics preset, got %q", p.DiagnosticsPreset())
		}
	}
}

func TestResolveBaseline(t *testing.T) {
	p, ok := Resolve("baseline")
	if !ok {
		t.Fatal("baseline must be recognized")
	}
	if !p.AllPages || !p.Video {
		t.Fatalf("baseline must enable all-pages + video: %+v", p)
	}
	if p.DiagnosticsPreset() != "full" {
		t.Fatalf("baseline diagnostics preset = %q, want full", p.DiagnosticsPreset())
	}
}

func TestResolveCaseInsensitive(t *testing.T) {
	p, ok := Resolve("BASELINE")
	if !ok || !p.AllPages {
		t.Fatalf("baseline should be case-insensitive: ok=%v %+v", ok, p)
	}
}

func TestResolveUnknownFallsToDefault(t *testing.T) {
	p, ok := Resolve("turbo")
	if ok {
		t.Fatal("unknown profile should report ok=false")
	}
	if p.AllPages || p.Video {
		t.Fatalf("unknown profile must degrade to default depth: %+v", p)
	}
}
