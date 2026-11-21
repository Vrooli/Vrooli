package delivery

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestArrangeRepeatedSegmentsHasExactDurationAndRetainsReceipt(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("FFmpeg required")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "take.wav")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=330:duration=3", "-ac", "2", "-ar", "48000", source).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	r := Recipe{Version: 1, Segments: []Segment{{Source: "take.wav", Start: 0, End: 1}, {Source: "take.wav", Start: 1, End: 2.5}, {Source: "take.wav", Start: .5, End: 1}}, Crossfade: .02, ExpectedSamples: 142080}
	raw, _ := json.Marshal(r)
	recipe := filepath.Join(dir, "recipe.json")
	os.WriteFile(recipe, raw, 0o644)
	result, err := Arrange(context.Background(), recipe, filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Samples != 142080 || result.Duration != 2.96 || result.Audition != "unreviewed" {
		t.Fatalf("unexpected receipt: %+v", result)
	}
	if _, err := Arrange(context.Background(), recipe, filepath.Join(dir, "out")); err == nil {
		t.Fatal("must preserve existing arrangement")
	}
	r.ExpectedSamples = 48000
	raw, _ = json.Marshal(r)
	os.WriteFile(recipe, raw, 0o644)
	if _, err := Arrange(context.Background(), recipe, filepath.Join(dir, "bad")); err == nil || !strings.Contains(err.Error(), "sample arithmetic mismatch") {
		t.Fatalf("bad arithmetic accepted: %v", err)
	}
	r.ExpectedSamples = 142080
	r.Segments[0].End = 4
	r.ExpectedSamples += 3 * 48000
	raw, _ = json.Marshal(r)
	os.WriteFile(recipe, raw, 0o644)
	if _, err := Arrange(context.Background(), recipe, filepath.Join(dir, "past-end")); err == nil || !strings.Contains(err.Error(), "beyond decoded source") {
		t.Fatalf("out-of-bounds trim accepted: %v", err)
	}
}

func TestArrangeRejectsNonfiniteAndInvalidBudgetBeforeDecoding(t *testing.T) {
	for _, raw := range []string{`{"version":1,"segments":[{"source":"missing","start":-1,"end":2}],"expected_samples":48000}`, `{"version":1,"segments":[],"expected_samples":48000}`, `{"version":1,"segments":[{"source":"missing","end":999}],"expected_samples":48000}`} {
		dir := t.TempDir()
		path := filepath.Join(dir, "r.json")
		os.WriteFile(path, []byte(raw), 0o644)
		if _, err := Arrange(context.Background(), path, filepath.Join(dir, "out")); err == nil {
			t.Fatal("invalid recipe accepted")
		}
	}
}

func TestMixPreservesExactSamplesAndCrossfade(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.pcm")
	data := make([]byte, 8*4)
	for i := 0; i < 8; i++ {
		for ch := 0; ch < 2; ch++ {
			binary.LittleEndian.PutUint16(data[(i*2+ch)*2:], uint16(1000+i*1000))
		}
	}
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(dir, "out.wav"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	// Four + four frames, with a two-frame linear overlap.
	r := Recipe{Segments: []Segment{{Source: "s", End: 4.0 / rate}, {Source: "s", Start: 4.0 / rate, End: 8.0 / rate}}, Crossfade: 2.0 / rate, ExpectedSamples: 6}
	if err = mix(context.Background(), out, r, map[string]pcmSource{sourcePath(dir, "s"): {source, 32}}, dir); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 44+24)
	if _, err = out.ReadAt(got, 0); err != nil {
		t.Fatal(err)
	}
	want := []int16{1000, 2000, 3667, 5333, 7000, 8000}
	if binary.LittleEndian.Uint32(got[40:]) != 24 {
		t.Fatal("wrong WAV frame count")
	}
	for i, v := range want {
		for ch := 0; ch < 2; ch++ {
			if n := int16(binary.LittleEndian.Uint16(got[44+(i*2+ch)*2:])); n != v {
				t.Fatalf("frame %d channel %d = %d want %d", i, ch, n, v)
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = mix(ctx, out, r, map[string]pcmSource{sourcePath(dir, "s"): {source, 32}}, dir); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
}

func TestRecipeRequiresSingleBoundedDocumentAndCanceledWorkPublishesNothing(t *testing.T) {
	dir := t.TempDir()
	recipe := filepath.Join(dir, "recipe.json")
	raw := `{"version":1,"segments":[{"source":"missing","end":1}],"expected_samples":48000}`
	os.WriteFile(recipe, []byte(raw+` {}`), 0o600)
	if _, _, err := readRecipe(recipe); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	os.WriteFile(recipe, []byte(raw), 0o600)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	output := filepath.Join(dir, "out")
	if _, err := Arrange(ctx, recipe, output); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("cancellation published output: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("scratch leaked: %v", entries)
	}
}

func TestFailedDecodeCleansOwnedScratchAndAllowsRetry(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("FFmpeg required")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.wav"), []byte("not audio"), 0o600)
	recipe := filepath.Join(dir, "recipe.json")
	os.WriteFile(recipe, []byte(`{"version":1,"segments":[{"source":"bad.wav","end":1}],"expected_samples":48000}`), 0o600)
	output := filepath.Join(dir, "out")
	if _, err := Arrange(context.Background(), recipe, output); err == nil {
		t.Fatal("invalid decoder output accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("partial output/scratch leaked: %v", entries)
	}
	if _, err := Arrange(context.Background(), recipe, output); err == nil || strings.Contains(err.Error(), "publication lock") {
		t.Fatalf("retry blocked by stale lock: %v", err)
	}
}
