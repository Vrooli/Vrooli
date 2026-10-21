// Package scenes owns pure, renderer-independent scene candidate values.
// Passing validation proves local consistency only; it does not qualify assets.
package scenes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"device-studio/internal/library"
	"device-studio/internal/physical"
)

const (
	CandidateVersion = "scene-candidate/v1"
	MaxArtifacts     = 64
	MaxParts         = 256
	MaxMaterials     = 128
	MaxSlots         = 256
	MaxFrames        = 128
	MaxCanonicalJSON = 1 << 20
)

type SceneCandidate struct {
	Version              string                 `json:"version"`
	SceneID              string                 `json:"scene_id"`
	CoordinateConvention string                 `json:"coordinate_convention"`
	Artifacts            []SceneArtifactBinding `json:"artifacts"`
	Materials            []SceneMaterial        `json:"materials"`
	MaterialSlots        []SceneMaterialSlot    `json:"material_slots"`
	Parts                []ScenePart            `json:"parts"`
	SupportFrames        []SceneSupportFrame    `json:"support_frames"`
	CameraFrames         []SceneCameraFrame     `json:"camera_frames"`
	LightFrames          []SceneLightFrame      `json:"light_frames"`
	Environment          SceneEnvironment       `json:"environment"`
	Regions              []SceneRegion          `json:"regions"`
}

type SceneArtifactBinding struct {
	ID                string               `json:"id"`
	Ref               physical.ArtifactRef `json:"ref"`
	SourceUnit        string               `json:"source_unit,omitempty"`
	SourceToCanonical *physical.TRS        `json:"source_to_canonical,omitempty"`
}

type MaterialProfile string

// MaterialEmissionV1 uses fixed emission strength 1.0; changing it requires a
// new profile version rather than a renderer default.
const (
	MaterialPrincipledV1 MaterialProfile = "PRINCIPLED_V1"
	MaterialEmissionV1   MaterialProfile = "EMISSION_V1"
)

type AlphaMode string

const (
	AlphaOpaque AlphaMode = "OPAQUE"
	AlphaMask   AlphaMode = "MASK"
	AlphaBlend  AlphaMode = "BLEND"
)

type SceneMaterial struct {
	ID                  string               `json:"id"`
	SurfaceProfile      MaterialProfile      `json:"surface_profile"`
	BaseColorLinearRGBA [4]float64           `json:"base_color_linear_rgba"`
	Metallic            float64              `json:"metallic"`
	Roughness           float64              `json:"roughness"`
	AlphaMode           AlphaMode            `json:"alpha_mode"`
	AlphaCutoff         *float64             `json:"alpha_cutoff,omitempty"`
	Inputs              []SceneMaterialInput `json:"inputs,omitempty"`
	Grade               *SceneMaterialGrade  `json:"grade,omitempty"`
}

type TextureSocket string

const (
	SocketBaseColor TextureSocket = "base_color"
	SocketRoughness TextureSocket = "roughness"
	SocketNormal    TextureSocket = "normal"
	SocketBump      TextureSocket = "bump_height"
)

type ColorSpace string

const (
	ColorSpaceSRGB     ColorSpace = "SRGB"
	ColorSpaceNonColor ColorSpace = "NON_COLOR"
)

type TextureChannel string

const (
	ChannelRGB TextureChannel = "RGB"
	ChannelRed TextureChannel = "RED"
)

type TextureSampler string

const (
	SamplerRepeat TextureSampler = "REPEAT"
	SamplerClamp  TextureSampler = "CLAMP"
)

type CoordinateMapping string

const (
	CoordinatesGeneratedBounds CoordinateMapping = "GENERATED_BOUNDS_X_NEG_Z_V1"
	CoordinatesUVSet           CoordinateMapping = "UV_SET_AFFINE_V1"
)

// SceneMaterialInput v1 fixes image interpolation to linear for every input.
// Sampler records only wrap behavior; interpolation is not caller-selectable.
type SceneMaterialInput struct {
	Socket           TextureSocket     `json:"socket"`
	TextureBindingID string            `json:"texture_binding_id"`
	Semantic         string            `json:"semantic"`
	ColorSpace       ColorSpace        `json:"color_space"`
	Channel          TextureChannel    `json:"channel"`
	Sampler          TextureSampler    `json:"sampler"`
	Coordinates      CoordinateMapping `json:"coordinates"`
	UVSet            string            `json:"uv_set,omitempty"`
	AffineUV         *[6]float64       `json:"affine_uv,omitempty"`
	PhysicalRepeatM  *[2]float64       `json:"physical_repeat_m,omitempty"`
	NormalConvention string            `json:"normal_convention,omitempty"`
	NormalStrength   *float64          `json:"normal_strength,omitempty"`
	BumpStrength     *float64          `json:"bump_strength,omitempty"`
	BumpDistanceM    *float64          `json:"bump_distance_m,omitempty"`
}

type GradeRecipe string

const GradeHueSaturationLinearMixV1 GradeRecipe = "HUE_SATURATION_THEN_LINEAR_MIX_V1"

type SceneMaterialGrade struct {
	Recipe       GradeRecipe `json:"recipe"`
	Saturation   float64     `json:"saturation"`
	Value        float64     `json:"value"`
	MixFactor    float64     `json:"mix_factor"`
	MixLinearRGB [3]float64  `json:"mix_linear_rgb"`
}

type SceneMaterialSlot struct {
	PartID         string `json:"part_id"`
	PrimitiveIndex uint32 `json:"primitive_index"`
	SlotIndex      uint32 `json:"slot_index"`
	MaterialID     string `json:"material_id"`
}

type ScenePart struct {
	ID                string       `json:"id"`
	Role              string       `json:"role"`
	ArtifactBindingID string       `json:"artifact_binding_id"`
	ParentID          string       `json:"parent_id,omitempty"`
	NodeIndex         uint32       `json:"node_index"`
	NodeName          string       `json:"node_name"`
	LocalFrame        physical.TRS `json:"local_frame"`
}

type SceneSupportFrame struct {
	ID                  string         `json:"id"`
	InterfaceID         string         `json:"interface_id"`
	FramePartID         string         `json:"frame_part_id"`
	LocalFrame          physical.TRS   `json:"local_frame"`
	ContactRegionPartID string         `json:"contact_region_part_id,omitempty"`
	LoadDirectionLocal  *physical.Vec3 `json:"load_direction_local,omitempty"`
}

type Projection string

const (
	ProjectionPerspective  Projection = "PERSPECTIVE"
	ProjectionOrthographic Projection = "ORTHOGRAPHIC"
)

type SensorFit string

const (
	SensorFitHorizontal SensorFit = "HORIZONTAL"
	SensorFitVertical   SensorFit = "VERTICAL"
	SensorFitAuto       SensorFit = "AUTO"
)

type SceneCameraFrame struct {
	ID                string       `json:"id"`
	Frame             physical.TRS `json:"frame"`
	Projection        Projection   `json:"projection"`
	FocalLengthMM     *float64     `json:"focal_length_mm,omitempty"`
	SensorWidthMM     *float64     `json:"sensor_width_mm,omitempty"`
	SensorHeightMM    *float64     `json:"sensor_height_mm,omitempty"`
	SensorFit         SensorFit    `json:"sensor_fit,omitempty"`
	NearClipM         float64      `json:"near_clip_m"`
	FarClipM          float64      `json:"far_clip_m"`
	OrthographicSpanM *float64     `json:"orthographic_vertical_span_m,omitempty"`
	Focus             *FocusTarget `json:"focus,omitempty"`
	DOFFNumber        *float64     `json:"dof_f_number,omitempty"`
}

type FocusMode string

const (
	FocusScenePart      FocusMode = "SCENE_PART"
	FocusSelectedDevice FocusMode = "SELECTED_DEVICE_ROOT"
	FocusDistance       FocusMode = "DISTANCE"
)

type FocusTarget struct {
	Mode        FocusMode `json:"mode"`
	ScenePartID string    `json:"scene_part_id,omitempty"`
	DistanceM   *float64  `json:"distance_m,omitempty"`
}

type LightShape string

const (
	LightRectangle LightShape = "RECTANGLE"
	LightDisk      LightShape = "DISK"
)

type SceneLightFrame struct {
	ID               string       `json:"id"`
	Frame            physical.TRS `json:"frame"`
	LocalAxisProfile string       `json:"local_axis_profile"`
	Shape            LightShape   `json:"shape"`
	DimensionsM      [2]float64   `json:"dimensions_m"`
	PowerWatts       float64      `json:"power_watts"`
	LinearRGB        [3]float64   `json:"linear_rgb"`
}

type SceneEnvironment struct {
	Model     string     `json:"model"`
	LinearRGB [3]float64 `json:"linear_rgb"`
	Strength  float64    `json:"strength"`
}

type RegionPurpose string

const (
	RegionContact   RegionPurpose = "CONTACT"
	RegionClearance RegionPurpose = "CLEARANCE"
	RegionExclusion RegionPurpose = "EXCLUSION"
)

type SceneRegion struct {
	ID      string        `json:"id"`
	PartID  string        `json:"part_id"`
	Purpose RegionPurpose `json:"purpose"`
}

// CanonicalizeCandidate validates local references, canonicalizes value fields,
// sorts collections by stable IDs and returns the bounded canonical JSON/hash.
func CanonicalizeCandidate(input SceneCandidate) (SceneCandidate, []byte, string, error) {
	if err := preflightCandidate(input); err != nil {
		return SceneCandidate{}, nil, "", err
	}
	candidate := cloneCandidate(input)
	if err := validateAndCanonicalize(&candidate); err != nil {
		return SceneCandidate{}, nil, "", err
	}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		return SceneCandidate{}, nil, "", fmt.Errorf("encode scene candidate: %w", err)
	}
	if len(encoded) > MaxCanonicalJSON {
		return SceneCandidate{}, nil, "", fmt.Errorf("scene candidate exceeds %d canonical JSON bytes", MaxCanonicalJSON)
	}
	digest := sha256.Sum256(encoded)
	return candidate, encoded, hex.EncodeToString(digest[:]), nil
}

func preflightCandidate(input SceneCandidate) error {
	if len(input.Artifacts) > MaxArtifacts || len(input.Parts) > MaxParts || len(input.Materials) > MaxMaterials || len(input.MaterialSlots) > MaxSlots || len(input.SupportFrames) > MaxFrames || len(input.CameraFrames) > MaxFrames || len(input.LightFrames) > MaxFrames || len(input.Regions) > MaxFrames {
		return errors.New("scene candidate collection exceeds preflight bounds")
	}
	for i := range input.Materials {
		if len(input.Materials[i].Inputs) > 4 {
			return fmt.Errorf("scene material %q exceeds the four supported texture sockets", input.Materials[i].ID)
		}
	}
	return nil
}

func validateAndCanonicalize(c *SceneCandidate) error {
	if c.Version != CandidateVersion || !library.IsValidStableID(c.SceneID) || c.CoordinateConvention != physical.CoordinateConvention {
		return errors.New("scene candidate identity or coordinate convention is invalid")
	}
	if len(c.Artifacts) == 0 || len(c.Artifacts) > MaxArtifacts || len(c.Parts) == 0 || len(c.Parts) > MaxParts || len(c.Materials) > MaxMaterials || len(c.MaterialSlots) > MaxSlots {
		return errors.New("scene candidate artifact, part, material, or slot count is outside bounds")
	}
	if len(c.SupportFrames) > MaxFrames || len(c.CameraFrames) > MaxFrames || len(c.LightFrames) > MaxFrames || len(c.Regions) > MaxFrames {
		return errors.New("scene candidate frame or region count exceeds bounds")
	}
	artifacts := make(map[string]SceneArtifactBinding, len(c.Artifacts))
	for i := range c.Artifacts {
		a := &c.Artifacts[i]
		if !library.IsValidStableID(a.ID) || artifacts[a.ID].ID != "" {
			return fmt.Errorf("artifact binding %q is invalid or duplicated", a.ID)
		}
		if err := validateArtifactRef(a.Ref); err != nil {
			return fmt.Errorf("artifact binding %q: %w", a.ID, err)
		}
		switch a.Ref.Role {
		case library.RoleGeometryGLB:
			if a.SourceUnit != "m" && a.SourceUnit != "cm" && a.SourceUnit != "mm" || a.SourceToCanonical == nil {
				return fmt.Errorf("geometry binding %q requires source unit and rigid correction", a.ID)
			}
			frame, err := physical.CanonicalizeTRS(*a.SourceToCanonical)
			if err != nil || frame.Scale != (physical.Vec3{X: 1, Y: 1, Z: 1}) {
				return fmt.Errorf("geometry binding %q correction must be finite and rigid", a.ID)
			}
			a.SourceToCanonical = &frame
		case library.RoleTextureImage:
			if a.SourceUnit != "" || a.SourceToCanonical != nil {
				return fmt.Errorf("texture binding %q cannot declare geometry conversion", a.ID)
			}
		default:
			return fmt.Errorf("artifact binding %q has unsupported role", a.ID)
		}
		artifacts[a.ID] = *a
	}
	parts := make(map[string]ScenePart, len(c.Parts))
	nodes := make(map[string]string, len(c.Parts))
	for i := range c.Parts {
		p := &c.Parts[i]
		if !library.IsValidStableID(p.ID) || !library.IsValidStableID(p.Role) || p.NodeName == "" || len(p.NodeName) > 256 || strings.TrimSpace(p.NodeName) != p.NodeName || parts[p.ID].ID != "" {
			return fmt.Errorf("scene part %q has invalid or duplicate identity", p.ID)
		}
		a, ok := artifacts[p.ArtifactBindingID]
		if !ok || a.Ref.Role != library.RoleGeometryGLB {
			return fmt.Errorf("scene part %q must bind a local geometry artifact", p.ID)
		}
		key := fmt.Sprintf("%s/%d", p.ArtifactBindingID, p.NodeIndex)
		if prior := nodes[key]; prior != "" {
			return fmt.Errorf("scene parts %q and %q bind the same local artifact node", prior, p.ID)
		}
		nodes[key] = p.ID
		frame, err := physical.CanonicalizeTRS(p.LocalFrame)
		if err != nil {
			return fmt.Errorf("scene part %q local frame: %w", p.ID, err)
		}
		p.LocalFrame = frame
		parts[p.ID] = *p
	}
	for _, p := range c.Parts {
		if p.ParentID != "" && (p.ParentID == p.ID || parts[p.ParentID].ID == "") {
			return fmt.Errorf("scene part %q has invalid parent %q", p.ID, p.ParentID)
		}
	}
	if err := validatePartGraph(parts); err != nil {
		return err
	}
	materials := make(map[string]SceneMaterial, len(c.Materials))
	for i := range c.Materials {
		m := &c.Materials[i]
		if !library.IsValidStableID(m.ID) || materials[m.ID].ID != "" {
			return fmt.Errorf("scene material %q is invalid or duplicated", m.ID)
		}
		if err := validateMaterial(m, artifacts); err != nil {
			return fmt.Errorf("scene material %q: %w", m.ID, err)
		}
		sort.Slice(m.Inputs, func(i, j int) bool { return m.Inputs[i].Socket < m.Inputs[j].Socket })
		materials[m.ID] = *m
	}
	if err := validateSlots(c.MaterialSlots, parts, materials); err != nil {
		return err
	}
	if err := validateSupportFrames(c.SupportFrames, parts); err != nil {
		return err
	}
	if err := validateCameras(c.CameraFrames, parts); err != nil {
		return err
	}
	if err := validateLights(c.LightFrames); err != nil {
		return err
	}
	if c.Environment.Model != "CONSTANT_V1" || !unitRGB(c.Environment.LinearRGB[:]) || !finiteNonnegative(c.Environment.Strength) {
		return errors.New("scene environment must be finite CONSTANT_V1 linear RGB with nonnegative strength")
	}
	seenRegions := make(map[string]bool, len(c.Regions))
	for _, region := range c.Regions {
		if !library.IsValidStableID(region.ID) || seenRegions[region.ID] || parts[region.PartID].ID == "" || (region.Purpose != RegionContact && region.Purpose != RegionClearance && region.Purpose != RegionExclusion) {
			return fmt.Errorf("scene region %q has invalid identity, part, or purpose", region.ID)
		}
		seenRegions[region.ID] = true
	}
	sort.Slice(c.Artifacts, func(i, j int) bool { return c.Artifacts[i].ID < c.Artifacts[j].ID })
	sort.Slice(c.Parts, func(i, j int) bool { return c.Parts[i].ID < c.Parts[j].ID })
	sort.Slice(c.Materials, func(i, j int) bool { return c.Materials[i].ID < c.Materials[j].ID })
	sort.Slice(c.MaterialSlots, func(i, j int) bool { return slotKey(c.MaterialSlots[i]) < slotKey(c.MaterialSlots[j]) })
	sort.Slice(c.SupportFrames, func(i, j int) bool { return c.SupportFrames[i].ID < c.SupportFrames[j].ID })
	sort.Slice(c.CameraFrames, func(i, j int) bool { return c.CameraFrames[i].ID < c.CameraFrames[j].ID })
	sort.Slice(c.LightFrames, func(i, j int) bool { return c.LightFrames[i].ID < c.LightFrames[j].ID })
	sort.Slice(c.Regions, func(i, j int) bool { return c.Regions[i].ID < c.Regions[j].ID })
	return nil
}

func validateArtifactRef(ref physical.ArtifactRef) error {
	if !library.IsValidStableID(ref.BundleID) || ref.BundleRevision == 0 || !library.IsValidStableID(ref.ArtifactID) || ref.SizeBytes <= 0 || !validDigest(ref.ManifestSHA256) || !validDigest(ref.ReviewSHA256) || !validDigest(ref.SHA256) {
		return errors.New("pinned bundle, review, artifact, size, or digest facts are incomplete")
	}
	switch ref.Role {
	case library.RoleGeometryGLB:
		if ref.SizeBytes > library.MaxGeometryBytes {
			return errors.New("geometry artifact exceeds the library role size limit")
		}
	case library.RoleTextureImage:
		if ref.SizeBytes > library.MaxTextureBytes {
			return errors.New("texture artifact exceeds the library role size limit")
		}
	default:
		return errors.New("artifact role is not supported by scene candidates")
	}
	return nil
}

func validatePartGraph(parts map[string]ScenePart) error {
	state := make(map[string]uint8, len(parts))
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("scene part parent graph contains a cycle at %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		if parent := parts[id].ParentID; parent != "" {
			if err := visit(parent); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range parts {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func validateMaterial(m *SceneMaterial, artifacts map[string]SceneArtifactBinding) error {
	if m.SurfaceProfile != MaterialPrincipledV1 && m.SurfaceProfile != MaterialEmissionV1 || !unitRGB(m.BaseColorLinearRGBA[:3]) || !unit(m.BaseColorLinearRGBA[3]) || !unit(m.Metallic) || !unit(m.Roughness) || (m.AlphaMode != AlphaOpaque && m.AlphaMode != AlphaMask && m.AlphaMode != AlphaBlend) {
		return errors.New("surface profile or material factors are unsupported")
	}
	if m.AlphaMode == AlphaMask {
		if m.AlphaCutoff == nil || !unit(*m.AlphaCutoff) {
			return errors.New("MASK alpha mode requires an explicit cutoff in [0,1]")
		}
	} else if m.AlphaCutoff != nil {
		return errors.New("alpha cutoff is valid only with MASK alpha mode")
	}
	if m.SurfaceProfile == MaterialEmissionV1 && (m.Grade != nil || m.Metallic != 0 || m.Roughness != 1) {
		return errors.New("emission material requires fixed non-principled metallic and roughness values")
	}
	seen := make(map[TextureSocket]bool, len(m.Inputs))
	for i := range m.Inputs {
		in := &m.Inputs[i]
		if m.SurfaceProfile == MaterialEmissionV1 && in.Socket != SocketBaseColor {
			return errors.New("emission material supports only a base-color image input")
		}
		artifact, ok := artifacts[in.TextureBindingID]
		if !ok || artifact.Ref.Role != library.RoleTextureImage || seen[in.Socket] {
			return fmt.Errorf("texture socket %q has a missing, non-texture, or duplicate binding", in.Socket)
		}
		seen[in.Socket] = true
		if err := validateTextureInput(*in); err != nil {
			return err
		}
	}
	if m.Grade != nil {
		g := m.Grade
		if m.SurfaceProfile != MaterialPrincipledV1 || !seen[SocketBaseColor] || g.Recipe != GradeHueSaturationLinearMixV1 || !finite(g.Saturation) || g.Saturation < 0 || g.Saturation > 2 || !finite(g.Value) || g.Value < 0 || g.Value > 2 || !unit(g.MixFactor) || !unitRGB(g.MixLinearRGB[:]) {
			return errors.New("grade recipe or its base-color input is invalid")
		}
	}
	return nil
}

func validateTextureInput(in SceneMaterialInput) error {
	var wantSpace ColorSpace
	var wantChannel TextureChannel
	switch in.Socket {
	case SocketBaseColor:
		wantSpace, wantChannel = ColorSpaceSRGB, ChannelRGB
		if in.Semantic != "base_color" || in.NormalConvention != "" || in.NormalStrength != nil || in.BumpStrength != nil || in.BumpDistanceM != nil {
			return errors.New("base_color socket requires base_color semantic")
		}
	case SocketRoughness:
		wantSpace, wantChannel = ColorSpaceNonColor, ChannelRed
		if in.Semantic != "roughness" || in.NormalConvention != "" || in.NormalStrength != nil || in.BumpStrength != nil || in.BumpDistanceM != nil {
			return errors.New("roughness socket requires roughness semantic")
		}
	case SocketNormal:
		wantSpace, wantChannel = ColorSpaceNonColor, ChannelRGB
		if in.Semantic != "normal_gl" || in.NormalConvention != "OPENGL_TANGENT_PLUS_Y_V1" || in.NormalStrength == nil || !finite(*in.NormalStrength) || *in.NormalStrength < 0 || *in.NormalStrength > 1 || in.BumpStrength != nil || in.BumpDistanceM != nil {
			return errors.New("normal socket requires the supported tangent +Y convention and strength")
		}
	case SocketBump:
		wantSpace, wantChannel = ColorSpaceNonColor, ChannelRed
		if in.Semantic != "bump_height" || in.BumpStrength == nil || !finite(*in.BumpStrength) || *in.BumpStrength < 0 || *in.BumpStrength > 1 || in.BumpDistanceM == nil || !finite(*in.BumpDistanceM) || *in.BumpDistanceM <= 0 || in.NormalConvention != "" || in.NormalStrength != nil {
			return errors.New("bump socket requires bounded strength and positive distance")
		}
	default:
		return fmt.Errorf("texture socket %q is unsupported", in.Socket)
	}
	if (in.Socket != SocketNormal && in.NormalConvention != "") || (in.Socket != SocketNormal && in.NormalStrength != nil) || (in.Socket != SocketBump && (in.BumpStrength != nil || in.BumpDistanceM != nil)) {
		return errors.New("texture input contains fields outside its socket variant")
	}
	if in.ColorSpace != wantSpace || in.Channel != wantChannel || (in.Sampler != SamplerRepeat && in.Sampler != SamplerClamp) {
		return errors.New("texture semantic, color-space, channel, or sampler does not match the socket")
	}
	switch in.Coordinates {
	case CoordinatesGeneratedBounds:
		if in.Sampler != SamplerRepeat || in.UVSet != "" || in.AffineUV != nil || in.PhysicalRepeatM == nil || !positive(in.PhysicalRepeatM[0]) || !positive(in.PhysicalRepeatM[1]) {
			return errors.New("generated-bounds mapping requires only positive physical repeat dimensions")
		}
	case CoordinatesUVSet:
		if !library.IsValidStableID(in.UVSet) || in.AffineUV == nil || in.PhysicalRepeatM != nil {
			return errors.New("UV-set mapping requires one named set and one affine transform")
		}
		for _, value := range in.AffineUV {
			if !finite(value) {
				return errors.New("UV affine transform must be finite")
			}
		}
		determinant := (*in.AffineUV)[0]*(*in.AffineUV)[3] - (*in.AffineUV)[1]*(*in.AffineUV)[2]
		if !finite(determinant) || determinant == 0 {
			return errors.New("UV affine transform must be invertible")
		}
	default:
		return errors.New("texture coordinate mapping is unsupported")
	}
	return nil
}

func validateSlots(slots []SceneMaterialSlot, parts map[string]ScenePart, materials map[string]SceneMaterial) error {
	seen := make(map[string]bool, len(slots))
	for _, slot := range slots {
		key := slotKey(slot)
		if parts[slot.PartID].ID == "" || materials[slot.MaterialID].ID == "" || seen[key] {
			return fmt.Errorf("material slot %q is duplicated or has an unknown part/material", key)
		}
		seen[key] = true
	}
	return nil
}

func validateSupportFrames(frames []SceneSupportFrame, parts map[string]ScenePart) error {
	seen := make(map[string]bool, len(frames))
	for i := range frames {
		f := &frames[i]
		if !library.IsValidStableID(f.ID) || !library.IsValidStableID(f.InterfaceID) || seen[f.ID] || parts[f.FramePartID].ID == "" || (f.ContactRegionPartID != "" && parts[f.ContactRegionPartID].ID == "") {
			return fmt.Errorf("support frame %q has invalid local references", f.ID)
		}
		seen[f.ID] = true
		canonical, err := rigidFrame(f.LocalFrame)
		if err != nil {
			return fmt.Errorf("support frame %q: %w", f.ID, err)
		}
		f.LocalFrame = canonical
		if f.LoadDirectionLocal != nil {
			if err := f.LoadDirectionLocal.Validate(); err != nil || (f.LoadDirectionLocal.X == 0 && f.LoadDirectionLocal.Y == 0 && f.LoadDirectionLocal.Z == 0) {
				return fmt.Errorf("support frame %q load direction must be finite and nonzero", f.ID)
			}
		}
	}
	return nil
}

func validateCameras(cameras []SceneCameraFrame, parts map[string]ScenePart) error {
	seen := make(map[string]bool, len(cameras))
	for i := range cameras {
		c := &cameras[i]
		if !library.IsValidStableID(c.ID) || seen[c.ID] || !positive(c.NearClipM) || !positive(c.FarClipM) || c.FarClipM <= c.NearClipM {
			return fmt.Errorf("camera frame %q has invalid identity or clipping planes", c.ID)
		}
		seen[c.ID] = true
		frame, err := rigidFrame(c.Frame)
		if err != nil {
			return fmt.Errorf("camera frame %q: %w", c.ID, err)
		}
		c.Frame = frame
		switch c.Projection {
		case ProjectionPerspective:
			if c.FocalLengthMM == nil || !positive(*c.FocalLengthMM) || c.SensorWidthMM == nil || !positive(*c.SensorWidthMM) || c.SensorHeightMM == nil || !positive(*c.SensorHeightMM) || (c.SensorFit != SensorFitHorizontal && c.SensorFit != SensorFitVertical && c.SensorFit != SensorFitAuto) || c.OrthographicSpanM != nil {
				return fmt.Errorf("camera frame %q perspective optics are incomplete or mixed with orthographic fields", c.ID)
			}
		case ProjectionOrthographic:
			if c.OrthographicSpanM == nil || !positive(*c.OrthographicSpanM) || c.FocalLengthMM != nil || c.SensorWidthMM != nil || c.SensorHeightMM != nil || c.SensorFit != "" {
				return fmt.Errorf("camera frame %q orthographic optics are incomplete or mixed with perspective fields", c.ID)
			}
		default:
			return fmt.Errorf("camera frame %q projection is unsupported", c.ID)
		}
		if c.Focus == nil {
			if c.DOFFNumber != nil {
				return fmt.Errorf("camera frame %q has f-number while depth of field is disabled", c.ID)
			}
			continue
		}
		if c.DOFFNumber == nil || !positive(*c.DOFFNumber) {
			return fmt.Errorf("camera frame %q enabled depth of field requires a positive f-number", c.ID)
		}
		switch c.Focus.Mode {
		case FocusScenePart:
			if !library.IsValidStableID(c.Focus.ScenePartID) || parts[c.Focus.ScenePartID].ID == "" || c.Focus.DistanceM != nil {
				return fmt.Errorf("camera frame %q scene-part focus target is invalid", c.ID)
			}
		case FocusSelectedDevice:
			if c.Focus.ScenePartID != "" || c.Focus.DistanceM != nil {
				return fmt.Errorf("camera frame %q selected-device focus has extraneous target fields", c.ID)
			}
		case FocusDistance:
			if c.Focus.ScenePartID != "" || c.Focus.DistanceM == nil || !positive(*c.Focus.DistanceM) {
				return fmt.Errorf("camera frame %q distance focus requires only a positive distance", c.ID)
			}
		default:
			return fmt.Errorf("camera frame %q focus mode is unsupported", c.ID)
		}
	}
	return nil
}

func validateLights(lights []SceneLightFrame) error {
	seen := make(map[string]bool, len(lights))
	for i := range lights {
		l := &lights[i]
		if !library.IsValidStableID(l.ID) || seen[l.ID] || l.LocalAxisProfile != "AREA_XY_MINUS_Z_V1" || (l.Shape != LightRectangle && l.Shape != LightDisk) || !positive(l.DimensionsM[0]) || (l.Shape == LightRectangle && !positive(l.DimensionsM[1])) || (l.Shape == LightDisk && l.DimensionsM[1] != 0) || !positive(l.PowerWatts) || !unitRGB(l.LinearRGB[:]) {
			return fmt.Errorf("light frame %q has invalid shape, axes, dimensions, power, or color", l.ID)
		}
		seen[l.ID] = true
		frame, err := rigidFrame(l.Frame)
		if err != nil {
			return fmt.Errorf("light frame %q: %w", l.ID, err)
		}
		l.Frame = frame
	}
	return nil
}

func rigidFrame(frame physical.TRS) (physical.TRS, error) {
	frame, err := physical.CanonicalizeTRS(frame)
	if err != nil || frame.Scale != (physical.Vec3{X: 1, Y: 1, Z: 1}) {
		return physical.TRS{}, errors.New("frame must be finite and rigid with unit scale")
	}
	return frame, nil
}

func cloneCandidate(in SceneCandidate) SceneCandidate {
	out := in
	out.Artifacts = append([]SceneArtifactBinding(nil), in.Artifacts...)
	for i := range out.Artifacts {
		if in.Artifacts[i].SourceToCanonical != nil {
			value := *in.Artifacts[i].SourceToCanonical
			out.Artifacts[i].SourceToCanonical = &value
		}
	}
	out.Materials = append([]SceneMaterial(nil), in.Materials...)
	for i := range out.Materials {
		out.Materials[i].Inputs = append([]SceneMaterialInput(nil), in.Materials[i].Inputs...)
		for j := range out.Materials[i].Inputs {
			src, dst := in.Materials[i].Inputs[j], &out.Materials[i].Inputs[j]
			if src.AffineUV != nil {
				value := *src.AffineUV
				dst.AffineUV = &value
			}
			if src.PhysicalRepeatM != nil {
				value := *src.PhysicalRepeatM
				dst.PhysicalRepeatM = &value
			}
			if src.NormalStrength != nil {
				value := *src.NormalStrength
				dst.NormalStrength = &value
			}
			if src.BumpStrength != nil {
				value := *src.BumpStrength
				dst.BumpStrength = &value
			}
			if src.BumpDistanceM != nil {
				value := *src.BumpDistanceM
				dst.BumpDistanceM = &value
			}
		}
		if in.Materials[i].Grade != nil {
			value := *in.Materials[i].Grade
			out.Materials[i].Grade = &value
		}
		if in.Materials[i].AlphaCutoff != nil {
			value := *in.Materials[i].AlphaCutoff
			out.Materials[i].AlphaCutoff = &value
		}
	}
	out.MaterialSlots = append([]SceneMaterialSlot(nil), in.MaterialSlots...)
	out.Parts = append([]ScenePart(nil), in.Parts...)
	out.SupportFrames = append([]SceneSupportFrame(nil), in.SupportFrames...)
	for i := range out.SupportFrames {
		if in.SupportFrames[i].LoadDirectionLocal != nil {
			value := *in.SupportFrames[i].LoadDirectionLocal
			out.SupportFrames[i].LoadDirectionLocal = &value
		}
	}
	out.CameraFrames = append([]SceneCameraFrame(nil), in.CameraFrames...)
	for i := range out.CameraFrames {
		src, dst := in.CameraFrames[i], &out.CameraFrames[i]
		if src.FocalLengthMM != nil {
			value := *src.FocalLengthMM
			dst.FocalLengthMM = &value
		}
		if src.SensorWidthMM != nil {
			value := *src.SensorWidthMM
			dst.SensorWidthMM = &value
		}
		if src.SensorHeightMM != nil {
			value := *src.SensorHeightMM
			dst.SensorHeightMM = &value
		}
		if src.OrthographicSpanM != nil {
			value := *src.OrthographicSpanM
			dst.OrthographicSpanM = &value
		}
		if src.DOFFNumber != nil {
			value := *src.DOFFNumber
			dst.DOFFNumber = &value
		}
		if src.Focus != nil {
			value := *src.Focus
			if src.Focus.DistanceM != nil {
				distance := *src.Focus.DistanceM
				value.DistanceM = &distance
			}
			dst.Focus = &value
		}
	}
	out.LightFrames = append([]SceneLightFrame(nil), in.LightFrames...)
	out.Regions = append([]SceneRegion(nil), in.Regions...)
	return out
}

func slotKey(slot SceneMaterialSlot) string {
	return fmt.Sprintf("%s/%010d/%010d", slot.PartID, slot.PrimitiveIndex, slot.SlotIndex)
}
func unit(value float64) bool { return finite(value) && value >= 0 && value <= 1 }
func unitRGB(values []float64) bool {
	for _, value := range values {
		if !unit(value) {
			return false
		}
	}
	return true
}
func positive(value float64) bool          { return finite(value) && value > 0 }
func finiteNonnegative(value float64) bool { return finite(value) && value >= 0 }
func finite(value float64) bool            { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
