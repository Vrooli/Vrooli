package renderjobs

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
	"regexp"
	"strings"
	"time"
	"unicode"

	"device-studio/internal/geometry"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	GeometryOutputProfile   = "local-ai-geometry-glb-v1"
	geometryRendererVersion = "device-studio-geometry-recipe-v1"
	geometryInputLimit      = 64 << 20
)

var creationCandidateID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type GeometryCreationRequest struct {
	ProjectID         string     `json:"project_id"`
	CandidateID       string     `json:"candidate_id"`
	IdempotencyKey    string     `json:"idempotency_key"`
	Brief             string     `json:"brief"`
	EnvelopeMM        [3]float64 `json:"envelope_mm"`
	ReferenceMediaIDs []string   `json:"reference_media_ids"`
}

func sameJSON(left, right any) bool {
	a, errA := json.Marshal(left)
	b, errB := json.Marshal(right)
	return errA == nil && errB == nil && bytes.Equal(a, b)
}

func validGeometryUsageProvenance(value inferencev1.UsageProvenance) bool {
	return value >= inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN && value <= inferencev1.UsageProvenance_USAGE_PROVENANCE_MIXED
}

func validGeometryAppliedSettings(value *sharedv1.AppliedSettings) bool {
	return value != nil && value.GetTemperatureSupport() >= sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED && value.GetTemperatureSupport() <= sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN && value.GetMaxOutputTokensSource() == sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST && value.GetMaxOutputTokensEffective() == 4096 && (value.TemperatureSent == nil || !math.IsNaN(value.GetTemperatureSent()) && !math.IsInf(value.GetTemperatureSent(), 0))
}

type geometryReference struct {
	MediaID   string `json:"media_id"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
	MediaType string `json:"media_type"`
	Width     uint32 `json:"width"`
	Height    uint32 `json:"height"`
}
type geometrySubjectData struct {
	Version    int                 `json:"version"`
	Brief      string              `json:"brief"`
	EnvelopeMM [3]float64          `json:"envelope_mm"`
	References []geometryReference `json:"references"`
}

func (c *Coordinator) SubmitGeometryCreation(ctx context.Context, req GeometryCreationRequest) (jobs.Job, bool, error) {
	ctx, cancel := boundedDetached(ctx, stageTimeout)
	defer cancel()
	req.ProjectID, req.CandidateID, req.IdempotencyKey, req.Brief = strings.TrimSpace(req.ProjectID), strings.TrimSpace(req.CandidateID), strings.TrimSpace(req.IdempotencyKey), strings.TrimSpace(req.Brief)
	if req.ProjectID == "" || !creationCandidateID.MatchString(req.CandidateID) || req.IdempotencyKey == "" || len(req.IdempotencyKey) > 200 || req.Brief == "" || len(req.Brief) > 4000 || len(req.ReferenceMediaIDs) < 1 || len(req.ReferenceMediaIDs) > 4 {
		return jobs.Job{}, false, errors.New("renderjobs: geometry creation requires project, candidate, brief, and 1..4 selected project images")
	}
	for _, v := range req.EnvelopeMM {
		if v <= 0 || v > 1000 {
			return jobs.Job{}, false, errors.New("renderjobs: locked dimensions are outside the supported range")
		}
	}
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, false, err
	}
	references := make([]geometryReference, 0, len(req.ReferenceMediaIDs))
	seen := map[string]bool{}
	var total int64
	for _, id := range req.ReferenceMediaIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return jobs.Job{}, false, errors.New("renderjobs: reference media IDs must be distinct")
		}
		seen[id] = true
		m, e := c.media.Get(ctx, req.ProjectID, id)
		if e != nil {
			return jobs.Job{}, false, e
		}
		if m.ProjectID != req.ProjectID || m.ID != id || m.Kind != projects.MediaKindImage || !m.RasterKnown || m.Width == 0 || m.Height == 0 || m.SizeBytes <= 0 || (m.MediaType != "image/png" && m.MediaType != "image/jpeg") || !validDigest(m.SHA256) {
			return jobs.Job{}, false, errors.New("renderjobs: reference is not a verified project PNG/JPEG image")
		}
		total += m.SizeBytes
		if total > 10<<20 {
			return jobs.Job{}, false, errors.New("renderjobs: selected reference images exceed 10 MiB total")
		}
		references = append(references, geometryReference{MediaID: m.ID, SHA256: m.SHA256, SizeBytes: m.SizeBytes, MediaType: m.MediaType, Width: m.Width, Height: m.Height})
	}
	subjectData, err := json.Marshal(geometrySubjectData{Version: 1, Brief: req.Brief, EnvelopeMM: req.EnvelopeMM, References: references})
	if err != nil {
		return jobs.Job{}, false, err
	}
	worker, _, workerSHA, err := c.readCurrentExecutable()
	if err != nil {
		return jobs.Job{}, false, err
	}
	workerKey := workerBlobPrefix + workerSHA + "/device-studio-api"
	if err = c.putVerified(ctx, route, workerKey, worker, "application/octet-stream", workerSHA); err != nil {
		return jobs.Job{}, false, err
	}
	profileBytes := []byte("device-studio-geometry-recipe-v1\x00" + geometry.SchemaJSON())
	profileHash := sha256.Sum256(append(append([]byte(nil), profileBytes...), []byte(workerSHA)...))
	profileDigest := hex.EncodeToString(profileHash[:])
	bindings := []jobs.ArtifactBinding{{Role: "worker_executable", ArtifactID: workerKey, RevisionID: "sha256:" + workerSHA, SHA256: workerSHA}, routeBinding(route)}
	for i, ref := range references {
		bindings = append(bindings, jobs.ArtifactBinding{Role: fmt.Sprintf("reference_%d", i), ArtifactID: "source-media:" + ref.MediaID, RevisionID: "sha256:" + ref.SHA256, SHA256: ref.SHA256})
	}
	in := jobs.Input{ProjectID: req.ProjectID, Bindings: bindings, ToolProfileDigest: profileDigest, RendererVersion: geometryRendererVersion, OutputProfile: GeometryOutputProfile, SubjectKind: jobs.SubjectGeometryCreate, SubjectID: req.CandidateID, SubjectData: subjectData}
	if err = c.requireRoute(ctx, route); err != nil {
		return jobs.Job{}, false, err
	}
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: req.IdempotencyKey, Input: in})
	if err != nil {
		return jobs.Job{}, false, err
	}
	if job.Input.SubjectKind != jobs.SubjectGeometryCreate || job.Input.ProjectID != req.ProjectID || job.Input.SubjectID != req.CandidateID || job.Input.OutputProfile != GeometryOutputProfile {
		return jobs.Job{}, false, jobs.ErrIdempotencyConflict
	}
	return job, created, nil
}

// FindGeometryCreationByKey performs a durable readback for an explicitly
// retried client operation. It never submits work and scopes the result to the
// requested project and geometry-candidate subject.
func (c *Coordinator) FindGeometryCreationByKey(ctx context.Context, projectID, key string) (jobs.Job, error) {
	job, err := c.jobs.FindByIdempotencyKey(ctx, key)
	if err != nil {
		return jobs.Job{}, err
	}
	if job.Input.SubjectKind != jobs.SubjectGeometryCreate || job.Input.ProjectID != strings.TrimSpace(projectID) {
		return jobs.Job{}, jobs.ErrNotFound
	}
	return job, nil
}

type geometryCreationAdapter struct{}

func (geometryCreationAdapter) prepare(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt) (preparedOperation, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return preparedOperation{}, err
	}
	if job.Input.SubjectKind != jobs.SubjectGeometryCreate || job.Input.OutputProfile != GeometryOutputProfile {
		return preparedOperation{}, errDeterministicPreparation
	}
	var subject geometrySubjectData
	d := json.NewDecoder(bytes.NewReader(job.Input.SubjectData))
	d.DisallowUnknownFields()
	if err := d.Decode(&subject); err != nil {
		return preparedOperation{}, err
	}
	canonical, _ := json.Marshal(subject)
	if !bytes.Equal(canonical, job.Input.SubjectData) || subject.Version != 1 || len(subject.References) < 1 || len(subject.References) > 4 {
		return preparedOperation{}, errDeterministicPreparation
	}
	manifest := rendering.GeometryWorkerManifest{Version: 1, JobID: job.ID, AttemptID: attempt.ID, InputFingerprint: job.InputFingerprint, WorkerSHA256: bindingDigest(job.Input, "worker_executable"), Brief: subject.Brief, Envelope: subject.EnvelopeMM, Images: make([]rendering.GeometryImageInput, 0, len(subject.References))}
	files := make([]stagedFile, 0, len(subject.References))
	for i, ref := range subject.References {
		ref := ref
		binding := bindingByRole(job.Input, fmt.Sprintf("reference_%d", i))
		if binding.ArtifactID != "source-media:"+ref.MediaID || binding.SHA256 != ref.SHA256 || !validDigest(ref.SHA256) || ref.SizeBytes <= 0 || ref.SizeBytes > 10<<20 || (ref.MediaType != "image/png" && ref.MediaType != "image/jpeg") {
			return preparedOperation{}, errDeterministicPreparation
		}
		name := fmt.Sprintf("reference-%d.%s", i, map[bool]string{true: "png", false: "jpg"}[ref.MediaType == "image/png"])
		manifest.Images = append(manifest.Images, rendering.GeometryImageInput{Name: name, SHA256: ref.SHA256, MediaType: ref.MediaType, Width: ref.Width, Height: ref.Height, SizeBytes: ref.SizeBytes})
		files = append(files, stagedFile{Name: name, SHA256: ref.SHA256, Size: ref.SizeBytes, Mode: manifestMode, MaxBytes: 10 << 20, Open: func(openCtx context.Context) (io.ReadCloser, error) {
			media, reader, err := c.media.Open(openCtx, job.Input.ProjectID, ref.MediaID)
			if err != nil {
				return nil, err
			}
			if media.ID != ref.MediaID || media.SHA256 != ref.SHA256 || media.SizeBytes != ref.SizeBytes || media.MediaType != ref.MediaType || media.Kind != projects.MediaKindImage {
				_ = reader.Close()
				return nil, errors.New("renderjobs: source owner returned changed geometry reference")
			}
			return reader, nil
		}})
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return preparedOperation{}, err
	}
	return preparedOperation{Mode: operationGeometryCreate, ManifestBytes: append(manifestBytes, '\n'), Files: files, MaxInputBytes: geometryInputLimit}, nil
}

func (geometryCreationAdapter) validateCompletion(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt, intent persistedIntent, nativeKey, nativeSHA string) (jobs.CompletionEvidence, error) {
	if c.generatedIntake == nil {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: generated library intake owner is unavailable")
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if err := validateNativeEvidence(ctx, c, route, job, attempt, intent, operationGeometryCreate, nativeKey, nativeSHA); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	workspace, err := attemptWorkspace(route.Root, job.ID, attempt.ID)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	receiptBytes, err := readWorkspaceFile(workspace, rendering.GeometryReceiptName, 64<<10)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	var receipt rendering.GeometryWorkerReceipt
	d := json.NewDecoder(bytes.NewReader(receiptBytes))
	d.DisallowUnknownFields()
	if err = d.Decode(&receipt); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	canonical, _ := json.Marshal(receipt)
	if !bytes.Equal(receiptBytes, append(canonical, '\n')) || receipt.Version != 2 || receipt.JobID != job.ID || receipt.AttemptID != attempt.ID || receipt.InputFingerprint != job.InputFingerprint || receipt.WorkerSHA256 != bindingDigest(job.Input, "worker_executable") || receipt.ProviderCalls != 1 || !validDigest(receipt.RecipeSHA256) || !validDigest(receipt.OutputSHA256) || !validDigest(receipt.InferenceOutcomeSHA256) || !validGeometryAppliedSettings(receipt.Applied) || !validGeometryUsageProvenance(receipt.InputProvenance) || !validGeometryUsageProvenance(receipt.OutputProvenance) || !validGeometryUsageProvenance(receipt.CostProvenance) {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: geometry receipt does not match immutable one-call job")
	}
	outcomeBytes, err := readWorkspaceFile(workspace, rendering.GeometryInferenceOutcomeName, 64<<10)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	var outcome rendering.GeometryInferenceOutcome
	if err = decodeJSON(outcomeBytes, &outcome); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	outcomeCanonical, err := json.Marshal(outcome)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if !bytes.Equal(outcomeBytes, append(outcomeCanonical, '\n')) || digestBytes(outcomeBytes) != receipt.InferenceOutcomeSHA256 || outcome.JobID != job.ID || outcome.AttemptID != attempt.ID || outcome.InputFingerprint != job.InputFingerprint || outcome.WorkerSHA256 != receipt.WorkerSHA256 || outcome.ProviderCalls != receipt.ProviderCalls || !outcome.Validated || outcome.ErrorCode != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSPECIFIED || !sameJSON(outcome.Applied, receipt.Applied) || !sameJSON(outcome.Usage, &inferencev1.Usage{InputTokens: receipt.InputTokens, OutputTokens: receipt.OutputTokens, CostMicros: receipt.CostMicros, InputTokensProvenance: receipt.InputProvenance, OutputTokensProvenance: receipt.OutputProvenance, CostProvenance: receipt.CostProvenance}) {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: geometry response outcome differs from successful receipt")
	}
	nativeBytes, err := c.readBlob(ctx, nativeKey, 64<<10, nativeSHA)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	var native nativeOutcomeEvidence
	if err = decodeJSON(nativeBytes, &native); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if native.InferenceOutcomeSHA256 != receipt.InferenceOutcomeSHA256 || !sameJSON(native.InferenceOutcome, &outcome) {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: native outcome owner did not bind the typed inference response")
	}
	recipeBytes, err := readWorkspaceFile(workspace, rendering.GeometryRecipeName, geometry.MaxRecipeBytes+1)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	recipe, canonicalRecipe, err := geometry.Decode(bytes.TrimSuffix(recipeBytes, []byte{'\n'}))
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if recipe.Operation != "construct" {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: correction recipe requires an explicit parent revision")
	}
	if !bytes.Equal(recipeBytes, append(canonicalRecipe, '\n')) || digestBytes(canonicalRecipe) != receipt.RecipeSHA256 {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: recipe bytes differ from worker receipt")
	}
	var subject geometrySubjectData
	if err = json.Unmarshal(job.Input.SubjectData, &subject); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	for i := range subject.EnvelopeMM {
		if subject.EnvelopeMM[i] != recipe.Envelope[i] {
			return jobs.CompletionEvidence{}, errors.New("renderjobs: recipe changed locked dimensions")
		}
	}
	glb, err := readWorkspaceFile(workspace, rendering.GeometryOutputName, geometry.MaxGLBBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if len(glb) != receipt.OutputBytes || digestBytes(glb) != receipt.OutputSHA256 {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: compiled geometry differs from worker receipt")
	}
	inspection, err := rendering.InspectPhysicalGLB(glb)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if inspection.TriangleCount > geometry.MaxTriangles {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: output exceeded triangle limit")
	}
	refs := make([]string, 0, len(subject.References))
	for _, ref := range subject.References {
		refs = append(refs, ref.SHA256)
	}
	claims := library.IntakeClaims{CandidateID: job.Input.SubjectID, DisplayName: geometryCandidateDisplayName(subject.Brief), SourceLabel: "Device Studio local AI construction", LicenseClaim: "unknown"}
	lineage := library.CreationLineage{JobID: job.ID, RecipeSHA256: receipt.RecipeSHA256, ReferenceSHA256: refs, Provider: receipt.Provider, Model: receipt.Model, ProviderCalls: receipt.ProviderCalls, InputTokens: receipt.InputTokens, OutputTokens: receipt.OutputTokens, CostMicros: receipt.CostMicros, InputTokensProvenance: receipt.InputProvenance, OutputTokensProvenance: receipt.OutputProvenance, CostProvenance: receipt.CostProvenance, AppliedSettings: receipt.Applied, InferenceOutcomeSHA256: receipt.InferenceOutcomeSHA256}
	candidate, err := c.generatedIntake.PersistGenerated(ctx, claims, job.ID, lineage, glb)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if candidate.ID != job.Input.SubjectID || candidate.ReviewStatus != "unreviewed" || candidate.RenderStatus != "not_reviewed" || candidate.OriginalSHA256 != receipt.OutputSHA256 {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: generated intake did not retain exact unreviewed GLB")
	}
	return jobs.CompletionEvidence{AttemptID: attempt.ID, ExecutionClaimID: attempt.Execution.ID, InputFingerprint: job.InputFingerprint, Outputs: []jobs.OutputRef{{Kind: "unreviewed_geometry_candidate", ArtifactID: "intake-candidate:" + candidate.ID + "/revision:1", SHA256: candidate.OriginalSHA256, MIMEType: "model/gltf-binary", SizeBytes: candidate.SizeBytes}}, ReceiptArtifactID: "geometry-receipt:" + job.ID, ReceiptSHA256: digestBytes(receiptBytes), CompletedAt: time.Now().UTC()}, nil
}

func geometryCandidateDisplayName(brief string) string {
	plain := strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, brief)), " ")
	runes := []rune(plain)
	if len(runes) > 120 {
		plain = string(runes[:120])
	}
	if strings.TrimSpace(plain) == "" {
		return "Local geometry candidate"
	}
	return plain
}
func (geometryCreationAdapter) finalizeTerminal(context.Context, *Coordinator, jobs.SubjectFinalization) error {
	return nil
}
