package devices

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"device-studio/internal/library"
	"device-studio/internal/physical"
)

const (
	DeviceCandidateVersion = "device-candidate/v1"
	MaxDeviceSnapshotBytes = 1 << 20
	MaxDeviceParts         = 256
	MaxDeviceBindings      = 128
)

var (
	ErrRevisionNotFound = errors.New("device revision not found")
	ErrRevisionConflict = errors.New("device revision head conflict")
	ErrInvalidCandidate = errors.New("invalid device candidate")
	sha256Pattern       = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// DeviceCandidate is an immutable, internally consistent candidate snapshot.
// It does not assert that node indices or names exist in the referenced GLB.
type DeviceCandidate struct {
	Version              string               `json:"version"`
	Definition           Definition           `json:"definition"`
	Geometry             physical.ArtifactRef `json:"geometry"`
	GeometryUnit         string               `json:"geometry_unit"`
	CoordinateConvention string               `json:"coordinate_convention"`
	SourceToCanonical    physical.TRS         `json:"source_to_canonical"`
	Parts                []SemanticPart       `json:"parts"`
	DisplayBindings      []DisplayPartBinding `json:"display_bindings"`
	SupportFrames        []DeviceSupportFrame `json:"support_frames"`
}

type SemanticPart struct {
	ID         string       `json:"id"`
	Role       string       `json:"role"`
	ParentID   string       `json:"parent_id,omitempty"`
	NodeIndex  uint32       `json:"node_index"`
	NodeName   string       `json:"node_name"`
	LocalFrame physical.TRS `json:"local_frame"`
}

type DisplayPartBinding struct {
	DisplayID            string       `json:"display_id"`
	PartID               string       `json:"part_id"`
	LocalFrame           physical.TRS `json:"local_frame"`
	ScreenPrimitiveIndex *uint32      `json:"screen_primitive_index,omitempty"`
	ScreenMaterialIndex  *uint32      `json:"screen_material_index,omitempty"`
}

type DeviceSupportFrame struct {
	ID          string       `json:"id"`
	InterfaceID string       `json:"interface_id"`
	PartID      string       `json:"part_id"`
	LocalFrame  physical.TRS `json:"local_frame"`
}

type DeviceRevision struct {
	DeviceID  string              `json:"device_id"`
	Revision  uint32              `json:"revision"`
	SHA256    string              `json:"sha256"`
	State     DeviceRevisionState `json:"state"`
	Candidate DeviceCandidate     `json:"candidate"`
	CreatedAt time.Time           `json:"created_at"`
	canonical []byte
}

type QualificationReviewRequest struct {
	ProjectID         string
	DeviceID          string
	CandidateRevision uint32
	CandidateSHA256   string
	ExpectedHead      uint32
	Review            QualificationReview
}

type QualificationReviewResult struct {
	Revision       DeviceRevision
	PresetID       string
	PresetRevision uint32
	PresetSHA256   string
}

type DeviceRevisionState string

const DeviceRevisionCandidate DeviceRevisionState = "candidate"
const DeviceRevisionQualified DeviceRevisionState = "qualified"

// CanonicalizeDeviceCandidate validates semantic references and normalizes
// transforms/order before producing the immutable candidate digest.
func CanonicalizeDeviceCandidate(input DeviceCandidate) (DeviceCandidate, []byte, string, error) {
	if input.Version != DeviceCandidateVersion {
		return DeviceCandidate{}, nil, "", fmt.Errorf("unsupported device candidate version %q", input.Version)
	}
	if err := input.Definition.Validate(); err != nil {
		return DeviceCandidate{}, nil, "", fmt.Errorf("device definition: %w", err)
	}
	if !library.IsValidStableID(input.Definition.ID) {
		return DeviceCandidate{}, nil, "", errors.New("device id must be a valid stable identifier")
	}
	if err := validateGeometryRef(input.Geometry); err != nil {
		return DeviceCandidate{}, nil, "", err
	}
	if input.GeometryUnit != "m" {
		return DeviceCandidate{}, nil, "", errors.New("whole-device GLB geometry unit must be meters")
	}
	if input.CoordinateConvention != physical.CoordinateConvention {
		return DeviceCandidate{}, nil, "", errors.New("unsupported device coordinate convention")
	}
	var err error
	input.SourceToCanonical, err = physical.CanonicalizeTRS(input.SourceToCanonical)
	if err != nil {
		return DeviceCandidate{}, nil, "", fmt.Errorf("source-to-canonical transform: %w", err)
	}
	if len(input.Parts) == 0 || len(input.Parts) > MaxDeviceParts {
		return DeviceCandidate{}, nil, "", fmt.Errorf("parts must contain between 1 and %d entries", MaxDeviceParts)
	}
	if len(input.DisplayBindings) > MaxDeviceBindings || len(input.SupportFrames) > MaxDeviceBindings {
		return DeviceCandidate{}, nil, "", errors.New("device display/support binding count exceeds limit")
	}
	input.Parts = append([]SemanticPart(nil), input.Parts...)
	parts := make(map[string]SemanticPart, len(input.Parts))
	nodes := make(map[uint32]string, len(input.Parts))
	for i := range input.Parts {
		part := &input.Parts[i]
		if !library.IsValidStableID(part.ID) || !library.IsValidStableID(part.Role) || strings.TrimSpace(part.NodeName) == "" || len(part.NodeName) > 256 {
			return DeviceCandidate{}, nil, "", fmt.Errorf("part %d has invalid stable id, role, or node name", i)
		}
		if _, exists := parts[part.ID]; exists {
			return DeviceCandidate{}, nil, "", fmt.Errorf("duplicate part id %q", part.ID)
		}
		if previous, exists := nodes[part.NodeIndex]; exists {
			return DeviceCandidate{}, nil, "", fmt.Errorf("GLB node index %d is bound by both %q and %q", part.NodeIndex, previous, part.ID)
		}
		part.LocalFrame, err = physical.CanonicalizeTRS(part.LocalFrame)
		if err != nil {
			return DeviceCandidate{}, nil, "", fmt.Errorf("part %q local frame: %w", part.ID, err)
		}
		parts[part.ID] = *part
		nodes[part.NodeIndex] = part.ID
	}
	for _, part := range input.Parts {
		if part.ParentID != "" {
			if _, ok := parts[part.ParentID]; !ok || part.ParentID == part.ID {
				return DeviceCandidate{}, nil, "", fmt.Errorf("part %q has an unknown or self parent", part.ID)
			}
		}
	}
	if err := validateParentGraph(input.Parts); err != nil {
		return DeviceCandidate{}, nil, "", err
	}
	if err := validateDefinitionBindings(input.Definition, parts, input.DisplayBindings, input.SupportFrames); err != nil {
		return DeviceCandidate{}, nil, "", err
	}
	input.DisplayBindings = append([]DisplayPartBinding(nil), input.DisplayBindings...)
	input.SupportFrames = append([]DeviceSupportFrame(nil), input.SupportFrames...)
	for i := range input.DisplayBindings {
		binding := &input.DisplayBindings[i]
		if (binding.ScreenPrimitiveIndex == nil) != (binding.ScreenMaterialIndex == nil) {
			return DeviceCandidate{}, nil, "", fmt.Errorf("display %q must pin both screen primitive and material indices", binding.DisplayID)
		}
		binding.LocalFrame, err = physical.CanonicalizeTRS(binding.LocalFrame)
		if err != nil {
			return DeviceCandidate{}, nil, "", fmt.Errorf("display %q local frame: %w", binding.DisplayID, err)
		}
	}
	for i := range input.SupportFrames {
		frame := &input.SupportFrames[i]
		frame.LocalFrame, err = physical.CanonicalizeTRS(frame.LocalFrame)
		if err != nil {
			return DeviceCandidate{}, nil, "", fmt.Errorf("support frame %q local frame: %w", frame.ID, err)
		}
	}
	sort.Slice(input.Parts, func(i, j int) bool { return input.Parts[i].ID < input.Parts[j].ID })
	sort.Slice(input.DisplayBindings, func(i, j int) bool { return input.DisplayBindings[i].DisplayID < input.DisplayBindings[j].DisplayID })
	sort.Slice(input.SupportFrames, func(i, j int) bool { return input.SupportFrames[i].ID < input.SupportFrames[j].ID })
	data, err := json.Marshal(input)
	if err != nil {
		return DeviceCandidate{}, nil, "", fmt.Errorf("encode device candidate: %w", err)
	}
	if len(data) > MaxDeviceSnapshotBytes {
		return DeviceCandidate{}, nil, "", errors.New("device candidate exceeds stored snapshot limit")
	}
	digest := sha256.Sum256(data)
	return input, data, hex.EncodeToString(digest[:]), nil
}

func validateGeometryRef(ref physical.ArtifactRef) error {
	if !library.IsValidStableID(ref.BundleID) || ref.BundleRevision == 0 || !library.IsValidStableID(ref.ArtifactID) ||
		ref.Role != library.RoleGeometryGLB || ref.SizeBytes <= 0 || ref.SizeBytes > library.MaxGeometryBytes ||
		!sha256Pattern.MatchString(ref.ManifestSHA256) || !sha256Pattern.MatchString(ref.ReviewSHA256) || !sha256Pattern.MatchString(ref.SHA256) {
		return errors.New("geometry reference must pin a reviewed GLB artifact, manifest, review, size, and SHA-256")
	}
	return nil
}

func validateParentGraph(parts []SemanticPart) error {
	parents := make(map[string]string, len(parts))
	for _, part := range parts {
		parents[part.ID] = part.ParentID
	}
	for id := range parents {
		seen := make(map[string]bool)
		for current := id; current != ""; current = parents[current] {
			if seen[current] {
				return fmt.Errorf("part parent graph contains a cycle at %q", current)
			}
			seen[current] = true
		}
	}
	return nil
}

func validateDefinitionBindings(def Definition, parts map[string]SemanticPart, displays []DisplayPartBinding, supports []DeviceSupportFrame) error {
	displayIDs := make(map[string]bool, len(def.Displays))
	for _, display := range def.Displays {
		displayIDs[display.ID] = true
	}
	seenDisplays := make(map[string]bool, len(displays))
	for _, binding := range displays {
		if !library.IsValidStableID(binding.DisplayID) || !displayIDs[binding.DisplayID] || !hasPart(parts, binding.PartID) || seenDisplays[binding.DisplayID] {
			return fmt.Errorf("display binding %q has an unknown semantic reference or duplicate", binding.DisplayID)
		}
		seenDisplays[binding.DisplayID] = true
	}
	interfaces := make(map[string]bool, len(def.Supports))
	for _, support := range def.Supports {
		if support.InterfaceID != "" {
			interfaces[support.InterfaceID] = true
		}
	}
	frameIDs := make(map[string]bool, len(supports))
	for _, frame := range supports {
		if !library.IsValidStableID(frame.ID) || !interfaces[frame.InterfaceID] || !hasPart(parts, frame.PartID) || frameIDs[frame.ID] {
			return fmt.Errorf("support frame %q has an unknown semantic reference or duplicate", frame.ID)
		}
		frameIDs[frame.ID] = true
	}
	jointIDs, jointGraph := make(map[string]bool), make(map[string]string)
	for _, joint := range def.Joints {
		if !library.IsValidStableID(joint.ID) || jointIDs[joint.ID] || !hasPart(parts, joint.ParentPart) || !hasPart(parts, joint.ChildPart) || joint.ParentPart == joint.ChildPart {
			return fmt.Errorf("joint %q has duplicate or unknown part references", joint.ID)
		}
		jointIDs[joint.ID] = true
		if _, exists := jointGraph[joint.ChildPart]; exists {
			return fmt.Errorf("joint child %q has multiple joints", joint.ChildPart)
		}
		if parts[joint.ChildPart].ParentID != joint.ParentPart {
			return fmt.Errorf("joint %q parent does not match declared part hierarchy", joint.ID)
		}
		jointGraph[joint.ChildPart] = joint.ParentPart
	}
	for child := range jointGraph {
		seen := map[string]bool{}
		for current := child; current != ""; current = jointGraph[current] {
			if seen[current] {
				return fmt.Errorf("joint parent graph contains a cycle at %q", current)
			}
			seen[current] = true
		}
	}
	for _, capability := range def.SupportedMotion {
		for _, id := range capability.JointIDs {
			if !jointIDs[id] {
				return fmt.Errorf("motion capability %q references unknown joint %q", capability.Name, id)
			}
		}
	}
	for _, pose := range def.SupportedPoses {
		for id := range pose.JointState {
			if !jointIDs[id] {
				return fmt.Errorf("pose %q references unknown joint %q", pose.Name, id)
			}
		}
	}
	return nil
}

func hasPart(parts map[string]SemanticPart, id string) bool {
	_, ok := parts[id]
	return ok
}

func decodeCanonicalCandidate(data []byte) (DeviceCandidate, string, error) {
	if len(data) == 0 || len(data) > MaxDeviceSnapshotBytes {
		return DeviceCandidate{}, "", errors.New("stored device candidate size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var candidate DeviceCandidate
	if err := decoder.Decode(&candidate); err != nil {
		return DeviceCandidate{}, "", fmt.Errorf("decode stored device candidate: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return DeviceCandidate{}, "", errors.New("stored device candidate has trailing JSON")
	}
	canonical, encoded, digest, err := CanonicalizeDeviceCandidate(candidate)
	if err != nil {
		return DeviceCandidate{}, "", fmt.Errorf("validate stored device candidate: %w", err)
	}
	if !bytes.Equal(encoded, data) {
		return DeviceCandidate{}, "", errors.New("stored device candidate is not canonical")
	}
	return canonical, digest, nil
}
