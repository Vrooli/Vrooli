package audio

import (
	"context"
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

// mapAudioErr maps internal errors to actionable connect codes:
//
//   - ErrFFmpegMissing → FailedPrecondition: a dependency is present in the
//     deployment contract but not installed; the operator must install ffmpeg.
//   - context.Canceled / context.DeadlineExceeded → Canceled / DeadlineExceeded:
//     the caller's own signal timed out; reporting InvalidArgument would blame
//     the request body for a transport-level abort.
//   - ErrInvalidArgument / ErrFfmpegExec → InvalidArgument: ffmpeg ran but
//     rejected the input (unsupported/corrupt audio or an unsupported output
//     format), or a request parameter is out of range. This is a
//     caller-fixable problem, not a server bug — the underlying ffmpeg stderr
//     is preserved in the wrapped error so the message stays diagnosable.
//   - everything else → Internal: a genuine unexpected failure.
//
// This is the honest-error contract: precondition/dependency/input failures
// no longer flatten to the useless "Internal error" the Diagnostics page used
// to show.
func mapAudioErr(err error) error {
	switch {
	case errors.Is(err, intaudio.ErrFFmpegMissing):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	case errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, intaudio.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, intaudio.ErrFfmpegExec):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
