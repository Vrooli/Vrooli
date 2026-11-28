package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
)

const (
	intakePreviewPanel        = 256
	intakePreviewViews        = 3
	intakePreviewHeight       = 256
	intakePreviewMaxTriangles = 200_000
	intakePreviewMaxVertices  = 2_000_000
	intakePreviewMaxWork      = 24_000_000
)

// IntakePreview is a bounded inspection image for an immutable, unreviewed
// intake revision. It is deliberately not a physical material or scale claim.
type IntakePreview struct {
	CandidateID    string
	Revision       uint32
	RevisionSHA256 string
	GLBSHA256      string
	PreviewSHA256  string
	PNG            []byte
	Width          uint32
	Height         uint32
	VertexColors   bool
}

type intakePreviewDoc struct {
	Buffers     []intakePreviewBuffer     `json:"buffers"`
	BufferViews []intakePreviewBufferView `json:"bufferViews"`
	Accessors   []intakePreviewAccessor   `json:"accessors"`
	Meshes      []intakePreviewMesh       `json:"meshes"`
	Nodes       []intakePreviewNode       `json:"nodes"`
}
type intakePreviewBuffer struct {
	URI        string `json:"uri"`
	ByteLength uint32 `json:"byteLength"`
}
type intakePreviewBufferView struct {
	Buffer     uint32 `json:"buffer"`
	ByteOffset uint32 `json:"byteOffset"`
	ByteLength uint32 `json:"byteLength"`
	ByteStride uint32 `json:"byteStride"`
}
type intakePreviewAccessor struct {
	BufferView    *uint32         `json:"bufferView"`
	ByteOffset    uint32          `json:"byteOffset"`
	ComponentType uint32          `json:"componentType"`
	Normalized    bool            `json:"normalized"`
	Count         uint32          `json:"count"`
	Type          string          `json:"type"`
	Sparse        json.RawMessage `json:"sparse"`
}
type intakePreviewMesh struct {
	Primitives []intakePreviewPrimitive `json:"primitives"`
}
type intakePreviewPrimitive struct {
	Attributes map[string]uint32 `json:"attributes"`
	Indices    *uint32           `json:"indices"`
	Mode       *uint32           `json:"mode"`
}
type intakePreviewNode struct {
	Mesh        *uint32   `json:"mesh"`
	Children    []uint32  `json:"children"`
	Translation []float64 `json:"translation"`
	Rotation    []float64 `json:"rotation"`
	Scale       []float64 `json:"scale"`
	Matrix      []float64 `json:"matrix"`
}
type intakePreviewVertex struct {
	position [3]float64
	color    [4]float64
}
type intakePreviewTriangle [3]uint32
type intakePreviewGeometry struct {
	vertices  []intakePreviewVertex
	triangles []intakePreviewTriangle
	hasColors bool
}

// Preview renders a fixed three-view, vertex-color-only PNG after reopening the
// exact immutable revision and checking its revision digest. No texture, PBR,
// physical scale, or render-readiness claim is synthesized.
func (s *IntakeService) Preview(ctx context.Context, id string, revision uint32, revisionSHA string) (IntakePreview, error) {
	candidate, data, err := s.readExactIntake(ctx, id, revision, revisionSHA)
	if err != nil {
		return IntakePreview{}, err
	}
	geometry, err := decodeIntakePreviewGLB(data)
	if err != nil {
		return IntakePreview{}, fmt.Errorf("%w: %v", ErrInvalidIntake, err)
	}
	imageData, err := renderIntakePreview(geometry)
	if err != nil {
		return IntakePreview{}, err
	}
	sum := sha256.Sum256(imageData)
	return IntakePreview{
		CandidateID: candidate.ID, Revision: candidate.Revision, RevisionSHA256: candidate.RevisionSHA256,
		GLBSHA256: candidate.OriginalSHA256, PreviewSHA256: hex.EncodeToString(sum[:]), PNG: imageData,
		Width: intakePreviewPanel * intakePreviewViews, Height: intakePreviewHeight, VertexColors: geometry.hasColors,
	}, nil
}

func decodeIntakePreviewGLB(data []byte) (intakePreviewGeometry, error) {
	docRaw, chunks, _, err := decodeMaterialGLB(data)
	if err != nil {
		return intakePreviewGeometry{}, err
	}
	docBytes, err := json.Marshal(docRaw)
	if err != nil {
		return intakePreviewGeometry{}, fmt.Errorf("encode preview GLB document: %w", err)
	}
	var doc intakePreviewDoc
	if err := json.Unmarshal(docBytes, &doc); err != nil {
		return intakePreviewGeometry{}, fmt.Errorf("decode preview GLB document: %w", err)
	}
	if len(doc.Buffers) != 1 || doc.Buffers[0].URI != "" || len(doc.Meshes) == 0 || len(doc.Nodes) == 0 {
		return intakePreviewGeometry{}, fmt.Errorf("%w: preview requires one embedded GLB buffer, meshes and nodes", ErrInvalidIntake)
	}
	var bin []byte
	for _, chunk := range chunks {
		if chunk.kind == 0x004e4942 {
			if bin != nil {
				return intakePreviewGeometry{}, fmt.Errorf("%w: preview does not support multiple BIN chunks", ErrInvalidIntake)
			}
			bin = chunk.data
		}
	}
	if uint64(doc.Buffers[0].ByteLength) > uint64(len(bin)) || bin == nil {
		return intakePreviewGeometry{}, fmt.Errorf("%w: preview buffer exceeds embedded BIN chunk", ErrInvalidIntake)
	}
	// GLB BIN chunks may contain up to three padding bytes. Accessors must stay
	// within the buffer's declared logical length, not the padded chunk length.
	bin = bin[:doc.Buffers[0].ByteLength]
	for _, node := range doc.Nodes {
		// Hierarchy itself is safe when every node transform is identity;
		// reject transforms on all nodes, including transform-only parents.
		if !previewIdentityTransform(node) {
			return intakePreviewGeometry{}, fmt.Errorf("%w: preview only supports identity node transforms", ErrInvalidIntake)
		}
	}
	usedMeshes := make(map[uint32]bool)
	for _, node := range doc.Nodes {
		if node.Mesh == nil {
			continue
		}
		if int(*node.Mesh) >= len(doc.Meshes) {
			return intakePreviewGeometry{}, fmt.Errorf("%w: preview node references an invalid mesh", ErrInvalidIntake)
		}
		usedMeshes[*node.Mesh] = true
	}
	if len(usedMeshes) == 0 {
		return intakePreviewGeometry{}, fmt.Errorf("%w: preview scene has no mesh nodes", ErrInvalidIntake)
	}
	var out intakePreviewGeometry
	for meshIndex := range doc.Meshes {
		if !usedMeshes[uint32(meshIndex)] {
			continue
		}
		mesh := doc.Meshes[meshIndex]
		if len(mesh.Primitives) == 0 || len(mesh.Primitives) > 32 {
			return intakePreviewGeometry{}, fmt.Errorf("%w: preview mesh primitive count is outside bounds", ErrInvalidIntake)
		}
		for _, primitive := range mesh.Primitives {
			mode := uint32(4)
			if primitive.Mode != nil {
				mode = *primitive.Mode
			}
			if mode != 4 {
				return intakePreviewGeometry{}, fmt.Errorf("%w: preview only supports triangle primitives", ErrInvalidIntake)
			}
			positionIndex, ok := primitive.Attributes["POSITION"]
			if !ok || int(positionIndex) >= len(doc.Accessors) {
				return intakePreviewGeometry{}, fmt.Errorf("%w: preview primitive has no valid POSITION accessor", ErrInvalidIntake)
			}
			positions, err := readPreviewAccessor(bin, doc.BufferViews, doc.Accessors[positionIndex], "VEC3", 5126)
			if err != nil {
				return intakePreviewGeometry{}, fmt.Errorf("preview POSITION: %w", err)
			}
			if len(out.vertices)+len(positions) > intakePreviewMaxVertices {
				return intakePreviewGeometry{}, fmt.Errorf("%w: preview vertex count exceeds bound", ErrInvalidIntake)
			}
			colors := make([][4]float64, len(positions))
			colorIndex, hasColors := primitive.Attributes["COLOR_0"]
			if hasColors {
				if int(colorIndex) >= len(doc.Accessors) {
					return intakePreviewGeometry{}, fmt.Errorf("%w: preview COLOR_0 accessor is invalid", ErrInvalidIntake)
				}
				colors, err = readPreviewColors(bin, doc.BufferViews, doc.Accessors[colorIndex], uint32(len(positions)))
				if err != nil {
					return intakePreviewGeometry{}, fmt.Errorf("preview COLOR_0: %w", err)
				}
				out.hasColors = true
			}
			base := uint32(len(out.vertices))
			for i, position := range positions {
				c := [4]float64{0.72, 0.73, 0.75, 1}
				if hasColors {
					c = colors[i]
				}
				out.vertices = append(out.vertices, intakePreviewVertex{position: position, color: c})
			}
			indices := make([]uint32, len(positions))
			if primitive.Indices != nil {
				if int(*primitive.Indices) >= len(doc.Accessors) {
					return intakePreviewGeometry{}, fmt.Errorf("%w: preview index accessor is invalid", ErrInvalidIntake)
				}
				indices, err = readPreviewIndices(bin, doc.BufferViews, doc.Accessors[*primitive.Indices])
				if err != nil {
					return intakePreviewGeometry{}, fmt.Errorf("preview indices: %w", err)
				}
			} else {
				for i := range indices {
					indices[i] = uint32(i)
				}
			}
			if len(indices)%3 != 0 || len(out.triangles)+len(indices)/3 > intakePreviewMaxTriangles {
				return intakePreviewGeometry{}, fmt.Errorf("%w: preview triangle count is invalid or exceeds bound", ErrInvalidIntake)
			}
			for i := 0; i < len(indices); i += 3 {
				for _, idx := range indices[i : i+3] {
					if int(idx) >= len(positions) {
						return intakePreviewGeometry{}, fmt.Errorf("%w: preview triangle index exceeds POSITION accessor", ErrInvalidIntake)
					}
				}
				out.triangles = append(out.triangles, intakePreviewTriangle{base + indices[i], base + indices[i+1], base + indices[i+2]})
			}
		}
	}
	if len(out.vertices) == 0 || len(out.triangles) == 0 {
		return intakePreviewGeometry{}, fmt.Errorf("%w: preview contains no triangle geometry", ErrInvalidIntake)
	}
	return out, nil
}

func previewIdentityTransform(node intakePreviewNode) bool {
	if len(node.Matrix) != 0 {
		return false
	}
	if len(node.Translation) != 0 && (len(node.Translation) != 3 || node.Translation[0] != 0 || node.Translation[1] != 0 || node.Translation[2] != 0) {
		return false
	}
	if len(node.Scale) != 0 && (len(node.Scale) != 3 || node.Scale[0] != 1 || node.Scale[1] != 1 || node.Scale[2] != 1) {
		return false
	}
	if len(node.Rotation) != 0 && (len(node.Rotation) != 4 || node.Rotation[0] != 0 || node.Rotation[1] != 0 || node.Rotation[2] != 0 || node.Rotation[3] != 1) {
		return false
	}
	return true
}

func previewAccessorBytes(bin []byte, views []intakePreviewBufferView, accessor intakePreviewAccessor, expectedType string, expectedComponent uint32) ([]byte, int, int, error) {
	if accessor.BufferView == nil || int(*accessor.BufferView) >= len(views) || len(accessor.Sparse) != 0 || accessor.Count == 0 {
		return nil, 0, 0, errors.New("buffer view, non-sparse data and non-zero count are required")
	}
	if accessor.Type != expectedType || accessor.ComponentType != expectedComponent {
		return nil, 0, 0, fmt.Errorf("expected %s component type %d", expectedType, expectedComponent)
	}
	view := views[*accessor.BufferView]
	if view.Buffer != 0 {
		return nil, 0, 0, errors.New("only embedded buffer zero is supported")
	}
	components := map[string]int{"SCALAR": 1, "VEC3": 3, "VEC4": 4}[accessor.Type]
	componentBytes := 0
	switch accessor.ComponentType {
	case 5120, 5121:
		componentBytes = 1
	case 5122, 5123:
		componentBytes = 2
	case 5125, 5126:
		componentBytes = 4
	default:
		return nil, 0, 0, errors.New("component type is unsupported")
	}
	elementBytes := components * componentBytes
	stride := int(view.ByteStride)
	if stride == 0 {
		stride = elementBytes
	}
	if stride < elementBytes || stride > 252 || accessor.ByteOffset%uint32(componentBytes) != 0 || stride%componentBytes != 0 {
		return nil, 0, 0, errors.New("accessor stride or alignment is invalid")
	}
	start := uint64(view.ByteOffset) + uint64(accessor.ByteOffset)
	end := start + uint64(accessor.Count-1)*uint64(stride) + uint64(elementBytes)
	viewEnd := uint64(view.ByteOffset) + uint64(view.ByteLength)
	if end < start || end > viewEnd || viewEnd > uint64(len(bin)) {
		return nil, 0, 0, errors.New("accessor exceeds its bounded buffer view")
	}
	return bin[start:end], stride, elementBytes, nil
}

func readPreviewAccessor(bin []byte, views []intakePreviewBufferView, accessor intakePreviewAccessor, kind string, component uint32) ([][3]float64, error) {
	if accessor.Count > intakePreviewMaxVertices || accessor.Normalized {
		return nil, errors.New("POSITION count or normalized flag is invalid")
	}
	data, stride, _, err := previewAccessorBytes(bin, views, accessor, kind, component)
	if err != nil {
		return nil, err
	}
	out := make([][3]float64, accessor.Count)
	for i := range out {
		for j := 0; j < 3; j++ {
			v := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*stride+j*4 : i*stride+j*4+4])))
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1e6 {
				return nil, errors.New("POSITION contains a non-finite or out-of-range value")
			}
			out[i][j] = v
		}
	}
	return out, nil
}

func readPreviewColors(bin []byte, views []intakePreviewBufferView, accessor intakePreviewAccessor, vertices uint32) ([][4]float64, error) {
	if accessor.Count != vertices || (accessor.Type != "VEC3" && accessor.Type != "VEC4") {
		return nil, errors.New("COLOR_0 count or vector size differs from POSITION")
	}
	if accessor.ComponentType != 5121 && accessor.ComponentType != 5126 {
		return nil, errors.New("COLOR_0 must use unsigned bytes or float32")
	}
	if accessor.ComponentType == 5121 && !accessor.Normalized || accessor.ComponentType == 5126 && accessor.Normalized {
		return nil, errors.New("COLOR_0 normalized flag is invalid")
	}
	data, stride, _, err := previewAccessorBytes(bin, views, accessor, accessor.Type, accessor.ComponentType)
	if err != nil {
		return nil, err
	}
	channels := 3
	if accessor.Type == "VEC4" {
		channels = 4
	}
	out := make([][4]float64, vertices)
	for i := range out {
		out[i][3] = 1
		for j := 0; j < channels; j++ {
			v := float64(data[i*stride+j]) / 255
			if accessor.ComponentType == 5126 {
				v = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i*stride+j*4 : i*stride+j*4+4])))
			}
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
				return nil, errors.New("COLOR_0 has non-finite or out-of-range channels")
			}
			out[i][j] = v
		}
	}
	return out, nil
}

func readPreviewIndices(bin []byte, views []intakePreviewBufferView, accessor intakePreviewAccessor) ([]uint32, error) {
	if accessor.Type != "SCALAR" || accessor.Count > intakePreviewMaxTriangles*3 || accessor.Normalized {
		return nil, errors.New("index accessor is not a bounded unsigned scalar array")
	}
	componentBytes := 0
	switch accessor.ComponentType {
	case 5121:
		componentBytes = 1
	case 5123:
		componentBytes = 2
	case 5125:
		componentBytes = 4
	default:
		return nil, errors.New("index component type is unsupported")
	}
	data, stride, _, err := previewAccessorBytes(bin, views, accessor, "SCALAR", accessor.ComponentType)
	if err != nil {
		return nil, err
	}
	if stride < componentBytes {
		return nil, errors.New("index stride is too short")
	}
	out := make([]uint32, accessor.Count)
	for i := range out {
		switch componentBytes {
		case 1:
			out[i] = uint32(data[i*stride])
		case 2:
			out[i] = uint32(binary.LittleEndian.Uint16(data[i*stride : i*stride+2]))
		case 4:
			out[i] = binary.LittleEndian.Uint32(data[i*stride : i*stride+4])
		}
	}
	return out, nil
}

type previewView struct {
	name        string
	u, v, depth int
}

func renderIntakePreview(geometry intakePreviewGeometry) ([]byte, error) {
	// Candidate orientation has not been reviewed, so label the actual
	// coordinate planes instead of implying front/side/top semantics.
	views := []previewView{{name: "XY", u: 0, v: 1, depth: 2}, {name: "ZY", u: 2, v: 1, depth: 0}, {name: "XZ", u: 0, v: 2, depth: 1}}
	img := image.NewRGBA(image.Rect(0, 0, intakePreviewPanel*len(views), intakePreviewHeight))
	fillPreview(img, color.RGBA{R: 248, G: 248, B: 246, A: 255})
	work := 0
	for index, view := range views {
		if index > 0 {
			for y := 0; y < intakePreviewHeight; y++ {
				img.Set(index*intakePreviewPanel, y, color.RGBA{R: 198, G: 198, B: 194, A: 255})
			}
		}
		if err := renderPreviewView(img, geometry, view, index*intakePreviewPanel, &work); err != nil {
			return nil, err
		}
		drawPreviewLabel(img, index*intakePreviewPanel+10, 10, view.name)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("encode geometry preview: %w", err)
	}
	return out.Bytes(), nil
}

func renderPreviewView(img *image.RGBA, geometry intakePreviewGeometry, view previewView, xOffset int, work *int) error {
	minU, maxU := math.Inf(1), math.Inf(-1)
	minV, maxV := math.Inf(1), math.Inf(-1)
	project := func(p [3]float64) (u, v, d float64) { return p[view.u], p[view.v], p[view.depth] }
	for _, vertex := range geometry.vertices {
		u, v, _ := project(vertex.position)
		minU, maxU = math.Min(minU, u), math.Max(maxU, u)
		minV, maxV = math.Min(minV, v), math.Max(maxV, v)
	}
	spanU, spanV := maxU-minU, maxV-minV
	if spanU <= 1e-9 {
		spanU = math.Max(spanV, 1) * 0.01
	}
	if spanV <= 1e-9 {
		spanV = math.Max(spanU, 1) * 0.01
	}
	contentTop, contentBottom := 28.0, float64(intakePreviewHeight-12)
	scale := math.Min(float64(intakePreviewPanel-24)/spanU, (contentBottom-contentTop)/spanV)
	centerU, centerV := (minU+maxU)/2, (minV+maxV)/2
	centerX, centerY := float64(xOffset+intakePreviewPanel/2), (contentTop+contentBottom)/2
	coords := make([][3]float64, len(geometry.vertices))
	depths := make([]float64, img.Bounds().Dx()*intakePreviewHeight)
	for i := range depths {
		depths[i] = math.Inf(-1)
	}
	for i, vertex := range geometry.vertices {
		u, v, d := project(vertex.position)
		coords[i] = [3]float64{centerX + (u-centerU)*scale, centerY - (v-centerV)*scale, d}
	}
	for _, triangle := range geometry.triangles {
		a, b, c := coords[triangle[0]], coords[triangle[1]], coords[triangle[2]]
		minX := maxInt(xOffset+2, int(math.Floor(math.Min(a[0], math.Min(b[0], c[0])))))
		maxX := minInt(xOffset+intakePreviewPanel-3, int(math.Ceil(math.Max(a[0], math.Max(b[0], c[0])))))
		minY := maxInt(24, int(math.Floor(math.Min(a[1], math.Min(b[1], c[1])))))
		maxY := minInt(intakePreviewHeight-3, int(math.Ceil(math.Max(a[1], math.Max(b[1], c[1])))))
		if maxX < minX || maxY < minY {
			continue
		}
		area := edge2(a[0], a[1], b[0], b[1], c[0], c[1])
		if math.Abs(area) < 1e-9 {
			continue
		}
		*work += (maxX - minX + 1) * (maxY - minY + 1)
		if *work > intakePreviewMaxWork {
			return fmt.Errorf("%w: preview raster work exceeds bound", ErrInvalidIntake)
		}
		p0, p1, p2 := geometry.vertices[triangle[0]].position, geometry.vertices[triangle[1]].position, geometry.vertices[triangle[2]].position
		normal := cross3(sub3(p1, p0), sub3(p2, p0))
		length := math.Sqrt(dot3(normal, normal))
		shade := 0.6
		if length > 1e-12 {
			shade = 0.32 + 0.68*math.Abs(normal[view.depth]/length)
		}
		ca, cb, cc := geometry.vertices[triangle[0]].color, geometry.vertices[triangle[1]].color, geometry.vertices[triangle[2]].color
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				w0 := edge2(b[0], b[1], c[0], c[1], px, py) / area
				w1 := edge2(c[0], c[1], a[0], a[1], px, py) / area
				w2 := 1 - w0 - w1
				if w0 < -1e-8 || w1 < -1e-8 || w2 < -1e-8 {
					continue
				}
				depth := w0*a[2] + w1*b[2] + w2*c[2]
				pos := y*img.Bounds().Dx() + x
				if depth <= depths[pos] {
					continue
				}
				depths[pos] = depth
				rgb := [3]uint8{}
				for channel := 0; channel < 3; channel++ {
					value := (w0*ca[channel] + w1*cb[channel] + w2*cc[channel]) * shade
					rgb[channel] = uint8(math.Round(math.Max(0, math.Min(1, value)) * 255))
				}
				img.SetRGBA(x, y, color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255})
			}
		}
	}
	return nil
}

func edge2(ax, ay, bx, by, px, py float64) float64 { return (px-ax)*(by-ay) - (py-ay)*(bx-ax) }
func sub3(a, b [3]float64) [3]float64              { return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func cross3(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func dot3(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func fillPreview(img *image.RGBA, fill color.RGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetRGBA(x, y, fill)
		}
	}
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var previewFont = map[rune][5]byte{
	'A': {7, 5, 7, 5, 5}, 'D': {6, 5, 5, 5, 6}, 'E': {7, 4, 6, 4, 7}, 'F': {7, 4, 6, 4, 4}, 'I': {7, 2, 2, 2, 7}, 'N': {5, 7, 7, 7, 5}, 'O': {7, 5, 5, 5, 7}, 'P': {6, 5, 6, 4, 4}, 'R': {6, 5, 6, 5, 5}, 'S': {7, 4, 7, 1, 7}, 'T': {7, 2, 2, 2, 2}, ' ': {0, 0, 0, 0, 0},
	'X': {5, 5, 2, 5, 5}, 'Y': {5, 5, 2, 2, 2}, 'Z': {7, 1, 2, 4, 7},
}

func drawPreviewLabel(img *image.RGBA, x, y int, value string) {
	ink := color.RGBA{R: 65, G: 67, B: 70, A: 255}
	for _, r := range value {
		glyph := previewFont[r]
		for row, bits := range glyph {
			for col := 0; col < 3; col++ {
				if bits&(1<<uint(2-col)) != 0 {
					img.SetRGBA(x+col, y+row, ink)
				}
			}
		}
		x += 4
	}
}
