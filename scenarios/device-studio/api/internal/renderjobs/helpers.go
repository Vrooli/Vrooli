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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"device-studio/internal/jobs"
	"device-studio/internal/presentations"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

type routePin struct {
	Root     string `json:"root"`
	TestMode bool   `json:"test_mode"`
}

const (
	inferenceOutcomeBound    = "bound"
	inferenceOutcomeNotBound = "not_bound"
)

type persistedIntent struct {
	Version          int                      `json:"version"`
	JobID            string                   `json:"job_id"`
	AttemptID        string                   `json:"attempt_id"`
	InputFingerprint string                   `json:"input_fingerprint"`
	Route            routePin                 `json:"route"`
	Intent           platform.OperationIntent `json:"intent"`
	Spec             platform.OperationSpec   `json:"spec"`
	ManifestSHA256   string                   `json:"manifest_sha256"`
	WorkerSHA256     string                   `json:"worker_sha256"`
	Mode             operationMode            `json:"mode,omitempty"`
}

type persistedQuiescence struct {
	Version                int                                 `json:"version"`
	JobID                  string                              `json:"job_id"`
	AttemptID              string                              `json:"attempt_id"`
	InputFingerprint       string                              `json:"input_fingerprint"`
	ClaimID                string                              `json:"execution_claim_id"`
	IntentKey              string                              `json:"intent_key"`
	Ref                    platform.OperationRef               `json:"operation_ref"`
	Route                  routePin                            `json:"route"`
	Stop                   platform.OperationStopReceipt       `json:"stop_receipt"`
	InferenceOutcomeStatus string                              `json:"inference_outcome_status,omitempty"`
	InferenceOutcome       *rendering.GeometryInferenceOutcome `json:"inference_outcome,omitempty"`
	InferenceOutcomeSHA256 string                              `json:"inference_outcome_sha256,omitempty"`
	NotDispatched          bool                                `json:"not_dispatched,omitempty"`
	Failure                string                              `json:"failure,omitempty"`
	EvidenceSHA256         string                              `json:"evidence_sha256,omitempty"`
	ObservedAt             time.Time                           `json:"observed_at"`
}

type nativeOutcomeEvidence struct {
	Version                int                                 `json:"version"`
	JobID                  string                              `json:"job_id"`
	AttemptID              string                              `json:"attempt_id"`
	InputFingerprint       string                              `json:"input_fingerprint"`
	ExecutionClaimID       string                              `json:"execution_claim_id"`
	Route                  routePin                            `json:"route"`
	Intent                 platform.OperationIntent            `json:"operation_intent"`
	Ref                    platform.OperationRef               `json:"operation_ref"`
	Observation            platform.OperationObservation       `json:"terminal_observation"`
	ObservedAt             time.Time                           `json:"observed_at"`
	InferenceOutcome       *rendering.GeometryInferenceOutcome `json:"inference_outcome,omitempty"`
	InferenceOutcomeSHA256 string                              `json:"inference_outcome_sha256,omitempty"`
	RejectedRecipe         *rejectedGeometryRecipeArtifact     `json:"rejected_recipe,omitempty"`
	RejectedRecipeStatus   string                              `json:"rejected_recipe_status,omitempty"`
}

type rejectedGeometryRecipeArtifact struct {
	Version          int    `json:"version"`
	JobID            string `json:"job_id"`
	AttemptID        string `json:"attempt_id"`
	InputFingerprint string `json:"input_fingerprint"`
	OutcomeSHA256    string `json:"outcome_sha256"`
	OwnerID          string `json:"owner_id"`
	ScopeID          string `json:"scope_id"`
	ArtifactID       string `json:"artifact_id"`
	SHA256           string `json:"sha256"`
	SizeBytes        int64  `json:"size_bytes"`
}

const geometryRejectedRecipeScope = "geometry-rejected-recipe"

type completedRenderEvidence struct {
	Version              int       `json:"version"`
	JobID                string    `json:"job_id"`
	AttemptID            string    `json:"attempt_id"`
	InputFingerprint     string    `json:"input_fingerprint"`
	ToolProfileDigest    string    `json:"tool_profile_digest"`
	NativeEvidenceKey    string    `json:"native_evidence_key"`
	NativeEvidenceSHA256 string    `json:"native_evidence_sha256"`
	WorkerReceiptKey     string    `json:"worker_receipt_key"`
	WorkerReceiptSHA256  string    `json:"worker_receipt_sha256"`
	OutputSHA256         string    `json:"output_sha256"`
	OutputSizeBytes      int64     `json:"output_size_bytes"`
	OutputWidth          int       `json:"output_width"`
	OutputHeight         int       `json:"output_height"`
	VerifiedAt           time.Time `json:"verified_at"`
}

func boundedDetached(ctx context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(ctx), budget)
}

func (c *Coordinator) route(ctx context.Context) (routePin, error) {
	root, err := c.roots.PickRequired(ctx, storage.ClassData)
	if err != nil {
		return routePin{}, fmt.Errorf("select required data route: %w", err)
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return routePin{}, errors.New("renderjobs: selected data root is not absolute and clean")
	}
	return routePin{Root: root, TestMode: database.IsTestMode(ctx)}, nil
}

func (c *Coordinator) requireRoute(ctx context.Context, expected routePin) error {
	current, err := c.route(ctx)
	if err != nil {
		return err
	}
	if current != expected {
		return ErrRouteMismatch
	}
	return nil
}

func routeBinding(route routePin) jobs.ArtifactBinding {
	data, _ := json.Marshal(route)
	sum := digestBytes(data)
	mode := "primary"
	if route.TestMode {
		mode = "test"
	}
	return jobs.ArtifactBinding{Role: "_internal_storage_route", ArtifactID: "route-sha256:" + sum, RevisionID: mode, SHA256: sum}
}

func verifyJobRoute(j jobs.Job, route routePin) error {
	want := routeBinding(route)
	for _, b := range j.Input.Bindings {
		if b.Role == want.Role {
			if b.ArtifactID == want.ArtifactID && b.RevisionID == want.RevisionID && b.SHA256 == want.SHA256 {
				return nil
			}
			return ErrRouteMismatch
		}
	}
	return ErrRouteMismatch
}

func validateSubmitRequest(req SubmitRequest) error {
	if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.ShotID) == "" || req.Revision <= 0 || strings.TrimSpace(req.IdempotencyKey) == "" || len(req.IdempotencyKey) > 200 {
		return fmt.Errorf("%w: project, shot, positive revision, and idempotency key are required", ErrInvalidRequest)
	}
	if req.OutputProfile == "" {
		req.OutputProfile = OutputProfilePNG
	}
	if req.OutputProfile != OutputProfilePNG && req.OutputProfile != OutputProfileMP4 && req.OutputProfile != OutputProfilePhysicalPreviewPNG && req.OutputProfile != OutputProfilePhysicalDiagnosticPNG && req.OutputProfile != OutputProfilePhysicalDiagnosticMotionMP4 {
		return fmt.Errorf("%w: output profile %q is unsupported", ErrInvalidRequest, req.OutputProfile)
	}
	return nil
}

func validateResolved(projectID, shotID string, revision int64, resolved shots.ResolvedShot) error {
	if resolved.ProjectID != projectID || resolved.Shot.ID != shotID || resolved.Shot.ProjectID != projectID || resolved.Revision.ShotID != shotID || resolved.Revision.Revision != revision || resolved.Revision.Snapshot == nil || !validDigest(resolved.Revision.RecipeSHA256) {
		return errors.New("renderjobs: resolved shot identity or canonical digest does not match request")
	}
	return nil
}

func validateStillProfile(revision shots.Revision) error {
	if !revision.RenderEligible {
		return ErrNotEligible
	}
	if revision.CompositorProfile != rendering.CompositorProfile && revision.CompositorProfile != presentations.SimulatedCompositorProfile {
		return fmt.Errorf("%w: compiled compositor profile %q is not supported", ErrNotEligible, revision.CompositorProfile)
	}
	return nil
}

func bindingsForShot(resolved shots.ResolvedShot, route routePin, workerKey, workerDigest string) ([]jobs.ArtifactBinding, error) {
	if !validDigest(resolved.Revision.RecipeSHA256) {
		return nil, errors.New("renderjobs: resolved shot has invalid recipe digest")
	}
	shotArtifact := "shot:" + resolved.Shot.ID + ":revision:" + strconv.FormatInt(resolved.Revision.Revision, 10)
	bindings := []jobs.ArtifactBinding{
		{Role: "shot_snapshot", ArtifactID: shotArtifact, RevisionID: "revision:" + strconv.FormatInt(resolved.Revision.Revision, 10), SHA256: resolved.Revision.RecipeSHA256},
		{Role: "worker_executable", ArtifactID: workerKey, RevisionID: "sha256:" + workerDigest, SHA256: workerDigest},
		routeBinding(route),
	}
	for role, media := range resolvedMedia(resolved.Revision) {
		if media == nil || media.GetId() == "" || !validDigest(media.GetSha256()) || media.GetSizeBytes() <= 0 || media.GetWidthPixels() == 0 || media.GetHeightPixels() == 0 {
			return nil, fmt.Errorf("renderjobs: resolved %s media metadata is incomplete", role)
		}
		bindings = append(bindings, jobs.ArtifactBinding{Role: role, ArtifactID: media.GetId(), RevisionID: "sha256:" + media.GetSha256(), SHA256: media.GetSha256()})
	}
	return bindings, nil
}

func resolvedMedia(rev shots.Revision) map[string]*shotsv1.ResolvedSourceMedia {
	out := map[string]*shotsv1.ResolvedSourceMedia{}
	if rev.Plate != nil {
		out["plate"] = rev.Plate
	}
	if rev.Screen != nil {
		out["screen"] = rev.Screen
	}
	if rev.ActiveMask != nil {
		out["active_mask"] = rev.ActiveMask
	}
	if rev.HardwareMask != nil {
		out["hardware_mask"] = rev.HardwareMask
	}
	return out
}

func shotRevisionID(shotID string, revision int64) string {
	return "shot:" + shotID + ":revision:" + strconv.FormatInt(revision, 10)
}

func parseShotRevision(in jobs.Input) (string, int64, error) {
	for _, binding := range in.Bindings {
		if binding.Role != "shot_snapshot" {
			continue
		}
		if !validDigest(binding.SHA256) || !strings.HasPrefix(binding.ArtifactID, "shot:") || binding.RevisionID != "revision:"+strconv.FormatInt(in.ShotRevision, 10) {
			return "", 0, errors.New("renderjobs: invalid canonical shot revision binding")
		}
		parts := strings.Split(binding.ArtifactID, ":")
		if len(parts) != 4 || parts[0] != "shot" || parts[2] != "revision" || parts[3] != strconv.FormatInt(in.ShotRevision, 10) || parts[1] == "" || shotRevisionID(parts[1], in.ShotRevision) != in.ShotRevisionID {
			return "", 0, errors.New("renderjobs: malformed canonical shot revision identity")
		}
		return parts[1], in.ShotRevision, nil
	}
	return "", 0, errors.New("renderjobs: job is missing its immutable shot revision binding")
}

func bindingsByRole(in jobs.Input) map[string]jobs.ArtifactBinding {
	out := make(map[string]jobs.ArtifactBinding, len(in.Bindings))
	for _, b := range in.Bindings {
		out[b.Role] = b
	}
	return out
}

func bindingByRole(in jobs.Input, role string) jobs.ArtifactBinding {
	return bindingsByRole(in)[role]
}

func bindingDigest(in jobs.Input, role string) string { return bindingByRole(in, role).SHA256 }

func stillClockDigest() string { return digestBytes([]byte("device-studio-still-source-clock-v1")) }

func toolProfileDigest(workerDigest, outputProfile, compositorProfile string) (string, error) {
	if !validDigest(workerDigest) || outputProfile != OutputProfilePNG || compositorProfile != rendering.CompositorProfile && compositorProfile != presentations.SimulatedCompositorProfile {
		return "", errors.New("renderjobs: invalid worker digest or output profile")
	}
	data, err := json.Marshal(struct {
		WorkerSHA256 string `json:"worker_sha256"`
		Compositor   string `json:"compositor"`
		Output       string `json:"output"`
	}{workerDigest, compositorProfile, outputProfile})
	if err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func operationSpec(workerPath, manifestPath, workspace string) platform.OperationSpec {
	spec, _ := operationSpecFor(operationStillRender, workerPath, manifestPath, workspace)
	return spec
}

func operationSpecFor(mode operationMode, workerPath, manifestPath, workspace string) (platform.OperationSpec, error) {
	command := ""
	wallTime := operationWallTime
	switch mode {
	case operationStillRender:
		command = "render-worker"
	case operationSourceInspect:
		command = "inspect-worker"
		wallTime = 5 * time.Minute
	case operationVideoRender:
		command = "video-render-worker"
		wallTime = 5 * time.Minute
	case operationPhysicalRender:
		command = "physical-render-worker"
		wallTime = time.Duration(rendering.PhysicalStillCPUWallTimeSeconds) * time.Second
	case operationPhysicalPreview:
		command = "physical-render-worker"
		wallTime = time.Duration(rendering.PhysicalPreviewWallTimeSeconds) * time.Second
	case operationPhysicalMotion:
		command = "physical-render-worker"
		wallTime = time.Duration(rendering.PhysicalMotionWallTimeSeconds) * time.Second
	case operationPhysicalMotionV2:
		command = "physical-render-worker"
		wallTime = time.Duration(rendering.PhysicalMotionV2WallTimeSeconds) * time.Second
	case operationPhysicalRecording:
		command = "physical-render-worker"
		wallTime = time.Duration(rendering.PhysicalRecordingWallTimeSeconds) * time.Second
	case operationGeometryCreate:
		command = "geometry-construction-worker"
		wallTime = 5 * time.Minute
	default:
		return platform.OperationSpec{}, errors.New("renderjobs: unsupported operation mode")
	}
	memory := int64(2 << 30)
	env := []string{}
	var err error
	if isPhysicalOperation(mode) {
		memory = 8 << 30
		env, err = physicalWorkerEnvironment(workspace)
		if err != nil {
			return platform.OperationSpec{}, err
		}
	} else if mode == operationGeometryCreate {
		env, err = geometryWorkerEnvironment(workspace)
		if err != nil {
			return platform.OperationSpec{}, err
		}
	}
	return platform.OperationSpec{
		Path: workerPath, Args: []string{command, "--manifest", manifestPath}, Env: env, Dir: workspace,
		Limits: platform.OperationLimits{MemoryMaxBytes: memory, TasksMax: 128, CPUWeight: 100, WallTime: wallTime},
	}, nil
}

func physicalWorkerEnvironment(workspace string) ([]string, error) {
	ownerHome, err := runtimeOwnerHome()
	if err != nil {
		return nil, err
	}
	nvidiaSMI, err := resolveNvidiaSMI()
	if err != nil {
		return nil, err
	}
	return physicalWorkerEnvironmentFor(ownerHome, workspace, os.Getenv(cliinvoke.BinaryEnvVar), nvidiaSMI)
}

func physicalWorkerEnvironmentFor(ownerHome, workspace, configuredCLI, nvidiaSMI string) ([]string, error) {
	env, err := geometryWorkerEnvironmentFor(ownerHome, workspace, configuredCLI)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(nvidiaSMI) || filepath.Clean(nvidiaSMI) != nvidiaSMI {
		return nil, errors.New("renderjobs: resolved nvidia-smi path must be absolute and clean")
	}
	if info, statErr := os.Stat(nvidiaSMI); statErr != nil || !info.Mode().IsRegular() {
		return nil, errors.New("renderjobs: resolved nvidia-smi path is not a regular file")
	}
	return append(env,
		"XDG_CONFIG_HOME="+filepath.Join(workspace, "config"),
		"XDG_CACHE_HOME="+filepath.Join(workspace, "cache"),
		"BLENDER_USER_CONFIG="+filepath.Join(workspace, "config", "blender"),
		"BLENDER_USER_SCRIPTS="+filepath.Join(workspace, "scripts"),
		"BLENDER_USER_DATAFILES="+filepath.Join(workspace, "datafiles"),
		"PYTHONPYCACHEPREFIX="+filepath.Join(workspace, "pycache"),
		"PYTHONDONTWRITEBYTECODE=1",
		"OMP_NUM_THREADS=2",
		"DEVICE_STUDIO_NVIDIA_SMI="+nvidiaSMI,
	), nil
}

func resolveNvidiaSMI() (string, error) {
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return "", fmt.Errorf("renderjobs: resolve nvidia-smi for physical render telemetry: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("renderjobs: resolve absolute nvidia-smi path: %w", err)
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("renderjobs: resolved nvidia-smi is not a regular file")
	}
	return path, nil
}

func geometryWorkerEnvironment(workspace string) ([]string, error) {
	ownerHome, err := runtimeOwnerHome()
	if err != nil {
		return nil, err
	}
	return geometryWorkerEnvironmentFor(ownerHome, workspace, os.Getenv(cliinvoke.BinaryEnvVar))
}

func geometryWorkerEnvironmentFor(ownerHome, workspace, configuredCLI string) ([]string, error) {
	if strings.TrimSpace(workspace) == "" || !filepath.IsAbs(workspace) || filepath.Clean(workspace) != workspace {
		return nil, errors.New("renderjobs: geometry worker workspace must be absolute and clean")
	}
	cliPath, err := resolvePhysicalWorkerCLI(ownerHome, configuredCLI)
	if err != nil {
		return nil, err
	}
	return []string{
		"HOME=" + ownerHome,
		cliinvoke.BinaryEnvVar + "=" + cliPath,
		"TMPDIR=" + filepath.Join(workspace, "tmp"),
	}, nil
}

func runtimeOwnerHome() (string, error) {
	ownerHome, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(ownerHome) || filepath.Clean(ownerHome) != ownerHome {
		return "", errors.New("renderjobs: runtime owner home is unavailable for canonical runtime discovery")
	}
	return ownerHome, nil
}

func resolvePhysicalWorkerCLI(ownerHome, configuredCLI string) (string, error) {
	if !filepath.IsAbs(ownerHome) || filepath.Clean(ownerHome) != ownerHome {
		return "", errors.New("renderjobs: runtime owner home is not an absolute clean path")
	}
	path, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{
		Explicit: configuredCLI, RuntimeHome: ownerHome,
		Getenv:   func(string) string { return "" },
		LookPath: func(string) (string, error) { return "", os.ErrNotExist },
	})
	if err != nil {
		return "", fmt.Errorf("renderjobs: resolve governed control-plane CLI for physical worker: %w", err)
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", errors.New("renderjobs: resolved control-plane CLI path must be absolute and clean")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("renderjobs: resolved control-plane CLI is not a regular file")
	}
	return path, nil
}

func validateOperationSpec(env persistedIntent, workspace string) error {
	mode := env.Mode
	if mode == "" {
		mode = operationStillRender
	}
	manifestName := rendering.ManifestName
	if mode == operationSourceInspect {
		manifestName = "inspection-manifest.json"
	}
	if mode == operationVideoRender {
		manifestName = "video-manifest.json"
	}
	if mode == operationGeometryCreate {
		manifestName = rendering.GeometryManifestName
	}
	if isPhysicalOperation(mode) {
		manifestName = rendering.PhysicalManifestName
	}
	want, err := operationSpecFor(mode, filepath.Join(workspace, workerFileName), filepath.Join(workspace, manifestName), workspace)
	if err != nil {
		return err
	}
	got, err := json.Marshal(env.Spec)
	if err != nil {
		return err
	}
	wantBytes, err := json.Marshal(want)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, wantBytes) || env.Intent.Owner != OwnerID || env.Intent.AttemptID != env.AttemptID || env.Intent.SpecDigest == "" {
		return errors.New("renderjobs: persisted operation specification does not match its fixed subject profile")
	}
	verifiedIntent, err := platform.NewOperationIntent(OwnerID, env.AttemptID, env.Spec)
	if err != nil || verifiedIntent.SpecDigest != env.Intent.SpecDigest {
		return errors.New("renderjobs: persisted operation intent digest does not bind fixed spec")
	}
	return nil
}

func attemptWorkspace(root, jobID, attemptID string) (string, error) {
	if !jobIDPattern.MatchString(jobID) || !jobIDPattern.MatchString(attemptID) || filepath.IsAbs(jobID) || filepath.IsAbs(attemptID) || strings.ContainsAny(jobID+attemptID, `/\\`) {
		return "", errors.New("renderjobs: job or attempt identity is unsafe for workspace mapping")
	}
	return filepath.Join(root, workspaceRelativeRoot, jobID, attemptID), nil
}

func validateDirectoryTree(dir string) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return errors.New("renderjobs: workspace directory must be absolute and clean")
	}
	volume := filepath.VolumeName(dir)
	current := volume + string(filepath.Separator)
	rest := strings.TrimPrefix(dir, current)
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("renderjobs: workspace path contains a symlink or non-directory component")
		}
	}
	return nil
}

func readWorkspaceFile(workspace, name string, max int64) ([]byte, error) {
	if strings.ContainsAny(name, `/\\`) || name == "" || max <= 0 {
		return nil, errors.New("renderjobs: invalid workspace filename or limit")
	}
	if err := validateDirectoryTree(workspace); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entry, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !entry.Mode().IsRegular() || entry.Mode()&os.ModeSymlink != 0 || entry.Size() <= 0 || entry.Size() > max {
		return nil, errors.New("renderjobs: workspace artifact is not a bounded regular file")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > max {
		return nil, errors.New("renderjobs: workspace artifact is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return nil, fmt.Errorf("read workspace artifact: %w", err)
	}
	if int64(len(data)) > max {
		return nil, errors.New("renderjobs: workspace artifact exceeds size limit")
	}
	return data, nil
}

func verifyManifestInputs(workspace string, manifest rendering.Manifest) error {
	inputs := []struct {
		role    string
		binding rendering.ImageBinding
	}{
		{"plate", manifest.Plate}, {"screen", manifest.Screen}, {"active_mask", manifest.ActiveMask},
	}
	if manifest.HardwareMask != nil {
		inputs = append(inputs, struct {
			role    string
			binding rendering.ImageBinding
		}{"hardware_mask", *manifest.HardwareMask})
	}
	wantNames := map[string]string{"plate": "plate.image", "screen": "screen.image", "active_mask": "active-mask.png", "hardware_mask": "hardware-mask.png"}
	for _, input := range inputs {
		if input.binding.Name != wantNames[input.role] || !validDigest(input.binding.SHA256) || input.binding.SizeBytes <= 0 {
			return fmt.Errorf("renderjobs: manifest has invalid %s input binding", input.role)
		}
		data, err := readWorkspaceFile(workspace, input.binding.Name, maxWorkerInputBytes)
		if err != nil {
			return fmt.Errorf("read staged %s input: %w", input.role, err)
		}
		if int64(len(data)) != input.binding.SizeBytes || digestBytes(data) != input.binding.SHA256 {
			return fmt.Errorf("renderjobs: staged %s input digest or size changed", input.role)
		}
	}
	return nil
}

func validateReceipt(receipt rendering.Receipt, manifest rendering.Manifest, manifestSHA string, j jobs.Job, a jobs.Attempt) error {
	if receipt.Version != manifest.Version || receipt.JobID != j.ID || receipt.AttemptID != a.ID || receipt.InputFingerprint != j.InputFingerprint || receipt.ManifestSHA256 != manifestSHA || receipt.Profile != manifest.Profile || receipt.Algorithm == "" || receipt.OutputName != rendering.OutputName || !validDigest(receipt.OutputSHA256) || receipt.OutputSizeBytes <= 0 || receipt.CompletedAt.IsZero() {
		return errors.New("renderjobs: worker receipt is incomplete or bound to another attempt")
	}
	if manifest.Version == 1 && receipt.Presentation != nil {
		return errors.New("renderjobs: legacy worker receipt unexpectedly includes a presentation")
	}
	if manifest.Version == 2 {
		presentation := receipt.Presentation
		semantic := "device-studio-presentation-v2"
		phoneV3 := manifest.Presentation != nil && manifest.Presentation.Phone != nil && manifest.Presentation.Phone.ProfileVersion == presentations.PhoneProfileVersionAuto
		if phoneV3 {
			semantic = "device-studio-presentation-v3"
		}
		if manifest.Presentation == nil || manifest.PresentationLayout == nil || presentation == nil || presentation.SemanticVersion != semantic || !validDigest(presentation.DisplayRasterSHA256) || !validDigest(presentation.Layout.ConfigSHA256) || !validDigest(presentation.Layout.FontSHA256) || presentation.Layout.Rasterizer == "" || presentation.Layout.FillPolicy == "" {
			return errors.New("renderjobs: worker presentation receipt is incomplete or mismatched")
		}
		if !phoneV3 && (presentation.Layout.ResolvedStatusInk != nil || presentation.Layout.ResolvedHomeInk != nil || (presentation.Config.Phone != nil && presentation.Config.Phone.ProfileVersion != "")) {
			return errors.New("renderjobs: v2 worker receipt contains v3 phone profile fields")
		}
		if phoneV3 && (presentation.Config.Phone == nil || presentation.Config.Phone.ProfileVersion != presentations.PhoneProfileVersionAuto || presentation.Layout.ResolvedStatusInk == nil || presentation.Layout.ResolvedHomeInk == nil || presentation.Layout.ProfileVersion != presentations.PhoneProfileVersionAuto) {
			return errors.New("renderjobs: v3 worker receipt lacks the pinned phone v3 identity")
		}
		wanted, _ := json.Marshal(manifest.Presentation)
		got, _ := json.Marshal(presentation.Config)
		if !bytes.Equal(wanted, got) {
			return errors.New("renderjobs: worker presentation receipt config differs from manifest")
		}
		wantLayout, _ := json.Marshal(manifest.PresentationLayout)
		gotLayout, _ := json.Marshal(presentation.Layout)
		if !bytes.Equal(wantLayout, gotLayout) {
			return errors.New("renderjobs: worker presentation layout differs from pinned manifest")
		}
	}
	if receipt.OutputSizeBytes > maxOutputBytes || int64(receipt.OutputWidth)*int64(receipt.OutputHeight) > maxOutputPixels || receipt.OutputWidth <= 0 || receipt.OutputHeight <= 0 || receipt.Composition.PlateWidth != receipt.OutputWidth || receipt.Composition.PlateHeight != receipt.OutputHeight {
		return errors.New("renderjobs: worker receipt output dimensions exceed bounds")
	}
	want := map[string]rendering.ImageBinding{"plate": manifest.Plate, "screen": manifest.Screen, "active_mask": manifest.ActiveMask}
	if manifest.HardwareMask != nil {
		want["hardware_mask"] = *manifest.HardwareMask
	}
	if len(receipt.Inputs) != len(want) {
		return errors.New("renderjobs: worker receipt input roles differ from manifest")
	}
	for _, input := range receipt.Inputs {
		binding, ok := want[input.Role]
		if !ok || input.Name != binding.Name || input.SHA256 != binding.SHA256 || input.SizeBytes != binding.SizeBytes {
			return errors.New("renderjobs: worker receipt input binding differs from manifest")
		}
		delete(want, input.Role)
	}
	if len(want) != 0 {
		return errors.New("renderjobs: worker receipt omitted an input role")
	}
	if receipt.Composition.Algorithm != receipt.Algorithm || receipt.Composition.ProtectedChangedPixelCount != 0 || !validDigest(receipt.Composition.ProtectedPixelsSHA256) {
		return errors.New("renderjobs: composition receipt does not certify protected pixels")
	}
	return nil
}

func decodeCanonical(data []byte, out *rendering.Manifest) error {
	if err := decodeJSON(data, out); err != nil {
		return err
	}
	canonical, err := json.Marshal(out)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, canonical) && !bytes.Equal(data, append(canonical, '\n')) {
		return errors.New("renderjobs: worker manifest is not canonical JSON")
	}
	return nil
}

func decodeJSON(data []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("renderjobs: artifact contains trailing JSON data")
	}
	return nil
}

func optionalImageBinding(bindings map[string]rendering.ImageBinding, role string) *rendering.ImageBinding {
	binding, ok := bindings[role]
	if !ok {
		return nil
	}
	return &binding
}

func currentAttempt(j jobs.Job) (jobs.Attempt, bool) {
	if len(j.Attempts) == 0 {
		return jobs.Attempt{}, false
	}
	return j.Attempts[len(j.Attempts)-1], true
}

func validDigest(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func boundedFailure(result string, status int) string {
	message := fmt.Sprintf("native render failed (manager result %q, exit status %d)", result, status)
	if len(message) > 512 {
		message = message[:512]
	}
	return message
}
