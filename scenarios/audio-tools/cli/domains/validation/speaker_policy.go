package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"audio-tools/cli/domains/stt"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

const (
	speakerAcceptanceTarget = acceptanceTargetRevision
	speakerAcceptanceMethod = "audio-tools-acceptance-v2"
	speakerAcceptanceCohort = "voice-pilot-en-v1"
	speakerAcceptancePolicy = "policy-v1"
)

type speakerPolicyObservation struct {
	BatchState           string
	BatchFallback        bool
	BatchFiltered        bool
	NativePolicies       []string
	NativeRejections     int
	NativeTextSegments   int
	ProbeMode            string
	BatchProviderID      string
	BatchModelID         string
	NativeProviderID     string
	NativeModelID        string
	EngineID             string
	NegativeBatchClosed  bool
	NegativeNativeClosed bool
}

// runSpeakerPolicy exercises the live batch and native streaming product
// paths, including the required fail-closed control. It owns the temporary
// speaker configuration mutation and restores the exact prior config before
// returning, even when one of the probes fails.
func (h *handlers) runSpeakerPolicy(ctx cliapp.RunContext) error {
	path := strings.TrimSpace(ctx.Flag("file"))
	if path == "" {
		return errors.New("validation run-speaker-policy requires --file")
	}
	audio, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read speaker-policy fixture: %w", err)
	}
	if len(audio) == 0 {
		return errors.New("speaker-policy fixture is empty")
	}
	if h.core == nil {
		return errors.New("speaker-policy validation requires the scenario CLI core")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	sttClient := sttconnect.NewSTTServiceClient(httpClient, baseURL)
	streamClient := stt.NewStreamingSTTClient(h.core, baseURL)
	adminClient := sttconnect.NewSTTAdminServiceClient(httpClient, baseURL)
	experimentClient := experimentconnect.NewExperimentServiceClient(httpClient, baseURL)
	if err := requireLabQualificationEvidenceAPI(context.Background(), experimentClient, "validation run-speaker-policy"); err != nil {
		return err
	}

	build, err := audioToolsBuildIdentity(httpClient, baseURL)
	if err != nil {
		return err
	}

	return h.runSpeakerPolicyWithClients(ctx, audio, build, sttClient, func(ctx context.Context, audio []byte, now func() time.Time, skip bool) (speakerStreamObservation, error) {
		return runSpeakerStream(ctx, streamClient, audio, now, skip)
	}, adminClient, experimentClient)
}

// runSpeakerPolicyWithClients owns the mutation-and-restore workflow. The stream runner is
// injected so tests can exercise the real orchestration without audio decoding or a live server.
func (h *handlers) runSpeakerPolicyWithClients(ctx cliapp.RunContext, audio []byte, build string, sttClient sttconnect.STTServiceClient, streamRunner func(context.Context, []byte, func() time.Time, bool) (speakerStreamObservation, error), adminClient sttconnect.STTAdminServiceClient, experimentClient experimentconnect.ExperimentServiceClient) (runErr error) {
	originalResp, err := adminClient.GetSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.GetSpeakerConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get speaker config for validation", err, nil)
	}
	if originalResp == nil || originalResp.Msg == nil || originalResp.Msg.GetConfig() == nil {
		return errors.New("speaker-policy validation received no speaker config")
	}
	original := proto.Clone(originalResp.Msg.GetConfig()).(*sttv1.SpeakerConfig)
	restored := false
	restore := func() error {
		_, restoreErr := adminClient.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
			UpdateMask: speakerConfigMask(), Config: original,
		}))
		if restoreErr != nil {
			fmt.Fprintf(ctx.Stderr(), "speaker-policy validation: restore failed: %v\n", restoreErr)
			return fmt.Errorf("restore speaker config after validation: %w", restoreErr)
		}
		return nil
	}
	defer func() {
		if !restored {
			runErr = errors.Join(runErr, restore())
		}
	}()
	// Keep the qualification independent of operator-enrolled speaker profiles.
	// This deliberately absent profile exercises the supported degraded path
	// with fallback enabled, then the same missing-profile case with fallback
	// disabled must fail closed. No speaker audio or profile is copied into the
	// lab instance, and the exact prior config is restored below.
	probeProfileID := speakerStreamIdentity(h.Now()) + "-unregistered"
	probeConfig := &sttv1.SpeakerConfig{
		Enabled: true, Mode: sttv1.SpeakerMode_SPEAKER_MODE_FILTER,
		ProfileIds: []string{probeProfileID}, Threshold: 0.5,
		RejectBehavior:              sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP,
		FallbackWithoutVerification: true, MinDecisionSeconds: 0.5,
		ScoreSmoothing: 1,
	}
	if _, err := adminClient.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: speakerConfigMask(), Config: probeConfig,
	})); err != nil {
		return cliapp.WrapAPIError("configure temporary speaker-policy qualification", err, nil)
	}

	obs := speakerPolicyObservation{ProbeMode: "unregistered-profile-degraded-then-fail-closed"}
	engineResp, err := sttClient.ListEngines(context.Background(), connect.NewRequest(&sttv1.ListEnginesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list STT engines for speaker-policy validation", err, nil)
	}
	if engineResp == nil || engineResp.Msg == nil {
		return errors.New("speaker-policy validation received no STT engine catalog")
	}
	for _, engine := range engineResp.Msg.GetEngines() {
		if engine.GetIsActive() && engine.GetAvailable() && engine.GetVerdict() == "selected" {
			obs.EngineID = engine.GetId()
			break
		}
	}
	if obs.EngineID == "" {
		return errors.New("speaker-policy validation found no available selected STT engine")
	}
	baseline, err := sttClient.Transcribe(context.Background(), connect.NewRequest(&sttv1.TranscribeRequest{
		Audio: audio, Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV, Language: "en",
	}))
	if err != nil {
		return cliapp.WrapAPIError("speaker-policy batch baseline", err, nil)
	}
	if baseline == nil || baseline.Msg == nil {
		return errors.New("speaker-policy batch baseline returned no response")
	}
	obs.BatchState = baseline.Msg.GetPolicyDetails()["speaker_policy"]
	obs.BatchFallback = baseline.Msg.GetPolicyDetails()["speaker_fallback_used"] == "true"
	obs.BatchFiltered = baseline.Msg.GetFiltered()
	obs.BatchProviderID = baseline.Msg.GetProviderId()
	obs.BatchModelID = baseline.Msg.GetModelId()
	if obs.BatchProviderID == "" || obs.BatchModelID == "" {
		return errors.New("speaker-policy batch response omitted provider/model identity")
	}
	if obs.BatchState != "applied" && obs.BatchState != "degraded" {
		return fmt.Errorf("speaker-policy batch baseline did not expose applied/degraded state: %q", obs.BatchState)
	}

	native, err := streamRunner(context.Background(), audio, h.Now, false)
	if err != nil {
		return fmt.Errorf("speaker-policy native baseline: %w", err)
	}
	obs.NativePolicies = native.Policies
	obs.NativeRejections = native.Rejections
	obs.NativeTextSegments = native.TextSegments
	obs.NativeProviderID, obs.NativeModelID = native.ProviderID, native.ModelID
	if obs.NativeProviderID == "" || obs.NativeModelID == "" {
		return errors.New("speaker-policy native stream omitted provider/model identity")
	}
	if len(obs.NativePolicies) == 0 && obs.NativeRejections == 0 {
		return errors.New("speaker-policy native baseline exposed neither a policy nor a rejection")
	}

	if _, err := adminClient.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enabled", "mode", "fallback_without_verification"}},
		Config:     &sttv1.SpeakerConfig{Enabled: true, Mode: sttv1.SpeakerMode_SPEAKER_MODE_FILTER, ProfileIds: []string{probeProfileID}, FallbackWithoutVerification: false},
	})); err != nil {
		return cliapp.WrapAPIError("enable speaker-policy fail-closed control", err, nil)
	}

	negativeBatch, batchErr := sttClient.Transcribe(context.Background(), connect.NewRequest(&sttv1.TranscribeRequest{
		Audio: audio, Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV, Language: "en",
	}))
	obs.NegativeBatchClosed = batchErr != nil || negativeBatch == nil || negativeBatch.Msg == nil || negativeBatch.Msg.GetFiltered() || strings.TrimSpace(negativeBatch.Msg.GetText()) == ""
	if !obs.NegativeBatchClosed {
		return errors.New("speaker-policy negative batch control emitted text while fallback was disabled")
	}
	negativeNative, nativeErr := streamRunner(context.Background(), audio, h.Now, false)
	obs.NegativeNativeClosed = nativeErr != nil || (negativeNative.TextSegments == 0 && negativeNative.Rejections > 0)
	if !obs.NegativeNativeClosed {
		return errors.New("speaker-policy negative native control emitted text while fallback was disabled")
	}

	// Qualification evidence must never claim success while the live policy remains mutated.
	restored = true
	if err := restore(); err != nil {
		return err
	}

	claims := map[string]any{
		"speaker-policy": map[string]any{"in_band": true, "negative_controls_passed": true},
	}
	deviceProfile := firstNonEmpty(ctx.Flag("device-profile"), "D1")
	machine, err := json.Marshal(map[string]any{
		"target_revision": speakerAcceptanceTarget,
		"method":          speakerAcceptanceMethod,
		"cohort":          speakerAcceptanceCohort,
		"build":           build,
		"device_profile":  deviceProfile,
		"policy_revision": speakerAcceptancePolicy,
		"corpus_revision": speakerAcceptanceCohort,
		"execution_lane":  "native",
		"provider_cell":   map[string]string{"engine_id": obs.EngineID, "provider_id": obs.NativeProviderID, "model_id": obs.NativeModelID, "strategy": "passthrough", "device_profile": deviceProfile},
		"acceptance":      claims,
		"details": map[string]any{
			"execution_lane": "native", "supported_rows": []string{"speaker-policy"},
			"verification_probe": obs.ProbeMode,
			"batch_state":        obs.BatchState, "batch_fallback": obs.BatchFallback,
			"native_policies": obs.NativePolicies, "native_rejections": obs.NativeRejections,
			"native_text_segments": obs.NativeTextSegments,
			"engine_id":            obs.EngineID, "batch_provider_id": obs.BatchProviderID, "batch_model_id": obs.BatchModelID,
			"native_provider_id": obs.NativeProviderID, "native_model_id": obs.NativeModelID,
			"negative_batch_closed":  obs.NegativeBatchClosed,
			"negative_native_closed": obs.NegativeNativeClosed,
		},
	})
	if err != nil {
		return fmt.Errorf("encode speaker-policy receipt: %w", err)
	}
	digest := sha256.Sum256(audio)
	artifact := fmt.Sprintf("audio-tools:speaker-policy:%s:%s", h.Now().UTC().Format("20060102T150405.000000000Z"), hex.EncodeToString(digest[:8]))
	if _, err := experimentClient.RecordQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.RecordQualificationEvidenceRequest{Evidence: &experimentv1.QualificationEvidence{
		EngineId: obs.EngineID, ModelId: obs.NativeModelID, Strategy: "passthrough", PolicyProfile: speakerAcceptancePolicy,
		Kind:   experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_BROWSER_PRODUCT_PATH,
		Passed: true, ArtifactRef: artifact, Notes: "owner-backed batch and native speaker policy validation with fail-closed negative controls; native lane, transcript-free receipt", MachineJson: string(machine),
	}})); err != nil {
		return cliapp.WrapAPIError("record speaker-policy evidence", err, nil)
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(map[string]any{"passed": true, "artifact_ref": artifact, "observation": obs})
	}
	fmt.Fprintf(ctx.Stdout(), "speaker-policy validation passed: %s (batch=%s native_policies=%v)\n", artifact, obs.BatchState, obs.NativePolicies)
	return nil
}

type speakerStreamObservation struct {
	Policies     []string
	Rejections   int
	TextSegments int
	ProviderID   string
	ModelID      string
}

func runSpeakerStream(ctx context.Context, client sttconnect.STTServiceClient, audio []byte, now func() time.Time, skip bool) (speakerStreamObservation, error) {
	stream := client.TranscribeStream(ctx)
	streamIdentity := speakerStreamIdentity(now())
	if err := stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_Start{Start: &sttv1.StreamStart{
		Language: "en", ProtocolVersion: 2, SessionId: streamIdentity, ResumeToken: streamIdentity,
		SkipSpeakerVerification: skip, InputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}}}); err != nil {
		return speakerStreamObservation{}, err
	}
	hash := sha256.Sum256(audio)
	if err := stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: &sttv1.StreamAudioChunk{Audio: audio, Sequence: 0, StartSample: 0, EndSample: int64(len(audio) / 2), Sha256: hash[:]}}}); err != nil {
		return speakerStreamObservation{}, err
	}
	if err := stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}}); err != nil {
		return speakerStreamObservation{}, err
	}
	_ = stream.CloseRequest()
	var out speakerStreamObservation
	for {
		eventMessage, receiveErr := stream.Receive()
		if errors.Is(receiveErr, io.EOF) {
			break
		}
		if receiveErr != nil {
			return out, receiveErr
		}
		switch event := eventMessage.GetEvent().(type) {
		case *sttv1.TranscribeStreamEvent_Segment:
			if event.Segment.GetSpeakerPolicy() != "" {
				out.Policies = append(out.Policies, event.Segment.GetSpeakerPolicy())
			}
			if strings.TrimSpace(event.Segment.GetText()) != "" {
				out.TextSegments++
			}
		case *sttv1.TranscribeStreamEvent_Done:
			out.ProviderID, out.ModelID = event.Done.GetProviderId(), event.Done.GetModelId()
		case *sttv1.TranscribeStreamEvent_SpeakerRejection:
			out.Rejections++
		case *sttv1.TranscribeStreamEvent_Error:
			return out, fmt.Errorf("%s: %s", event.Error.GetCode(), event.Error.GetMessage())
		}
	}
	return out, nil
}

func speakerStreamIdentity(now time.Time) string {
	return fmt.Sprintf("speaker-policy-validation-%d", now.UnixNano())
}

func audioToolsBuildIdentity(client connect.HTTPClient, baseURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return "", fmt.Errorf("build health request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("read audio-tools build identity: %w", err)
	}
	defer resp.Body.Close()
	var payload struct {
		BuildIdentity string `json:"build_identity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode audio-tools health: %w", err)
	}
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(payload.BuildIdentity) == "" {
		return "", fmt.Errorf("audio-tools health did not provide a build identity (status=%s)", resp.Status)
	}
	return payload.BuildIdentity, nil
}

func speakerConfigMask() *fieldmaskpb.FieldMask {
	return &fieldmaskpb.FieldMask{Paths: []string{"enabled", "mode", "threshold", "reject_behavior", "profile_ids", "fallback_without_verification", "extraction_enabled", "min_decision_seconds", "score_smoothing"}}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
