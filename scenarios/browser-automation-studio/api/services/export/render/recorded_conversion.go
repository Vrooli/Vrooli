package render

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ConvertRecordedMedia owns its temporary conversion and preserves original bytes.
func ConvertRecordedMedia(ctx context.Context, input, format string) (*RenderedMedia, error) {
	if err := validateRecordedMedia(ctx, input); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "bas-export-convert-")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	path := filepath.Join(directory, "recording."+format)
	extension := strings.ToLower(filepath.Ext(input))
	encoder := NewFFmpegEncoder(DetectFFmpegBinary())
	switch {
	case format == "webm" && extension != ".webm":
		err = fmt.Errorf("source video is not WebM")
	case format == "webm" || format == "mp4" && extension == ".mp4":
		err = copyRecording(ctx, input, path)
	case format == "mp4":
		err = encoder.ConvertToMP4(ctx, input, path)
	case format == "gif":
		err = encoder.ConvertToGIF(ctx, input, path, 0, 0)
	default:
		err = fmt.Errorf("unsupported recorded format %q", format)
	}
	if err != nil {
		cleanup()
		return nil, err
	}
	contentType := "video/" + format
	if format == "gif" {
		contentType = "image/gif"
	}
	return &RenderedMedia{Path: path, ContentType: contentType, Filename: "recording." + format, cleanup: cleanup}, nil
}

func copyRecording(ctx context.Context, input, output string) error {
	source, err := os.Open(input)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.Create(output)
	if err != nil {
		return err
	}
	buffer := make([]byte, 128*1024)
	for err == nil {
		if err = ctx.Err(); err != nil {
			break
		}
		var n int
		n, err = source.Read(buffer)
		if n > 0 {
			if _, writeErr := target.Write(buffer[:n]); writeErr != nil {
				err = writeErr
				break
			}
		}
		if err == io.EOF {
			err = nil
			break
		}
	}
	closeErr := target.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// Raw delivery requires measured video metadata and a complete strict video decode.
// Copy shortcuts additionally require a sniffed container matching the extension.
func validateRecordedMedia(ctx context.Context, input string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Lstat(input)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("recording must be a regular file")
	}
	file, err := os.Open(input)
	if err != nil {
		return err
	}
	var header [512]byte
	n, readErr := file.Read(header[:])
	closeErr := file.Close()
	if readErr != nil && readErr != io.EOF {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	detected := http.DetectContentType(header[:n])
	extension := strings.ToLower(filepath.Ext(input))
	if extension == ".mp4" && detected != "video/mp4" || extension == ".webm" && detected != "video/webm" {
		return fmt.Errorf("recorded container does not match %s source extension", extension)
	}
	meta, err := probeRecordedVideo(ctx, input)
	if err != nil {
		return err
	}
	if extension == ".mp4" && !strings.Contains(meta.format, "mp4") || extension == ".webm" && !strings.Contains(meta.format, "webm") {
		return fmt.Errorf("recorded container does not match source extension")
	}
	decodeCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	command := exec.CommandContext(decodeCtx, DetectFFmpegBinary(), "-nostdin", "-v", "error", "-xerror", "-err_detect", "explode", "-threads", "1", "-filter_threads", "1", "-i", input, "-map", "0:v:0", "-map", "0:a?", "-progress", "pipe:1", "-nostats", "-threads", "1", "-f", "null", "-")
	command.WaitDelay = 2 * time.Second
	diagnostic := &recordingDiagnostic{}
	progress := &recordingProgress{}
	command.Stdout = progress
	command.Stderr = diagnostic
	if err = command.Run(); err != nil {
		return fmt.Errorf("recorded video full decode failed: %w: %s", err, diagnostic.String())
	}
	if progress.frames < 1 {
		return fmt.Errorf("recording contains no decoded video frames")
	}
	return nil
}

type recordingDiagnostic struct{ bytes.Buffer }

func (d *recordingDiagnostic) Write(data []byte) (int, error) {
	n := len(data)
	if d.Len() < 8192 {
		_, _ = d.Buffer.Write(data[:min(n, 8192-d.Len())])
	}
	return n, nil
}

// Parse the decoder's machine progress without retaining an unbounded transcript.
type recordingProgress struct {
	pending string
	frames  int64
}

func (p *recordingProgress) Write(data []byte) (int, error) {
	lines := strings.Split(p.pending+string(data), "\n")
	p.pending = lines[len(lines)-1]
	if len(p.pending) > 1024 {
		p.pending = ""
	}
	for _, line := range lines[:len(lines)-1] {
		if strings.HasPrefix(line, "frame=") {
			if n, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "frame=")), 10, 64); err == nil && n > p.frames {
				p.frames = n
			}
		}
	}
	return len(data), nil
}
