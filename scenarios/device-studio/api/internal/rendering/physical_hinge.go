package rendering

import (
	"errors"
	"fmt"
	"math"

	"device-studio/internal/devices"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
)

// PhysicalJointBinding is derived from immutable candidate snapshots and the
// inspected GLB. It is receipt evidence, not a second source of joint values.
type PhysicalJointBinding struct {
	JointID      string          `json:"joint_id"`
	Status       string          `json:"candidate_status"`
	ParentPartID string          `json:"parent_part_id"`
	ChildPartID  string          `json:"child_part_id"`
	ParentNode   uint32          `json:"parent_node_index"`
	ChildNode    uint32          `json:"child_node_index"`
	Axis         devices.Vector3 `json:"axis_parent_local"`
	PivotMeters  devices.Vector3 `json:"pivot_parent_local_m"`
	LowerRadians float64         `json:"lower_radians"`
	UpperRadians float64         `json:"upper_radians"`
	RenderScope  string          `json:"render_scope"`
}

// ValidatePhysicalArticulation resolves exactly one hinge track and verifies
// every sampled angle against its pinned candidate. When inventory is present,
// it also checks the semantic rest hierarchy against the exact imported GLB.
func ValidatePhysicalArticulation(device devices.DeviceCandidate, motion motions.MotionCandidate, samples []PhysicalMotionSample, inventory PhysicalGLBInventory) (PhysicalJointBinding, error) {
	var result PhysicalJointBinding
	jointID := ""
	jointTracks := 0
	for _, track := range motion.Tracks {
		if track.Target.JointID == "" {
			continue
		}
		jointTracks++
		if jointTracks > 1 {
			return result, errors.New("single-hinge profile does not support multiple joint tracks")
		}
		jointID = track.Target.JointID
	}
	if jointID == "" || len(samples) == 0 {
		return result, errors.New("single-hinge profile requires one selected joint and sampled poses")
	}
	var joint *devices.Joint
	for i := range device.Definition.Joints {
		if device.Definition.Joints[i].ID == jointID {
			joint = &device.Definition.Joints[i]
			break
		}
	}
	if joint == nil || (joint.Status != "candidate_only" && joint.Status != "qualified") || joint.Axis == nil || joint.PivotMeters == nil || joint.LowerRadians == nil || joint.UpperRadians == nil {
		return result, errors.New("selected hinge must have explicit candidate or qualified axis, pivot, and limits")
	}
	if !finiteJointVector(*joint.Axis) || !finiteJointVector(*joint.PivotMeters) || math.Abs(math.Sqrt(joint.Axis.X*joint.Axis.X+joint.Axis.Y*joint.Axis.Y+joint.Axis.Z*joint.Axis.Z)-1) > 1e-6 ||
		!finiteJointNumber(*joint.LowerRadians) || !finiteJointNumber(*joint.UpperRadians) || *joint.LowerRadians >= *joint.UpperRadians {
		return result, errors.New("selected hinge axis, pivot, or radian interval is invalid")
	}
	if err := validateUniformPositiveScale(device.SourceToCanonical.Scale); err != nil {
		return result, fmt.Errorf("single-hinge source transform: %w", err)
	}
	parts := make(map[string]devices.SemanticPart, len(device.Parts))
	for _, part := range device.Parts {
		parts[part.ID] = part
	}
	parent, parentOK := parts[joint.ParentPart]
	child, childOK := parts[joint.ChildPart]
	if !parentOK || !childOK || child.ParentID != parent.ID || parent.ID == child.ID {
		return result, errors.New("selected hinge does not match the semantic parent-child part hierarchy")
	}
	for ordinal, sample := range samples {
		angle, ok := sample.JointAngles[jointID]
		if !ok || len(sample.JointAngles) != 1 || !finiteJointNumber(angle) || angle < *joint.LowerRadians || angle > *joint.UpperRadians {
			return result, fmt.Errorf("sample %d has a missing, unsupported, or out-of-range hinge angle", ordinal)
		}
	}
	result = PhysicalJointBinding{
		JointID: joint.ID, Status: joint.Status, ParentPartID: parent.ID, ChildPartID: child.ID,
		ParentNode: parent.NodeIndex, ChildNode: child.NodeIndex, Axis: *joint.Axis, PivotMeters: *joint.PivotMeters,
		LowerRadians: *joint.LowerRadians, UpperRadians: *joint.UpperRadians, RenderScope: "local-candidate-render-v1",
	}
	if len(inventory.Nodes) > 0 {
		if err := validatePhysicalHingeHierarchy(device.Parts, inventory, result); err != nil {
			return PhysicalJointBinding{}, err
		}
	}
	return result, nil
}

func physicalArticulationFromManifest(manifest PhysicalRenderManifest, deviceGLB []byte) (PhysicalJointBinding, error) {
	device, err := decodeDevice(manifest.Device.CanonicalJSON)
	if err != nil {
		return PhysicalJointBinding{}, fmt.Errorf("decode articulated device candidate: %w", err)
	}
	motion, err := decodeMotion(manifest.Motion.CanonicalJSON)
	if err != nil {
		return PhysicalJointBinding{}, fmt.Errorf("decode articulated motion candidate: %w", err)
	}
	inventory, err := ValidateDevicePhysicalGLB(deviceGLB, device, manifest.Screen)
	if err != nil {
		return PhysicalJointBinding{}, fmt.Errorf("validate articulated device GLB: %w", err)
	}
	return ValidatePhysicalArticulation(device, motion, manifest.Output.Samples, inventory)
}

func validatePhysicalHingeHierarchy(parts []devices.SemanticPart, inventory PhysicalGLBInventory, binding PhysicalJointBinding) error {
	nodes := make(map[uint32]PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodes[node.Index] = node
	}
	parent, parentOK := nodes[binding.ParentNode]
	child, childOK := nodes[binding.ChildNode]
	partByID := make(map[string]devices.SemanticPart, len(parts))
	for _, part := range parts {
		partByID[part.ID] = part
	}
	if !parentOK || !childOK || parent.Name != partByID[binding.ParentPartID].NodeName || child.Name != partByID[binding.ChildPartID].NodeName ||
		!trsNearlyEqual(parent.Frame, partByID[binding.ParentPartID].LocalFrame, 1e-6) || !trsNearlyEqual(child.Frame, partByID[binding.ChildPartID].LocalFrame, 1e-6) {
		return errors.New("selected hinge semantic nodes do not match their exact GLB rest frames")
	}
	if !glbNodeDescendsFrom(nodes, child.Index, parent.Index) {
		return errors.New("selected hinge child node is not below its parent node in the GLB hierarchy")
	}
	for current := parent; ; {
		if err := validateUniformPositiveScale(current.Frame.Scale); err != nil {
			return fmt.Errorf("selected hinge parent world chain: %w", err)
		}
		if current.Parent == nil {
			break
		}
		ancestor, ok := nodes[*current.Parent]
		if !ok {
			return errors.New("selected hinge parent world chain contains an unknown node")
		}
		current = ancestor
	}
	for current := child; current.Parent != nil && *current.Parent != parent.Index; {
		parentNode, ok := nodes[*current.Parent]
		if !ok {
			return errors.New("selected hinge GLB ancestry contains an unknown node")
		}
		current = parentNode
		if current.Mesh != nil {
			return errors.New("selected hinge GLB path has an unbound mesh-bearing intermediary")
		}
	}
	for _, part := range parts {
		if semanticPartDescendsFrom(partByID, part.ID, binding.ChildPartID) && !glbNodeDescendsFrom(nodes, part.NodeIndex, binding.ChildNode) {
			return fmt.Errorf("moving semantic part %q is outside the selected GLB child subtree", part.ID)
		}
	}
	return nil
}

func validateUniformPositiveScale(scale physical.Vec3) error {
	if !finiteJointNumber(scale.X) || !finiteJointNumber(scale.Y) || !finiteJointNumber(scale.Z) || scale.X <= 0 || scale.Y <= 0 || scale.Z <= 0 {
		return errors.New("scale must be finite and positive")
	}
	if math.Abs(scale.X-scale.Y) > 1e-7 || math.Abs(scale.X-scale.Z) > 1e-7 {
		return errors.New("nonuniform scale is unsupported by the single-revolute-hinge profile")
	}
	return nil
}

func glbNodeDescendsFrom(nodes map[uint32]PhysicalGLBNode, child, ancestor uint32) bool {
	seen := make(map[uint32]bool)
	for child != ancestor {
		if seen[child] {
			return false
		}
		seen[child] = true
		node, ok := nodes[child]
		if !ok || node.Parent == nil {
			return false
		}
		child = *node.Parent
	}
	return true
}

func semanticPartDescendsFrom(parts map[string]devices.SemanticPart, child, ancestor string) bool {
	seen := make(map[string]bool)
	for child != "" && child != ancestor {
		if seen[child] {
			return false
		}
		seen[child] = true
		part, ok := parts[child]
		if !ok {
			return false
		}
		child = part.ParentID
	}
	return child == ancestor
}

func finiteJointVector(v devices.Vector3) bool {
	return finiteJointNumber(v.X) && finiteJointNumber(v.Y) && finiteJointNumber(v.Z)
}

func finiteJointNumber(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
