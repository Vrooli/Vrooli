package ai

import (
	"regexp"
	"strings"
)

// identityInstruction is appended to an edit_instruct prompt. In the 2026-11-21
// showcase "the same guinea pig sitting on the grass in a sunny park" made FLUX,
// Gemini and Seedream each draw a different-looking animal, and adding "keeping
// its exact pose, face and fur markings" fixed all three. The wording keeps the
// colours only when the instruction does not restyle the image, so it does not
// fight an anime or painting edit.
const identityInstruction = "Preserve the subject's identity: keep its exact pose, face and distinguishing markings (and, unless the instruction restyles the image, its colors and proportions). Change only what the instruction asks for."

// Identity-preservation states reported in the job meta.
const (
	identityApplied        = "applied"
	identityAlreadyPresent = "already_present"
	identityOff            = "off"
)

// identityStated matches a prompt that already asks for the subject to be kept,
// so the instruction is not stacked on top of the caller's own.
var identityStated = regexp.MustCompile(`(?i)\b(keep|keeping|preserv\w*|retain\w*|maintain\w*)\b[^.]{0,60}\b(exact|exactly|identity|pose|face|markings?|same)\b|\bexactly the same\b|\bsame (pose|face|markings?)\b`)

// effectivePrompt returns the prompt the backend receives and which state
// produced it. preserve=false sends the prompt exactly as written.
func effectivePrompt(prompt string, preserve bool) (effective, state string) {
	trimmed := strings.TrimSpace(prompt)
	switch {
	case !preserve:
		return prompt, identityOff
	case trimmed == "":
		return prompt, identityOff
	case identityStated.MatchString(trimmed):
		return prompt, identityAlreadyPresent
	}
	sep := " "
	if !strings.HasSuffix(trimmed, ".") && !strings.HasSuffix(trimmed, "!") && !strings.HasSuffix(trimmed, "?") {
		sep = ". "
	}
	return trimmed + sep + identityInstruction, identityApplied
}

// editPromptParams applies identity preservation to an edit_instruct request
// and returns the params to run with plus the meta that records the effective
// prompt. The input map is not modified.
func editPromptParams(params map[string]string, preserve *bool) (map[string]string, map[string]string) {
	original := params["prompt"]
	effective, state := effectivePrompt(original, preserve == nil || *preserve)
	out := cloneParams(params)
	out["prompt"] = effective
	meta := map[string]string{"identity_preservation": state, "effective_prompt": effective}
	if effective != original {
		meta["prompt_original"] = original
	}
	return out, meta
}
