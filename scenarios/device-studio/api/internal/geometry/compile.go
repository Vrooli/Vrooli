package geometry

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

const MaxTriangles = 100_000
const MaxGLBBytes = 8 << 20

type vec3 [3]float32
type vec2 [2]float32
type glbMaterial struct {
	Name string `json:"name"`
	PBR  pbr    `json:"pbrMetallicRoughness"`
}
type pbr struct {
	BaseColor [4]float64 `json:"baseColorFactor"`
	Metallic  float64    `json:"metallicFactor"`
	Roughness float64    `json:"roughnessFactor"`
}
type glbPrimitive struct {
	Attributes map[string]int    `json:"attributes"`
	Indices    int               `json:"indices"`
	Material   int               `json:"material"`
	Extras     map[string]string `json:"extras,omitempty"`
}
type glbMesh struct {
	Name       string         `json:"name"`
	Primitives []glbPrimitive `json:"primitives"`
}
type glbNode struct {
	Name string `json:"name"`
	Mesh int    `json:"mesh"`
}
type glbJSON struct {
	Asset  map[string]string `json:"asset"`
	Scene  int               `json:"scene"`
	Scenes []struct {
		Nodes []int `json:"nodes"`
	} `json:"scenes"`
	Nodes       []glbNode        `json:"nodes"`
	Meshes      []glbMesh        `json:"meshes"`
	Materials   []glbMaterial    `json:"materials"`
	Accessors   []map[string]any `json:"accessors"`
	BufferViews []map[string]any `json:"bufferViews"`
	Buffers     []map[string]any `json:"buffers"`
}
type materialPreset struct {
	color        [3]float64
	metal, rough float64
}

var presets = map[string]materialPreset{
	"matte_polymer": {color: [3]float64{.16, .17, .18}, metal: 0, rough: .82}, "gloss_polymer": {color: [3]float64{.12, .13, .14}, metal: 0, rough: .24}, "rubber": {color: [3]float64{.07, .075, .08}, metal: 0, rough: .94},
	"glass": {color: [3]float64{.025, .03, .035}, metal: 0, rough: .08}, "anodized_metal": {color: [3]float64{.48, .50, .52}, metal: .82, rough: .32}, "neutral_metal": {color: [3]float64{.28, .29, .30}, metal: .68, rough: .48},
}

// Compile returns byte-stable GLB data for a validated recipe.
func Compile(r Recipe) ([]byte, error) {
	if err := Validate(r); err != nil {
		return nil, err
	}
	var bin bytes.Buffer
	doc := glbJSON{Asset: map[string]string{"version": "2.0", "generator": "device-studio-geometry-v1"}}
	doc.Scenes = []struct {
		Nodes []int `json:"nodes"`
	}{{}}
	materialIndex := map[string]int{}
	addMaterial := func(name string, preset materialPreset) int {
		if n, ok := materialIndex[name]; ok {
			return n
		}
		i := len(doc.Materials)
		materialIndex[name] = i
		doc.Materials = append(doc.Materials, glbMaterial{Name: name, PBR: pbr{BaseColor: [4]float64{preset.color[0], preset.color[1], preset.color[2], 1}, Metallic: preset.metal, Roughness: preset.rough}})
		return i
	}
	appendMesh := func(name string, verts []vec3, indices []uint16, material int, texcoords []vec2, smooth bool) error {
		verts, indices, texcoords = prepareNormals(verts, indices, texcoords, smooth)
		normals, err := vertexNormals(verts, indices)
		if err != nil {
			return err
		}
		align4(&bin)
		posOffset := bin.Len()
		var lo, hi [3]float64
		for i, v := range verts {
			for axis := 0; axis < 3; axis++ {
				f := float64(v[axis])
				if !finite(f) {
					return fmt.Errorf("geometry: non-finite vertex")
				}
				if i == 0 || f < lo[axis] {
					lo[axis] = f
				}
				if i == 0 || f > hi[axis] {
					hi[axis] = f
				}
				binary.Write(&bin, binary.LittleEndian, v[axis])
			}
		}
		posView := len(doc.BufferViews)
		doc.BufferViews = append(doc.BufferViews, map[string]any{"buffer": 0, "byteOffset": posOffset, "byteLength": len(verts) * 12, "target": 34962})
		posAccessor := len(doc.Accessors)
		doc.Accessors = append(doc.Accessors, map[string]any{"bufferView": posView, "componentType": 5126, "count": len(verts), "type": "VEC3", "min": lo, "max": hi})
		align4(&bin)
		normOffset := bin.Len()
		for _, n := range normals {
			for _, f := range n {
				binary.Write(&bin, binary.LittleEndian, f)
			}
		}
		normView := len(doc.BufferViews)
		doc.BufferViews = append(doc.BufferViews, map[string]any{"buffer": 0, "byteOffset": normOffset, "byteLength": len(normals) * 12, "target": 34962})
		normAccessor := len(doc.Accessors)
		doc.Accessors = append(doc.Accessors, map[string]any{"bufferView": normView, "componentType": 5126, "count": len(normals), "type": "VEC3"})
		attributes := map[string]int{"POSITION": posAccessor, "NORMAL": normAccessor}
		if len(texcoords) > 0 {
			if len(texcoords) != len(verts) {
				return fmt.Errorf("geometry: texture coordinates do not match vertices")
			}
			align4(&bin)
			uvOffset := bin.Len()
			for _, uv := range texcoords {
				binary.Write(&bin, binary.LittleEndian, uv[0])
				binary.Write(&bin, binary.LittleEndian, uv[1])
			}
			uvView := len(doc.BufferViews)
			doc.BufferViews = append(doc.BufferViews, map[string]any{"buffer": 0, "byteOffset": uvOffset, "byteLength": len(texcoords) * 8, "target": 34962})
			uvAccessor := len(doc.Accessors)
			doc.Accessors = append(doc.Accessors, map[string]any{"bufferView": uvView, "componentType": 5126, "count": len(texcoords), "type": "VEC2"})
			attributes["TEXCOORD_0"] = uvAccessor
		}
		align4(&bin)
		indexOffset := bin.Len()
		for _, v := range indices {
			binary.Write(&bin, binary.LittleEndian, v)
		}
		idxView := len(doc.BufferViews)
		doc.BufferViews = append(doc.BufferViews, map[string]any{"buffer": 0, "byteOffset": indexOffset, "byteLength": len(indices) * 2, "target": 34963})
		idxAccessor := len(doc.Accessors)
		doc.Accessors = append(doc.Accessors, map[string]any{"bufferView": idxView, "componentType": 5123, "count": len(indices), "type": "SCALAR"})
		mesh := len(doc.Meshes)
		primitive := glbPrimitive{Attributes: attributes, Indices: idxAccessor, Material: material}
		if name == "screen-surface" {
			primitive.Extras = map[string]string{"semantic": "independent_display_surface"}
		}
		doc.Meshes = append(doc.Meshes, glbMesh{Name: name, Primitives: []glbPrimitive{primitive}})
		doc.Nodes = append(doc.Nodes, glbNode{Name: name, Mesh: mesh})
		doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, len(doc.Nodes)-1)
		return nil
	}
	triangles := 0
	displayMesh := -1
	for _, p := range r.Parts {
		v, ix := box(p.Center, p.Size)
		if p.Kind == "rounded_box" {
			v, ix = roundedBox(p.Center, p.Size, p.Bevel, 8)
		}
		if p.Kind == "cylinder" {
			v, ix = cylinder(p.Center, p.Size)
		}
		rotateVertices(v, p.Center, p.Rotation)
		orientTriangles(v, ix, p.Center)
		scaleVertices(v)
		triangles += len(ix) / 3
		if triangles > MaxTriangles {
			return nil, fmt.Errorf("geometry: triangle limit exceeded")
		}
		if err := appendMesh(p.ID, v, ix, addMaterial(p.Material, presets[p.Material]), nil, p.Kind == "rounded_box" || p.Kind == "cylinder"); err != nil {
			return nil, err
		}
		if p.Role == "display" {
			displayMesh = len(doc.Meshes) - 1
		}
	}
	frame := screenPlane(r)
	triangles += 2
	if triangles > MaxTriangles {
		return nil, fmt.Errorf("geometry: triangle limit exceeded")
	}
	screenVerts := plane(frame)
	orientTriangles(screenVerts, []uint16{0, 1, 2, 0, 2, 3}, [3]float64{frame.center[0] - frame.normal[0], frame.center[1] - frame.normal[1], frame.center[2] - frame.normal[2]})
	scaleVertices(screenVerts)
	uv := []vec2{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	if err := appendMesh("screen-surface", screenVerts, []uint16{0, 1, 2, 0, 2, 3}, addMaterial("screen-glass", presets["glass"]), uv, false); err != nil {
		return nil, err
	}
	if displayMesh < 0 {
		return nil, fmt.Errorf("geometry: display mesh is unavailable")
	}
	screenMesh := len(doc.Meshes) - 1
	doc.Meshes[displayMesh].Primitives = append(doc.Meshes[displayMesh].Primitives, doc.Meshes[screenMesh].Primitives[0])
	doc.Meshes = doc.Meshes[:screenMesh]
	doc.Nodes = doc.Nodes[:len(doc.Nodes)-1]
	doc.Scenes[0].Nodes = doc.Scenes[0].Nodes[:len(doc.Scenes[0].Nodes)-1]
	doc.Buffers = []map[string]any{{"byteLength": bin.Len()}}
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	alignBytes(&jsonBytes, 0x20)
	binBytes := append([]byte(nil), bin.Bytes()...)
	alignBytes(&binBytes, 0)
	length := 12 + 8 + len(jsonBytes) + 8 + len(binBytes)
	if length > MaxGLBBytes {
		return nil, fmt.Errorf("geometry: GLB byte limit exceeded")
	}
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint32(0x46546c67))
	binary.Write(&out, binary.LittleEndian, uint32(2))
	binary.Write(&out, binary.LittleEndian, uint32(length))
	binary.Write(&out, binary.LittleEndian, uint32(len(jsonBytes)))
	binary.Write(&out, binary.LittleEndian, uint32(0x4e4f534a))
	out.Write(jsonBytes)
	binary.Write(&out, binary.LittleEndian, uint32(len(binBytes)))
	binary.Write(&out, binary.LittleEndian, uint32(0x004e4942))
	out.Write(binBytes)
	return out.Bytes(), nil
}

func rotateVertices(vertices []vec3, center [3]float64, rotation [3]int) {
	for i := range vertices {
		v := [3]float64{float64(vertices[i][0]) - center[0], float64(vertices[i][1]) - center[1], float64(vertices[i][2]) - center[2]}
		for axis, deg := range rotation {
			for n := 0; n < deg/90; n++ {
				a, b := (axis+1)%3, (axis+2)%3
				v[a], v[b] = -v[b], v[a]
			}
		}
		for a := 0; a < 3; a++ {
			vertices[i][a] = float32(center[a] + v[a])
		}
	}
}

func box(c, s [3]float64) ([]vec3, []uint16) {
	var v []vec3
	for _, z := range []float64{-1, 1} {
		for _, y := range []float64{-1, 1} {
			for _, x := range []float64{-1, 1} {
				v = append(v, vec3{float32(c[0] + x*s[0]/2), float32(c[1] + y*s[1]/2), float32(c[2] + z*s[2]/2)})
			}
		}
	}
	ix := []uint16{0, 2, 1, 1, 2, 3, 4, 5, 6, 5, 7, 6, 0, 1, 4, 1, 5, 4, 2, 6, 3, 3, 6, 7, 0, 4, 2, 2, 4, 6, 1, 3, 5, 3, 7, 5}
	return v, ix
}

// bevelBox emits six inset faces, twelve flat bevel strips and eight corner
// triangles. The fixed topology keeps rounded edges bounded and deterministic.
func bevelBox(c, s [3]float64, b float64) ([]vec3, []uint16) {
	h := [3]float64{s[0] / 2, s[1] / 2, s[2] / 2}
	var v []vec3
	var ix []uint16
	add := func(points ...[3]float64) {
		start := uint16(len(v))
		for _, p := range points {
			v = append(v, vec3{float32(c[0] + p[0]), float32(c[1] + p[1]), float32(c[2] + p[2])})
		}
		for i := 1; i < len(points)-1; i++ {
			ix = append(ix, start, start+uint16(i), start+uint16(i+1))
		}
	}
	// Each face is bounded away from the twelve edge bevels.
	for axis := 0; axis < 3; axis++ {
		u, w := (axis+1)%3, (axis+2)%3
		for _, sgn := range []float64{-1, 1} {
			p := [4][3]float64{}
			for i, uw := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
				p[i][axis] = sgn * h[axis]
				p[i][u] = uw[0] * (h[u] - b)
				p[i][w] = uw[1] * (h[w] - b)
			}
			add(p[:]...)
		}
	}
	// Edge strips join the two adjacent inset faces.
	for axis := 0; axis < 3; axis++ {
		u, w := (axis+1)%3, (axis+2)%3
		for _, su := range []float64{-1, 1} {
			for _, sw := range []float64{-1, 1} {
				p := [4][3]float64{}
				x0, x1 := -(h[axis] - b), h[axis]-b
				p[0][axis], p[1][axis], p[2][axis], p[3][axis] = x0, x1, x1, x0
				p[0][u], p[3][u] = su*(h[u]-b), su*(h[u]-b)
				p[1][u], p[2][u] = su*h[u], su*h[u]
				p[0][w], p[1][w] = sw*h[w], sw*h[w]
				p[2][w], p[3][w] = sw*(h[w]-b), sw*(h[w]-b)
				add(p[:]...)
			}
		}
	}
	// Corner caps close the remaining triangular gaps.
	for _, sx := range []float64{-1, 1} {
		for _, sy := range []float64{-1, 1} {
			for _, sz := range []float64{-1, 1} {
				add([3]float64{sx * h[0], sy * (h[1] - b), sz * (h[2] - b)}, [3]float64{sx * (h[0] - b), sy * h[1], sz * (h[2] - b)}, [3]float64{sx * (h[0] - b), sy * (h[1] - b), sz * h[2]})
			}
		}
	}
	return v, ix
}
func cylinder(c, s [3]float64) ([]vec3, []uint16) {
	const n = 16
	v := make([]vec3, 0, n*4+2)
	for _, z := range []float64{-1, 1} {
		for i := 0; i < n; i++ {
			a := 2 * math.Pi * float64(i) / n
			v = append(v, vec3{float32(c[0] + math.Cos(a)*s[0]/2), float32(c[1] + math.Sin(a)*s[1]/2), float32(c[2] + z*s[2]/2)})
		}
	}
	for _, z := range []float64{-1, 1} {
		for i := 0; i < n; i++ {
			a := 2 * math.Pi * float64(i) / n
			v = append(v, vec3{float32(c[0] + math.Cos(a)*s[0]/2), float32(c[1] + math.Sin(a)*s[1]/2), float32(c[2] + z*s[2]/2)})
		}
	}
	v = append(v, vec3{float32(c[0]), float32(c[1]), float32(c[2] - s[2]/2)}, vec3{float32(c[0]), float32(c[1]), float32(c[2] + s[2]/2)})
	ix := make([]uint16, 0, n*12)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		ix = append(ix, uint16(i), uint16(j), uint16(n+j), uint16(i), uint16(n+j), uint16(n+i), uint16(4*n), uint16(2*n+i), uint16(2*n+j), uint16(4*n+1), uint16(3*n+j), uint16(3*n+i))
	}
	return v, ix
}

type screenFrame struct {
	center, u, v, normal [3]float64
	width, height        float64
}

func screenPlane(r Recipe) screenFrame {
	var p Part
	for _, v := range r.Parts {
		if v.ID == r.Screen.PartID {
			p = v
			break
		}
	}
	axis, ua, va := 2, 0, 1
	us, vs, sign := 1.0, 1.0, 1.0
	switch r.Screen.Face {
	case "back:-z":
		sign = -1
		us = -1
	case "left:-x":
		axis, ua, va = 0, 2, 1
		sign = -1
		us = 1
	case "right:+x":
		axis, ua, va = 0, 2, 1
		us = -1
	case "top:+y":
		axis, ua, va = 1, 0, 2
		vs = -1
	case "bottom:-y":
		axis, ua, va = 1, 0, 2
		sign = -1
	}
	f := screenFrame{center: p.Center, width: p.Size[ua] * r.Screen.Size[0], height: p.Size[va] * r.Screen.Size[1]}
	f.normal[axis] = sign
	f.u[ua] = us
	f.v[va] = vs
	f.center[axis] += sign*p.Size[axis]/2 + sign*.02 // 0.02 mm stand-off, on the requested face
	f.center[ua] += us * (r.Screen.Center[0] - .5) * p.Size[ua]
	f.center[va] += vs * (r.Screen.Center[1] - .5) * p.Size[va]
	rotatePoint(&f.center, p.Center, p.Rotation)
	rotateVector(&f.u, p.Rotation)
	rotateVector(&f.v, p.Rotation)
	rotateVector(&f.normal, p.Rotation)
	turns := r.Screen.Rotation / 90 % 4
	for n := 0; n < turns; n++ {
		f.u, f.v = f.v, neg(f.u)
		f.width, f.height = f.height, f.width
	}
	return f
}
func plane(f screenFrame) []vec3 {
	var out []vec3
	for _, q := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
		p := [3]float64{}
		for i := 0; i < 3; i++ {
			p[i] = f.center[i] + q[0]*f.u[i]*f.width/2 + q[1]*f.v[i]*f.height/2
		}
		out = append(out, vec3{float32(p[0]), float32(p[1]), float32(p[2])})
	}
	return out
}

func rotatePoint(p *[3]float64, center [3]float64, rotation [3]int) {
	v := [3]float64{p[0] - center[0], p[1] - center[1], p[2] - center[2]}
	rotateVector(&v, rotation)
	for i := range p {
		p[i] = center[i] + v[i]
	}
}
func rotateVector(v *[3]float64, rotation [3]int) {
	for axis, deg := range rotation {
		for n := 0; n < deg/90; n++ {
			a, b := (axis+1)%3, (axis+2)%3
			v[a], v[b] = -v[b], v[a]
		}
	}
}
func neg(v [3]float64) [3]float64 { return [3]float64{-v[0], -v[1], -v[2]} }
func scaleVertices(v []vec3) {
	for i := range v {
		for axis := range v[i] {
			v[i][axis] *= .001
		}
	}
}
func orientTriangles(v []vec3, ix []uint16, inside [3]float64) {
	for i := 0; i+2 < len(ix); i += 3 {
		a, b, c := v[ix[i]], v[ix[i+1]], v[ix[i+2]]
		n := cross(sub(b, a), sub(c, a))
		center := [3]float64{(float64(a[0]) + float64(b[0]) + float64(c[0])) / 3, (float64(a[1]) + float64(b[1]) + float64(c[1])) / 3, (float64(a[2]) + float64(b[2]) + float64(c[2])) / 3}
		if dot(n, [3]float64{center[0] - inside[0], center[1] - inside[1], center[2] - inside[2]}) < 0 {
			ix[i+1], ix[i+2] = ix[i+2], ix[i+1]
		}
	}
}
func vertexNormals(v []vec3, ix []uint16) ([][3]float32, error) {
	n := make([][3]float64, len(v))
	for i := 0; i+2 < len(ix); i += 3 {
		a, b, c := v[ix[i]], v[ix[i+1]], v[ix[i+2]]
		face := cross(sub(b, a), sub(c, a))
		length := math.Sqrt(dot(face, face))
		if length == 0 || !finite(length) {
			return nil, fmt.Errorf("geometry: degenerate triangle")
		}
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				n[ix[i+j]][k] += face[k] / length
			}
		}
	}
	out := make([][3]float32, len(v))
	for i := range n {
		length := math.Sqrt(dot(n[i], n[i]))
		if length < 1e-12 {
			return nil, fmt.Errorf("geometry: vertex has no finite normal")
		}
		for k := 0; k < 3; k++ {
			out[i][k] = float32(n[i][k] / length)
		}
	}
	return out, nil
}

func prepareNormals(v []vec3, ix []uint16, uv []vec2, smooth bool) ([]vec3, []uint16, []vec2) {
	if smooth {
		return v, ix, uv
	}
	outV := make([]vec3, 0, len(ix))
	outI := make([]uint16, 0, len(ix))
	var outUV []vec2
	if len(uv) > 0 {
		outUV = make([]vec2, 0, len(ix))
	}
	for _, index := range ix {
		outI = append(outI, uint16(len(outV)))
		outV = append(outV, v[index])
		if len(uv) > 0 {
			outUV = append(outUV, uv[index])
		}
	}
	return outV, outI, outUV
}

// roundedBox creates a bounded 8-segment rounded cuboid by projecting a
// subdivided cube onto the Minkowski sum of its inset core and a sphere.
func roundedBox(c, s [3]float64, r float64, segments int) ([]vec3, []uint16) {
	h := [3]float64{s[0] / 2, s[1] / 2, s[2] / 2}
	var verts []vec3
	var indices []uint16
	lookup := map[[3]int64]uint16{}
	add := func(p [3]float64) uint16 {
		inner := [3]float64{}
		delta := [3]float64{}
		norm := 0.0
		for i := 0; i < 3; i++ {
			inner[i] = math.Max(-h[i]+r, math.Min(h[i]-r, p[i]))
			delta[i] = p[i] - inner[i]
			norm += delta[i] * delta[i]
		}
		length := math.Sqrt(norm)
		if length > 0 {
			for i := 0; i < 3; i++ {
				p[i] = inner[i] + delta[i]*r/length
			}
		}
		key := [3]int64{int64(math.Round(p[0] * 1e6)), int64(math.Round(p[1] * 1e6)), int64(math.Round(p[2] * 1e6))}
		if old, ok := lookup[key]; ok {
			return old
		}
		idx := uint16(len(verts))
		lookup[key] = idx
		verts = append(verts, vec3{float32(c[0] + p[0]), float32(c[1] + p[1]), float32(c[2] + p[2])})
		return idx
	}
	for axis := 0; axis < 3; axis++ {
		u, w := (axis+1)%3, (axis+2)%3
		for _, sign := range []float64{-1, 1} {
			grid := make([][]uint16, segments+1)
			for i := 0; i <= segments; i++ {
				grid[i] = make([]uint16, segments+1)
				for j := 0; j <= segments; j++ {
					p := [3]float64{}
					p[axis] = sign * h[axis]
					p[u] = -h[u] + 2*h[u]*float64(i)/float64(segments)
					p[w] = -h[w] + 2*h[w]*float64(j)/float64(segments)
					grid[i][j] = add(p)
				}
			}
			for i := 0; i < segments; i++ {
				for j := 0; j < segments; j++ {
					indices = append(indices, grid[i][j], grid[i+1][j], grid[i+1][j+1], grid[i][j], grid[i+1][j+1], grid[i][j+1])
				}
			}
		}
	}
	orientTriangles(verts, indices, c)
	return verts, indices
}
func sub(a, b vec3) [3]float64 {
	return [3]float64{float64(a[0] - b[0]), float64(a[1] - b[1]), float64(a[2] - b[2])}
}
func cross(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func dot(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func align4(b *bytes.Buffer) {
	for b.Len()%4 != 0 {
		b.WriteByte(0)
	}
}
func alignBytes(b *[]byte, v byte) {
	for len(*b)%4 != 0 {
		*b = append(*b, v)
	}
}
