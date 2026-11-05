package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	if err := runGeometryManifestForTest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake); err != nil {
		t.Fatal(err)
	}
	if fake.request.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY || fake.request.GetRole() != "extract.structured" || fake.request.GetMaxProviderCalls() != 1 || fake.request.GetMaxOutputTokens() != 4096 {
		t.Fatalf("unsafe or unbounded Gateway request: %+v", fake.request)
	}
	if !strings.Contains(fake.request.GetInstruction(), "[-W/2,+W/2]") || !strings.Contains(fake.request.GetInstruction(), "flat box display face") || !strings.Contains(fake.request.GetInstruction(), "stand-off") {
		t.Fatalf("fixed geometry instruction must state the locked envelope and flat screen face: %s", fake.request.GetInstruction())
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

func TestGeometryWorkerRejectsScriptAndArchiveManifestExtensionsBeforeDispatch(t *testing.T) {
	for _, field := range []string{"script", "archive_ref"} {
		t.Run(field, func(t *testing.T) {
			workspace := t.TempDir()
			image := []byte("pinned")
			imageSHA := sha256.Sum256(image)
			if err := os.WriteFile(filepath.Join(workspace, "reference.png"), image, 0o600); err != nil {
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
			manifest, err := json.Marshal(GeometryWorkerManifest{Version: 1, JobID: "job-1", AttemptID: "attempt-1", InputFingerprint: hex.EncodeToString(make([]byte, 32)), WorkerSHA256: hex.EncodeToString(exeSHA[:]), Brief: "bounded device", Envelope: [3]float64{80, 150, 12}, Images: []GeometryImageInput{{Name: "reference.png", SHA256: hex.EncodeToString(imageSHA[:]), MediaType: "image/png", Width: 1, Height: 1, SizeBytes: int64(len(image))}}})
			if err != nil {
				t.Fatal(err)
			}
			payload := append(append([]byte(nil), manifest[:len(manifest)-1]...), []byte(`,"`+field+`":"../../tmp/untrusted.py"}`)...)
			if err := os.WriteFile(filepath.Join(workspace, GeometryManifestName), append(payload, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			fake := &geometryGatewayFake{}
			err = runGeometryManifestForTest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake)
			if err == nil {
				t.Fatal("unsupported executable or archive manifest input was accepted")
			}
			if fake.request != nil {
				t.Fatal("invalid manifest reached the inference owner")
			}
			if _, err := os.Stat(filepath.Join(workspace, GeometryOutputName)); !os.IsNotExist(err) {
				t.Fatalf("invalid manifest produced geometry output: stat err=%v", err)
			}
		})
	}
}

func TestGeometryWorkerFundingRoutesSelectOneProviderAndRejectProviderDrift(t *testing.T) {
	for _, tc := range []struct {
		route, role, provider string
		profile               sharedv1.Profile
	}{
		{"", "extract.structured", "ollama", sharedv1.Profile_PROFILE_LOCAL_ONLY},
		{"local", "extract.structured", "ollama", sharedv1.Profile_PROFILE_LOCAL_ONLY},
		{"byok", "device.geometry.byok", "openrouter", sharedv1.Profile_PROFILE_REMOTE_ONLY},
		{"vrooli_credits", "device.geometry.vrooli_credits", "lpbs", sharedv1.Profile_PROFILE_REMOTE_ONLY},
	} {
		t.Run(tc.route, func(t *testing.T) {
			role, profile, provider, err := geometryInferenceRoute(tc.route)
			if err != nil || role != tc.role || profile != tc.profile || provider != tc.provider {
				t.Fatalf("route mapping=(%q,%v,%q,%v), want (%q,%v,%q,nil)", role, profile, provider, err, tc.role, tc.profile, tc.provider)
			}
		})
	}
	if _, _, _, err := geometryInferenceRoute("unexpected"); err == nil {
		t.Fatal("unknown route must fail before Gateway dispatch")
	}

	workspace := t.TempDir()
	image := []byte("pinned")
	imageSHA := sha256.Sum256(image)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	exeSHA := sha256.Sum256(exe)
	m := GeometryWorkerManifest{Version: 1, JobID: "job", AttemptID: "attempt", InputFingerprint: hex.EncodeToString(make([]byte, 32)), WorkerSHA256: hex.EncodeToString(exeSHA[:]), Brief: "device", Envelope: [3]float64{80, 150, 12}, FundingRoute: "local", Images: []GeometryImageInput{{Name: "ref.png", SHA256: hex.EncodeToString(imageSHA[:]), MediaType: "image/png", Width: 1, Height: 1, SizeBytes: int64(len(image))}}}
	manifest, _ := json.Marshal(m)
	_ = os.WriteFile(filepath.Join(workspace, "ref.png"), image, 0o600)
	_ = os.WriteFile(filepath.Join(workspace, GeometryManifestName), append(manifest, '\n'), 0o600)
	fake := &geometryGatewayFake{response: &inferencev1.RunResponse{ValueJson: `{}`, Validated: true, Provider: "openrouter", ProviderCallsMade: 1}}
	if err := runGeometryManifestForTest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("provider drift must fail after one recorded response, got %v", err)
	}
	if fake.request.GetRole() != "extract.structured" || fake.request.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY || fake.request.GetMaxProviderCalls() != 1 {
		t.Fatalf("unexpected route request: %+v", fake.request)
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
	fake := &geometryGatewayFake{response: &inferencev1.RunResponse{ValueJson: `{"wrong":true}`, Validated: true, ProviderCallsMade: 1, Applied: geometryAppliedSettings(), Usage: &inferencev1.Usage{InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}}}
	if err := runGeometryManifestForTest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake); err == nil {
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
	if err != nil || json.Unmarshal(data, &outcome) != nil || !outcome.Validated || outcome.ValueSHA256 != workerDigest([]byte(`{"wrong":true}`)) || outcome.ValueDisposition != "retained_exact" || outcome.ValueFile != GeometryResponseValueName {
		t.Fatalf("rejected typed response must leave bounded outcome evidence: %v", err)
	}
	if retained := mustRead(t, filepath.Join(workspace, GeometryResponseValueName)); string(retained) != `{"wrong":true}` {
		t.Fatal("rejected recipe value must be retained byte-for-byte before semantic validation")
	}
}

func TestGeometryWorkerRetainsTypedEnvelopeIssueForExplicitCorrection(t *testing.T) {
	workspace := t.TempDir()
	prior := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{{ID: "housing", Kind: "box", Role: "housing", Center: [3]float64{0, 370, 0}, Size: [3]float64{20, 20, 10}, Material: "matte_polymer"}, {ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5}, Size: [3]float64{10, 10, 1}, Material: "glass"}}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	rejected, err := json.Marshal(prior)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := geometry.Decode(rejected); err == nil {
		t.Fatal("fixture must violate the envelope")
	} else if issue, ok := geometry.IssueFromError(err); !ok || issue.Axis != "y" {
		t.Fatalf("fixture should produce typed envelope feedback: %+v %v", issue, err)
	}
	_, fake := geometryWorkerFixture(t, workspace, string(rejected))
	runErr := runGeometryManifestForTest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake)
	if runErr == nil {
		t.Fatal("a part outside the locked envelope must fail for review")
	}
	var outcome GeometryInferenceOutcome
	if err := json.Unmarshal(mustRead(t, filepath.Join(workspace, GeometryInferenceOutcomeName)), &outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.SemanticIssue == nil || outcome.SemanticIssue.Code != "part_outside_envelope" || outcome.SemanticIssue.PartID != "housing" || outcome.SemanticIssue.Axis != "y" || outcome.SemanticIssue.Message != "part exceeds the locked envelope" {
		t.Fatalf("expected bounded semantic feedback for the rejected part: %+v; outcome=%+v; err=%v", outcome.SemanticIssue, outcome, runErr)
	}
	if got := mustRead(t, filepath.Join(workspace, GeometryResponseValueName)); string(got) != string(rejected) || workerDigest(got) != outcome.ValueSHA256 {
		t.Fatal("correction evidence must retain the exact rejected recipe bytes and digest")
	}
	if _, err := os.Stat(filepath.Join(workspace, GeometryOutputName)); !os.IsNotExist(err) {
		t.Fatal("a rejected recipe must not produce a GLB")
	}
}

func TestGeometryWorkerOmitsOversizedResponseWithoutTruncating(t *testing.T) {
	workspace := t.TempDir()
	oversized := strings.Repeat("x", geometryResponseValueMax+1)
	_, fake := geometryWorkerFixture(t, workspace, oversized)
	if err := runGeometryManifestForTest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake); err == nil {
		t.Fatal("oversized invalid model result must fail recipe validation")
	}
	var outcome GeometryInferenceOutcome
	if err := json.Unmarshal(mustRead(t, filepath.Join(workspace, GeometryInferenceOutcomeName)), &outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.ValueBytes != len(oversized) || outcome.ValueSHA256 != workerDigest([]byte(oversized)) || outcome.ValueDisposition != "omitted_over_limit" || outcome.ValueFile != "" {
		t.Fatalf("oversized value must retain exact digest and size without claiming a file: %+v", outcome)
	}
	if _, err := os.Stat(filepath.Join(workspace, GeometryResponseValueName)); !os.IsNotExist(err) {
		t.Fatal("oversized response must not be truncated or retained as a partial file")
	}
}

func TestGeometryWorkerRefusesCorrectionWithoutParentLineage(t *testing.T) {
	workspace := t.TempDir()
	corrective := geometryRecipeJSON(t, "correct")
	_, fake := geometryWorkerFixture(t, workspace, string(corrective))
	if err := runGeometryManifestForTest(context.Background(), filepath.Join(workspace, GeometryManifestName), fake); err == nil {
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

func runGeometryManifestForTest(ctx context.Context, path string, client geometryInference) error {
	return runGeometryManifest(ctx, path, client, func(ctx context.Context, workspace string, manifest GeometryWorkerManifest) error {
		root, err := os.OpenRoot(workspace)
		if err != nil {
			return err
		}
		defer root.Close()
		data, err := readGeometryFile(ctx, root, GeometryRecipeName, geometry.MaxRecipeBytes)
		if err != nil {
			return err
		}
		if len(data) > 0 && data[len(data)-1] == '\n' {
			data = data[:len(data)-1]
		}
		recipe, canonical, err := geometry.Decode(data)
		if err != nil || !bytes.Equal(data, canonical) || recipe.Operation != "construct" {
			return errors.New("test compile received invalid recipe")
		}
		for i := range manifest.Envelope {
			if recipe.Envelope[i] != manifest.Envelope[i] {
				return errors.New("test compile changed locked envelope")
			}
		}
		glb, err := geometry.Compile(recipe)
		if err != nil {
			return err
		}
		return writeGeometryFile(root, GeometryOutputName, glb)
	})
}
