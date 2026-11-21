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
)

type voiceProxyResolver struct {
	url string
	err error
}

func (r voiceProxyResolver) Resolve() (string, error) { return r.url, r.err }

func TestVoiceConnectProxyForwardsBidiMessagesAndEnvelope(t *testing.T) {
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
			chunk, err := stream.Receive()
			if err != nil || chunk.GetAudioChunk().Sequence != 0 || string(chunk.GetAudioChunk().Audio) != "pcm" {
				t.Errorf("audio chunk = %#v, error = %v", chunk, err)
				return err
			}
			if err := stream.Send(&sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Partial{Partial: &sttv1.StreamPartial{Text: "heard"}}}); err != nil {
				return err
			}
			end, err := stream.Receive()
			if err != nil || end.GetEnd() == nil {
				t.Errorf("end = %#v, error = %v", end, err)
				return err
			}
			return stream.Send(&sttv1.TranscribeStreamEvent{Event: &sttv1.TranscribeStreamEvent_Done{Done: &sttv1.StreamDone{FinalText: "complete"}}})
		},
	)
	upstream := httptest.NewUnstartedServer(upstreamHandler)
	upstream.EnableHTTP2 = true
	upstream.StartTLS()
	defer upstream.Close()

	proxyHandler := connect.NewBidiStreamHandler(
		sttconnect.STTServiceTranscribeStreamProcedure,
		(&voiceConnectProxy{resolver: voiceProxyResolver{url: upstream.URL}, http: upstream.Client()}).transcribeStream,
	)
	proxy := httptest.NewUnstartedServer(proxyHandler)
	proxy.EnableHTTP2 = true
	proxy.StartTLS()
	defer proxy.Close()
	client := sttconnect.NewSTTServiceClient(proxy.Client(), proxy.URL)
	stream := client.TranscribeStream(context.Background())
	stream.RequestHeader().Set("X-Audio-User-Identity", "voice-user")
	if err := stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_Start{Start: &sttv1.StreamStart{ProtocolVersion: 2, SessionId: "proxy-session", ResumeToken: "resume"}}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: &sttv1.StreamAudioChunk{Audio: []byte("pcm"), Sequence: 0}}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&sttv1.TranscribeStreamRequest{Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.CloseRequest(); err != nil {
		t.Fatal(err)
	}

	var events []string
	for {
		event, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if partial := event.GetPartial(); partial != nil {
			events = append(events, partial.Text)
		}
		if done := event.GetDone(); done != nil {
			events = append(events, done.FinalText)
		}
	}
	if len(events) != 2 || events[0] != "heard" || events[1] != "complete" {
		t.Fatalf("events = %v", events)
	}
}
