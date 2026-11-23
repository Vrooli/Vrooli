// AUTO-GENERATED — do not edit by hand.
//
// Source : api/internal/capability/manifests.seed.json (params_schema)
// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)
// Drift  : api/internal/capability/render_paramtypes_test.go fails the build if this file is stale.
//
// One typed view per operation that declares a params_schema. Persisted job
// payloads keep the string map; these decoders are the single place that turns
// it back into typed values, so an argument builder cannot misspell a key or
// disagree about a type.

package opparams

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// float64Bits is the bit size the decoders parse a JSON number with.
const float64Bits = 64

// InpaintParams is the typed view of this operation's parameter object.
type InpaintParams struct {
	ContextPixels          *int  `json:"inpaint_context_pixels,omitempty"`
	CropToMask             *bool `json:"inpaint_crop_to_mask,omitempty"`
	FeatherPx              *int  `json:"inpaint_feather_px,omitempty"`
	GrowPx                 *int  `json:"inpaint_grow_px,omitempty"`
	MaxEdgePx              *int  `json:"inpaint_max_edge_px,omitempty"`
	NeutralizeMaskedPixels *bool `json:"inpaint_neutralize_masked_pixels,omitempty"`
}

// DecodeInpaintParams turns the persisted string map into a typed view of the "inpaint"
// parameters. It fails closed on a value that no longer parses.
func DecodeInpaintParams(m map[string]string) (InpaintParams, error) {
	var out InpaintParams
	if raw, ok := m["inpaint_context_pixels"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("inpaint: inpaint_context_pixels: %w", err)
		}
		out.ContextPixels = &v
	}
	if raw, ok := m["inpaint_crop_to_mask"]; ok && raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return out, fmt.Errorf("inpaint: inpaint_crop_to_mask: %w", err)
		}
		out.CropToMask = &v
	}
	if raw, ok := m["inpaint_feather_px"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("inpaint: inpaint_feather_px: %w", err)
		}
		out.FeatherPx = &v
	}
	if raw, ok := m["inpaint_grow_px"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("inpaint: inpaint_grow_px: %w", err)
		}
		out.GrowPx = &v
	}
	if raw, ok := m["inpaint_max_edge_px"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("inpaint: inpaint_max_edge_px: %w", err)
		}
		out.MaxEdgePx = &v
	}
	if raw, ok := m["inpaint_neutralize_masked_pixels"]; ok && raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return out, fmt.Errorf("inpaint: inpaint_neutralize_masked_pixels: %w", err)
		}
		out.NeutralizeMaskedPixels = &v
	}
	return out, nil
}

// ObjectRemovalParamsBackgroundRepairRemovalRegionsItem is the typed view of this operation's parameter object.
type ObjectRemovalParamsBackgroundRepairRemovalRegionsItem struct {
	Height float64 `json:"height"`
	Width  float64 `json:"width"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// ObjectRemovalParamsBackgroundRepair is the typed view of this operation's parameter object.
type ObjectRemovalParamsBackgroundRepair struct {
	MaskExpansionPixels *int                                                    `json:"maskExpansionPixels,omitempty"`
	RemovalRegions      []ObjectRemovalParamsBackgroundRepairRemovalRegionsItem `json:"removalRegions,omitempty"`
}

// ObjectRemovalParams is the typed view of this operation's parameter object.
type ObjectRemovalParams struct {
	BackgroundRepair *ObjectRemovalParamsBackgroundRepair `json:"background_repair,omitempty"`
	GrowPx           *int                                 `json:"grow_px,omitempty"`
	HaloThreshold    *int                                 `json:"halo_threshold,omitempty"`
}

// DecodeObjectRemovalParams turns the persisted string map into a typed view of the "object_removal"
// parameters. It fails closed on a value that no longer parses.
func DecodeObjectRemovalParams(m map[string]string) (ObjectRemovalParams, error) {
	var out ObjectRemovalParams
	if raw, ok := m["background_repair"]; ok && raw != "" {
		var v ObjectRemovalParamsBackgroundRepair
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return out, fmt.Errorf("object_removal: background_repair: %w", err)
		}
		out.BackgroundRepair = &v
	}
	if raw, ok := m["grow_px"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("object_removal: grow_px: %w", err)
		}
		out.GrowPx = &v
	}
	if raw, ok := m["halo_threshold"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("object_removal: halo_threshold: %w", err)
		}
		out.HaloThreshold = &v
	}
	return out, nil
}

// FaceSwapParams is the typed view of this operation's parameter object.
type FaceSwapParams struct {
	AllFaces     *bool    `json:"all_faces,omitempty"`
	ColorMatch   *float64 `json:"color_match,omitempty"`
	Extent       *string  `json:"extent,omitempty"`
	FaceStrength *float64 `json:"face_strength,omitempty"`
	Match        []string `json:"match,omitempty"`
	Occlusion    *bool    `json:"occlusion,omitempty"`
	Prompt       *string  `json:"prompt,omitempty"`
	Seed         *int     `json:"seed,omitempty"`
	SkinTarget   *string  `json:"skin_target,omitempty"`
	SourceFace   *int     `json:"source_face,omitempty"`
	TargetFace   *int     `json:"target_face,omitempty"`
}

// DecodeFaceSwapParams turns the persisted string map into a typed view of the "face_swap"
// parameters. It fails closed on a value that no longer parses.
func DecodeFaceSwapParams(m map[string]string) (FaceSwapParams, error) {
	var out FaceSwapParams
	if raw, ok := m["all_faces"]; ok && raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return out, fmt.Errorf("face_swap: all_faces: %w", err)
		}
		out.AllFaces = &v
	}
	if raw, ok := m["color_match"]; ok && raw != "" {
		v, err := strconv.ParseFloat(raw, float64Bits)
		if err != nil {
			return out, fmt.Errorf("face_swap: color_match: %w", err)
		}
		out.ColorMatch = &v
	}
	if raw, ok := m["extent"]; ok && raw != "" {
		v := raw
		out.Extent = &v
	}
	if raw, ok := m["face_strength"]; ok && raw != "" {
		v, err := strconv.ParseFloat(raw, float64Bits)
		if err != nil {
			return out, fmt.Errorf("face_swap: face_strength: %w", err)
		}
		out.FaceStrength = &v
	}
	if raw, ok := m["match"]; ok && raw != "" {
		var v []string
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return out, fmt.Errorf("face_swap: match: %w", err)
		}
		out.Match = v
	}
	if raw, ok := m["occlusion"]; ok && raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return out, fmt.Errorf("face_swap: occlusion: %w", err)
		}
		out.Occlusion = &v
	}
	if raw, ok := m["prompt"]; ok && raw != "" {
		v := raw
		out.Prompt = &v
	}
	if raw, ok := m["seed"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("face_swap: seed: %w", err)
		}
		out.Seed = &v
	}
	if raw, ok := m["skin_target"]; ok && raw != "" {
		v := raw
		out.SkinTarget = &v
	}
	if raw, ok := m["source_face"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("face_swap: source_face: %w", err)
		}
		out.SourceFace = &v
	}
	if raw, ok := m["target_face"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("face_swap: target_face: %w", err)
		}
		out.TargetFace = &v
	}
	return out, nil
}

// GarmentReplaceParams is the typed view of this operation's parameter object.
type GarmentReplaceParams struct {
	InpaintFeatherPx *int `json:"inpaint_feather_px,omitempty"`
	InpaintGrowPx    *int `json:"inpaint_grow_px,omitempty"`
}

// DecodeGarmentReplaceParams turns the persisted string map into a typed view of the "garment_replace"
// parameters. It fails closed on a value that no longer parses.
func DecodeGarmentReplaceParams(m map[string]string) (GarmentReplaceParams, error) {
	var out GarmentReplaceParams
	if raw, ok := m["inpaint_feather_px"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("garment_replace: inpaint_feather_px: %w", err)
		}
		out.InpaintFeatherPx = &v
	}
	if raw, ok := m["inpaint_grow_px"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("garment_replace: inpaint_grow_px: %w", err)
		}
		out.InpaintGrowPx = &v
	}
	return out, nil
}

// ColorTransferParams is the typed view of this operation's parameter object.
type ColorTransferParams struct {
	FeatherPx *float64 `json:"feather_px,omitempty"`
	Flatten   *float64 `json:"flatten,omitempty"`
	Luminance *float64 `json:"luminance,omitempty"`
	Strength  *float64 `json:"strength,omitempty"`
}

// DecodeColorTransferParams turns the persisted string map into a typed view of the "color_transfer"
// parameters. It fails closed on a value that no longer parses.
func DecodeColorTransferParams(m map[string]string) (ColorTransferParams, error) {
	var out ColorTransferParams
	if raw, ok := m["feather_px"]; ok && raw != "" {
		v, err := strconv.ParseFloat(raw, float64Bits)
		if err != nil {
			return out, fmt.Errorf("color_transfer: feather_px: %w", err)
		}
		out.FeatherPx = &v
	}
	if raw, ok := m["flatten"]; ok && raw != "" {
		v, err := strconv.ParseFloat(raw, float64Bits)
		if err != nil {
			return out, fmt.Errorf("color_transfer: flatten: %w", err)
		}
		out.Flatten = &v
	}
	if raw, ok := m["luminance"]; ok && raw != "" {
		v, err := strconv.ParseFloat(raw, float64Bits)
		if err != nil {
			return out, fmt.Errorf("color_transfer: luminance: %w", err)
		}
		out.Luminance = &v
	}
	if raw, ok := m["strength"]; ok && raw != "" {
		v, err := strconv.ParseFloat(raw, float64Bits)
		if err != nil {
			return out, fmt.Errorf("color_transfer: strength: %w", err)
		}
		out.Strength = &v
	}
	return out, nil
}

// MaskCompositeParams is the typed view of this operation's parameter object.
type MaskCompositeParams struct {
	FeatherPx *float64 `json:"feather_px,omitempty"`
	GrowPx    *float64 `json:"grow_px,omitempty"`
}

// DecodeMaskCompositeParams turns the persisted string map into a typed view of the "mask_composite"
// parameters. It fails closed on a value that no longer parses.
func DecodeMaskCompositeParams(m map[string]string) (MaskCompositeParams, error) {
	var out MaskCompositeParams
	if raw, ok := m["feather_px"]; ok && raw != "" {
		v, err := strconv.ParseFloat(raw, float64Bits)
		if err != nil {
			return out, fmt.Errorf("mask_composite: feather_px: %w", err)
		}
		out.FeatherPx = &v
	}
	if raw, ok := m["grow_px"]; ok && raw != "" {
		v, err := strconv.ParseFloat(raw, float64Bits)
		if err != nil {
			return out, fmt.Errorf("mask_composite: grow_px: %w", err)
		}
		out.GrowPx = &v
	}
	return out, nil
}

// SegmentParamsSegmentationBox is the typed view of this operation's parameter object.
type SegmentParamsSegmentationBox struct {
	Height float64 `json:"height"`
	Width  float64 `json:"width"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// SegmentParamsSegmentationPointsItem is the typed view of this operation's parameter object.
type SegmentParamsSegmentationPointsItem struct {
	Negative *bool   `json:"negative,omitempty"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

// SegmentParamsSegmentation is the typed view of this operation's parameter object.
type SegmentParamsSegmentation struct {
	Box    *SegmentParamsSegmentationBox         `json:"box,omitempty"`
	Points []SegmentParamsSegmentationPointsItem `json:"points,omitempty"`
	Text   *string                               `json:"text,omitempty"`
}

// SegmentParams is the typed view of this operation's parameter object.
type SegmentParams struct {
	Segmentation *SegmentParamsSegmentation `json:"segmentation,omitempty"`
}

// DecodeSegmentParams turns the persisted string map into a typed view of the "segment"
// parameters. It fails closed on a value that no longer parses.
func DecodeSegmentParams(m map[string]string) (SegmentParams, error) {
	var out SegmentParams
	if raw, ok := m["segmentation"]; ok && raw != "" {
		var v SegmentParamsSegmentation
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return out, fmt.Errorf("segment: segmentation: %w", err)
		}
		out.Segmentation = &v
	}
	return out, nil
}

// SceneGeometryParamsSupportRoi is the typed view of this operation's parameter object.
type SceneGeometryParamsSupportRoi struct {
	Height float64 `json:"height"`
	Width  float64 `json:"width"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// SceneGeometryParams is the typed view of this operation's parameter object.
type SceneGeometryParams struct {
	SupportRoi *SceneGeometryParamsSupportRoi `json:"support_roi,omitempty"`
}

// DecodeSceneGeometryParams turns the persisted string map into a typed view of the "scene_geometry"
// parameters. It fails closed on a value that no longer parses.
func DecodeSceneGeometryParams(m map[string]string) (SceneGeometryParams, error) {
	var out SceneGeometryParams
	if raw, ok := m["support_roi"]; ok && raw != "" {
		var v SceneGeometryParamsSupportRoi
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return out, fmt.Errorf("scene_geometry: support_roi: %w", err)
		}
		out.SupportRoi = &v
	}
	return out, nil
}

// ImageToObjectParams is the typed view of this operation's parameter object.
type ImageToObjectParams struct {
	MeshResolution *int `json:"mesh_resolution,omitempty"`
}

// DecodeImageToObjectParams turns the persisted string map into a typed view of the "image_to_object"
// parameters. It fails closed on a value that no longer parses.
func DecodeImageToObjectParams(m map[string]string) (ImageToObjectParams, error) {
	var out ImageToObjectParams
	if raw, ok := m["mesh_resolution"]; ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return out, fmt.Errorf("image_to_object: mesh_resolution: %w", err)
		}
		out.MeshResolution = &v
	}
	return out, nil
}
