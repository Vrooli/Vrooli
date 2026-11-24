package segmenter

import (
	"context"
	"errors"
	"fmt"
	"sync"

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

	codec, first, haveFirst, empty, err := resolveInputCodec(ctx, start.InputFormat, chunks)
	if err != nil {
		emitTerminal(events, err)
		return nil, start, func() {}, err
	}
	if empty {
		return closedChunks(), start, func() {}, nil
	}

	dec, err := engine.NewStreamDecoder(ctx, codec)
	if err != nil {
		emitTerminal(events, err)
		return nil, start, func() {}, err
	}

	pcmCh := make(chan sttchain.AudioChunk, 16)
	nctx, ncancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	workers.Add(2)
	go feedDecoder(nctx, chunks, dec, first, haveFirst, &workers)
	go adaptDecoderOutput(ctx, nctx, dec, pcmCh, events, &workers)

	nstart := start
	nstart.InputFormat = audioformat.CodecPCMS16LE.String()
	cleanup := func() {
		ncancel()
		_ = dec.Close()
		// Close first (it unblocks any Write/Read), then wait for the two
		// workers. Waiting before Close could deadlock a feeder parked in
		// dec.Write against a process that has not been killed yet.
		workers.Wait()
	}
	return pcmCh, nstart, cleanup, nil
}

// resolveInputCodec preserves the first chunk when an undeclared stream needs
// sniffing. empty is true only for a clean EOF before any bytes arrive.
func resolveInputCodec(
	ctx context.Context,
	inputFormat string,
	chunks <-chan sttchain.AudioChunk,
) (audioformat.Codec, sttchain.AudioChunk, bool, bool, error) {
	codec := audioformat.CodecFromString(inputFormat)
	if codec != audioformat.CodecUnknown {
		return codec, sttchain.AudioChunk{}, false, false, nil
	}

	var first sttchain.AudioChunk
	select {
	case chunk, ok := <-chunks:
		if !ok {
			return audioformat.CodecUnknown, first, false, true, nil
		}
		first = chunk
	case <-ctx.Done():
		return audioformat.CodecUnknown, first, false, true, nil
	}

	codec, err := audioformat.Detect(audioformat.CodecUnknown, first.Audio)
	return codec, first, true, false, err
}

func closedChunks() <-chan sttchain.AudioChunk {
	chunks := make(chan sttchain.AudioChunk)
	close(chunks)
	return chunks
}

// feedDecoder writes inbound chunks in order and closes decoder input on EOF,
// cancellation, or the session byte cap so buffered decoder output can flush.
func feedDecoder(
	ctx context.Context,
	chunks <-chan sttchain.AudioChunk,
	dec audioformat.StreamDecoder,
	first sttchain.AudioChunk,
	haveFirst bool,
	workers *sync.WaitGroup,
) {
	defer workers.Done()
	var fed int
	write := func(data []byte) (stop bool) {
		if err := dec.Write(data); err != nil {
			return true
		}
		fed += len(data)
		if fed >= MaxSessionInputBytes {
			_ = dec.CloseInput()
			return true
		}
		return false
	}
	if haveFirst && write(first.Audio) {
		return
	}
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				_ = dec.CloseInput()
				return
			}
			if write(chunk.Audio) {
				return
			}
		case <-ctx.Done():
			// Flush the decoder's tail on cancel too, not only on a clean EOF.
			_ = dec.CloseInput()
			return
		}
	}
}

// adaptDecoderOutput forwards decoded PCM with contiguous sample offsets and
// reports decoder failures that would otherwise look like a clean EOF.
func adaptDecoderOutput(
	ctx context.Context,
	nctx context.Context,
	dec audioformat.StreamDecoder,
	pcmCh chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
	workers *sync.WaitGroup,
) {
	defer workers.Done()
	defer close(pcmCh)
	var sequence uint64
	var sampleCursor int64
	for {
		select {
		case frame, ok := <-dec.Frames():
			if !ok {
				if decodeErr := dec.Err(); decodeErr != nil && ctx.Err() == nil && !errors.Is(decodeErr, context.Canceled) {
					emitDecoderFailure(nctx, events, decodeErr)
				}
				return
			}
			chunk := sttchain.AudioChunk{Audio: frame, Sequence: sequence, StartSample: sampleCursor, EndSample: sampleCursor + int64(len(frame)/2)}
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
}

func emitDecoderFailure(ctx context.Context, events chan<- sttchain.StreamEvent, decodeErr error) {
	event := sttchain.StreamEvent{
		Kind:  sttchain.StreamEventError,
		Error: fmt.Errorf("%w: %v", audioformat.ErrStreamDecode, decodeErr),
	}
	select {
	case events <- event:
	case <-ctx.Done():
	}
}
