package audio

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// maxBatchedSplitChunks bounds filter-graph size, per-request temp files, and
// ffmpeg's number of simultaneously open outputs. Larger splits retain the
// established one-process-per-chunk route.
const maxBatchedSplitChunks = 32

// splitUniformBatch performs a small uniform split in one ffmpeg process.
// Each branch is trimmed on the decoded timestamp timeline and written to a
// separate temporary file, so output muxers retain their normal framing.
// Explicit boundaries and larger batches intentionally stay on Split's
// existing trim path.
func splitUniformBatch(ctx context.Context, input []byte, format, outFormat string, starts, ends []float64) ([][]byte, bool, error) {
	runner, ok := DefaultRunner.(FileOutputRunner)
	if !ok || len(starts) < 2 || len(starts) > maxBatchedSplitChunks {
		return nil, false, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}
	tmpInput, err := writeTempInputs([][]byte{input})
	if err != nil {
		return nil, true, err
	}
	defer tmpInput.cleanup()

	outputPaths := make([]string, 0, len(starts))
	cleanupOutputs := func() {
		for _, path := range outputPaths {
			_ = os.Remove(path)
		}
	}
	defer cleanupOutputs()
	for range starts {
		f, err := createTempFile("audio-tools-split-output-*.bin")
		if err != nil {
			return nil, true, err
		}
		path := f.Name()
		outputPaths = append(outputPaths, path)
		if err := f.Close(); err != nil {
			return nil, true, err
		}
	}

	var graph strings.Builder
	fmt.Fprintf(&graph, "[0:a]asplit=%d", len(starts))
	for i := range starts {
		fmt.Fprintf(&graph, "[in%d]", i)
	}
	for i, start := range starts {
		fmt.Fprintf(&graph, ";[in%d]atrim=start=%s", i, strconv.FormatFloat(start, 'f', -1, 64))
		if ends[i] > 0 {
			fmt.Fprintf(&graph, ":duration=%s", strconv.FormatFloat(ends[i]-start, 'f', -1, 64))
		}
		fmt.Fprintf(&graph, ",asetpts=PTS-STARTPTS[out%d]", i)
	}

	args := []string{}
	if format != "" {
		args = append(args, "-f", demuxerFor(format))
	}
	args = append(args, "-i", tmpInput.paths[0], "-filter_complex", graph.String())
	for i, path := range outputPaths {
		args = append(args, "-map", fmt.Sprintf("[out%d]", i), "-f", muxerFor(outFormat), path)
	}
	args = append(append([]string(nil), ffmpegBaseArgs...), args...)
	if err := runner.RunToFiles(ctx, "ffmpeg", args...); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, true, fmt.Errorf("%w: %w", ctxErr, err)
		}
		return nil, true, fmt.Errorf("%w: %w", ErrFfmpegExec, err)
	}

	chunks := make([][]byte, len(outputPaths))
	for i, path := range outputPaths {
		if err := ctx.Err(); err != nil {
			return nil, true, err
		}
		chunk, err := os.ReadFile(path)
		if err != nil {
			return nil, true, fmt.Errorf("read split output %d: %w", i, err)
		}
		if len(chunk) == 0 {
			return nil, true, fmt.Errorf("%w: ffmpeg produced empty audio for split output %d", ErrFfmpegExec, i)
		}
		chunks[i] = chunk
	}
	return chunks, true, nil
}
