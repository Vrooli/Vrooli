package stt

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/sttengine"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

type recordingCapacityReporter struct{ resource string }

func (r *recordingCapacityReporter) Active(_ context.Context, resource string) string {
	r.resource = resource
	return "claim-1"
}
func (*recordingCapacityReporter) Idle(context.Context, string) {}

// TestProtoForEvent_VadStateRoundtrip asserts that the chain→proto bridge
// faithfully carries every field of VadStateEvent into the Connect contract.
func TestProtoForEvent_VadStateRoundtrip(t *testing.T) {
	in := sttchain.StreamEvent{
		Kind: sttchain.StreamEventVadState,
		VadState: &sttchain.VadStateEvent{
			Voiced:           true,
			SilenceElapsedMs: 123,
			SilenceTimeoutMs: 1500,
			TickSeq:          9,
		},
	}
	got := protoForEvent(in)
	require.NotNil(t, got, "vad-state event must produce a non-nil proto envelope")

	vs, ok := got.Event.(*sttv1.TranscribeStreamEvent_VadState)
	require.True(t, ok, "envelope must hold a VadState oneof variant; got %T", got.Event)
	require.NotNil(t, vs.VadState)
	require.Equal(t, true, vs.VadState.Voiced)
	require.Equal(t, int64(123), vs.VadState.SilenceElapsedMs)
	require.Equal(t, int64(1500), vs.VadState.SilenceTimeoutMs)
	require.Equal(t, uint64(9), vs.VadState.TickSeq)
}

// TestProtoForEvent_VadStateNilDropsEvent asserts that a malformed
// vad-state event (nil VadState pointer) returns nil instead of panicking.
// Matches the defensive nil-guard pattern other branches use.
func TestProtoForEvent_VadStateNilDropsEvent(t *testing.T) {
	got := protoForEvent(sttchain.StreamEvent{Kind: sttchain.StreamEventVadState, VadState: nil})
	require.Nil(t, got)
}

// TestProtoForEvent_SpeakerRejectionCarriesScore asserts the chain→proto
// bridge forwards the similarity score and threshold so Connect clients can
// explain a rejection precisely — without them the rejection
// banner can only show 0.00/0.00.
func TestProtoForEvent_SpeakerRejectionCarriesScore(t *testing.T) {
	in := sttchain.StreamEvent{
		Kind: sttchain.StreamEventSpeakerRejection,
		SpeakerRejection: &sttchain.SpeakerRejectionEvent{
			Reason:    "speaker verification failed",
			Score:     0.42,
			Threshold: 0.7,
		},
	}
	got := protoForEvent(in)
	require.NotNil(t, got)
	sr, ok := got.Event.(*sttv1.TranscribeStreamEvent_SpeakerRejection)
	require.True(t, ok, "envelope must hold a SpeakerRejection oneof; got %T", got.Event)
	require.InDelta(t, 0.42, sr.SpeakerRejection.GetScore(), 1e-9)
	require.InDelta(t, 0.7, sr.SpeakerRejection.GetThreshold(), 1e-9)
}

func TestProtoForEvent_SegmentCarriesSpeakerPolicyAndObservedTier(t *testing.T) {
	got := protoForEvent(sttchain.StreamEvent{
		Kind:    sttchain.StreamEventSegment,
		Segment: &sttchain.SegmentEvent{Text: "verified", SpeakerPolicy: "degraded", ProviderTier: sttchain.TierBYOK},
	})
	require.NotNil(t, got)
	segment, ok := got.Event.(*sttv1.TranscribeStreamEvent_Segment)
	require.True(t, ok)
	require.Equal(t, "degraded", segment.Segment.GetSpeakerPolicy())
	require.Equal(t, commonv1.ProviderTier_PROVIDER_TIER_BYOK, segment.Segment.GetProviderTier())
}

func TestProtoForEvent_MapsAllTranscriptEventKinds(t *testing.T) {
	cases := []sttchain.StreamEvent{
		{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: "partial"}},
		{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: "final", StartMs: 1, EndMs: 2, ProviderTier: sttchain.TierLocal, ModelID: "base", SegmentID: "seg"}},
		{Kind: sttchain.StreamEventWakeWord, WakeWord: &sttchain.WakeWordEvent{Score: 0.9, SampleID: "wake"}},
		{Kind: sttchain.StreamEventError, Error: errors.New("stream failure")},
	}
	for _, event := range cases {
		got := protoForEvent(event)
		require.NotNil(t, got, "kind %s", event.Kind)
	}
}

func TestProtoForEvent_NilPayloadsAndUnknownDrop(t *testing.T) {
	for _, event := range []sttchain.StreamEvent{
		{Kind: sttchain.StreamEventPartial}, {Kind: sttchain.StreamEventSegment}, {Kind: sttchain.StreamEventWakeWord}, {Kind: "unknown"},
	} {
		require.Nil(t, protoForEvent(event))
	}
}

func TestStreamErrorCode_UsesBackendState(t *testing.T) {
	require.Equal(t, "stt_busy", streamErrorCode(errors.New("stt_busy: capacity exhausted")))
	require.Equal(t, "input_decode_failed", streamErrorCode(fmt.Errorf("wrap: %w", audioformat.ErrStreamDecode)))
	require.Equal(t, "provider_failure", streamErrorCode(nil))
}

func TestReportStreamActive_OnlyClaimsLocalResources(t *testing.T) {
	reporter := &recordingCapacityReporter{}
	h := &connectHandler{deps: Deps{Registry: sttengine.Default(), Capacity: reporter}}
	claim := h.reportStreamActive(context.Background(), "kyutai")
	require.Equal(t, "claim-1", claim)
	require.Equal(t, "kyutai-stt", reporter.resource)
	require.Empty(t, h.reportStreamActive(context.Background(), "byok"))
}

func TestReportStreamActive_DoesNotWaitForColdEngineInventory(t *testing.T) {
	probeStarted := make(chan struct{}, 2)
	releaseProbes := make(chan struct{})
	defer close(releaseProbes)
	resolver := sttengine.NewLiveResolver(func(ctx context.Context, args ...string) ([]byte, error) {
		probeStarted <- struct{}{}
		select {
		case <-releaseProbes:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if len(args) > 1 && args[1] == "acceleration" {
			return []byte(`{"selected":"cpu","placement":{"state":"ok"}}`), nil
		}
		return []byte(`{"installed":true,"running":true,"healthy":true,"status_code":"ok","resource":{"observed_mode":"cpu"}}`), nil
	}, time.Minute)
	registry := sttengine.New(sttengine.Manifest{Engines: []sttengine.Engine{{ID: "engine", Kind: sttengine.KindLocalResource, Resource: "resource"}}})
	reporter := &recordingCapacityReporter{}
	h := &connectHandler{deps: Deps{Registry: registry, EngineResolver: resolver, Capacity: reporter}}

	started := time.Now()
	require.Empty(t, h.reportStreamActive(context.Background(), ""))
	require.Less(t, time.Since(started), 100*time.Millisecond, "capacity accounting must not block stream admission on cold inventory")
	require.Empty(t, reporter.resource, "unknown inventory must not produce a capacity claim")
	select {
	case <-probeStarted:
	case <-time.After(time.Second):
		t.Fatal("background inventory refresh did not start")
	}
}

func TestProtoForEvent_DecodeErrorKeepsClientMapping(t *testing.T) {
	got := protoForEvent(sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf("decoder: %w", audioformat.ErrStreamDecode)})
	require.Equal(t, "input_decode_failed", got.GetError().GetCode())
	require.Contains(t, got.GetError().GetMessage(), audioformat.ErrStreamDecode.Error())
}

func TestProtoForEvent_ReadyStatusCarriesResolvedProviderIdentity(t *testing.T) {
	got := protoForEvent(sttchain.StreamEvent{
		Kind: sttchain.StreamEventSessionStatus,
		SessionStatus: &sttchain.SessionStatusEvent{
			State: "ready", ProviderID: "catalog-provider", ModelID: "catalog-model", ProviderTier: sttchain.TierLocal,
		},
	})
	status := got.GetSessionStatus()
	require.Equal(t, "catalog-provider", status.GetProviderId())
	require.Equal(t, "catalog-model", status.GetModelId())
	require.Equal(t, commonv1.ProviderTier_PROVIDER_TIER_LOCAL, status.GetProviderTier())
}

func TestProtoForEvent_ReadyStatusMapsEngineIdentityToCanonicalProvider(t *testing.T) {
	got := protoForEvent(sttchain.StreamEvent{
		Kind: sttchain.StreamEventSessionStatus,
		SessionStatus: &sttchain.SessionStatusEvent{
			State: "ready", ProviderID: "sherpa-streaming", ModelID: "zipformer", ProviderTier: sttchain.TierLocal,
		},
	})
	require.Equal(t, "sherpa-stt", got.GetSessionStatus().GetProviderId())
	require.Equal(t, "zipformer", got.GetSessionStatus().GetModelId())
}

func TestProtoForEvent_TypedNilPayloadsAreDropped(t *testing.T) {
	var partial *sttchain.PartialEvent
	var segment *sttchain.SegmentEvent
	var wakeWord *sttchain.WakeWordEvent
	var rejection *sttchain.SpeakerRejectionEvent
	var vad *sttchain.VadStateEvent
	var acknowledgement *sttchain.AcknowledgementEvent
	var status *sttchain.SessionStatusEvent
	for _, event := range []sttchain.StreamEvent{
		{Kind: sttchain.StreamEventPartial, Partial: partial},
		{Kind: sttchain.StreamEventSegment, Segment: segment},
		{Kind: sttchain.StreamEventWakeWord, WakeWord: wakeWord},
		{Kind: sttchain.StreamEventSpeakerRejection, SpeakerRejection: rejection},
		{Kind: sttchain.StreamEventVadState, VadState: vad},
		{Kind: sttchain.StreamEventAcknowledgement, Acknowledgement: acknowledgement},
		{Kind: sttchain.StreamEventSessionStatus, SessionStatus: status},
	} {
		require.Nil(t, protoForEvent(event), "kind %s", event.Kind)
	}
}
