package rendering

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"

	"device-studio/internal/scenes"
)

func TestInspectPhysicalGLBValidatesBoundedGeometrySubset(t *testing.T) {
	data := physicalFixtureGLB(t)
	inventory, err := InspectPhysicalGLB(data)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Version != 2 || inventory.SHA256 != sha256Bytes(data) || len(inventory.Nodes) != 1 || len(inventory.Meshes) != 1 || inventory.VertexCount != 3 || inventory.TriangleCount != 1 {
		t.Fatalf("unexpected inventory: %+v", inventory)
	}
	primitive := inventory.Meshes[0].Primitives[0]
	if primitive.IndexCount != 3 || primitive.UV0 == nil || primitive.Normal == nil || primitive.Material != 0 {
		t.Fatalf("primitive facts were not retained: %+v", primitive)
	}
}

func TestInspectPhysicalGLBPreservesStandardNormalTextureScale(t *testing.T) {
	for _, tc := range []struct {
		name  string
		scale any
		want  float64
	}{
		{name: "omitted default", want: 1},
		{name: "explicit zero", scale: float64(0), want: 0},
		{name: "source scale", scale: float64(.1), want: .1},
		{name: "negative finite standard value", scale: float64(-2), want: -2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := physicalFixtureWithNormalScale(t, tc.scale)
			inventory, err := InspectPhysicalGLB(data)
			if err != nil {
				t.Fatalf("inspect supported normal texture scale: %v", err)
			}
			if got := inventory.Materials[0].NormalScale; got != tc.want {
				t.Fatalf("effective normal scale = %v, want %v", got, tc.want)
			}
		})
	}
	if _, err := InspectPhysicalGLB(physicalFixtureWithNormalScale(t, json.Number("1e999"))); err == nil {
		t.Fatal("out-of-range non-finite normal texture scale was accepted")
	}
}

func TestInspectPhysicalGLBValidatesStandardClearcoatAndIOR(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ior       float64
		coat      float64
		wantError bool
	}{
		{name: "standard clearcoat and ior", ior: 1.45, coat: .6},
		{name: "ior zero sentinel", ior: 0, coat: 1},
		{name: "subunit nonzero ior", ior: .9, coat: .2, wantError: true},
		{name: "clearcoat out of range", ior: 1.5, coat: 1.1, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := physicalFixtureDocument()
			doc["extensionsUsed"] = []any{"KHR_materials_ior", "KHR_materials_clearcoat"}
			material := doc["materials"].([]any)[0].(map[string]any)
			material["extensions"] = map[string]any{
				"KHR_materials_ior":       map[string]any{"ior": tc.ior},
				"KHR_materials_clearcoat": map[string]any{"clearcoatFactor": tc.coat, "clearcoatRoughnessFactor": .25},
			}
			data := encodePhysicalGLB(t, doc)
			inventory, err := InspectPhysicalGLB(data)
			if tc.wantError {
				if err == nil {
					t.Fatal("malformed standard material extension was accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := inventory.Materials[0]
			if got.IOR != tc.ior || got.Clearcoat != tc.coat || got.ClearcoatRoughness != .25 {
				t.Fatalf("standard extension values were not retained: %+v", got)
			}
		})
	}
}

func physicalFixtureWithNormalScale(t *testing.T, scale any) []byte {
	t.Helper()
	doc := physicalFixtureDocument()
	material := doc["materials"].([]any)[0].(map[string]any)
	textureInfo := map[string]any{"index": 0}
	if scale != nil {
		textureInfo["scale"] = scale
	}
	material["normalTexture"] = textureInfo
	doc["textures"] = []any{map[string]any{"source": 0}}
	imageBytes := bytes.Buffer{}
	pixel := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pixel.Set(0, 0, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	if err := png.Encode(&imageBytes, pixel); err != nil {
		t.Fatal(err)
	}
	chunks, err := splitGLBChunks(physicalFixtureGLB(t))
	if err != nil {
		t.Fatal(err)
	}
	doc["images"] = []any{map[string]any{"bufferView": 4, "mimeType": "image/png"}}
	views := doc["bufferViews"].([]any)
	views = append(views, map[string]any{"buffer": 0, "byteOffset": len(chunks[1]), "byteLength": imageBytes.Len()})
	doc["bufferViews"] = views
	buffer := doc["buffers"].([]any)[0].(map[string]any)
	buffer["byteLength"] = len(chunks[1]) + imageBytes.Len()
	bin := append(append([]byte(nil), chunks[1]...), imageBytes.Bytes()...)
	return encodePhysicalGLBWithBinary(t, doc, bin)
}

func TestNormalizePhysicalGLBNamesPreservesGeometryAndUsesStableAliases(t *testing.T) {
	data := physicalFixtureGLB(t)
	prepared, source, err := NormalizePhysicalGLBNames(data, "device")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(prepared, data) {
		t.Fatal("derived GLB did not receive stable importer aliases")
	}
	got, err := InspectPhysicalGLB(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if got.Nodes[0].Name != "DS_device_NODE_00000" || got.Meshes[0].Name != "DS_device_MESH_00000" || got.Materials[0].Name != "DS_device_MATERIAL_00000" || got.SHA256 == source.SHA256 {
		t.Fatalf("derived identity is not uniquely pinned: %+v", got)
	}
	if !samePhysicalGLBStructure(source, got, "device") {
		t.Fatal("derived aliases changed source geometry, transforms, materials, or primitive bindings")
	}
	if _, _, err := NormalizePhysicalGLBNames(data, "arbitrary"); err == nil {
		t.Fatal("caller-controlled alias namespace was accepted")
	}
}

func TestInspectPhysicalGLBAcceptsAndPreservesBoundedAssetExtras(t *testing.T) {
	doc := physicalFixtureDocument()
	asset := doc["asset"].(map[string]any)
	asset["extras"] = map[string]any{
		"author":  "fixture author",
		"license": "CC-BY-4.0",
		"source":  "https://example.invalid/model",
	}
	data := encodePhysicalGLB(t, doc)
	if _, err := InspectPhysicalGLB(data); err != nil {
		t.Fatalf("standard asset-level extras must not reject otherwise supported geometry: %v", err)
	}
	prepared, _, err := NormalizePhysicalGLBNames(data, "device")
	if err != nil {
		t.Fatalf("normalize asset with extras: %v", err)
	}
	chunks, err := splitGLBChunks(prepared)
	if err != nil {
		t.Fatal(err)
	}
	var normalized glbDocument
	if err := json.Unmarshal(chunks[0], &normalized); err != nil {
		t.Fatal(err)
	}
	originalExtras := asset["extras"]
	var normalizedExtras any
	if err := json.Unmarshal(normalized.Asset.Extras, &normalizedExtras); err != nil {
		t.Fatal(err)
	}
	originalJSON, _ := json.Marshal(originalExtras)
	normalizedJSON, _ := json.Marshal(normalizedExtras)
	if !bytes.Equal(originalJSON, normalizedJSON) {
		t.Fatalf("alias normalization changed asset extras: got %s want %s", normalizedJSON, originalJSON)
	}
}

func TestInspectPhysicalGLBStillRejectsUnknownAssetStructure(t *testing.T) {
	doc := physicalFixtureDocument()
	doc["asset"].(map[string]any)["arbitraryExecutableMetadata"] = map[string]any{"script": "must not be interpreted"}
	if _, err := InspectPhysicalGLB(encodePhysicalGLB(t, doc)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown structural metadata field was accepted: %v", err)
	}
}

func TestResolvePhysicalScreenGeometryDerivesOnlyUniqueFullRectPrimitive(t *testing.T) {
	data := physicalFixtureGLB(t)
	manifest := validPhysicalManifest(t)
	candidate, err := decodeDevice(manifest.Device.CanonicalJSON)
	if err != nil {
		t.Fatal(err)
	}
	candidate.Geometry.SizeBytes = int64(len(data))
	candidate.Geometry.SHA256 = sha256Bytes(data)
	mapping, err := ResolvePhysicalScreenGeometry(data, candidate, "front")
	if err != nil {
		t.Fatal(err)
	}
	if mapping.PartID != "body" || mapping.NodeIndex != 0 || mapping.MeshIndex != 0 || mapping.PrimitiveIndex != 0 || mapping.MaterialIndex != 0 || mapping.BlenderUVLayer != PhysicalBlenderUV0LayerName || mapping.UVMin != [2]float64{0, 0} || mapping.UVMax != [2]float64{1, 1} {
		t.Fatalf("screen geometry was not derived from the pinned display node: %+v", mapping)
	}
	doc := physicalFixtureDocument()
	mesh := doc["meshes"].([]any)[0].(map[string]any)
	mesh["primitives"] = append(mesh["primitives"].([]any), mesh["primitives"].([]any)[0])
	ambiguous := encodePhysicalGLB(t, doc)
	candidate.Geometry.SizeBytes = int64(len(ambiguous))
	candidate.Geometry.SHA256 = sha256Bytes(ambiguous)
	if _, err := ResolvePhysicalScreenGeometry(ambiguous, candidate, "front"); err == nil || !strings.Contains(err.Error(), "multiple full-rectangle") {
		t.Fatalf("ambiguous screen primitives were not rejected: %v", err)
	}
	primitiveIndex, materialIndex := uint32(0), uint32(0)
	candidate.DisplayBindings[0].ScreenPrimitiveIndex = &primitiveIndex
	candidate.DisplayBindings[0].ScreenMaterialIndex = &materialIndex
	mapping, err = ResolvePhysicalScreenGeometry(ambiguous, candidate, "front")
	if err != nil || mapping.PrimitiveIndex != 0 || mapping.MaterialIndex != 0 {
		t.Fatalf("explicit primitive/material zeroes did not select one split display surface: mapping=%+v err=%v", mapping, err)
	}
	wrongMaterial := uint32(1)
	candidate.DisplayBindings[0].ScreenMaterialIndex = &wrongMaterial
	if _, err := ResolvePhysicalScreenGeometry(ambiguous, candidate, "front"); err == nil || !strings.Contains(err.Error(), "primitive/material pair") {
		t.Fatalf("mismatched pinned primitive/material pair was accepted: %v", err)
	}
	candidate.DisplayBindings[0].ScreenMaterialIndex = nil
	if _, err := ResolvePhysicalScreenGeometry(ambiguous, candidate, "front"); err == nil || !strings.Contains(err.Error(), "complete screen primitive/material pair") {
		t.Fatalf("one-sided pinned screen selector was accepted: %v", err)
	}
}

func TestPhysicalScreenUVBoundsUseOnlyIndexedVertices(t *testing.T) {
	doc := physicalFixtureDocument()
	accessors := doc["accessors"].([]any)
	for _, index := range []int{0, 1, 2} {
		accessors[index].(map[string]any)["count"] = 4
	}
	views := doc["bufferViews"].([]any)
	views[0] = map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": 48}
	views[1] = map[string]any{"buffer": 0, "byteOffset": 48, "byteLength": 48}
	views[2] = map[string]any{"buffer": 0, "byteOffset": 96, "byteLength": 32}
	views[3] = map[string]any{"buffer": 0, "byteOffset": 128, "byteLength": 6}
	doc["buffers"].([]any)[0].(map[string]any)["byteLength"] = 134
	bin := make([]byte, 136)
	write := func(offset int, values ...float32) {
		for _, value := range values {
			binary.LittleEndian.PutUint32(bin[offset:offset+4], math.Float32bits(value))
			offset += 4
		}
	}
	write(0, 0, 0, 0, 1, 0, 0, 0, 1, 0, .5, .5, 0)
	write(48, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1)
	write(96, .1, .1, .9, .1, .1, .9, 0, 1)
	binary.LittleEndian.PutUint16(bin[128:130], 0)
	binary.LittleEndian.PutUint16(bin[130:132], 1)
	binary.LittleEndian.PutUint16(bin[132:134], 2)
	data := encodePhysicalGLBWithBinary(t, doc, bin)
	inventory, err := InspectPhysicalGLB(data)
	if err != nil {
		t.Fatal(err)
	}
	primitive := inventory.Meshes[0].Primitives[0]
	if math.Abs(primitive.UVMin[0]-.1) > 1e-6 || math.Abs(primitive.UVMin[1]-.1) > 1e-6 || math.Abs(primitive.UVMax[0]-.9) > 1e-6 || math.Abs(primitive.UVMax[1]-.9) > 1e-6 {
		t.Fatalf("unreferenced UV extrema widened primitive bounds: min=%v max=%v", primitive.UVMin, primitive.UVMax)
	}
	manifest := validPhysicalManifest(t)
	candidate, err := decodeDevice(manifest.Device.CanonicalJSON)
	if err != nil {
		t.Fatal(err)
	}
	candidate.Geometry.SizeBytes, candidate.Geometry.SHA256 = int64(len(data)), sha256Bytes(data)
	if _, err := ResolvePhysicalScreenGeometry(data, candidate, "front"); err == nil || !strings.Contains(err.Error(), "no supported UV0 full-rectangle") {
		t.Fatalf("unreferenced unit-rectangle UV values incorrectly qualified the screen: %v", err)
	}
}

func TestValidateScenePhysicalGLBPreflightsLogicalUVSetAgainstPinnedPrimitive(t *testing.T) {
	makeCandidate := func(data []byte, uvSet string) scenes.SceneCandidate {
		manifest := validPhysicalManifest(t)
		candidate, err := decodeScene(manifest.Scene.CanonicalJSON)
		if err != nil {
			t.Fatal(err)
		}
		candidate.Artifacts[0].Ref.SizeBytes = int64(len(data))
		candidate.Artifacts[0].Ref.SHA256 = sha256Bytes(data)
		candidate.Parts[0].NodeName = "Body"
		candidate.Materials = []scenes.SceneMaterial{{
			ID: "surface", SurfaceProfile: scenes.MaterialPrincipledV1,
			BaseColorLinearRGBA: [4]float64{0.5, 0.5, 0.5, 1}, AlphaMode: scenes.AlphaOpaque,
			Inputs: []scenes.SceneMaterialInput{{Socket: scenes.SocketBaseColor, TextureBindingID: "texture", Semantic: "COLOR", ColorSpace: scenes.ColorSpaceSRGB, Channel: scenes.ChannelRGB, Sampler: scenes.SamplerRepeat, Coordinates: scenes.CoordinatesUVSet, UVSet: uvSet, AffineUV: &[6]float64{1, 0, 0, 0, 1, 0}}},
		}}
		candidate.MaterialSlots = []scenes.SceneMaterialSlot{{PartID: "table", PrimitiveIndex: 0, SlotIndex: 0, MaterialID: "surface"}}
		return candidate
	}

	withUV0 := physicalFixtureGLB(t)
	if _, err := ValidateScenePhysicalGLB(withUV0, makeCandidate(withUV0, "TEXCOORD_0")); err != nil {
		t.Fatalf("valid logical TEXCOORD_0 binding was rejected: %v", err)
	}

	wrongSet := makeCandidate(withUV0, "TEXCOORD_1")
	if _, err := ValidateScenePhysicalGLB(withUV0, wrongSet); err == nil || !strings.Contains(err.Error(), "outside the supported imported GLB subset") {
		t.Fatalf("unsupported logical UV attribute was not rejected before worker execution: %v", err)
	}

	doc := physicalFixtureDocument()
	attrs := doc["meshes"].([]any)[0].(map[string]any)["primitives"].([]any)[0].(map[string]any)["attributes"].(map[string]any)
	delete(attrs, "TEXCOORD_0")
	withoutUV := encodePhysicalGLB(t, doc)
	if _, err := ValidateScenePhysicalGLB(withoutUV, makeCandidate(withoutUV, "TEXCOORD_0")); err == nil || !strings.Contains(err.Error(), "requires TEXCOORD_0") {
		t.Fatalf("missing source UV attribute was not rejected before worker execution: %v", err)
	}
}

func TestInspectPhysicalGLBRejectsUnsupportedOrMalformedInputs(t *testing.T) {
	tests := []struct {
		name string
		edit func(map[string]any)
	}{
		{"unknown root extension", func(doc map[string]any) { doc["extensionsUsed"] = []any{"EXT_external_scene"} }},
		{"external buffer", func(doc map[string]any) { doc["buffers"].([]any)[0].(map[string]any)["uri"] = "file:///tmp/mesh.bin" }},
		{"matrix node", func(doc map[string]any) {
			doc["nodes"].([]any)[0].(map[string]any)["matrix"] = []any{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
		}},
		{"unsupported attribute", func(doc map[string]any) {
			doc["meshes"].([]any)[0].(map[string]any)["primitives"].([]any)[0].(map[string]any)["attributes"].(map[string]any)["JOINTS_0"] = 0
		}},
		{"morph target", func(doc map[string]any) {
			doc["meshes"].([]any)[0].(map[string]any)["primitives"].([]any)[0].(map[string]any)["targets"] = []any{map[string]any{}}
		}},
		{"invalid extension object", func(doc map[string]any) {
			doc["extensions"] = map[string]any{"KHR_materials_specular": "not-an-object"}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := physicalFixtureDocument()
			tt.edit(doc)
			if _, err := InspectPhysicalGLB(encodePhysicalGLB(t, doc)); err == nil {
				t.Fatal("invalid GLB was accepted")
			}
		})
	}
}

func TestInspectPhysicalGLBRejectsTruncationAndOversizedDeclaredChunks(t *testing.T) {
	data := physicalFixtureGLB(t)
	for _, invalid := range [][]byte{data[:len(data)-1], append([]byte(nil), data...)} {
		if len(invalid) > 12 {
			binary.LittleEndian.PutUint32(invalid[12:16], math.MaxUint32)
		}
		if _, err := InspectPhysicalGLB(invalid); err == nil {
			t.Fatal("truncated or oversized chunk was accepted")
		}
	}
}

func physicalFixtureGLB(t *testing.T) []byte {
	t.Helper()
	return encodePhysicalGLB(t, physicalFixtureDocument())
}

func physicalFixtureDocument() map[string]any {
	return map[string]any{
		"asset":  map[string]any{"version": "2.0"},
		"scene":  0,
		"scenes": []any{map[string]any{"nodes": []any{0}}},
		"nodes":  []any{map[string]any{"name": "Body", "mesh": 0}},
		"meshes": []any{map[string]any{"name": "BodyMesh", "primitives": []any{map[string]any{
			"attributes": map[string]any{"POSITION": 0, "NORMAL": 1, "TEXCOORD_0": 2},
			"indices":    3,
			"material":   0,
			"mode":       4,
		}}}},
		"materials": []any{map[string]any{"name": "Mat", "pbrMetallicRoughness": map[string]any{}}},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3", "min": []any{0, 0, 0}, "max": []any{1, 1, 0}},
			map[string]any{"bufferView": 1, "componentType": 5126, "count": 3, "type": "VEC3"},
			map[string]any{"bufferView": 2, "componentType": 5126, "count": 3, "type": "VEC2"},
			map[string]any{"bufferView": 3, "componentType": 5123, "count": 3, "type": "SCALAR"},
		},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": 0, "byteLength": 36},
			map[string]any{"buffer": 0, "byteOffset": 36, "byteLength": 36},
			map[string]any{"buffer": 0, "byteOffset": 72, "byteLength": 24},
			map[string]any{"buffer": 0, "byteOffset": 96, "byteLength": 6},
		},
		"buffers": []any{map[string]any{"byteLength": 102}},
	}
}

func encodePhysicalGLB(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	bin := make([]byte, 104)
	putFloat := func(offset int, values ...float32) {
		for _, value := range values {
			binary.LittleEndian.PutUint32(bin[offset:offset+4], math.Float32bits(value))
			offset += 4
		}
	}
	putFloat(0, 0, 0, 0, 1, 0, 0, 0, 1, 0)
	putFloat(36, 0, 0, 1, 0, 0, 1, 0, 0, 1)
	putFloat(72, 0, 0, 1, 0, 0, 1)
	binary.LittleEndian.PutUint16(bin[96:98], 0)
	binary.LittleEndian.PutUint16(bin[98:100], 1)
	binary.LittleEndian.PutUint16(bin[100:102], 2)
	bin = bin[:102]
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	return encodePhysicalGLBWithBinary(t, doc, bin)
}

func encodePhysicalGLBWithBinary(t *testing.T, doc map[string]any, bin []byte) []byte {
	t.Helper()
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
	var out bytes.Buffer
	out.WriteString("glTF")
	_ = binary.Write(&out, binary.LittleEndian, uint32(2))
	total := 12 + 8 + len(jsonBytes) + 8 + len(bin)
	_ = binary.Write(&out, binary.LittleEndian, uint32(total))
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(jsonBytes)))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x4e4f534a))
	out.Write(jsonBytes)
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(bin)))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x004e4942))
	out.Write(bin)
	return out.Bytes()
}
