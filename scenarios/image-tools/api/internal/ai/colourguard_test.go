package ai

import (
	"strings"
	"testing"
)

func TestColourGuardAddsColourToAPhotorealPrompt(t *testing.T) {
	params := map[string]string{
		"prompt":          "Fine-art nude photograph of a 32-year-old woman, medium format film look",
		"negative_prompt": "deformed, blurry",
	}
	got, meta := colourPromptParams("text_to_image", params)
	if !strings.Contains(got["prompt"], "colour photograph, not monochrome") {
		t.Fatalf("prompt = %q", got["prompt"])
	}
	if !strings.Contains(got["negative_prompt"], "monochrome, black and white, grayscale") {
		t.Fatalf("negative prompt = %q", got["negative_prompt"])
	}
	if !strings.Contains(got["negative_prompt"], "deformed, blurry") {
		t.Fatalf("the original negative prompt was lost: %q", got["negative_prompt"])
	}
	if meta["prompt_colour_guard"] == "" {
		t.Fatal("the guard must be recorded on the job")
	}
	if params["prompt"] == got["prompt"] {
		t.Fatal("the input map must not be mutated")
	}
}

func TestColourGuardLeavesAnExplicitMonochromeRequestAlone(t *testing.T) {
	for _, prompt := range []string{
		"black and white portrait photograph",
		"a monochrome studio photo",
		"sepia-toned photograph",
		"a grayscale noir portrait",
	} {
		got, meta := colourPromptParams("text_to_image", map[string]string{"prompt": prompt})
		if got["prompt"] != prompt || meta != nil {
			t.Errorf("%q changed to %q (meta %v)", prompt, got["prompt"], meta)
		}
	}
}

func TestColourGuardLeavesIllustrationPromptsAlone(t *testing.T) {
	for _, prompt := range []string{
		"anime portrait of a woman",
		"pencil sketch of a face",
		"a realistic-looking oil painting",
		"line art of a standing figure",
	} {
		got, meta := colourPromptParams("text_to_image", map[string]string{"prompt": prompt})
		if got["prompt"] != prompt || meta != nil {
			t.Errorf("%q changed to %q (meta %v)", prompt, got["prompt"], meta)
		}
	}
}

func TestColourGuardDoesNotTouchANonPhotorealOrNonGenerationPrompt(t *testing.T) {
	text := "a serene landscape with rolling hills"
	if got, meta := colourPromptParams("text_to_image", map[string]string{"prompt": text}); got["prompt"] != text || meta != nil {
		t.Fatalf("a non-photoreal prompt changed: %q", got["prompt"])
	}
	photo := "a portrait photograph"
	if got, meta := colourPromptParams("caption", map[string]string{"prompt": photo}); got["prompt"] != photo || meta != nil {
		t.Fatalf("a non-generation op changed: %q", got["prompt"])
	}
}
