package stt

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/stt/session"
)

// [REQ:ATD-P0-004] Browser v2 chunk identity is parsed without inference.
func TestDecodeWSV2AudioFrame(t *testing.T) {
	frame := make([]byte, wsV2AudioHeaderBytes+4)
	copy(frame, "ATV2")
	binary.BigEndian.PutUint64(frame[4:12], 7)
	binary.BigEndian.PutUint64(frame[12:20], 160)
	binary.BigEndian.PutUint64(frame[20:28], 162)
	audio := []byte{1, 2, 3, 4}
	digest := sha256.Sum256(audio)
	copy(frame[28:60], digest[:])
	copy(frame[60:], audio)
	got, err := decodeWSV2AudioFrame(frame)
	require.NoError(t, err)
	require.Equal(t, uint64(7), got.Sequence)
	require.Equal(t, int64(160), got.StartSample)
	require.Equal(t, int64(162), got.EndSample)
	require.Equal(t, []byte{1, 2, 3, 4}, got.Audio)
	require.Equal(t, digest[:], got.Digest)
}

func encodeWSV2AudioFrameForTest(sequence uint64, startSample, endSample int64, audio []byte) []byte {
	frame := make([]byte, wsV2AudioHeaderBytes+len(audio))
	copy(frame, "ATV2")
	binary.BigEndian.PutUint64(frame[4:12], sequence)
	binary.BigEndian.PutUint64(frame[12:20], uint64(startSample))
	binary.BigEndian.PutUint64(frame[20:28], uint64(endSample))
	digest := sha256.Sum256(audio)
	copy(frame[28:60], digest[:])
	copy(frame[60:], audio)
	return frame
}

func TestDecodeWSV2AudioFrameRejectsMalformedInput(t *testing.T) {
	_, err := decodeWSV2AudioFrame([]byte("ATV2"))
	require.Error(t, err)
	frame := make([]byte, wsV2AudioHeaderBytes+1)
	copy(frame, "bad!")
	_, err = decodeWSV2AudioFrame(frame)
	require.Error(t, err)
}

func TestDecodeWSV2AudioFrameRejectsDigestMismatch(t *testing.T) {
	frame := make([]byte, wsV2AudioHeaderBytes+2)
	copy(frame, "ATV2")
	binary.BigEndian.PutUint64(frame[12:20], 0)
	binary.BigEndian.PutUint64(frame[20:28], 1)
	copy(frame[60:], []byte{1, 2})
	_, err := decodeWSV2AudioFrame(frame)
	require.ErrorContains(t, err, "digest mismatch")
}

func TestReceivedDuplicateIsSkippedOnlyOnInitialConnection(t *testing.T) {
	require.True(t, shouldSkipReceivedDuplicate(session.ReceivedDuplicate, false))
	require.False(t, shouldSkipReceivedDuplicate(session.ReceivedDuplicate, true), "a resumed duplicate must re-enter the fresh segmenter")
	require.False(t, shouldSkipReceivedDuplicate(session.ReceivedNew, false))
}
