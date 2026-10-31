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
	speakerAcceptanceTarget = "portable-voice-v1"
	speakerAcceptanceMethod = "audio-tools-acceptance-v1"
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

	build, err := audioToolsBuildIdentity(httpClient, baseURL)
	if err != nil {
		return err
	}

	originalResp, err := adminClient.GetSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.GetSpeakerConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get speaker config for validation", err, nil)
	}
	if originalResp == nil || originalResp.Msg == nil || originalResp.Msg.GetConfig() == nil {
		return errors.New("speaker-policy validation received no speaker config")
	}
	original := proto.Clone(originalResp.Msg.GetConfig()).(*sttv1.SpeakerConfig)
	defer func() {
		_, restoreErr := adminClient.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
			UpdateMask: speakerConfigMask(), Config: original,
		}))
		if restoreErr != nil {
			fmt.Fprintf(ctx.Stderr(), "speaker-policy validation: restore failed: %v\n", restoreErr)
		}
	}()

	obs := speakerPolicyObservation{}
	baseline, err := sttClient.Transcribe(context.Background(), connect.NewRequest(&sttv1.TranscribeRequest{
		Audio: audio, Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV, Language: "en",
	}))
	if err != nil {
		return cliapp.WrapAPIError("speaker-policy batch baseline", err, nil)
	}
	obs.BatchState = baseline.Msg.GetPolicyDetails()["speaker_policy"]
	obs.BatchFallback = baseline.Msg.GetPolicyDetails()["speaker_fallback_used"] == "true"
	obs.BatchFiltered = baseline.Msg.GetFiltered()
	if obs.BatchState != "applied" && obs.BatchState != "degraded" {
		return fmt.Errorf("speaker-policy batch baseline did not expose applied/degraded state: %q", obs.BatchState)
	}

	native, err := runSpeakerStream(context.Background(), streamClient, audio, false)
	if err != nil {
		return fmt.Errorf("speaker-policy native baseline: %w", err)
	}
	obs.NativePolicies = native.Policies
	obs.NativeRejections = native.Rejections
	obs.NativeTextSegments = native.TextSegments
	if len(obs.NativePolicies) == 0 && obs.NativeRejections == 0 {
		return errors.New("speaker-policy native baseline exposed neither a policy nor a rejection")
	}

	if _, err := adminClient.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enabled", "mode", "fallback_without_verification"}},
		Config:     &sttv1.SpeakerConfig{Enabled: true, Mode: sttv1.SpeakerMode_SPEAKER_MODE_FILTER, FallbackWithoutVerification: false},
	})); err != nil {
		return cliapp.WrapAPIError("enable speaker-policy fail-closed control", err, nil)
	}

	negativeBatch, batchErr := sttClient.Transcribe(context.Background(), connect.NewRequest(&sttv1.TranscribeRequest{
		Audio: audio, Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV, Language: "en",
	}))
	obs.NegativeBatchClosed = batchErr != nil || negativeBatch.Msg.GetFiltered() || strings.TrimSpace(negativeBatch.Msg.GetText()) == ""
	if !obs.NegativeBatchClosed {
		return errors.New("speaker-policy negative batch control emitted text while fallback was disabled")
	}
	negativeNative, nativeErr := runSpeakerStream(context.Background(), streamClient, audio, false)
	obs.NegativeNativeClosed = nativeErr != nil || (negativeNative.TextSegments == 0 && negativeNative.Rejections > 0)
	if !obs.NegativeNativeClosed {
		return errors.New("speaker-policy negative native control emitted text while fallback was disabled")
	}

	claims := map[string]any{
		"speaker-policy": map[string]any{"in_band": true, "negative_controls_passed": true},
	}
	machine, err := json.Marshal(map[string]any{
		"target_revision": speakerAcceptanceTarget,
		"method":          speakerAcceptanceMethod,
		"cohort":          speakerAcceptanceCohort,
		"build":           build,
		"device_profile":  firstNonEmpty(ctx.Flag("device-profile"), "D1"),
		"policy_revision": speakerAcceptancePolicy,
		"corpus_revision": speakerAcceptanceCohort,
		"execution_lane":  "native",
		"provider_cell":   map[string]string{"engine_id": "kyutai", "model_id": "kyutai/stt-1b-en_fr", "strategy": "passthrough"},
		"acceptance":      claims,
		"details": map[string]any{
			"execution_lane": "native", "supported_rows": []string{"speaker-policy"},
			"batch_state": obs.BatchState, "batch_fallback": obs.BatchFallback,
			"native_policies": obs.NativePolicies, "native_rejections": obs.NativeRejections,
			"native_text_segments":   obs.NativeTextSegments,
			"negative_batch_closed":  obs.NegativeBatchClosed,
			"negative_native_closed": obs.NegativeNativeClosed,
		},
	})
	if err != nil {
		return fmt.Errorf("encode speaker-policy receipt: %w", err)
	}
	digest := sha256.Sum256(audio)
	artifact := fmt.Sprintf("audio-tools:speaker-policy:%s:%s", h.now().UTC().Format("20060102T150405.000000000Z"), hex.EncodeToString(digest[:8]))
	if _, err := experimentClient.RecordQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.RecordQualificationEvidenceRequest{Evidence: &experimentv1.QualificationEvidence{
		EngineId: "kyutai", ModelId: "kyutai/stt-1b-en_fr", Strategy: "passthrough", PolicyProfile: speakerAcceptancePolicy,
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
}

func runSpeakerStream(ctx context.Context, client sttconnect.STTServiceClient, audio []byte, skip bool) (speakerStreamObservation, error) {
	stream := client.TranscribeStream(ctx)
	streamIdentity := fmt.Sprintf("speaker-policy-validation-%d", time.Now().UnixNano())
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
		case *sttv1.TranscribeStreamEvent_SpeakerRejection:
			out.Rejections++
		case *sttv1.TranscribeStreamEvent_Error:
			return out, fmt.Errorf("%s: %s", event.Error.GetCode(), event.Error.GetMessage())
		}
	}
	return out, nil
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
