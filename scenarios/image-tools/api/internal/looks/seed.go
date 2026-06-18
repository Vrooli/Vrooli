package looks

import (
	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// det builds a deterministic (internal/ops) step.
func det(op string, params map[string]string) *looksv1.LookStep {
	return &looksv1.LookStep{Operation: op, Kind: looksv1.StepKind_STEP_KIND_DETERMINISTIC, Params: params}
}

// ai builds a model-backed (internal/ai) step.
func aiStep(op string, params map[string]string) *looksv1.LookStep {
	if params == nil {
		params = map[string]string{}
	}
	return &looksv1.LookStep{Operation: op, Kind: looksv1.StepKind_STEP_KIND_AI, Params: params}
}

// BuiltinLooks returns the seeded, read-only Look library. Film/camera Looks are
// pure-deterministic (their RenderPreview runs fully on any host with no model);
// style/enhance Looks carry AI steps whose preview is deferred to a Workspace
// run. The set is deliberately small and curated — the library is meant to grow
// via operator-created custom Looks (CreateLook), which this baseline seeds the
// vocabulary for.
//
// Order here is the stable display order (built-ins always sort before custom).
func BuiltinLooks() []*looksv1.Look {
	return []*looksv1.Look{
		{
			Id:          "polaroid-600",
			Name:        "Polaroid 600",
			Description: "Warm, faded instant-film grade — lifted blacks, soft contrast, creamy midtones.",
			Kind:        looksv1.LookKind_LOOK_KIND_FILM,
			Builtin:     true,
			Steps: []*looksv1.LookStep{
				det("adjust", map[string]string{"brightness": "6", "contrast": "-8", "saturation": "-12", "gamma": "1.08"}),
			},
		},
		{
			Id:          "noir",
			Name:        "Noir",
			Description: "High-contrast black & white — dramatic, filmic monochrome.",
			Kind:        looksv1.LookKind_LOOK_KIND_FILM,
			Builtin:     true,
			Steps: []*looksv1.LookStep{
				det("filter", map[string]string{"filter": "grayscale"}),
				det("adjust", map[string]string{"contrast": "18"}),
			},
		},
		{
			Id:          "old-ipod",
			Name:        "Early Camera Phone",
			Description: "Washed, soft, low-saturation look of a 2000s phone camera.",
			Kind:        looksv1.LookKind_LOOK_KIND_CAMERA,
			Builtin:     true,
			Steps: []*looksv1.LookStep{
				det("adjust", map[string]string{"saturation": "-25", "contrast": "-6"}),
				det("filter", map[string]string{"filter": "blur", "amount": "0.6"}),
			},
		},
		{
			Id:          "golden-hour",
			Name:        "Golden Hour",
			Description: "Warm sunset color grade — amber highlights, gentle glow.",
			Kind:        looksv1.LookKind_LOOK_KIND_FILM,
			Builtin:     true,
			Steps: []*looksv1.LookStep{
				det("adjust", map[string]string{"brightness": "5", "saturation": "12", "hue": "8", "gamma": "0.95"}),
			},
		},
		{
			Id:          "vivid-pop",
			Name:        "Vivid Pop",
			Description: "Punchy, saturated, high-contrast — social-ready vibrance.",
			Kind:        looksv1.LookKind_LOOK_KIND_CAMERA,
			Builtin:     true,
			Steps: []*looksv1.LookStep{
				det("adjust", map[string]string{"saturation": "30", "contrast": "12"}),
			},
		},
		{
			Id:             "anime",
			Name:           "Anime",
			Description:    "Redraw the subject in clean anime style — cel shading, vibrant flat colors. (model-backed)",
			Kind:           looksv1.LookKind_LOOK_KIND_STYLE,
			Builtin:        true,
			PromptTemplate: "Redraw {subject} in a clean anime art style: cel shading, bold outlines, vibrant flat colors. {prompt}",
			Params:         map[string]string{"strength": "0.6"},
			Steps: []*looksv1.LookStep{
				aiStep("edit_instruct", nil),
			},
		},
		{
			Id:             "photoreal",
			Name:           "Photoreal",
			Description:    "Make the subject look like a real photograph — natural lighting, real skin texture. (model-backed)",
			Kind:           looksv1.LookKind_LOOK_KIND_STYLE,
			Builtin:        true,
			PromptTemplate: "Make {subject} look like a photorealistic photograph: natural lighting, realistic skin texture and pores, true-to-life color. {prompt}",
			Params:         map[string]string{"strength": "0.5"},
			Steps: []*looksv1.LookStep{
				aiStep("edit_instruct", nil),
			},
		},
		{
			Id:          "restore-natural",
			Name:        "Restore & Naturalize",
			Description: "Upscale, then re-add realistic micro-texture so the result doesn't look plastic. (model-backed)",
			Kind:        looksv1.LookKind_LOOK_KIND_ENHANCE,
			Builtin:     true,
			Steps: []*looksv1.LookStep{
				aiStep("upscale", map[string]string{"scale": "2"}),
				aiStep("naturalize", map[string]string{"realism": "0.6", "face_aware": "true"}),
			},
		},
	}
}
