package imageengine

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateParamsAcceptsWhatTheEngineTakes(t *testing.T) {
	for _, tc := range []struct{ op, raw string }{
		{"duotone", `{"dark":"#1B3FD8","light":"#EDE6D2","normalize":true}`},
		{"halftone", `{"lpi":72,"angle":15,"dot":"circle"}`},
		{"posterize", `{"levels":7,"normalize":true}`},
		{"dither_diffusion", `{"dark":"#111827","light":"#fef3c7","normalize":true}`},
		{"line_screen", `{"spacing":6,"angle":45,"dark":"#1B3FD8","light":"#EDE6D2"}`},
		{"engraving", `{"spacing":7,"dark":"#1B3FD8","light":"#EDE6D2"}`},
		{"ascii_mosaic", `{"block_size":7,"dark":"#1B3FD8","light":"#EDE6D2"}`},
		{"grain", `{"amount":0.05,"contrast_multiplier":1.0,"seed":11}`},
		{"aberration", `{"distance":9}`},
		{"displacement", `{"amplitude":18,"spacing":26,"seed":5}`},
		// An unresolved brand slot is legitimate at write time: it lands in a
		// string field and binds when a brand is selected.
		{"duotone", `{"dark":"$brand.primary","light":"$brand.background"}`},
		// No override at all is legitimate — the op runs on its defaults.
		{"grain", ``},
	} {
		require.NoErrorf(t, ValidateParams(tc.op, tc.raw), "op %s raw %s", tc.op, tc.raw)
	}
}

// TestValidateParamsRejectsTheDefectThatShipped is the direct regression test.
// These are the exact parameter shapes authored on 2026-08-12 against a proto
// that had no field for them: they stored cleanly and failed at first render.
func TestValidateParamsRejectsUnknownFields(t *testing.T) {
	for _, tc := range []struct{ op, raw, want string }{
		{"duotone", `{"dark":"#111","nonexistent_knob":true}`, "not accepted by image-tools"},
		{"halftone", `{"lpi":72,"screen_angle":15}`, "not accepted by image-tools"},
		{"bloom", `{"treshold":0.6}`, "not accepted by image-tools"},
	} {
		err := ValidateParams(tc.op, tc.raw)
		require.Errorf(t, err, "op %s should have been rejected", tc.op)
		require.Contains(t, err.Error(), tc.want)
		// The rejected payload is echoed so the author can see what was sent.
		require.Contains(t, err.Error(), tc.raw)
	}
}

func TestValidateParamsRejectsMalformedAndNonObject(t *testing.T) {
	// Malformed JSON used to fall through the merge step and silently ship the
	// default, so a style promised a look it never rendered.
	require.ErrorContains(t, ValidateParams("duotone", `{"dark":`), "must be a JSON object")
	require.ErrorContains(t, ValidateParams("duotone", `"just-a-string"`), "must be a JSON object")
	require.ErrorContains(t, ValidateParams("duotone", `[1,2,3]`), "must be a JSON object")
	require.ErrorContains(t, ValidateParams("", `{}`), "operation is required")
}

func TestValidateChainRejectsParamsForOperationsTheStyleDoesNotRun(t *testing.T) {
	err := ValidateChain([]string{"duotone", "halftone"}, map[string]string{
		"duotone":  `{"normalize":true}`,
		"posterze": `{"levels":5}`, // typo for "posterize"
	})
	require.ErrorContains(t, err, `operation "posterze"`)
	require.ErrorContains(t, err, "does not run")
	// The declared chain is named so the author can see the intended spelling.
	require.ErrorContains(t, err, "duotone, halftone")

	require.NoError(t, ValidateChain([]string{"duotone"}, map[string]string{"duotone": `{"normalize":true}`}))
	require.NoError(t, ValidateChain([]string{"duotone"}, nil))
}

// TestValidateChainErrorsAreStable guards against map-iteration order leaking
// into the message an author sees.
func TestValidateChainErrorsAreStable(t *testing.T) {
	params := map[string]string{"aaa": `{}`, "zzz": `{}`, "mmm": `{}`}
	first := ValidateChain([]string{"duotone"}, params)
	require.Error(t, first)
	for i := 0; i < 20; i++ {
		require.Equal(t, first.Error(), ValidateChain([]string{"duotone"}, params).Error())
	}
	require.True(t, strings.Contains(first.Error(), `"aaa"`), "expected the first operation in sorted order")
}
