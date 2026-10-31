package physicalruntime

import (
	"math"
	"testing"

	"device-studio/internal/devices"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
)

func TestPhysicalGLBJointSweepBoundsIncludesInteriorExtremaAndFixedGeometry(t *testing.T) {
	parent, child := uint32(0), uint32(1)
	inventory := rendering.PhysicalGLBInventory{
		Nodes: []rendering.PhysicalGLBNode{
			{Index: 0, Name: "root", Frame: identityTRS()},
			{Index: 1, Name: "hinge", Parent: &parent, Frame: identityTRS(), Mesh: uint32Ptr(0)},
			{Index: 2, Name: "fixed", Frame: physical.TRS{Translation: physical.Vec3{X: 3}, Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}, Mesh: uint32Ptr(1)},
		},
		Meshes: []rendering.PhysicalGLBMesh{
			{Index: 0, Primitives: []rendering.PhysicalGLBPrimitive{{PositionMin: [3]float64{1, 0, 0}, PositionMax: [3]float64{1, 0, 0}}}},
			{Index: 1, Primitives: []rendering.PhysicalGLBPrimitive{{PositionMin: [3]float64{0, 0, 0}, PositionMax: [3]float64{0, 0, 0}}}},
		},
	}
	low, high := .2, 2.0
	candidate := devices.DeviceCandidate{
		Parts:      []devices.SemanticPart{{ID: "parent", NodeIndex: 0}, {ID: "child", ParentID: "parent", NodeIndex: 1}},
		Definition: devices.Definition{Joints: []devices.Joint{{ID: "lid", ParentPart: "parent", ChildPart: "child", Axis: &physical.Vec3{Z: 1}, PivotMeters: &physical.Vec3{}, LowerRadians: &low, UpperRadians: &high}}},
	}
	motion := motions.MotionCandidate{Tracks: []motions.MotionTrack{{Target: motions.TrackTarget{JointID: "lid"}, Keys: []motions.MotionKey{{Tick: 0, AngleRad: &low}, {Tick: 1, AngleRad: &high}}}}}
	bounds, err := physicalGLBJointSweepBounds(inventory, candidate, motion, "lid")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(bounds.min.X-math.Cos(high)) > 1e-12 || math.Abs(bounds.max.X-3) > 1e-12 || math.Abs(bounds.max.Y-1) > 1e-12 || math.Abs(bounds.min.Y) > 1e-12 {
		t.Fatalf("analytic sweep bounds=%+v; want endpoint x minimum, fixed x maximum 3, and interior y maximum 1", bounds)
	}
	if !isGLBDescendant(map[uint32]rendering.PhysicalGLBNode{0: inventory.Nodes[0], 1: inventory.Nodes[1]}, 1, child) || isGLBDescendant(map[uint32]rendering.PhysicalGLBNode{0: inventory.Nodes[0], 1: inventory.Nodes[1]}, 0, child) {
		t.Fatal("hinge descendant classification does not distinguish moving child geometry from fixed parent geometry")
	}
}

func uint32Ptr(value uint32) *uint32 { return &value }
