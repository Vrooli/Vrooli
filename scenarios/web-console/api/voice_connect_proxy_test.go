package main

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
	audiocommonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_common"
	audioruntimev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_runtime"
	audioruntimeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_runtime/audio_runtime_v1connect"
	"google.golang.org/protobuf/reflect/protoreflect"

	audioRuntimeH "web-console/handlers/audio_runtime"
)

type voiceProxyResolver struct {
	url string
	err error
}

func (r voiceProxyResolver) Resolve() (string, error) { return r.url, r.err }

func TestWebConsoleStreamingMessagesMatchAudioToolsWireShape(t *testing.T) {
	pairs := [][2]protoreflect.MessageDescriptor{
		{(&audioruntimev1.TranscribeStreamRequest{}).ProtoReflect().Descriptor(), (&sttv1.TranscribeStreamRequest{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamStart{}).ProtoReflect().Descriptor(), (&sttv1.StreamStart{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamConfig{}).ProtoReflect().Descriptor(), (&sttv1.StreamConfig{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamAudioChunk{}).ProtoReflect().Descriptor(), (&sttv1.StreamAudioChunk{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamEnd{}).ProtoReflect().Descriptor(), (&sttv1.StreamEnd{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.TranscribeStreamEvent{}).ProtoReflect().Descriptor(), (&sttv1.TranscribeStreamEvent{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamSegment{}).ProtoReflect().Descriptor(), (&sttv1.StreamSegment{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamPartial{}).ProtoReflect().Descriptor(), (&sttv1.StreamPartial{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamWakeWord{}).ProtoReflect().Descriptor(), (&sttv1.StreamWakeWord{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamSpeakerRejection{}).ProtoReflect().Descriptor(), (&sttv1.StreamSpeakerRejection{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamError{}).ProtoReflect().Descriptor(), (&sttv1.StreamError{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamVadState{}).ProtoReflect().Descriptor(), (&sttv1.StreamVadState{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamAcknowledgement{}).ProtoReflect().Descriptor(), (&sttv1.StreamAcknowledgement{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamSessionStatus{}).ProtoReflect().Descriptor(), (&sttv1.StreamSessionStatus{}).ProtoReflect().Descriptor()},
		{(&audioruntimev1.StreamDone{}).ProtoReflect().Descriptor(), (&sttv1.StreamDone{}).ProtoReflect().Descriptor()},
	}
	for _, pair := range pairs {
		assertWireMessageShape(t, pair[0], pair[1], map[string]bool{})
	}
}

func assertWireMessageShape(t *testing.T, webConsole, audioTools protoreflect.MessageDescriptor, visited map[string]bool) {
	t.Helper()
	pairKey := string(webConsole.FullName()) + "=" + string(audioTools.FullName())
	if visited[pairKey] {
		return
	}
	visited[pairKey] = true
	if webConsole.Fields().Len() != audioTools.Fields().Len() {
		t.Fatalf("%s has %d fields; %s has %d", webConsole.FullName(), webConsole.Fields().Len(), audioTools.FullName(), audioTools.Fields().Len())
	}
	for i := 0; i < webConsole.Fields().Len(); i++ {
		left, right := webConsole.Fields().Get(i), audioTools.Fields().Get(i)
		if left.Name() != right.Name() || left.Number() != right.Number() || left.Kind() != right.Kind() || left.Cardinality() != right.Cardinality() {
			t.Fatalf("field wire shape differs: %s.%s (%d %s %s), %s.%s (%d %s %s)", webConsole.FullName(), left.Name(), left.Number(), left.Kind(), left.Cardinality(), audioTools.FullName(), right.Name(), right.Number(), right.Kind(), right.Cardinality())
		}
		if (left.ContainingOneof() == nil) != (right.ContainingOneof() == nil) || (left.ContainingOneof() != nil && left.ContainingOneof().Name() != right.ContainingOneof().Name()) {
			t.Fatalf("oneof differs for field %s.%s", webConsole.FullName(), left.Name())
		}
		switch left.Kind() {
		case protoreflect.MessageKind:
			assertWireMessageShape(t, left.Message(), right.Message(), visited)
		case protoreflect.EnumKind:
			assertWireEnumShape(t, left.Enum(), right.Enum())
		}
	}
}

func assertWireEnumShape(t *testing.T, webConsole, audioTools protoreflect.EnumDescriptor) {
	t.Helper()
	if webConsole.Values().Len() != audioTools.Values().Len() {
		t.Fatalf("enum shape differs: %s has %d values, %s has %d", webConsole.FullName(), webConsole.Values().Len(), audioTools.FullName(), audioTools.Values().Len())
	}
	for i := 0; i < webConsole.Values().Len(); i++ {
		left, right := webConsole.Values().Get(i), audioTools.Values().Get(i)
		if left.Name() != right.Name() || left.Number() != right.Number() {
			t.Fatalf("enum wire value differs: %s.%s=%d, %s.%s=%d", webConsole.FullName(), left.Name(), left.Number(), audioTools.FullName(), right.Name(), right.Number())
		}
	}
}

func TestVoiceConnectProxyForwardsWebConsoleBidiContractAndEnvelope(t *testing.T) {
	upstreamHandler := connect.NewBidiStreamHandler(
		sttconnect.STTServiceTranscribeStreamProcedure,
		func(_ context.Context, stream *connect.BidiStream[sttv1.TranscribeStreamRequest, sttv1.TranscribeStreamEvent]) error {
			if got := stream.RequestHeader().Get("X-Audio-User-Identity"); got != "voice-user" {
				t.Errorf("forwarded user identity = %q", got)
			}
			start, err := stream.Receive()
			if err != nil || start.GetStart().SessionId != "proxy-session" {
				t.Errorf("start = %#v, error = %v", start, err)
				return err
			}
			if start.GetStart().GetInputFormat() != 8 || start.GetStart().GetInputSampleRateHz() != 16000 || start.GetStart().GetConfig().GetOverlapMaxStallRejects() != 4 {
				t.Errorf("start contract fields were not preserved: %#v", start.GetStart())
			}
			chunk, err := stream.Receive()
			if err != nil || chunk.GetAudioChunk().Sequence != 12 || chunk.GetAudioChunk().StartSample != 640 || string(chunk.GetAudioChunk().Audio) != "pcm" {
				t.Errorf("audio chunk = %#v, error = %v", chunk, err)
				return err
			}
			if err := stream.Send(&sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_VadState{VadState: &sttv1.StreamVadState{SilenceTimedOut: true, TickSeq: 9}}}); err != nil {
				return err
			}
			end, err := stream.Receive()
			if err != nil || end.GetEnd() == nil {
				t.Errorf("end = %#v, error = %v", end, err)
				return err
			}
			return stream.Send(&sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Done{Done: &sttv1.StreamDone{FinalText: "complete", ProviderTier: 2, ProcessedSequence: 12}}})
		},
	)
	upstream := httptest.NewUnstartedServer(upstreamHandler)
	upstream.EnableHTTP2 = true
	upstream.StartTLS()
	defer upstream.Close()

	proxy := newVoiceConnectProxy(voiceProxyResolver{url: upstream.URL})
	proxy.http = upstream.Client()
	localHandler := connect.NewBidiStreamHandler(
		audioruntimeconnect.AudioRuntimeServiceTranscribeStreamProcedure,
		audioRuntimeH.NewConnectHandler(audioRuntimeH.Deps{VoiceStream: proxy}).TranscribeStream,
	)
	local := httptest.NewUnstartedServer(localHandler)
	local.EnableHTTP2 = true
	local.StartTLS()
	defer local.Close()

	client := audioruntimeconnect.NewAudioRuntimeServiceClient(local.Client(), local.URL)
	stream := client.TranscribeStream(context.Background())
	stream.RequestHeader().Set("X-Audio-User-Identity", "voice-user")
	if err := stream.Send(&audioruntimev1.TranscribeStreamRequest{Payload: &audioruntimev1.TranscribeStreamRequest_Start{Start: &audioruntimev1.StreamStart{
		ProtocolVersion:   2,
		SessionId:         "proxy-session",
		ResumeToken:       "resume",
		InputFormat:       audiocommonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE,
		InputSampleRateHz: 16000,
		Config:            &audioruntimev1.StreamConfig{OverlapMaxStallRejects: 4},
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&audioruntimev1.TranscribeStreamRequest{Payload: &audioruntimev1.TranscribeStreamRequest_AudioChunk{AudioChunk: &audioruntimev1.StreamAudioChunk{Audio: []byte("pcm"), Sequence: 12, StartSample: 640, EndSample: 1280}}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&audioruntimev1.TranscribeStreamRequest{Payload: &audioruntimev1.TranscribeStreamRequest_End{End: &audioruntimev1.StreamEnd{}}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.CloseRequest(); err != nil {
		t.Fatal(err)
	}

	var gotVad, gotDone bool
	for {
		event, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if event.GetVadState() != nil && event.GetVadState().SilenceTimedOut && event.GetVadState().TickSeq == 9 {
			gotVad = true
		}
		if event.GetDone() != nil && event.GetDone().FinalText == "complete" && event.GetDone().ProcessedSequence == 12 {
			gotDone = true
		}
	}
	if !gotVad || !gotDone {
		t.Fatalf("stream events lost through proxy: vad=%v done=%v", gotVad, gotDone)
	}
}
