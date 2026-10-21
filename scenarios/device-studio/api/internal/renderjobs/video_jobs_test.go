package renderjobs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/jobs"
	"device-studio/internal/presentations"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	platform "github.com/vrooli/platform-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestVideoCompletionAcceptsExactVideoNativeEvidenceBeforeReadingArtifacts(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	jobID, attemptID := "video-native-job", "video-native-attempt"
	workerSHA := digestBytes([]byte("pinned worker"))
	inputFingerprint := digestBytes([]byte("immutable video input"))
	job := jobs.Job{ID: jobID, InputFingerprint: inputFingerprint, Input: jobs.Input{Bindings: []jobs.ArtifactBinding{
		routeBinding(route), {Role: "worker_executable", ArtifactID: "worker", SHA256: workerSHA},
	}}}
	workspace, err := attemptWorkspace(route.Root, jobID, attemptID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	spec, err := operationSpecFor(operationVideoRender, filepath.Join(workspace, workerFileName), filepath.Join(workspace, rendering.VideoManifestName), workspace)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := platform.NewOperationIntent(OwnerID, attemptID, spec)
	if err != nil {
		t.Fatal(err)
	}
	attempt := jobs.Attempt{ID: attemptID, Execution: &jobs.ExecutionClaim{ID: "claim-video", JobID: jobID, AttemptID: attemptID}}
	env := persistedIntent{Version: 1, JobID: jobID, AttemptID: attemptID, InputFingerprint: inputFingerprint, Route: route, Intent: intent, Spec: spec, ManifestSHA256: digestBytes([]byte("manifest")), WorkerSHA256: workerSHA, Mode: operationVideoRender}
	ref := platform.OperationRef{Intent: intent, Unit: "device-studio-video-test.service", NativeInvocationID: "video-native-invocation"}
	obs := platform.OperationObservation{Ref: ref, State: platform.OperationStoppedOutcomeKnown, ExitKnown: true}
	evidence := nativeOutcomeEvidence{Version: 1, JobID: jobID, AttemptID: attemptID, InputFingerprint: inputFingerprint, ExecutionClaimID: attempt.Execution.ID, Route: route, Intent: intent, Ref: ref, Observation: obs}
	evidenceBytes, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	evidenceSHA := digestBytes(evidenceBytes)
	evidenceKey := operationEvidenceBlobPrefix + jobID + "/" + attemptID + "/native-" + evidenceSHA + ".json"
	if err := c.putVerified(ctx, route, evidenceKey, evidenceBytes, "application/json", evidenceSHA); err != nil {
		t.Fatal(err)
	}
	_, err = (videoSubjectAdapter{}).validateCompletion(ctx, c, route, job, attempt, env, evidenceKey, evidenceSHA)
	if err == nil || !errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "native evidence reference is invalid") {
		t.Fatalf("video native evidence must pass its mode-aware claim checks before the expected missing-manifest error; got %v", err)
	}
}

func TestPhysicalMotionCompletionIsServedByVerifiedVideoOutputOwner(t *testing.T) {
	testPhysicalMotionCompletionIsServedByVerifiedVideoOutputOwner(t, OutputProfilePhysicalMotionMP4)
}

func TestPhysicalMotionV2CompletionIsServedByVerifiedVideoOutputOwner(t *testing.T) {
	testPhysicalMotionCompletionIsServedByVerifiedVideoOutputOwner(t, OutputProfilePhysicalMotionMP4V2)
}

func TestPhysicalRecordingCompletionIsServedByVerifiedVideoOutputOwner(t *testing.T) {
	testPhysicalMotionCompletionIsServedByVerifiedVideoOutputOwner(t, OutputProfilePhysicalRecordingMP4)
}

func testPhysicalMotionCompletionIsServedByVerifiedVideoOutputOwner(t *testing.T, profile string) {
	t.Helper()
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 64)
	job, _, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: "physical-motion-output-" + profile, Input: jobs.Input{
		ProjectID: "project", ShotRevisionID: "shot:revision:1", ShotRevision: 1,
		ToolProfileDigest: strings.Repeat("b", 64), RendererVersion: "physical-cycles-v1",
		OutputProfile: profile, SubjectKind: jobs.SubjectPhysicalRender, SubjectID: "shot",
		Bindings: []jobs.ArtifactBinding{routeBinding(route), {Role: "shot_snapshot", ArtifactID: "shot:snapshot", RevisionID: "revision:1", SHA256: sha}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	attempt := job.Attempts[0]
	job, err = c.jobs.Start(ctx, job.ID, jobs.ExecutionClaimInput{AttemptID: attempt.ID, AttemptNumber: 1, OwnerID: OwnerID, ScopeID: "physical-motion-output-scope", ArtifactID: "intent", SHA256: sha}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	output := []byte("verified mp4 fixture bytes")
	outputSHA := digestBytes(output)
	outputKey := outputBlobPrefix + job.ID + "/" + attempt.ID + "/" + outputSHA + ".mp4"
	if err := c.blobs.Put(ctx, outputKey, bytes.NewReader(output), "video/mp4"); err != nil {
		t.Fatal(err)
	}
	job, err = c.jobs.Complete(ctx, job.ID, attempt.ID, jobs.CompletionEvidence{
		AttemptID: attempt.ID, ExecutionClaimID: job.Attempts[0].Execution.ID, InputFingerprint: job.InputFingerprint,
		Outputs:           []jobs.OutputRef{{Kind: "physical_motion_mp4", ArtifactID: outputKey, SHA256: outputSHA, MIMEType: "video/mp4", SizeBytes: int64(len(output)), Width: 1920, Height: 1080}},
		ReceiptArtifactID: "physical-receipt", ReceiptSHA256: sha, CompletedAt: time.Now().UTC(),
	}, job.Version)
	if err != nil || job.State != jobs.Succeeded {
		t.Fatalf("complete physical MP4 job: state=%q err=%v", job.State, err)
	}
	opened, ref, reader, err := c.OpenVideoOutput(ctx, "project", job.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if opened.ID != job.ID || ref.MIMEType != "video/mp4" || ref.ArtifactID != outputKey || !bytes.Equal(got, output) {
		t.Fatalf("physical MP4 output projection mismatch: job=%s ref=%+v bytesEqual=%t", opened.ID, ref, bytes.Equal(got, output))
	}
}

type testReadCloser struct{ *bytes.Reader }

func (r testReadCloser) Close() error { return nil }

type zeroOnceReader struct {
	reader *strings.Reader
	zeroed bool
}

func (r *zeroOnceReader) Read(p []byte) (int, error) {
	if !r.zeroed {
		r.zeroed = true
		return 0, nil
	}
	return r.reader.Read(p)
}

func (r *zeroOnceReader) Close() error { return nil }

func TestVerifiedOutputReaderRequiresCompleteExactBytes(t *testing.T) {
	data := []byte("bounded mp4 bytes")
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	t.Run("valid", func(t *testing.T) {
		r := &verifiedOutputReader{reader: testReadCloser{bytes.NewReader(data)}, expectedSize: int64(len(data)), expectedSHA: sha, hash: sha256.New()}
		got, err := io.ReadAll(r)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("read output: %q %v", got, err)
		}
		if err := r.Close(); err != nil {
			t.Fatalf("close verified stream: %v", err)
		}
	})
	t.Run("corrupt digest", func(t *testing.T) {
		r := &verifiedOutputReader{reader: testReadCloser{bytes.NewReader(data)}, expectedSize: int64(len(data)), expectedSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", hash: sha256.New()}
		got, err := io.ReadAll(r)
		if err == nil {
			t.Fatal("digest mismatch was reported as a complete stream")
		}
		if len(got) != len(data)-1 || !bytes.Equal(got, data[:len(data)-1]) {
			t.Fatalf("corrupt stream exposed final byte: got %d bytes, want %d", len(got), len(data)-1)
		}
	})
	t.Run("one byte exact size", func(t *testing.T) {
		one := []byte{0x7f}
		oneSum := sha256.Sum256(one)
		r := &verifiedOutputReader{reader: testReadCloser{bytes.NewReader(one)}, expectedSize: 1, expectedSHA: hex.EncodeToString(oneSum[:]), hash: sha256.New()}
		got, err := io.ReadAll(r)
		if err != nil || !bytes.Equal(got, one) {
			t.Fatalf("one-byte stream=(%x,%v)", got, err)
		}
		if err := r.Close(); err != nil {
			t.Fatalf("close verified stream: %v", err)
		}
	})
	t.Run("large corrupt response withholds final byte", func(t *testing.T) {
		large := bytes.Repeat([]byte("v"), 128<<10)
		r := &verifiedOutputReader{reader: testReadCloser{bytes.NewReader(large)}, expectedSize: int64(len(large)), expectedSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", hash: sha256.New()}
		got, err := io.ReadAll(r)
		if err == nil {
			t.Fatal("large digest mismatch was reported as complete")
		}
		if len(got) != len(large)-1 || !bytes.Equal(got, large[:len(large)-1]) {
			t.Fatalf("large corrupt stream exposed %d bytes, want %d", len(got), len(large)-1)
		}
	})
	t.Run("zero nil probe does not lose final byte", func(t *testing.T) {
		data := []byte("x")
		sum := sha256.Sum256(data)
		r := &verifiedOutputReader{reader: &zeroOnceReader{reader: strings.NewReader(string(data))}, expectedSize: 1, expectedSHA: hex.EncodeToString(sum[:]), hash: sha256.New()}
		got, err := io.ReadAll(r)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("zero-nil source stream=(%x,%v)", got, err)
		}
	})
	t.Run("truncated", func(t *testing.T) {
		r := &verifiedOutputReader{reader: testReadCloser{bytes.NewReader(data[:len(data)-1])}, expectedSize: int64(len(data)), expectedSHA: sha, hash: sha256.New()}
		if _, err := io.ReadAll(r); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("truncated output error=%v", err)
		}
	})
	t.Run("partial close", func(t *testing.T) {
		r := &verifiedOutputReader{reader: testReadCloser{bytes.NewReader(data)}, expectedSize: int64(len(data)), expectedSHA: sha, hash: sha256.New()}
		buf := make([]byte, 2)
		if n, err := r.Read(buf); err != nil || n != len(buf) {
			t.Fatalf("partial read=(%d,%v)", n, err)
		}
		if err := r.Close(); err == nil {
			t.Fatal("early close was reported as verified")
		}
	})
}

func TestManualAndBrowserVideoSamplesPreviewWithoutRevisionBinding(t *testing.T) {
	manual := presentations.Config{Target: image.Pt(1080, 1920), Phone: &presentations.PhoneConfig{TopFill: &presentations.RGB{R: 10}, BottomFill: &presentations.RGB{B: 10}}}
	browser := presentations.Config{Target: image.Pt(1920, 1080), Browser: &presentations.BrowserConfig{}}
	autoPhone := presentations.Config{Target: image.Pt(1080, 1920), Phone: &presentations.PhoneConfig{ProfileVersion: presentations.PhoneProfileVersionAuto}}
	for name, cfg := range map[string]presentations.Config{"manual phone": manual, "browser desktop": browser} {
		t.Run(name, func(t *testing.T) {
			if !requiresPinnedVideoLayout(rendering.VideoModeSample, cfg) {
				t.Fatal("manual/browser sample preview must pin the explicit layout")
			}
			if isAutoPhoneVideoSample(rendering.VideoModeSample, cfg) {
				t.Fatal("manual/browser preview must not be eligible for source-derived sample binding")
			}
			if !requiresPinnedVideoLayout(rendering.VideoModeFinal, cfg) {
				t.Fatal("final render must use the pinned presentation layout")
			}
		})
	}
	if requiresPinnedVideoLayout(rendering.VideoModeSample, autoPhone) || !isAutoPhoneVideoSample(rendering.VideoModeSample, autoPhone) {
		t.Fatal("AUTO phone sample must derive its layout from the verified trim-start frame")
	}
	if !requiresPinnedVideoLayout(rendering.VideoModeFinal, autoPhone) {
		t.Fatal("AUTO phone final render must use its saved explicit/resolved layout")
	}
}

func TestPreparedVideoPlanStagesAcceptedInspectionManifestThroughExecutorStager(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	media := c.media.(*integrationMedia)
	putMedia := func(id string, kind projects.MediaKind, mime string, data []byte) string {
		digest := digestBytes(data)
		media.values[id] = projects.SourceMedia{ID: id, ProjectID: "project", Kind: kind, MediaType: mime, SHA256: digest, SizeBytes: int64(len(data)), Readiness: projects.ReadinessReady}
		media.data[id] = append([]byte(nil), data...)
		return digest
	}
	sourceSHA := putMedia("video-source", projects.MediaKindVideo, "video/mp4", []byte("video bytes"))
	plateSHA := putMedia("plate", projects.MediaKindImage, "image/png", []byte("plate bytes"))
	activeSHA := putMedia("active", projects.MediaKindImage, "image/png", []byte("active bytes"))
	workerSHA := digestBytes([]byte("worker"))
	manifestBytes := []byte("{\"original\":true}\n")
	manifestSHA := digestBytes(manifestBytes)
	receiptBytes, clockBytes, profileBytes := []byte("receipt"), []byte("clock"), []byte("profile")
	pinned := map[string]struct {
		role string
		data []byte
	}{
		"manifest": {"inspection_manifest", manifestBytes},
		"receipt":  {"inspection_receipt", receiptBytes},
		"clock":    {"inspection_clock", clockBytes},
		"profile":  {"renderer_tool_profile", profileBytes},
	}
	bindings := []jobs.ArtifactBinding{
		routeBinding(route),
		{Role: "shot_snapshot", ArtifactID: "shot:shot:revision:1", RevisionID: "revision:1", SHA256: digestBytes([]byte("recipe"))},
		{Role: "worker_executable", ArtifactID: workerBlobPrefix + workerSHA + "/device-studio-api", RevisionID: "sha256:" + workerSHA, SHA256: workerSHA},
		{Role: "source_media", ArtifactID: "video-source", SHA256: sourceSHA},
		{Role: "video_plate", ArtifactID: "plate", SHA256: plateSHA},
		{Role: "video_active_mask", ArtifactID: "active", SHA256: activeSHA},
	}
	var inspectionManifest, inspectionReceipt, inspectionClock, profileKey string
	for key, value := range pinned {
		artifactID := "pinned/" + key
		digest := digestBytes(value.data)
		if err := c.putVerified(ctx, route, artifactID, value.data, "application/octet-stream", digest); err != nil {
			t.Fatal(err)
		}
		bindings = append(bindings, jobs.ArtifactBinding{Role: value.role, ArtifactID: artifactID, RevisionID: "sha256:" + digest, SHA256: digest})
		switch value.role {
		case "inspection_manifest":
			inspectionManifest = artifactID
		case "inspection_receipt":
			inspectionReceipt = artifactID
		case "inspection_clock":
			inspectionClock = artifactID
		case "renderer_tool_profile":
			profileKey = artifactID
		}
	}
	_ = inspectionManifest
	_ = inspectionReceipt
	_ = inspectionClock
	_ = profileKey
	closure := videoClosure{
		Version: 1, ProjectID: "project", ShotID: "shot", ShotRevision: 1, RecipeSHA256: digestBytes([]byte("recipe")),
		WorkerKey: workerBlobPrefix + workerSHA + "/device-studio-api", WorkerSHA256: workerSHA,
		ToolProfileKey: profileKey, ToolProfileSHA256: digestBytes(profileBytes),
		InspectionManifest: inspectionManifest, InspectionSHA256: manifestSHA,
		InspectionReceipt: inspectionReceipt, ReceiptSHA256: digestBytes(receiptBytes),
		ClockKey: inspectionClock, ClockSHA256: digestBytes(clockBytes), SourceMediaID: "video-source", PlateMediaID: "plate", ActiveMaskMediaID: "active",
		Manifest: validPreparedVideoManifest(t, sourceSHA, int64(len(media.data["video-source"])), manifestSHA, int64(len(manifestBytes)), digestBytes(receiptBytes), int64(len(receiptBytes)), digestBytes(clockBytes), int64(len(clockBytes)), plateSHA, int64(len(media.data["plate"])), activeSHA, int64(len(media.data["active"]))),
	}
	closureBytes, err := json.Marshal(closure)
	if err != nil {
		t.Fatal(err)
	}
	closureBytes = append(closureBytes, '\n')
	closureSHA := digestBytes(closureBytes)
	closureKey := closureBlobPrefix + closureSHA + ".json"
	if err := c.putVerified(ctx, route, closureKey, closureBytes, "application/json", closureSHA); err != nil {
		t.Fatal(err)
	}
	job := jobs.Job{ID: "video-job", Input: jobs.Input{ProjectID: "project", SubjectKind: jobs.SubjectVideoRender, SubjectID: "shot", ShotRevision: 1, Bindings: append(bindings, jobs.ArtifactBinding{Role: "video_closure", ArtifactID: closureKey, SHA256: closureSHA})}}
	job.InputFingerprint = digestBytes([]byte("input"))
	job.Input.Bindings = append(job.Input.Bindings, jobs.ArtifactBinding{Role: "shot_snapshot", ArtifactID: "shot:shot:revision:1", SHA256: closure.RecipeSHA256})
	job.Input.Bindings = append(job.Input.Bindings, jobs.ArtifactBinding{Role: "worker_executable", ArtifactID: closure.WorkerKey, SHA256: workerSHA})
	// Replace duplicates with the canonical role binding lookup values.
	job.Input.Bindings = dedupeBindings(job.Input.Bindings)
	attempt := jobs.Attempt{ID: "attempt-1", Number: 1}
	prepared, err := (videoSubjectAdapter{}).prepare(ctx, c, route, job, attempt)
	if err != nil {
		t.Fatalf("prepare video execution plan: %v", err)
	}
	workspace, err := c.createAttemptWorkspace(ctx, route, job, attempt)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range prepared.Files {
		if err := c.stagePreparedFile(ctx, route, workspace, job, file, maxVideoInputBytes); err != nil {
			t.Fatalf("stage prepared video file %q through executor: %v", file.Name, err)
		}
	}
	staged, err := os.ReadFile(filepath.Join(workspace, rendering.AcceptedInspectionManifestName))
	if err != nil || !bytes.Equal(staged, manifestBytes) || digestBytes(staged) != manifestSHA {
		t.Fatalf("accepted source manifest was not staged byte-exactly: sha=%s err=%v", digestBytes(staged), err)
	}
	if _, err := os.Stat(filepath.Join(workspace, rendering.InspectionManifestName)); !os.IsNotExist(err) {
		t.Fatalf("executor-reserved inspection manifest unexpectedly staged: err=%v", err)
	}
	var stagedManifest rendering.VideoManifest
	if err := json.Unmarshal(prepared.ManifestBytes, &stagedManifest); err != nil {
		t.Fatal(err)
	}
	if stagedManifest.Inspection.Manifest.Name != rendering.AcceptedInspectionManifestName || stagedManifest.Inspection.Manifest.SHA256 != manifestSHA {
		t.Fatalf("video closure did not pin the accepted-manifest alias with original digest: %+v", stagedManifest.Inspection.Manifest)
	}
	if stagedManifest.VideoProfile != rendering.VideoProfileID || stagedManifest.CompositorProfile != rendering.VideoCompositorID {
		t.Fatalf("prepared video manifest does not pin worker profiles: video=%q compositor=%q", stagedManifest.VideoProfile, stagedManifest.CompositorProfile)
	}
	stagedManifest.CompositorProfile = ""
	if err := rendering.ValidateVideoManifest(stagedManifest); err == nil || !strings.Contains(err.Error(), "video renderer profile is unsupported") {
		t.Fatalf("worker contract accepted an omitted compositor profile: %v", err)
	}
	closure.Manifest.CompositorProfile = ""
	invalidClosure, err := json.Marshal(closure)
	if err != nil {
		t.Fatal(err)
	}
	invalidClosure = append(invalidClosure, '\n')
	invalidSHA := digestBytes(invalidClosure)
	invalidKey := closureBlobPrefix + invalidSHA + ".json"
	if err := c.putVerified(ctx, route, invalidKey, invalidClosure, "application/json", invalidSHA); err != nil {
		t.Fatal(err)
	}
	for i := range job.Input.Bindings {
		if job.Input.Bindings[i].Role == "video_closure" {
			job.Input.Bindings[i] = jobs.ArtifactBinding{Role: "video_closure", ArtifactID: invalidKey, RevisionID: "sha256:" + invalidSHA, SHA256: invalidSHA}
		}
	}
	_, err = (videoSubjectAdapter{}).prepare(ctx, c, route, job, attempt)
	if !errors.Is(err, errDeterministicPreparation) {
		t.Fatalf("worker manifest validation failure was not classified as deterministic: %v", err)
	}
}

type lookupInspectionJobService struct {
	jobs.Service
	job jobs.Job
}

func (s lookupInspectionJobService) Get(_ context.Context, id string) (jobs.Job, error) {
	if id != s.job.ID {
		return jobs.Job{}, jobs.ErrNotFound
	}
	return s.job, nil
}

func TestMakeVideoClosureProducesWorkerValidCompositorProfile(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	media := c.media.(*integrationMedia)
	sourceBytes := []byte("accepted source fixture")
	sourceSHA := digestBytes(sourceBytes)
	media.data["screen"] = sourceBytes
	profileDigest := strings.Repeat("c", 64)
	profile := &cliv1.RendererToolProfile{ProfileId: "btbn-ffmpeg-video-n9.0.2-28-g669f5ab083-linux64-gpl", ArtifactIdentity: profileDigest}
	profileBytes, err := protojson.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	profileSHA := digestBytes(profileBytes)
	profileKey := inputBlobPrefix + profileSHA + "/inspection-profile.json"
	if err := c.putVerified(ctx, route, profileKey, profileBytes, "application/json", profileSHA); err != nil {
		t.Fatal(err)
	}
	workerBytes, _, workerSHA, err := c.readCurrentExecutable()
	if err != nil {
		t.Fatal(err)
	}
	workerKey := workerBlobPrefix + workerSHA + "/device-studio-api"
	if err := c.putVerified(ctx, route, workerKey, workerBytes, "application/octet-stream", workerSHA); err != nil {
		t.Fatal(err)
	}
	const inspectionJobID, inspectionAttemptID = "inspection-job", "inspection-attempt"
	inputFingerprint := digestBytes([]byte("inspection-input"))
	input := jobs.Input{
		ProjectID: "project", ToolProfileDigest: profileDigest, SubjectKind: jobs.SubjectSourceInspection, SubjectID: "screen",
		Bindings: []jobs.ArtifactBinding{
			routeBinding(route),
			{Role: "source_media", ArtifactID: "source-media:screen", RevisionID: "sha256:" + sourceSHA, SHA256: sourceSHA},
			{Role: "source_size", ArtifactID: "source-media-size:screen", RevisionID: "size:" + strconv.Itoa(len(sourceBytes)), SHA256: sourceSHA},
			{Role: "inspection_profile", ArtifactID: profileKey, RevisionID: "sha256:" + profileSHA, SHA256: profileSHA},
			{Role: "worker_executable", ArtifactID: workerKey, RevisionID: "sha256:" + workerSHA, SHA256: workerSHA},
		},
	}
	inspectionJob := jobs.Job{
		ID: inspectionJobID, Input: input, InputFingerprint: inputFingerprint, State: jobs.Succeeded,
		Attempts: []jobs.Attempt{{ID: inspectionAttemptID, State: jobs.AttemptSucceeded, Completion: &jobs.CompletionEvidence{AttemptID: inspectionAttemptID}}},
	}
	manifest, err := inspectionManifestFromInput(inspectionJob, inspectionJob.Attempts[0], profile)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	clockBytes := make([]byte, 12+3*12)
	copy(clockBytes[:8], "DSCLK001")
	binary.BigEndian.PutUint32(clockBytes[8:12], 3)
	for ordinal, pts := range []uint32{0, 30, 60} {
		offset := 12 + ordinal*12
		binary.BigEndian.PutUint32(clockBytes[offset:offset+4], uint32(ordinal))
		binary.BigEndian.PutUint64(clockBytes[offset+4:offset+12], uint64(pts))
	}
	clockSHA := digestBytes(clockBytes)
	startPTS, endPTS, streamIndex := int64(0), int64(90), uint32(0)
	inspectionReceipt := rendering.InspectionReceipt{
		Version: 1, JobID: inspectionJob.ID, AttemptID: inspectionAttemptID, InputFingerprint: inputFingerprint, WorkerSHA256: workerSHA,
		ManifestSHA256: digestBytes(manifestBytes), SourceSHA256: sourceSHA, SourceSizeBytes: int64(len(sourceBytes)),
		ProfileID: profile.GetProfileId(), ProfileDigest: profileDigest, ProtocolVersion: 1, Classification: rendering.InspectionAccepted,
		Container: "mov,mp4,m4a", StreamCount: 1, VideoStreamIndex: &streamIndex, Codec: "h264", CodedWidth: 64, CodedHeight: 64,
		DisplayWidth: 64, DisplayHeight: 64, SampleAspectNum: 1, SampleAspectDen: 1, ColorMatrix: "bt709", ColorRange: "tv",
		ColorPolicy: "sdr-unmarked-default709-v1", TimeBaseNum: 1, TimeBaseDen: 30, StreamStartPTS: &startPTS, SourceEndPTS: &endPTS,
		LastFrameDurationTicks: 30, DurationNumerator: 3, DurationDenominator: 1, FrameCount: 3,
		EOFPolicy: "last_frame_duration", ClockArtifactRef: rendering.InspectionClockName, ClockArtifactSizeBytes: int64(len(clockBytes)), ClockSHA256: clockSHA,
	}
	receiptBytes, err := json.Marshal(inspectionReceipt)
	if err != nil {
		t.Fatal(err)
	}
	receiptSHA := digestBytes(receiptBytes)
	media.receiptBytes, media.clockBytes = receiptBytes, clockBytes
	media.values["screen"] = projects.SourceMedia{
		ID: "screen", ProjectID: "project", SHA256: sourceSHA, SizeBytes: int64(len(sourceBytes)), MediaType: "video/mp4", Width: 64, Height: 64,
		Kind: projects.MediaKindVideo, Readiness: projects.ReadinessReady, InspectionJobID: inspectionJobID, InspectionAttemptID: inspectionAttemptID,
		InspectionReceipt: &projects.InspectionReceipt{ID: receiptSHA, SHA256: receiptSHA, ClockSHA256: clockSHA, JobID: inspectionJobID, AttemptID: inspectionAttemptID, SourceSHA256: sourceSHA, SourceSizeBytes: int64(len(sourceBytes)), Classification: "accepted"},
	}
	media.values["plate"] = projects.SourceMedia{ID: "plate", ProjectID: "project", SHA256: media.values["plate"].SHA256, SizeBytes: media.values["plate"].SizeBytes, MediaType: "image/png", Kind: projects.MediaKindImage, Width: 1920, Height: 1080}
	c.jobs = lookupInspectionJobService{Service: c.jobs, job: inspectionJob}

	resolved := integrationResolved(media)
	resolved.Revision.PresentationVersion = presentations.SemanticVersionV2
	resolved.Revision.Plate.WidthPixels, resolved.Revision.Plate.HeightPixels = 1920, 1080
	scene := resolved.Revision.Snapshot.GetScene()
	scene.DisplayWidthPixels, scene.DisplayHeightPixels = 430, 880
	scene.PlateQuad = &shotsv1.PlateQuad{
		TopLeft: &shotsv1.ShotPoint{X: 0, Y: 0}, TopRight: &shotsv1.ShotPoint{X: 1920, Y: 0},
		BottomRight: &shotsv1.ShotPoint{X: 1920, Y: 1080}, BottomLeft: &shotsv1.ShotPoint{X: 0, Y: 1080},
	}
	resolved.Revision.Snapshot.Presentation = &shotsv1.ScreenPresentation{
		SourceMediaId: "screen", CaptureArea: shotsv1.CaptureArea_CAPTURE_AREA_APP_ONLY,
		Kind: shotsv1.PresentationKind_PRESENTATION_KIND_BROWSER_DESKTOP, FitMode: shotsv1.ScreenFitMode_SCREEN_FIT_MODE_CONTAIN,
		Matte:                &shotsv1.PresentationRGB{Red: 12, Green: 16, Blue: 20},
		BrowserDesktopChrome: &shotsv1.BrowserDesktopChrome{DesktopColor: &shotsv1.PresentationRGB{Red: 20, Green: 24, Blue: 30}},
	}
	resolved.Revision.Snapshot.VideoTiming = &shotsv1.VideoTiming{
		SourceMediaId: "screen", SourceSha256: sourceSHA, SourceSizeBytes: int64(len(sourceBytes)),
		InspectionJobId: inspectionJobID, InspectionAttemptId: inspectionAttemptID, InspectionReceiptId: receiptSHA, InspectionReceiptSha256: receiptSHA,
		ClockArtifactId: "clock-" + clockSHA, ClockSha256: clockSHA, ClockSizeBytes: int64(len(clockBytes)), SelectedVideoStream: 0,
		TimeBaseNum: 1, TimeBaseDen: 30, StreamStartPts: &startPTS, SourceEndPts: endPTS, LastFrameDurationTicks: 30,
		TrimStartPts: 0, TrimEndPts: endPTS, PlaybackRateNum: 1, PlaybackRateDen: 1, OutputRateNum: 24, OutputRateDen: 1,
		EofPolicy:                 shotsv1.VideoEOFPolicy_VIDEO_EOF_POLICY_REJECT,
		VideoCompositionProfileId: rendering.VideoProfileID, VideoCompositionProfileVersion: 1,
	}
	cfg, configErr := presentations.ConfigFromSavedVersion(image.Pt(430, 880), resolved.Revision.Snapshot.Presentation, presentations.SemanticVersionV2)
	if configErr != nil {
		t.Fatalf("test fixture saved presentation is invalid: %v", configErr)
	}
	if _, layoutErr := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, 64, 64), cfg); layoutErr != nil {
		t.Fatalf("test fixture presentation cannot build inspection-raster layout: %v", layoutErr)
	}
	request := SubmitRequest{ProjectID: "project", ShotID: "shot", Revision: 1, VideoMode: rendering.VideoModeFinal}
	closure, err := c.makeVideoClosure(ctx, route, request, resolved, workerKey, workerSHA)
	if err != nil {
		t.Fatalf("build closure from accepted source owner facts: %v", err)
	}
	closure.Manifest.JobID, closure.Manifest.AttemptID = "video-job", "video-attempt"
	closure.Manifest.InputFingerprint = digestBytes([]byte("video-input"))
	closure.Manifest.WorkerSHA256 = workerSHA
	if closure.Manifest.CompositorProfile != rendering.VideoCompositorID {
		t.Fatalf("makeVideoClosure omitted worker compositor profile: got %q want %q", closure.Manifest.CompositorProfile, rendering.VideoCompositorID)
	}
	if err := rendering.ValidateVideoManifest(closure.Manifest); err != nil {
		t.Fatalf("makeVideoClosure produced a manifest rejected by worker contract: %v", err)
	}

	cameraResolved := resolved
	cameraResolved.Revision.Snapshot = proto.Clone(resolved.Revision.Snapshot).(*shotsv1.ShotSnapshot)
	cameraResolved.Revision.Snapshot.MotionMode = shotsv1.MotionMode_MOTION_MODE_CROP_PUSH
	cameraResolved.Revision.Snapshot.CameraMotion = &shotsv1.PhotographicCameraMotion{
		ProfileId: shots.CameraMotionProfileID, ProfileVersion: shots.CameraMotionProfileVersion,
		DurationOutputFrames: 24, MatteRgb: &shotsv1.PresentationRGB{Red: 4, Green: 5, Blue: 6},
	}
	cameraResolved.Revision.Snapshot.VideoTiming.VideoCompositionProfileId = shots.VideoProfileScreenEntry
	cameraClosure, err := c.makeVideoClosure(ctx, route, request, cameraResolved, workerKey, workerSHA)
	if err != nil {
		t.Fatalf("build screen-entry closure: %v", err)
	}
	cameraClosure.Manifest.JobID, cameraClosure.Manifest.AttemptID = "camera-job", "camera-attempt"
	cameraClosure.Manifest.InputFingerprint = digestBytes([]byte("camera input"))
	cameraClosure.Manifest.WorkerSHA256 = workerSHA
	camera := cameraClosure.Manifest.Camera
	if cameraClosure.Manifest.VideoProfile != rendering.VideoScreenEntryProfileID || camera == nil || camera.ProfileID != shots.CameraMotionProfileID || camera.ProfileVersion != 1 || camera.DurationOutputFrames != 24 || camera.MatteRGB == nil || camera.MatteRGB.R != 4 || camera.MatteRGB.G != 5 || camera.MatteRGB.B != 6 {
		t.Fatalf("camera closure mapping mismatch: profile=%q camera=%+v", cameraClosure.Manifest.VideoProfile, cameraClosure.Manifest.Camera)
	}
	if cameraClosure.Manifest.Scene.PlateWidth != 1920 || cameraClosure.Manifest.Scene.PlateHeight != 1080 {
		t.Fatalf("camera closure omitted pinned plate raster: %dx%d", cameraClosure.Manifest.Scene.PlateWidth, cameraClosure.Manifest.Scene.PlateHeight)
	}
	if err := rendering.ValidateVideoManifest(cameraClosure.Manifest); err != nil {
		t.Fatalf("camera closure is rejected by worker contract: %v", err)
	}
}

func validPreparedVideoManifest(t *testing.T, sourceSHA string, sourceSize int64, manifestSHA string, manifestSize int64, receiptSHA string, receiptSize int64, clockSHA string, clockSize int64, plateSHA string, plateSize int64, activeSHA string, activeSize int64) rendering.VideoManifest {
	t.Helper()
	digest := strings.Repeat("a", 64)
	fill := presentations.RGB{R: 24, G: 28, B: 38}
	presentation := &presentations.Config{Target: image.Pt(430, 880), Browser: &presentations.BrowserConfig{DesktopColor: &fill}}
	layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, 1440, 800), *presentation)
	if err != nil {
		t.Fatalf("build prepared test layout: %v", err)
	}
	return rendering.VideoManifest{
		Version: 1, ProtocolVersion: 1, JobID: "video-job", AttemptID: "attempt-1", InputFingerprint: digest, WorkerSHA256: digest,
		Mode: rendering.VideoModeFinal, VideoProfile: rendering.VideoProfileID, CompositorProfile: rendering.VideoCompositorID,
		Inspection: rendering.VideoInspection{
			Source:    rendering.VideoBinding{Name: "source.media", SHA256: sourceSHA, SizeBytes: sourceSize},
			Manifest:  rendering.VideoBinding{Name: rendering.AcceptedInspectionManifestName, SHA256: manifestSHA, SizeBytes: manifestSize},
			Receipt:   rendering.VideoBinding{Name: rendering.InspectionReceiptName, SHA256: receiptSHA, SizeBytes: receiptSize},
			Clock:     rendering.VideoBinding{Name: rendering.InspectionClockName, SHA256: clockSHA, SizeBytes: clockSize},
			ProfileID: "btbn-ffmpeg-video-n9.0.2-28-g669f5ab083-linux64-gpl", ProfileDigest: digest, ProtocolVersion: 1,
			TimeBase: composition.Rational{Num: 1, Den: 24}, SourceEndPTS: 240, LastFrameDurationTicks: 10,
			EOFPolicy: "last_frame_duration", FrameCount: 230, Width: 1440, Height: 800, ColorPolicy: "sdr-unmarked-default709-v1",
		},
		TrimStartPTS: 0, TrimEndPTS: 240, PlaybackRate: composition.Rational{Num: 1, Den: 1}, OutputRate: composition.Rational{Num: 24, Den: 1},
		Width: 1920, Height: 1080,
		Scene: rendering.VideoScene{
			Plate:        rendering.VideoBinding{Name: "plate.image", SHA256: plateSHA, SizeBytes: plateSize},
			ActiveMask:   rendering.VideoBinding{Name: "active-mask.png", SHA256: activeSHA, SizeBytes: activeSize},
			SurfaceWidth: 430, SurfaceHeight: 880, Quad: [4]rendering.Corner{{X: 10, Y: 10}, {X: 20, Y: 10}, {X: 20, Y: 30}, {X: 10, Y: 30}}, Samples: 1,
		},
		Presentation: presentation, PresentationLayout: &layout,
		ShotRevision: 1, BaseShotRevision: 1, BaseRecipeVersion: "device-studio-presentation-v2", BaseRecipeSHA256: digest, RecipeSHA256: digest,
	}
}

func dedupeBindings(input []jobs.ArtifactBinding) []jobs.ArtifactBinding {
	byRole := make(map[string]jobs.ArtifactBinding, len(input))
	for _, binding := range input {
		byRole[binding.Role] = binding
	}
	out := make([]jobs.ArtifactBinding, 0, len(byRole))
	for _, binding := range byRole {
		out = append(out, binding)
	}
	return out
}
