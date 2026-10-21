package rendering

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"reflect"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/physical"
	"device-studio/internal/scenes"
)

const (
	PhysicalScreenMappingProfile = "device-studio-glb-uv0-full-rect-image-v1"
	PhysicalBlenderUV0LayerName  = "UVMap"
	physicalGLBMaxBytes          = 128 << 20
	physicalGLBMaxNodes          = 512
	physicalGLBMaxMeshes         = 512
	physicalGLBMaxMaterials      = 256
	physicalGLBMaxAccessors      = 4096
	physicalGLBMaxViews          = 4096
	physicalGLBMaxVertices       = 4_000_000
)

type PhysicalGLBInventory struct {
	Version       uint32                `json:"version"`
	SHA256        string                `json:"sha256"`
	SizeBytes     int                   `json:"size_bytes"`
	Nodes         []PhysicalGLBNode     `json:"nodes"`
	Meshes        []PhysicalGLBMesh     `json:"meshes"`
	Materials     []PhysicalGLBMaterial `json:"materials"`
	Extensions    []string              `json:"extensions"`
	VertexCount   uint64                `json:"vertex_count"`
	TriangleCount uint64                `json:"triangle_count"`
}

type PhysicalGLBNode struct {
	Index    uint32       `json:"index"`
	Name     string       `json:"name"`
	Parent   *uint32      `json:"parent,omitempty"`
	Mesh     *uint32      `json:"mesh,omitempty"`
	Children []uint32     `json:"children,omitempty"`
	Frame    physical.TRS `json:"frame"`
}

type PhysicalGLBMesh struct {
	Index      uint32                 `json:"index"`
	Name       string                 `json:"name"`
	Primitives []PhysicalGLBPrimitive `json:"primitives"`
}

type PhysicalGLBPrimitive struct {
	Index         uint32     `json:"index"`
	Mode          uint32     `json:"mode"`
	Material      uint32     `json:"material"`
	IndexAccessor *uint32    `json:"index_accessor,omitempty"`
	Position      uint32     `json:"position_accessor"`
	Normal        *uint32    `json:"normal_accessor,omitempty"`
	UV0           *uint32    `json:"uv0_accessor,omitempty"`
	VertexCount   uint32     `json:"vertex_count"`
	IndexCount    uint32     `json:"index_count"`
	TriangleCount uint32     `json:"triangle_count"`
	PositionMin   [3]float64 `json:"position_min"`
	PositionMax   [3]float64 `json:"position_max"`
	UVMin         [2]float64 `json:"uv_min"`
	UVMax         [2]float64 `json:"uv_max"`
}

type PhysicalGLBMaterial struct {
	Index          uint32     `json:"index"`
	Name           string     `json:"name"`
	AlphaMode      string     `json:"alpha_mode"`
	BaseColor      [4]float64 `json:"base_color_factor"`
	Metallic       float64    `json:"metallic_factor"`
	Roughness      float64    `json:"roughness_factor"`
	SpecularFactor float64    `json:"specular_factor"`
	SpecularColor  [3]float64 `json:"specular_color_factor"`
	Transmission   float64    `json:"transmission_factor"`
	HasBaseMap     bool       `json:"has_base_color_texture"`
	HasNormal      bool       `json:"has_normal_texture"`
	NormalScale    float64    `json:"normal_texture_scale"`
}

// PhysicalScreenGeometryBinding is derived from the pinned GLB and candidate;
// callers cannot choose an arbitrary primitive or material slot.
type PhysicalScreenGeometryBinding struct {
	DisplayID      string     `json:"display_id"`
	PartID         string     `json:"part_id"`
	NodeIndex      uint32     `json:"node_index"`
	NodeName       string     `json:"node_name"`
	MeshIndex      uint32     `json:"mesh_index"`
	PrimitiveIndex uint32     `json:"primitive_index"`
	MaterialIndex  uint32     `json:"material_index"`
	BlenderUVLayer string     `json:"blender_uv_layer"`
	UVMin          [2]float64 `json:"uv_min"`
	UVMax          [2]float64 `json:"uv_max"`
}

type glbDocument struct {
	Asset struct {
		Version    string          `json:"version"`
		MinVersion string          `json:"minVersion,omitempty"`
		Generator  string          `json:"generator,omitempty"`
		Copyright  string          `json:"copyright,omitempty"`
		Extras     json.RawMessage `json:"extras,omitempty"`
	} `json:"asset"`
	Scene            *uint32                    `json:"scene,omitempty"`
	Scenes           []glbScene                 `json:"scenes"`
	Nodes            []glbNode                  `json:"nodes"`
	Meshes           []glbMesh                  `json:"meshes"`
	Materials        []glbMaterial              `json:"materials"`
	Accessors        []glbAccessor              `json:"accessors"`
	BufferViews      []glbBufferView            `json:"bufferViews"`
	Buffers          []glbBuffer                `json:"buffers"`
	Images           []glbImage                 `json:"images,omitempty"`
	Textures         []glbTexture               `json:"textures,omitempty"`
	Samplers         []glbSampler               `json:"samplers,omitempty"`
	Animations       []json.RawMessage          `json:"animations,omitempty"`
	Skins            []json.RawMessage          `json:"skins,omitempty"`
	Cameras          []json.RawMessage          `json:"cameras,omitempty"`
	ExtensionsUsed   []string                   `json:"extensionsUsed,omitempty"`
	ExtensionsNeeded []string                   `json:"extensionsRequired,omitempty"`
	Extensions       map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras           json.RawMessage            `json:"extras,omitempty"`
}

type glbScene struct {
	Name   string          `json:"name,omitempty"`
	Nodes  []uint32        `json:"nodes"`
	Extras json.RawMessage `json:"extras,omitempty"`
}
type glbNode struct {
	Name        string                     `json:"name"`
	Children    []uint32                   `json:"children,omitempty"`
	Mesh        *uint32                    `json:"mesh,omitempty"`
	Translation []float64                  `json:"translation,omitempty"`
	Rotation    []float64                  `json:"rotation,omitempty"`
	Scale       []float64                  `json:"scale,omitempty"`
	Matrix      []float64                  `json:"matrix,omitempty"`
	Extensions  map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras      json.RawMessage            `json:"extras,omitempty"`
	Weights     []float64                  `json:"weights,omitempty"`
	Camera      *uint32                    `json:"camera,omitempty"`
	Skin        *uint32                    `json:"skin,omitempty"`
}
type glbMesh struct {
	Name       string                     `json:"name"`
	Primitives []glbPrimitive             `json:"primitives"`
	Weights    []float64                  `json:"weights,omitempty"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage            `json:"extras,omitempty"`
}
type glbPrimitive struct {
	Attributes map[string]uint32          `json:"attributes"`
	Indices    *uint32                    `json:"indices,omitempty"`
	Material   *uint32                    `json:"material,omitempty"`
	Mode       *uint32                    `json:"mode,omitempty"`
	Targets    []json.RawMessage          `json:"targets,omitempty"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage            `json:"extras,omitempty"`
}
type glbMaterial struct {
	Name string `json:"name"`
	PBR  struct {
		BaseColor         []float64 `json:"baseColorFactor,omitempty"`
		Metallic          *float64  `json:"metallicFactor,omitempty"`
		Roughness         *float64  `json:"roughnessFactor,omitempty"`
		MetalRoughTexture *struct {
			Index    uint32 `json:"index"`
			TexCoord uint32 `json:"texCoord,omitempty"`
		} `json:"metallicRoughnessTexture,omitempty"`
		BaseTexture *struct {
			Index    uint32 `json:"index"`
			TexCoord uint32 `json:"texCoord,omitempty"`
		} `json:"baseColorTexture,omitempty"`
	} `json:"pbrMetallicRoughness"`
	NormalTexture *struct {
		Index    uint32   `json:"index"`
		TexCoord uint32   `json:"texCoord,omitempty"`
		Scale    *float64 `json:"scale,omitempty"`
	} `json:"normalTexture,omitempty"`
	OcclusionTexture *struct {
		Index    uint32   `json:"index"`
		TexCoord uint32   `json:"texCoord,omitempty"`
		Strength *float64 `json:"strength,omitempty"`
	} `json:"occlusionTexture,omitempty"`
	EmissiveTexture *struct {
		Index    uint32 `json:"index"`
		TexCoord uint32 `json:"texCoord,omitempty"`
	} `json:"emissiveTexture,omitempty"`
	EmissiveFactor []float64                  `json:"emissiveFactor,omitempty"`
	AlphaMode      string                     `json:"alphaMode,omitempty"`
	AlphaCutoff    *float64                   `json:"alphaCutoff,omitempty"`
	DoubleSided    bool                       `json:"doubleSided,omitempty"`
	Extensions     map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras         json.RawMessage            `json:"extras,omitempty"`
}
type glbAccessor struct {
	Name          string                     `json:"name,omitempty"`
	BufferView    *uint32                    `json:"bufferView,omitempty"`
	ByteOffset    uint32                     `json:"byteOffset,omitempty"`
	ComponentType uint32                     `json:"componentType"`
	Normalized    bool                       `json:"normalized,omitempty"`
	Count         uint32                     `json:"count"`
	Type          string                     `json:"type"`
	Min           []float64                  `json:"min,omitempty"`
	Max           []float64                  `json:"max,omitempty"`
	Sparse        json.RawMessage            `json:"sparse,omitempty"`
	Extensions    map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras        json.RawMessage            `json:"extras,omitempty"`
}
type glbBufferView struct {
	Name       string                     `json:"name,omitempty"`
	Buffer     uint32                     `json:"buffer"`
	ByteOffset uint32                     `json:"byteOffset,omitempty"`
	ByteLength uint32                     `json:"byteLength"`
	ByteStride uint32                     `json:"byteStride,omitempty"`
	Target     uint32                     `json:"target,omitempty"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage            `json:"extras,omitempty"`
}
type glbBuffer struct {
	Name       string                     `json:"name,omitempty"`
	URI        string                     `json:"uri,omitempty"`
	ByteLength uint32                     `json:"byteLength"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage            `json:"extras,omitempty"`
}
type glbImage struct {
	Name       string                     `json:"name,omitempty"`
	URI        string                     `json:"uri,omitempty"`
	BufferView *uint32                    `json:"bufferView,omitempty"`
	MIMEType   string                     `json:"mimeType,omitempty"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage            `json:"extras,omitempty"`
}
type glbTexture struct {
	Name       string                     `json:"name,omitempty"`
	Sampler    *uint32                    `json:"sampler,omitempty"`
	Source     *uint32                    `json:"source,omitempty"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage            `json:"extras,omitempty"`
}
type glbSampler struct {
	Name       string                     `json:"name,omitempty"`
	MagFilter  *uint32                    `json:"magFilter,omitempty"`
	MinFilter  *uint32                    `json:"minFilter,omitempty"`
	WrapS      *uint32                    `json:"wrapS,omitempty"`
	WrapT      *uint32                    `json:"wrapT,omitempty"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage            `json:"extras,omitempty"`
}

// InspectPhysicalGLB parses only the bounded profile subset; it never opens a
// path or invokes the DCC importer. Caller separately pins the exact artifact.
func InspectPhysicalGLB(data []byte) (PhysicalGLBInventory, error) {
	if len(data) < 20 || len(data) > physicalGLBMaxBytes || string(data[:4]) != "glTF" {
		return PhysicalGLBInventory{}, errors.New("GLB length or magic is invalid")
	}
	if binary.LittleEndian.Uint32(data[4:8]) != 2 || uint64(binary.LittleEndian.Uint32(data[8:12])) != uint64(len(data)) {
		return PhysicalGLBInventory{}, errors.New("GLB header version or declared size differs")
	}
	chunks, err := splitGLBChunks(data)
	if err != nil {
		return PhysicalGLBInventory{}, err
	}
	var doc glbDocument
	decoder := json.NewDecoder(bytes.NewReader(chunks[0]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return PhysicalGLBInventory{}, fmt.Errorf("decode GLB JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return PhysicalGLBInventory{}, errors.New("GLB JSON chunk contains trailing data")
	}
	if doc.Asset.Version != "2.0" || doc.Asset.MinVersion != "" && doc.Asset.MinVersion != "2.0" || len(doc.Scenes) != 1 || doc.Scene != nil && *doc.Scene != 0 {
		return PhysicalGLBInventory{}, errors.New("GLB must use version 2.0 and one unambiguous scene")
	}
	if len(doc.Nodes) == 0 || len(doc.Nodes) > physicalGLBMaxNodes || len(doc.Meshes) > physicalGLBMaxMeshes || len(doc.Materials) == 0 || len(doc.Materials) > physicalGLBMaxMaterials || len(doc.Accessors) == 0 || len(doc.Accessors) > physicalGLBMaxAccessors || len(doc.BufferViews) > physicalGLBMaxViews {
		return PhysicalGLBInventory{}, errors.New("GLB collection count exceeds the physical profile")
	}
	if len(doc.Animations) != 0 || len(doc.Skins) != 0 || len(doc.Cameras) != 0 {
		return PhysicalGLBInventory{}, errors.New("GLB animation, skin, and camera data are unsupported in this profile")
	}
	if err := validateGLBExtensions(doc.ExtensionsUsed, doc.ExtensionsNeeded); err != nil {
		return PhysicalGLBInventory{}, err
	}
	if err := validateExtensionMap(doc.Extensions); err != nil {
		return PhysicalGLBInventory{}, err
	}
	if len(doc.Buffers) != 1 || doc.Buffers[0].URI != "" || uint64(doc.Buffers[0].ByteLength) > uint64(len(chunks[1])) || uint64(len(chunks[1]))-uint64(doc.Buffers[0].ByteLength) > 3 {
		return PhysicalGLBInventory{}, errors.New("GLB must have one bounded embedded binary buffer")
	}
	if err := validateExtensionMap(doc.Buffers[0].Extensions); err != nil {
		return PhysicalGLBInventory{}, fmt.Errorf("GLB buffer: %w", err)
	}
	for i, view := range doc.BufferViews {
		if err := validateExtensionMap(view.Extensions); err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB buffer view %d: %w", i, err)
		}
		if view.Buffer != 0 || uint64(view.ByteOffset)+uint64(view.ByteLength) > uint64(doc.Buffers[0].ByteLength) || view.ByteLength == 0 || view.ByteStride != 0 && (view.ByteStride < 4 || view.ByteStride > 252) {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB buffer view %d is invalid", i)
		}
	}
	for i, accessor := range doc.Accessors {
		if err := validateExtensionMap(accessor.Extensions); err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB accessor %d: %w", i, err)
		}
		if accessor.Count == 0 || accessor.Count > physicalGLBMaxVertices || accessor.BufferView == nil || int(*accessor.BufferView) >= len(doc.BufferViews) || len(accessor.Sparse) != 0 {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB accessor %d is missing a supported buffer view or exceeds bounds", i)
		}
		if err := validateAccessorExtent(accessor, doc.BufferViews[*accessor.BufferView]); err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB accessor %d: %w", i, err)
		}
	}
	for i, img := range doc.Images {
		if err := validateExtensionMap(img.Extensions); err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB image %d: %w", i, err)
		}
		if img.URI != "" || img.BufferView == nil || int(*img.BufferView) >= len(doc.BufferViews) || (img.MIMEType != "image/png" && img.MIMEType != "image/jpeg") {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB image %d must be embedded PNG or JPEG", i)
		}
		view := doc.BufferViews[*img.BufferView]
		start, end := int(view.ByteOffset), int(view.ByteOffset+view.ByteLength)
		config, format, err := image.DecodeConfig(bytes.NewReader(chunks[1][start:end]))
		if err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB embedded image %d is invalid: %w", i, err)
		}
		if (img.MIMEType == "image/png" && format != "png") || (img.MIMEType == "image/jpeg" && format != "jpeg") || config.Width < 1 || config.Height < 1 || config.Width > 16384 || config.Height > 16384 || uint64(config.Width)*uint64(config.Height) > 64_000_000 {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB embedded image %d format or dimensions exceed the physical profile", i)
		}
	}
	for i, texture := range doc.Textures {
		if err := validateExtensionMap(texture.Extensions); err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB texture %d: %w", i, err)
		}
		if texture.Source == nil || int(*texture.Source) >= len(doc.Images) || texture.Sampler != nil && int(*texture.Sampler) >= len(doc.Samplers) {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB texture %d has an invalid image or sampler", i)
		}
	}
	for i, sampler := range doc.Samplers {
		if err := validateExtensionMap(sampler.Extensions); err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB sampler %d: %w", i, err)
		}
		if !validSamplerEnum(sampler.MagFilter, 9728, 9729) || !validSamplerEnum(sampler.MinFilter, 9728, 9729, 9984, 9985, 9986, 9987) || !validSamplerEnum(sampler.WrapS, 33071, 33648, 10497) || !validSamplerEnum(sampler.WrapT, 33071, 33648, 10497) {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB sampler %d uses an unsupported filter or wrap mode", i)
		}
	}
	if err := validateGLBMaterials(doc.Materials, uint32(len(doc.Textures))); err != nil {
		return PhysicalGLBInventory{}, err
	}
	parents, reachable, err := validateGLBNodes(doc)
	if err != nil {
		return PhysicalGLBInventory{}, err
	}
	inventory := PhysicalGLBInventory{Version: 2, SizeBytes: len(data), Extensions: append([]string(nil), doc.ExtensionsUsed...), Nodes: make([]PhysicalGLBNode, len(doc.Nodes)), Meshes: make([]PhysicalGLBMesh, len(doc.Meshes)), Materials: make([]PhysicalGLBMaterial, len(doc.Materials))}
	fileHash := sha256Bytes(data)
	inventory.SHA256 = fileHash
	for i, node := range doc.Nodes {
		if !reachable[i] {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB node %d is not reachable from the selected scene", i)
		}
		frame, err := glbNodeFrame(node)
		if err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB node %d: %w", i, err)
		}
		inventory.Nodes[i] = PhysicalGLBNode{Index: uint32(i), Name: node.Name, Parent: parents[i], Mesh: node.Mesh, Children: append([]uint32(nil), node.Children...), Frame: frame}
		if node.Mesh != nil && int(*node.Mesh) >= len(doc.Meshes) {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB node %d references an invalid mesh", i)
		}
	}
	for i, mesh := range doc.Meshes {
		if err := validateExtensionMap(mesh.Extensions); err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB mesh %d: %w", i, err)
		}
		if len(mesh.Primitives) == 0 || len(mesh.Primitives) > 128 {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB mesh %d primitive count is outside bounds", i)
		}
		out := PhysicalGLBMesh{Index: uint32(i), Name: mesh.Name, Primitives: make([]PhysicalGLBPrimitive, 0, len(mesh.Primitives))}
		for j, primitive := range mesh.Primitives {
			item, err := inspectGLBPrimitive(uint32(j), primitive, doc)
			if err != nil {
				return PhysicalGLBInventory{}, fmt.Errorf("GLB mesh %d primitive %d: %w", i, j, err)
			}
			uvMin, uvMax, err := validatePrimitiveBufferData(item, doc, chunks[1])
			if err != nil {
				return PhysicalGLBInventory{}, fmt.Errorf("GLB mesh %d primitive %d: %w", i, j, err)
			}
			item.UVMin, item.UVMax = uvMin, uvMax
			inventory.VertexCount += uint64(item.VertexCount)
			inventory.TriangleCount += uint64(item.TriangleCount)
			if inventory.VertexCount > physicalGLBMaxVertices*8 {
				return PhysicalGLBInventory{}, errors.New("GLB aggregate vertex count exceeds bound")
			}
			out.Primitives = append(out.Primitives, item)
		}
		inventory.Meshes[i] = out
	}
	for i, material := range doc.Materials {
		alpha := material.AlphaMode
		if alpha == "" {
			alpha = "OPAQUE"
		}
		if alpha != "OPAQUE" && alpha != "MASK" && alpha != "BLEND" {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB material %d has unsupported alpha mode", i)
		}
		factors := []float64{1, 1, 1, 1}
		if len(material.PBR.BaseColor) != 0 {
			if len(material.PBR.BaseColor) != 4 {
				return PhysicalGLBInventory{}, fmt.Errorf("GLB material %d base-color factor must have four components", i)
			}
			copy(factors, material.PBR.BaseColor)
		}
		metallic, roughness := 1.0, 1.0
		if material.PBR.Metallic != nil {
			metallic = *material.PBR.Metallic
		}
		if material.PBR.Roughness != nil {
			roughness = *material.PBR.Roughness
		}
		if !unitFactors(factors) || !unitValue(metallic) || !unitValue(roughness) {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB material %d factors are outside [0,1]", i)
		}
		materialFacts, err := inspectGLBMaterialExtensions(material, uint32(len(doc.Textures)))
		if err != nil {
			return PhysicalGLBInventory{}, fmt.Errorf("GLB material %d: %w", i, err)
		}
		normalScale := 1.0
		if material.NormalTexture != nil && material.NormalTexture.Scale != nil {
			normalScale = *material.NormalTexture.Scale
		}
		inventory.Materials[i] = PhysicalGLBMaterial{Index: uint32(i), Name: material.Name, AlphaMode: alpha, BaseColor: [4]float64{factors[0], factors[1], factors[2], factors[3]}, Metallic: metallic, Roughness: roughness, SpecularFactor: materialFacts.specularFactor, SpecularColor: materialFacts.specularColor, Transmission: materialFacts.transmission, HasBaseMap: material.PBR.BaseTexture != nil, HasNormal: material.NormalTexture != nil, NormalScale: normalScale}
	}
	return inventory, nil
}

func ValidateDevicePhysicalGLB(data []byte, candidate devices.DeviceCandidate, screen PhysicalScreenInput) (PhysicalGLBInventory, error) {
	inventory, err := InspectPhysicalGLB(data)
	if err != nil {
		return PhysicalGLBInventory{}, err
	}
	if int64(len(data)) != candidate.Geometry.SizeBytes || sha256Bytes(data) != candidate.Geometry.SHA256 {
		return PhysicalGLBInventory{}, errors.New("device GLB bytes differ from the immutable geometry artifact")
	}
	nodeByIndex := make(map[uint32]PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodeByIndex[node.Index] = node
	}
	for _, part := range candidate.Parts {
		node, ok := nodeByIndex[part.NodeIndex]
		if !ok || node.Name != part.NodeName {
			return PhysicalGLBInventory{}, fmt.Errorf("device semantic part %q does not match the exact GLB node", part.ID)
		}
		if !trsNearlyEqual(node.Frame, part.LocalFrame, 1e-6) {
			return PhysicalGLBInventory{}, fmt.Errorf("device semantic part %q rest frame differs from its GLB node", part.ID)
		}
	}
	mapping, err := ResolvePhysicalScreenGeometry(data, candidate, screen.DisplayID)
	if err != nil {
		return PhysicalGLBInventory{}, err
	}
	if screen.UVSet != "TEXCOORD_0" || screen.MappingProfile != PhysicalScreenMappingProfile || mapping.BlenderUVLayer != PhysicalBlenderUV0LayerName || mapping.PrimitiveIndex != screen.PrimitiveIndex || mapping.MaterialIndex != screen.MaterialSlot {
		return PhysicalGLBInventory{}, errors.New("screen binding does not match a supported UV and material slot")
	}
	return inventory, nil
}

// ResolvePhysicalScreenGeometry selects the sole UV0 full-rectangle primitive
// on the canonical semantic display node, unless the reviewed candidate pins
// one exact primitive/material pair for a split display surface.
func ResolvePhysicalScreenGeometry(data []byte, candidate devices.DeviceCandidate, displayID string) (PhysicalScreenGeometryBinding, error) {
	var result PhysicalScreenGeometryBinding
	inventory, err := InspectPhysicalGLB(data)
	if err != nil {
		return result, err
	}
	if int64(len(data)) != candidate.Geometry.SizeBytes || sha256Bytes(data) != candidate.Geometry.SHA256 {
		return result, errors.New("device GLB bytes differ from the immutable geometry artifact")
	}
	var binding *devices.DisplayPartBinding
	for i := range candidate.DisplayBindings {
		display := &candidate.DisplayBindings[i]
		if display.DisplayID == displayID {
			if binding != nil {
				return result, errors.New("display ID has ambiguous semantic bindings")
			}
			binding = display
		}
	}
	if binding == nil || (binding.ScreenPrimitiveIndex == nil) != (binding.ScreenMaterialIndex == nil) {
		return result, errors.New("display binding must select one semantic part and an optional complete screen primitive/material pair")
	}
	partID := binding.PartID
	var part *devices.SemanticPart
	for i := range candidate.Parts {
		if candidate.Parts[i].ID == partID {
			part = &candidate.Parts[i]
			break
		}
	}
	if part == nil || int(part.NodeIndex) >= len(inventory.Nodes) {
		return result, errors.New("display binding does not select a GLB node")
	}
	node := inventory.Nodes[part.NodeIndex]
	if node.Name != part.NodeName || node.Mesh == nil || !trsNearlyEqual(node.Frame, part.LocalFrame, 1e-6) || int(*node.Mesh) >= len(inventory.Meshes) {
		return result, errors.New("display semantic part differs from its exact GLB node or rest frame")
	}
	mesh := inventory.Meshes[*node.Mesh]
	var selected *PhysicalGLBPrimitive
	if binding.ScreenPrimitiveIndex != nil {
		for i := range mesh.Primitives {
			primitive := &mesh.Primitives[i]
			if primitive.Index == *binding.ScreenPrimitiveIndex {
				selected = primitive
				break
			}
		}
		if selected == nil || selected.Material != *binding.ScreenMaterialIndex || selected.UV0 == nil || !unitRectUV(selected.UVMin, selected.UVMax) {
			return result, errors.New("pinned display primitive/material pair is absent or lacks referenced UV0 unit-rectangle geometry")
		}
	} else {
		for i := range mesh.Primitives {
			primitive := &mesh.Primitives[i]
			if primitive.UV0 != nil && unitRectUV(primitive.UVMin, primitive.UVMax) {
				if selected != nil {
					return result, errors.New("display node has multiple full-rectangle UV primitives; a narrower reviewed asset is required")
				}
				selected = primitive
			}
		}
		if selected == nil {
			return result, errors.New("display node has no supported UV0 full-rectangle primitive")
		}
	}
	return PhysicalScreenGeometryBinding{DisplayID: displayID, PartID: partID, NodeIndex: node.Index, NodeName: node.Name, MeshIndex: mesh.Index, PrimitiveIndex: selected.Index, MaterialIndex: selected.Material, BlenderUVLayer: PhysicalBlenderUV0LayerName, UVMin: selected.UVMin, UVMax: selected.UVMax}, nil
}

func ValidateScenePhysicalGLB(data []byte, candidate scenes.SceneCandidate) (PhysicalGLBInventory, error) {
	inventory, err := InspectPhysicalGLB(data)
	if err != nil {
		return PhysicalGLBInventory{}, err
	}
	for _, binding := range candidate.Artifacts {
		if binding.Ref.Role != library.RoleGeometryGLB {
			continue
		}
		if int64(len(data)) != binding.Ref.SizeBytes || sha256Bytes(data) != binding.Ref.SHA256 {
			return PhysicalGLBInventory{}, errors.New("scene GLB bytes differ from the immutable geometry artifact")
		}
		for _, part := range candidate.Parts {
			if part.ArtifactBindingID != binding.ID {
				continue
			}
			if int(part.NodeIndex) >= len(inventory.Nodes) || inventory.Nodes[part.NodeIndex].Name != part.NodeName || !trsNearlyEqual(inventory.Nodes[part.NodeIndex].Frame, scaleFrame(part.LocalFrame, binding.SourceUnit), 1e-6) {
				return PhysicalGLBInventory{}, fmt.Errorf("scene part %q does not match the exact GLB node frame", part.ID)
			}
		}
		for _, slot := range candidate.MaterialSlots {
			var part *scenes.ScenePart
			for i := range candidate.Parts {
				if candidate.Parts[i].ID == slot.PartID && candidate.Parts[i].ArtifactBindingID == binding.ID {
					part = &candidate.Parts[i]
					break
				}
			}
			if part == nil || int(part.NodeIndex) >= len(inventory.Nodes) || inventory.Nodes[part.NodeIndex].Mesh == nil || int(*inventory.Nodes[part.NodeIndex].Mesh) >= len(inventory.Meshes) {
				continue
			}
			mesh := inventory.Meshes[*inventory.Nodes[part.NodeIndex].Mesh]
			var primitive *PhysicalGLBPrimitive
			for i := range mesh.Primitives {
				if mesh.Primitives[i].Index == slot.PrimitiveIndex {
					primitive = &mesh.Primitives[i]
					break
				}
			}
			var material *scenes.SceneMaterial
			for i := range candidate.Materials {
				if candidate.Materials[i].ID == slot.MaterialID {
					material = &candidate.Materials[i]
					break
				}
			}
			if primitive == nil || material == nil {
				continue // Candidate canonicalization reports unresolved references.
			}
			for _, input := range material.Inputs {
				if input.Coordinates != scenes.CoordinatesUVSet {
					continue
				}
				if input.UVSet != "TEXCOORD_0" {
					return PhysicalGLBInventory{}, fmt.Errorf("scene material %q UV set %q is outside the supported imported GLB subset", material.ID, input.UVSet)
				}
				if primitive.UV0 == nil {
					return PhysicalGLBInventory{}, fmt.Errorf("scene material %q requires TEXCOORD_0 on part %q primitive %d", material.ID, part.ID, primitive.Index)
				}
			}
		}
		return inventory, nil
	}
	return PhysicalGLBInventory{}, errors.New("scene candidate has no geometry GLB")
}

// NormalizePhysicalGLBNames returns a derived GLB whose node, mesh and material
// names are stable index aliases. The source bytes are never changed. Blender's
// importer may suffix duplicate display names, so worker mapping uses these
// unique aliases and the receipt retains the original inventory and digest.
func NormalizePhysicalGLBNames(data []byte, prefix string) ([]byte, PhysicalGLBInventory, error) {
	if prefix != "device" && prefix != "scene" {
		return nil, PhysicalGLBInventory{}, errors.New("physical GLB alias prefix is unsupported")
	}
	inventory, err := InspectPhysicalGLB(data)
	if err != nil {
		return nil, PhysicalGLBInventory{}, err
	}
	chunks, err := splitGLBChunks(data)
	if err != nil {
		return nil, PhysicalGLBInventory{}, err
	}
	var doc glbDocument
	if err := json.Unmarshal(chunks[0], &doc); err != nil {
		return nil, PhysicalGLBInventory{}, fmt.Errorf("decode GLB names: %w", err)
	}
	for index := range doc.Nodes {
		doc.Nodes[index].Name = fmt.Sprintf("DS_%s_NODE_%05d", prefix, index)
	}
	for index := range doc.Meshes {
		doc.Meshes[index].Name = fmt.Sprintf("DS_%s_MESH_%05d", prefix, index)
	}
	for index := range doc.Materials {
		doc.Materials[index].Name = fmt.Sprintf("DS_%s_MATERIAL_%05d", prefix, index)
	}
	jsonChunk, err := json.Marshal(doc)
	if err != nil {
		return nil, PhysicalGLBInventory{}, fmt.Errorf("encode normalized GLB: %w", err)
	}
	for len(jsonChunk)%4 != 0 {
		jsonChunk = append(jsonChunk, ' ')
	}
	binChunk := chunks[1]
	total := uint64(12 + 8 + len(jsonChunk) + 8 + len(binChunk))
	if total > physicalGLBMaxBytes || total > math.MaxUint32 {
		return nil, PhysicalGLBInventory{}, errors.New("normalized GLB exceeds the physical profile size limit")
	}
	var out bytes.Buffer
	out.Grow(int(total))
	out.WriteString("glTF")
	_ = binary.Write(&out, binary.LittleEndian, uint32(2))
	_ = binary.Write(&out, binary.LittleEndian, uint32(total))
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(jsonChunk)))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x4e4f534a))
	out.Write(jsonChunk)
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(binChunk)))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x004e4942))
	out.Write(binChunk)
	prepared := out.Bytes()
	preparedInventory, err := InspectPhysicalGLB(prepared)
	if err != nil {
		return nil, PhysicalGLBInventory{}, fmt.Errorf("validate normalized GLB: %w", err)
	}
	if !samePhysicalGLBStructure(inventory, preparedInventory, prefix) {
		return nil, PhysicalGLBInventory{}, errors.New("normalized GLB changed geometry or binding structure")
	}
	return bytes.Clone(prepared), inventory, nil
}

func samePhysicalGLBStructure(source, prepared PhysicalGLBInventory, prefix string) bool {
	if len(source.Nodes) != len(prepared.Nodes) || len(source.Meshes) != len(prepared.Meshes) || len(source.Materials) != len(prepared.Materials) {
		return false
	}
	for i := range source.Nodes {
		if prepared.Nodes[i].Name != fmt.Sprintf("DS_%s_NODE_%05d", prefix, i) {
			return false
		}
		prepared.Nodes[i].Name = source.Nodes[i].Name
	}
	for i := range source.Meshes {
		if prepared.Meshes[i].Name != fmt.Sprintf("DS_%s_MESH_%05d", prefix, i) {
			return false
		}
		prepared.Meshes[i].Name = source.Meshes[i].Name
	}
	for i := range source.Materials {
		if prepared.Materials[i].Name != fmt.Sprintf("DS_%s_MATERIAL_%05d", prefix, i) {
			return false
		}
		prepared.Materials[i].Name = source.Materials[i].Name
	}
	prepared.SHA256, prepared.SizeBytes = source.SHA256, source.SizeBytes
	return reflect.DeepEqual(source, prepared)
}

func splitGLBChunks(data []byte) ([][]byte, error) {
	var chunks [][]byte
	for offset := uint64(12); offset < uint64(len(data)); {
		if offset+8 > uint64(len(data)) {
			return nil, errors.New("truncated GLB chunk header")
		}
		size := uint64(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8
		if size > uint64(len(data))-offset {
			return nil, errors.New("GLB chunk exceeds file bounds")
		}
		if len(chunks) == 0 && kind != 0x4e4f534a || len(chunks) == 1 && kind != 0x004e4942 || len(chunks) >= 2 {
			return nil, errors.New("GLB must contain one JSON chunk followed by one BIN chunk")
		}
		chunks = append(chunks, data[offset:offset+size])
		offset += size
	}
	if len(chunks) != 2 {
		return nil, errors.New("GLB must contain exactly two chunks")
	}
	return chunks, nil
}

func validateGLBExtensions(used, required []string) error {
	allowed := map[string]bool{"KHR_materials_transmission": true, "KHR_materials_specular": true}
	seen := make(map[string]bool, len(used))
	for _, ext := range used {
		if !allowed[ext] || seen[ext] {
			return fmt.Errorf("GLB extension %q is unsupported or duplicated", ext)
		}
		seen[ext] = true
	}
	for _, ext := range required {
		if !seen[ext] {
			return fmt.Errorf("required GLB extension %q is not declared as used", ext)
		}
	}
	return nil
}

func validateExtensionMap(extensions map[string]json.RawMessage) error {
	for name, value := range extensions {
		if name != "KHR_materials_transmission" && name != "KHR_materials_specular" {
			return fmt.Errorf("GLB extension %q is unsupported", name)
		}
		var fields map[string]json.RawMessage
		if len(value) == 0 || json.Unmarshal(value, &fields) != nil || fields == nil {
			return fmt.Errorf("GLB extension %q must contain an object", name)
		}
	}
	return nil
}

func validateAccessorExtent(accessor glbAccessor, view glbBufferView) error {
	components := map[string]uint64{"SCALAR": 1, "VEC2": 2, "VEC3": 3, "VEC4": 4, "MAT2": 4, "MAT3": 9, "MAT4": 16}[accessor.Type]
	componentBytes := map[uint32]uint64{5120: 1, 5121: 1, 5122: 2, 5123: 2, 5125: 4, 5126: 4}[accessor.ComponentType]
	if components == 0 || componentBytes == 0 {
		return errors.New("accessor component type or shape is unsupported")
	}
	stride := uint64(view.ByteStride)
	packed := components * componentBytes
	if stride == 0 {
		stride = packed
	}
	if stride < packed || uint64(accessor.ByteOffset)+stride*uint64(accessor.Count-1)+packed > uint64(view.ByteLength) {
		return errors.New("accessor exceeds its buffer view")
	}
	if accessor.Type == "VEC3" && accessor.ComponentType == 5126 && accessor.Min != nil {
		if len(accessor.Min) != 3 || len(accessor.Max) != 3 {
			return errors.New("position accessor bounds need three components")
		}
		for i := 0; i < 3; i++ {
			if !finiteNumber(accessor.Min[i]) || !finiteNumber(accessor.Max[i]) || accessor.Min[i] > accessor.Max[i] {
				return errors.New("position accessor bounds are invalid")
			}
		}
	}
	return nil
}

func validateGLBMaterials(materials []glbMaterial, textureCount uint32) error {
	for i, material := range materials {
		if err := validateExtensionMap(material.Extensions); err != nil {
			return fmt.Errorf("GLB material %d: %w", i, err)
		}
		if material.PBR.BaseTexture != nil && (material.PBR.BaseTexture.Index >= textureCount || material.PBR.BaseTexture.TexCoord > 0) {
			return fmt.Errorf("GLB material %d base map binding is invalid", i)
		}
		if material.NormalTexture != nil && (material.NormalTexture.Index >= textureCount || material.NormalTexture.TexCoord > 0 || material.NormalTexture.Scale != nil && !finiteNumber(*material.NormalTexture.Scale)) {
			return fmt.Errorf("GLB material %d normal map binding is invalid", i)
		}
		if material.PBR.MetalRoughTexture != nil && (material.PBR.MetalRoughTexture.Index >= textureCount || material.PBR.MetalRoughTexture.TexCoord > 0) {
			return fmt.Errorf("GLB material %d metallic-roughness map binding is invalid", i)
		}
		if material.OcclusionTexture != nil && (material.OcclusionTexture.Index >= textureCount || material.OcclusionTexture.TexCoord > 0 || material.OcclusionTexture.Strength != nil && !unitValue(*material.OcclusionTexture.Strength)) {
			return fmt.Errorf("GLB material %d occlusion map binding is invalid", i)
		}
		if material.EmissiveTexture != nil && (material.EmissiveTexture.Index >= textureCount || material.EmissiveTexture.TexCoord > 0) {
			return fmt.Errorf("GLB material %d emissive map binding is invalid", i)
		}
		if len(material.EmissiveFactor) != 0 && (len(material.EmissiveFactor) != 3 || !unitFactors(material.EmissiveFactor)) {
			return fmt.Errorf("GLB material %d emissive factor is invalid", i)
		}
		if material.AlphaMode == "MASK" && (material.AlphaCutoff == nil || !unitValue(*material.AlphaCutoff)) || material.AlphaMode != "MASK" && material.AlphaCutoff != nil {
			return fmt.Errorf("GLB material %d alpha cutoff disagrees with its mode", i)
		}
	}
	return nil
}

type glbMaterialFacts struct {
	specularFactor float64
	specularColor  [3]float64
	transmission   float64
}

func inspectGLBMaterialExtensions(material glbMaterial, textureCount uint32) (glbMaterialFacts, error) {
	facts := glbMaterialFacts{specularFactor: 1, specularColor: [3]float64{1, 1, 1}}
	if raw, ok := material.Extensions["KHR_materials_specular"]; ok {
		var extension struct {
			SpecularFactor      *float64  `json:"specularFactor,omitempty"`
			SpecularColorFactor []float64 `json:"specularColorFactor,omitempty"`
			SpecularTexture     *struct {
				Index    uint32 `json:"index"`
				TexCoord uint32 `json:"texCoord,omitempty"`
			} `json:"specularTexture,omitempty"`
			SpecularColorTexture *struct {
				Index    uint32 `json:"index"`
				TexCoord uint32 `json:"texCoord,omitempty"`
			} `json:"specularColorTexture,omitempty"`
		}
		if err := decodeExtension(raw, &extension); err != nil {
			return facts, fmt.Errorf("KHR_materials_specular: %w", err)
		}
		if extension.SpecularFactor != nil {
			facts.specularFactor = *extension.SpecularFactor
		}
		if len(extension.SpecularColorFactor) != 0 {
			if len(extension.SpecularColorFactor) != 3 {
				return facts, errors.New("KHR_materials_specular color factor needs three components")
			}
			copy(facts.specularColor[:], extension.SpecularColorFactor)
		}
		if !unitValue(facts.specularFactor) || !unitFactors(facts.specularColor[:]) || !validTextureInfo(extension.SpecularTexture, textureCount) || !validTextureInfo(extension.SpecularColorTexture, textureCount) {
			return facts, errors.New("KHR_materials_specular factors or texture bindings are invalid")
		}
	}
	if raw, ok := material.Extensions["KHR_materials_transmission"]; ok {
		var extension struct {
			TransmissionFactor  *float64 `json:"transmissionFactor,omitempty"`
			TransmissionTexture *struct {
				Index    uint32 `json:"index"`
				TexCoord uint32 `json:"texCoord,omitempty"`
			} `json:"transmissionTexture,omitempty"`
		}
		if err := decodeExtension(raw, &extension); err != nil {
			return facts, fmt.Errorf("KHR_materials_transmission: %w", err)
		}
		if extension.TransmissionFactor != nil {
			facts.transmission = *extension.TransmissionFactor
		}
		if !unitValue(facts.transmission) || !validTextureInfo(extension.TransmissionTexture, textureCount) {
			return facts, errors.New("KHR_materials_transmission factor or texture binding is invalid")
		}
	}
	return facts, nil
}

func decodeExtension(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("extension contains trailing JSON")
	}
	return nil
}

func validTextureInfo(info *struct {
	Index    uint32 `json:"index"`
	TexCoord uint32 `json:"texCoord,omitempty"`
}, textureCount uint32) bool {
	return info == nil || info.Index < textureCount && info.TexCoord == 0
}

func validSamplerEnum(value *uint32, choices ...uint32) bool {
	if value == nil {
		return true
	}
	for _, choice := range choices {
		if *value == choice {
			return true
		}
	}
	return false
}

func validatePrimitiveBufferData(primitive PhysicalGLBPrimitive, doc glbDocument, binaryChunk []byte) ([2]float64, [2]float64, error) {
	var uvMin, uvMax [2]float64
	position := doc.Accessors[primitive.Position]
	referenced := make([]bool, position.Count)
	if primitive.IndexAccessor == nil {
		for i := range referenced {
			referenced[i] = true
		}
	} else {
		indices := doc.Accessors[*primitive.IndexAccessor]
		view := doc.BufferViews[*indices.BufferView]
		if view.ByteStride != 0 {
			return uvMin, uvMax, errors.New("index accessor cannot use a strided buffer view")
		}
		bytesPerIndex := uint64(2)
		switch indices.ComponentType {
		case 5121:
			bytesPerIndex = 1
		case 5123:
			bytesPerIndex = 2
		case 5125:
			bytesPerIndex = 4
		}
		start := uint64(view.ByteOffset) + uint64(indices.ByteOffset)
		for index := uint32(0); index < indices.Count; index++ {
			offset := start + uint64(index)*bytesPerIndex
			var value uint32
			switch bytesPerIndex {
			case 1:
				value = uint32(binaryChunk[offset])
			case 2:
				value = uint32(binary.LittleEndian.Uint16(binaryChunk[offset : offset+2]))
			case 4:
				value = binary.LittleEndian.Uint32(binaryChunk[offset : offset+4])
			}
			if value >= position.Count {
				return uvMin, uvMax, errors.New("index accessor references a vertex outside POSITION")
			}
			referenced[value] = true
		}
	}
	positionView := doc.BufferViews[*position.BufferView]
	positionStart := uint64(positionView.ByteOffset) + uint64(position.ByteOffset)
	stride := uint64(positionView.ByteStride)
	if stride == 0 {
		stride = 12
	}
	minima, maxima := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}, [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for vertex := uint32(0); vertex < position.Count; vertex++ {
		for axis := 0; axis < 3; axis++ {
			offset := positionStart + uint64(vertex)*stride + uint64(axis*4)
			value := float64(math.Float32frombits(binary.LittleEndian.Uint32(binaryChunk[offset : offset+4])))
			if !finiteNumber(value) {
				return uvMin, uvMax, errors.New("position accessor contains a non-finite vertex")
			}
			minima[axis] = math.Min(minima[axis], value)
			maxima[axis] = math.Max(maxima[axis], value)
		}
	}
	for axis := 0; axis < 3; axis++ {
		if math.Abs(minima[axis]-primitive.PositionMin[axis]) > 1e-5 || math.Abs(maxima[axis]-primitive.PositionMax[axis]) > 1e-5 {
			return uvMin, uvMax, errors.New("position accessor bounds differ from decoded vertices")
		}
	}
	if primitive.UV0 != nil {
		accessor := doc.Accessors[*primitive.UV0]
		view := doc.BufferViews[*accessor.BufferView]
		stride := uint64(view.ByteStride)
		componentBytes := uint64(4)
		if stride == 0 {
			stride = 8
		}
		if accessor.ComponentType == 5121 {
			componentBytes = 1
		} else if accessor.ComponentType == 5123 {
			componentBytes = 2
		}
		uvMin = [2]float64{math.Inf(1), math.Inf(1)}
		uvMax = [2]float64{math.Inf(-1), math.Inf(-1)}
		for vertex := uint32(0); vertex < accessor.Count; vertex++ {
			for axis := 0; axis < 2; axis++ {
				offset := uint64(view.ByteOffset) + uint64(accessor.ByteOffset) + uint64(vertex)*stride + uint64(axis)*componentBytes
				var value float64
				switch accessor.ComponentType {
				case 5126:
					value = float64(math.Float32frombits(binary.LittleEndian.Uint32(binaryChunk[offset : offset+4])))
				case 5121:
					value = float64(binaryChunk[offset])
					if accessor.Normalized {
						value /= math.MaxUint8
					}
				case 5123:
					value = float64(binary.LittleEndian.Uint16(binaryChunk[offset : offset+2]))
					if accessor.Normalized {
						value /= math.MaxUint16
					}
				}
				if !finiteNumber(value) {
					return [2]float64{}, [2]float64{}, errors.New("UV accessor contains a non-finite value")
				}
				if referenced[vertex] {
					uvMin[axis] = math.Min(uvMin[axis], value)
					uvMax[axis] = math.Max(uvMax[axis], value)
				}
			}
		}
	}
	return uvMin, uvMax, nil
}

func unitRectUV(minimum, maximum [2]float64) bool {
	const epsilon = 1e-6
	return math.Abs(minimum[0]) <= epsilon && math.Abs(minimum[1]) <= epsilon && math.Abs(maximum[0]-1) <= epsilon && math.Abs(maximum[1]-1) <= epsilon
}

func validateGLBNodes(doc glbDocument) ([]*uint32, []bool, error) {
	parents := make([]*uint32, len(doc.Nodes))
	for parent, node := range doc.Nodes {
		if err := validateExtensionMap(node.Extensions); err != nil {
			return nil, nil, fmt.Errorf("GLB node %d: %w", parent, err)
		}
		if node.Name == "" || len(node.Name) > 256 || node.Matrix != nil {
			return nil, nil, fmt.Errorf("GLB node %d needs a name and decomposed TRS", parent)
		}
		for _, child := range node.Children {
			if int(child) >= len(doc.Nodes) || parents[child] != nil {
				return nil, nil, errors.New("GLB node hierarchy has an invalid or multiply-parented child")
			}
			p := uint32(parent)
			parents[child] = &p
		}
	}
	reachable := make([]bool, len(doc.Nodes))
	visiting := make([]bool, len(doc.Nodes))
	var visit func(uint32) error
	visit = func(index uint32) error {
		if visiting[index] {
			return errors.New("GLB node hierarchy contains a cycle")
		}
		if reachable[index] {
			return errors.New("GLB node is reachable by multiple paths")
		}
		visiting[index] = true
		for _, child := range doc.Nodes[index].Children {
			if err := visit(child); err != nil {
				return err
			}
		}
		visiting[index], reachable[index] = false, true
		return nil
	}
	for _, root := range doc.Scenes[0].Nodes {
		if int(root) >= len(doc.Nodes) || parents[root] != nil {
			return nil, nil, errors.New("GLB selected scene root is invalid")
		}
		if err := visit(root); err != nil {
			return nil, nil, err
		}
	}
	return parents, reachable, nil
}

func glbNodeFrame(node glbNode) (physical.TRS, error) {
	frame := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	if node.Translation != nil {
		if len(node.Translation) != 3 {
			return physical.TRS{}, errors.New("translation must have three components")
		}
		frame.Translation = physical.Vec3{X: node.Translation[0], Y: node.Translation[1], Z: node.Translation[2]}
	}
	if node.Rotation != nil {
		if len(node.Rotation) != 4 {
			return physical.TRS{}, errors.New("rotation must have four components")
		}
		frame.Rotation = physical.QuatXYZW{X: node.Rotation[0], Y: node.Rotation[1], Z: node.Rotation[2], W: node.Rotation[3]}
	}
	if node.Scale != nil {
		if len(node.Scale) != 3 {
			return physical.TRS{}, errors.New("scale must have three components")
		}
		frame.Scale = physical.Vec3{X: node.Scale[0], Y: node.Scale[1], Z: node.Scale[2]}
	}
	return physical.CanonicalizeTRS(frame)
}

func inspectGLBPrimitive(index uint32, primitive glbPrimitive, doc glbDocument) (PhysicalGLBPrimitive, error) {
	if primitive.Mode != nil && *primitive.Mode != 4 || len(primitive.Targets) != 0 || primitive.Material == nil || int(*primitive.Material) >= len(doc.Materials) {
		return PhysicalGLBPrimitive{}, errors.New("primitive must be a material-bound triangle list without morph targets")
	}
	for semantic := range primitive.Attributes {
		if semantic != "POSITION" && semantic != "NORMAL" && semantic != "TEXCOORD_0" {
			return PhysicalGLBPrimitive{}, fmt.Errorf("attribute %q is outside the supported subset", semantic)
		}
	}
	positionIndex, ok := primitive.Attributes["POSITION"]
	if !ok || int(positionIndex) >= len(doc.Accessors) {
		return PhysicalGLBPrimitive{}, errors.New("primitive has no valid POSITION accessor")
	}
	position := doc.Accessors[positionIndex]
	if position.ComponentType != 5126 || position.Type != "VEC3" || len(position.Min) != 3 || len(position.Max) != 3 {
		return PhysicalGLBPrimitive{}, errors.New("POSITION must be float32 VEC3 with finite bounds")
	}
	result := PhysicalGLBPrimitive{Index: index, Mode: 4, Material: *primitive.Material, Position: positionIndex, VertexCount: position.Count, PositionMin: [3]float64{position.Min[0], position.Min[1], position.Min[2]}, PositionMax: [3]float64{position.Max[0], position.Max[1], position.Max[2]}}
	if normal, ok := primitive.Attributes["NORMAL"]; ok {
		if int(normal) >= len(doc.Accessors) || doc.Accessors[normal].ComponentType != 5126 || doc.Accessors[normal].Type != "VEC3" || doc.Accessors[normal].Count != position.Count {
			return PhysicalGLBPrimitive{}, errors.New("NORMAL accessor is incompatible with POSITION")
		}
		result.Normal = &normal
	}
	if uv, ok := primitive.Attributes["TEXCOORD_0"]; ok {
		if int(uv) >= len(doc.Accessors) || doc.Accessors[uv].Type != "VEC2" || doc.Accessors[uv].Count != position.Count || (doc.Accessors[uv].ComponentType != 5126 && doc.Accessors[uv].ComponentType != 5123 && doc.Accessors[uv].ComponentType != 5121) {
			return PhysicalGLBPrimitive{}, errors.New("TEXCOORD_0 accessor is unsupported")
		}
		if doc.Accessors[uv].ComponentType == 5126 && doc.Accessors[uv].Normalized || doc.Accessors[uv].ComponentType != 5126 && !doc.Accessors[uv].Normalized {
			return PhysicalGLBPrimitive{}, errors.New("TEXCOORD_0 normalization flag is invalid")
		}
		result.UV0 = &uv
	}
	indexCount := position.Count
	if primitive.Indices != nil {
		if int(*primitive.Indices) >= len(doc.Accessors) {
			return PhysicalGLBPrimitive{}, errors.New("index accessor is outside the accessor table")
		}
		accessor := doc.Accessors[*primitive.Indices]
		if accessor.Type != "SCALAR" || accessor.ComponentType != 5121 && accessor.ComponentType != 5123 && accessor.ComponentType != 5125 {
			return PhysicalGLBPrimitive{}, errors.New("index accessor must be unsigned scalar")
		}
		result.IndexAccessor = primitive.Indices
		indexCount = accessor.Count
	}
	if indexCount%3 != 0 {
		return PhysicalGLBPrimitive{}, errors.New("triangle index count is not divisible by three")
	}
	result.IndexCount, result.TriangleCount = indexCount, indexCount/3
	return result, nil
}

func scaleFrame(frame physical.TRS, unit string) physical.TRS {
	factor := map[string]float64{"m": 1, "cm": 0.01, "mm": 0.001}[unit]
	frame.Translation.X *= factor
	frame.Translation.Y *= factor
	frame.Translation.Z *= factor
	return frame
}

func trsNearlyEqual(a, b physical.TRS, epsilon float64) bool {
	return vecNearlyEqual(a.Translation, b.Translation, epsilon) && vecNearlyEqual(a.Scale, b.Scale, epsilon) &&
		math.Abs(a.Rotation.X-b.Rotation.X) <= epsilon && math.Abs(a.Rotation.Y-b.Rotation.Y) <= epsilon && math.Abs(a.Rotation.Z-b.Rotation.Z) <= epsilon && math.Abs(a.Rotation.W-b.Rotation.W) <= epsilon
}
func vecNearlyEqual(a, b physical.Vec3, epsilon float64) bool {
	return math.Abs(a.X-b.X) <= epsilon && math.Abs(a.Y-b.Y) <= epsilon && math.Abs(a.Z-b.Z) <= epsilon
}
func unitFactors(values []float64) bool {
	for _, value := range values {
		if !unitValue(value) {
			return false
		}
	}
	return true
}
func unitValue(value float64) bool    { return finiteNumber(value) && value >= 0 && value <= 1 }
func finiteNumber(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func sha256Bytes(data []byte) string  { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
