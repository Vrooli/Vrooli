package capability

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Stage kinds an extent may declare.
const (
	// ExtentSwap runs the operation's own model: it produces the first image.
	ExtentSwap = "swap"
	// ExtentRecolour takes the colour of a region of the reference onto the same
	// region of the working image (segment both, then colour transfer).
	ExtentRecolour = "recolour"
	// ExtentRestyle has a general editor redo a region with the reference's look
	// (the hairstyle), then lays only that region of its result into the working
	// image, so the face the swap produced is not re-rendered.
	ExtentRestyle = "restyle"
)

// ExtentSpec declares how much of a person an operation transfers. Each stage is
// one independent change (the inner face, skin tone, hair colour, the whole
// head); presets are named compositions of stages. The UI, the CLI and the
// engine all read this declaration, so adding a stage is data plus, when it
// needs a new kind, one executor.
type ExtentSpec struct {
	// Stages are listed in execution order: a swap stage first, then the stages
	// that adjust its result.
	Stages  []ExtentStage  `json:"stages"`
	Presets []ExtentPreset `json:"presets"`
	// DefaultPreset applies when the request names neither a preset nor stages.
	DefaultPreset string `json:"default_preset"`
}

// ExtentStage is one declared stage.
type ExtentStage struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	LabelKey string `json:"label_key"`
	DescKey  string `json:"desc_key"`
	// Replaces names the stage this one stands in for (the whole head replaces the
	// inner face), so both can never run.
	Replaces string `json:"replaces,omitempty"`
	// Model pins the model that runs a swap stage (a general editor for the
	// whole head) and Prompt is the instruction it is given.
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	// StrengthParam is the op parameter that scales a swap stage, 0..1.
	StrengthParam string `json:"strength_param,omitempty"`
	// Region and ReferenceRegion are the phrases that select the part of the
	// working image and of the reference a recolour stage matches; MaskModel is
	// the parser that resolves them. A restyle stage uses Region alone: the part
	// of the working image and of the editor's result that it lays in.
	Region          string `json:"region,omitempty"`
	ReferenceRegion string `json:"reference_region,omitempty"`
	MaskModel       string `json:"mask_model,omitempty"`
	// ReferenceFrom is where a recolour stage reads its colour: ReferenceImage
	// (default) or ReferenceWorking.
	ReferenceFrom string `json:"reference_from,omitempty"`
	// Transfer is the colour-transfer parameters of a recolour stage.
	Transfer map[string]float64 `json:"transfer,omitempty"`
	// Option names the op parameter that picks how a stage runs (skin_target:
	// source or body); Variants declare each choice and DefaultVariant applies when
	// the request does not name one. A variant overrides the stage's own region,
	// reference and transfer fields.
	Option         string                   `json:"option,omitempty"`
	DefaultVariant string                   `json:"default_variant,omitempty"`
	Variants       map[string]ExtentVariant `json:"variants,omitempty"`
	// Composite is the mask-composite parameters of a restyle stage.
	Composite map[string]float64 `json:"composite,omitempty"`
}

// Where a recolour stage reads the colour it moves from.
const (
	// ReferenceImage is the reference photo the request supplies (the source person).
	ReferenceImage = "reference"
	// ReferenceWorking is the working image itself (the target's own body).
	ReferenceWorking = "working"
)

// ExtentVariant is one way a stage can run. Empty fields keep the stage's own.
type ExtentVariant struct {
	LabelKey string `json:"label_key"`
	DescKey  string `json:"desc_key"`
	// Region and ReferenceRegion are the phrases selecting the part of the working
	// image to recolour and the part of the colour source to take the colour from.
	Region          string `json:"region,omitempty"`
	ReferenceRegion string `json:"reference_region,omitempty"`
	// ReferenceFrom is ReferenceImage (default) or ReferenceWorking.
	ReferenceFrom string `json:"reference_from,omitempty"`
	// Transfer overrides entries of the stage's colour-transfer parameters.
	Transfer map[string]float64 `json:"transfer,omitempty"`
}

// ExtentPreset is a named set of stages.
type ExtentPreset struct {
	ID       string   `json:"id"`
	LabelKey string   `json:"label_key"`
	DescKey  string   `json:"desc_key"`
	Stages   []string `json:"stages"`
}

// Stage returns the declared stage with this id.
func (s ExtentSpec) Stage(id string) (ExtentStage, bool) {
	for _, st := range s.Stages {
		if st.ID == id {
			return st, true
		}
	}
	return ExtentStage{}, false
}

// Resolve returns the stages a request runs, in execution order: the preset's
// stages (the default preset when none is named) plus any extra stages, minus
// those another selected stage replaces. Ids may use dashes or underscores.
func (s ExtentSpec) Resolve(preset string, match []string) ([]ExtentStage, error) {
	selected := map[string]bool{}
	base := normaliseID(preset)
	if base == "" {
		if len(match) > 0 {
			base = s.baseOnly()
		} else {
			base = s.DefaultPreset
		}
	}
	if base != "" {
		found := false
		for _, p := range s.Presets {
			if p.ID == base {
				found = true
				for _, id := range p.Stages {
					selected[id] = true
				}
			}
		}
		if !found && base != s.baseOnly() {
			return nil, fmt.Errorf("unknown swap extent %q; choose one of: %s", preset, strings.Join(s.presetIDs(), ", "))
		}
	}
	for _, raw := range match {
		id := normaliseID(raw)
		if _, ok := s.Stage(id); !ok {
			return nil, fmt.Errorf("unknown match option %q; choose from: %s", raw, strings.Join(s.stageIDs(), ", "))
		}
		selected[id] = true
	}
	var out []ExtentStage
	for _, st := range s.Stages {
		if !selected[st.ID] {
			continue
		}
		replaced := false
		for other := range selected {
			if o, _ := s.Stage(other); o.Replaces == st.ID {
				replaced = true
			}
		}
		if !replaced {
			out = append(out, st)
		}
	}
	swaps := 0
	for _, st := range out {
		if st.Kind == ExtentSwap {
			swaps++
		}
	}
	if swaps != 1 {
		return nil, errors.New("a swap extent needs exactly one swap stage (the face or the whole head)")
	}
	return out, nil
}

// ChooseVariants applies each stage's declared option to the request's params:
// the named variant (or the stage's default) overrides the stage's own fields. A
// stage without an option is returned unchanged.
func ChooseVariants(stages []ExtentStage, params map[string]string) ([]ExtentStage, error) {
	out := make([]ExtentStage, len(stages))
	for i, st := range stages {
		out[i] = st
		if st.Option == "" {
			continue
		}
		name := params[st.Option]
		if name == "" {
			name = st.DefaultVariant
		}
		name = normaliseID(name)
		v, ok := st.Variants[name]
		if !ok {
			names := make([]string, 0, len(st.Variants))
			for n := range st.Variants {
				names = append(names, n)
			}
			slices.Sort(names)
			return nil, fmt.Errorf("unknown %s %q; choose one of: %s", st.Option, params[st.Option], strings.Join(names, ", "))
		}
		out[i].Region = firstNonEmpty(v.Region, st.Region)
		out[i].ReferenceRegion = firstNonEmpty(v.ReferenceRegion, st.ReferenceRegion)
		out[i].ReferenceFrom = firstNonEmpty(v.ReferenceFrom, st.ReferenceFrom, ReferenceImage)
		out[i].DefaultVariant = name // the variant that ran
		if len(v.Transfer) > 0 {
			merged := make(map[string]float64, len(st.Transfer)+len(v.Transfer))
			for k, x := range st.Transfer {
				merged[k] = x
			}
			for k, x := range v.Transfer {
				merged[k] = x
			}
			out[i].Transfer = merged
		}
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// baseOnly is the id of the lowest preset: the plain swap that match options add
// to. It is the first preset by declaration.
func (s ExtentSpec) baseOnly() string {
	if len(s.Presets) == 0 {
		return ""
	}
	return s.Presets[0].ID
}

func (s ExtentSpec) stageIDs() []string {
	ids := make([]string, 0, len(s.Stages))
	for _, st := range s.Stages {
		ids = append(ids, st.ID)
	}
	return ids
}

func (s ExtentSpec) presetIDs() []string {
	ids := make([]string, 0, len(s.Presets))
	for _, p := range s.Presets {
		ids = append(ids, p.ID)
	}
	return ids
}

func normaliseID(id string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(id)), "-", "_")
}

// validateExtent checks a declaration is internally consistent and that the
// operation's parameter schema offers exactly the presets and stages it declares.
func validateExtent(opID string, schema json.RawMessage, e *ExtentSpec) error {
	if e == nil {
		return nil
	}
	ids := map[string]bool{}
	for _, st := range e.Stages {
		if st.ID == "" || ids[st.ID] {
			return fmt.Errorf("%s: extent stage id %q is empty or repeated", opID, st.ID)
		}
		ids[st.ID] = true
		if st.LabelKey == "" || st.DescKey == "" {
			return fmt.Errorf("%s: extent stage %q needs label_key and desc_key", opID, st.ID)
		}
		switch st.Kind {
		case ExtentSwap:
		case ExtentRecolour:
			if st.MaskModel == "" {
				return fmt.Errorf("%s: recolour stage %q needs a mask_model", opID, st.ID)
			}
			if st.Option == "" && (st.Region == "" || st.ReferenceRegion == "") {
				return fmt.Errorf("%s: recolour stage %q needs region and reference_region", opID, st.ID)
			}
		case ExtentRestyle:
			if st.Region == "" || st.MaskModel == "" || st.Model == "" || st.Prompt == "" {
				return fmt.Errorf("%s: restyle stage %q needs region, mask_model, model and prompt", opID, st.ID)
			}
		default:
			return fmt.Errorf("%s: extent stage %q has unknown kind %q", opID, st.ID, st.Kind)
		}
	}
	for _, st := range e.Stages {
		if err := validateOption(opID, st); err != nil {
			return err
		}
	}
	for _, st := range e.Stages {
		if st.Replaces != "" && !ids[st.Replaces] {
			return fmt.Errorf("%s: extent stage %q replaces unknown stage %q", opID, st.ID, st.Replaces)
		}
	}
	if len(e.Stages) == 0 || e.Stages[0].Kind != ExtentSwap {
		return fmt.Errorf("%s: the first extent stage must be a swap", opID)
	}
	presets := map[string]bool{}
	for _, p := range e.Presets {
		if p.ID == "" || presets[p.ID] {
			return fmt.Errorf("%s: extent preset id %q is empty or repeated", opID, p.ID)
		}
		presets[p.ID] = true
		if p.LabelKey == "" || p.DescKey == "" {
			return fmt.Errorf("%s: extent preset %q needs label_key and desc_key", opID, p.ID)
		}
		for _, id := range p.Stages {
			if !ids[id] {
				return fmt.Errorf("%s: extent preset %q names unknown stage %q", opID, p.ID, id)
			}
		}
		if _, err := e.Resolve(p.ID, nil); err != nil {
			return fmt.Errorf("%s: extent preset %q: %w", opID, p.ID, err)
		}
	}
	if !presets[e.DefaultPreset] {
		return fmt.Errorf("%s: default_preset %q is not a declared preset", opID, e.DefaultPreset)
	}
	var doc struct {
		Properties map[string]struct {
			Enum  []string `json:"enum"`
			Items struct {
				Enum []string `json:"enum"`
			} `json:"items"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return fmt.Errorf("%s: params_schema: %w", opID, err)
	}
	offered := map[string][]string{"extent": doc.Properties["extent"].Enum, "match": doc.Properties["match"].Items.Enum}
	for param, want := range map[string][]string{"extent": e.presetIDs(), "match": e.stageIDs()} {
		got := slices.Clone(offered[param])
		want = slices.Clone(want)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			return fmt.Errorf("%s: params_schema %q offers %v but the extent declares %v", opID, param, got, want)
		}
	}
	for _, st := range e.Stages {
		if st.Option == "" {
			continue
		}
		property, ok := doc.Properties[st.Option]
		if !ok {
			return fmt.Errorf("%s: stage %q option %q is not in params_schema", opID, st.ID, st.Option)
		}
		want := make([]string, 0, len(st.Variants))
		for name := range st.Variants {
			want = append(want, name)
		}
		got := slices.Clone(property.Enum)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			return fmt.Errorf("%s: params_schema %q offers %v but stage %q declares %v", opID, st.Option, got, st.ID, want)
		}
	}
	for _, st := range e.Stages {
		if st.StrengthParam != "" {
			if _, ok := doc.Properties[st.StrengthParam]; !ok {
				return fmt.Errorf("%s: stage %q strength_param %q is not in params_schema", opID, st.ID, st.StrengthParam)
			}
		}
	}
	return nil
}

// validateOption checks a stage's option and variants are complete: every variant
// is labelled and, for a recolour stage, names a full region and reference.
func validateOption(opID string, st ExtentStage) error {
	if st.Option == "" {
		if len(st.Variants) > 0 || st.DefaultVariant != "" {
			return fmt.Errorf("%s: stage %q declares variants without an option", opID, st.ID)
		}
		return nil
	}
	if _, ok := st.Variants[st.DefaultVariant]; !ok {
		return fmt.Errorf("%s: stage %q default_variant %q is not a declared variant", opID, st.ID, st.DefaultVariant)
	}
	for name, v := range st.Variants {
		if name != normaliseID(name) {
			return fmt.Errorf("%s: stage %q variant %q must be lower-case with underscores", opID, st.ID, name)
		}
		if v.LabelKey == "" || v.DescKey == "" {
			return fmt.Errorf("%s: stage %q variant %q needs label_key and desc_key", opID, st.ID, name)
		}
		if v.ReferenceFrom != "" && v.ReferenceFrom != ReferenceImage && v.ReferenceFrom != ReferenceWorking {
			return fmt.Errorf("%s: stage %q variant %q has unknown reference_from %q", opID, st.ID, name, v.ReferenceFrom)
		}
		incomplete := firstNonEmpty(v.Region, st.Region) == "" || firstNonEmpty(v.ReferenceRegion, st.ReferenceRegion) == ""
		if st.Kind == ExtentRecolour && incomplete {
			return fmt.Errorf("%s: recolour stage %q variant %q needs region and reference_region", opID, st.ID, name)
		}
	}
	return nil
}
