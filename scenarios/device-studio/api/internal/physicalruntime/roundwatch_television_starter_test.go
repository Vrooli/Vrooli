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
