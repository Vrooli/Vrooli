package physicalruntime

import (
	"errors"
	"fmt"
	"math"

	"device-studio/internal/devices"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
)

// physicalGLBJointSweepBounds returns a conservative bound for fixed geometry
// and every point in one supported revolute-joint interval. Primitive AABBs
// are expanded to corners; articulated corners are rotated analytically,
// including interior coordinate extrema between authored motion keys.
func physicalGLBJointSweepBounds(inventory rendering.PhysicalGLBInventory, candidate devices.DeviceCandidate, motion motions.MotionCandidate, jointID string) (physicalBounds, error) {
	var joint *devices.Joint
	for i := range candidate.Definition.Joints {
		if candidate.Definition.Joints[i].ID == jointID {
			if joint != nil {
				return physicalBounds{}, errors.New("hinge sweep has duplicate joint identities")
			}
			joint = &candidate.Definition.Joints[i]
		}
	}
	if joint == nil || joint.Axis == nil || joint.PivotMeters == nil || joint.LowerRadians == nil || joint.UpperRadians == nil {
		return physicalBounds{}, errors.New("hinge sweep requires one fully bounded joint")
	}
	axis := *joint.Axis
	axisLength := math.Sqrt(dotVec(axis, axis))
	if !positiveFinite(axisLength) || math.Abs(axisLength-1) > 1e-6 || !positiveFinite(*joint.UpperRadians-*joint.LowerRadians) {
		return physicalBounds{}, errors.New("hinge sweep axis or angle interval is invalid")
	}
	axis = scaleVec(axis, 1/axisLength)
	partNodes := make(map[string]uint32, len(candidate.Parts))
	for _, part := range candidate.Parts {
		partNodes[part.ID] = part.NodeIndex
	}
	parentIndex, parentOK := partNodes[joint.ParentPart]
	childIndex, childOK := partNodes[joint.ChildPart]
	nodeParentCheck := make(map[uint32]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodeParentCheck[node.Index] = node
	}
	if !parentOK || !childOK || parentIndex == childIndex || !isGLBDescendant(nodeParentCheck, childIndex, parentIndex) {
		return physicalBounds{}, fmt.Errorf("hinge sweep joint does not resolve distinct semantic GLB parts: parent=%s/%d child=%s/%d parent_ok=%t child_ok=%t desc=%t", joint.ParentPart, parentIndex, joint.ChildPart, childIndex, parentOK, childOK, isGLBDescendant(nodeParentCheck, childIndex, parentIndex))
	}
	angles := make([]float64, 0, 4)
	matchedTracks := 0
	for _, track := range motion.Tracks {
		if track.Target.JointID != jointID {
			continue
		}
		matchedTracks++
		for _, key := range track.Keys {
			if key.AngleRad == nil {
				return physicalBounds{}, errors.New("hinge sweep motion contains a non-angular joint key")
			}
			angles = append(angles, *key.AngleRad)
		}
	}
	if len(angles) == 0 || matchedTracks != 1 || len(motion.Tracks) != 1 {
		return physicalBounds{}, errors.New("hinge sweep motion must contain exactly one track for the selected joint")
	}
	low, high := angles[0], angles[0]
	for _, angle := range angles[1:] {
		low, high = math.Min(low, angle), math.Max(high, angle)
	}
	if low < *joint.LowerRadians-1e-9 || high > *joint.UpperRadians+1e-9 || math.Abs(low-*joint.LowerRadians) > 1e-9 || math.Abs(high-*joint.UpperRadians) > 1e-9 || high-low > 2*math.Pi {
		return physicalBounds{}, errors.New("hinge sweep keys do not span the bounded joint interval")
	}
	nodes := make(map[uint32]rendering.PhysicalGLBNode, len(inventory.Nodes))
	meshes := make(map[uint32]rendering.PhysicalGLBMesh, len(inventory.Meshes))
	for _, node := range inventory.Nodes {
		nodes[node.Index] = node
	}
	for _, mesh := range inventory.Meshes {
		meshes[mesh.Index] = mesh
	}
	world := make(map[uint32]physical.Matrix4, len(nodes))
	visiting := make(map[uint32]bool, len(nodes))
	var matrixFor func(uint32) (physical.Matrix4, error)
	matrixFor = func(index uint32) (physical.Matrix4, error) {
		if m, ok := world[index]; ok {
			return m, nil
		}
		if visiting[index] {
			return physical.Matrix4{}, errors.New("hinge sweep GLB hierarchy contains a cycle")
		}
		node, ok := nodes[index]
		if !ok {
			return physical.Matrix4{}, fmt.Errorf("hinge sweep GLB node %d is absent", index)
		}
		visiting[index] = true
		m, err := node.Frame.Matrix()
		if err != nil {
			return physical.Matrix4{}, err
		}
		if node.Parent != nil {
			parent, err := matrixFor(*node.Parent)
			if err != nil {
				return physical.Matrix4{}, err
			}
			m, err = physical.Compose(parent, m)
			if err != nil {
				return physical.Matrix4{}, err
			}
		}
		visiting[index] = false
		world[index] = m
		return m, nil
	}
	parentWorld, err := matrixFor(parentIndex)
	if err != nil {
		return physicalBounds{}, err
	}
	parentInverse, err := inverseAffine(parentWorld)
	if err != nil {
		return physicalBounds{}, err
	}
	bounds := physicalBounds{min: physical.Vec3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}, max: physical.Vec3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}}
	for _, node := range inventory.Nodes {
		if node.Mesh == nil {
			continue
		}
		mesh, ok := meshes[*node.Mesh]
		if !ok {
			return physicalBounds{}, fmt.Errorf("hinge sweep node %d references absent mesh", node.Index)
		}
		nodeWorld, err := matrixFor(node.Index)
		if err != nil {
			return physicalBounds{}, err
		}
		moving := isGLBDescendant(nodes, node.Index, childIndex)
		for _, primitive := range mesh.Primitives {
			for mask := 0; mask < 8; mask++ {
				p := physical.Vec3{}
				for axisIndex := 0; axisIndex < 3; axisIndex++ {
					v := primitive.PositionMin[axisIndex]
					if mask&(1<<axisIndex) != 0 {
						v = primitive.PositionMax[axisIndex]
					}
					switch axisIndex {
					case 0:
						p.X = v
					case 1:
						p.Y = v
					case 2:
						p.Z = v
					}
				}
				if !moving {
					p, err = physical.TransformPoint(nodeWorld, p)
					if err != nil {
						return physicalBounds{}, err
					}
					bounds.include(p)
					continue
				}
				relative, err := physical.Compose(parentInverse, nodeWorld)
				if err != nil {
					return physicalBounds{}, err
				}
				p, err = physical.TransformPoint(relative, p)
				if err != nil {
					return physicalBounds{}, err
				}
				for _, angle := range hingeExtremaAngles(p, *joint.PivotMeters, axis, low, high) {
					rotated := rotateAroundAxis(p, *joint.PivotMeters, axis, angle)
					pWorld, err := physical.TransformPoint(parentWorld, rotated)
					if err != nil {
						return physicalBounds{}, err
					}
					bounds.include(pWorld)
				}
			}
		}
	}
	if !bounds.valid() {
		return physicalBounds{}, errors.New("hinge sweep produced no finite geometry bounds")
	}
	return bounds, nil
}

func isGLBDescendant(nodes map[uint32]rendering.PhysicalGLBNode, index, ancestor uint32) bool {
	seen := make(map[uint32]struct{}, len(nodes))
	for {
		if index == ancestor {
			return true
		}
		if _, ok := seen[index]; ok {
			return false
		}
		seen[index] = struct{}{}
		node, ok := nodes[index]
		if !ok || node.Parent == nil {
			return false
		}
		index = *node.Parent
	}
}

func hingeExtremaAngles(point, pivot, axis physical.Vec3, low, high float64) []float64 {
	delta := subVec(point, pivot)
	parallel := scaleVec(axis, dotVec(axis, delta))
	u := subVec(delta, parallel)
	v := crossVec(axis, delta)
	angles := []float64{low, high}
	for _, pair := range [][2]float64{{u.X, v.X}, {u.Y, v.Y}, {u.Z, v.Z}} {
		if math.Abs(pair[0])+math.Abs(pair[1]) < 1e-15 {
			continue
		}
		base := math.Atan2(pair[1], pair[0])
		first := int(math.Ceil((low - base) / math.Pi))
		last := int(math.Floor((high - base) / math.Pi))
		for k := first; k <= last; k++ {
			angles = append(angles, base+float64(k)*math.Pi)
		}
	}
	return angles
}

func rotateAroundAxis(point, pivot, axis physical.Vec3, angle float64) physical.Vec3 {
	delta := subVec(point, pivot)
	cos, sin := math.Cos(angle), math.Sin(angle)
	parallel := scaleVec(axis, dotVec(axis, delta))
	u := subVec(delta, parallel)
	return addVec(pivot, addVec(parallel, addVec(scaleVec(u, cos), scaleVec(crossVec(axis, delta), sin))))
}

func crossVec(a, b physical.Vec3) physical.Vec3 {
	return physical.Vec3{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}

func inverseAffine(m physical.Matrix4) (physical.Matrix4, error) {
	a, b, c := m[0], m[4], m[8]
	d, e, f := m[1], m[5], m[9]
	g, h, i := m[2], m[6], m[10]
	det := a*(e*i-f*h) - b*(d*i-f*g) + c*(d*h-e*g)
	if !positiveFinite(math.Abs(det)) || math.Abs(det) < 1e-15 || math.Abs(m[3])+math.Abs(m[7])+math.Abs(m[11])+math.Abs(m[15]-1) > 1e-12 {
		return physical.Matrix4{}, errors.New("hinge parent transform is not invertible affine")
	}
	inv := 1 / det
	r00, r01, r02 := (e*i-f*h)*inv, (c*h-b*i)*inv, (b*f-c*e)*inv
	r10, r11, r12 := (f*g-d*i)*inv, (a*i-c*g)*inv, (c*d-a*f)*inv
	r20, r21, r22 := (d*h-e*g)*inv, (b*g-a*h)*inv, (a*e-b*d)*inv
	out := physical.Matrix4{r00, r10, r20, 0, r01, r11, r21, 0, r02, r12, r22, 0, 0, 0, 0, 1}
	tx, ty, tz := m[12], m[13], m[14]
	out[12] = -(r00*tx + r01*ty + r02*tz)
	out[13] = -(r10*tx + r11*ty + r12*tz)
	out[14] = -(r20*tx + r21*ty + r22*tz)
	return out, nil
}
