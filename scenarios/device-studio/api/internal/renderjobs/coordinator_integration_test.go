package renderjobs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"device-studio/internal/geometry"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	platform "github.com/vrooli/platform-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
)

func TestNativeOutcomeBindsTypedGeometryResponseOnSuccessAndFailure(t *testing.T) {
	for _, exitCode := range []int{0, 1} {
		t.Run(strconv.Itoa(exitCode), func(t *testing.T) {
			ops := &integrationOps{}
			c, ctx, route, running, _, env, _ := newClaimedPreDispatchJob(t, ops, "geometry-outcome-"+strconv.Itoa(exitCode))
			running.Input.SubjectKind = jobs.SubjectGeometryCreate
			attempt := running.Attempts[0]
			workspace, err := attemptWorkspace(route.Root, running.ID, attempt.ID)
			if err != nil {
				t.Fatal(err)
			}
			outcome := rendering.GeometryInferenceOutcome{Version: 1, JobID: running.ID, AttemptID: attempt.ID, InputFingerprint: running.InputFingerprint, WorkerSHA256: bindingDigest(running.Input, "worker_executable"), Provider: "ollama", Model: "local", ProviderCalls: 1, Usage: &inferencev1.Usage{InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}, Validated: exitCode == 0, ValueSHA256: digestBytes([]byte("value")), ValueBytes: 5, ReceivedAt: time.Now().UTC()}
			var rejected []byte
			if exitCode != 0 {
				rejected = []byte(`{"version":"rejected"}`)
				outcome.SemanticIssue = &geometry.SemanticIssue{Code: "part_outside_envelope", PartID: "housing", Axis: "y", Message: "part exceeds the locked envelope"}
				outcome.ValueSHA256, outcome.ValueBytes, outcome.ValueDisposition, outcome.ValueFile = digestBytes(rejected), len(rejected), "retained_exact", rendering.GeometryResponseValueName
				if err := os.WriteFile(filepath.Join(workspace, rendering.GeometryResponseValueName), rejected, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			outcomeBytes, err := json.Marshal(outcome)
			if err != nil {
				t.Fatal(err)
			}
			outcomeBytes = append(outcomeBytes, '\n')
			if err := os.WriteFile(filepath.Join(workspace, rendering.GeometryInferenceOutcomeName), outcomeBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			ref := platform.OperationRef{Intent: env.Intent, Unit: "test", NativeInvocationID: "invocation"}
			obs := platform.OperationObservation{Ref: ref, State: platform.OperationStoppedOutcomeKnown, ExitKnown: true, ExecMainStatus: exitCode}
			key, sha, err := c.persistNativeOutcome(ctx, route, running, attempt, env, ref, obs)
			if err != nil {
				t.Fatal(err)
			}
			persisted, err := c.readBlob(ctx, key, 64<<10, sha)
			if err != nil {
				t.Fatal(err)
			}
			var evidence nativeOutcomeEvidence
			if err := json.Unmarshal(persisted, &evidence); err != nil {
				t.Fatal(err)
			}
			if evidence.InferenceOutcome == nil || evidence.InferenceOutcome.Validated != (exitCode == 0) || evidence.InferenceOutcomeSHA256 != digestBytes(outcomeBytes) || bytes.Contains(persisted, []byte(`"value":`)) {
				t.Fatalf("native outcome did not bind the typed response for exit status %d: %+v", exitCode, evidence)
			}
			if exitCode != 0 {
				if evidence.RejectedRecipeStatus != "available" || evidence.RejectedRecipe == nil || evidence.RejectedRecipe.OutcomeSHA256 != digestBytes(outcomeBytes) || evidence.RejectedRecipe.SHA256 != digestBytes(rejected) {
					t.Fatalf("failed result did not bind the exact rejected recipe to its native outcome: %+v", evidence.RejectedRecipe)
				}
				got, err := c.readBlobForRoute(ctx, route, evidence.RejectedRecipe.ArtifactID, geometry.MaxRecipeBytes, evidence.RejectedRecipe.SHA256)
				if err != nil || !bytes.Equal(got, rejected) {
					t.Fatalf("bound correction response was not returned as exact bytes: %v", err)
				}
			}
		})
	}
}

type capturingGeneratedIntake struct {
	claims library.IntakeClaims
	data   []byte
}

func (c *capturingGeneratedIntake) PersistGenerated(_ context.Context, claims library.IntakeClaims, _ string, _ library.CreationLineage, data []byte) (library.IntakeCandidate, error) {
	c.claims, c.data = claims, append([]byte(nil), data...)
	return library.IntakeCandidate{ID: claims.CandidateID, OriginalSHA256: digestBytes(data), SizeBytes: int64(len(data)), ReviewStatus: "unreviewed", RenderStatus: "not_reviewed"}, nil
}

func TestGeometryCompletionKeepsLongMultilinePromptAndUsesSafeDisplayTitle(t *testing.T) {
	brief := strings.Repeat("compact field phone with physical controls ", 30) + "\nsecond line for camera and speaker details"
	if len(brief) < 1024 {
		t.Fatalf("test brief must exceed intake title limit: %d", len(brief))
	}
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	const jobID, attemptID, candidateID = "geometry-long-brief", "geometry-attempt", "geometry-candidate"
	fingerprint := digestBytes([]byte("immutable long geometry brief"))
	worker := []byte("geometry worker bytes")
	workerSHA := digestBytes(worker)
	input := jobs.Input{ProjectID: "project", SubjectID: candidateID, SubjectKind: jobs.SubjectGeometryCreate, ToolProfileDigest: fingerprint, RendererVersion: geometryRendererVersion, OutputProfile: GeometryOutputProfile, Bindings: []jobs.ArtifactBinding{{Role: "worker_executable", ArtifactID: "worker", SHA256: workerSHA}}}
	input.SubjectData, _ = json.Marshal(geometrySubjectData{Version: 1, Brief: brief, EnvelopeMM: [3]float64{80, 150, 12}})
	job := jobs.Job{ID: jobID, Input: input, InputFingerprint: fingerprint}
	attempt := jobs.Attempt{ID: attemptID, Number: 1}
	workspace, err := attemptWorkspace(route.Root, jobID, attemptID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, workerFileName), worker, 0o700); err != nil {
		t.Fatal(err)
	}
	spec, err := operationSpecFor(operationGeometryCreate, filepath.Join(workspace, workerFileName), filepath.Join(workspace, rendering.GeometryManifestName), workspace)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := platform.NewOperationIntent(OwnerID, attemptID, spec)
	if err != nil {
		t.Fatal(err)
	}
	attempt.Execution = &jobs.ExecutionClaim{ID: "geometry-claim", JobID: jobID, AttemptID: attemptID, OwnerID: OwnerID, ScopeID: intent.InvocationID}
	env := persistedIntent{Version: 1, JobID: jobID, AttemptID: attemptID, InputFingerprint: fingerprint, Route: route, Intent: intent, Spec: spec, WorkerSHA256: workerSHA, Mode: operationGeometryCreate}
	recipe := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{{ID: "housing", Kind: "rounded_box", Role: "housing", Size: [3]float64{78, 148, 10}, Bevel: 1, Material: "matte_polymer"}, {ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5.2}, Size: [3]float64{64, 124, 1}, Material: "glass"}}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	recipeBytes, err := json.Marshal(recipe)
	if err != nil {
		t.Fatal(err)
	}
	glb, err := geometry.Compile(recipe)
	if err != nil {
		t.Fatal(err)
	}
	applied := &sharedv1.AppliedSettings{TemperatureSupport: sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED, MaxOutputTokensEffective: 4096, MaxOutputTokensSource: sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST}
	usage := &inferencev1.Usage{InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}
	outcome := rendering.GeometryInferenceOutcome{Version: 1, JobID: jobID, AttemptID: attemptID, InputFingerprint: fingerprint, WorkerSHA256: workerSHA, Provider: "ollama", Model: "local-model", ProviderCalls: 1, Applied: applied, Usage: usage, Validated: true, ValueSHA256: digestBytes([]byte("recipe response")), ValueBytes: 16, ReceivedAt: time.Now().UTC()}
	outcomeBytes, _ := json.Marshal(outcome)
	outcomeBytes = append(outcomeBytes, '\n')
	if err := os.WriteFile(filepath.Join(workspace, rendering.GeometryRecipeName), append(recipeBytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, rendering.GeometryOutputName), glb, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, rendering.GeometryInferenceOutcomeName), outcomeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	receipt := rendering.GeometryWorkerReceipt{Version: 2, JobID: jobID, AttemptID: attemptID, InputFingerprint: fingerprint, WorkerSHA256: workerSHA, RecipeSHA256: digestBytes(recipeBytes), OutputSHA256: digestBytes(glb), OutputBytes: len(glb), Provider: "ollama", Model: "local-model", ProviderCalls: 1, Applied: applied, InferenceOutcomeSHA256: digestBytes(outcomeBytes), CompletedAt: time.Now().UTC(), InputProvenance: usage.InputTokensProvenance, OutputProvenance: usage.OutputTokensProvenance, CostProvenance: usage.CostProvenance}
	receiptBytes, _ := json.Marshal(receipt)
	if err := os.WriteFile(filepath.Join(workspace, rendering.GeometryReceiptName), append(receiptBytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	ref := platform.OperationRef{Intent: intent, Unit: "test", NativeInvocationID: "geometry-native"}
	if err := validateOperationSpec(env, workspace); err != nil {
		t.Fatalf("geometry operation spec is invalid: %v", err)
	}
	nativeKey, nativeSHA, err := c.persistNativeOutcome(ctx, route, job, attempt, env, ref, platform.OperationObservation{Ref: ref, State: platform.OperationStoppedOutcomeKnown, ExitKnown: true})
	if err != nil {
		t.Fatal(err)
	}
	c.generatedIntake = &capturingGeneratedIntake{}
	if _, err := (geometryCreationAdapter{}).validateCompletion(ctx, c, route, job, attempt, env, nativeKey, nativeSHA); err != nil {
		t.Fatalf("long/multiline prompt should complete as an unreviewed candidate: %v", err)
	}
	capture := c.generatedIntake.(*capturingGeneratedIntake)
	var stored geometrySubjectData
	if err := json.Unmarshal(job.Input.SubjectData, &stored); err != nil || stored.Brief != brief {
		t.Fatalf("full prompt was not retained in immutable job subject: %v", err)
	}
	if len(capture.claims.DisplayName) > 1024 || strings.ContainsAny(capture.claims.DisplayName, "\r\n") || capture.claims.DisplayName == brief || !bytes.Equal(capture.data, glb) {
		t.Fatalf("candidate title should be sanitized/bounded while exact geometry and full prompt remain: claims=%+v", capture.claims)
	}
}

func TestGeometryCancellationBindsOnlyCompleteCanonicalInferenceOutcome(t *testing.T) {
	for _, scenario := range []string{"complete", "missing", "partial", "mismatched identity"} {
		t.Run(scenario, func(t *testing.T) {
			ops := &integrationOps{}
			c, ctx, route, running, _, env, intentKey := newClaimedPreDispatchJob(t, ops, "geometry-cancel-outcome-"+strings.ReplaceAll(scenario, " ", "-"))
			running.Input.SubjectKind = jobs.SubjectGeometryCreate
			attempt := running.Attempts[0]
			workspace, err := attemptWorkspace(route.Root, running.ID, attempt.ID)
			if err != nil {
				t.Fatal(err)
			}
			outcomeBytes := []byte(nil)
			if scenario != "missing" {
				jobID := running.ID
				if scenario == "mismatched identity" {
					jobID = "another-job"
				}
				outcome := rendering.GeometryInferenceOutcome{Version: 1, JobID: jobID, AttemptID: attempt.ID, InputFingerprint: running.InputFingerprint, WorkerSHA256: bindingDigest(running.Input, "worker_executable"), Provider: "ollama", Model: "local", ProviderCalls: 1, Usage: &inferencev1.Usage{InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}, ValueSHA256: digestBytes([]byte("response")), ValueBytes: 8, ReceivedAt: time.Now().UTC()}
				if scenario == "partial" {
					outcomeBytes = []byte(`{"version":1`)
				} else {
					outcomeBytes, err = json.Marshal(outcome)
					if err != nil {
						t.Fatal(err)
					}
					outcomeBytes = append(outcomeBytes, '\n')
				}
				if err := os.WriteFile(filepath.Join(workspace, rendering.GeometryInferenceOutcomeName), outcomeBytes, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			requested, err := c.jobs.RequestCancel(ctx, running.ID, running.Version)
			if err != nil {
				t.Fatal(err)
			}
			requested.Input.SubjectKind = jobs.SubjectGeometryCreate
			ref := platform.OperationRef{Intent: env.Intent, Unit: "test", NativeInvocationID: "cancel-invocation"}
			ops.ref = ref
			cancelled, err := c.cancelWithRef(ctx, route, requested, attempt, env, intentKey, ref)
			if err != nil || cancelled.State != jobs.Cancelled || cancelled.Quiescence == nil {
				t.Fatalf("optional sidecar affected cancellation: job=%+v err=%v", cancelled, err)
			}
			proofBytes, err := c.readBlob(ctx, cancelled.Quiescence.ProofArtifactID, 64<<10, cancelled.Quiescence.ProofSHA256)
			if err != nil {
				t.Fatal(err)
			}
			var proof persistedQuiescence
			if err := json.Unmarshal(proofBytes, &proof); err != nil {
				t.Fatal(err)
			}
			if scenario == "complete" {
				if proof.InferenceOutcomeStatus != inferenceOutcomeBound || proof.InferenceOutcome == nil || proof.InferenceOutcomeSHA256 != digestBytes(outcomeBytes) {
					t.Fatalf("complete canonical response was not bound: %+v", proof)
				}
			} else if proof.InferenceOutcomeStatus != inferenceOutcomeNotBound || proof.InferenceOutcome != nil || proof.InferenceOutcomeSHA256 != "" {
				t.Fatalf("incomplete response evidence must remain explicitly unbound: %+v", proof)
			}
		})
	}
}

type integrationShot struct{ resolved shots.ResolvedShot }

func (s integrationShot) ResolveForRender(_ context.Context, id string, revision int64) (shots.ResolvedShot, error) {
	if id != s.resolved.Shot.ID || revision != s.resolved.Revision.Revision {
		return shots.ResolvedShot{}, errors.New("unexpected shot revision")
	}
	return s.resolved, nil
}

type unavailableIntegrationShot struct{}

func (unavailableIntegrationShot) ResolveForRender(context.Context, string, int64) (shots.ResolvedShot, error) {
	return shots.ResolvedShot{}, errors.New("current shot resolver must not be consulted")
}

type integrationMedia struct {
	values       map[string]projects.SourceMedia
	data         map[string][]byte
	receiptBytes []byte
	clockBytes   []byte
	projections  []projects.InspectionProjection
}

func (m *integrationMedia) Import(context.Context, string, string, int64, io.Reader) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("unused")
}

func (m *integrationMedia) BeginImport(context.Context, string, string, int64, io.Reader) (projects.StagedSourceMediaImport, error) {
	return nil, errors.New("unused")
}

func (m *integrationMedia) BindInspectionJob(context.Context, string, string, string, string) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("unused")
}

func (m *integrationMedia) BindInspectionAttempt(_ context.Context, projectID, mediaID, sourceSHA, jobID, expectedAttemptID, attemptID string) (projects.SourceMedia, error) {
	row, err := m.Get(context.Background(), projectID, mediaID)
	if err != nil {
		return projects.SourceMedia{}, err
	}
	if row.SHA256 != sourceSHA || row.InspectionAttemptID != expectedAttemptID || (row.InspectionJobID != "" && row.InspectionJobID != jobID) {
		return projects.SourceMedia{}, projects.ErrInspectionConflict
	}
	row.InspectionJobID, row.InspectionAttemptID = jobID, attemptID
	row.Readiness = projects.ReadinessPending
	row.InspectionIssueCode, row.InspectionIssueMessage, row.InspectionReceipt = "", "", nil
	m.values[mediaID] = row
	return row, nil
}

func (m *integrationMedia) StoreInspectionArtifacts(_ context.Context, _, _, _, _, receiptSHA, clockSHA string, receipt, clock []byte) (string, string, error) {
	if digestBytes(receipt) != receiptSHA || (clockSHA == "" && len(clock) != 0) || (clockSHA != "" && digestBytes(clock) != clockSHA) {
		return "", "", errors.New("inspection artifact digest mismatch")
	}
	m.receiptBytes = append([]byte(nil), receipt...)
	m.clockBytes = append([]byte(nil), clock...)
	clockRef := ""
	if clockSHA != "" {
		clockRef = "clock-" + clockSHA
	}
	return "receipt-" + receiptSHA, clockRef, nil
}

func (m *integrationMedia) OpenInspectionArtifacts(_ context.Context, _, _, _, _, receiptSHA, clockSHA string) ([]byte, []byte, error) {
	if digestBytes(m.receiptBytes) != receiptSHA || (clockSHA == "" && len(m.clockBytes) != 0) || (clockSHA != "" && digestBytes(m.clockBytes) != clockSHA) {
		return nil, nil, errors.New("inspection artifact digest mismatch")
	}
	return append([]byte(nil), m.receiptBytes...), append([]byte(nil), m.clockBytes...), nil
}

func (m *integrationMedia) ProjectInspection(_ context.Context, projection projects.InspectionProjection) error {
	m.projections = append(m.projections, projection)
	row := m.values[projection.MediaID]
	row.Readiness = projection.Readiness
	row.InspectionIssueCode = projection.IssueCode
	row.InspectionIssueMessage = projection.IssueMessage
	row.InspectionReceipt = projection.Receipt
	m.values[projection.MediaID] = row
	return nil
}

func (m *integrationMedia) Get(_ context.Context, projectID, id string) (projects.SourceMedia, error) {
	v := m.values[id]
	if v.ProjectID != projectID {
		return projects.SourceMedia{}, errors.New("wrong project")
	}
	return v, nil
}

func (m *integrationMedia) Open(_ context.Context, projectID, id string) (projects.SourceMedia, io.ReadCloser, error) {
	v, err := m.Get(context.Background(), projectID, id)
	if err != nil {
		return v, nil, err
	}
	return v, io.NopCloser(bytes.NewReader(m.data[id])), nil
}

func (m *integrationMedia) List(context.Context, string, int32, string) ([]projects.SourceMedia, string, error) {
	return nil, "", nil
}

type integrationOps struct {
	mu           sync.Mutex
	starts       int
	deferRun     bool
	lastSpec     platform.OperationSpec
	lookups      int
	stagedWorker []byte
	intent       platform.OperationIntent
	ref          platform.OperationRef
	startErr     error
	terminal     bool
	releaseCount int
}

func (o *integrationOps) NewIntent(owner, attempt string, spec platform.OperationSpec) (platform.OperationIntent, error) {
	return platform.NewOperationIntent(owner, attempt, spec)
}

func (o *integrationOps) Start(ctx context.Context, intent platform.OperationIntent, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
	o.mu.Lock()
	o.starts++
	o.lastSpec = spec
	o.intent = intent
	o.ref = platform.OperationRef{Intent: intent, Unit: "device-studio-test.service", NativeInvocationID: "native-invocation"}
	startErr := o.startErr
	o.mu.Unlock()
	data, err := os.ReadFile(spec.Path)
	if err != nil {
		return nil, err
	}
	o.mu.Lock()
	o.stagedWorker = append([]byte(nil), data...)
	o.mu.Unlock()
	if startErr != nil {
		return nil, startErr
	}
	if o.deferRun {
		return nil, nil
	}
	if err := rendering.RunManifest(ctx, filepath.Join(spec.Dir, rendering.ManifestName)); err != nil {
		return nil, err
	}
	o.mu.Lock()
	o.terminal = true
	o.mu.Unlock()
	return nil, nil
}

func TestEnsureInspectionAdvanceDispatchesRegisteredAdapterOnceOnMatchingRoute(t *testing.T) {
	ops := &integrationOps{deferRun: true}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	media := c.media.(*integrationMedia)
	data := []byte("bounded fake video source")
	sourceSHA := digestBytes(data)
	media.values["video"] = projects.SourceMedia{
		ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: int64(len(data)),
		MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending,
	}
	media.data["video"] = data
	profile := &cliv1.RendererToolProfile{ProfileId: "qualified-test-ffmpeg", ArtifactIdentity: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	c.inspectionProfile = func(context.Context) (*cliv1.RendererToolProfile, error) { return profile, nil }

	job, created, err := c.EnsureSourceInspection(ctx, "project", "video")
	if err != nil || !created {
		t.Fatalf("EnsureSourceInspection created=%v err=%v", created, err)
	}
	if media.values["video"].InspectionJobID != job.ID || media.values["video"].InspectionAttemptID == "" {
		t.Fatalf("ensure did not bind the submitted job attempt: %+v", media.values["video"])
	}
	advanced, err := c.Advance(ctx, job.ID)
	if err != nil {
		t.Fatalf("Advance inspection job: %v", err)
	}
	if advanced.State != jobs.Running {
		t.Fatalf("inspection dispatch state = %s, want running", advanced.State)
	}
	ops.mu.Lock()
	starts, spec := ops.starts, ops.lastSpec
	ops.mu.Unlock()
	command := ""
	if len(spec.Args) > 0 {
		command = spec.Args[0]
	}
	if starts != 1 || command != "inspect-worker" {
		t.Fatalf("registered inspection adapter dispatches=%d command=%q, want one inspect-worker dispatch", starts, command)
	}
}

func TestRetryInspectionStartsLatestAttemptAndDispatchesNativeWorker(t *testing.T) {
	ops := &integrationOps{deferRun: true}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	media := c.media.(*integrationMedia)
	data := []byte("bounded fake video source")
	sourceSHA := digestBytes(data)
	media.values["video"] = projects.SourceMedia{ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: int64(len(data)), MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending}
	media.data["video"] = data
	profile := &cliv1.RendererToolProfile{ProfileId: "qualified-test-ffmpeg", ArtifactIdentity: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	c.inspectionProfile = func(context.Context) (*cliv1.RendererToolProfile, error) { return profile, nil }
	job, created, err := c.EnsureSourceInspection(ctx, "project", "video")
	if err != nil || !created {
		t.Fatalf("ensure inspection: created=%v err=%v", created, err)
	}
	first := job.Attempts[0]
	running, err := c.jobs.Start(ctx, job.ID, jobs.ExecutionClaimInput{AttemptID: first.ID, AttemptNumber: first.Number, OwnerID: OwnerID, ScopeID: "first-inspection-attempt", ArtifactID: "first-claim", SHA256: sourceSHA}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := c.jobs.Fail(ctx, running.ID, first.ID, "known failed inspection", running.Version)
	if err != nil {
		t.Fatal(err)
	}
	failed, err = c.jobs.RecordReleased(ctx, failed.ID, first.ID, failed.Version, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	settled, err := c.Advance(ctx, failed.ID)
	if err != nil {
		t.Fatalf("finalize failed attempt before retry: %v", err)
	}
	retried, err := c.RetrySourceInspection(ctx, "project", "video", settled.Version)
	if err != nil || retried.State != jobs.Queued || len(retried.Attempts) != 2 {
		t.Fatalf("retry failed job: state=%s attempts=%d err=%v", retried.State, len(retried.Attempts), err)
	}
	latest := retried.Attempts[1]
	started, err := c.Advance(ctx, retried.ID)
	if err != nil || started.State != jobs.Running || len(started.Attempts) != 2 || started.Attempts[0].State != jobs.AttemptFailed || started.Attempts[0].ExecutionReleasedAt == nil || started.Attempts[1].ID != latest.ID || started.Attempts[1].State != jobs.AttemptRunning {
		t.Fatalf("retry did not start the latest attempt while preserving release history: job=%+v err=%v", started, err)
	}
	ops.mu.Lock()
	starts, intent, spec := ops.starts, ops.intent, ops.lastSpec
	ops.mu.Unlock()
	command := ""
	if len(spec.Args) > 0 {
		command = spec.Args[0]
	}
	if starts != 1 || intent.AttemptID != latest.ID || command != "inspect-worker" {
		t.Fatalf("retry native dispatch starts=%d attempt=%s command=%q, want one dispatch of attempt %s", starts, intent.AttemptID, command, latest.ID)
	}
}

func (o *integrationOps) Lookup(_ context.Context, intent platform.OperationIntent) (platform.OperationRef, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.lookups++
	if intent != o.intent || !o.terminal {
		return platform.OperationRef{}, platform.ErrOperationNotFound
	}
	return o.ref, nil
}

func (o *integrationOps) Inspect(_ context.Context, ref platform.OperationRef) (platform.OperationObservation, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return platform.OperationObservation{Ref: ref, State: platform.OperationStoppedOutcomeKnown, Result: "success", ExecMainCode: 1, ExecMainStatus: 0, ExitKnown: true}, nil
}

func (o *integrationOps) Stop(_ context.Context, ref platform.OperationRef) (platform.OperationStopReceipt, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return platform.OperationStopReceipt{Ref: ref, NativeQuiescent: true}, nil
}

func (o *integrationOps) Release(_ context.Context, ref platform.OperationRef) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if ref != o.ref {
		return platform.ErrOperationIdentity
	}
	o.releaseCount++
	return nil
}

type cancelAfterClaim struct{ jobs.Service }

func (s cancelAfterClaim) Start(ctx context.Context, id string, claim jobs.ExecutionClaimInput, version int64) (jobs.Job, error) {
	j, err := s.Service.Start(ctx, id, claim, version)
	if err != nil {
		return j, err
	}
	return s.Service.RequestCancel(ctx, id, j.Version)
}

type cancelBeforeComplete struct{ jobs.Service }

func (s cancelBeforeComplete) Complete(ctx context.Context, id, attempt string, evidence jobs.CompletionEvidence, version int64) (jobs.Job, error) {
	if _, err := s.Service.RequestCancel(ctx, id, version); err != nil {
		return jobs.Job{}, err
	}
	return s.Service.Complete(ctx, id, attempt, evidence, version)
}

func integrationCoordinator(t *testing.T, ops *integrationOps, cancelAtClaim bool) (*Coordinator, context.Context, string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "jobs.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.Exec(jobs.Schema()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	service := jobs.NewService(jobs.NewSQLiteRepository(db), time.Now)
	if cancelAtClaim {
		service = cancelAfterClaim{service}
	}
	primary, testRoot := t.TempDir(), t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: primary})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: testRoot}, "integration-lease", time.Hour); err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	workerPath := filepath.Join(t.TempDir(), "device-studio-api")
	if err := os.WriteFile(workerPath, []byte("worker-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	media := newIntegrationMedia(t)
	resolved := integrationResolved(media)
	c, err := New(Dependencies{Shots: integrationShot{resolved}, Media: media, Jobs: service, Blobs: blobstore.NewMemoryBlobStore(), Roots: roots, Ops: ops, Exe: func() (string, error) { return workerPath, nil }})
	if err != nil {
		t.Fatal(err)
	}
	return c, ctx, workerPath
}

func newIntegrationMedia(t *testing.T) *integrationMedia {
	t.Helper()
	m := &integrationMedia{values: map[string]projects.SourceMedia{}, data: map[string][]byte{}}
	for id, gray := range map[string]bool{"plate": false, "screen": false, "active": true} {
		var img image.Image
		if gray {
			g := image.NewGray(image.Rect(0, 0, 4, 4))
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					g.SetGray(x, y, color.Gray{Y: 255})
				}
			}
			img = g
		} else {
			rgba := image.NewRGBA(image.Rect(0, 0, 4, 4))
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					rgba.Set(x, y, color.RGBA{R: uint8(20 + x*30), G: uint8(30 + y*25), B: 100, A: 255})
				}
			}
			img = rgba
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatal(err)
		}
		data := buf.Bytes()
		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		model := "rgba"
		if gray {
			model = "grayscale"
		}
		m.data[id] = data
		m.values[id] = projects.SourceMedia{ID: id, ProjectID: "project", SHA256: digest, SizeBytes: int64(len(data)), MediaType: "image/png", Width: 4, Height: 4, ColorModel: model, ColorProfileStatus: "srgb"}
	}
	return m
}

func TestEnsureInspectionRecoversSubmittedJobBeforeResolvingProfile(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	media := c.media.(*integrationMedia)
	const sourceSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	media.values["video"] = projects.SourceMedia{ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: 128, MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending}
	key := inspectionIdempotencyKey("project", "video", sourceSHA)
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: key, Input: jobs.Input{
		ProjectID: "project", RendererVersion: inspectionRendererVersion, OutputProfile: inspectionOutputProfile, ToolProfileDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SubjectKind: jobs.SubjectSourceInspection, SubjectID: "video",
		Bindings: []jobs.ArtifactBinding{
			{Role: "source_media", ArtifactID: "source-media:video", RevisionID: "sha256:" + sourceSHA, SHA256: sourceSHA},
			{Role: "source_size", ArtifactID: "source-media-size:video", RevisionID: "size:128", SHA256: sourceSHA},
			routeBinding(route),
		},
	}})
	if err != nil || !created {
		t.Fatalf("seed durable inspection submission: created=%v err=%v", created, err)
	}
	got, created, err := c.EnsureSourceInspection(ctx, "project", "video")
	if err != nil {
		t.Fatalf("recover existing submission without resolving an unavailable profile: %v", err)
	}
	if created || got.ID != job.ID || media.values["video"].InspectionJobID != job.ID || media.values["video"].InspectionAttemptID != job.Attempts[0].ID {
		t.Fatalf("existing submission was not recovered and bound: created=%v job=%+v media=%+v", created, got, media.values["video"])
	}
}

func TestInspectionRouteBindingRejectsRecoveryAndProgressOnDifferentStorageRoot(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	routeA, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	media := c.media.(*integrationMedia)
	const sourceSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	media.values["video"] = projects.SourceMedia{ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: 128, MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending}
	key := inspectionIdempotencyKey("project", "video", sourceSHA)
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: key, Input: jobs.Input{
		ProjectID: "project", RendererVersion: inspectionRendererVersion, OutputProfile: inspectionOutputProfile, ToolProfileDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SubjectKind: jobs.SubjectSourceInspection, SubjectID: "video",
		Bindings: []jobs.ArtifactBinding{
			{Role: "source_media", ArtifactID: "source-media:video", RevisionID: "sha256:" + sourceSHA, SHA256: sourceSHA},
			{Role: "source_size", ArtifactID: "source-media-size:video", RevisionID: "size:128", SHA256: sourceSHA},
			routeBinding(routeA),
		},
	}})
	if err != nil || !created {
		t.Fatalf("seed route-A inspection job: created=%v err=%v", created, err)
	}

	primaryB, testRootB := t.TempDir(), t.TempDir()
	rootsB := filerouting.New(storage.Paths{DataDir: primaryB})
	if err := rootsB.InstallTestRoots(storage.Paths{DataDir: testRootB}, "other-integration-lease", time.Hour); err != nil {
		t.Fatal(err)
	}
	cB, err := New(Dependencies{Shots: c.shots, Media: c.media, Jobs: c.jobs, Blobs: c.blobs, Roots: rootsB, Ops: ops, Exe: c.exe})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := cB.EnsureSourceInspection(ctx, "project", "video"); !errors.Is(err, ErrRouteMismatch) {
		t.Fatalf("Ensure recovered across a changed storage root: %v", err)
	}
	if _, err := cB.Advance(ctx, job.ID); !errors.Is(err, ErrRouteMismatch) {
		t.Fatalf("Advance progressed on a different storage root: %v", err)
	}
	if media.values["video"].InspectionJobID != "" || ops.starts != 0 {
		t.Fatalf("wrong-route recovery bound or dispatched work: media=%+v starts=%d", media.values["video"], ops.starts)
	}
}

func TestRejectedInspectionReceiptCompletesTerminalSubjectFinalization(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	media := c.media.(*integrationMedia)
	const sourceSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const profileSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const workerSHA = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	media.values["video"] = projects.SourceMedia{ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: 128, MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending}
	profile := &cliv1.RendererToolProfile{ProfileId: "btbn-ffmpeg-video-n9.0.2-28-g669f5ab083-linux64-gpl", ArtifactIdentity: profileSHA}
	profileBytes, err := protojson.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	profileSHAOnDisk := digestBytes(profileBytes)
	profileKey := inputBlobPrefix + profileSHAOnDisk + "/inspection-profile"
	if err := c.blobs.Put(ctx, profileKey, bytes.NewReader(profileBytes), "application/x-protobuf+json"); err != nil {
		t.Fatal(err)
	}
	workerKey := workerBlobPrefix + workerSHA + "/device-studio-api"
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: "rejected-finalization", Input: jobs.Input{
		ProjectID: "project", RendererVersion: inspectionRendererVersion, OutputProfile: inspectionOutputProfile,
		ToolProfileDigest: profileSHA, SubjectKind: jobs.SubjectSourceInspection, SubjectID: "video",
		Bindings: []jobs.ArtifactBinding{
			{Role: "source_media", ArtifactID: "source-media:video", RevisionID: "sha256:" + sourceSHA, SHA256: sourceSHA},
			{Role: "source_size", ArtifactID: "source-media-size:video", RevisionID: "size:128", SHA256: sourceSHA},
			{Role: "inspection_profile", ArtifactID: profileKey, RevisionID: "sha256:" + profileSHAOnDisk, SHA256: profileSHAOnDisk},
			{Role: "worker_executable", ArtifactID: workerKey, RevisionID: "sha256:" + workerSHA, SHA256: workerSHA},
			routeBinding(route),
		},
	}})
	if err != nil || !created {
		t.Fatalf("submit rejected inspection: created=%v err=%v", created, err)
	}
	attempt := job.Attempts[0]
	row := media.values["video"]
	row.InspectionJobID, row.InspectionAttemptID = job.ID, attempt.ID
	media.values["video"] = row
	job, err = c.jobs.Start(ctx, job.ID, jobs.ExecutionClaimInput{AttemptID: attempt.ID, AttemptNumber: 1, OwnerID: OwnerID, ScopeID: "rejected-inspection-test", ArtifactID: "claim", SHA256: sourceSHA}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := inspectionManifestFromInput(job, job.Attempts[0], profile)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	receipt := rendering.InspectionReceipt{
		Version: 1, JobID: job.ID, AttemptID: attempt.ID, InputFingerprint: job.InputFingerprint, WorkerSHA256: workerSHA,
		ManifestSHA256: digestBytes(manifestBytes), SourceSHA256: sourceSHA, SourceSizeBytes: 128,
		ProfileID: profile.GetProfileId(), ProfileDigest: profileSHA, ProtocolVersion: 1,
		Classification: rendering.InspectionRejected, IssueCode: "video_raster_unsupported", IssueMessage: "The source raster is outside the supported profile.",
	}
	media.receiptBytes, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	receiptSHA := digestBytes(media.receiptBytes)
	job, err = c.jobs.Complete(ctx, job.ID, attempt.ID, jobs.CompletionEvidence{
		AttemptID: attempt.ID, ExecutionClaimID: job.Attempts[0].Execution.ID, InputFingerprint: job.InputFingerprint,
		Outputs:           []jobs.OutputRef{{Kind: "inspection_receipt", ArtifactID: "opaque-receipt", SHA256: receiptSHA, MIMEType: "application/json", SizeBytes: int64(len(media.receiptBytes))}},
		ReceiptArtifactID: "opaque-receipt", ReceiptSHA256: receiptSHA, CompletedAt: time.Now().UTC(),
	}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.finalizeTerminal(ctx, route, job); err != nil {
		t.Fatalf("verified worker rejection did not finalize its source state: %v", err)
	}
	if len(media.projections) != 1 || media.projections[0].Readiness != projects.ReadinessRejected || media.projections[0].Receipt == nil || media.projections[0].Receipt.Classification != "rejected" || media.projections[0].Receipt.FrameCount != 0 || media.projections[0].Receipt.ClockSHA256 != "" {
		t.Fatalf("rejection did not project only its observed facts: %+v", media.projections)
	}
	if _, err := c.jobs.PendingSubjectFinalization(ctx, job.ID); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("terminal finalization marker remains pending after rejection projection: %v", err)
	}
}

func TestNativeRejectedInspectionWithoutClockCompletesAndProjects(t *testing.T) {
	ops := &integrationOps{deferRun: true}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	media := c.media.(*integrationMedia)
	source := bytes.Repeat([]byte{0x7f}, 128)
	sourceSHA := digestBytes(source)
	media.values["video"] = projects.SourceMedia{ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: int64(len(source)), MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending}
	media.data["video"] = source
	profile := &cliv1.RendererToolProfile{ProfileId: "btbn-ffmpeg-video-n9.0.2-28-g669f5ab083-linux64-gpl", ArtifactIdentity: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	c.inspectionProfile = func(context.Context) (*cliv1.RendererToolProfile, error) { return profile, nil }

	job, created, err := c.EnsureSourceInspection(ctx, "project", "video")
	if err != nil || !created {
		t.Fatalf("ensure inspection: created=%v err=%v", created, err)
	}
	running, err := c.Advance(ctx, job.ID)
	if err != nil || running.State != jobs.Running {
		t.Fatalf("dispatch inspection: state=%s err=%v", running.State, err)
	}
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := attemptWorkspace(route.Root, job.ID, running.Attempts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := readWorkspaceFile(workspace, rendering.InspectionManifestName, maxManifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	var manifest rendering.InspectionManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	receipt, err := json.Marshal(rendering.InspectionReceipt{
		Version: 1, JobID: job.ID, AttemptID: running.Attempts[0].ID, InputFingerprint: job.InputFingerprint,
		WorkerSHA256: manifest.WorkerSHA256, ManifestSHA256: digestBytes(manifestBytes), SourceSHA256: sourceSHA,
		SourceSizeBytes: int64(len(source)), ProfileID: manifest.ProfileID, ProfileDigest: manifest.ProfileDigest,
		ProtocolVersion: 1, Classification: rendering.InspectionRejected, IssueCode: "stream_count_unsupported",
		IssueMessage: "The source must contain between one and eight streams.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, rendering.InspectionReceiptName), receipt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace, rendering.InspectionClockName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected inspection unexpectedly had clock artifact: %v", err)
	}
	ops.mu.Lock()
	ops.terminal = true
	ops.mu.Unlock()
	completed, err := c.Advance(ctx, job.ID)
	if err != nil || completed.State != jobs.Succeeded {
		t.Fatalf("successful native rejection completion: state=%s err=%v", completed.State, err)
	}
	if _, err := c.Advance(ctx, job.ID); err != nil {
		t.Fatalf("project rejected receipt and acknowledge finalization: %v", err)
	}
	projected := media.values["video"]
	if projected.Readiness != projects.ReadinessRejected || projected.InspectionReceipt == nil || projected.InspectionReceipt.Classification != "rejected" || projected.InspectionReceipt.ClockSHA256 != "" || projected.InspectionIssueCode != "stream_count_unsupported" {
		t.Fatalf("rejected receipt without clock was not projected truthfully: %+v", projected)
	}
}

func TestKnownInspectionFailureAndCancellationFinalizeAsRetryableRejection(t *testing.T) {
	for _, terminal := range []jobs.State{jobs.Failed, jobs.Cancelled} {
		t.Run(string(terminal), func(t *testing.T) {
			ops := &integrationOps{}
			c, ctx, _ := integrationCoordinator(t, ops, false)
			media := c.media.(*integrationMedia)
			data := []byte("bounded fake video source")
			sourceSHA := digestBytes(data)
			media.values["video"] = projects.SourceMedia{ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: int64(len(data)), MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending}
			media.data["video"] = data
			profile := &cliv1.RendererToolProfile{ProfileId: "qualified-test-ffmpeg", ArtifactIdentity: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
			c.inspectionProfile = func(context.Context) (*cliv1.RendererToolProfile, error) { return profile, nil }
			job, created, err := c.EnsureSourceInspection(ctx, "project", "video")
			if err != nil || !created {
				t.Fatalf("EnsureSourceInspection created=%v err=%v", created, err)
			}
			attempt := job.Attempts[0]
			if terminal == jobs.Failed {
				job, err = c.jobs.Start(ctx, job.ID, jobs.ExecutionClaimInput{AttemptID: attempt.ID, AttemptNumber: 1, OwnerID: OwnerID, ScopeID: "known-failed-inspection", ArtifactID: "claim", SHA256: sourceSHA}, job.Version)
				if err != nil {
					t.Fatal(err)
				}
				job, err = c.jobs.Fail(ctx, job.ID, attempt.ID, "known terminal failure", job.Version)
				if err != nil {
					t.Fatal(err)
				}
				job, err = c.jobs.RecordReleased(ctx, job.ID, attempt.ID, job.Version, time.Now().UTC())
				if err != nil {
					t.Fatal(err)
				}
			} else {
				job, err = c.Cancel(ctx, job.ID, job.Version)
				if err != nil {
					t.Fatal(err)
				}
			}
			settled, err := c.Advance(ctx, job.ID)
			if err != nil {
				t.Fatalf("terminal finalization: %v", err)
			}
			row := media.values["video"]
			wantCode := "inspection_execution_" + string(terminal)
			if row.Readiness != projects.ReadinessRejected || row.InspectionIssueCode != wantCode || row.InspectionReceipt != nil || row.InspectionAttemptID != attempt.ID {
				t.Fatalf("known terminal outcome did not project as receipt-free rejection: %+v", row)
			}
			if _, err := c.jobs.PendingSubjectFinalization(ctx, job.ID); !errors.Is(err, jobs.ErrNotFound) {
				t.Fatalf("terminal finalization marker was not acknowledged: %v", err)
			}
			retried, err := c.RetrySourceInspection(ctx, "project", "video", settled.Version)
			if err != nil {
				t.Fatalf("known terminal outcome was not retryable: %v", err)
			}
			row = media.values["video"]
			if retried.State != jobs.Queued || len(retried.Attempts) != 2 || row.Readiness != projects.ReadinessPending || row.InspectionAttemptID != retried.Attempts[1].ID || row.InspectionIssueCode != "" || row.InspectionReceipt != nil {
				t.Fatalf("retry did not create a fresh pending attempt: job=%+v media=%+v", retried, row)
			}
		})
	}
}

func TestUnknownInspectionOutcomeCannotFinalizeAsKnownRejection(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	media := c.media.(*integrationMedia)
	data := []byte("bounded fake video source")
	sourceSHA := digestBytes(data)
	media.values["video"] = projects.SourceMedia{ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: int64(len(data)), MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending}
	media.data["video"] = data
	profile := &cliv1.RendererToolProfile{ProfileId: "qualified-test-ffmpeg", ArtifactIdentity: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	c.inspectionProfile = func(context.Context) (*cliv1.RendererToolProfile, error) { return profile, nil }
	job, _, err := c.EnsureSourceInspection(ctx, "project", "video")
	if err != nil {
		t.Fatal(err)
	}
	pending := jobs.SubjectFinalization{JobID: job.ID, AttemptID: job.Attempts[0].ID, Kind: jobs.SubjectSourceInspection, State: jobs.Running}
	if err := (sourceInspectionAdapter{}).finalize(ctx, c, pending); err == nil {
		t.Fatal("unknown native outcome was projected as a known terminal result")
	}
	if len(media.projections) != 0 || media.values["video"].Readiness != projects.ReadinessPending {
		t.Fatalf("unknown outcome changed media readiness: projections=%d media=%+v", len(media.projections), media.values["video"])
	}
}

type inspectionSubmitRaceService struct {
	jobs.Service
	key         string
	firstLookup bool
	submitErr   error
}

func (s *inspectionSubmitRaceService) FindByIdempotencyKey(ctx context.Context, key string) (jobs.Job, error) {
	if key == s.key && !s.firstLookup {
		s.firstLookup = true
		return jobs.Job{}, jobs.ErrNotFound
	}
	return s.Service.FindByIdempotencyKey(ctx, key)
}

func (s *inspectionSubmitRaceService) Submit(context.Context, jobs.SubmitInput) (jobs.Job, bool, error) {
	return jobs.Job{}, false, s.submitErr
}

func TestEnsureInspectionRecoversConcurrentProfileSnapshotWinner(t *testing.T) {
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	media := c.media.(*integrationMedia)
	const sourceSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	media.values["video"] = projects.SourceMedia{ID: "video", ProjectID: "project", SHA256: sourceSHA, SizeBytes: 128, MediaType: "video/mp4", Kind: projects.MediaKindVideo, Readiness: projects.ReadinessPending}
	key := inspectionIdempotencyKey("project", "video", sourceSHA)
	winner, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: key, Input: jobs.Input{
		ProjectID: "project", RendererVersion: inspectionRendererVersion, OutputProfile: inspectionOutputProfile, ToolProfileDigest: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		SubjectKind: jobs.SubjectSourceInspection, SubjectID: "video",
		Bindings: []jobs.ArtifactBinding{
			{Role: "source_media", ArtifactID: "source-media:video", RevisionID: "sha256:" + sourceSHA, SHA256: sourceSHA},
			{Role: "source_size", ArtifactID: "source-media-size:video", RevisionID: "size:128", SHA256: sourceSHA},
			routeBinding(route),
		},
	}})
	if err != nil || !created {
		t.Fatalf("seed concurrent winner: created=%v err=%v", created, err)
	}
	service := &inspectionSubmitRaceService{Service: c.jobs, key: key, submitErr: jobs.ErrIdempotencyConflict}
	c.jobs = service
	c.inspectionProfile = func(context.Context) (*cliv1.RendererToolProfile, error) {
		return &cliv1.RendererToolProfile{ProfileId: "profile-newer", ArtifactIdentity: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}, nil
	}
	got, created, err := c.EnsureSourceInspection(ctx, "project", "video")
	if err != nil {
		t.Fatalf("recover committed winner after profile conflict: %v", err)
	}
	if created || got.ID != winner.ID || media.values["video"].InspectionJobID != winner.ID || media.values["video"].InspectionAttemptID != winner.Attempts[0].ID {
		t.Fatalf("concurrent ensure did not preserve original immutable winner: created=%v job=%+v media=%+v", created, got, media.values["video"])
	}
}

func integrationResolved(m *integrationMedia) shots.ResolvedShot {
	media := func(id string) *shotsv1.ResolvedSourceMedia {
		v := m.values[id]
		return &shotsv1.ResolvedSourceMedia{Id: id, Sha256: v.SHA256, SizeBytes: v.SizeBytes, MediaType: v.MediaType, WidthPixels: v.Width, HeightPixels: v.Height}
	}
	snapshot := &shotsv1.ShotSnapshot{Scene: &shotsv1.PhotographicScene{PlateMediaId: "plate", ActiveMaskMediaId: "active", DisplayWidthPixels: 4, DisplayHeightPixels: 4, CoordinateConvention: shotsv1.CoordinateConvention_COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1, PlateQuad: &shotsv1.PlateQuad{TopLeft: &shotsv1.ShotPoint{X: 0, Y: 0}, TopRight: &shotsv1.ShotPoint{X: 4, Y: 0}, BottomRight: &shotsv1.ShotPoint{X: 4, Y: 4}, BottomLeft: &shotsv1.ShotPoint{X: 0, Y: 4}}}, Presentation: &shotsv1.ScreenPresentation{SourceMediaId: "screen", CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY, Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PLAIN, FitMode: shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN}, MotionMode: shotsv1.MotionMode_MOTION_MODE_FIXED}
	return shots.ResolvedShot{ProjectID: "project", Shot: shots.Shot{ID: "shot", ProjectID: "project"}, Revision: shots.Revision{ShotID: "shot", Revision: 1, Snapshot: snapshot, RecipeSHA256: digestBytes([]byte("recipe")), SceneSHA256: digestBytes([]byte("scene")), PresentationSHA256: digestBytes([]byte("presentation")), SceneVersion: "device-studio-scene-v1", PresentationVersion: "device-studio-presentation-v1", CompositorProfile: rendering.CompositorProfile, RenderEligible: true, Plate: media("plate"), ActiveMask: media("active"), Screen: media("screen")}}
}

func submitIntegrationJob(t *testing.T, c *Coordinator, ctx context.Context) jobs.Job {
	t.Helper()
	j, created, err := c.Submit(ctx, SubmitRequest{ProjectID: "project", ShotID: "shot", Revision: 1, IdempotencyKey: "integration"})
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected a new job")
	}
	return j
}

func TestListProjectJobsScansAcrossInspectionOnlyPrefixes(t *testing.T) {
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	const digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for i := 0; i < 105; i++ {
		n := strconv.Itoa(i)
		input := jobs.Input{ProjectID: "project", RendererVersion: inspectionRendererVersion, OutputProfile: inspectionOutputProfile, ToolProfileDigest: digest, SubjectKind: jobs.SubjectSourceInspection, SubjectID: "media-" + n, Bindings: []jobs.ArtifactBinding{
			routeBinding(route),
			{Role: "source_media", ArtifactID: "source-media:media-" + n, RevisionID: "sha256:" + digest, SHA256: digest},
			{Role: "source_size", ArtifactID: "source-media-size:media-" + n, RevisionID: "size:128", SHA256: digest},
		}}
		if _, _, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: "inspection-" + n, Input: input}); err != nil {
			t.Fatalf("submit inspection prefix %d: %v", i, err)
		}
	}
	for i := 0; i < 3; i++ {
		input := jobs.Input{ProjectID: "project", ShotRevisionID: "shot:older-" + strconv.Itoa(i) + ":revision:1", ShotRevision: 1, SourceClockDigest: digest, RendererVersion: "still-v1", OutputProfile: OutputProfilePNG, ToolProfileDigest: digest, Bindings: []jobs.ArtifactBinding{routeBinding(route)}}
		if _, _, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: "still-" + strconv.Itoa(i), Input: input}); err != nil {
			t.Fatalf("submit still render %d: %v", i, err)
		}
	}
	rows, err := c.ListProjectJobs(ctx, "project", jobs.Cursor{}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("visible renders after hidden prefix = %d, want 3", len(rows))
	}
	for i, row := range rows {
		if row.Input.SubjectKind != "" || row.ID == "" {
			t.Fatalf("row %d is not a public still render: %+v", i, row)
		}
	}
}

func TestDispatchUsesPinnedExecutableAndNeverRedispatchesAmbiguousStart(t *testing.T) { // [REQ:DS-JOB-DURABILITY][REQ:DS-JOB-RECOVERY]
	ops := &integrationOps{startErr: errors.New("injected ambiguous start")}
	c, ctx, workerPath := integrationCoordinator(t, ops, false)
	j := submitIntegrationJob(t, c, ctx)
	if err := os.WriteFile(workerPath, []byte("worker-v2"), 0o700); err != nil {
		t.Fatal(err)
	}
	c.shots = unavailableIntegrationShot{}
	c.media = nil
	reused, created, err := c.Submit(ctx, SubmitRequest{ProjectID: "project", ShotID: "shot", Revision: 1, IdempotencyKey: "integration"})
	if err != nil || created || reused.ID != j.ID {
		t.Fatalf("idempotent retry consulted mutable sources or changed accepted job: job=%s created=%v err=%v", reused.ID, created, err)
	}
	if _, _, err := c.Submit(ctx, SubmitRequest{ProjectID: "project", ShotID: "different-shot", Revision: 1, IdempotencyKey: "integration"}); !errors.Is(err, jobs.ErrIdempotencyConflict) {
		t.Fatalf("same key with a different stable request should conflict, got %v", err)
	}
	if _, err := c.Advance(ctx, j.ID); err == nil {
		t.Fatal("expected ambiguous native start")
	}
	ops.mu.Lock()
	starts := ops.starts
	staged := string(ops.stagedWorker)
	ops.mu.Unlock()
	if starts != 1 || staged != "worker-v1" {
		t.Fatalf("dispatch count/staged executable=%d/%q", starts, staged)
	}
	if _, err := c.Advance(ctx, j.ID); !errors.Is(err, ErrUnknownOutcome) {
		t.Fatalf("ambiguous operation should require reconciliation, got %v", err)
	}
	ops.mu.Lock()
	defer ops.mu.Unlock()
	if ops.starts != 1 {
		t.Fatalf("ambiguous operation redispatched %d times", ops.starts)
	}
}

func TestTestRoutedJobCannotFallBackToPrimaryStorageRoute(t *testing.T) {
	ops := &integrationOps{startErr: errors.New("must not dispatch")}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	j := submitIntegrationJob(t, c, ctx)
	if _, err := c.Advance(context.Background(), j.ID); !errors.Is(err, ErrRouteMismatch) {
		t.Fatalf("lost test route must fail closed, got %v", err)
	}
	ops.mu.Lock()
	starts := ops.starts
	ops.mu.Unlock()
	if starts != 0 {
		t.Fatalf("wrong route dispatched %d native operations", starts)
	}
	if got := c.roots.LeaseStats().PrimaryWritesDuringTestMode; got != 0 {
		t.Fatalf("writes escaped to primary during test route: %d", got)
	}
}

func TestCancellationDuringStagingWinsBeforeNativeStart(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, true)
	j := submitIntegrationJob(t, c, ctx)
	got, err := c.Advance(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != jobs.Cancelled || got.Quiescence == nil || !got.Quiescence.NoOwnedExecution {
		t.Fatalf("staging cancellation did not persist predispatch proof: %#v", got)
	}
	ops.mu.Lock()
	defer ops.mu.Unlock()
	if ops.starts != 0 {
		t.Fatalf("native start raced a persisted cancellation: %d", ops.starts)
	}
}

func TestConcurrentAdvanceHasOneWorkspaceAndOneNativeDispatchWinner(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	j := submitIntegrationJob(t, c, ctx)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := c.Advance(ctx, j.ID); results <- err }()
	}
	wg.Wait()
	close(results)
	// Exactly one dispatch succeeds; the CAS loser cannot alter the winner's
	// staged workspace or dispatch a second native operation.
	var successes int
	var failures []string
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures = append(failures, err.Error())
		}
	}
	ops.mu.Lock()
	starts := ops.starts
	ops.mu.Unlock()
	if successes == 0 || successes > 2 || starts != 1 {
		t.Fatalf("dispatch successes/native starts = %d/%d; advance errors=%v", successes, starts, failures)
	}
	completed, err := c.Reconcile(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != jobs.Succeeded {
		t.Fatalf("winner did not reconcile: %s", completed.State)
	}
}

func TestCancellationWinningCompletionRaceRetainsUnitUntilQuiescence(t *testing.T) { // [REQ:DS-JOB-CANCEL]
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	j := submitIntegrationJob(t, c, ctx)
	if _, err := c.Advance(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	c.jobs = cancelBeforeComplete{c.jobs}
	if _, err := c.Reconcile(ctx, j.ID); err == nil {
		t.Fatal("completion should lose the serialized cancellation race")
	}
	requested, err := c.jobs.Get(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if requested.State != jobs.CancelRequested {
		t.Fatalf("cancel did not win: %s", requested.State)
	}
	ops.mu.Lock()
	releasedBeforeProof := ops.releaseCount
	ops.mu.Unlock()
	if releasedBeforeProof != 0 {
		t.Fatal("native unit was released before cancellation quiescence was persisted")
	}
	cancelled, err := c.Cancel(ctx, j.ID, requested.Version)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != jobs.Cancelled || cancelled.Quiescence == nil || !cancelled.Quiescence.NoOwnedExecution {
		t.Fatalf("cancellation proof missing: %#v", cancelled)
	}
	ops.mu.Lock()
	defer ops.mu.Unlock()
	if ops.releaseCount != 1 {
		t.Fatalf("exact unit releases after proof = %d", ops.releaseCount)
	}
}

func TestSuccessPublishesVerifiedPNGAndTamperFailsAfterNativeQuiescence(t *testing.T) { // [REQ:DS-PHOTO-REPLAY][REQ:DS-JOB-DURABILITY]
	for _, tamper := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "tampered-output"}[tamper], func(t *testing.T) {
			ops := &integrationOps{}
			c, ctx, _ := integrationCoordinator(t, ops, false)
			j := submitIntegrationJob(t, c, ctx)
			if _, err := c.Advance(ctx, j.ID); err != nil {
				t.Fatal(err)
			}
			workspace, _ := attemptWorkspace(func() string { r, _ := c.route(ctx); return r.Root }(), j.ID, j.Attempts[0].ID)
			if tamper {
				if err := os.WriteFile(filepath.Join(workspace, rendering.OutputName), []byte("changed"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := c.Reconcile(ctx, j.ID)
			if tamper {
				if err == nil || got.State != jobs.Failed {
					t.Fatalf("tampered artifact should fail only after native terminal proof; state=%s err=%v", got.State, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.State != jobs.Succeeded || len(got.Attempts[0].Completion.Outputs) != 1 {
				t.Fatalf("verified output not completed: %#v", got)
			}
		})
	}
}
