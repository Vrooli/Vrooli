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
	"strings"
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
	GeometryResponseValueName    = "geometry-response-value.json"
	geometryResponseValueMax     = 1 << 20
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
type GeometryCorrectionPrompt struct {
	Mode                   string                  `json:"mode,omitempty"`
	SourceJobID            string                  `json:"source_job_id"`
	SourceInputFingerprint string                  `json:"source_input_fingerprint"`
	SourceBriefSHA256      string                  `json:"source_brief_sha256"`
	RecipeSHA256           string                  `json:"recipe_sha256"`
	RecipeJSON             string                  `json:"recipe_json"`
	Issue                  *geometry.SemanticIssue `json:"issue,omitempty"`
	Feedback               string                  `json:"feedback"`
}
type GeometryWorkerManifest struct {
	Version          int                       `json:"version"`
	JobID            string                    `json:"job_id"`
	AttemptID        string                    `json:"attempt_id"`
	InputFingerprint string                    `json:"input_fingerprint"`
	WorkerSHA256     string                    `json:"worker_sha256"`
	Brief            string                    `json:"brief"`
	Envelope         [3]float64                `json:"envelope_mm"`
	FundingRoute     string                    `json:"funding_route,omitempty"`
	MaxCostMicros    int64                     `json:"max_cost_micros,omitempty"`
	Images           []GeometryImageInput      `json:"images"`
	Correction       *GeometryCorrectionPrompt `json:"correction,omitempty"`
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
	FundingRoute           string                      `json:"funding_route,omitempty"`
	ProviderCalls          int32                       `json:"provider_calls_made"`
	InputTokens            int64                       `json:"input_tokens,omitempty"`
	OutputTokens           int64                       `json:"output_tokens,omitempty"`
	CostMicros             int64                       `json:"cost_micros,omitempty"`
	CreditsCharged         *int64                      `json:"credits_charged,omitempty"`
	AccountingStatus       string                      `json:"accounting_status,omitempty"`
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
	Version            int                             `json:"version"`
	JobID              string                          `json:"job_id"`
	AttemptID          string                          `json:"attempt_id"`
	InputFingerprint   string                          `json:"input_fingerprint"`
	WorkerSHA256       string                          `json:"worker_sha256"`
	Provider           string                          `json:"provider,omitempty"`
	Model              string                          `json:"model,omitempty"`
	FundingRoute       string                          `json:"funding_route,omitempty"`
	ProviderCalls      int32                           `json:"provider_calls_made"`
	Applied            *sharedv1.AppliedSettings       `json:"applied_settings,omitempty"`
	SpendAuthorization *inferencev1.SpendAuthorization `json:"spend_authorization,omitempty"`
	Usage              *inferencev1.Usage              `json:"usage"`
	Validated          bool                            `json:"validated"`
	ErrorCode          inferencev1.InferenceErrorCode  `json:"error_code"`
	ErrorConstruct     string                          `json:"error_construct,omitempty"`
	ValueSHA256        string                          `json:"value_sha256"`
	ValueBytes         int                             `json:"value_bytes"`
	ValueDisposition   string                          `json:"value_disposition"`
	ValueFile          string                          `json:"value_file,omitempty"`
	SemanticIssue      *geometry.SemanticIssue         `json:"semantic_issue,omitempty"`
	ReceivedAt         time.Time                       `json:"received_at"`
}

type geometryInference interface {
	Run(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}

func IsGeometryWorkerInvocation(args []string) bool {
	return len(args) > 0 && args[0] == "geometry-construction-worker"
}

func IsGeometryCompileWorkerInvocation(args []string) bool {
	return len(args) > 0 && args[0] == "geometry-compile-worker"
}

// RunGeometryCompileWorkerCommand is the fixed deterministic stage that the
// geometry inference owner dispatches through workspace-sandbox. It accepts
// only the canonical recipe in the owned attempt workspace; it does not run
// model-authored code or scripts.
func RunGeometryCompileWorkerCommand(args []string) error {
	if len(args) != 3 || args[0] != "geometry-compile-worker" || args[1] != "--manifest" || !filepath.IsAbs(args[2]) || filepath.Clean(args[2]) != args[2] || filepath.Base(args[2]) != GeometryManifestName {
		return errors.New("geometry-compile-worker requires exactly --manifest <absolute attempt manifest>")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return RunGeometryCompileManifest(ctx, args[2])
}
func RunGeometryWorkerCommand(args []string) error {
	if len(args) != 3 || args[0] != "geometry-construction-worker" || args[1] != "--manifest" {
		return errors.New("geometry-construction-worker requires exactly --manifest <absolute attempt manifest>")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return RunGeometryManifest(ctx, args[2], nil)
}

func geometryInferenceRoute(route string) (string, sharedv1.Profile, string, error) {
	if route == "" { // Legacy manifests were local-only.
		route = "local"
	}
	switch route {
	case "local":
		return "extract.structured", sharedv1.Profile_PROFILE_LOCAL_ONLY, "ollama", nil
	case "byok":
		return "device.geometry.byok", sharedv1.Profile_PROFILE_REMOTE_ONLY, "openrouter", nil
	case "vrooli_credits":
		return "device.geometry.vrooli_credits", sharedv1.Profile_PROFILE_REMOTE_ONLY, "lpbs", nil
	default:
		return "", sharedv1.Profile_PROFILE_UNSPECIFIED, "", errors.New("geometry manifest selects an unsupported funding route")
	}
}

func ValidateGeometryCorrectionPrompt(correction *GeometryCorrectionPrompt) bool {
	if correction == nil || !validGeometryJobID(correction.SourceJobID) || !validWorkerDigest(correction.SourceInputFingerprint) || !validWorkerDigest(correction.SourceBriefSHA256) || !validWorkerDigest(correction.RecipeSHA256) || len(correction.RecipeJSON) == 0 || len(correction.RecipeJSON) > geometry.MaxRecipeBytes || workerDigest([]byte(correction.RecipeJSON)) != correction.RecipeSHA256 || strings.TrimSpace(correction.Feedback) == "" || len(correction.Feedback) > 1200 {
		return false
	}
	for _, ch := range correction.SourceJobID {
		if ch != '-' && (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	for _, ch := range correction.Feedback {
		if ch < 0x20 && ch != '\n' && ch != '\t' {
			return false
		}
	}
	mode := correction.Mode
	if mode == "" {
		mode = "semantic_repair"
	} // Legacy failed-recipe snapshots.
	switch mode {
	case "semantic_repair":
		if correction.Issue == nil || correction.Issue.PartID == "" || len(correction.Issue.PartID) > 48 {
			return false
		}
		issue, ok := geometry.IssueFromError(func() error { _, _, err := geometry.Decode([]byte(correction.RecipeJSON)); return err }())
		if !ok || issue != *correction.Issue {
			return false
		}
		switch issue.Code {
		case "part_outside_envelope", "rotated_part_outside_envelope":
			return issue.Axis == "x" || issue.Axis == "y" || issue.Axis == "z"
		case "bevel_exceeds_part_dimensions":
			return issue.Axis == "" && issue.BevelMM > 0 && issue.MinimumDimensionMM > 0 && issue.MaxBevelExclusiveMM > 0
		default:
			return false
		}
	case "visual_refinement":
		if correction.Issue != nil {
			return false
		}
		recipe, canonical, err := geometry.Decode([]byte(correction.RecipeJSON))
		return err == nil && recipe.Operation == "construct" && bytes.Equal(canonical, []byte(correction.RecipeJSON))
	default:
		return false
	}
}

func validGeometryJobID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, ch := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if ch != '-' {
				return false
			}
			continue
		}
		if ch < '0' || ch > '9' {
			if ch < 'a' || ch > 'f' {
				return false
			}
		}
	}
	return true
}

// RunGeometryManifest performs one disclosed local-only inference request,
// validates its finite recipe, compiles GLB bytes and writes a verifiable receipt.
func RunGeometryManifest(ctx context.Context, manifestPath string, client geometryInference) error {
	return runGeometryManifest(ctx, manifestPath, client, runGeometryCompileSandbox)
}

func runGeometryManifest(ctx context.Context, manifestPath string, client geometryInference, compile func(context.Context, string, GeometryWorkerManifest) error) error {
	if ctx == nil || !filepath.IsAbs(manifestPath) || filepath.Clean(manifestPath) != manifestPath || filepath.Base(manifestPath) != GeometryManifestName {
		return errors.New("geometry worker requires a fixed manifest in its owned workspace")
	}
	if compile == nil {
		return errors.New("geometry worker compile owner is unavailable")
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
	if !bytes.Equal(data, append(canonical, '\n')) || m.Version != 1 || m.JobID == "" || m.AttemptID == "" || !validWorkerDigest(m.InputFingerprint) || !validWorkerDigest(m.WorkerSHA256) || len(m.Images) < 1 || len(m.Images) > 4 || len(m.Brief) > 4000 || (m.FundingRoute == "" || m.FundingRoute == "local") && m.MaxCostMicros != 0 || m.FundingRoute != "" && m.FundingRoute != "local" && m.MaxCostMicros <= 0 {
		return errors.New("geometry manifest is not canonical or exceeds fixed bounds")
	}
	fundingRoute := strings.TrimSpace(m.FundingRoute)
	role, profile, expectedProvider, routeErr := geometryInferenceRoute(fundingRoute)
	if routeErr != nil {
		return routeErr
	}
	if fundingRoute == "" {
		fundingRoute = "local"
	}
	if err := verifyCurrentExecutable(m.WorkerSHA256); err != nil {
		return err
	}
	if client == nil {
		base, resolveErr := discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		if resolveErr != nil || base == "" {
			return errors.New("AI Gateway is unavailable for the selected geometry funding route")
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
	if m.Correction != nil {
		if !ValidateGeometryCorrectionPrompt(m.Correction) {
			return errors.New("geometry correction source is not bound to its verified prior recipe")
		}
		source += geometryCorrectionContext(m.Correction)
	}
	var spendLimit *inferencev1.SpendLimit
	if fundingRoute != "local" {
		references := make([]string, 0, len(m.Images))
		for _, image := range m.Images {
			references = append(references, image.SHA256)
		}
		spendLimit = &inferencev1.SpendLimit{MaxCostMicros: m.MaxCostMicros, OperationId: m.AttemptID, InputFingerprint: m.InputFingerprint, ReferenceSha256: references, ImageCount: int32(len(m.Images)), FundingRoute: fundingRoute}
	}
	request := &inferencev1.RunRequest{Source: source, SchemaJson: schema, Instruction: "Use the attached user-selected project reference images as visual evidence. Separate observed facts from uncertain inference. Output finite part geometry and an independently mapped screen. Center the housing at [0,0,0]. Keep each part's complete rotated half-extents within [-W/2,+W/2] x [-H/2,+H/2] x [-D/2,+D/2]. Put a rectangular screen only on a flat box display face, with its stand-off still inside that same locked envelope. For every rounded_box, require 0 < bevel_mm < min(size_mm)/2; the upper bound is strict, so equality is invalid. For boxes and cylinders, bevel_mm must be 0. Use only the locked dimensions.", Role: role, Attachments: attachments, Profile: profile, MaxOutputTokens: 4096, MaxProviderCalls: 1, SpendLimit: spendLimit}
	wrapped, callErr := client.Run(ctx, connect.NewRequest(request))
	var response *inferencev1.RunResponse
	if wrapped != nil {
		response = wrapped.Msg
	}
	var outcome *GeometryInferenceOutcome
	var outcomeSHA string
	if response != nil {
		outcome = geometryOutcome(m, response, time.Now().UTC())
		value := []byte(response.GetValueJson())
		if len(value) <= geometryResponseValueMax {
			if writeErr := writeGeometryFile(root, GeometryResponseValueName, value); writeErr != nil {
				return writeErr
			}
			if verifyErr := verifyGeometryResponseValue(root, outcome); verifyErr != nil {
				return verifyErr
			}
			outcome.ValueDisposition = "retained_exact"
			outcome.ValueFile = GeometryResponseValueName
		} else {
			outcome.ValueDisposition = "omitted_over_limit"
		}
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
		return fmt.Errorf("selected AI Gateway route failed: %w", callErr)
	}
	if response == nil || response.GetError() != nil || !response.GetValidated() || response.GetProviderCallsMade() > 1 {
		return errors.New("selected AI Gateway route could not return one validated bounded recipe")
	}
	if response.GetProviderCallsMade() != 1 {
		return errors.New("selected AI Gateway route did not confirm a provider dispatch attempt")
	}
	if response.GetProvider() != expectedProvider {
		return errors.New("AI Gateway response provider does not match the explicitly selected funding route")
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
		if issue, ok := geometry.IssueFromError(err); ok {
			outcome.SemanticIssue = &issue
			outcomeBytes, marshalErr := json.Marshal(outcome)
			if marshalErr != nil {
				return marshalErr
			}
			if writeErr := replaceGeometryOutcome(root, append(outcomeBytes, '\n')); writeErr != nil {
				return writeErr
			}
		}
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
	if err := writeGeometryFile(root, GeometryRecipeName, append(canonicalRecipe, '\n')); err != nil {
		return err
	}
	if err := compile(ctx, workspace, m); err != nil {
		return err
	}
	glb, err := readGeometryFile(ctx, root, GeometryOutputName, geometry.MaxGLBBytes)
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
	usage := response.GetUsage()
	var creditsCharged *int64
	if usage.CreditsCharged != nil && *usage.CreditsCharged >= 0 {
		charged := *usage.CreditsCharged
		creditsCharged = &charged
	}
	r := GeometryWorkerReceipt{Version: 2, JobID: m.JobID, AttemptID: m.AttemptID, InputFingerprint: m.InputFingerprint, WorkerSHA256: m.WorkerSHA256, RecipeSHA256: workerDigest(canonicalRecipe), OutputSHA256: workerDigest(glb), OutputBytes: len(glb), Provider: response.GetProvider(), Model: response.GetModel(), FundingRoute: fundingRoute, ProviderCalls: response.GetProviderCallsMade(), InputTokens: usage.GetInputTokens(), OutputTokens: usage.GetOutputTokens(), CostMicros: usage.GetCostMicros(), CreditsCharged: creditsCharged, AccountingStatus: usage.GetAccountingStatus(), InputProvenance: usage.GetInputTokensProvenance(), OutputProvenance: usage.GetOutputTokensProvenance(), CostProvenance: usage.GetCostProvenance(), Applied: response.GetApplied(), InferenceOutcomeSHA256: outcomeSHA, CompletedAt: time.Now().UTC()}
	receipt, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return writeGeometryFile(root, GeometryReceiptName, append(receipt, '\n'))
}

func geometryCorrectionContext(correction *GeometryCorrectionPrompt) string {
	mode := correction.Mode
	if mode == "" {
		mode = "semantic_repair"
	}
	if mode == "visual_refinement" {
		return fmt.Sprintf("\n\nExplicit user-directed visual refinement of accepted successful job %s. Prior accepted recipe (sha256 %s):\n%s\nUse the held-out reference images to refine visual proportions while preserving the locked envelope and all established dimensions. Do not invent unverified product facts. Operator feedback: %s", correction.SourceJobID, correction.RecipeSHA256, correction.RecipeJSON, correction.Feedback)
	}
	if correction.Issue.Code == "bevel_exceeds_part_dimensions" {
		i := correction.Issue
		return fmt.Sprintf("\n\nExplicit user-directed correction of failed job %s. Prior recipe (sha256 %s):\n%s\nVerified semantic issue: part %s has bevel_mm %.9g mm, while its smallest size_mm dimension is %.9g mm. The exact strict rule is 0 < bevel_mm < %.9g mm (half the smallest dimension); equality is invalid. Correct this part's bevel below that exclusive upper bound. Keep every rotated part half-extent within the locked envelope centered at the origin. Operator feedback: %s", correction.SourceJobID, correction.RecipeSHA256, correction.RecipeJSON, i.PartID, i.BevelMM, i.MinimumDimensionMM, i.MaxBevelExclusiveMM, correction.Feedback)
	}
	return fmt.Sprintf("\n\nExplicit user-directed correction of failed job %s. Prior recipe (sha256 %s):\n%s\nVerified semantic issue: %s; part %s; axis %s. Keep every rotated part half-extent within the locked envelope centered at the origin. Operator feedback: %s", correction.SourceJobID, correction.RecipeSHA256, correction.RecipeJSON, correction.Issue.Code, correction.Issue.PartID, correction.Issue.Axis, correction.Feedback)
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
	return &GeometryInferenceOutcome{Version: 1, JobID: m.JobID, AttemptID: m.AttemptID, InputFingerprint: m.InputFingerprint, WorkerSHA256: m.WorkerSHA256, Provider: provider, Model: model, ProviderCalls: response.GetProviderCallsMade(), Applied: response.GetApplied(), SpendAuthorization: response.GetSpendAuthorization(), Usage: usage, Validated: response.GetValidated(), ErrorCode: errorCode, ErrorConstruct: errorConstruct, ValueSHA256: workerDigest(value), ValueBytes: len(value), ReceivedAt: now}
}

func verifyGeometryResponseValue(root *os.Root, outcome *GeometryInferenceOutcome) error {
	f, err := root.Open(GeometryResponseValueName)
	if err != nil {
		return err
	}
	defer f.Close()
	value, err := io.ReadAll(io.LimitReader(f, geometryResponseValueMax+1))
	if err != nil || len(value) != outcome.ValueBytes || workerDigest(value) != outcome.ValueSHA256 {
		return errors.New("geometry response value differs from outcome digest or size")
	}
	return nil
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
func replaceGeometryOutcome(root *os.Root, data []byte) error {
	const tempName = ".geometry-inference-outcome-update"
	_ = root.Remove(tempName)
	f, err := root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if n, writeErr := f.Write(data); writeErr != nil || n != len(data) {
		_ = f.Close()
		_ = root.Remove(tempName)
		return errors.New("geometry worker could not write complete outcome update")
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = root.Remove(tempName)
		return err
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(tempName)
		return err
	}
	if err := root.Rename(tempName, GeometryInferenceOutcomeName); err != nil {
		_ = root.Remove(tempName)
		return err
	}
	return nil
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
