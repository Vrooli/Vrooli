package screenrecording

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestUsefulFramesRejectsStaticDesktop(t *testing.T) {
	if usefulFrames("lavfi.signalstats.YAVG=50.0045\nlavfi.signalstats.YAVG=51.2\n") {
		t.Fatal("static dark desktop must not be useful evidence")
	}
}

func TestUsefulFramesAcceptsVisibleApplication(t *testing.T) {
	if !usefulFrames("lavfi.signalstats.YAVG=50.0045\nlavfi.signalstats.YAVG=54.2\n") {
		t.Fatal("visible application frame should be useful evidence")
	}
}

func TestInspectVideoRejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.mp4")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectVideo(context.Background(), path); err == nil {
		t.Fatal("empty recording should be rejected")
	}
}

func TestInspectVideoUsesProbeAndFrameGates(t *testing.T) {
	old := mediaCommand
	t.Cleanup(func() { mediaCommand = old })
	mediaCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "ffprobe" {
			return exec.CommandContext(ctx, "sh", "-c", `printf '%s' '{"streams":[{"codec_name":"h264","codec_type":"video","width":1920,"height":1080,"avg_frame_rate":"15/1","duration":"2","nb_frames":"30"}],"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"2"}}'`)
		}
		return exec.CommandContext(ctx, "sh", "-c", `printf '%s\n' 'lavfi.signalstats.YAVG=220'`)
	}
	path := filepath.Join(t.TempDir(), "recording.mp4")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	inspection, err := InspectVideo(context.Background(), path)
	if err != nil {
		t.Fatalf("InspectVideo() error = %v", err)
	}
	if inspection.Codec != "h264" || inspection.DurationMs != 2000 || !inspection.UsefulFrame {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
}
