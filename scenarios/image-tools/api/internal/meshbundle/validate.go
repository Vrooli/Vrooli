// Package meshbundle validates Image Tools image_to_object durable artifacts.
package meshbundle

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

const (
	MaxArchiveBytes = 128 << 20
	MaxGLBBytes     = 120 << 20
	maxVertices     = 1_500_000
	maxTriangles    = 3_000_000
)

type manifest struct {
	SchemaVersion int    `json:"schema_version"`
	Operation     string `json:"operation"`
	Model         struct {
		ID               string `json:"id"`
		Revision         string `json:"revision"`
		CheckpointSHA256 string `json:"checkpoint_sha256"`
	} `json:"model"`
	SourceCode struct {
		Commit        string `json:"commit"`
		SourceProfile struct {
			TreeSHA256 string `json:"tree_sha256"`
		} `json:"source_profile"`
	} `json:"source_code"`
	Runtime struct {
		Python                 string  `json:"python"`
		Torch                  string  `json:"torch"`
		Device                 string  `json:"device"`
		GPUUUID                string  `json:"gpu_uuid"`
		WorkerPID              int     `json:"worker_pid"`
		ProcessPeakRSSBytes    uint64  `json:"process_peak_rss_bytes"`
		CUDAPeakAllocatedBytes *uint64 `json:"cuda_peak_allocated_bytes"`
		CUDAPeakReservedBytes  *uint64 `json:"cuda_peak_reserved_bytes"`
	} `json:"runtime"`
	Input struct {
		SHA256 string `json:"sha256"`
	} `json:"input"`
	MeshInspection struct {
		GLBSHA256               string      `json:"glb_sha256"`
		Vertices                int         `json:"vertex_count"`
		Triangles               int         `json:"triangle_count"`
		Bounds                  [][]float64 `json:"bounds_model_units"`
		UVs                     bool        `json:"uv_texture_coordinates"`
		MetricScale             string      `json:"metric_scale"`
		Watertight              bool        `json:"watertight"`
		WindingConsistent       bool        `json:"winding_consistent"`
		ConnectedComponents     int         `json:"connected_components"`
		ClosedComponentCount    int         `json:"closed_component_count"`
		OpenComponentCount      int         `json:"open_component_count"`
		ClosedComponentsOutward bool        `json:"closed_components_outward"`
		Volume                  *float64    `json:"volume_model_units_cubed"`
	} `json:"mesh_inspection"`
}

// Validate checks the bounded v1 ZIP members, GLB framing, provenance digests,
// disclosure fields, and mesh budgets. It does not extract or render the mesh.
func Validate(data []byte) error {
	if len(data) < 4 || len(data) > MaxArchiveBytes || !bytes.Equal(data[:4], []byte{'P', 'K', 3, 4}) {
		return fmt.Errorf("image_to_object output is not a bounded ZIP archive")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("image_to_object ZIP is invalid: %w", err)
	}
	if len(reader.File) != 2 {
		return fmt.Errorf("image_to_object ZIP must contain exactly model.glb and manifest.json")
	}
	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		if file.Name != "model.glb" && file.Name != "manifest.json" {
			return fmt.Errorf("image_to_object ZIP contains unexpected member %q", file.Name)
		}
		if _, duplicate := files[file.Name]; duplicate {
			return fmt.Errorf("image_to_object ZIP contains duplicate member %q", file.Name)
		}
		if file.Method != zip.Store && file.Method != zip.Deflate {
			return fmt.Errorf("image_to_object ZIP member %q uses an unsupported compression method", file.Name)
		}
		if file.UncompressedSize64 > MaxGLBBytes || file.CompressedSize64 > MaxArchiveBytes {
			return fmt.Errorf("image_to_object ZIP member %q exceeds size limits", file.Name)
		}
		files[file.Name] = file
	}
	if files["model.glb"] == nil || files["manifest.json"] == nil {
		return fmt.Errorf("image_to_object ZIP must contain exactly model.glb and manifest.json")
	}
	glb, err := readMember(files["model.glb"], MaxGLBBytes)
	if err != nil {
		return fmt.Errorf("read image_to_object model.glb: %w", err)
	}
	if len(glb) < 20 || string(glb[:4]) != "glTF" || binary.LittleEndian.Uint32(glb[4:8]) != 2 || int(binary.LittleEndian.Uint32(glb[8:12])) != len(glb) {
		return fmt.Errorf("image_to_object model.glb has invalid GLB 2 framing")
	}
	glbFacts, err := inspectGLB(glb)
	if err != nil {
		return fmt.Errorf("inspect image_to_object GLB: %w", err)
	}
	manifestBytes, err := readMember(files["manifest.json"], 1<<20)
	if err != nil {
		return fmt.Errorf("read image_to_object manifest: %w", err)
	}
	var record manifest
	if err := json.Unmarshal(manifestBytes, &record); err != nil {
		return fmt.Errorf("decode image_to_object manifest: %w", err)
	}
	if record.SchemaVersion != 1 || record.Operation != "image_to_object" {
		return fmt.Errorf("image_to_object manifest has unsupported schema or operation")
	}
	if record.Model.ID != "triposr-vision-to-mesh" || !isSHA256(record.Model.CheckpointSHA256) || record.Model.Revision == "" || strings.TrimSpace(record.Model.Revision) != record.Model.Revision {
		return fmt.Errorf("image_to_object manifest is missing pinned checkpoint provenance")
	}
	if record.SourceCode.Commit == "" || !isSHA256(record.SourceCode.SourceProfile.TreeSHA256) {
		return fmt.Errorf("image_to_object manifest is missing pinned source-profile provenance")
	}
	if record.Runtime.Python == "" || record.Runtime.Torch == "" || (record.Runtime.Device != "cpu" && !strings.HasPrefix(record.Runtime.Device, "cuda")) {
		return fmt.Errorf("image_to_object manifest is missing runtime identity")
	}
	if strings.HasPrefix(record.Runtime.Device, "cuda") {
		if strings.TrimSpace(record.Runtime.GPUUUID) == "" || record.Runtime.WorkerPID <= 1 || record.Runtime.ProcessPeakRSSBytes == 0 || record.Runtime.CUDAPeakAllocatedBytes == nil || record.Runtime.CUDAPeakReservedBytes == nil || *record.Runtime.CUDAPeakReservedBytes < *record.Runtime.CUDAPeakAllocatedBytes {
			return fmt.Errorf("CUDA image_to_object manifest is missing reserved-device and worker memory measurements")
		}
	}
	if !isSHA256(record.Input.SHA256) || !isSHA256(record.MeshInspection.GLBSHA256) {
		return fmt.Errorf("image_to_object manifest is missing input or GLB digest")
	}
	digest := sha256.Sum256(glb)
	if record.MeshInspection.GLBSHA256 != hex.EncodeToString(digest[:]) {
		return fmt.Errorf("image_to_object manifest GLB digest does not match model.glb")
	}
	if record.MeshInspection.Vertices <= 0 || record.MeshInspection.Vertices > maxVertices || record.MeshInspection.Triangles <= 0 || record.MeshInspection.Triangles > maxTriangles {
		return fmt.Errorf("image_to_object mesh counts are outside the allowed inspection budget")
	}
	if record.MeshInspection.Vertices != glbFacts.vertices || record.MeshInspection.Triangles != glbFacts.triangles {
		return fmt.Errorf("image_to_object manifest mesh counts do not match GLB accessors")
	}
	closed, open := 0, 0
	closedOutward := false
	for _, component := range glbFacts.components {
		if component.watertight {
			closed++
			if !component.consistent || component.volume <= 0 {
				return fmt.Errorf("image_to_object GLB has a closed component without consistent outward winding")
			}
			closedOutward = true
		} else {
			open++
		}
	}
	if record.MeshInspection.ConnectedComponents != len(glbFacts.components) || record.MeshInspection.ClosedComponentCount != closed || record.MeshInspection.OpenComponentCount != open || record.MeshInspection.ClosedComponentsOutward != closedOutward {
		return fmt.Errorf("image_to_object manifest component orientation facts do not match GLB triangles")
	}
	if record.MeshInspection.Watertight != (open == 0) || record.MeshInspection.WindingConsistent != glbFacts.windingConsistent {
		return fmt.Errorf("image_to_object manifest topology facts do not match GLB triangles")
	}
	if open == 0 {
		if record.MeshInspection.Volume == nil || *record.MeshInspection.Volume <= 0 || glbFacts.volume <= 0 {
			return fmt.Errorf("image_to_object watertight mesh must have positive signed volume")
		}
		if abs(*record.MeshInspection.Volume-glbFacts.volume) > math.Max(1e-6, abs(glbFacts.volume)*1e-5) {
			return fmt.Errorf("image_to_object manifest signed volume does not match GLB triangles")
		}
	} else if record.MeshInspection.Volume != nil {
		return fmt.Errorf("image_to_object open mesh must leave signed volume unreported")
	}
	if record.MeshInspection.UVs || record.MeshInspection.MetricScale != "unavailable; model-normalized bounds are not measurements" {
		return fmt.Errorf("image_to_object manifest must disclose absent UVs and unavailable metric scale")
	}
	if len(record.MeshInspection.Bounds) != 2 || len(record.MeshInspection.Bounds[0]) != 3 || len(record.MeshInspection.Bounds[1]) != 3 {
		return fmt.Errorf("image_to_object manifest is missing model-space bounds")
	}
	for _, row := range record.MeshInspection.Bounds {
		for _, value := range row {
			if value != value {
				return fmt.Errorf("image_to_object bounds contain non-finite values")
			}
		}
	}
	for axis := 0; axis < 3; axis++ {
		if abs(record.MeshInspection.Bounds[0][axis]-glbFacts.min[axis]) > 1e-5 || abs(record.MeshInspection.Bounds[1][axis]-glbFacts.max[axis]) > 1e-5 {
			return fmt.Errorf("image_to_object manifest bounds do not match GLB positions")
		}
	}
	return nil
}

type glbMeshFacts struct {
	vertices, triangles int
	min, max            [3]float64
	volume              float64
	components          []glbComponentFacts
	windingConsistent   bool
}

type glbComponentFacts struct {
	volume     float64
	watertight bool
	consistent bool
}

type glbEdgeKey struct{ a, b uint32 }

type glbEdgeUse struct {
	face    int
	forward bool
}
type gltfDocument struct {
	Buffers []struct {
		ByteLength int `json:"byteLength"`
	} `json:"buffers"`
	Views []struct {
		Buffer     int `json:"buffer"`
		ByteOffset int `json:"byteOffset"`
		ByteLength int `json:"byteLength"`
		ByteStride int `json:"byteStride"`
	} `json:"bufferViews"`
	Accessors []struct {
		BufferView    int    `json:"bufferView"`
		ByteOffset    int    `json:"byteOffset"`
		ComponentType int    `json:"componentType"`
		Count         int    `json:"count"`
		Type          string `json:"type"`
	} `json:"accessors"`
	Meshes []struct {
		Primitives []struct {
			Attributes map[string]int `json:"attributes"`
			Indices    int            `json:"indices"`
		} `json:"primitives"`
	} `json:"meshes"`
}

func inspectGLB(data []byte) (glbMeshFacts, error) {
	var facts glbMeshFacts
	var doc gltfDocument
	var bin []byte
	for offset := 12; offset < len(data); {
		if offset+8 > len(data) {
			return facts, fmt.Errorf("truncated chunk header")
		}
		length := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8
		if length < 0 || offset+length > len(data) {
			return facts, fmt.Errorf("chunk exceeds GLB bounds")
		}
		switch kind {
		case 0x4E4F534A:
			if len(doc.Meshes) > 0 {
				return facts, fmt.Errorf("duplicate JSON chunk")
			}
			if err := json.Unmarshal(bytes.TrimRight(data[offset:offset+length], " \x00"), &doc); err != nil {
				return facts, err
			}
		case 0x004E4942:
			if bin != nil {
				return facts, fmt.Errorf("duplicate BIN chunk")
			}
			bin = data[offset : offset+length]
		default:
			return facts, fmt.Errorf("unsupported GLB chunk type")
		}
		offset += length
	}
	if len(doc.Meshes) != 1 || len(doc.Meshes[0].Primitives) != 1 || len(doc.Buffers) != 1 || bin == nil || doc.Buffers[0].ByteLength > len(bin) {
		return facts, fmt.Errorf("expected one bounded mesh, primitive, and binary buffer")
	}
	primitive := doc.Meshes[0].Primitives[0]
	positionIndex, ok := primitive.Attributes["POSITION"]
	if !ok || positionIndex < 0 || positionIndex >= len(doc.Accessors) || primitive.Indices < 0 || primitive.Indices >= len(doc.Accessors) {
		return facts, fmt.Errorf("mesh POSITION or indices accessor is missing")
	}
	position := doc.Accessors[positionIndex]
	indices := doc.Accessors[primitive.Indices]
	if position.ComponentType != 5126 || position.Type != "VEC3" || position.Count <= 0 || position.Count > maxVertices || indices.Type != "SCALAR" || indices.Count <= 0 || indices.Count%3 != 0 || indices.Count/3 > maxTriangles {
		return facts, fmt.Errorf("mesh accessors are unsupported or over budget")
	}
	facts.vertices, facts.triangles = position.Count, indices.Count/3
	for axis := range facts.min {
		facts.min[axis] = 1e300
		facts.max[axis] = -1e300
	}
	view := func(i int, accessorOffset int) (int, int, error) {
		if i < 0 || i >= len(doc.Views) {
			return 0, 0, fmt.Errorf("accessor buffer view index invalid")
		}
		v := doc.Views[i]
		start := v.ByteOffset + accessorOffset
		end := v.ByteOffset + v.ByteLength
		if v.Buffer != 0 || start < v.ByteOffset || end > len(bin) {
			return 0, 0, fmt.Errorf("buffer view exceeds GLB buffer")
		}
		return start, end, nil
	}
	start, end, err := view(position.BufferView, position.ByteOffset)
	if err != nil {
		return facts, err
	}
	stride := doc.Views[position.BufferView].ByteStride
	if stride == 0 {
		stride = 12
	}
	if stride < 12 || start+(position.Count-1)*stride+12 > end {
		return facts, fmt.Errorf("position accessor exceeds its buffer view")
	}
	meshVertices := make([][3]float64, position.Count)
	for i := 0; i < position.Count; i++ {
		for axis := 0; axis < 3; axis++ {
			v := float64(math.Float32frombits(binary.LittleEndian.Uint32(bin[start+i*stride+axis*4:])))
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return facts, fmt.Errorf("non-finite position")
			}
			if v < facts.min[axis] {
				facts.min[axis] = v
			}
			if v > facts.max[axis] {
				facts.max[axis] = v
			}
			meshVertices[i][axis] = v
		}
	}
	indexViewStart, indexViewEnd, err := view(indices.BufferView, indices.ByteOffset)
	if err != nil {
		return facts, err
	}
	width := map[int]int{5121: 1, 5123: 2, 5125: 4}[indices.ComponentType]
	if width == 0 || indexViewStart+indices.Count*width > indexViewEnd {
		return facts, fmt.Errorf("indices accessor exceeds its buffer view")
	}
	meshIndices := make([]uint32, indices.Count)
	for i := 0; i < indices.Count; i++ {
		var index uint32
		switch width {
		case 1:
			index = uint32(bin[indexViewStart+i])
		case 2:
			index = uint32(binary.LittleEndian.Uint16(bin[indexViewStart+i*2:]))
		case 4:
			index = binary.LittleEndian.Uint32(bin[indexViewStart+i*4:])
		}
		if index >= uint32(position.Count) {
			return facts, fmt.Errorf("mesh index exceeds vertex count")
		}
		meshIndices[i] = index
	}
	parent := make([]int, facts.triangles)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(value int) int {
		if parent[value] != value {
			parent[value] = find(parent[value])
		}
		return parent[value]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	edges := make(map[glbEdgeKey][]glbEdgeUse, len(meshIndices))
	for face := 0; face < facts.triangles; face++ {
		tri := meshIndices[face*3 : face*3+3]
		for edge := 0; edge < 3; edge++ {
			a, b := tri[edge], tri[(edge+1)%3]
			key := glbEdgeKey{a: a, b: b}
			forward := true
			if key.a > key.b {
				key.a, key.b = key.b, key.a
				forward = false
			}
			uses := edges[key]
			if len(uses) > 0 {
				union(face, uses[0].face)
			}
			edges[key] = append(uses, glbEdgeUse{face: face, forward: forward})
		}
	}
	watertightByRoot := make(map[int]bool, facts.triangles)
	consistentByRoot := make(map[int]bool, facts.triangles)
	volumeByRoot := make(map[int]float64, facts.triangles)
	for face := 0; face < facts.triangles; face++ {
		root := find(face)
		if _, ok := watertightByRoot[root]; !ok {
			watertightByRoot[root], consistentByRoot[root] = true, true
		}
		tri := meshIndices[face*3 : face*3+3]
		a, b, c := meshVertices[tri[0]], meshVertices[tri[1]], meshVertices[tri[2]]
		volumeByRoot[root] += dot(a, cross(b, c)) / 6.0
	}
	for _, uses := range edges {
		root := find(uses[0].face)
		if len(uses) != 2 {
			watertightByRoot[root] = false
			continue
		}
		if uses[0].forward == uses[1].forward {
			consistentByRoot[root] = false
		}
	}
	facts.windingConsistent = true
	for root, watertight := range watertightByRoot {
		component := glbComponentFacts{volume: volumeByRoot[root], watertight: watertight, consistent: consistentByRoot[root]}
		facts.components = append(facts.components, component)
		facts.volume += component.volume
		facts.windingConsistent = facts.windingConsistent && component.consistent
	}
	return facts, nil
}

func dot(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func cross(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func readMember(file *zip.File, limit uint64) ([]byte, error) {
	if file.UncompressedSize64 > limit {
		return nil, fmt.Errorf("uncompressed size %d exceeds %d", file.UncompressedSize64, limit)
	}
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	data, err := io.ReadAll(io.LimitReader(stream, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) != file.UncompressedSize64 || uint64(len(data)) > limit {
		return nil, fmt.Errorf("member size changed or exceeded its limit")
	}
	return data, nil
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
