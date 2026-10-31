package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
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
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/vrooli/cli-core/cliutil"
)

const (
	yueModel    = "audio-cpp/Yue2-3B-GGUF:yue2-q8"
	yueRevision = "cca74d8980c00f30ab8d52248a367c01ad633ec2"
	yueSource   = "audio.cpp@41c1f8ffa0d13a1b7e366e30ea5fe5ab8d169d48"
)

var (
	digestPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	operationPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

type songComposeRequest struct {
	Profile  string  `json:"profile"`
	Prompt   string  `json:"prompt"`
	Lyrics   string  `json:"lyrics"`
	Duration float64 `json:"duration"`
	Seed     *uint32 `json:"seed"`
	Steps    int     `json:"steps"`
	Guidance float64 `json:"guidance"`
	Revision string  `json:"expected_revision"`
	Timeout  int     `json:"timeout_seconds"`
	Score    *string `json:"score,omitempty"`
}

type scoreArtifact struct {
	SHA       string `json:"sha256"`
	Bytes     int    `json:"bytes"`
	Text      string `json:"text"`
	Source    string `json:"source"`
	Truncated *bool  `json:"truncated"`
}

type songComposeReceipt struct {
	Version     int                `json:"schema_version"`
	Status      string             `json:"status"`
	Operation   string             `json:"operation_id"`
	RequestSHA  string             `json:"request_sha256"`
	Effective   songComposeRequest `json:"effective_request"`
	CatalogSHA  string             `json:"catalog_sha256"`
	ArtifactSHA string             `json:"artifact_sha256"`
	Model       struct {
		ID       string `json:"id"`
		Revision string `json:"revision"`
		Source   string `json:"source_revision"`
	} `json:"model"`
	Worker struct {
		Seed          *uint32 `json:"seed"`
		ModelID       string  `json:"model_id"`
		Revision      string  `json:"revision"`
		FilesVerified int     `json:"weight_files_verified"`
		ChildReaped   bool    `json:"native_child_reaped"`
		Sampling      struct {
			Steps       int     `json:"steps"`
			Guidance    float64 `json:"guidance"`
			ScoreSource string  `json:"score_source"`
		} `json:"sampling"`
		Native struct {
			Backend string `json:"backend"`
			Version string `json:"version"`
		} `json:"native_runtime"`
		Artifacts map[string]scoreArtifact `json:"composition_artifacts"`
	} `json:"worker"`
	Output struct {
		SHA      string  `json:"sha256"`
		Bytes    int64   `json:"bytes"`
		Frames   int64   `json:"frames"`
		Rate     int     `json:"sample_rate"`
		Channels int     `json:"channels"`
		Duration float64 `json:"duration_seconds"`
	} `json:"output"`
}

func decodeSongCompose(data []byte) (songComposeRequest, error) {
	var r songComposeRequest
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, fmt.Errorf("decode compose request: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return r, errors.New("request must contain exactly one JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return r, err
	}
	if value, supplied := fields["score"]; supplied && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return r, errors.New("score must be inline ABC, not null")
	}
	if r.Profile != "yue2-q8" || r.Revision != yueRevision || r.Seed == nil ||
		strings.TrimSpace(r.Prompt) == "" || utf8.RuneCountInString(r.Prompt) > 12000 ||
		strings.TrimSpace(r.Lyrics) == "" || utf8.RuneCountInString(r.Lyrics) > 16000 ||
		r.Duration < 1 || r.Duration > 180 || r.Steps < 1 || r.Steps > 100 ||
		r.Guidance < 0.01 || r.Guidance > 20 || r.Timeout < 1 || r.Timeout > 1800 {
		return r, errors.New("invalid YuE2 profile/revision, text, required uint32 seed, duration or sampling controls")
	}
	if r.Score != nil && (len(*r.Score) == 0 || len(*r.Score) > 16384 ||
		!regexp.MustCompile(`(?m)^X:\s*\d+\s*$`).MatchString(*r.Score) ||
		!regexp.MustCompile(`(?m)^K:\S+\s*$`).MatchString(*r.Score)) {
		return r, errors.New("score must be inline ABC with X/K headers and at most16384 UTF-8 bytes")
	}
	return r, nil
}

func songComposeCommand(args []string) error {
	fs := flag.NewFlagSet("content compose", flag.ContinueOnError)
	requestPath := fs.String("request", "", "Bounded JSON request, with optional inline ABC score")
	output := fs.String("out", "", "New directory for verified audio, receipt and score witnesses")
	expected := fs.String("expect-artifact", "", "Exact artifact digest selected by normal resource lifecycle")
	timeout := fs.Duration("timeout", 32*time.Minute, "Caller deadline; cancellation requests owner drain")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *requestPath == "" || *output == "" || fs.NArg() != 0 || *timeout < time.Second || *timeout > 35*time.Minute {
		return errors.New("--request, --out and --expect-artifact are required; timeout must be1s..35m")
	}
	origin := os.Getenv("SONGGEN_URL")
	if origin == "" {
		origin = "http://127.0.0.1:8897"
	}
	if origin != "http://127.0.0.1:8897" && origin != "http://localhost:8897" {
		return errors.New("SONGGEN_URL must be the declared local resource origin")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	// Explicit canonical attribution preserves authority errors; no token copying
	// or alternate upstream port is used. Proxies and redirects are disabled.
	transport := &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext}
	defer transport.CloseIdleConnections()
	result, err := songComposeToDirectory(ctx, &http.Client{Transport: transport}, cliutil.ApplyAgentIdentity,
		origin, *requestPath, *output, *expected)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func songComposeToDirectory(ctx context.Context, client *http.Client, attribute func(*http.Request) error,
	origin, requestPath, output, expectedArtifact string,
) (map[string]string, error) {
	base, err := url.Parse(origin)
	if err != nil || base.Scheme != "http" || base.User != nil || base.RawQuery != "" || base.Fragment != "" ||
		(base.Path != "" && base.Path != "/") || (base.Hostname() != "localhost" && !net.ParseIP(base.Hostname()).IsLoopback()) {
		return nil, errors.New("resource must be an HTTP loopback origin without credentials, path or query")
	}
	if !digestPattern.MatchString(expectedArtifact) || expectedArtifact == strings.Repeat("0", 64) {
		return nil, errors.New("--expect-artifact requires an exact nonzero SHA256")
	}
	if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("output must be a new directory")
	}
	file, err := openSongRegular(requestPath)
	if err != nil {
		return nil, err
	}
	data, err := songReadBounded(file, 65536)
	file.Close()
	if err != nil {
		return nil, err
	}
	request, err := decodeSongCompose(data)
	if err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(output), ".songgen-export-")
	if err != nil {
		return nil, fmt.Errorf("prepare export before generation: %w", err)
	}
	defer os.RemoveAll(stage)
	bounded := *client
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("resource redirects refused") }
	do := func(method, path string, body []byte) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(origin, "/")+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if err := attribute(req); err != nil {
			return nil, err
		}
		return bounded.Do(req)
	}
	healthResponse, err := do(http.MethodGet, "/health", nil)
	if err != nil {
		return nil, err
	}
	healthData, err := songReadBounded(healthResponse.Body, 65536)
	healthResponse.Body.Close()
	if err != nil || healthResponse.StatusCode != http.StatusOK {
		return nil, errors.New("resource health is not ready")
	}
	var health struct {
		Status      string `json:"status"`
		Protocol    string `json:"protocol"`
		ArtifactSHA string `json:"artifact_sha256"`
		CatalogSHA  string `json:"catalog_sha256"`
		Profiles    map[string]struct {
			Available bool   `json:"available"`
			ModelID   string `json:"model_id"`
			Revision  string `json:"revision"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(healthData, &health); err != nil {
		return nil, err
	}
	profile := health.Profiles[request.Profile]
	if health.Status != "ready" || health.Protocol != "song-v1" || health.ArtifactSHA != expectedArtifact ||
		!digestPattern.MatchString(health.CatalogSHA) || !profile.Available || profile.ModelID != yueModel || profile.Revision != yueRevision {
		return nil, errors.New("served artifact/catalog/profile does not match the requested qualified runtime")
	}
	response, err := do(http.MethodPost, "/v1/compose", data)
	if err != nil {
		return nil, fmt.Errorf("compose transport failed; owner work may be draining or completed, reconcile before retry: %w", err)
	}
	receiptData, err := songReadBounded(response.Body, 512<<10)
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resource compose returned %s: %s", response.Status, receiptData)
	}
	var receipt songComposeReceipt
	if err := json.Unmarshal(receiptData, &receipt); err != nil {
		return nil, err
	}
	w := receipt.Worker
	scoreSource := "generated"
	if request.Score != nil {
		scoreSource = "provided"
	}
	if receipt.Version != 1 || receipt.Status != "completed" || !operationPattern.MatchString(receipt.Operation) ||
		receipt.RequestSHA != fmt.Sprintf("%x", sha256.Sum256(data)) || !reflect.DeepEqual(receipt.Effective, request) ||
		receipt.ArtifactSHA != expectedArtifact || receipt.CatalogSHA != health.CatalogSHA ||
		receipt.Model.ID != yueModel || receipt.Model.Revision != yueRevision || receipt.Model.Source != yueSource ||
		w.Seed == nil || *w.Seed != *request.Seed || w.ModelID != yueModel || w.Revision != yueRevision || w.FilesVerified != 8 ||
		w.Sampling.Steps != request.Steps || w.Sampling.Guidance != request.Guidance || w.Sampling.ScoreSource != scoreSource ||
		w.Native.Backend != "cuda" || w.Native.Version != "0.9.3" || !w.ChildReaped {
		return nil, errors.New("receipt does not verify exact input/model/runtime/seed/sampling/score conditioning")
	}
	for _, name := range []string{"score", "semantic"} {
		a, ok := w.Artifacts[name]
		if !ok || a.Text == "" || len(a.Text) > 262144 || a.Bytes != len(a.Text) || a.Truncated == nil ||
			a.SHA != fmt.Sprintf("%x", sha256.Sum256([]byte(a.Text))) ||
			(name == "score" && (a.Source != scoreSource || (request.Score != nil && a.Text != *request.Score))) {
			return nil, fmt.Errorf("missing or mismatched %s witness", name)
		}
		if name == "semantic" {
			var tokens []int
			if err := json.Unmarshal([]byte(a.Text), &tokens); err != nil || len(tokens) == 0 || len(tokens) > int(request.Duration*25) {
				return nil, errors.New("invalid semantic frame witness")
			}
			for _, token := range tokens {
				if token < 0 || token >= 32768 {
					return nil, errors.New("semantic token outside codec range")
				}
			}
		}
		filename := "score.abc"
		if name == "semantic" {
			filename = "semantic.json"
		}
		if err := os.WriteFile(filepath.Join(stage, filename), []byte(a.Text), 0o600); err != nil {
			return nil, err
		}
	}
	o := receipt.Output
	if !digestPattern.MatchString(o.SHA) || o.Bytes < 44 || o.Bytes > 128<<20 || o.Rate != 48000 || o.Channels != 2 ||
		o.Frames < 1 || float64(o.Frames)/48000 > request.Duration+1 || o.Duration != float64(o.Frames)/48000 {
		return nil, errors.New("receipt has invalid native PCM output geometry")
	}
	download, err := do(http.MethodGet, "/v1/operations/"+receipt.Operation+"/audio", nil)
	if err != nil {
		return nil, err
	}
	defer download.Body.Close()
	if download.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("audio download returned %s", download.Status)
	}
	wavPath := filepath.Join(stage, "audio.wav")
	wav, err := os.OpenFile(wavPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(wav, hasher), io.LimitReader(download.Body, o.Bytes+1))
	closeErr := wav.Close()
	if copyErr != nil || closeErr != nil || written != o.Bytes || fmt.Sprintf("%x", hasher.Sum(nil)) != o.SHA {
		return nil, errors.New("audio export length/hash/write does not match receipt")
	}
	if err := verifySongWAV(wavPath, o.Frames); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(stage, "receipt.json"), receiptData, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(stage, "request.json"), data, 0o600); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := publishSongDirectory(stage, output); err != nil {
		return nil, fmt.Errorf("publish new output without replacement: %w", err)
	}
	abs, _ := filepath.Abs(output)
	return map[string]string{
		"status": "completed", "output": abs, "operation_id": receipt.Operation,
		"audio_sha256": o.SHA, "receipt_sha256": fmt.Sprintf("%x", sha256.Sum256(receiptData)),
	}, nil
}

func songReadBounded(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if int64(len(data)) > limit {
		return nil, errors.New("response exceeds bounded byte limit")
	}
	return data, err
}

func verifySongWAV(path string, frames int64) error {
	data, err := os.ReadFile(path) // bounded by the verified download limit
	if err != nil {
		return err
	}
	if len(data) < 44 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" ||
		uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
		return errors.New("invalid RIFF WAV envelope")
	}
	format, audio := false, false
	for offset := 12; offset < len(data); {
		if offset+8 > len(data) {
			return errors.New("incomplete WAV chunk")
		}
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		start, end := offset+8, offset+8+size
		if end > len(data) {
			return errors.New("truncated WAV chunk")
		}
		switch string(data[offset : offset+4]) {
		case "fmt ":
			if format || size < 16 {
				return errors.New("invalid WAV format chunk")
			}
			f := data[start:end]
			if binary.LittleEndian.Uint16(f[:2]) != 1 || binary.LittleEndian.Uint16(f[2:4]) != 2 ||
				binary.LittleEndian.Uint32(f[4:8]) != 48000 || binary.LittleEndian.Uint32(f[8:12]) != 192000 ||
				binary.LittleEndian.Uint16(f[12:14]) != 4 || binary.LittleEndian.Uint16(f[14:16]) != 16 {
				return errors.New("native output must be stereo48kHz PCM16")
			}
			format = true
		case "data":
			if audio || int64(size) != frames*4 {
				return errors.New("WAV frames differ from receipt")
			}
			audio = true
		}
		offset = end + size%2
		if offset > len(data) {
			return errors.New("missing WAV padding byte")
		}
	}
	if !format || !audio {
		return errors.New("WAV lacks PCM format or audio frames")
	}
	return nil
}
