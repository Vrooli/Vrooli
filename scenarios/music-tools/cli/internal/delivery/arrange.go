// Package delivery owns bounded deterministic caller-local PCM arrangement.
package delivery

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	rate                = 48000
	channels            = 2
	maxPCM              = 128 * 1024 * 1024
	maxTotalPCM         = 256 * 1024 * 1024
	maxSourceBytes      = 256 * 1024 * 1024
	maxTotalSourceBytes = 512 * 1024 * 1024
	algorithm           = "pcm-linear-crossfade-v2"
)

type (
	Segment struct {
		Source string  `json:"source"`
		Start  float64 `json:"start"`
		End    float64 `json:"end"`
		SHA256 string  `json:"sha256,omitempty"`
	}
	Recipe struct {
		Version         int       `json:"version"`
		Segments        []Segment `json:"segments"`
		Crossfade       float64   `json:"crossfade"`
		ExpectedSamples int       `json:"expected_samples"`
	}
	Receipt struct {
		Version      int               `json:"version"`
		Status       string            `json:"status"`
		SampleRate   int               `json:"sample_rate"`
		Channels     int               `json:"channels"`
		Samples      int               `json:"samples"`
		Duration     float64           `json:"duration"`
		SHA256       string            `json:"sha256"`
		Sources      map[string]string `json:"sources"`
		RecipeSHA256 string            `json:"recipe_sha256"`
		Audition     string            `json:"audition"`
		Algorithm    string            `json:"algorithm"`
		Decoder      string            `json:"decoder"`
	}
)

func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
func hashFile(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.r.Read(p)
}

type limitedWriter struct {
	w         io.Writer
	remaining int64
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, fmt.Errorf("media byte budget exceeded")
	}
	n, e := w.w.Write(p)
	w.remaining -= int64(n)
	return n, e
}

type diagnosticBuffer struct{ bytes.Buffer }

func (b *diagnosticBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < 8192 {
		keep := min(len(p), 8192-b.Len())
		_, _ = b.Buffer.Write(p[:keep])
	}
	return n, nil
}

type pcmSource struct {
	path string
	size int64
}

func sourcePath(root, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, _ = filepath.Abs(path)
	return filepath.Clean(path)
}
func readRecipe(path string) ([]byte, Recipe, error) {
	var r Recipe
	f, e := os.Open(path)
	if e != nil {
		return nil, r, e
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, 128*1024+1))
	if e != nil {
		return nil, r, e
	}
	if len(raw) > 128*1024 {
		return nil, r, fmt.Errorf("recipe exceeds 128 KiB")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e = d.Decode(&r); e != nil {
		return nil, r, e
	}
	var extra any
	if e = d.Decode(&extra); e != io.EOF {
		return nil, r, fmt.Errorf("one recipe JSON document required")
	}
	if r.Version != 1 || len(r.Segments) < 1 || len(r.Segments) > 32 || !finite(r.Crossfade) || r.Crossfade < 0 || r.Crossfade > 2 {
		return nil, r, fmt.Errorf("version 1, 1–32 segments and crossfade 0–2 seconds required")
	}
	if r.ExpectedSamples < 1 || r.ExpectedSamples > 600*rate {
		return nil, r, fmt.Errorf("expected_samples must be 1–28800000 stereo frames (600 seconds)")
	}
	expected := 0
	fade := int(math.Round(r.Crossfade * rate))
	for _, s := range r.Segments {
		if s.Source == "" || !finite(s.Start) || !finite(s.End) || s.Start < 0 || s.End <= s.Start || s.End > 600 {
			return nil, r, fmt.Errorf("segment requires a source and finite 0 <= start < end <= 600")
		}
		count := int(math.Round(s.End*rate)) - int(math.Round(s.Start*rate))
		if count < 1 || count < 2*fade {
			return nil, r, fmt.Errorf("crossfade exceeds half a segment or empty rounded segment")
		}
		expected += count
	}
	expected -= fade * (len(r.Segments) - 1)
	if expected != r.ExpectedSamples {
		return nil, r, fmt.Errorf("sample arithmetic mismatch: segments imply %d; expected_samples is %d", expected, r.ExpectedSamples)
	}
	return raw, r, nil
}

// Arrange snapshots bytes actually decoded, mixes fixed-size blocks and publishes
// only a complete bundle. The sibling lock serializes owner invocations; external
// writers must honor that lock. No existing destination is replaced.
func Arrange(ctx context.Context, recipeFile, output string) (*Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	raw, r, e := readRecipe(recipeFile)
	if e != nil {
		return nil, e
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	output, e = filepath.Abs(output)
	if e != nil {
		return nil, e
	}
	lock := output + ".arrange-lock"
	if e = os.Mkdir(lock, 0o700); e != nil {
		return nil, fmt.Errorf("publication lock: %w", e)
	}
	defer os.Remove(lock)
	if _, e = os.Lstat(output); e == nil {
		return nil, fmt.Errorf("output already exists: %s", output)
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	stage, e := os.MkdirTemp(filepath.Dir(output), ".music-arrange-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(stage)
	scratch, e := os.MkdirTemp(filepath.Dir(output), ".music-pcm-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(scratch)
	var identity diagnosticBuffer
	cmd := exec.CommandContext(ctx, "ffmpeg", "-version")
	cmd.Stdout = &identity
	cmd.Stderr = &identity
	cmd.WaitDelay = 2 * time.Second
	if e = cmd.Run(); e != nil {
		return nil, fmt.Errorf("decoder identity: %w", e)
	}
	decoder := strings.SplitN(identity.String(), "\n", 2)[0]
	sources := map[string]pcmSource{}
	hashes := map[string]string{}
	sourceTotal := int64(0)
	pcmTotal := int64(0)
	for _, s := range r.Segments {
		path := sourcePath(filepath.Dir(recipeFile), s.Source)
		if _, ok := sources[path]; ok {
			if s.SHA256 != "" && s.SHA256 != hashes[path] {
				return nil, fmt.Errorf("source digest mismatch: %s", path)
			}
			continue
		}
		if len(sources) >= 8 {
			return nil, fmt.Errorf("at most eight source files per arrangement")
		}
		info, e := os.Lstat(path)
		if e != nil {
			return nil, e
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("source must be a regular file: %s", path)
		}
		input, e := os.Open(path)
		if e != nil {
			return nil, e
		}
		snapshot := filepath.Join(scratch, fmt.Sprintf("source-%d", len(sources)))
		f, e := os.Create(snapshot)
		if e != nil {
			input.Close()
			return nil, e
		}
		h := sha256.New()
		budget := min(int64(maxSourceBytes), int64(maxTotalSourceBytes)-sourceTotal)
		sink := limitedWriter{w: io.MultiWriter(f, h), remaining: budget}
		copied, copyErr := io.CopyBuffer(&sink, contextReader{ctx, input}, make([]byte, 64*1024))
		input.Close()
		closeErr := f.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		sourceTotal += copied
		hashes[path] = hex.EncodeToString(h.Sum(nil))
		if s.SHA256 != "" && s.SHA256 != hashes[path] {
			return nil, fmt.Errorf("source digest mismatch: %s", path)
		}
		pcm := snapshot + ".pcm"
		out, e := os.Create(pcm)
		if e != nil {
			return nil, e
		}
		sink = limitedWriter{w: out, remaining: min(int64(maxPCM), int64(maxTotalPCM)-pcmTotal)}
		var diagnostic diagnosticBuffer
		cmd = exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-v", "error", "-threads", "2", "-i", snapshot, "-vn", "-ac", "2", "-ar", "48000", "-f", "s16le", "-")
		cmd.WaitDelay = 2 * time.Second
		cmd.Stdout = &sink
		cmd.Stderr = &diagnostic
		runErr := cmd.Run()
		closeErr = out.Close()
		if runErr != nil {
			return nil, fmt.Errorf("decode %s: %w: %s", path, runErr, diagnostic.String())
		}
		if closeErr != nil {
			return nil, closeErr
		}
		stat, e := os.Stat(pcm)
		if e != nil {
			return nil, e
		}
		if stat.Size()%4 != 0 {
			return nil, fmt.Errorf("decoder returned incomplete stereo frame")
		}
		pcmTotal += stat.Size()
		sources[path] = pcmSource{pcm, stat.Size()}
	}
	wav := filepath.Join(stage, "arranged.wav")
	out, e := os.OpenFile(wav, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if e != nil {
		return nil, e
	}
	e = mix(ctx, out, r, sources, filepath.Dir(recipeFile))
	closeErr := out.Close()
	if e != nil {
		return nil, e
	}
	if closeErr != nil {
		return nil, closeErr
	}
	hash, e := hashFile(wav)
	if e != nil {
		return nil, e
	}
	recipeHash := sha256.Sum256(raw)
	receipt := &Receipt{Version: 1, Status: "pass", SampleRate: rate, Channels: channels, Samples: r.ExpectedSamples, Duration: float64(r.ExpectedSamples) / rate, SHA256: hash, Sources: hashes, RecipeSHA256: hex.EncodeToString(recipeHash[:]), Audition: "unreviewed", Algorithm: algorithm, Decoder: decoder}
	encoded, e := json.MarshalIndent(receipt, "", "  ")
	if e != nil {
		return nil, e
	}
	if e = os.WriteFile(filepath.Join(stage, "recipe.json"), raw, 0o644); e != nil {
		return nil, e
	}
	if e = os.WriteFile(filepath.Join(stage, "receipt.json"), append(encoded, '\n'), 0o644); e != nil {
		return nil, e
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	if _, e = os.Lstat(output); e == nil {
		return nil, fmt.Errorf("output already exists: %s", output)
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	if e = os.Rename(stage, output); e != nil {
		return nil, e
	}
	return receipt, nil
}

func mix(ctx context.Context, out *os.File, r Recipe, sources map[string]pcmSource, root string) error {
	header := make([]byte, 44)
	size := r.ExpectedSamples * 4
	copy(header, "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+size))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], channels)
	binary.LittleEndian.PutUint32(header[24:], rate)
	binary.LittleEndian.PutUint32(header[28:], rate*4)
	binary.LittleEndian.PutUint16(header[32:], 4)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(size))
	if _, e := out.Write(header); e != nil {
		return e
	}
	fade := int(math.Round(r.Crossfade * rate))
	position := 0
	block := make([]byte, 4096*4)
	old := make([]byte, len(block))
	for i, s := range r.Segments {
		source := sources[sourcePath(root, s.Source)]
		start, end := int(math.Round(s.Start*rate)), int(math.Round(s.End*rate))
		if int64(end)*4 > source.size {
			return fmt.Errorf("segment ends beyond decoded source: %s", s.Source)
		}
		input, e := os.Open(source.path)
		if e != nil {
			return e
		}
		if i > 0 {
			position -= fade
		}
		for frame := 0; frame < end-start; {
			if e = ctx.Err(); e != nil {
				input.Close()
				return e
			}
			count := min(4096, end-start-frame)
			chunk := block[:count*4]
			if _, e = input.ReadAt(chunk, int64(start+frame)*4); e != nil {
				input.Close()
				return e
			}
			overlap := 0
			if i > 0 {
				overlap = min(count, max(0, fade-frame))
			}
			offset := int64(44 + (position+frame)*4)
			if overlap > 0 {
				if _, e = out.ReadAt(old[:overlap*4], offset); e != nil {
					input.Close()
					return e
				}
				for f := 0; f < overlap; f++ {
					weight := float64(frame+f+1) / float64(fade+1)
					for ch := 0; ch < channels; ch++ {
						j := (f*channels + ch) * 2
						a := float64(int16(binary.LittleEndian.Uint16(old[j:])))
						b := float64(int16(binary.LittleEndian.Uint16(chunk[j:])))
						v := math.Round(a*(1-weight) + b*weight)
						binary.LittleEndian.PutUint16(chunk[j:], uint16(int16(v)))
					}
				}
			}
			if _, e = out.WriteAt(chunk, offset); e != nil {
				input.Close()
				return e
			}
			frame += count
		}
		input.Close()
		position += end - start
	}
	if position != r.ExpectedSamples {
		return fmt.Errorf("internal sample count mismatch")
	}
	return ctx.Err()
}
