package segmenter

import (
	"context"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
)

// MaxSessionInputBytes bounds the raw bytes a single streaming session may
// feed into the decode process, capping ffmpeg resource use and untrusted-
// input exposure. The WS transport bounds session duration by inactivity
// (SessionIdleTimeoutMs); this bounds a fast flood independent of time. At the
// cap the feeder stops and flushes what it
// has (CloseInput), so the session ends cleanly rather than erroring.
const MaxSessionInputBytes = 256 << 20 // 256 MiB

// normalizeChunks builds a per-session decoder that converts the inbound
// chunk stream to canonical PCM and returns a new chunk channel carrying
// that PCM, a StreamStart with InputFormat rewritten to "pcm_s16le" (so
// the strategy's Request.Format matches the bytes it now holds), and a
// cleanup func the caller MUST defer (it cancels the feeder/adapter
// goroutines and tears the decoder down — required for goroutine-leak
// freedom on both clean EOF and ctx cancellation).
//
// Codec resolution is declare-first: when StreamStart.InputFormat names a
// codec it is used directly; otherwise the first chunk is sniffed. An
// undeclared, unrecognizable stream emits a terminal error + Done and
// returns the error.
func (s *Segmenter) normalizeChunks(
	ctx context.Context,
	start sttchain.StreamStart,
	chunks <-chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
) (<-chan sttchain.AudioChunk, sttchain.StreamStart, func(), error) {
	engine := s.deps.Engine
	if engine == nil {
		engine = audioformat.New()
	}

	codec := audioformat.CodecFromString(start.InputFormat)
	var first sttchain.AudioChunk
	haveFirst := false
	if codec == audioformat.CodecUnknown {
		// Undeclared: sniff the first chunk. Hold it so the decode does not
		// lose the leading bytes.
		select {
		case c, ok := <-chunks:
			if !ok {
				// Empty stream: nothing to decode. Hand the strategy a closed
				// empty channel so it emits an empty terminal Done.
				empty := make(chan sttchain.AudioChunk)
				close(empty)
				return empty, start, func() {}, nil
			}
			first, haveFirst = c, true
		case <-ctx.Done():
			empty := make(chan sttchain.AudioChunk)
			close(empty)
			return empty, start, func() {}, nil
		}
		detected, derr := audioformat.Detect(audioformat.CodecUnknown, first.Audio)
		if derr != nil {
			emitTerminal(events, derr)
			return nil, start, func() {}, derr
		}
		codec = detected
	}

	dec, derr := engine.NewStreamDecoder(ctx, codec)
	if derr != nil {
		emitTerminal(events, derr)
		return nil, start, func() {}, derr
	}

	pcmCh := make(chan sttchain.AudioChunk, 16)
	nctx, ncancel := context.WithCancel(ctx)

	// Feeder: inbound chunks → decoder stdin. CloseInput on clean EOF so
	// the decoder flushes its tail; exit on nctx cancel.
	go func() {
		var fed int
		write := func(b []byte) (stop bool) {
			if err := dec.Write(b); err != nil {
				return true
			}
			fed += len(b)
			if fed >= MaxSessionInputBytes {
				_ = dec.CloseInput() // cap reached: flush + end cleanly
				return true
			}
			return false
		}
		if haveFirst {
			if write(first.Audio) {
				return
			}
		}
		for {
			select {
			case c, ok := <-chunks:
				if !ok {
					_ = dec.CloseInput()
					return
				}
				if write(c.Audio) {
					return
				}
			case <-nctx.Done():
				// Flush the decoder's tail on cancel too, not only on a clean
				// EOF: CloseInput lets ffmpeg emit the trailing PCM it has
				// buffered so a drain-then-close teardown (idle timeout, client
				// stop, request cancel) still delivers the final audio instead
				// of dropping it with the killed process.
				_ = dec.CloseInput()
				return
			}
		}
	}()

	// Adapter: decoder PCM frames → strategy chunk channel. Closes pcmCh
	// when the decoder's Frames channel closes (clean EOF / process death)
	// or on nctx cancel, so the strategy's chunk loop terminates.
	go func() {
		defer close(pcmCh)
		var sequence uint64
		var sampleCursor int64
		for {
			select {
			case f, ok := <-dec.Frames():
				if !ok {
					return
				}
				chunk := sttchain.AudioChunk{Audio: f, Sequence: sequence, StartSample: sampleCursor, EndSample: sampleCursor + int64(len(f)/2)}
				sequence++
				sampleCursor = chunk.EndSample
				select {
				case pcmCh <- chunk:
				case <-nctx.Done():
					return
				}
			case <-nctx.Done():
				return
			}
		}
	}()

	nstart := start
	nstart.InputFormat = audioformat.CodecPCMS16LE.String()
	cleanup := func() {
		ncancel()
		_ = dec.Close()
	}
	return pcmCh, nstart, cleanup, nil
}
