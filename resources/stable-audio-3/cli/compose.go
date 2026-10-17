package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

const modelID = "stabilityai/stable-audio-3-medium"
const modelRevision = "d4d3579395a8ac0df4ab13b5b1d7138ce7290f81"
const sourceRevision = "77e0398c26f7d22d801ec1bb036307e8b231f741"
const maxAudioBytes = 64 << 20

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

type composeRequest struct {
	Prompt                string  `json:"prompt"`
	Duration              float64 `json:"duration"`
	Seed                  *uint32 `json:"seed"`
	Steps                 int     `json:"steps"`
	CFGScale              float64 `json:"cfg_scale"`
	TimeoutSeconds        int     `json:"timeout_seconds"`
	ExpectedModelRevision string  `json:"expected_model_revision"`
}

type generationReceipt struct {
	SchemaVersion int    `json:"schema_version"`
	Status        string `json:"status"`
	OperationID   string `json:"operation_id"`
	RequestSHA256 string `json:"request_sha256"`
	Model         struct {
		ID       string `json:"model_id"`
		Revision string `json:"model_revision"`
		Source   string `json:"source_revision"`
	} `json:"model"`
	Runtime struct {
		Torch            string            `json:"torch"`
		CUDA             string            `json:"cuda"`
		Transformers     string            `json:"transformers"`
		FlashAttention   string            `json:"flash_attn"`
		CXX11ABI         bool              `json:"cxx11_abi"`
		ComposedSHA256   string            `json:"declared_composed_sha256"`
		SourceFiles      map[string]string `json:"source_files"`
		AllocatorCeiling int64             `json:"torch_allocator_ceiling_bytes"`
		BrokerClaim      int64             `json:"broker_claim_bytes"`
	} `json:"runtime"`
	Sampling struct {
		Seed  *uint32  `json:"seed"`
		Steps int      `json:"steps"`
		CFG   *float64 `json:"cfg_scale"`
	} `json:"sampling"`
	Output struct {
		Name       string `json:"name"`
		SHA256     string `json:"sha256"`
		Bytes      int64  `json:"bytes"`
		Frames     int64  `json:"frames"`
		SampleRate int    `json:"sample_rate"`
		Channels   int    `json:"channels"`
		Format     string `json:"format"`
		Subtype    string `json:"subtype"`
	} `json:"output"`
}

func decodeRequest(data []byte) (composeRequest, error) {
	r := composeRequest{Duration: 48, Steps: 8, CFGScale: 1, TimeoutSeconds: 300, ExpectedModelRevision: modelRevision}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, fmt.Errorf("decode compose request: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return r, errors.New("request must contain exactly one JSON object")
	}
	if strings.TrimSpace(r.Prompt) == "" || len(r.Prompt) > 48000 || r.Seed == nil || r.Duration < 1 || r.Duration > 120 || r.Steps < 1 || r.Steps > 32 || r.CFGScale < 0 || r.CFGScale > 20 || r.TimeoutSeconds < 1 || r.TimeoutSeconds > 300 || r.ExpectedModelRevision != modelRevision {
		return r, errors.New("invalid prompt, required uint32 seed, duration, sampling settings or model revision")
	}
	return r, nil
}

func composeCommand(args []string) error {
	fs := flag.NewFlagSet("content compose", flag.ContinueOnError)
	var requestPath, output, endpoint string
	var timeout time.Duration
	fs.StringVar(&requestPath, "request", "", "Bounded JSON request file")
	fs.StringVar(&output, "out", "", "New output directory; existing output is refused")
	fs.StringVar(&endpoint, "url", os.Getenv("STABLE_AUDIO_3_URL"), "Local managed resource URL")
	fs.DurationVar(&timeout, "timeout", 330*time.Second, "Caller deadline including download; cancellation requests drain, not immediate GPU stop")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if requestPath == "" || output == "" || fs.NArg() != 0 || timeout < time.Second || timeout > 10*time.Minute {
		return errors.New("--request and --out are required; timeout must be 1s..10m")
	}
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8896"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := composeToDirectory(ctx, http.DefaultClient, endpoint, requestPath, output)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func composeToDirectory(ctx context.Context, client *http.Client, endpoint, requestPath, output string) (map[string]string, error) {
	base, err := url.Parse(endpoint)
	if err != nil || base.Scheme != "http" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Path != "" && base.Path != "/") || (base.Hostname() != "localhost" && !net.ParseIP(base.Hostname()).IsLoopback()) {
		return nil, errors.New("resource URL must be an HTTP loopback origin without credentials, query or path")
	}
	if _, err := os.Lstat(output); err == nil || !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("output must be a new directory")
	}
	data, err := readRegularContext(ctx, requestPath, 65536)
	if err != nil {
		return nil, err
	}
	request, err := decodeRequest(data)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Prove the caller-owned export parent is usable before requesting GPU work.
	// Publication still uses NOREPLACE: this stage does not reserve the final name.
	stage, err := os.MkdirTemp(filepath.Dir(output), ".sa3-export-")
	if err != nil {
		return nil, fmt.Errorf("prepare export before generation: %w", err)
	}
	defer os.RemoveAll(stage)
	post, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/v1/compose", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	post.Header.Set("Content-Type", "application/json")
	// The local owner never redirects operations/downloads to another origin.
	boundedClient := *client
	boundedClient.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("resource redirects refused") }
	response, err := boundedClient.Do(post)
	if err != nil {
		return nil, fmt.Errorf("compose request failed; remote work may be draining or completed: %w", err)
	}
	receiptData, readErr := readBounded(response.Body, 512<<10)
	response.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resource compose returned %s: %s", response.Status, string(receiptData))
	}
	var receipt generationReceipt
	if err := json.Unmarshal(receiptData, &receipt); err != nil {
		return nil, fmt.Errorf("decode generation receipt: %w", err)
	}
	requestHash := fmt.Sprintf("%x", sha256.Sum256(data))
	if receipt.SchemaVersion != 1 || receipt.Status != "completed" || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(receipt.OperationID) || receipt.RequestSHA256 != requestHash || receipt.Model.ID != modelID || receipt.Model.Revision != modelRevision || receipt.Model.Source != sourceRevision || receipt.Sampling.Seed == nil || *receipt.Sampling.Seed != *request.Seed || receipt.Sampling.Steps != request.Steps || receipt.Sampling.CFG == nil || *receipt.Sampling.CFG != request.CFGScale {
		return nil, errors.New("generation receipt does not match requested model/input/seed/sampling")
	}
	runtime := receipt.Runtime
	if runtime.Torch != "2.7.1+cu126" || runtime.CUDA != "12.6" || runtime.Transformers != "5.10.4" || strings.Split(runtime.FlashAttention, "+")[0] != "2.8.3.post2" || !runtime.CXX11ABI || !hex64.MatchString(runtime.ComposedSHA256) || runtime.ComposedSHA256 == strings.Repeat("0", 64) || runtime.AllocatorCeiling != 8<<30 || runtime.BrokerClaim != 9<<30 {
		return nil, errors.New("generation receipt has missing/unreviewed runtime or memory profile")
	}
	for _, file := range []string{"server.py", "loader.py", "requirements.lock"} {
		if !hex64.MatchString(runtime.SourceFiles[file]) {
			return nil, fmt.Errorf("generation receipt lacks source identity for %s", file)
		}
	}
	o := receipt.Output
	if o.Name != "audio.wav" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(o.SHA256) || o.Bytes < 44 || o.Bytes > maxAudioBytes || o.Frames != int64(request.Duration*44100) || o.SampleRate != 44100 || o.Channels != 2 || o.Format != "WAV" || o.Subtype != "PCM_24" {
		return nil, errors.New("generation receipt has unsupported output format/length/hash")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	get, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/v1/outputs/"+receipt.OperationID+"/audio", nil)
	download, err := boundedClient.Do(get)
	if err != nil {
		return nil, err
	}
	defer download.Body.Close()
	if download.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned %s", download.Status)
	}
	wavPath := filepath.Join(stage, "audio.wav")
	wav, err := os.OpenFile(wavPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(wav, hasher), io.LimitReader(download.Body, o.Bytes+1))
	closeErr := wav.Close()
	if copyErr != nil || closeErr != nil || written != o.Bytes || hex.EncodeToString(hasher.Sum(nil)) != o.SHA256 {
		return nil, fmt.Errorf("download length/hash/write verification failed: %w", errors.Join(copyErr, closeErr, errors.New("receipt mismatch")))
	}
	if err := verifyWAV(wavPath, o.Frames); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(stage, "generation-receipt.json"), receiptData, 0o600); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := publishDirectory(stage, output); err != nil {
		return nil, fmt.Errorf("publish new output directory: %w", err)
	}
	abs, _ := filepath.Abs(output)
	return map[string]string{"status": "completed", "output": abs, "operation_id": receipt.OperationID, "audio_sha256": o.SHA256, "receipt_sha256": fmt.Sprintf("%x", sha256.Sum256(receiptData))}, nil
}

func readBounded(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if len(data) > int(limit) {
		return nil, errors.New("response exceeds bounded byte limit")
	}
	return data, err
}

func readRegularContext(ctx context.Context, path string, limit int64) ([]byte, error) {
	file, err := openRegular(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return readBounded(file, limit)
}

// Verify the encoded PCM contract independently of receipt metadata.
func verifyWAV(path string, frames int64) error {
	data, err := os.ReadFile(path) // already bounded during download
	if err != nil {
		return err
	}
	if len(data) < 44 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" || int64(binary.LittleEndian.Uint32(data[4:8]))+8 != int64(len(data)) {
		return errors.New("invalid RIFF WAV envelope")
	}
	formatOK, audioOK, seenFormat, seenAudio := false, false, false, false
	for offset := 12; offset < len(data); {
		if offset+8 > len(data) {
			return errors.New("trailing incomplete WAV chunk")
		}
		size := int64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		start, end := int64(offset+8), int64(offset+8)+size
		if end > int64(len(data)) {
			return errors.New("truncated WAV chunk")
		}
		switch string(data[offset : offset+4]) {
		case "fmt ":
			if seenFormat || size < 16 {
				return errors.New("invalid WAV format chunk")
			}
			seenFormat = true
			f := data[start:end]
			formatOK = binary.LittleEndian.Uint16(f[:2]) == 1 && binary.LittleEndian.Uint16(f[2:4]) == 2 && binary.LittleEndian.Uint32(f[4:8]) == 44100 && binary.LittleEndian.Uint32(f[8:12]) == 264600 && binary.LittleEndian.Uint16(f[12:14]) == 6 && binary.LittleEndian.Uint16(f[14:16]) == 24
		case "data":
			if seenAudio {
				return errors.New("duplicate WAV audio chunk")
			}
			seenAudio = true
			audioOK = size == frames*6
		}
		offset = int(end + size%2)
		if offset > len(data) {
			return errors.New("missing WAV padding byte")
		}
	}
	if !formatOK || !audioOK {
		return errors.New("WAV does not match stereo44.1kHz PCM24 frame contract")
	}
	return nil
}
