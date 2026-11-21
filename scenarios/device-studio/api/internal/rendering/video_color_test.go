package rendering

import (
	"bytes"
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestVideoInputFilterPreservesDarkSRGB(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg unavailable; native color qualification retained separately")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rgb := bytes.Repeat([]byte{7, 11, 21}, 64*64)
	encode := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-threads", "1", "-filter_threads", "1", "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", "64x64", "-i", "pipe:0", "-frames:v", "1", "-vf", "zscale=matrixin=gbr:rangein=full:primariesin=709:transferin=iec61966-2-1:matrix=709:range=limited:primaries=709:transfer=709,format=yuv420p", "-c:v", "libx264", "-threads", "1", "-crf", "0", "-f", "matroska", "pipe:1")
	encode.Stdin = bytes.NewReader(rgb)
	encoded, err := encode.Output()
	if err != nil {
		t.Fatal(err)
	}
	decode := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-threads", "1", "-filter_threads", "1", "-i", "pipe:0", "-frames:v", "1", "-vf", videoInputFilter(VideoInspection{ColorMatrix: "bt709", ColorRange: "tv", ColorPrimaries: "bt709", ColorTransfer: "bt709"}), "-f", "rawvideo", "-pix_fmt", "rgb24", "pipe:1")
	decode.Stdin = bytes.NewReader(encoded)
	actual, err := decode.Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(actual) != len(rgb) {
		t.Fatalf("decoded %d bytes; expected %d", len(actual), len(rgb))
	}
	for i, v := range actual {
		delta := int(v) - int(rgb[i])
		if delta < -6 || delta > 6 {
			t.Fatalf("dark source lifted/crushed: channel %d=%d; source=%d", i%3, v, rgb[i])
		}
	}
}
