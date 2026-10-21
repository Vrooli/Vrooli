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
	GeometryFundingLocal    = "local"
	GeometryFundingBYOK     = "byok"
	GeometryFundingCredits  = "vrooli_credits"
)

var creationCandidateID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type GeometryCreationRequest struct {
	ProjectID              string     `json:"project_id"`
	CandidateID            string     `json:"candidate_id"`
	IdempotencyKey         string     `json:"idempotency_key"`
	Brief                  string     `json:"brief"`
	EnvelopeMM             [3]float64 `json:"envelope_mm"`
	ReferenceMediaIDs      []string   `json:"reference_media_ids"`
	FundingRoute           string     `json:"funding_route,omitempty"`
	CorrectionSourceJobID  string     `json:"correction_source_job_id,omitempty"`
	CorrectionRecipeSHA256 string     `json:"correction_recipe_sha256,omitempty"`
	CorrectionFeedback     string     `json:"correction_feedback,omitempty"`
}

type GeometryCorrectionDetails struct {
	JobID                  string                  `json:"job_id"`
	ProjectID              string                  `json:"project_id"`
	AttemptID              string                  `json:"attempt_id,omitempty"`
	SourceInputFingerprint string                  `json:"source_input_fingerprint,omitempty"`
	SourceBriefSHA256      string                  `json:"source_brief_sha256,omitempty"`
	Disposition            string                  `json:"disposition"`
	Issue                  *geometry.SemanticIssue `json:"issue,omitempty"`
	PriorRecipeJSON        string                  `json:"prior_recipe_json,omitempty"`
	RecipeSHA256           string                  `json:"recipe_sha256,omitempty"`
	RecipeBytes            int64                   `json:"recipe_bytes,omitempty"`
	EnvelopeMM             [3]float64              `json:"envelope_mm"`
}

// OpenGeometryCorrection reads only a failed job's verified, attempt-bound
// recipe artifact. It never returns a workspace path or advances the job.
func (c *Coordinator) OpenGeometryCorrection(ctx context.Context, jobID string) (GeometryCorrectionDetails, error) {
	ctx, cancel := boundedDetached(ctx, ownerCallTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil {
		return GeometryCorrectionDetails{}, err
	}
	return c.openGeometryCorrection(ctx, route, jobID)
}

func (c *Coordinator) openGeometryCorrection(ctx context.Context, route routePin, jobID string) (GeometryCorrectionDetails, error) {
	job, err := c.jobs.Get(ctx, jobID)
	if err != nil {
		return GeometryCorrectionDetails{}, err
	}
	if err := verifyJobRoute(job, route); err != nil || job.Input.SubjectKind != jobs.SubjectGeometryCreate {
		return GeometryCorrectionDetails{}, jobs.ErrNotFound
	}
	if job.State != jobs.Failed {
		return GeometryCorrectionDetails{}, errors.New("geometry correction details require a failed job")
	}
	details := GeometryCorrectionDetails{JobID: job.ID, ProjectID: job.Input.ProjectID, SourceInputFingerprint: job.InputFingerprint, Disposition: "unavailable"}
	var subject geometrySubjectData
	d := json.NewDecoder(bytes.NewReader(job.Input.SubjectData))
	d.DisallowUnknownFields()
	if d.Decode(&subject) != nil || d.Decode(new(any)) != io.EOF || subject.Version != 1 || subject.Brief == "" {
		return details, nil
	}
	canonical, err := json.Marshal(subject)
	if err != nil || !bytes.Equal(canonical, job.Input.SubjectData) {
		return details, nil
	}
	details.EnvelopeMM = subject.EnvelopeMM
	details.SourceBriefSHA256 = digestBytes([]byte(subject.Brief))
	attempt, ok := currentAttempt(job)
	if !ok || attempt.State != jobs.AttemptFailed || attempt.Execution == nil || attempt.ExecutionReleasedAt == nil {
		return details, nil
	}
	details.AttemptID = attempt.ID
	nativeSHA := failureEvidenceDigest(job.Failure)
	if nativeSHA == "" {
		return details, nil
	}
	nativeKey := operationEvidenceBlobPrefix + job.ID + "/" + attempt.ID + "/native-" + nativeSHA + ".json"
	nativeBytes, err := c.readBlobForRoute(ctx, route, nativeKey, 64<<10, nativeSHA)
	if err != nil {
		return details, nil
	}
	var native nativeOutcomeEvidence
	if decodeJSON(nativeBytes, &native) != nil || native.JobID != job.ID || native.AttemptID != attempt.ID || native.InputFingerprint != job.InputFingerprint || native.Route != route || native.InferenceOutcome == nil || native.InferenceOutcomeSHA256 == "" || native.InferenceOutcome.SemanticIssue == nil || !validGeometrySemanticIssue(native.InferenceOutcome.SemanticIssue) {
		return details, nil
	}
	details.Issue = native.InferenceOutcome.SemanticIssue
	ref := native.RejectedRecipe
	if native.RejectedRecipeStatus != "available" || ref == nil || ref.Version != 1 || ref.JobID != job.ID || ref.AttemptID != attempt.ID || ref.InputFingerprint != job.InputFingerprint || ref.OutcomeSHA256 != native.InferenceOutcomeSHA256 || ref.OwnerID != OwnerID || ref.ScopeID != geometryRejectedRecipeScope || ref.SizeBytes <= 0 || ref.SizeBytes > geometry.MaxRecipeBytes || !validDigest(ref.SHA256) || ref.ArtifactID != operationEvidenceBlobPrefix+job.ID+"/"+attempt.ID+"/rejected-recipe-"+ref.SHA256+".json" {
		return details, nil
	}
	value := native.InferenceOutcome
	if value.ValueDisposition != "retained_exact" || value.ValueFile != rendering.GeometryResponseValueName || value.ValueSHA256 != ref.SHA256 || value.ValueBytes != int(ref.SizeBytes) {
		return details, nil
	}
	recipeBytes, err := c.readBlobForRoute(ctx, route, ref.ArtifactID, geometry.MaxRecipeBytes, ref.SHA256)
	if err != nil || int64(len(recipeBytes)) != ref.SizeBytes || !json.Valid(recipeBytes) {
		return details, nil
	}
	details.Disposition = "available"
	details.PriorRecipeJSON = string(recipeBytes)
	details.RecipeSHA256 = ref.SHA256
	details.RecipeBytes = ref.SizeBytes
	return details, nil
}

func failureEvidenceDigest(failure string) string {
	const marker = "native-evidence-sha256="
	parts := strings.Split(failure, marker)
	if len(parts) != 2 {
		return ""
	}
	digest := strings.Fields(parts[1])
	if len(digest) != 1 || !validDigest(digest[0]) {
		return ""
	}
	return digest[0]
}

func NormalizeGeometryFundingRoute(route string) string {
	switch strings.TrimSpace(route) {
	case "", GeometryFundingLocal:
		return GeometryFundingLocal
	case GeometryFundingBYOK:
		return GeometryFundingBYOK
	case GeometryFundingCredits:
		return GeometryFundingCredits
	default:
		return ""
	}
}

func sameJSON(left, right any) bool {
	a, errA := json.Marshal(left)
	b, errB := json.Marshal(right)
	return errA == nil && errB == nil && bytes.Equal(a, b)
}

func validGeometryCreationJobID(value string) bool {
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

func validCorrectionFeedback(value string) bool {
	if len(value) > 1200 {
		return false
	}
	for _, ch := range value {
		if ch < 0x20 && ch != '\n' && ch != '\t' {
			return false
		}
	}
	return true
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
	Version      int                                 `json:"version"`
	Brief        string                              `json:"brief"`
	EnvelopeMM   [3]float64                          `json:"envelope_mm"`
	References   []geometryReference                 `json:"references"`
	FundingRoute string                              `json:"funding_route,omitempty"`
	Correction   *rendering.GeometryCorrectionPrompt `json:"correction,omitempty"`
}

func (c *Coordinator) SubmitGeometryCreation(ctx context.Context, req GeometryCreationRequest) (jobs.Job, bool, error) {
	ctx, cancel := boundedDetached(ctx, stageTimeout)
	defer cancel()
	req.ProjectID, req.CandidateID, req.IdempotencyKey, req.Brief = strings.TrimSpace(req.ProjectID), strings.TrimSpace(req.CandidateID), strings.TrimSpace(req.IdempotencyKey), strings.TrimSpace(req.Brief)
	req.FundingRoute = NormalizeGeometryFundingRoute(req.FundingRoute)
	if req.ProjectID == "" || !creationCandidateID.MatchString(req.CandidateID) || req.IdempotencyKey == "" || len(req.IdempotencyKey) > 200 || req.Brief == "" || len(req.Brief) > 4000 || len(req.ReferenceMediaIDs) < 1 || len(req.ReferenceMediaIDs) > 4 {
		return jobs.Job{}, false, errors.New("renderjobs: geometry creation requires project, candidate, brief, and 1..4 selected project images")
	}
	if req.FundingRoute == "" {
		return jobs.Job{}, false, errors.New("renderjobs: unsupported geometry funding route")
	}
	if (req.CorrectionSourceJobID == "") != (req.CorrectionRecipeSHA256 == "") || (req.CorrectionSourceJobID == "") != (strings.TrimSpace(req.CorrectionFeedback) == "") || len(req.CorrectionFeedback) > 1200 || !validCorrectionFeedback(req.CorrectionFeedback) {
		return jobs.Job{}, false, errors.New("renderjobs: correction source and bounded operator feedback must be supplied together")
	}
	for _, v := range req.EnvelopeMM {
		if v <= 0 || v > 1000 {
			return jobs.Job{}, false, errors.New("renderjobs: locked dimensions are outside the supported range")
		}
	}
	ids := make([]string, 0, len(req.ReferenceMediaIDs))
	seen := map[string]bool{}
	for _, id := range req.ReferenceMediaIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return jobs.Job{}, false, errors.New("renderjobs: reference media IDs must be distinct")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	req.ReferenceMediaIDs = ids
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, false, err
	}
	if existing, found, err := c.readGeometryReplay(ctx, route, req); err != nil {
		return jobs.Job{}, false, err
	} else if found {
		return existing, false, nil
	}
	var correction *rendering.GeometryCorrectionPrompt
	if req.CorrectionSourceJobID != "" {
		if !validGeometryCreationJobID(req.CorrectionSourceJobID) || !validDigest(req.CorrectionRecipeSHA256) {
			return jobs.Job{}, false, errors.New("renderjobs: correction source identity is invalid")
		}
		details, correctionErr := c.openGeometryCorrection(ctx, route, req.CorrectionSourceJobID)
		if correctionErr != nil || details.Disposition != "available" || details.ProjectID != req.ProjectID || details.RecipeSHA256 != req.CorrectionRecipeSHA256 || details.Issue == nil {
			return jobs.Job{}, false, errors.New("renderjobs: verified failed recipe is unavailable for correction")
		}
		correction = &rendering.GeometryCorrectionPrompt{SourceJobID: details.JobID, SourceInputFingerprint: details.SourceInputFingerprint, SourceBriefSHA256: details.SourceBriefSHA256, RecipeSHA256: details.RecipeSHA256, RecipeJSON: details.PriorRecipeJSON, Issue: *details.Issue, Feedback: strings.TrimSpace(req.CorrectionFeedback)}
	}
	references := make([]geometryReference, 0, len(req.ReferenceMediaIDs))
	var total int64
	for _, id := range req.ReferenceMediaIDs {
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
	subjectData, err := json.Marshal(geometrySubjectData{Version: 1, Brief: req.Brief, EnvelopeMM: req.EnvelopeMM, References: references, FundingRoute: req.FundingRoute, Correction: correction})
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
		if errors.Is(err, jobs.ErrIdempotencyConflict) {
			if existing, found, readErr := c.readGeometryReplay(ctx, route, req); readErr != nil {
				return jobs.Job{}, false, readErr
			} else if found {
				return existing, false, nil
			}
		}
		return jobs.Job{}, false, err
	}
	if job.Input.SubjectKind != jobs.SubjectGeometryCreate || job.Input.ProjectID != req.ProjectID || job.Input.SubjectID != req.CandidateID || job.Input.OutputProfile != GeometryOutputProfile {
		return jobs.Job{}, false, jobs.ErrIdempotencyConflict
	}
	return job, created, nil
}

// readGeometryReplay returns an existing job only when its immutable subject
// represents the same user request. It deliberately avoids rereading current
// media or pinning the current executable: a replay recovers the original
// source and worker snapshot across application updates.
func (c *Coordinator) readGeometryReplay(ctx context.Context, route routePin, req GeometryCreationRequest) (jobs.Job, bool, error) {
	job, err := c.jobs.FindByIdempotencyKey(ctx, req.IdempotencyKey)
	if errors.Is(err, jobs.ErrNotFound) {
		return jobs.Job{}, false, nil
	}
	if err != nil {
		return jobs.Job{}, false, err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.Job{}, false, err
	}
	if err := verifyJobRoute(job, route); err != nil {
		return jobs.Job{}, false, err
	}
	if !sameGeometryCreationRequest(job, req) {
		return jobs.Job{}, false, jobs.ErrIdempotencyConflict
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.Job{}, false, err
	}
	return job, true, nil
}

func sameGeometryCreationRequest(job jobs.Job, req GeometryCreationRequest) bool {
	input, fingerprint, err := jobs.CanonicalInput(job.Input)
	if err != nil || fingerprint != job.InputFingerprint || !sameJSON(input, job.Input) ||
		input.SubjectKind != jobs.SubjectGeometryCreate || input.ProjectID != req.ProjectID ||
		input.SubjectID != req.CandidateID || input.RendererVersion != geometryRendererVersion ||
		input.OutputProfile != GeometryOutputProfile || input.AccountingRequired || input.AccountingOperationID != "" {
		return false
	}
	if len(input.Bindings) != len(req.ReferenceMediaIDs)+2 {
		return false
	}
	worker := bindingByRole(input, "worker_executable")
	workerSHA := worker.SHA256
	if !validDigest(workerSHA) || worker.ArtifactID != workerBlobPrefix+workerSHA+"/device-studio-api" || worker.RevisionID != "sha256:"+workerSHA {
		return false
	}
	var subject geometrySubjectData
	d := json.NewDecoder(bytes.NewReader(input.SubjectData))
	d.DisallowUnknownFields()
	if d.Decode(&subject) != nil || d.Decode(new(any)) != io.EOF {
		return false
	}
	if subject.FundingRoute == "" {
		subject.FundingRoute = GeometryFundingLocal
	}
	canonical, err := json.Marshal(subject)
	if err != nil || !bytes.Equal(canonical, input.SubjectData) || subject.Version != 1 ||
		subject.Brief != req.Brief || subject.EnvelopeMM != req.EnvelopeMM ||
		subject.FundingRoute != NormalizeGeometryFundingRoute(req.FundingRoute) ||
		len(subject.References) != len(req.ReferenceMediaIDs) || subject.Brief == "" || len(subject.Brief) > 4000 {
		return false
	}
	if req.CorrectionSourceJobID == "" {
		if subject.Correction != nil || req.CorrectionRecipeSHA256 != "" || strings.TrimSpace(req.CorrectionFeedback) != "" {
			return false
		}
	} else {
		correction := subject.Correction
		if correction == nil || correction.SourceJobID != req.CorrectionSourceJobID || correction.RecipeSHA256 != req.CorrectionRecipeSHA256 || correction.Feedback != strings.TrimSpace(req.CorrectionFeedback) || !validDigest(correction.SourceInputFingerprint) || !validDigest(correction.SourceBriefSHA256) || len(correction.RecipeJSON) == 0 || len(correction.RecipeJSON) > geometry.MaxRecipeBytes || digestBytes([]byte(correction.RecipeJSON)) != correction.RecipeSHA256 || !validGeometrySemanticIssue(&correction.Issue) {
			return false
		}
		issue, ok := geometry.IssueFromError(func() error { _, _, err := geometry.Decode([]byte(correction.RecipeJSON)); return err }())
		if !ok || issue != correction.Issue {
			return false
		}
	}
	var total int64
	seen := make(map[string]bool, len(subject.References))
	for i, reference := range subject.References {
		if reference.MediaID != req.ReferenceMediaIDs[i] || reference.MediaID == "" || seen[reference.MediaID] ||
			!validDigest(reference.SHA256) || reference.SizeBytes <= 0 || reference.SizeBytes > 10<<20 ||
			reference.Width == 0 || reference.Height == 0 ||
			(reference.MediaType != "image/png" && reference.MediaType != "image/jpeg") {
			return false
		}
		seen[reference.MediaID] = true
		total += reference.SizeBytes
		binding := bindingByRole(input, fmt.Sprintf("reference_%d", i))
		if binding.ArtifactID != "source-media:"+reference.MediaID || binding.RevisionID != "sha256:"+reference.SHA256 || binding.SHA256 != reference.SHA256 {
			return false
		}
	}
	return total <= 10<<20
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
	fundingRoute := NormalizeGeometryFundingRoute(subject.FundingRoute)
	if fundingRoute == "" {
		fundingRoute = GeometryFundingLocal // Older canonical subjects predate explicit route selection.
	}
	if subject.Correction != nil && !rendering.ValidateGeometryCorrectionPrompt(subject.Correction) {
		return preparedOperation{}, errDeterministicPreparation
	}
	manifest := rendering.GeometryWorkerManifest{Version: 1, JobID: job.ID, AttemptID: attempt.ID, InputFingerprint: job.InputFingerprint, WorkerSHA256: bindingDigest(job.Input, "worker_executable"), Brief: subject.Brief, Envelope: subject.EnvelopeMM, FundingRoute: fundingRoute, Images: make([]rendering.GeometryImageInput, 0, len(subject.References)), Correction: subject.Correction}
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
	if subject.Correction != nil {
		correction := subject.Correction
		lineage.CorrectionSource = &library.CreationCorrectionSource{JobID: correction.SourceJobID, InputFingerprint: correction.SourceInputFingerprint, BriefSHA256: correction.SourceBriefSHA256, RecipeSHA256: correction.RecipeSHA256, IssueCode: correction.Issue.Code, PartID: correction.Issue.PartID, Axis: correction.Issue.Axis}
	}
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
