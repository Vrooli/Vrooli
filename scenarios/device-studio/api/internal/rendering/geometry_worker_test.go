package rendering

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"device-studio/internal/devices"
	"device-studio/internal/geometry"
	"device-studio/internal/library"
	"device-studio/internal/physical"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type geometryGatewayFake struct {
	request  *inferencev1.RunRequest
	response *inferencev1.RunResponse
}

func (f *geometryGatewayFake) Run(_ context.Context, r *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.request = r.Msg
	return connect.NewResponse(f.response), nil
}

func TestGeometryWorkerCallsGatewayOnceLocalOnlyAndWritesInspectedGLB(t *testing.T) {
	workspace := t.TempDir()
	image := []byte("pinned-user-image")
	sha := sha256.Sum256(image)
	imageSHA := hex.EncodeToString(sha[:])
	if err := os.WriteFile(filepath.Join(workspace, "reference-0.jpg"), image, 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	exeSHA := sha256.Sum256(exe)
	r := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{{ID: "housing", Kind: "rounded_box", Role: "housing", Center: [3]float64{}, Size: [3]float64{78, 148, 10}, Bevel: 1, Material: "matte_polymer"}, {ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5.2}, Size: [3]float64{64, 124, 1}, Material: "glass"}}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	recipe, _ := json.Marshal(r)
	m := GeometryWorkerManifest{Version: 1, JobID: "job-1", AttemptID: "attempt-1", InputFingerprint: hex.EncodeToString(make([]byte, 32)), WorkerSHA256: hex.EncodeToString(exeSHA[:]), Brief: "a compact field device", Envelope: [3]float64{80, 150, 12}, Images: []GeometryImageInput{{Name: "reference-0.jpg", SHA256: imageSHA, MediaType: "image/jpeg", Width: 1, Height: 1, SizeBytes: int64(len(image))}}}
	manifest, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(workspace, GeometryManifestName), append(manifest, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := &geometryGatewayFake{response: &inferencev1.RunResponse{ValueJson: string(recipe), Validated: true, Provider: "ollama", Model: "local-model", ProviderCallsMade: 1, Applied: geometryAppliedSettings(), Usage: &inferencev1.Usage{InputTokens: 12, OutputTokens: 24, InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}}}
	if err := RunGeometryManifest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake); err != nil {
		t.Fatal(err)
	}
	if fake.request.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY || fake.request.GetRole() != "extract.structured" || fake.request.GetMaxProviderCalls() != 1 || fake.request.GetMaxOutputTokens() != 4096 {
		t.Fatalf("unsafe or unbounded Gateway request: %+v", fake.request)
	}
	if len(fake.request.GetAttachments()) != 1 || string(fake.request.GetAttachments()[0].GetInlineBytes()) != string(image) {
		t.Fatal("worker did not send exact pinned image bytes")
	}
	glb, err := os.ReadFile(filepath.Join(workspace, GeometryOutputName))
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := InspectPhysicalGLB(glb)
	if err != nil {
		t.Fatalf("compiled output failed shared physical inspection: %v", err)
	}
	if inventory.TriangleCount == 0 || inventory.TriangleCount > geometry.MaxTriangles {
		t.Fatalf("compiled topology must remain non-empty and within the geometry budget: %d", inventory.TriangleCount)
	}
	identity := physical.TRS{Rotation: physical.QuatXYZW{W: 1}, Scale: physical.Vec3{X: 1, Y: 1, Z: 1}}
	primitive, material := uint32(1), uint32(2)
	candidate := devices.DeviceCandidate{
		Geometry:        physical.ArtifactRef{Role: library.RoleGeometryGLB, SizeBytes: int64(len(glb)), SHA256: workerDigest(glb)},
		Parts:           []devices.SemanticPart{{ID: "housing", Role: "housing", NodeIndex: 0, NodeName: "housing", LocalFrame: identity}, {ID: "display", Role: "display", NodeIndex: 1, NodeName: "display", LocalFrame: identity}},
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: "front", PartID: "display", LocalFrame: identity, ScreenPrimitiveIndex: &primitive, ScreenMaterialIndex: &material}},
	}
	screen := PhysicalScreenInput{DisplayID: "front", UVSet: "TEXCOORD_0", MappingProfile: PhysicalScreenMappingProfile, PrimitiveIndex: primitive, MaterialSlot: material}
	if _, err := ValidateDevicePhysicalGLB(glb, candidate, screen); err != nil {
		t.Fatalf("generated display node/UV primitive must satisfy the shared candidate screen binding owner: %v", err)
	}
	mapping, err := ResolvePhysicalScreenGeometry(glb, candidate, "front")
	if err != nil || mapping.NodeIndex != 1 || mapping.PrimitiveIndex != primitive || mapping.MaterialIndex != material {
		t.Fatalf("generated screen did not resolve through the candidate selector: %+v %v", mapping, err)
	}
	var receipt GeometryWorkerReceipt
	data, err := os.ReadFile(filepath.Join(workspace, GeometryReceiptName))
	if err != nil || json.Unmarshal(data, &receipt) != nil {
		t.Fatalf("worker receipt missing or malformed: %v", err)
	}
	if receipt.Version != 2 || receipt.ProviderCalls != 1 || receipt.OutputSHA256 != workerDigest(glb) || receipt.Applied.GetMaxOutputTokensEffective() != 4096 || receipt.Applied.TemperatureSent == nil || *receipt.Applied.TemperatureSent != 0.25 || receipt.InputProvenance != inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE {
		t.Fatal("receipt does not bind the one local attempt and exact output")
	}
	if receipt.InferenceOutcomeSHA256 != workerDigest(mustRead(t, filepath.Join(workspace, GeometryInferenceOutcomeName))) {
		t.Fatal("receipt does not bind the typed inference outcome")
	}
}

func TestGeometryWorkerDoesNotRepairInvalidProviderRecipe(t *testing.T) {
	workspace := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(exe)
	imageDigest := sha256.Sum256([]byte("x"))
	m := GeometryWorkerManifest{Version: 1, JobID: "j", AttemptID: "a", InputFingerprint: hex.EncodeToString(make([]byte, 32)), WorkerSHA256: hex.EncodeToString(digest[:]), Brief: "device", Envelope: [3]float64{80, 150, 12}, Images: []GeometryImageInput{{Name: "ref.png", SHA256: hex.EncodeToString(imageDigest[:]), MediaType: "image/png", Width: 1, Height: 1, SizeBytes: 1}}}
	manifest, _ := json.Marshal(m)
	_ = os.WriteFile(filepath.Join(workspace, "ref.png"), []byte("x"), 0o600)
	_ = os.WriteFile(filepath.Join(workspace, GeometryManifestName), append(manifest, '\n'), 0o600)
	fake := &geometryGatewayFake{response: &inferencev1.RunResponse{ValueJson: `{"wrong":true}`, Validated: false, ProviderCallsMade: 1, Applied: geometryAppliedSettings(), Usage: &inferencev1.Usage{InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}}}
	if err := RunGeometryManifest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake); err == nil {
		t.Fatal("invalid recipe must fail without hidden retry")
	}
	if fake.request.GetMaxProviderCalls() != 1 {
		t.Fatal("request did not enforce one total provider call")
	}
	if _, err := os.Stat(filepath.Join(workspace, GeometryOutputName)); !os.IsNotExist(err) {
		t.Fatal("invalid model result must not produce geometry")
	}
	var outcome GeometryInferenceOutcome
	data, err := os.ReadFile(filepath.Join(workspace, GeometryInferenceOutcomeName))
	if err != nil || json.Unmarshal(data, &outcome) != nil || outcome.Validated || outcome.ValueSHA256 != workerDigest([]byte(`{"wrong":true}`)) {
		t.Fatalf("rejected typed response must leave bounded outcome evidence: %v", err)
	}
}

func TestGeometryWorkerRefusesCorrectionWithoutParentLineage(t *testing.T) {
	workspace := t.TempDir()
	corrective := geometryRecipeJSON(t, "correct")
	_, fake := geometryWorkerFixture(t, workspace, string(corrective))
	if err := RunGeometryManifest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake); err == nil {
		t.Fatal("the construction endpoint must not persist a correction without pinned parent lineage")
	}
	if _, err := os.Stat(filepath.Join(workspace, GeometryInferenceOutcomeName)); err != nil {
		t.Fatal("typed response outcome should survive correction refusal")
	}
}

func geometryWorkerFixture(t *testing.T, workspace, recipe string) (GeometryWorkerManifest, *geometryGatewayFake) {
	t.Helper()
	image := []byte("x")
	imageDigest := sha256.Sum256(image)
	if err := os.WriteFile(filepath.Join(workspace, "ref.png"), image, 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(exe)
	m := GeometryWorkerManifest{Version: 1, JobID: "j", AttemptID: "a", InputFingerprint: hex.EncodeToString(make([]byte, 32)), WorkerSHA256: hex.EncodeToString(digest[:]), Brief: "device", Envelope: [3]float64{80, 150, 12}, Images: []GeometryImageInput{{Name: "ref.png", SHA256: hex.EncodeToString(imageDigest[:]), MediaType: "image/png", Width: 1, Height: 1, SizeBytes: 1}}}
	manifest, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, GeometryManifestName), append(manifest, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	var parsed inferencev1.RunResponse
	parsed.ValueJson, parsed.Validated, parsed.Provider, parsed.Model, parsed.ProviderCallsMade = recipe, true, "ollama", "local", 1
	parsed.Applied, parsed.Usage = geometryAppliedSettings(), unknownGatewayUsage()
	return m, &geometryGatewayFake{response: &parsed}
}

func geometryRecipeJSON(t *testing.T, operation string) []byte {
	t.Helper()
	recipe := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: operation, Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{{ID: "housing", Kind: "box", Role: "housing", Center: [3]float64{}, Size: [3]float64{78, 148, 10}, Material: "matte_polymer"}, {ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5}, Size: [3]float64{64, 124, 1}, Material: "glass"}}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	data, err := json.Marshal(recipe)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func geometryAppliedSettings() *sharedv1.AppliedSettings {
	temperature := 0.25
	return &sharedv1.AppliedSettings{TemperatureSent: &temperature, TemperatureSupport: sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN, MaxOutputTokensEffective: 4096, MaxOutputTokensSource: sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST}
}

func unknownGatewayUsage() *inferencev1.Usage {
	return &inferencev1.Usage{InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
