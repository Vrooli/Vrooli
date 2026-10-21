package physicalruntime

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// flipScreenUVV appends a private TEXCOORD_0 view for one explicitly selected
// glTF primitive. It leaves all source BIN bytes and unrelated accessors intact.
func flipScreenUVV(data []byte, nodeName string, primitiveIndex int) ([]byte, error) {
	if len(data) < 28 || binary.LittleEndian.Uint32(data[:4]) != 0x46546c67 || binary.LittleEndian.Uint32(data[4:8]) != 2 || int(binary.LittleEndian.Uint32(data[8:12])) != len(data) {
		return nil, errors.New("screen UV derivative requires a complete GLB 2.0")
	}
	var jsonChunk, binChunk []byte
	for offset := 12; offset+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8
		if size < 0 || offset+size > len(data) {
			return nil, errors.New("screen UV derivative found an unbounded GLB chunk")
		}
		switch kind {
		case 0x4e4f534a:
			jsonChunk = data[offset : offset+size]
		case 0x004e4942:
			binChunk = data[offset : offset+size]
		}
		offset += size
	}
	if len(jsonChunk) == 0 || len(binChunk) == 0 {
		return nil, errors.New("screen UV derivative requires JSON and BIN chunks")
	}
	var doc map[string]any
	if err := json.Unmarshal(jsonChunk, &doc); err != nil {
		return nil, fmt.Errorf("decode screen GLB JSON: %w", err)
	}
	nodes, ok := doc["nodes"].([]any)
	if !ok {
		return nil, errors.New("screen GLB has no nodes")
	}
	meshes, ok := doc["meshes"].([]any)
	if !ok {
		return nil, errors.New("screen GLB has no meshes")
	}
	accessors, ok := doc["accessors"].([]any)
	if !ok {
		return nil, errors.New("screen GLB has no accessors")
	}
	views, ok := doc["bufferViews"].([]any)
	if !ok {
		return nil, errors.New("screen GLB has no buffer views")
	}
	var selected map[string]any
	for _, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok || node["name"] != nodeName {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("screen node %q is ambiguous", nodeName)
		}
		selected = node
	}
	if selected == nil {
		return nil, fmt.Errorf("screen node %q is missing", nodeName)
	}
	meshIndex, ok := glbIndex(selected["mesh"], len(meshes))
	if !ok {
		return nil, errors.New("screen node mesh reference is invalid")
	}
	mesh, ok := meshes[meshIndex].(map[string]any)
	if !ok {
		return nil, errors.New("screen mesh is invalid")
	}
	primitives, ok := mesh["primitives"].([]any)
	if !ok || primitiveIndex < 0 || primitiveIndex >= len(primitives) {
		return nil, errors.New("screen primitive reference is invalid")
	}
	primitive, ok := primitives[primitiveIndex].(map[string]any)
	if !ok {
		return nil, errors.New("screen primitive is invalid")
	}
	attributes, ok := primitive["attributes"].(map[string]any)
	if !ok {
		return nil, errors.New("screen primitive has no attributes")
	}
	uvIndex, ok := glbIndex(attributes["TEXCOORD_0"], len(accessors))
	if !ok {
		return nil, errors.New("screen primitive has no valid TEXCOORD_0")
	}
	accessor, ok := accessors[uvIndex].(map[string]any)
	if !ok || accessor["componentType"] != float64(5126) || accessor["type"] != "VEC2" {
		return nil, errors.New("screen UV accessor must be float32 VEC2")
	}
	count, ok := glbIndex(accessor["count"], math.MaxInt32)
	viewIndex, viewOK := glbIndex(accessor["bufferView"], len(views))
	if !ok || !viewOK || count < 3 || count > len(binChunk)/8 {
		return nil, errors.New("screen UV accessor dimensions are invalid")
	}
	view, ok := views[viewIndex].(map[string]any)
	if !ok || (view["buffer"] != nil && view["buffer"] != float64(0)) {
		return nil, errors.New("screen UV accessor must use the embedded BIN buffer")
	}
	if stride, exists := view["byteStride"]; exists && stride != float64(8) {
		return nil, errors.New("screen UV accessor must be tightly packed float32 VEC2")
	}
	viewOffset, ok := glbOptionalIndex(view["byteOffset"])
	accessorOffset, accessorOffsetOK := glbOptionalIndex(accessor["byteOffset"])
	viewLength, lengthOK := glbIndex(view["byteLength"], len(binChunk))
	if !ok || !accessorOffsetOK || !lengthOK || accessorOffset+count*8 > viewLength || viewOffset+viewLength > len(binChunk) {
		return nil, errors.New("screen UV accessor exceeds its bounded buffer view")
	}
	start := viewOffset + accessorOffset
	end := start + count*8
	privateBIN := append([]byte(nil), binChunk...)
	for offset := start; offset < end; offset += 4 {
		value := math.Float32frombits(binary.LittleEndian.Uint32(binChunk[offset : offset+4]))
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || value < 0 || value > 1 {
			return nil, errors.New("screen UV values must be finite and normalized")
		}
	}
	for len(privateBIN)%4 != 0 {
		privateBIN = append(privateBIN, 0)
	}
	newViewOffset := len(privateBIN)
	privateBIN = append(privateBIN, binChunk[start:end]...)
	for offset := newViewOffset + 4; offset < len(privateBIN); offset += 8 {
		v := math.Float32frombits(binary.LittleEndian.Uint32(binChunk[start+offset-newViewOffset : start+offset-newViewOffset+4]))
		binary.LittleEndian.PutUint32(privateBIN[offset:offset+4], math.Float32bits(1-v))
	}
	newView := map[string]any{"buffer": 0, "byteOffset": newViewOffset, "byteLength": count * 8, "target": 34962}
	views = append(views, newView)
	newAccessor := make(map[string]any, len(accessor)+1)
	for key, value := range accessor {
		newAccessor[key] = value
	}
	newAccessor["bufferView"] = len(views) - 1
	delete(newAccessor, "byteOffset")
	attributes["TEXCOORD_0"] = len(accessors)
	accessors = append(accessors, newAccessor)
	doc["bufferViews"], doc["accessors"] = views, accessors
	buffers, ok := doc["buffers"].([]any)
	if !ok || len(buffers) != 1 {
		return nil, errors.New("screen GLB must have one embedded buffer")
	}
	buffer, ok := buffers[0].(map[string]any)
	if !ok {
		return nil, errors.New("screen GLB buffer is invalid")
	}
	buffer["byteLength"] = len(privateBIN)
	return encodeGLB(doc, privateBIN)
}

func glbIndex(value any, limit int) (int, bool) {
	n, ok := value.(float64)
	if !ok || n < 0 || n != math.Trunc(n) || n >= float64(limit) {
		return 0, false
	}
	return int(n), true
}

func glbOptionalIndex(value any) (int, bool) {
	if value == nil {
		return 0, true
	}
	return glbIndex(value, math.MaxInt32)
}

func encodeGLB(doc map[string]any, bin []byte) ([]byte, error) {
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	for len(jsonBytes)%4 != 0 {
		jsonBytes = append(jsonBytes, ' ')
	}
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	total := 12 + 8 + len(jsonBytes) + 8 + len(bin)
	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x46546c67))
	_ = binary.Write(&out, binary.LittleEndian, uint32(2))
	_ = binary.Write(&out, binary.LittleEndian, uint32(total))
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(jsonBytes)))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x4e4f534a))
	out.Write(jsonBytes)
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(bin)))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x004e4942))
	out.Write(bin)
	return out.Bytes(), nil
}
