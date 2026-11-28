package ai

import "strings"

// colourGuardOps are the image-producing operations whose prompt can silently
// turn monochrome. A fine-art photograph prompt with a "medium format film look"
// produced a black-and-white generation (mean saturation 0.13) while the same
// seed and model with ", colour photograph, not monochrome" produced colour
// (0.27). The guard is a wording default, not a model setting; it is recorded on
// the job so it is never hidden.
var colourGuardOps = map[string]bool{
	"text_to_image":      true,
	"image_to_image":     true,
	"inpaint":            true,
	"outpaint":           true,
	"background_replace": true,
	"garment_replace":    true,
	"edit_instruct":      true,
}

const colourGuardPrompt = ", colour photograph, not monochrome"

// colourPromptParams appends the colour instruction to a photoreal prompt that
// did not ask for a monochrome or non-photographic look. An explicit request
// (black and white, sepia, noir) or an illustration/anime/sketch prompt is left
// exactly as written. The effective prompt is the caller's with the suffix.
func colourPromptParams(op string, params map[string]string) (map[string]string, map[string]string) {
	if !colourGuardOps[op] {
		return params, nil
	}
	prompt := strings.TrimSpace(params["prompt"])
	if prompt == "" {
		return params, nil
	}
	lower := strings.ToLower(prompt)
	if mentionsMonochrome(lower) || !looksPhotoreal(lower) {
		return params, nil
	}
	out := make(map[string]string, len(params)+2)
	for k, v := range params {
		out[k] = v
	}
	out["prompt"] = prompt + colourGuardPrompt
	// The measured case needed both sides: the positive phrase alone still came
	// back monochrome on one seed, while the phrase plus these negative terms
	// doubled the chroma. Leave an existing negative prompt in place.
	negative := strings.TrimSpace(params["negative_prompt"])
	if negative != "" {
		negative += ", "
	}
	out["negative_prompt"] = negative + "monochrome, black and white, grayscale"
	return out, map[string]string{
		"prompt_colour_guard": "appended colour photograph, not monochrome (prompt and negative)",
	}
}

func mentionsMonochrome(prompt string) bool {
	for _, phrase := range []string{
		"black and white", "black-and-white", "black & white", "b&w", "monochrome",
		"grayscale", "greyscale", "sepia", "noir", "silhouette",
	} {
		if strings.Contains(prompt, phrase) {
			return true
		}
	}
	return false
}

// looksPhotoreal is deliberately conservative: a prompt that mixes a photo word
// with an illustration word is ambiguous and is left alone.
func looksPhotoreal(prompt string) bool {
	photo := false
	for _, word := range []string{"photo", "photograph", "photoreal", "realistic", "portrait"} {
		if strings.Contains(prompt, word) {
			photo = true
			break
		}
	}
	if !photo {
		return false
	}
	for _, word := range []string{
		"illustration", "anime", "cartoon", "drawing", "sketch", "painting",
		"watercolour", "watercolor", "line art", "3d render", "render of",
	} {
		if strings.Contains(prompt, word) {
			return false
		}
	}
	return true
}
