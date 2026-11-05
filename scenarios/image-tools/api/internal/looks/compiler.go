package looks

import (
	"strings"

	"image-tools/internal/ai"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// Compile resolves a Look + a subject/prompt into the ordered, ready-to-submit
// op/AI request shapes. It is a pure function (no IO) and is the compose-seam
// other scenarios call: each CompiledStep names an operation, its engine
// (deterministic vs AI), and the fully-merged params (Look defaults < step
// params, with the prompt template filled for prompt-driven AI steps).
//
// requires_image/requires_mask are derived from the steps: every deterministic
// op edits the input image, and an AI op's needs come from the AI catalog.
func Compile(look *looksv1.Look, subject, prompt string, hasInput bool) *looksv1.CompileLookResponse {
	resp := &looksv1.CompileLookResponse{}
	if look == nil {
		return resp
	}
	resolvedPrompt := fillTemplate(look.GetPromptTemplate(), subject, prompt)
	hasAI := false

	for _, step := range look.GetSteps() {
		params := map[string]string{}
		// Look defaults first, then step params override.
		for k, v := range look.GetParams() {
			params[k] = v
		}
		for k, v := range step.GetParams() {
			params[k] = v
		}

		switch step.GetKind() {
		case looksv1.StepKind_STEP_KIND_AI:
			hasAI = true
			meta, known := ai.Get(step.GetOperation())
			if known && meta.PromptDriven {
				// Fill the prompt: an explicit step prompt wins; else the resolved
				// Look template; else the caller's free-form prompt.
				if params["prompt"] == "" {
					if resolvedPrompt != "" {
						params["prompt"] = resolvedPrompt
					} else if prompt != "" {
						params["prompt"] = prompt
					}
				}
				if resp.PrimaryPrompt == "" {
					resp.PrimaryPrompt = params["prompt"]
				}
			}
			if known && meta.RequiresImage {
				resp.RequiresImage = true
			}
			if known && meta.RequiresMask {
				resp.RequiresMask = true
			}
		default: // deterministic
			// Deterministic ops transform an input image.
			resp.RequiresImage = true
		}

		resp.Steps = append(resp.Steps, &looksv1.CompiledStep{
			Operation: step.GetOperation(),
			Kind:      step.GetKind(),
			Params:    params,
		})
	}

	if resp.RequiresImage && !hasInput {
		resp.Warnings = append(resp.Warnings, "this Look edits an input image, but none was provided")
	}
	if hasAI {
		resp.Warnings = append(resp.Warnings, "this Look contains model-backed steps; a backend + model weights must be installed (or run it via the Workspace, which surfaces the install/cost path)")
	}
	return resp
}

// fillTemplate substitutes the {subject} and {prompt} placeholders and collapses
// the leftover whitespace. An empty subject falls back to "the image" so the
// resolved instruction still reads naturally.
func fillTemplate(tmpl, subject, prompt string) string {
	if tmpl == "" {
		return ""
	}
	subj := strings.TrimSpace(subject)
	if subj == "" {
		subj = "the image"
	}
	out := strings.ReplaceAll(tmpl, "{subject}", subj)
	out = strings.ReplaceAll(out, "{prompt}", strings.TrimSpace(prompt))
	return strings.Join(strings.Fields(out), " ")
}
