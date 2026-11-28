package meshbundle

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"testing"
)

func fixtureArchive(t *testing.T) []byte {
	t.Helper()
	return fixtureArchiveForGLB(t, fixtureGLB(t), true)
}

func fixtureArchiveForGLB(t *testing.T, glb []byte, claimClosedOutward bool) []byte {
	t.Helper()
	glbHash := sha256.Sum256(glb)
	facts, err := inspectGLB(glb)
	if err != nil {
		t.Fatal(err)
	}
	closed, open := 0, 0
	for _, component := range facts.components {
		if component.watertight {
			closed++
		} else {
			open++
		}
	}
	manifest := map[string]any{
		"schema_version": 1,
		"operation":      "image_to_object",
		"model": map[string]any{
			"id": "triposr-vision-to-mesh", "revision": "5b521936b01fbe1890f6f9baed0254ab6351c04a", "checkpoint_sha256": hex.EncodeToString(make([]byte, 32)),
		},
		"source_code": map[string]any{
			"commit":         "107cefdc244c39106fa830359024f6a2f1c78871",
			"source_profile": map[string]any{"tree_sha256": hex.EncodeToString(make([]byte, 32))},
		},
		"runtime": map[string]any{
			"python": "3.12.3", "torch": "2.12.1+cu130", "device": "cuda:0", "gpu_uuid": "GPU-test",
			"worker_pid": 1234, "process_peak_rss_bytes": 1024, "cuda_peak_allocated_bytes": 512, "cuda_peak_reserved_bytes": 1024,
		},
		"input": map[string]any{"sha256": hex.EncodeToString(make([]byte, 32))},
		"mesh_inspection": map[string]any{
			"glb_sha256": hex.EncodeToString(glbHash[:]), "vertex_count": facts.vertices, "triangle_count": facts.triangles,
			"bounds_model_units": [][]float64{{facts.min[0], facts.min[1], facts.min[2]}, {facts.max[0], facts.max[1], facts.max[2]}}, "uv_texture_coordinates": false,
			"metric_scale": "unavailable; model-normalized bounds are not measurements",
			"watertight":   open == 0, "winding_consistent": facts.windingConsistent,
			"volume_model_units_cubed": func() any {
				if open == 0 {
					return facts.volume
				}
				return nil
			}(),
			"connected_components": len(facts.components), "closed_component_count": closed, "open_component_count": open,
			"closed_components_outward": claimClosedOutward,
		},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for name, content := range map[string][]byte{"model.glb": glb, "manifest.json": manifestBytes} {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func fixtureGLB(t *testing.T) []byte {
	t.Helper()
	positions := make([]byte, 48)
	for i, v := range []float32{0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1} {
		binary.LittleEndian.PutUint32(positions[i*4:], math.Float32bits(v))
	}
	bin := append(positions, []byte{0, 0, 2, 0, 1, 0, 0, 0, 1, 0, 3, 0, 0, 0, 3, 0, 2, 0, 1, 0, 2, 0, 3, 0}...)
	doc := map[string]any{"asset": map[string]any{"version": "2.0"}, "buffers": []any{map[string]any{"byteLength": 72}}, "bufferViews": []any{map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": 48}, map[string]any{"buffer": 0, "byteOffset": 48, "byteLength": 24}}, "accessors": []any{map[string]any{"bufferView": 0, "componentType": 5126, "count": 4, "type": "VEC3"}, map[string]any{"bufferView": 1, "componentType": 5123, "count": 12, "type": "SCALAR"}}, "meshes": []any{map[string]any{"primitives": []any{map[string]any{"attributes": map[string]any{"POSITION": 0}, "indices": 1}}}}}
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	for len(jsonBytes)%4 != 0 {
		jsonBytes = append(jsonBytes, ' ')
	}
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	out := make([]byte, 12)
	copy(out, "glTF")
	binary.LittleEndian.PutUint32(out[4:], 2)
	binary.LittleEndian.PutUint32(out[8:], uint32(12+8+len(jsonBytes)+8+len(bin)))
	chunk := make([]byte, 8)
	binary.LittleEndian.PutUint32(chunk, uint32(len(jsonBytes)))
	binary.LittleEndian.PutUint32(chunk[4:], 0x4E4F534A)
	out = append(out, chunk...)
	out = append(out, jsonBytes...)
	chunk = make([]byte, 8)
	binary.LittleEndian.PutUint32(chunk, uint32(len(bin)))
	binary.LittleEndian.PutUint32(chunk[4:], 0x004E4942)
	out = append(out, chunk...)
	out = append(out, bin...)
	return out
}

func TestValidateAcceptsTheBoundedPublicEnvelope(t *testing.T) {
	if err := Validate(fixtureArchive(t)); err != nil {
		t.Fatalf("valid image-to-object bundle rejected: %v", err)
	}
}

func TestValidateRejectsInwardClosedMeshManifest(t *testing.T) {
	data := fixtureArchive(t)
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, file := range reader.File {
		body, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content := new(bytes.Buffer)
		_, _ = content.ReadFrom(body)
		_ = body.Close()
		if file.Name == "manifest.json" {
			var manifest map[string]any
			if err := json.Unmarshal(content.Bytes(), &manifest); err != nil {
				t.Fatal(err)
			}
			inspection := manifest["mesh_inspection"].(map[string]any)
			inspection["volume_model_units_cubed"] = -1.0 / 6.0
			content.Reset()
			if err := json.NewEncoder(content).Encode(manifest); err != nil {
				t.Fatal(err)
			}
		}
		member, err := writer.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(content.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Validate(out.Bytes()); err == nil {
		t.Fatal("inward-facing closed mesh manifest was accepted")
	}
}

func TestValidateRejectsGPUArchiveWithoutMeasurements(t *testing.T) {
	data := fixtureArchive(t)
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, file := range reader.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content := new(bytes.Buffer)
		_, _ = content.ReadFrom(stream)
		_ = stream.Close()
		if file.Name == "manifest.json" {
			var manifest map[string]any
			if err := json.Unmarshal(content.Bytes(), &manifest); err != nil {
				t.Fatal(err)
			}
			delete(manifest["runtime"].(map[string]any), "gpu_uuid")
			content.Reset()
			if err := json.NewEncoder(content).Encode(manifest); err != nil {
				t.Fatal(err)
			}
		}
		member, err := writer.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(content.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Validate(out.Bytes()); err == nil {
		t.Fatal("CUDA artifact without reserved-device identity was accepted")
	}
}

func TestInspectGLBReportsEachClosedComponentOrientation(t *testing.T) {
	facts, err := inspectGLB(fixtureDoubleTetraGLB(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.components) != 2 {
		t.Fatalf("component count = %d, want 2", len(facts.components))
	}
	positive, negative := 0, 0
	for _, component := range facts.components {
		if !component.watertight || !component.consistent {
			t.Fatalf("component topology = %+v, want closed and locally consistent", component)
		}
		if component.volume > 0 {
			positive++
		} else if component.volume < 0 {
			negative++
		}
	}
	if positive != 1 || negative != 1 {
		t.Fatalf("outward/inward component counts = %d/%d, want 1/1", positive, negative)
	}
	if facts.volume <= 0 {
		t.Fatalf("combined signed volume = %f, want positive despite the smaller inward island", facts.volume)
	}
	if err := Validate(fixtureArchiveForGLB(t, fixtureDoubleTetraGLB(t, true), true)); err == nil {
		t.Fatal("mesh with a locally consistent inward closed component was accepted")
	}
}

func fixtureDoubleTetraGLB(t *testing.T, reverseSecond bool) []byte {
	t.Helper()
	points := [][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {3, 0, 0}, {3.5, 0, 0}, {3, 0.5, 0}, {3, 0, 0.5}}
	positions := make([]byte, len(points)*12)
	for i, point := range points {
		for axis, value := range point {
			binary.LittleEndian.PutUint32(positions[i*12+axis*4:], math.Float32bits(value))
		}
	}
	faces := [][3]uint16{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}}
	indices := make([]byte, 0, len(faces)*2*3*2)
	for component := 0; component < 2; component++ {
		for _, face := range faces {
			if component == 1 {
				for i := range face {
					face[i] += 4
				}
				if reverseSecond {
					face[1], face[2] = face[2], face[1]
				}
			}
			for _, index := range face {
				encoded := make([]byte, 2)
				binary.LittleEndian.PutUint16(encoded, index)
				indices = append(indices, encoded...)
			}
		}
	}
	bin := append(positions, indices...)
	doc := map[string]any{
		"asset":   map[string]any{"version": "2.0"},
		"buffers": []any{map[string]any{"byteLength": len(bin)}},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": len(positions)},
			map[string]any{"buffer": 0, "byteOffset": len(positions), "byteLength": len(indices)},
		},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": len(points), "type": "VEC3"},
			map[string]any{"bufferView": 1, "componentType": 5123, "count": len(indices) / 2, "type": "SCALAR"},
		},
		"meshes": []any{map[string]any{"primitives": []any{map[string]any{"attributes": map[string]any{"POSITION": 0}, "indices": 1}}}},
	}
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	for len(jsonBytes)%4 != 0 {
		jsonBytes = append(jsonBytes, ' ')
	}
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	out := make([]byte, 12)
	copy(out, "glTF")
	binary.LittleEndian.PutUint32(out[4:], 2)
	binary.LittleEndian.PutUint32(out[8:], uint32(12+8+len(jsonBytes)+8+len(bin)))
	chunk := make([]byte, 8)
	binary.LittleEndian.PutUint32(chunk, uint32(len(jsonBytes)))
	binary.LittleEndian.PutUint32(chunk[4:], 0x4E4F534A)
	out = append(out, chunk...)
	out = append(out, jsonBytes...)
	chunk = make([]byte, 8)
	binary.LittleEndian.PutUint32(chunk, uint32(len(bin)))
	binary.LittleEndian.PutUint32(chunk[4:], 0x004E4942)
	out = append(out, chunk...)
	return append(out, bin...)
}

func TestValidateRejectsUnexpectedMembers(t *testing.T) {
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for name, body := range map[string][]byte{"model.glb": make([]byte, 20), "manifest.json": []byte(`{"schema_version":1,"operation":"image_to_object"}`), "extra.npy": nil} {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = member.Write(body)
	}
	_ = writer.Close()
	if err := Validate(out.Bytes()); err == nil {
		t.Fatal("unexpected ZIP member was accepted")
	}
}
