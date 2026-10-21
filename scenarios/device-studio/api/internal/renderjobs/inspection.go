package renderjobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"device-studio/internal/jobs"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	platform "github.com/vrooli/platform-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	inspectionRendererVersion = "ffmpeg-video-inspection-v1"
	inspectionOutputProfile   = "source-inspection-receipt-v1"
)

type sourceInspectionAdapter struct{}

func (sourceInspectionAdapter) prepare(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt) (preparedOperation, error) {
	media, err := c.repairInspectionAttempt(ctx, job, attempt)
	if err != nil {
		return preparedOperation{}, err
	}
	profile, profileBytes, err := c.inspectionProfileForJob(ctx, route, job)
	if err != nil {
		return preparedOperation{}, err
	}
	manifest, err := inspectionManifestFromInput(job, attempt, profile)
	if err != nil {
		return preparedOperation{}, err
	}
	if media.SHA256 != manifest.Source.SHA256 || media.SizeBytes != manifest.Source.SizeBytes || media.SizeBytes <= 0 {
		return preparedOperation{}, errors.New("renderjobs: inspection source no longer matches immutable job input")
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return preparedOperation{}, err
	}
	fileOpen := func(openCtx context.Context) (io.ReadCloser, error) {
		opened, reader, openErr := c.media.Open(openCtx, job.Input.ProjectID, media.ID)
		if openErr != nil {
			return nil, openErr
		}
		if opened.SHA256 != media.SHA256 || opened.SizeBytes != media.SizeBytes || opened.Kind != projects.MediaKindVideo {
			_ = reader.Close()
			return nil, errors.New("renderjobs: source owner returned different inspection bytes")
		}
		return reader, nil
	}
	return preparedOperation{Mode: operationSourceInspect, ManifestBytes: append(manifestBytes, '\n'), MaxInputBytes: maxInspectionInputBytes, Files: []stagedFile{
		{Name: rendering.InspectionSourceName, SHA256: media.SHA256, Size: media.SizeBytes, Mode: manifestMode, Open: fileOpen, MaxBytes: maxInspectionInputBytes},
		{Name: rendering.InspectionProfileName, SHA256: bindingDigest(job.Input, "inspection_profile"), Size: int64(len(profileBytes)), Data: profileBytes, Mode: manifestMode, MaxBytes: maxInspectionSidecarBytes},
	}}, nil
}

func (sourceInspectionAdapter) validateCompletion(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt, intent persistedIntent, nativeKey, nativeSHA string) (jobs.CompletionEvidence, error) {
	if err := validateNativeEvidence(ctx, c, route, job, attempt, intent, operationSourceInspect, nativeKey, nativeSHA); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	workspace, err := attemptWorkspace(route.Root, job.ID, attempt.ID)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	manifestBytes, err := readWorkspaceFile(workspace, rendering.InspectionManifestName, maxManifestBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	profile, _, err := c.inspectionProfileForJob(ctx, route, job)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	media, err := c.media.Get(ctx, job.Input.ProjectID, job.Input.SubjectID)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	manifest, err := inspectionManifestFromInput(job, attempt, profile)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if media.Kind != projects.MediaKindVideo || media.SHA256 != manifest.Source.SHA256 || media.SizeBytes != manifest.Source.SizeBytes {
		return jobs.CompletionEvidence{}, projects.ErrInspectionConflict
	}
	expectedManifest, err := json.Marshal(manifest)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	expectedManifest = append(expectedManifest, '\n')
	if intent.JobID != job.ID || intent.AttemptID != attempt.ID || intent.InputFingerprint != job.InputFingerprint || intent.WorkerSHA256 != bindingDigest(job.Input, "worker_executable") || intent.ManifestSHA256 != digestBytes(expectedManifest) || digestBytes(manifestBytes) != intent.ManifestSHA256 || !bytes.Equal(manifestBytes, expectedManifest) {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: inspection manifest differs from persisted immutable intent")
	}
	receiptBytes, err := readWorkspaceFile(workspace, rendering.InspectionReceiptName, maxReceiptBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	clockBytes, err := readWorkspaceFile(workspace, rendering.InspectionClockName, 1<<20)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return jobs.CompletionEvidence{}, err
	}
	receipt, err := rendering.VerifyInspectionArtifacts(manifestBytes, receiptBytes, clockBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("verify inspection result: %w", err)
	}
	if receipt.WorkerSHA256 != bindingDigest(job.Input, "worker_executable") || receipt.ProfileDigest != job.Input.ToolProfileDigest || receipt.SourceSHA256 != bindingDigest(job.Input, "source_media") || receipt.SourceSizeBytes != bindingSize(job.Input, "source_media") || receipt.JobID != job.ID || receipt.AttemptID != attempt.ID || receipt.InputFingerprint != job.InputFingerprint {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: inspection receipt differs from immutable worker/profile input")
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	receiptSHA, clockSHA := digestBytes(receiptBytes), ""
	if len(clockBytes) > 0 {
		clockSHA = digestBytes(clockBytes)
	}
	receiptKey, clockRef, err := c.media.StoreInspectionArtifacts(ctx, job.Input.ProjectID, job.Input.SubjectID, job.ID, attempt.ID, receiptSHA, clockSHA, receiptBytes, clockBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("store source inspection result: %w", err)
	}
	outputs := []jobs.OutputRef{{Kind: "inspection_receipt", ArtifactID: receiptKey, SHA256: receiptSHA, MIMEType: "application/json", SizeBytes: int64(len(receiptBytes))}}
	if len(clockBytes) > 0 {
		clockKey := projects.InspectionClockObjectKey(job.Input.ProjectID, job.Input.SubjectID, job.ID, clockRef)
		outputs = append(outputs, jobs.OutputRef{Kind: "inspection_clock", ArtifactID: clockKey, SHA256: clockSHA, MIMEType: "application/octet-stream", SizeBytes: int64(len(clockBytes))})
	}
	return jobs.CompletionEvidence{AttemptID: attempt.ID, ExecutionClaimID: attempt.Execution.ID, InputFingerprint: job.InputFingerprint, Outputs: outputs, ReceiptArtifactID: receiptKey, ReceiptSHA256: receiptSHA, CompletedAt: c.now().UTC()}, nil
}

func (sourceInspectionAdapter) finalizeTerminal(ctx context.Context, c *Coordinator, pending jobs.SubjectFinalization) error {
	return sourceInspectionAdapter{}.finalize(ctx, c, pending)
}

func (a sourceInspectionAdapter) finalize(ctx context.Context, c *Coordinator, pending jobs.SubjectFinalization) error {
	job, err := c.jobs.Get(ctx, pending.JobID)
	if err != nil {
		return err
	}
	attempt, ok := currentAttempt(job)
	if !ok || attempt.ID != pending.AttemptID || job.Input.SubjectKind != jobs.SubjectSourceInspection || pending.Kind != jobs.SubjectSourceInspection {
		return jobs.ErrConcurrentTransition
	}
	if _, err := c.repairInspectionAttempt(ctx, job, attempt); err != nil {
		return err
	}
	media, err := c.media.Get(ctx, job.Input.ProjectID, job.Input.SubjectID)
	if err != nil {
		return err
	}
	if media.SHA256 != bindingDigest(job.Input, "source_media") || media.InspectionJobID != job.ID || media.InspectionAttemptID != attempt.ID {
		return projects.ErrInspectionConflict
	}
	projection := projects.InspectionProjection{ProjectID: media.ProjectID, MediaID: media.ID, SourceSHA256: media.SHA256, JobID: job.ID, AttemptID: attempt.ID}
	if pending.State == jobs.Succeeded {
		if attempt.Completion == nil {
			return errors.New("renderjobs: successful inspection lacks completion receipt")
		}
		var receiptOutput, clockOutput *jobs.OutputRef
		for i := range attempt.Completion.Outputs {
			switch attempt.Completion.Outputs[i].Kind {
			case "inspection_receipt":
				receiptOutput = &attempt.Completion.Outputs[i]
			case "inspection_clock":
				clockOutput = &attempt.Completion.Outputs[i]
			}
		}
		if receiptOutput == nil {
			return errors.New("renderjobs: inspection completion lacks receipt output")
		}
		clockSHA := ""
		if clockOutput != nil {
			clockSHA = clockOutput.SHA256
		}
		receiptBytes, clockBytes, err := c.media.OpenInspectionArtifacts(ctx, media.ProjectID, media.ID, job.ID, attempt.ID, receiptOutput.SHA256, clockSHA)
		if err != nil {
			return err
		}
		route, err := c.route(ctx)
		if err != nil {
			return err
		}
		profile, _, err := c.inspectionProfileForJob(ctx, route, job)
		if err != nil {
			return err
		}
		manifest, err := inspectionManifestFromInput(job, attempt, profile)
		if err != nil {
			return err
		}
		if media.SHA256 != manifest.Source.SHA256 || media.SizeBytes != manifest.Source.SizeBytes {
			return projects.ErrInspectionConflict
		}
		manifestBytes, err := json.Marshal(manifest)
		if err != nil {
			return err
		}
		manifestBytes = append(manifestBytes, '\n')
		receipt, err := rendering.VerifyInspectionArtifacts(manifestBytes, receiptBytes, clockBytes)
		if err != nil {
			return fmt.Errorf("reverify stored inspection result: %w", err)
		}
		projection.Receipt = projectReceipt(receipt, receiptOutput.SHA256)
		projection.ReceiptStorageKey = projects.InspectionReceiptObjectKey(media.ProjectID, media.ID, job.ID, projection.Receipt.ID)
		if receipt.Classification == rendering.InspectionRejected {
			projection.Readiness, projection.IssueCode, projection.IssueMessage = projects.ReadinessRejected, receipt.IssueCode, receipt.IssueMessage
		} else {
			projection.Readiness = projects.ReadinessReady
		}
	} else if pending.State == jobs.Failed || pending.State == jobs.Cancelled {
		projection.Readiness = projects.ReadinessRejected
		projection.IssueCode = "inspection_execution_" + string(pending.State)
		if pending.State == jobs.Cancelled {
			projection.IssueMessage = "Inspection was cancelled after its owned operation was confirmed stopped. Retry is available."
		} else {
			projection.IssueMessage = "Inspection failed after its owned operation reached a known terminal state. Retry is available."
		}
	} else {
		// Unknown native outcomes are nonterminal and must remain pending for
		// reconciliation; they cannot be projected as a known rejection.
		return errors.New("renderjobs: nonterminal inspection outcome cannot be finalized as a known result")
	}
	return c.media.ProjectInspection(ctx, projection)
}

func projectReceipt(r rendering.InspectionReceipt, receiptSHA string) *projects.InspectionReceipt {
	var audio *projects.AudioInspectionFacts
	if r.Audio != nil {
		audio = &projects.AudioInspectionFacts{Version: r.Audio.Version, Streams: make([]projects.AudioStreamFact, 0, len(r.Audio.Streams))}
		for _, stream := range r.Audio.Streams {
			audio.Streams = append(audio.Streams, projects.AudioStreamFact{OriginalStreamIndex: stream.OriginalStreamIndex, Codec: stream.Codec})
		}
	}
	var stream, start, end *int64
	if r.VideoStreamIndex != nil {
		v := int64(*r.VideoStreamIndex)
		stream = &v
	}
	if r.StreamStartPTS != nil {
		v := *r.StreamStartPTS
		start = &v
	}
	if r.SourceEndPTS != nil {
		v := *r.SourceEndPTS
		end = &v
	}
	return &projects.InspectionReceipt{Audio: audio, Version: uint32(r.Version), ID: receiptSHA, SHA256: receiptSHA, SourceSHA256: r.SourceSHA256, SourceSizeBytes: r.SourceSizeBytes, JobID: r.JobID, AttemptID: r.AttemptID, ToolProfileID: r.ProfileID, ToolProfileDigest: r.ProfileDigest, InspectionProtocolVersion: r.ProtocolVersion, Classification: string(r.Classification), IssueCode: r.IssueCode, IssueMessage: r.IssueMessage, Container: r.Container, StreamCount: r.StreamCount, VideoStreamIndex: stream, Codec: r.Codec, CodedWidth: r.CodedWidth, CodedHeight: r.CodedHeight, VideoWidth: r.DisplayWidth, VideoHeight: r.DisplayHeight, SampleAspectNum: r.SampleAspectNum, SampleAspectDen: r.SampleAspectDen, RotationDegrees: r.RotationDegrees, ColorPolicy: r.ColorPolicy, ColorPrimaries: r.ColorPrimaries, ColorTransfer: r.ColorTransfer, ColorMatrix: r.ColorMatrix, ColorRange: r.ColorRange, TimeBaseNumerator: r.TimeBaseNum, TimeBaseDenominator: r.TimeBaseDen, StreamStartPTS: start, SourceEndPTS: end, LastFrameDurationTicks: r.LastFrameDurationTicks, DurationNumerator: r.DurationNumerator, DurationDenominator: r.DurationDenominator, FrameCount: r.FrameCount, EOFPolicy: r.EOFPolicy, ClockSHA256: r.ClockSHA256, ClockArtifactRef: r.ClockArtifactRef, ClockArtifactSizeBytes: r.ClockArtifactSizeBytes}
}

func (c *Coordinator) inspectionProfileForJob(ctx context.Context, route routePin, job jobs.Job) (*cliv1.RendererToolProfile, []byte, error) {
	b := bindingByRole(job.Input, "inspection_profile")
	if b.ArtifactID == "" || !validDigest(b.SHA256) || b.RevisionID != "sha256:"+b.SHA256 {
		return nil, nil, errors.New("renderjobs: inspection job lacks pinned profile binding")
	}
	data, err := c.readBlob(ctx, b.ArtifactID, 64<<10, b.SHA256)
	if err != nil {
		return nil, nil, err
	}
	var profile cliv1.RendererToolProfile
	if err := protojson.Unmarshal(data, &profile); err != nil {
		return nil, nil, err
	}
	if profile.GetArtifactIdentity() != job.Input.ToolProfileDigest {
		return nil, nil, errors.New("renderjobs: stored inspection profile differs from job input")
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return nil, nil, err
	}
	return &profile, data, nil
}

func inspectionManifestFromInput(job jobs.Job, attempt jobs.Attempt, profile *cliv1.RendererToolProfile) (rendering.InspectionManifest, error) {
	sourceSize := bindingSize(job.Input, "source_media")
	if profile == nil || profile.GetProfileId() == "" || !validDigest(bindingDigest(job.Input, "source_media")) || sourceSize <= 0 {
		return rendering.InspectionManifest{}, errors.New("renderjobs: inspection source/profile closure is incomplete")
	}
	return rendering.InspectionManifest{Version: 1, JobID: job.ID, AttemptID: attempt.ID, InputFingerprint: job.InputFingerprint, WorkerSHA256: bindingDigest(job.Input, "worker_executable"), Source: rendering.InspectionSource{Name: rendering.InspectionSourceName, SHA256: bindingDigest(job.Input, "source_media"), SizeBytes: sourceSize}, ProfileID: profile.GetProfileId(), ProfileDigest: job.Input.ToolProfileDigest, ProtocolVersion: 1, Limits: rendering.DefaultInspectionLimits()}, nil
}

func inspectionSourceSize(in jobs.Input) (int64, error) {
	return bindingSizeStrict(in, "source_media")
}

func bindingSizeStrict(in jobs.Input, role string) (int64, error) {
	for _, b := range in.Bindings {
		if b.Role != "source_size" || !strings.HasPrefix(b.ArtifactID, "source-media-size:") || b.SHA256 != bindingDigest(in, role) {
			continue
		}
		if !strings.HasPrefix(b.RevisionID, "size:") {
			return 0, jobs.ErrIdempotencyConflict
		}
		n, err := strconv.ParseInt(strings.TrimPrefix(b.RevisionID, "size:"), 10, 64)
		if err != nil || n <= 0 {
			return 0, jobs.ErrIdempotencyConflict
		}
		return n, nil
	}
	return 0, jobs.ErrIdempotencyConflict
}

func (c *Coordinator) repairInspectionAttempt(ctx context.Context, job jobs.Job, attempt jobs.Attempt) (projects.SourceMedia, error) {
	media, err := c.media.Get(ctx, job.Input.ProjectID, job.Input.SubjectID)
	if err != nil {
		return projects.SourceMedia{}, err
	}
	sourceSize, sizeErr := inspectionSourceSize(job.Input)
	if sizeErr != nil || media.Kind != projects.MediaKindVideo || media.SHA256 != bindingDigest(job.Input, "source_media") || media.SizeBytes != sourceSize {
		return projects.SourceMedia{}, projects.ErrInspectionConflict
	}
	if media.InspectionJobID == job.ID && media.InspectionAttemptID == attempt.ID {
		return media, nil
	}
	expected := ""
	if attempt.Number > 1 && attempt.RetryFromVersion > 0 && len(job.Attempts) >= 2 {
		previous := job.Attempts[len(job.Attempts)-2]
		if previous.ID != media.InspectionAttemptID {
			return projects.SourceMedia{}, projects.ErrInspectionConflict
		}
		expected = previous.ID
	}
	if media.InspectionJobID != "" && media.InspectionJobID != job.ID {
		return projects.SourceMedia{}, projects.ErrInspectionConflict
	}
	return c.media.BindInspectionAttempt(ctx, media.ProjectID, media.ID, media.SHA256, job.ID, expected, attempt.ID)
}

func validateNativeEvidence(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt, intent persistedIntent, expectedMode operationMode, key, sha string) error {
	workspace, err := attemptWorkspace(route.Root, job.ID, attempt.ID)
	if err != nil {
		return err
	}
	if intent.Mode != expectedMode || intent.JobID != job.ID || intent.AttemptID != attempt.ID || intent.InputFingerprint != job.InputFingerprint || intent.WorkerSHA256 != bindingDigest(job.Input, "worker_executable") || validateOperationSpec(intent, workspace) != nil || !strings.HasPrefix(key, operationEvidenceBlobPrefix+job.ID+"/"+attempt.ID+"/native-") || !validDigest(sha) || attempt.Execution == nil {
		return errors.New("renderjobs: inspection native evidence reference is invalid")
	}
	data, err := c.readBlob(ctx, key, 64<<10, sha)
	if err != nil {
		return err
	}
	var evidence nativeOutcomeEvidence
	if err := decodeJSON(data, &evidence); err != nil {
		return err
	}
	if evidence.Version != 1 || evidence.JobID != job.ID || evidence.AttemptID != attempt.ID || evidence.InputFingerprint != job.InputFingerprint || evidence.ExecutionClaimID != attempt.Execution.ID || evidence.Route != route || evidence.Intent != intent.Intent || evidence.Ref.Intent != intent.Intent || !evidence.Observation.ExitKnown || evidence.Observation.Populated || evidence.Observation.RootAlive || evidence.Observation.State != platform.OperationStoppedOutcomeKnown {
		return errors.New("renderjobs: inspection native outcome does not match immutable execution claim")
	}
	return c.requireRoute(ctx, route)
}

func bindingSize(in jobs.Input, role string) int64 {
	n, _ := bindingSizeStrict(in, role)
	return n
}

// EnsureSourceInspection submits or recovers one immutable job; it never
// retries a terminal or unresolved job.
func (c *Coordinator) EnsureSourceInspection(ctx context.Context, projectID, mediaID string) (jobs.Job, bool, error) {
	ctx, cancel := boundedDetached(ctx, stageTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, false, err
	}
	media, err := c.media.Get(ctx, projectID, mediaID)
	if err != nil {
		return jobs.Job{}, false, err
	}
	if media.Kind != projects.MediaKindVideo {
		return jobs.Job{}, false, ErrNotEligible
	}
	key := inspectionIdempotencyKey(projectID, media.ID, media.SHA256)
	if media.InspectionJobID != "" {
		job, getErr := c.jobs.Get(ctx, media.InspectionJobID)
		if getErr != nil {
			return jobs.Job{}, false, getErr
		}
		if err := verifyInspectionJob(job, projectID, media, route); err != nil {
			return jobs.Job{}, false, err
		}
		attempt, ok := currentAttempt(job)
		if !ok {
			return jobs.Job{}, false, ErrUnknownOutcome
		}
		if _, err := c.repairInspectionAttempt(ctx, job, attempt); err != nil {
			return jobs.Job{}, false, err
		}
		return job, false, nil
	}
	if media.Readiness != projects.ReadinessPending {
		return jobs.Job{}, false, ErrNotEligible
	}
	// Recover a durable submit whose first media binding was interrupted. This
	// lookup uses source identity only, so a new profile/executable cannot make
	// Ensure silently submit a second inspection for the same immutable media.
	prior, lookupErr := c.jobs.FindByIdempotencyKey(ctx, key)
	if lookupErr == nil {
		if err := verifyInspectionJob(prior, projectID, media, route); err != nil {
			return jobs.Job{}, false, err
		}
		attempt, ok := currentAttempt(prior)
		if !ok {
			return jobs.Job{}, false, ErrUnknownOutcome
		}
		if _, err := c.repairInspectionAttempt(ctx, prior, attempt); err != nil {
			return prior, false, err
		}
		return prior, false, nil
	}
	if !errors.Is(lookupErr, jobs.ErrNotFound) {
		return jobs.Job{}, false, lookupErr
	}
	if c.inspectionProfile == nil {
		return jobs.Job{}, false, ErrNotEligible
	}
	profile, err := c.inspectionProfile(ctx)
	if err != nil {
		return jobs.Job{}, false, fmt.Errorf("resolve qualified inspection profile: %w", err)
	}
	profileBytes, err := protojson.Marshal(profile)
	if err != nil {
		return jobs.Job{}, false, err
	}
	profileID, profileDigest := profile.GetProfileId(), profile.GetArtifactIdentity()
	if profileID == "" || !validDigest(profileDigest) || len(profileBytes) == 0 || len(profileBytes) > 64<<10 {
		return jobs.Job{}, false, errors.New("renderjobs: inspection profile is invalid")
	}
	workerBytes, _, workerSHA, err := c.readCurrentExecutable()
	if err != nil {
		return jobs.Job{}, false, err
	}
	workerKey := workerBlobPrefix + workerSHA + "/device-studio-api"
	if err := c.putVerified(ctx, route, workerKey, workerBytes, "application/octet-stream", workerSHA); err != nil {
		return jobs.Job{}, false, err
	}
	profileSHA := digestBytes(profileBytes)
	profileKey := inputBlobPrefix + profileSHA + "/inspection-profile"
	if err := c.putVerified(ctx, route, profileKey, profileBytes, "application/x-protobuf+json", profileSHA); err != nil {
		return jobs.Job{}, false, err
	}
	bindings := []jobs.ArtifactBinding{
		{Role: "source_media", ArtifactID: "source-media:" + media.ID, RevisionID: "sha256:" + media.SHA256, SHA256: media.SHA256},
		{Role: "source_size", ArtifactID: "source-media-size:" + media.ID, RevisionID: fmt.Sprintf("size:%d", media.SizeBytes), SHA256: media.SHA256},
		{Role: "inspection_profile", ArtifactID: profileKey, RevisionID: "sha256:" + profileSHA, SHA256: profileSHA},
		{Role: "worker_executable", ArtifactID: workerKey, RevisionID: "sha256:" + workerSHA, SHA256: workerSHA},
		routeBinding(route),
	}
	in := jobs.Input{ProjectID: projectID, Bindings: bindings, ToolProfileDigest: profileDigest, RendererVersion: inspectionRendererVersion, OutputProfile: inspectionOutputProfile, SubjectKind: jobs.SubjectSourceInspection, SubjectID: media.ID}
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: key, Input: in})
	if err != nil {
		// Concurrent resolvers may observe different host profile timestamps or
		// worker bytes while the same source idempotency key is being submitted.
		// Recover the committed immutable winner instead of surfacing a false
		// conflict or replacing its closure.
		prior, lookupErr := c.jobs.FindByIdempotencyKey(ctx, key)
		if lookupErr == nil {
			if verifyErr := verifyInspectionJob(prior, projectID, media, route); verifyErr != nil {
				return jobs.Job{}, false, verifyErr
			}
			attempt, ok := currentAttempt(prior)
			if !ok {
				return jobs.Job{}, false, ErrUnknownOutcome
			}
			if _, repairErr := c.repairInspectionAttempt(ctx, prior, attempt); repairErr != nil {
				return prior, false, repairErr
			}
			return prior, false, nil
		}
		return jobs.Job{}, false, err
	}
	if err := verifyInspectionJob(job, projectID, media, route); err != nil {
		return jobs.Job{}, false, err
	}
	attempt, ok := currentAttempt(job)
	if !ok {
		return jobs.Job{}, false, ErrUnknownOutcome
	}
	if _, err := c.repairInspectionAttempt(ctx, job, attempt); err != nil {
		return job, created, err
	}
	return job, created, nil
}

func inspectionIdempotencyKey(projectID, mediaID, sourceSHA string) string {
	return "source-inspection-v1:" + digestBytes([]byte(projectID+"\x00"+mediaID+"\x00"+sourceSHA+"\x00v1"))
}

func (c *Coordinator) RetrySourceInspection(ctx context.Context, projectID, mediaID string, expectedVersion int64) (jobs.Job, error) {
	ctx, cancel := boundedDetached(ctx, stageTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, err
	}
	media, err := c.media.Get(ctx, projectID, mediaID)
	if err != nil {
		return jobs.Job{}, err
	}
	if media.Kind != projects.MediaKindVideo || media.InspectionJobID == "" {
		return jobs.Job{}, jobs.ErrNotFound
	}
	job, err := c.jobs.Get(ctx, media.InspectionJobID)
	if err != nil {
		return jobs.Job{}, err
	}
	if err := verifyInspectionJob(job, projectID, media, route); err != nil {
		return jobs.Job{}, err
	}
	job, _, err = c.jobs.RetrySourceInspection(ctx, job.ID, expectedVersion)
	if err != nil {
		return jobs.Job{}, err
	}
	attempt, ok := currentAttempt(job)
	if !ok {
		return jobs.Job{}, ErrUnknownOutcome
	}
	if _, err := c.repairInspectionAttempt(ctx, job, attempt); err != nil {
		return job, err
	}
	return job, nil
}

// GetSourceInspectionStatus returns the current owner-ledger state without
// advancing or reconciling the inspection.
func (c *Coordinator) GetSourceInspectionStatus(ctx context.Context, projectID, mediaID string) (jobs.Job, error) {
	ctx, cancel := boundedDetached(ctx, ownerCallTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, err
	}
	media, err := c.media.Get(ctx, projectID, mediaID)
	if err != nil {
		return jobs.Job{}, err
	}
	if media.Kind != projects.MediaKindVideo || media.InspectionJobID == "" {
		return jobs.Job{}, jobs.ErrNotFound
	}
	job, err := c.jobs.Get(ctx, media.InspectionJobID)
	if err != nil {
		return jobs.Job{}, err
	}
	if err := verifyInspectionJob(job, projectID, media, route); err != nil {
		return jobs.Job{}, err
	}
	return job, nil
}

// CancelSourceInspection keeps inspection cancellation in the projects/media
// boundary while using the shared native operation and jobs lifecycle.
func (c *Coordinator) CancelSourceInspection(ctx context.Context, projectID, mediaID string, expectedVersion int64) (jobs.Job, error) {
	job, err := c.GetSourceInspectionStatus(ctx, projectID, mediaID)
	if err != nil {
		return jobs.Job{}, err
	}
	job, err = c.Cancel(ctx, job.ID, expectedVersion)
	if err != nil {
		return job, err
	}
	if job.State == jobs.Cancelled || job.State == jobs.Failed || job.State == jobs.Succeeded {
		return c.Advance(ctx, job.ID)
	}
	return job, nil
}

func verifyInspectionJob(job jobs.Job, projectID string, media projects.SourceMedia, route routePin) error {
	if job.Input.SubjectKind != jobs.SubjectSourceInspection || job.Input.SubjectID != media.ID || job.Input.ProjectID != projectID || bindingDigest(job.Input, "source_media") != media.SHA256 || bindingSize(job.Input, "source_media") != media.SizeBytes {
		return jobs.ErrIdempotencyConflict
	}
	return verifyJobRoute(job, route)
}
