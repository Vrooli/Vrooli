package capability

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func faceSwapExtent(t *testing.T) *ExtentSpec {
	t.Helper()
	m, ok := Get("face_swap")
	if !ok || m.Extent == nil {
		t.Fatal("face_swap declares no extent")
	}
	return m.Extent
}

func ids(stages []ExtentStage) []string {
	out := make([]string, 0, len(stages))
	for _, s := range stages {
		out = append(out, s.ID)
	}
	return out
}

func TestExtentPresetsResolveToTheirStagesInExecutionOrder(t *testing.T) {
	spec := faceSwapExtent(t)
	for preset, want := range map[string][]string{
		"face":       {"face"},
		"face_skin":  {"face", "skin"},
		"full_look":  {"face", "skin", "hair_colour"},
		"whole_head": {"head"},
		"":           {"face"}, // the default preset
	} {
		got, err := spec.Resolve(preset, nil)
		if err != nil || !reflect.DeepEqual(ids(got), want) {
			t.Errorf("preset %q -> %v, %v; want %v", preset, ids(got), err, want)
		}
	}
}

func TestExtentStagesComposeIndependently(t *testing.T) {
	spec := faceSwapExtent(t)
	got, err := spec.Resolve("", []string{"hair-colour"})
	if err != nil || !reflect.DeepEqual(ids(got), []string{"face", "hair_colour"}) {
		t.Fatalf("hair colour alone: %v, %v", ids(got), err)
	}
	got, err = spec.Resolve("face_skin", []string{"hair_colour", "skin"})
	if err != nil || !reflect.DeepEqual(ids(got), []string{"face", "skin", "hair_colour"}) {
		t.Fatalf("a repeated stage must not run twice: %v, %v", ids(got), err)
	}
}

func TestExtentWholeHeadReplacesTheInnerFace(t *testing.T) {
	spec := faceSwapExtent(t)
	got, err := spec.Resolve("full_look", []string{"head"})
	if err != nil || !reflect.DeepEqual(ids(got), []string{"skin", "hair_colour", "head"}) {
		t.Fatalf("head with face stages: %v, %v", ids(got), err)
	}
	if got[2].Model == "" || got[2].Prompt == "" {
		t.Fatalf("the whole-head stage must pin its editor and instruction: %+v", got[2])
	}
}

func TestExtentRefusesWhatItDoesNotDeclare(t *testing.T) {
	spec := faceSwapExtent(t)
	if _, err := spec.Resolve("everything", nil); err == nil || !strings.Contains(err.Error(), "whole_head") {
		t.Fatalf("unknown preset must list the known ones: %v", err)
	}
	if _, err := spec.Resolve("", []string{"freckles"}); err == nil || !strings.Contains(err.Error(), "skin") {
		t.Fatalf("unknown stage must list the known ones: %v", err)
	}
}

func TestExtentSchemaMustOfferExactlyTheDeclaredVocabulary(t *testing.T) {
	spec := *faceSwapExtent(t)
	schema := func(extent, match string) json.RawMessage {
		return json.RawMessage(`{"type":"object","properties":{"face_strength":{"type":"number"},` +
			`"skin_target":{"type":"string","enum":["source","body"]},` +
			`"extent":{"type":"string","enum":` + extent + `},"match":{"type":"array","items":{"type":"string","enum":` + match + `}}}}`)
	}
	good := schema(`["face","face_skin","full_look","whole_head"]`, `["face","skin","hair_style","hair_colour","head"]`)
	if err := validateExtent("t", good, &spec); err != nil {
		t.Fatal(err)
	}
	if err := validateExtent("t", schema(`["face"]`, `["face","skin","hair_style","hair_colour","head"]`), &spec); err == nil {
		t.Error("a schema offering fewer presets than declared must fail")
	}
	if err := validateExtent("t", schema(`["face","face_skin","full_look","whole_head"]`, `["face","skin"]`), &spec); err == nil {
		t.Error("a schema offering fewer stages than declared must fail")
	}
}

func TestExtentDeclarationsAreCheckedForConsistency(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"extent":{"type":"string","enum":["a"]},"match":{"type":"array","items":{"type":"string","enum":["a","b"]}}}}`)
	base := func() ExtentSpec {
		return ExtentSpec{
			DefaultPreset: "a",
			Stages: []ExtentStage{
				{ID: "a", Kind: ExtentSwap, LabelKey: "l", DescKey: "d"},
				{ID: "b", Kind: ExtentRecolour, LabelKey: "l", DescKey: "d", Region: "r", ReferenceRegion: "r", MaskModel: "m"},
			},
			Presets: []ExtentPreset{{ID: "a", LabelKey: "l", DescKey: "d", Stages: []string{"a"}}},
		}
	}
	ok := base()
	if err := validateExtent("t", schema, &ok); err != nil {
		t.Fatalf("a consistent declaration was refused: %v", err)
	}
	for name, mutate := range map[string]func(*ExtentSpec){
		"unknown kind":          func(s *ExtentSpec) { s.Stages[1].Kind = "paint" },
		"recolour without mask": func(s *ExtentSpec) { s.Stages[1].MaskModel = "" },
		"swap not first":        func(s *ExtentSpec) { s.Stages[0], s.Stages[1] = s.Stages[1], s.Stages[0] },
		"repeated stage":        func(s *ExtentSpec) { s.Stages[1].ID = "a" },
		"preset of unknown":     func(s *ExtentSpec) { s.Presets[0].Stages = []string{"zzz"} },
		"preset with no swap":   func(s *ExtentSpec) { s.Presets[0].Stages = []string{"b"} },
		"missing default":       func(s *ExtentSpec) { s.DefaultPreset = "nope" },
		"replaces unknown":      func(s *ExtentSpec) { s.Stages[1].Replaces = "zzz" },
		"missing label":         func(s *ExtentSpec) { s.Stages[0].LabelKey = "" },
	} {
		spec := base()
		mutate(&spec)
		if err := validateExtent("t", schema, &spec); err == nil {
			t.Errorf("%s: an inconsistent declaration was accepted", name)
		}
	}
}

func TestExtentHairStyleIsARestyleStageBeforeTheHairColour(t *testing.T) {
	spec := faceSwapExtent(t)
	got, err := spec.Resolve("full_look", []string{"hair_colour", "hair-style"})
	if err != nil || !reflect.DeepEqual(ids(got), []string{"face", "skin", "hair_style", "hair_colour"}) {
		t.Fatalf("stages = %v, %v", ids(got), err)
	}
	style := got[2]
	if style.Kind != ExtentRestyle || style.Model == "" || style.Prompt == "" || style.Region == "" || style.MaskModel == "" {
		t.Fatalf("the hair style stage must name its editor, instruction and hair region: %+v", style)
	}
}

func TestExtentRestyleStageNeedsItsEditorAndRegion(t *testing.T) {
	spec := *faceSwapExtent(t)
	spec.Stages = append([]ExtentStage(nil), spec.Stages...)
	for i, st := range spec.Stages {
		if st.Kind == ExtentRestyle {
			spec.Stages[i].Model = ""
		}
	}
	schema := json.RawMessage(`{"type":"object","properties":{"face_strength":{"type":"number"},"skin_target":{"enum":["source","body"]},"extent":{"enum":["face","face_skin","full_look","whole_head"]},"match":{"items":{"enum":["face","skin","hair_style","hair_colour","head"]}}}}`)
	if err := validateExtent("t", schema, &spec); err == nil || !strings.Contains(err.Error(), "restyle") {
		t.Fatalf("a restyle stage without a model must be refused: %v", err)
	}
}

func TestExtentSkinTargetVariantsOverrideTheStageAndDefaultToTheSource(t *testing.T) {
	spec := faceSwapExtent(t)
	stages, err := spec.Resolve("face_skin", nil)
	if err != nil {
		t.Fatal(err)
	}
	defaulted, err := ChooseVariants(stages, nil)
	if err != nil {
		t.Fatal(err)
	}
	skin := defaulted[1]
	if skin.DefaultVariant != "source" || skin.ReferenceFrom != ReferenceImage || skin.Region != "face arms" || skin.Transfer["feather_px"] < 20 {
		t.Fatalf("the default skin stage must take the reference person's skin with a wide feather: %+v", skin)
	}
	body, err := ChooseVariants(stages, map[string]string{"skin_target": "body"})
	if err != nil {
		t.Fatal(err)
	}
	skin = body[1]
	if skin.ReferenceFrom != ReferenceWorking || skin.Region != "face" || skin.ReferenceRegion != "arms" || skin.Transfer["luminance"] != 1 {
		t.Fatalf("body must match the face to the working image's own arms: %+v", skin)
	}
	if stages[1].Region != "" || stages[1].ReferenceFrom != "" {
		t.Fatal("choosing a variant must not edit the declared stage")
	}
	if _, err := ChooseVariants(stages, map[string]string{"skin_target": "everywhere"}); err == nil || !strings.Contains(err.Error(), "body") {
		t.Fatalf("an unknown variant must list the known ones: %v", err)
	}
	if plain, _ := spec.Resolve("face", nil); len(plain) != 1 {
		t.Fatal("a swap alone has no variants to choose")
	}
}

func TestExtentVariantsMustBeCompleteAndMatchTheSchema(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"face_strength":{"type":"number"},"skin_target":{"enum":["source","body"]},"extent":{"enum":["face","face_skin","full_look","whole_head"]},"match":{"items":{"enum":["face","skin","hair_style","hair_colour","head"]}}}}`)
	broken := func(edit func(*ExtentStage)) error {
		spec := *faceSwapExtent(t)
		spec.Stages = append([]ExtentStage(nil), spec.Stages...)
		for i, st := range spec.Stages {
			if st.ID == "skin" {
				variants := map[string]ExtentVariant{}
				for k, v := range st.Variants {
					variants[k] = v
				}
				st.Variants = variants
				edit(&st)
				spec.Stages[i] = st
			}
		}
		return validateExtent("t", schema, &spec)
	}
	if err := broken(func(*ExtentStage) {}); err != nil {
		t.Fatalf("the declared extent must validate: %v", err)
	}
	if err := broken(func(st *ExtentStage) { st.DefaultVariant = "everywhere" }); err == nil {
		t.Error("a default that is not a variant must be refused")
	}
	if err := broken(func(st *ExtentStage) { delete(st.Variants, "body") }); err == nil {
		t.Error("a schema offering a variant the stage does not declare must be refused")
	}
	if err := broken(func(st *ExtentStage) {
		v := st.Variants["body"]
		v.ReferenceRegion = ""
		st.Variants["body"] = v
	}); err == nil {
		t.Error("a recolour variant without a reference region must be refused")
	}
	if err := broken(func(st *ExtentStage) {
		v := st.Variants["body"]
		v.ReferenceFrom = "moon"
		st.Variants["body"] = v
	}); err == nil {
		t.Error("an unknown reference_from must be refused")
	}
}
