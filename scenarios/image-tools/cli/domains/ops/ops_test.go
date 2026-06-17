package ops

import (
	"sort"
	"testing"
)

// TestRunCommandsCoverReq01 asserts the CLI exposes one run command per
// deterministic operation (the headless surface req IMG-P0-001 requires). The
// live end-to-end behavior — every op running from the CLI against a started
// scenario with no UI/ComfyUI/GPU — is exercised by the BAS smoke flow and the
// documented headless-completeness acceptance (docs/internal/TESTING.md).
func TestRunCommandsCoverReq01(t *testing.T) {
	h := newHandlers(nil) // runCommands only captures h in closures; not invoked here
	want := []string{
		"adjust", "canvas", "compress", "convert", "crop", "deskew",
		"filter", "flip", "metadata", "overlay", "resize", "rotate", "thumbnail",
	}
	got := make([]string, 0)
	for _, c := range h.runCommands() {
		if c.RunCtx == nil {
			t.Errorf("command %q has no RunCtx handler", c.Name)
		}
		got = append(got, c.Name)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("run commands = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("run commands = %v, want %v", got, want)
		}
	}
}

func TestExtToFormat(t *testing.T) {
	cases := map[string]string{
		"out.png": "png", "out.JPG": "jpeg", "a/b.jpeg": "jpeg", "x.webp": "webp",
		"y.tif": "tiff", "z.avif": "avif", "n.bmp": "bmp", "g.gif": "gif",
		"noext": "", "x.heic": "", "x.svg": "",
	}
	for in, want := range cases {
		if got := extToFormat(in); got != want {
			t.Errorf("extToFormat(%q) = %q, want %q", in, got, want)
		}
	}
}
