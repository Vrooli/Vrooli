package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"device-studio/internal/geometry"
	"github.com/vrooli/api-core/discovery"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	GeometryManifestName         = "geometry-manifest.json"
	GeometryRecipeName           = "geometry-recipe.json"
	GeometryOutputName           = "geometry-output.glb"
	GeometryReceiptName          = "geometry-receipt.json"
	GeometryInferenceOutcomeName = "geometry-inference-outcome.json"
	geometryImageTotalMax        = 10 << 20
	geometryWorkerTimeout        = 4 * time.Minute
)

type GeometryImageInput struct {
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	MediaType string `json:"media_type"`
	Width     uint32 `json:"width"`
	Height    uint32 `json:"height"`
	SizeBytes int64  `json:"size_bytes"`
}
type GeometryWorkerManifest struct {
	Version          int                  `json:"version"`
	JobID            string               `json:"job_id"`
	AttemptID        string               `json:"attempt_id"`
	InputFingerprint string               `json:"input_fingerprint"`
	WorkerSHA256     string               `json:"worker_sha256"`
	Brief            string               `json:"brief"`
	Envelope         [3]float64           `json:"envelope_mm"`
	Images           []GeometryImageInput `json:"images"`
}
type GeometryWorkerReceipt struct {
	Version                int                         `json:"version"`
	JobID                  string                      `json:"job_id"`
	AttemptID              string                      `json:"attempt_id"`
	InputFingerprint       string                      `json:"input_fingerprint"`
	WorkerSHA256           string                      `json:"worker_sha256"`
	RecipeSHA256           string                      `json:"recipe_sha256"`
	OutputSHA256           string                      `json:"output_sha256"`
	OutputBytes            int                         `json:"output_bytes"`
	Provider               string                      `json:"provider,omitempty"`
	Model                  string                      `json:"model,omitempty"`
	ProviderCalls          int32                       `json:"provider_calls_made"`
	InputTokens            int64                       `json:"input_tokens,omitempty"`
	OutputTokens           int64                       `json:"output_tokens,omitempty"`
	CostMicros             int64                       `json:"cost_micros,omitempty"`
	CostProvenance         inferencev1.UsageProvenance `json:"cost_provenance"`
	InputProvenance        inferencev1.UsageProvenance `json:"input_tokens_provenance"`
	OutputProvenance       inferencev1.UsageProvenance `json:"output_tokens_provenance"`
	Applied                *sharedv1.AppliedSettings   `json:"applied_settings"`
	InferenceOutcomeSHA256 string                      `json:"inference_outcome_sha256"`
	CompletedAt            time.Time                   `json:"completed_at"`
}

// GeometryInferenceOutcome records bounded metadata from any typed response,
// including rejected recipes. It deliberately excludes prompts and values.
type GeometryInferenceOutcome struct {
	Version          int                            `json:"version"`
	JobID            string                         `json:"job_id"`
	AttemptID        string                         `json:"attempt_id"`
	InputFingerprint string                         `json:"input_fingerprint"`
	WorkerSHA256     string                         `json:"worker_sha256"`
	Provider         string                         `json:"provider,omitempty"`
	Model            string                         `json:"model,omitempty"`
	ProviderCalls    int32                          `json:"provider_calls_made"`
	Applied          *sharedv1.AppliedSettings      `json:"applied_settings,omitempty"`
	Usage            *inferencev1.Usage             `json:"usage"`
	Validated        bool                           `json:"validated"`
	ErrorCode        inferencev1.InferenceErrorCode `json:"error_code"`
	ErrorConstruct   string                         `json:"error_construct,omitempty"`
	ValueSHA256      string                         `json:"value_sha256"`
	ValueBytes       int                            `json:"value_bytes"`
	ReceivedAt       time.Time                      `json:"received_at"`
}

type geometryInference interface {
	Run(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}

func IsGeometryWorkerInvocation(args []string) bool {
	return len(args) > 0 && args[0] == "geometry-construction-worker"
}
func RunGeometryWorkerCommand(args []string) error {
	if len(args) != 3 || args[0] != "geometry-construction-worker" || args[1] != "--manifest" {
		return errors.New("geometry-construction-worker requires exactly --manifest <absolute attempt manifest>")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return RunGeometryManifest(ctx, args[2], nil)
}

// RunGeometryManifest performs one disclosed local-only inference request,
// validates its finite recipe, compiles GLB bytes and writes a verifiable receipt.
func RunGeometryManifest(ctx context.Context, manifestPath string, client geometryInference) error {
	if ctx == nil || !filepath.IsAbs(manifestPath) || filepath.Clean(manifestPath) != manifestPath || filepath.Base(manifestPath) != GeometryManifestName {
		return errors.New("geometry worker requires a fixed manifest in its owned workspace")
	}
	ctx, cancel := context.WithTimeout(ctx, geometryWorkerTimeout)
	defer cancel()
	workspace := filepath.Dir(manifestPath)
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return err
	}
	defer root.Close()
	data, err := readGeometryFile(ctx, root, GeometryManifestName, 64<<10)
	if err != nil {
		return err
	}
	var m GeometryWorkerManifest
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err = d.Decode(&m); err != nil {
		return err
	}
	canonical, _ := json.Marshal(m)
	if !bytes.Equal(data, append(canonical, '\n')) || m.Version != 1 || m.JobID == "" || m.AttemptID == "" || !validWorkerDigest(m.InputFingerprint) || !validWorkerDigest(m.WorkerSHA256) || len(m.Images) < 1 || len(m.Images) > 4 || len(m.Brief) > 4000 {
		return errors.New("geometry manifest is not canonical or exceeds fixed bounds")
	}
	if err := verifyCurrentExecutable(m.WorkerSHA256); err != nil {
		return err
	}
	if client == nil {
		base, resolveErr := discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		if resolveErr != nil || base == "" {
			return errors.New("AI Gateway is unavailable for local-only creation")
		}
		client = inferenceconnect.NewInferenceServiceClient(&http.Client{Timeout: geometryWorkerTimeout}, base)
	}
	attachments := make([]*sharedv1.Attachment, 0, len(m.Images))
	var total int64
	for _, im := range m.Images {
		if filepath.Base(im.Name) != im.Name || !validWorkerDigest(im.SHA256) || im.SizeBytes <= 0 || im.Width == 0 || im.Height == 0 || im.MediaType != "image/png" && im.MediaType != "image/jpeg" {
			return errors.New("geometry reference manifest is invalid")
		}
		total += im.SizeBytes
		if total > geometryImageTotalMax {
			return errors.New("geometry references exceed total size bound")
		}
		imageBytes, readErr := readGeometryFile(ctx, root, im.Name, geometryImageTotalMax)
		if readErr != nil {
			return readErr
		}
		if int64(len(imageBytes)) != im.SizeBytes || workerDigest(imageBytes) != im.SHA256 {
			return errors.New("geometry reference bytes differ from pinned digest")
		}
		attachments = append(attachments, &sharedv1.Attachment{Modality: sharedv1.Modality_MODALITY_IMAGE, MediaType: im.MediaType, Width: im.Width, Height: im.Height, Bytes: uint64(len(imageBytes)), Payload: &sharedv1.Attachment_InlineBytes{InlineBytes: imageBytes}})
	}
	schema := geometry.SchemaJSON()
	source := fmt.Sprintf("Create a device geometry recipe. Product brief: %s\nLocked outer envelope (millimeters, x width/y height/z depth): %.4f x %.4f x %.4f. Return only the schema-conforming recipe. Do not change the locked envelope.", m.Brief, m.Envelope[0], m.Envelope[1], m.Envelope[2])
	request := &inferencev1.RunRequest{Source: source, SchemaJson: schema, Instruction: "Use the attached user-selected project reference images as visual evidence. Separate observed facts from uncertain inference. Output finite part geometry and an independently mapped screen. Use only the locked dimensions.", Role: "extract.structured", Attachments: attachments, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, MaxOutputTokens: 4096, MaxProviderCalls: 1}
	wrapped, callErr := client.Run(ctx, connect.NewRequest(request))
	var response *inferencev1.RunResponse
	if wrapped != nil {
		response = wrapped.Msg
	}
	var outcome *GeometryInferenceOutcome
	var outcomeSHA string
	if response != nil {
		outcome = geometryOutcome(m, response, time.Now().UTC())
		outcomeBytes, marshalErr := json.Marshal(outcome)
		if marshalErr != nil {
			return marshalErr
		}
		outcomeBytes = append(outcomeBytes, '\n')
		if writeErr := writeGeometryFile(root, GeometryInferenceOutcomeName, outcomeBytes); writeErr != nil {
			return writeErr
		}
		outcomeSHA = workerDigest(outcomeBytes)
	}
	if callErr != nil {
		return fmt.Errorf("local AI Gateway call failed: %w", callErr)
	}
	if response == nil || response.GetError() != nil || !response.GetValidated() || response.GetProviderCallsMade() > 1 {
		return errors.New("local AI Gateway could not return one validated bounded recipe")
	}
	if response.GetProviderCallsMade() != 1 {
		return errors.New("local AI Gateway did not confirm a provider dispatch attempt")
	}
	if response.GetApplied() == nil || !validAppliedSettings(response.GetApplied()) || outcomeSHA == "" {
		return errors.New("local AI Gateway response omitted trustworthy applied settings or outcome evidence")
	}
	if !validGatewayUsage(response.GetUsage()) {
		return errors.New("local AI Gateway response omitted explicit usage provenance")
	}
	recipeBytes := []byte(response.GetValueJson())
	recipe, canonicalRecipe, err := geometry.Decode(recipeBytes)
	if err != nil {
		return err
	}
	if recipe.Operation != "construct" {
		return errors.New("geometry creation accepts construct recipes only; correction requires an explicit parent revision")
	}
	for i := range m.Envelope {
		if recipe.Envelope[i] != m.Envelope[i] {
			return errors.New("recipe changed a locked outer dimension")
		}
	}
	glb, err := geometry.Compile(recipe)
	if err != nil {
		return err
	}
	inventory, err := InspectPhysicalGLB(glb)
	if err != nil {
		return fmt.Errorf("compiled geometry failed physical GLB inspection: %w", err)
	}
	if inventory.TriangleCount > geometry.MaxTriangles || len(glb) > geometry.MaxGLBBytes {
		return errors.New("compiled geometry exceeds output bounds")
	}
	if err := writeGeometryFile(root, GeometryRecipeName, append(canonicalRecipe, '\n')); err != nil {
		return err
	}
	if err := writeGeometryFile(root, GeometryOutputName, glb); err != nil {
		return err
	}
	usage := response.GetUsage()
	r := GeometryWorkerReceipt{Version: 2, JobID: m.JobID, AttemptID: m.AttemptID, InputFingerprint: m.InputFingerprint, WorkerSHA256: m.WorkerSHA256, RecipeSHA256: workerDigest(canonicalRecipe), OutputSHA256: workerDigest(glb), OutputBytes: len(glb), Provider: response.GetProvider(), Model: response.GetModel(), ProviderCalls: response.GetProviderCallsMade(), InputTokens: usage.GetInputTokens(), OutputTokens: usage.GetOutputTokens(), CostMicros: usage.GetCostMicros(), InputProvenance: usage.GetInputTokensProvenance(), OutputProvenance: usage.GetOutputTokensProvenance(), CostProvenance: usage.GetCostProvenance(), Applied: response.GetApplied(), InferenceOutcomeSHA256: outcomeSHA, CompletedAt: time.Now().UTC()}
	receipt, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return writeGeometryFile(root, GeometryReceiptName, append(receipt, '\n'))
}

func geometryOutcome(m GeometryWorkerManifest, response *inferencev1.RunResponse, now time.Time) *GeometryInferenceOutcome {
	value := []byte(response.GetValueJson())
	errorCode := inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSPECIFIED
	errorConstruct := ""
	if response.GetError() != nil {
		errorCode = response.GetError().GetCode()
		errorConstruct = response.GetError().GetConstruct()
	}
	usage := response.GetUsage()
	if usage == nil {
		usage = &inferencev1.Usage{InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}
	}
	provider, model := response.GetProvider(), response.GetModel()
	if len(provider) > 256 {
		provider = ""
	}
	if len(model) > 256 {
		model = ""
	}
	if len(errorConstruct) > 256 {
		errorConstruct = ""
	}
	return &GeometryInferenceOutcome{Version: 1, JobID: m.JobID, AttemptID: m.AttemptID, InputFingerprint: m.InputFingerprint, WorkerSHA256: m.WorkerSHA256, Provider: provider, Model: model, ProviderCalls: response.GetProviderCallsMade(), Applied: response.GetApplied(), Usage: usage, Validated: response.GetValidated(), ErrorCode: errorCode, ErrorConstruct: errorConstruct, ValueSHA256: workerDigest(value), ValueBytes: len(value), ReceivedAt: now}
}

func validGatewayUsage(value *inferencev1.Usage) bool {
	if value == nil || value.GetInputTokens() < 0 || value.GetOutputTokens() < 0 || value.GetCostMicros() < 0 {
		return false
	}
	valid := func(v inferencev1.UsageProvenance) bool {
		return v >= inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN && v <= inferencev1.UsageProvenance_USAGE_PROVENANCE_MIXED
	}
	return valid(value.GetInputTokensProvenance()) && valid(value.GetOutputTokensProvenance()) && valid(value.GetCostProvenance())
}

func validAppliedSettings(value *sharedv1.AppliedSettings) bool {
	if value == nil || value.GetTemperatureSupport() < sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED || value.GetTemperatureSupport() > sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN || value.GetMaxOutputTokensSource() != sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST || value.GetMaxOutputTokensEffective() != 4096 || value.TemperatureSent != nil && (math.IsNaN(value.GetTemperatureSent()) || math.IsInf(value.GetTemperatureSent(), 0)) {
		return false
	}
	return true
}

func readGeometryFile(ctx context.Context, root *os.Root, name string, max int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > max {
		return nil, errors.New("geometry worker file is outside fixed bounds")
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(b)) > max {
		return nil, errors.New("geometry worker file read exceeded fixed bounds")
	}
	return b, nil
}
func writeGeometryFile(root *os.Root, name string, data []byte) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := f.Write(data)
	if err != nil || n != len(data) {
		return errors.New("geometry worker could not write complete output")
	}
	return f.Sync()
}
func validWorkerDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func workerDigest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func verifyCurrentExecutable(expected string) error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	b, err := readRegularPath(path, 512<<20)
	if err != nil || workerDigest(b) != expected {
		return errors.New("geometry worker executable differs from immutable job closure")
	}
	return nil
}
