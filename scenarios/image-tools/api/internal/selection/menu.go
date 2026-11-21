package selection

import "image-tools/internal/ai"

// SuggestedEdit is one contextual edit for a region — a compiled, ready-to-submit
// AI request shape. The caller submits it to the AI edge with the segment mask
// as the `mask` part when RequiresMask.
type SuggestedEdit struct {
	ID             string
	Label          string
	Description    string
	Operation      string
	Prompt         string
	RequiresPrompt bool
	RequiresMask   bool
	Params         map[string]string
}

// RegionClass is one class and its contextual edit menu.
type RegionClass struct {
	Name    string
	Summary string
	Edits   []SuggestedEdit
}

// classOrder is the stable display order for ListClasses.
var classOrder = []string{ClassPerson, ClassSky, ClassFoliage, ClassBackground, ClassObject}

// menus is the contextual edit table: region class → ordered edits. Each edit
// names an AI op that EXISTS in the catalog (asserted by a test) and pre-fills a
// prompt. requires_mask reflects whether the op consumes the selection mask
// (whole-image ops like background_removal do not). This is the heart of the
// compose-seam: it encodes "what edits make sense for a sky / a person / an
// object" once, so callers don't re-derive it.
var menus = map[string]RegionClass{
	ClassPerson: {
		Name:    ClassPerson,
		Summary: "A skin-tone-dominant region (a person/subject). Never identity recognition.",
		Edits: []SuggestedEdit{
			{ID: "naturalize_skin", Label: "Naturalize skin", Description: "Re-add realistic pore-level texture to plasticky/over-smoothed skin.", Operation: "edit_instruct", Prompt: "make the skin look natural and realistic with subtle pores and texture, remove the plastic over-smoothed look, keep the same person", RequiresMask: true},
			{ID: "change_clothing", Label: "Change clothing…", Description: "Replace the subject's outfit (describe it).", Operation: "edit_instruct", Prompt: "change the clothing to ", RequiresPrompt: true, RequiresMask: true},
			{ID: "adjust_lighting", Label: "Soften lighting on subject", Description: "Even out harsh lighting on the selected subject.", Operation: "edit_instruct", Prompt: "soften and even out the lighting on the selected subject, keep the same person and pose", RequiresMask: true},
			{ID: "remove", Label: "Remove person", Description: "Remove the subject and fill the gap from the surroundings.", Operation: "object_removal", RequiresMask: true},
		},
	},
	ClassSky: {
		Name:    ClassSky,
		Summary: "An upper, bright, low-saturation or bluish region (the sky).",
		Edits: []SuggestedEdit{
			{ID: "replace", Label: "Replace sky…", Description: "Regenerate the sky region from a description.", Operation: "inpaint", Prompt: "a dramatic sunset sky with soft clouds", RequiresMask: true},
			{ID: "clear_blue", Label: "Make sky clear blue", Description: "Replace with a clear blue daytime sky.", Operation: "inpaint", Prompt: "a clear blue daytime sky with a few soft clouds", RequiresMask: true},
			{ID: "recolor", Label: "Recolor sky…", Description: "Shift the sky's colour/mood (describe it).", Operation: "edit_instruct", Prompt: "change the sky to ", RequiresPrompt: true, RequiresMask: true},
		},
	},
	ClassFoliage: {
		Name:    ClassFoliage,
		Summary: "A green-dominant region (plants/trees/grass).",
		Edits: []SuggestedEdit{
			{ID: "lush", Label: "Make foliage lush", Description: "Make the foliage greener and healthier.", Operation: "edit_instruct", Prompt: "make the foliage lush, green and healthy", RequiresMask: true},
			{ID: "autumn", Label: "Autumn colors", Description: "Shift the foliage to autumn tones.", Operation: "edit_instruct", Prompt: "change the foliage to warm autumn colors (orange, red, gold)", RequiresMask: true},
			{ID: "remove", Label: "Remove foliage", Description: "Remove the foliage and fill the gap.", Operation: "object_removal", RequiresMask: true},
		},
	},
	ClassBackground: {
		Name:    ClassBackground,
		Summary: "A large region touching most of the frame (the background).",
		Edits: []SuggestedEdit{
			{ID: "remove_bg", Label: "Remove background", Description: "Cut the subject out to transparency (whole-image).", Operation: "background_removal"},
			{ID: "replace", Label: "Replace background…", Description: "Regenerate the background region from a description.", Operation: "inpaint", Prompt: "a clean studio backdrop", RequiresMask: true},
			{ID: "blur", Label: "Blur background", Description: "Soften the background for a shallow-depth look.", Operation: "edit_instruct", Prompt: "blur the background to create a shallow depth of field, keep the subject sharp", RequiresMask: true},
		},
	},
	ClassObject: {
		Name:    ClassObject,
		Summary: "A generic foreground object (the fallback class).",
		Edits: []SuggestedEdit{
			{ID: "remove", Label: "Remove this object", Description: "Remove the selected object and fill the gap.", Operation: "object_removal", RequiresMask: true},
			{ID: "replace", Label: "Replace with…", Description: "Regenerate the selection from a description.", Operation: "inpaint", Prompt: "", RequiresPrompt: true, RequiresMask: true},
			{ID: "recolor", Label: "Recolor…", Description: "Change the object's color/material (describe it).", Operation: "edit_instruct", Prompt: "change the selected object to ", RequiresPrompt: true, RequiresMask: true},
		},
	},
}

// ListClasses returns every region class + its contextual edit menu, in stable
// display order. Pure.
func ListClasses() []RegionClass {
	out := make([]RegionClass, 0, len(classOrder))
	for _, name := range classOrder {
		out = append(out, menus[name])
	}
	return out
}

// Suggest returns the resolved class name and its contextual edit menu. An
// unknown/empty class resolves to the generic "object" menu. Pure.
func Suggest(class string) (resolved string, edits []SuggestedEdit) {
	rc, ok := menus[class]
	if !ok {
		rc = menus[ClassObject]
	}
	return rc.Name, append([]SuggestedEdit(nil), rc.Edits...)
}

// menuOps reports the distinct AI operations referenced by the menus (for the
// catalog-consistency test).
func menuOps() map[string]bool {
	ops := map[string]bool{}
	for _, rc := range menus {
		for _, e := range rc.Edits {
			ops[e.Operation] = true
		}
	}
	return ops
}

// knownAIOp is the seam the catalog-consistency test asserts against (so the
// menu can never reference an op that does not exist).
var knownAIOp = ai.Has
