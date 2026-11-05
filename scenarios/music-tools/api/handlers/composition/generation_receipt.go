package composition

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"music-tools/internal/styles"
)

// generationProfile contains bounded facts returned by a completed resource
// response. LyricsSHA256 records our exact input; it does not classify audio.
type generationProfile struct {
	Seed           uint32  `json:"seed"`
	Planner        string  `json:"planner"`
	InferenceSteps int     `json:"inference_steps"`
	GuidanceScale  float64 `json:"guidance_scale"`
	Shift          float64 `json:"shift"`
	LMCodeSHA256   string  `json:"lm_code_sha256,omitempty"`
	LMCodeCount    int     `json:"lm_code_count,omitempty"`
	LyricsSHA256   string  `json:"lyrics_sha256"`
}

func readGenerationProfile(h http.Header, seed int64, request composeRequest, compiled styles.Compiled) (generationProfile, error) {
	var p generationProfile
	fail := func(name string) (generationProfile, error) {
		return p, fmt.Errorf("resource generation receipt missing, invalid or mismatched: %s", name)
	}
	integer := func(name string, max uint64) (uint64, bool) {
		raw := h.Get(name)
		n, e := strconv.ParseUint(raw, 10, 64)
		return n, e == nil && n <= max && strconv.FormatUint(n, 10) == raw
	}
	n, ok := integer("X-ACE-Step-Seed", math.MaxUint32)
	if !ok || seed < 0 || uint64(seed) != n {
		return fail("seed")
	}
	p.Seed = uint32(n)
	p.Planner = h.Get("X-ACE-Step-Planner")
	if p.Planner != "acestep-5Hz-lm-0.6B" && p.Planner != "acestep-5Hz-lm-1.7B" {
		return fail("planner")
	}
	n, ok = integer("X-ACE-Step-Inference-Steps", 100)
	if !ok || n < 1 || (compiled.Params.Steps > 0 && int(n) != compiled.Params.Steps) {
		return fail("inference_steps")
	}
	p.InferenceSteps = int(n)
	number := func(name string, min, max float64) (float64, bool) {
		raw := h.Get(name)
		if len(raw) == 0 || len(raw) > 32 {
			return 0, false
		}
		v, e := strconv.ParseFloat(raw, 64)
		return v, e == nil && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= min && v <= max
	}
	if p.GuidanceScale, ok = number("X-ACE-Step-Guidance-Scale", 0, 20); !ok {
		return fail("guidance_scale")
	}
	// SFT applies authored CFG. Turbo normalizes effective CFG to1, so its
	// receipt legitimately differs. Use actual model identity for legacy styles
	// without an expected_variant; that identity was validated before this call.
	if compiled.Params.Guidance != 0 && strings.HasSuffix(h.Get("X-ACE-Step-Model-ID"), ":acestep-v15-sft") && p.GuidanceScale != compiled.Params.Guidance {
		return fail("guidance_scale does not match authored SFT guidance")
	}
	if p.Shift, ok = number("X-ACE-Step-Shift", 1, 5); !ok || (request.Shift != nil && p.Shift != *request.Shift) {
		return fail("shift")
	}
	p.LMCodeSHA256 = h.Get("X-ACE-Step-LM-Code-SHA256")
	count := h.Get("X-ACE-Step-LM-Code-Count")
	if p.LMCodeSHA256 != "" || count != "" {
		n, ok = integer("X-ACE-Step-LM-Code-Count", 100000)
		if !ok || n < 1 || len(p.LMCodeSHA256) != 64 || strings.IndexFunc(p.LMCodeSHA256, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) >= 0 {
			return fail("lm_code")
		}
		p.LMCodeCount = int(n)
	}
	p.LyricsSHA256 = lyricsHash(styles.ResolveLyrics(compiled.Lyrics))
	return p, nil
}
