package physicalruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"device-studio/internal/devices"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func TestAuthoredTelevisionRecipeIsFiniteAndPinned(t *testing.T) {
	data, digest, err := compileTelevisionAsset()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("authored television GLB SHA-256 %s (%d bytes)", digest, len(data))
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Nodes) != len(televisionRecipe().Parts) || inventory.TriangleCount == 0 || len(inventory.Extensions) != 0 {
		t.Fatalf("unexpected finite television inventory: nodes=%d triangles=%d extensions=%v", len(inventory.Nodes), inventory.TriangleCount, inventory.Extensions)
	}
	if televisionRecipe().Screen.PartID != "display" || televisionRecipe().Screen.Face != "front:+z" {
		t.Fatal("television app screen is not bound to the actual authored flat front face")
	}
}

func TestFamilyScreenUVVCorrectionsAreVersionedAndScoped(t *testing.T) {
	for _, test := range []struct {
		name, path, node string
	}{
		{name: "smart display", path: smartDisplayAssetFile, node: "MainDisplay"},
		{name: "vehicle center stack", path: vehicleStackAssetFile, node: "CenterDisplay"},
	} {
		source, err := starterAssets.ReadFile(test.path)
		if err != nil {
			t.Fatal(err)
		}
		before, err := screenPrimitiveCorners(source, test.node, 0)
		if err != nil {
			t.Fatalf("%s source screen: %v", test.name, err)
		}
		if before.top[1] < before.bottom[1] {
			t.Fatalf("%s source unexpectedly has upright UV orientation: top=%v bottom=%v", test.name, before.top, before.bottom)
		}
		derived, err := flipScreenUVV(source, test.node, 0)
		if err != nil {
			t.Fatalf("%s derivative: %v", test.name, err)
		}
		_, sourceBIN := testGLBChunks(t, source)
		_, derivedBIN := testGLBChunks(t, derived)
		if !bytes.HasPrefix(derivedBIN, sourceBIN) {
			t.Fatalf("%s derivative did not preserve the exact source BIN prefix", test.name)
		}
		after, err := screenPrimitiveCorners(derived, test.node, 0)
		if err != nil || after.top[1] >= after.bottom[1] {
			t.Fatalf("%s derivative must map top to lower V: corners=%+v err=%v", test.name, after, err)
		}
		if sha256Hex(derived) != map[string]string{"smart display": smartDisplayScreenV2SHA, "vehicle center stack": vehicleStackScreenV2SHA}[test.name] {
			t.Fatalf("%s versioned derivative digest changed", test.name)
		}
	}

	frame, err := starterAssets.ReadFile(digitalFrameSpec(false).file)
	if err != nil {
		t.Fatal(err)
	}
	frameCorners, err := screenPrimitiveCorners(frame, "Unprinted frame insert", 0)
	if err != nil || frameCorners.top[1] >= frameCorners.bottom[1] {
		t.Fatalf("digital frame front face already maps top to low V and must remain unchanged: corners=%+v err=%v", frameCorners, err)
	}

	tv, _, err := compileTelevisionAsset()
	if err != nil {
		t.Fatal(err)
	}
	tvCorners, err := screenPrimitiveCorners(tv, "display", 1)
	if err != nil || tvCorners.top[1] >= tvCorners.bottom[1] {
		t.Fatalf("television derivative must map top to lower V: corners=%+v err=%v", tvCorners, err)
	}
}

func TestRoundWatchV3ChangesOnlyPlanarSiliconeStrapNormals(t *testing.T) {
	source, err := starterAssets.ReadFile(roundWatchV2AssetFile)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := starterAssets.ReadFile(roundWatchV3AssetFile)
	if err != nil {
		t.Fatal(err)
	}
	if sha256Hex(derived) != roundWatchV3AssetSHA {
		t.Fatal("round-watch v3 asset digest changed")
	}
	sourceJSON, sourceBIN := testGLBChunks(t, source)
	derivedJSON, derivedBIN := testGLBChunks(t, derived)
	var src, dst struct {
		Nodes []struct {
			Name string `json:"name"`
			Mesh int    `json:"mesh"`
		} `json:"nodes"`
		Meshes []struct {
			Primitives []struct {
				Attributes map[string]int `json:"attributes"`
				Indices    int            `json:"indices"`
				Material   int            `json:"material"`
			} `json:"primitives"`
		} `json:"meshes"`
		Accessors []struct {
			BufferView    int    `json:"bufferView"`
			ByteOffset    int    `json:"byteOffset"`
			Count         int    `json:"count"`
			ComponentType int    `json:"componentType"`
			Type          string `json:"type"`
		} `json:"accessors"`
		BufferViews []struct {
			ByteOffset int `json:"byteOffset"`
			ByteLength int `json:"byteLength"`
		} `json:"bufferViews"`
	}
	if err := json.Unmarshal(sourceJSON, &src); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(derivedJSON, &dst); err != nil {
		t.Fatal(err)
	}
	accessorBytes := func(accessors []struct {
		BufferView    int    `json:"bufferView"`
		ByteOffset    int    `json:"byteOffset"`
		Count         int    `json:"count"`
		ComponentType int    `json:"componentType"`
		Type          string `json:"type"`
	}, views []struct {
		ByteOffset int `json:"byteOffset"`
		ByteLength int `json:"byteLength"`
	}, bin []byte, index int, elementBytes int) []byte {
		a, v := accessors[index], views[accessors[index].BufferView]
		start, end := v.ByteOffset+a.ByteOffset, v.ByteOffset+a.ByteOffset+a.Count*elementBytes
		if start < 0 || end > len(bin) {
			t.Fatalf("accessor %d exceeds BIN", index)
		}
		return bin[start:end]
	}
	for _, name := range []string{"lower charcoal silicone strap", "upper charcoal silicone strap"} {
		find := func(nodes []struct {
			Name string `json:"name"`
			Mesh int    `json:"mesh"`
		}, meshes []struct {
			Primitives []struct {
				Attributes map[string]int `json:"attributes"`
				Indices    int            `json:"indices"`
				Material   int            `json:"material"`
			} `json:"primitives"`
		}) (map[string]int, int, int) {
			for _, node := range nodes {
				if node.Name == name {
					p := meshes[node.Mesh].Primitives[0]
					return p.Attributes, p.Indices, p.Material
				}
			}
			return nil, -1, -1
		}
		srcAttr, srcIndices, srcMaterial := find(src.Nodes, src.Meshes)
		dstAttr, dstIndices, dstMaterial := find(dst.Nodes, dst.Meshes)
		if srcAttr == nil || dstAttr == nil || srcIndices != dstIndices || srcMaterial != dstMaterial || srcAttr["POSITION"] != dstAttr["POSITION"] || srcAttr["TEXCOORD_0"] != dstAttr["TEXCOORD_0"] || srcAttr["NORMAL"] != dstAttr["NORMAL"] {
			t.Fatalf("%s non-normal bindings changed", name)
		}
		if src.Accessors[srcAttr["POSITION"]].Count != dst.Accessors[dstAttr["POSITION"]].Count || src.Accessors[srcAttr["NORMAL"]].Count != 144 || dst.Accessors[dstAttr["NORMAL"]].Count != 144 {
			t.Fatalf("%s accessor topology changed", name)
		}
		for attr, stride := range map[string]int{"POSITION": 12, "TEXCOORD_0": 8} {
			if !bytes.Equal(accessorBytes(src.Accessors, src.BufferViews, sourceBIN, srcAttr[attr], stride), accessorBytes(dst.Accessors, dst.BufferViews, derivedBIN, dstAttr[attr], stride)) {
				t.Fatalf("%s %s bytes changed", name, attr)
			}
		}
		srcNormals := accessorBytes(src.Accessors, src.BufferViews, sourceBIN, srcAttr["NORMAL"], 12)
		dstNormals := accessorBytes(dst.Accessors, dst.BufferViews, derivedBIN, dstAttr["NORMAL"], 12)
		for i := 0; i < 144; i++ {
			for axis := 0; axis < 3; axis++ {
				got := math.Float32frombits(binary.LittleEndian.Uint32(dstNormals[i*12+axis*4:]))
				want := math.Float32frombits(binary.LittleEndian.Uint32(srcNormals[i*12+axis*4:]))
				if i < 36 {
					want = 0
					if axis == 2 {
						want = -1
					}
				} else if i >= 108 {
					want = 0
					if axis == 2 {
						want = 1
					}
				}
				if got != want {
					t.Fatalf("%s strap normal vertex %d axis %d=%v want %v", name, i, axis, got, want)
				}
			}
		}
	}
}

func TestVehicleStudioBackdropRemovesOnlyActiveTableAndWindowContext(t *testing.T) {
	source, err := starterAssets.ReadFile("assets/laptop-study-environment/laptop-study-environment-no-cup.glb")
	if err != nil {
		t.Fatal(err)
	}
	derived, err := starterAssets.ReadFile(vehicleStudioSceneFile)
	if err != nil {
		t.Fatal(err)
	}
	if sha256Hex(derived) != vehicleStudioSceneSHA {
		t.Fatal("vehicle studio scene digest changed")
	}
	sourceJSON, sourceBIN := testGLBChunks(t, source)
	derivedJSON, derivedBIN := testGLBChunks(t, derived)
	if !bytes.Equal(sourceBIN, derivedBIN) {
		t.Fatal("vehicle studio backdrop changed source binary geometry/material bytes")
	}
	var src, dst struct {
		Nodes []struct {
			Name string `json:"name"`
			Mesh *int   `json:"mesh"`
		} `json:"nodes"`
		Scenes []struct {
			Nodes []int `json:"nodes"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(sourceJSON, &src); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(derivedJSON, &dst); err != nil {
		t.Fatal(err)
	}
	if len(dst.Nodes) != len(src.Nodes) || len(dst.Scenes) != 1 || len(dst.Scenes[0].Nodes) != 1 {
		t.Fatal("vehicle studio derivative should preserve node records and activate only the wall")
	}
	active := dst.Scenes[0].Nodes[0]
	if active < 0 || active >= len(dst.Nodes) || dst.Nodes[active].Name != "RoomBackWall" {
		t.Fatalf("active studio context is not the neutral back wall: %d", active)
	}
	for _, node := range dst.Nodes {
		if node.Name == "TableTop" || node.Name == "TableLeg-NE" {
			if node.Mesh == nil {
				t.Fatalf("source table node %s lost its immutable geometry record", node.Name)
			}
		}
	}
}

type screenCorners struct{ top, bottom [2]float32 }

func screenPrimitiveCorners(data []byte, nodeName string, primitiveIndex int) (screenCorners, error) {
	jsonChunk, bin := testGLBChunksBytes(data)
	var doc map[string]any
	if err := json.Unmarshal(jsonChunk, &doc); err != nil {
		return screenCorners{}, err
	}
	index := func(value any) int { return int(value.(float64)) }
	nodes := doc["nodes"].([]any)
	var node map[string]any
	for _, raw := range nodes {
		candidate := raw.(map[string]any)
		if candidate["name"] == nodeName {
			node = candidate
			break
		}
	}
	if node == nil {
		return screenCorners{}, fmt.Errorf("node %q not found", nodeName)
	}
	meshes := doc["meshes"].([]any)
	mesh := meshes[index(node["mesh"])].(map[string]any)
	primitives := mesh["primitives"].([]any)
	primitive := primitives[primitiveIndex].(map[string]any)
	attributes := primitive["attributes"].(map[string]any)
	accessors, views := doc["accessors"].([]any), doc["bufferViews"].([]any)
	read := func(accessorIndex int, stride int) ([]float32, error) {
		accessor := accessors[accessorIndex].(map[string]any)
		view := views[index(accessor["bufferView"])].(map[string]any)
		start := intOrZero(view["byteOffset"]) + intOrZero(accessor["byteOffset"])
		count := index(accessor["count"])
		if start < 0 || start+count*stride > len(bin) {
			return nil, errors.New("accessor exceeds BIN")
		}
		values := make([]float32, count*stride/4)
		for i := range values {
			values[i] = math.Float32frombits(binary.LittleEndian.Uint32(bin[start+i*4 : start+i*4+4]))
		}
		return values, nil
	}
	positions, err := read(index(attributes["POSITION"]), 12)
	if err != nil {
		return screenCorners{}, err
	}
	uvs, err := read(index(attributes["TEXCOORD_0"]), 8)
	if err != nil || len(positions)/3 != len(uvs)/2 {
		return screenCorners{}, errors.New("screen position and UV accessor counts differ")
	}
	maxZ := float32(math.Inf(-1))
	for i := 0; i < len(positions); i += 3 {
		if positions[i+2] > maxZ {
			maxZ = positions[i+2]
		}
	}
	minY, maxY := float32(math.Inf(1)), float32(math.Inf(-1))
	for i := 0; i < len(positions); i += 3 {
		if positions[i+2] == maxZ {
			if positions[i+1] < minY {
				minY = positions[i+1]
			}
			if positions[i+1] > maxY {
				maxY = positions[i+1]
			}
		}
	}
	var out screenCorners
	for i := 0; i < len(positions); i += 3 {
		if positions[i+2] == maxZ && positions[i+1] == maxY {
			out.top = [2]float32{uvs[i/3*2], uvs[i/3*2+1]}
		}
		if positions[i+2] == maxZ && positions[i+1] == minY {
			out.bottom = [2]float32{uvs[i/3*2], uvs[i/3*2+1]}
		}
	}
	return out, nil
}

func intOrZero(value any) int {
	if value == nil {
		return 0
	}
	return int(value.(float64))
}

func testGLBChunksBytes(data []byte) ([]byte, []byte) {
	var jsonChunk, binChunk []byte
	for offset := 12; offset+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8
		if offset+size > len(data) {
			return nil, nil
		}
		if kind == 0x4e4f534a {
			jsonChunk = data[offset : offset+size]
		}
		if kind == 0x004e4942 {
			binChunk = data[offset : offset+size]
		}
		offset += size
	}
	return jsonChunk, binChunk
}

func TestRoundWatchDerivativePreservesSourceAndRemovesDialGraphics(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "..", "docs", "internal", "goal", "evidence", "E4", "free-models", "anonymous-candidates-20261020", "grab3d-smartwatch", "smartwatch.glb")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if digest := sha256.Sum256(source); hex.EncodeToString(digest[:]) != roundWatchSourceSHA {
		t.Fatalf("reviewed Grab3D source changed: %x", digest)
	}
	derived, err := starterAssets.ReadFile(roundWatchAssetFile)
	if err != nil {
		t.Fatal(err)
	}
	if digest := sha256.Sum256(derived); hex.EncodeToString(digest[:]) != roundWatchAssetSHA {
		t.Fatalf("pinned neutral derivative changed: %x", digest)
	}
	derivation, err := starterAssets.ReadFile("assets/roundwatch/roundwatch-source-derivative.json")
	if err != nil {
		t.Fatal(err)
	}
	if digest := sha256.Sum256(derivation); hex.EncodeToString(digest[:]) != "9b1a2eda89df387d0aed5c56f3de0e949b61917ac4300dbf32131f88c1ade391" {
		t.Fatalf("round-watch derivation manifest changed: %x", digest)
	}
	sourceJSON, sourceBIN := testGLBChunks(t, source)
	derivedJSON, derivedBIN := testGLBChunks(t, derived)
	type primitive struct {
		Attributes map[string]int `json:"attributes"`
	}
	type mesh struct {
		Name       string      `json:"name"`
		Primitives []primitive `json:"primitives"`
	}
	type accessor struct {
		BufferView    int    `json:"bufferView"`
		ByteOffset    int    `json:"byteOffset"`
		Count         int    `json:"count"`
		ComponentType int    `json:"componentType"`
		Type          string `json:"type"`
	}
	type view struct {
		ByteOffset int `json:"byteOffset"`
	}
	type gltf struct {
		Nodes       []json.RawMessage `json:"nodes"`
		Meshes      []mesh            `json:"meshes"`
		Materials   []json.RawMessage `json:"materials"`
		Images      []json.RawMessage `json:"images"`
		Accessors   []accessor        `json:"accessors"`
		BufferViews []view            `json:"bufferViews"`
	}
	var original, output gltf
	if err := json.Unmarshal(sourceJSON, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(derivedJSON, &output); err != nil {
		t.Fatal(err)
	}
	if len(original.Meshes) != 2 || len(original.Meshes[1].Primitives) != 5 || len(output.Meshes) != 2 || len(output.Meshes[1].Primitives) != 1 || output.Meshes[1].Name != "smartwatch_display_glass" {
		t.Fatalf("derivative must remove only the four dial-overlay primitives: source=%+v derived=%+v", original.Meshes, output.Meshes)
	}
	if !reflect.DeepEqual(original.Nodes, output.Nodes) || !reflect.DeepEqual(original.Meshes[0], output.Meshes[0]) || !reflect.DeepEqual(original.Meshes[1].Primitives[0], output.Meshes[1].Primitives[0]) || !reflect.DeepEqual(original.Materials, output.Materials) || !reflect.DeepEqual(original.Images, output.Images) {
		t.Fatal("derivative changed source hierarchy, body geometry/material, retained display geometry/material, or embedded textures")
	}
	if len(sourceBIN) != len(derivedBIN) {
		t.Fatal("derivative changed the source BIN chunk length")
	}
	uvAccessorIndex := original.Meshes[1].Primitives[0].Attributes["TEXCOORD_0"]
	uv := original.Accessors[uvAccessorIndex]
	if uv.ComponentType != 5126 || uv.Type != "VEC2" {
		t.Fatalf("unexpected source display UV accessor: %+v", uv)
	}
	start := original.BufferViews[uv.BufferView].ByteOffset + uv.ByteOffset
	end := start + uv.Count*8
	if start < 0 || end > len(sourceBIN) {
		t.Fatalf("source UV accessor falls outside BIN chunk: [%d,%d)/%d", start, end, len(sourceBIN))
	}
	if !bytes.Equal(sourceBIN[:start], derivedBIN[:start]) || !bytes.Equal(sourceBIN[end:], derivedBIN[end:]) {
		t.Fatal("derivative changed BIN bytes outside the separate display UV accessor")
	}
	if bytes.Equal(sourceBIN[start:end], derivedBIN[start:end]) {
		t.Fatal("derivative did not normalize the display UV accessor")
	}
}

func testGLBChunks(t *testing.T, data []byte) ([]byte, []byte) {
	t.Helper()
	if len(data) < 20 || binary.LittleEndian.Uint32(data[:4]) != 0x46546c67 || binary.LittleEndian.Uint32(data[4:8]) != 2 || int(binary.LittleEndian.Uint32(data[8:12])) != len(data) {
		t.Fatal("invalid GLB header")
	}
	var jsonChunk, binChunk []byte
	for offset := 12; offset+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		kind := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8
		if size < 0 || offset+size > len(data) {
			t.Fatal("GLB chunk exceeds file bounds")
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
		t.Fatal(fmt.Sprintf("GLB chunks missing: JSON=%d BIN=%d", len(jsonChunk), len(binChunk)))
	}
	return jsonChunk, binChunk
}

func TestRoundWatchAndTelevisionStartersAreOptInSourceBoundAndReplayable(t *testing.T) {
	ctx := context.Background()
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema),
		database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema),
		database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema),
	}
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(c context.Context, pool *sql.DB) error { return database.EnsureSchemas(c, pool, providers...) })
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "primary-data")})
	const lease = "roundwatch-television-replay-lease"
	if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(t.TempDir(), "leased-data")}, lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots(lease); _ = db.ClearTestPool(lease) })
	repo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(repo, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	media := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: repo.(projects.SourceMediaRepository), Store: mediaStore})
	owners, err := New(db, roots, media)
	if err != nil {
		t.Fatal(err)
	}
	testCtx := database.WithTestMode(ctx)
	if _, err := owners.InitializeReviewedStarter(testCtx); err != nil {
		t.Fatalf("initialize ordinary starter: %v", err)
	}
	for _, id := range []string{roundWatchDeviceID, televisionDeviceID} {
		if _, err := owners.Devices.GetHead(testCtx, id); !errors.Is(err, devices.ErrRevisionNotFound) {
			t.Fatalf("normal starter unexpectedly published opt-in candidate %q: %v", id, err)
		}
	}
	for _, id := range []string{"generic-round-smartwatch-source", "generic-television-concept"} {
		first, err := owners.InitializePhysicalStarter(testCtx, id)
		if err != nil {
			t.Fatalf("initialize %s: %v", id, err)
		}
		second, err := owners.InitializePhysicalStarter(testCtx, id)
		if err != nil {
			t.Fatalf("replay %s: %v", id, err)
		}
		if first.SHA256 != second.SHA256 {
			t.Fatalf("%s preset replay changed immutable bytes", id)
		}
	}
	watch, err := owners.Devices.GetHead(testCtx, roundWatchDeviceID)
	if err != nil {
		t.Fatal(err)
	}
	if watch.State != devices.DeviceRevisionCandidate || watch.Candidate.Definition.IdentityStatus != devices.IdentityGeneric || watch.Candidate.Definition.Qualification != "unqualified" || watch.Candidate.Definition.Displays[0].ActiveArea.Shape != "circle" || watch.Candidate.Definition.Displays[0].ActiveArea.Status != "candidate_only" {
		t.Fatalf("round-watch candidate overclaims identity/qualification or lacks its round content mask: %+v", watch.Candidate.Definition)
	}
	if !strings.Contains(watch.Candidate.Definition.Provenance[0].Qualifier, "39 x 66 x 63 mm") || !strings.Contains(watch.Candidate.Definition.Provenance[0].Qualifier, "36 x 43 mm case") {
		t.Fatal("creator-declared closed-strap and case size claims were not retained explicitly")
	}
	watchGLB, err := starterAssets.ReadFile(roundWatchAssetFile)
	if err != nil {
		t.Fatal(err)
	}
	if sha256Hex(watchGLB) != roundWatchAssetSHA {
		t.Fatal("round-watch embedded derivative hash changed")
	}
	if _, err := rendering.ResolvePhysicalScreenGeometry(watchGLB, watch.Candidate, roundWatchDisplayID); err != nil {
		t.Fatalf("round-watch screen cannot receive app content: %v", err)
	}
	television, err := owners.Devices.GetHead(testCtx, televisionDeviceID)
	if err != nil {
		t.Fatal(err)
	}
	definition := television.Candidate.Definition
	if television.State != devices.DeviceRevisionCandidate || definition.IdentityStatus != devices.IdentityConcept || definition.Qualification != "unqualified" || definition.Kind != "generic_television_concept" || len(definition.ModelCandidates) != 1 || definition.ModelCandidates[0].License != "AGPL-3.0-only" {
		t.Fatalf("television candidate overclaims identity/qualification or lost authored provenance: %+v", definition)
	}
	if len(definition.SupportedPoses) != 2 || definition.SupportedPoses[0].Name != "stand-supported-tabletop-landscape" || definition.SupportedPoses[1].Name != "rear-mount-wall-supported-landscape" || len(definition.Supports) != 2 || definition.Supports[0].Status != "candidate_only" || definition.Supports[1].Status != "candidate_only" {
		t.Fatalf("television stand/wall poses are not both explicitly candidate-only: poses=%+v supports=%+v", definition.SupportedPoses, definition.Supports)
	}
	if len(definition.Supports[1].Evidence) != 1 || !strings.Contains(definition.Supports[1].Evidence[0].Qualifier, "not VESA asserted") {
		t.Fatal("television wall-mount pattern is not explicitly unverified")
	}
	tvGLB, digest, err := compileTelevisionAsset()
	if err != nil {
		t.Fatal(err)
	}
	if television.Candidate.Geometry.SHA256 != digest {
		t.Fatalf("television candidate is not bound to compiled authored GLB: %s != %s", television.Candidate.Geometry.SHA256, digest)
	}
	if _, err := rendering.ResolvePhysicalScreenGeometry(tvGLB, television.Candidate, televisionDisplayID); err != nil {
		t.Fatalf("television screen cannot receive app content: %v", err)
	}
	if len(definition.Provenance) < 2 || definition.Provenance[1].Kind != "comparison_reference" || !strings.Contains(definition.Provenance[1].Qualifier, "comparison only") {
		t.Fatalf("manufacturer dimensions were not kept as comparison-only provenance: %+v", definition.Provenance)
	}
}
