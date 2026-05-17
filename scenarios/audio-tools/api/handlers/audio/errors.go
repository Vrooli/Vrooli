package audio

import (
	"errors"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"
)

// requireBytes is a small guard: every audio op requires a non-empty payload.
func requireBytes(b []byte) error {
	if len(b) == 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("audio bytes are required"))
	}
	return nil
}

// mapAudioErr maps internal errors to connect codes. ErrFFmpegMissing
// is FailedPrecondition (operator action required); everything else is
// Internal so the client sees a fatal upstream failure.
func mapAudioErr(err error) error {
	if errors.Is(err, intaudio.ErrFFmpegMissing) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
