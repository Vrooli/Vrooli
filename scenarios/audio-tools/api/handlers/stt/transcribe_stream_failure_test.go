package stt

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt/session"
	"audio-tools/internal/testutil/mocks"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

type captureStreamTransport struct {
	events []*sttv1.TranscribeStreamEvent
}

func (s *captureStreamTransport) Receive() (*sttv1.TranscribeStreamRequest, error) { return nil, nil }
func (s *captureStreamTransport) Send(event *sttv1.TranscribeStreamEvent) error {
	s.events = append(s.events, event)
	return nil
}
func (s *captureStreamTransport) RequestHeader() http.Header { return make(http.Header) }

func TestLogStreamPumpFailureEmitsSafeStructuredMetadata(t *testing.T) {
	lg := mocks.NewFakeLogger()
	secretMarker := "caller-data-must-not-be-logged"
	err := &streamPumpError{stage: "receive", cause: connect.NewError(connect.CodeUnavailable, errors.New(secretMarker))}
	state := session.Snapshot{
		ReceivedSequence: 567, ProcessedSequence: -1, RetainedBytes: 18_612_180,
		Replay: []session.Chunk{{EndSample: 9_306_090}}, Committed: make([]session.Segment, 60),
	}

	logStreamPumpFailure(lg, err, state)

	require.Len(t, lg.Entries(), 1)
	line := lg.Entries()[0]
	for _, want := range []string{
		"event=stt_stream_input_failure", "stage=receive", "cause_kind=connect_error",
		"connect_code=unavailable", "outcome=error", "received_sequence=567",
		"processed_sequence=-1", "retained_bytes=18612180", "received_end_sample=9306090",
		"committed_segments=60",
	} {
		require.Contains(t, line, want)
	}
	require.NotContains(t, line, secretMarker)
	require.NotContains(t, line, "text=")
	require.NotContains(t, line, "audio=")
	require.NotContains(t, line, "session_id=")
	_, kind, code, outcome := streamPumpFailureMetadata(err)
	require.Equal(t, "connect_error", kind)
	require.Equal(t, "unavailable", code)
	require.Equal(t, "error", outcome)
}

func TestPipelineFailureCannotBePromotedToSuccessfulEOF(t *testing.T) {
	ledger, err := session.New(session.Config{SessionID: "session-a", ResumeToken: "token-a", MaxBytes: 8})
	require.NoError(t, err)
	_, err = ledger.Receive(session.Chunk{Sequence: 0, StartSample: 0, EndSample: 2, Audio: []byte{1, 2}})
	require.NoError(t, err)

	// The stream path drains runErr after EOF and then enters finishStream. A
	// pipeline failure must make that normal-input EOF explicitly unsuccessful.
	markStreamPipelineFailure(ledger)

	state := ledger.Snapshot()
	require.Equal(t, session.TerminalReason("pipeline_failed"), state.TerminalReason)
	require.Equal(t, int64(-1), state.ProcessedSequence)
	require.Len(t, state.Replay, 1, "unprocessed audio must remain recoverable")
	require.ErrorIs(t, ledger.Complete(), session.ErrTerminal, "EOF completion must not override pipeline failure")
}

func TestResumedStreamPreservesPriorCommitWhenReplayMovesVADBoundary(t *testing.T) {
	registry := session.NewRegistry(0)
	ledger, _, err := registry.Open("resume-segment", "resume-token")
	require.NoError(t, err)
	existing := session.Segment{ID: "resume-segment:0:0:2", Text: "the first transcript", StartSample: 0, EndSample: 2}
	_, err = ledger.Commit(existing)
	require.NoError(t, err)
	segment := &sttchain.SegmentEvent{SegmentID: existing.ID, Text: "a revised transcript", StartSample: 0, EndSample: 3, SpeakerPolicy: "revised"}
	start := &sttchain.StreamStart{SessionID: "resume-segment"}

	err = commitStreamSegment(context.Background(), registry, ledger, start, segment, 0, true)
	require.NoError(t, err)
	require.Equal(t, existing.Text, segment.Text, "replay must preserve the already committed wording for the same stable segment identity")
	require.Equal(t, existing.EndSample, segment.EndSample, "replay must preserve the original committed range when VAD boundaries move")
	require.Empty(t, segment.SpeakerPolicy)
	state := ledger.Snapshot()
	require.Equal(t, session.TerminalNone, state.TerminalReason)
	require.Equal(t, []session.Segment{existing}, state.Committed)
}

func TestResumedStreamBackfillsCommittedSegmentsWithoutChangingCursors(t *testing.T) {
	ledger, err := session.New(session.Config{SessionID: "backfill-session", ResumeToken: "backfill-token", MaxBytes: 16})
	require.NoError(t, err)
	_, err = ledger.Receive(session.Chunk{Sequence: 0, StartSample: 0, EndSample: 1600, Audio: []byte{1, 2}})
	require.NoError(t, err)
	require.NoError(t, ledger.AcknowledgeProcessed(0)) // PCM is compacted, segment remains durable.
	_, err = ledger.Commit(session.Segment{ID: "stable-segment", Text: "recovered words", StartSample: 0, EndSample: 1600})
	require.NoError(t, err)
	before := ledger.Snapshot()

	stream := &captureStreamTransport{}
	start := &sttv1.StreamStart{SessionId: "backfill-session", Generation: 3}
	require.NoError(t, replayCommittedSegments(stream, ledger, start))
	require.Len(t, stream.events, 1)
	segment := stream.events[0].GetSegment()
	require.Equal(t, "stable-segment", segment.GetSegmentId())
	require.Equal(t, "recovered words", segment.GetText())
	require.Equal(t, uint64(3), segment.GetGeneration())
	require.Equal(t, int64(1600), segment.GetEndSample())
	after := ledger.Snapshot()
	require.Equal(t, before.ProcessedSequence, after.ProcessedSequence, "backfill must not roll back or advance the coverage cursor")
	require.Equal(t, before.ReceivedSequence, after.ReceivedSequence)
	require.Equal(t, before.Committed, after.Committed, "replay must not create duplicate durable commits")
	require.Empty(t, after.Replay, "acknowledged PCM remains compacted")
}
