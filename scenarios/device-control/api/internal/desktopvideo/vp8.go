// Package desktopvideo encodes bounded native desktop frames for WebRTC.
package desktopvideo

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

const maxFrameBytes = 64 << 20

var ErrEncode = errors.New("desktop VP8 encode failed")

func EncodePNG(ctx context.Context, frame []byte) ([]byte, error) {
	return encode(ctx, frame, "-f", "image2pipe", "-vcodec", "png")
}

func EncodeRGBA(ctx context.Context, width, height int, frame []byte) ([]byte, error) {
	if width <= 0 || height <= 0 || width > 8192 || height > 8192 || width*height > maxFrameBytes/4 || len(frame) != width*height*4 {
		return nil, ErrEncode
	}
	return encode(ctx, frame, "-f", "rawvideo", "-pixel_format", "rgba", "-video_size", fmt.Sprintf("%dx%d", width, height))
}

func encode(ctx context.Context, frame []byte, input ...string) ([]byte, error) {
	if len(frame) == 0 || len(frame) > maxFrameBytes {
		return nil, ErrEncode
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	args := append([]string{"-hide_banner", "-loglevel", "error"}, input...)
	args = append(args, "-i", "pipe:0", "-frames:v", "1", "-vf", "format=yuv420p",
		"-c:v", "libvpx", "-deadline", "realtime", "-cpu-used", "8", "-threads", "4", "-f", "ivf", "pipe:1")
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.WaitDelay = 100 * time.Millisecond
	cmd.Stdin = bytes.NewReader(frame)
	cmd.Stderr = io.Discard
	var output boundedOutput
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return nil, ErrEncode
	}
	data := output.Bytes()
	if len(data) < 44 || string(data[:4]) != "DKIF" || string(data[8:12]) != "VP80" {
		return nil, ErrEncode
	}
	length := int(binary.LittleEndian.Uint32(data[32:36]))
	if length <= 0 || length != len(data)-44 {
		return nil, ErrEncode
	}
	return data[44:], nil
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(data []byte) (int, error) {
	if len(data) > maxFrameBytes-b.Len() {
		return 0, ErrEncode
	}
	return b.Buffer.Write(data)
}
