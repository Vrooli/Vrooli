package ops

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// TestRunCommandsMatchGeneratedSpecs asserts the command surface is exactly the
// generated table plus the two layered views. The per-operation command list is
// no longer restated here: it is rendered from OpParams, so a hand-written copy
// could only drift.
func TestRunCommandsMatchGeneratedSpecs(t *testing.T) {
	h := newHandlers(nil) // runCommands only captures h in closures; not invoked here
	seen := make(map[string]struct{})
	for _, c := range h.runCommands() {
		if c.RunCtx == nil {
			t.Errorf("command %q has no RunCtx handler", c.Name)
		}
		if _, dup := seen[c.Name]; dup {
			t.Errorf("command %q is registered twice", c.Name)
		}
		seen[c.Name] = struct{}{}
	}
	for _, spec := range generatedRunSpecs {
		if _, ok := seen[spec.Op]; !ok {
			t.Errorf("generated operation %q has no CLI command", spec.Op)
		}
	}
	for _, layered := range []string{"composite", "parallax"} {
		if _, ok := seen[layered]; !ok {
			t.Errorf("layered operation %q has no CLI command", layered)
		}
	}
	if want := len(generatedRunSpecs) + 2; len(seen) != want {
		t.Errorf("command count = %d, want %d (generated + 2 layered)", len(seen), want)
	}
}

// TestFlagDescriptionsCoverGeneratedFlags keeps `--help` complete. The flag set
// is generated from the wire contract, so a new parameter would otherwise reach
// the CLI undocumented.
func TestFlagDescriptionsCoverGeneratedFlags(t *testing.T) {
	for _, spec := range generatedRunSpecs {
		for _, f := range spec.Flags {
			if flagDescriptions[f.Name] == "" {
				t.Errorf("operation %q flag --%s has no description", spec.Op, f.Name)
			}
		}
	}
}

// TestBuildParamsBindsFlagsToWireFields drives the descriptor-driven binder for
// three shapes — plain scalars, booleans and doubles, and repeated values — so a
// wrong flag→field binding fails here rather than as a rejected render.
func TestBuildParamsBindsFlagsToWireFields(t *testing.T) {
	resize := specFor(t, "resize")
	params, err := buildParams(resize, cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:    resizeArgs(resize),
		Flags:     map[string]string{"width": "640", "height": "480", "fit": "fill"},
		BoolFlags: map[string]bool{},
	}))
	if err != nil {
		t.Fatalf("buildParams(resize): %v", err)
	}
	got := params.GetResize()
	if got == nil {
		t.Fatalf("resize params are unset: %s", params)
	}
	if got.GetWidth() != 640 || got.GetHeight() != 480 || got.GetFit() != "fill" {
		t.Errorf("resize params = %v, want width=640 height=480 fit=fill", got)
	}

	canny := specFor(t, "canny")
	params, err = buildParams(canny, cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:    resizeArgs(canny),
		Flags:     map[string]string{"amount": "1.5", "high-threshold": "200"},
		BoolFlags: map[string]bool{},
	}))
	if err != nil {
		t.Fatalf("buildParams(canny): %v", err)
	}
	if got := params.GetCanny(); got == nil || got.GetAmount() != 1.5 || got.GetHighThreshold() != 200 {
		t.Errorf("canny params = %v, want amount=1.5 high_threshold=200", got)
	}

	vectorize := specFor(t, "vectorize")
	params, err = buildParams(vectorize, cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:    resizeArgs(vectorize),
		Flags:     map[string]string{"colors": "6", "smoothing": ""},
		FlagLists: map[string][]string{"keep-colors": {"#ffffff", "#22d3ee"}},
		BoolFlags: map[string]bool{"smoothing": true},
	}))
	if err != nil {
		t.Fatalf("buildParams(vectorize): %v", err)
	}
	vec := params.GetVectorize()
	if vec == nil {
		t.Fatalf("vectorize params are unset: %s", params)
	}
	if vec.GetColors() != 6 || !vec.GetSmoothing() {
		t.Errorf("vectorize params = %v, want colors=6 smoothing=true", vec)
	}
	if want := []string{"#ffffff", "#22d3ee"}; len(vec.GetKeepColors()) != len(want) ||
		vec.GetKeepColors()[0] != want[0] || vec.GetKeepColors()[1] != want[1] {
		t.Errorf("vectorize keep_colors = %v, want %v", vec.GetKeepColors(), want)
	}
}

func specFor(t *testing.T, op string) runSpec {
	t.Helper()
	for _, spec := range generatedRunSpecs {
		if spec.Op == op {
			return spec
		}
	}
	t.Fatalf("no generated spec for %q", op)
	return runSpec{}
}

// resizeArgs builds the ArgSchema a spec's command declares, so the test context
// resolves flags exactly as the parser would.
func resizeArgs(spec runSpec) cliapp.ArgSchema {
	flags := make([]cliapp.Flag, 0, len(spec.Flags))
	for _, f := range spec.Flags {
		flags = append(flags, flagConnector(f))
	}
	return cliapp.ArgSchema{Positionals: []cliapp.Positional{inputArg()}, Flags: flags}
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
