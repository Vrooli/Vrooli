package library

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"image/png"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func vertexColorPreviewFixtureGLB() []byte {
	bin := make([]byte, 88)
	positions := [][3]float32{{-1, -1, 0}, {1, -1, 0}, {0, 1, 0}, {0, 0, 1}}
	offset := 0
	for _, p := range positions {
		for _, v := range p {
			binary.LittleEndian.PutUint32(bin[offset:offset+4], math.Float32bits(v))
			offset += 4
		}
	}
	copy(bin[offset:offset+16], []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 0, 255})
	offset += 16
	for _, index := range []uint16{0, 1, 2, 0, 3, 1, 1, 3, 2, 2, 3, 0} {
		binary.LittleEndian.PutUint16(bin[offset:offset+2], index)
		offset += 2
	}
	doc := map[string]any{
		"asset":   map[string]any{"version": "2.0"},
		"scene":   0,
		"scenes":  []any{map[string]any{"nodes": []int{0}}},
		"nodes":   []any{map[string]any{"mesh": 0}},
		"meshes":  []any{map[string]any{"primitives": []any{map[string]any{"attributes": map[string]any{"POSITION": 0, "COLOR_0": 1}, "indices": 2, "mode": 4}}}},
		"buffers": []any{map[string]any{"byteLength": len(bin)}},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": 48},
			map[string]any{"buffer": 0, "byteOffset": 48, "byteLength": 16},
			map[string]any{"buffer": 0, "byteOffset": 64, "byteLength": 24},
		},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 4, "type": "VEC3"},
			map[string]any{"bufferView": 1, "componentType": 5121, "normalized": true, "count": 4, "type": "VEC4"},
			map[string]any{"bufferView": 2, "componentType": 5123, "count": 12, "type": "SCALAR"},
		},
	}
	jsonChunk, _ := json.Marshal(doc)
	return encodePreviewTestGLB(jsonChunk, bin)
}

func encodePreviewTestGLB(jsonChunk, bin []byte) []byte {
	for len(jsonChunk)%4 != 0 {
		jsonChunk = append(jsonChunk, ' ')
	}
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	out := make([]byte, 12+8+len(jsonChunk)+8+len(bin))
	copy(out[:4], "glTF")
	binary.LittleEndian.PutUint32(out[4:8], 2)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(jsonChunk)))
	binary.LittleEndian.PutUint32(out[16:20], 0x4e4f534a)
	copy(out[20:], jsonChunk)
	position := 20 + len(jsonChunk)
	binary.LittleEndian.PutUint32(out[position:position+4], uint32(len(bin)))
	binary.LittleEndian.PutUint32(out[position+4:position+8], 0x004e4942)
	copy(out[position+8:], bin)
	return out
}

func TestIntakePreviewUsesVertexColorsAndRendersBoundedThreeViews(t *testing.T) {
	geometry, err := decodeIntakePreviewGLB(vertexColorPreviewFixtureGLB())
	require.NoError(t, err)
	require.True(t, geometry.hasColors)
	require.Len(t, geometry.vertices, 4)
	require.Len(t, geometry.triangles, 4)
	require.InDelta(t, 1, geometry.vertices[0].color[0], 1e-9)
	require.InDelta(t, 1, geometry.vertices[1].color[1], 1e-9)
	require.InDelta(t, 1, geometry.vertices[2].color[2], 1e-9)
	pngBytes, err := renderIntakePreview(geometry)
	require.NoError(t, err)
	decoded, err := png.Decode(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	require.Equal(t, intakePreviewPanel*intakePreviewViews, decoded.Bounds().Dx())
	require.Equal(t, intakePreviewHeight, decoded.Bounds().Dy())
	for _, x0 := range []int{0, intakePreviewPanel, 2 * intakePreviewPanel} {
		r, g, b, _ := decoded.At(x0+10, 10).RGBA()
		require.Less(t, maxU16(r, g, b), uint32(40000), "each axis-plane label is visible")
		colored := 0
		for y := 28; y < intakePreviewHeight; y++ {
			for x := x0 + 1; x < x0+intakePreviewPanel-1; x++ {
				r, g, b, _ := decoded.At(x, y).RGBA()
				if maxU16(r, g, b)-minU16(r, g, b) > 9000 {
					colored++
				}
			}
		}
		require.Greater(t, colored, 0, "each orthographic view should show the embedded vertex colors")
	}
}

func TestIntakePreviewRejectsAccessorOverrunAndTransformAmbiguity(t *testing.T) {
	data := vertexColorPreviewFixtureGLB()
	_, chunks, _, err := decodeMaterialGLB(data)
	require.NoError(t, err)
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(chunks[0].data, &doc))
	var views []map[string]any
	require.NoError(t, json.Unmarshal(doc["bufferViews"], &views))
	views[0]["byteLength"] = float64(4)
	doc["bufferViews"], _ = json.Marshal(views)
	encoded, _ := json.Marshal(doc)
	badBounds := encodePreviewTestGLB(encoded, chunks[1].data)
	_, err = decodeIntakePreviewGLB(badBounds)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds its bounded buffer view")

	require.NoError(t, json.Unmarshal(chunks[0].data, &doc))
	var nodes []map[string]any
	require.NoError(t, json.Unmarshal(doc["nodes"], &nodes))
	nodes[0]["translation"] = []float64{1, 0, 0}
	doc["nodes"], _ = json.Marshal(nodes)
	encoded, _ = json.Marshal(doc)
	transformed := encodePreviewTestGLB(encoded, chunks[1].data)
	_, err = decodeIntakePreviewGLB(transformed)
	require.Error(t, err)
	require.Contains(t, err.Error(), "identity node transforms")

	require.NoError(t, json.Unmarshal(chunks[0].data, &doc))
	nodes = nil
	require.NoError(t, json.Unmarshal(doc["nodes"], &nodes))
	nodes = append(nodes, map[string]any{"translation": []float64{1, 0, 0}})
	nodes[0]["children"] = []int{1}
	doc["nodes"], _ = json.Marshal(nodes)
	encoded, _ = json.Marshal(doc)
	hierarchical := encodePreviewTestGLB(encoded, chunks[1].data)
	_, err = decodeIntakePreviewGLB(hierarchical)
	require.Error(t, err)
	require.Contains(t, err.Error(), "identity node transforms")
	nodes[1] = map[string]any{}
	doc["nodes"], _ = json.Marshal(nodes)
	encoded, _ = json.Marshal(doc)
	identityHierarchy := encodePreviewTestGLB(encoded, chunks[1].data)
	_, err = decodeIntakePreviewGLB(identityHierarchy)
	require.NoError(t, err, "identity-only node hierarchy does not alter projection")
}

func TestIntakePreviewDoesNotReadGLBBINPaddingPastLogicalBuffer(t *testing.T) {
	data := vertexColorPreviewFixtureGLB()
	_, chunks, _, err := decodeMaterialGLB(data)
	require.NoError(t, err)
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(chunks[0].data, &doc))
	var buffers []map[string]any
	require.NoError(t, json.Unmarshal(doc["buffers"], &buffers))
	// The fixture's BIN payload is already four-byte aligned. Declaring one
	// fewer byte leaves the final byte within the GLB chunk but outside the
	// buffer's legal accessor range.
	buffers[0]["byteLength"] = float64(87)
	doc["buffers"], _ = json.Marshal(buffers)
	encoded, _ := json.Marshal(doc)
	malformed := encodePreviewTestGLB(encoded, chunks[1].data)
	_, err = decodeIntakePreviewGLB(malformed)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds its bounded buffer view")
}

func TestIntakePreviewBindsExactImmutableCandidateRevision(t *testing.T) {
	ctx, service, _, _ := setupIntake(t, func(context.Context, []byte) (IntakeInspection, error) {
		return IntakeInspection{Profile: "preview-test", ProfileVersion: 1, Supported: false}, nil
	})
	data := vertexColorPreviewFixtureGLB()
	staged, err := service.BeginImport(ctx, intakeClaims("preview-candidate"), "preview.glb", int64(len(data)), bytes.NewReader(data))
	require.NoError(t, err)
	candidate, err := staged.Commit(ctx)
	require.NoError(t, err)
	preview, err := service.Preview(ctx, candidate.ID, candidate.Revision, candidate.RevisionSHA256)
	require.NoError(t, err)
	require.Equal(t, candidate.ID, preview.CandidateID)
	require.Equal(t, candidate.RevisionSHA256, preview.RevisionSHA256)
	require.Equal(t, candidate.OriginalSHA256, preview.GLBSHA256)
	require.True(t, preview.VertexColors)
	require.Len(t, preview.PreviewSHA256, 64)
	require.True(t, bytes.HasPrefix(preview.PNG, []byte("\x89PNG\r\n\x1a\n")))
	_, err = service.Preview(ctx, candidate.ID, candidate.Revision, strings.Repeat("0", 64))
	require.Error(t, err)
	require.True(t, errorsIsInvalidIntake(err))
}

func maxU16(a, b, c uint32) uint32 { return maxUint32(a, maxUint32(b, c)) }
func minU16(a, b, c uint32) uint32 { return minUint32(a, minUint32(b, c)) }
func maxUint32(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}
func minUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
func errorsIsInvalidIntake(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrInvalidIntake.Error())
}
