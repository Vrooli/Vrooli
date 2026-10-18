package composition

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"

	core "music-tools/internal/composition"
	"music-tools/internal/styles"
)

const (
	sa3Model     = "stabilityai/stable-audio-3-medium"
	sa3Revision  = "d4d3579395a8ac0df4ab13b5b1d7138ce7290f81"
	sa3Source    = "77e0398c26f7d22d801ec1bb036307e8b231f741"
	sa3Composed  = "6c2f17b97e8a814cbc61fa70d2b97d2ab460ad605632c2702801b22b0e2cd7f0"
	sa3ModelLock = "dfa4287e6957673fa8c7d21224f88cd7bcff83585e2f1af3535c3fe758f3991d"
)

var (
	receiptHex   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	operationHex = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

type sa3Request struct {
	Prompt   string  `json:"prompt"`
	Duration float64 `json:"duration"`
	Seed     uint32  `json:"seed"`
	Steps    int     `json:"steps"`
	CFG      float64 `json:"cfg_scale"`
	Timeout  int     `json:"timeout_seconds"`
	Revision string  `json:"expected_model_revision"`
}

type sa3Receipt struct {
	Version    int        `json:"schema_version"`
	Status     string     `json:"status"`
	Operation  string     `json:"operation_id"`
	RequestSHA string     `json:"request_sha256"`
	Effective  sa3Request `json:"effective_request"`
	Model      struct {
		ID       string `json:"model_id"`
		Revision string `json:"model_revision"`
		Source   string `json:"source_revision"`
		License  string `json:"license"`
		LockSHA  string `json:"model_lock_sha256"`
		Files    map[string]struct {
			SHA   string `json:"sha256"`
			Bytes int64  `json:"bytes"`
		} `json:"files"`
	} `json:"model"`
	Runtime struct {
		Torch            string            `json:"torch"`
		CUDA             string            `json:"cuda"`
		Transformers     string            `json:"transformers"`
		FA               string            `json:"flash_attn"`
		ABI              bool              `json:"cxx11_abi"`
		Composed         string            `json:"declared_composed_sha256"`
		Sources          map[string]string `json:"source_files"`
		Cap              int64             `json:"torch_allocator_ceiling_bytes"`
		Claim            int64             `json:"broker_claim_bytes"`
		Half             bool              `json:"model_half"`
		Compile          *bool             `json:"torch_compile"`
		MainDtype        string            `json:"main_parameter_dtype"`
		ConditionerDtype string            `json:"conditioner_parameter_dtype"`
		Attention        struct {
			Bindings bool `json:"upstream_fa2_bindings_verified"`
		} `json:"attention"`
	} `json:"runtime"`
	Sampling struct {
		Seed      *uint32  `json:"seed"`
		Steps     int      `json:"steps"`
		CFG       *float64 `json:"cfg_scale"`
		Sampler   string   `json:"sampler"`
		Objective string   `json:"diffusion_objective"`
		Chunked   bool     `json:"chunked_decode"`
		Tokens    int      `json:"prompt_tokens"`
		Limit     int      `json:"prompt_token_limit"`
		Truncated bool     `json:"prompt_truncated"`
	} `json:"sampling"`
	Output struct {
		Name     string  `json:"name"`
		SHA      string  `json:"sha256"`
		Bytes    int64   `json:"bytes"`
		Frames   int64   `json:"frames"`
		Rate     int     `json:"sample_rate"`
		Channels int     `json:"channels"`
		Duration float64 `json:"duration_seconds"`
		Format   string  `json:"format"`
		Subtype  string  `json:"subtype"`
	} `json:"output"`
}

type sa3Profile struct {
	Provider      string  `json:"provider"`
	ReceiptRef    string  `json:"receipt_ref"`
	ReceiptSHA256 string  `json:"receipt_sha256"`
	AudioSHA256   string  `json:"audio_sha256"`
	Seed          uint32  `json:"seed"`
	Steps         int     `json:"steps"`
	CFG           float64 `json:"cfg_scale"`
}

func (s *ModuleState) sa3HTTP(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	endpoint := s.SA3URL
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8896"
	}
	u, e := url.Parse(endpoint)
	if e != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || !(u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return nil, fmt.Errorf("STABLE_AUDIO_3_URL must be an HTTP loopback origin")
	}
	if s.HTTP == nil {
		return nil, fmt.Errorf("resource HTTP client missing")
	}
	request, e := http.NewRequestWithContext(ctx, method, endpoint+path, bytes.NewReader(body))
	if e != nil {
		return nil, e
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	client := *s.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return fmt.Errorf("local resource redirects refused") }
	return client.Do(request)
}

func boundedResourceBody(r io.Reader, limit int64) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, limit+1))
	if e == nil && int64(len(b)) > limit {
		e = fmt.Errorf("resource body exceeds %d bytes", limit)
	}
	return b, e
}

func (s *ModuleState) sa3Ready(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, e := s.sa3HTTP(ctx, http.MethodGet, "/health", nil)
	if e != nil {
		return fmt.Errorf("Stable Audio 3 unavailable; use vrooli resource start stable-audio-3: %w", e)
	}
	defer response.Body.Close()
	body, e := boundedResourceBody(response.Body, 65536)
	if e != nil {
		return e
	}
	var health struct {
		Loaded   bool   `json:"model_loaded"`
		ID       string `json:"model_id"`
		Revision string `json:"model_revision"`
		Device   string `json:"device"`
		Error    string `json:"error"`
	}
	if e = json.Unmarshal(body, &health); e != nil || response.StatusCode != http.StatusOK || !health.Loaded || health.ID != sa3Model || health.Revision != sa3Revision || health.Device != "cuda" || health.Error != "" {
		return fmt.Errorf("Stable Audio 3 not ready or model identity mismatched; use vrooli resource start stable-audio-3 (%s)", response.Status)
	}
	return nil
}

func validateSA3Receipt(receipt sa3Receipt, request sa3Request, rawRequest []byte) error {
	fail := func(field string) error { return fmt.Errorf("Stable Audio 3 receipt mismatch: %s", field) }
	if receipt.Version != 1 || receipt.Status != "completed" || !operationHex.MatchString(receipt.Operation) || receipt.RequestSHA != fmt.Sprintf("%x", sha256.Sum256(rawRequest)) || receipt.Effective != request {
		return fail("request/effective input")
	}
	m := receipt.Model
	if m.ID != sa3Model || m.Revision != sa3Revision || m.Source != sa3Source || m.LockSHA != sa3ModelLock || m.License != "Stability AI Community License; Gemma terms" || len(m.Files) != 12 {
		return fail("model/license/closure identity")
	}
	for _, file := range m.Files {
		if !receiptHex.MatchString(file.SHA) || file.Bytes <= 0 {
			return fail("model file identity")
		}
	}
	if m.Files["model.safetensors"].SHA != "48d9c65e290e7bcd5194e0633bfc2424a59ee9683f5c2d58762d997b7d8ce0b5" {
		return fail("model weight identity")
	}
	rt := receipt.Runtime
	if rt.Torch != "2.7.1+cu126" || rt.CUDA != "12.6" || rt.Transformers != "5.10.4" || strings.Split(rt.FA, "+")[0] != "2.8.3.post2" || !rt.ABI || rt.Composed != sa3Composed || !rt.Half || rt.Compile == nil || *rt.Compile || rt.MainDtype != "torch.float16" || rt.ConditionerDtype != "torch.bfloat16" || !rt.Attention.Bindings || rt.Cap != 8<<30 || rt.Claim != 9<<30 {
		return fail("runtime/profile identity")
	}
	for file, expected := range map[string]string{
		"server.py":         "f8e62209c6998dfe612b3a28adf0f5d1441334e6326271ff96aaf1237664bd9c",
		"loader.py":         "8072238f61ed978f9a4f04cb205771405f8edf1935dddf8ea3bba218759e2d10",
		"requirements.lock": "7312b94a843cfdaeb1bdc42b8a23a763b3cd4999e1fee09d1a931ace9817abc3",
	} {
		if rt.Sources[file] != expected {
			return fail("source identity")
		}
	}
	sample := receipt.Sampling
	if sample.Seed == nil || *sample.Seed != request.Seed || sample.Steps != request.Steps || sample.CFG == nil || *sample.CFG != request.CFG || sample.Sampler != "pingpong" || sample.Objective != "rf_denoiser" || !sample.Chunked || sample.Truncated || sample.Tokens < 1 || sample.Limit != 256 || sample.Tokens > sample.Limit {
		return fail("actual sampling/tokenizer")
	}
	o := receipt.Output
	if o.Name != "audio.wav" || !receiptHex.MatchString(o.SHA) || o.Frames != int64(request.Duration)*44100 || o.Bytes != 44+o.Frames*6 || o.Bytes > 64<<20 || o.Rate != 44100 || o.Channels != 2 || o.Duration != float64(request.Duration) || o.Format != "WAV" || o.Subtype != "PCM_24" {
		return fail("output identity")
	}
	return nil
}

// The maintained resource emits a canonical stereo PCM24 WAV. Validate actual
// bytes before BlobStore publication, not MIME/receipt claims alone.
func validateSA3WAV(data []byte, frames int64) error {
	if len(data) < 44 || string(data[:4]) != "RIFF" || int64(binary.LittleEndian.Uint32(data[4:8]))+8 != int64(len(data)) || string(data[8:16]) != "WAVEfmt " || binary.LittleEndian.Uint32(data[16:20]) != 16 || binary.LittleEndian.Uint16(data[20:22]) != 1 || binary.LittleEndian.Uint16(data[22:24]) != 2 || binary.LittleEndian.Uint32(data[24:28]) != 44100 || binary.LittleEndian.Uint32(data[28:32]) != 264600 || binary.LittleEndian.Uint16(data[32:34]) != 6 || binary.LittleEndian.Uint16(data[34:36]) != 24 || string(data[36:40]) != "data" || int64(binary.LittleEndian.Uint32(data[40:44])) != frames*6 || int64(len(data)) != 44+frames*6 {
		return fmt.Errorf("Stable Audio 3 actual WAV format/frames mismatch")
	}
	return nil
}

func (s *ModuleState) generateSA3Take(ctx context.Context, take *core.Take, request composeRequest, compiled styles.Compiled) (sa3Profile, error) {
	var profile sa3Profile
	if take.Provenance.Seed < 0 || take.Provenance.Seed > math.MaxUint32 {
		return profile, fmt.Errorf("effective seed is not uint32")
	}
	if e := validateProviderInput(providerSA3, compiled, request, s.PermissiveOnly); e != nil {
		return profile, e
	}
	if e := s.sa3Ready(ctx); e != nil {
		return profile, e
	}
	steps := compiled.Params.Steps
	if steps == 0 {
		steps = 8
	}
	cfg := compiled.Params.Guidance
	if cfg == 0 {
		cfg = 1
	}
	payload := sa3Request{Prompt: compiled.Caption, Duration: float64(request.Duration), Seed: uint32(take.Provenance.Seed), Steps: steps, CFG: cfg, Timeout: 300, Revision: sa3Revision}
	raw, e := json.Marshal(payload)
	if e != nil {
		return profile, e
	}
	response, e := s.sa3HTTP(ctx, http.MethodPost, "/v1/compose", raw)
	if e != nil {
		return profile, fmt.Errorf("SA3 compose failed; remote work may be draining/completed: %w", e)
	}
	receiptData, e := boundedResourceBody(response.Body, 512<<10)
	response.Body.Close()
	if e != nil {
		return profile, e
	}
	if response.StatusCode != http.StatusOK {
		return profile, fmt.Errorf("Stable Audio 3 returned %s: %s", response.Status, string(receiptData))
	}
	var receipt sa3Receipt
	if e = json.Unmarshal(receiptData, &receipt); e != nil {
		return profile, e
	}
	if e = validateSA3Receipt(receipt, payload, raw); e != nil {
		return profile, e
	}
	response, e = s.sa3HTTP(ctx, http.MethodGet, "/v1/outputs/"+receipt.Operation+"/audio", nil)
	if e != nil {
		return profile, e
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return profile, fmt.Errorf("SA3 audio returned %s", response.Status)
	}
	audio, e := boundedResourceBody(response.Body, receipt.Output.Bytes)
	if e != nil {
		return profile, e
	}
	if e = validateSA3WAV(audio, receipt.Output.Frames); e != nil {
		return profile, e
	}
	if fmt.Sprintf("%x", sha256.Sum256(audio)) != receipt.Output.SHA {
		return profile, fmt.Errorf("SA3 audio checksum mismatch")
	}
	if e = ctx.Err(); e != nil {
		return profile, e
	}
	if s.Blobs == nil {
		return profile, fmt.Errorf("managed audio storage required")
	}
	audioRef := "out/" + take.ID + ".wav"
	receiptRef := "generation-receipts/" + take.ID + ".json"

	// Keep the original resource output intact. Retain its exact receipt before
	// publishing managed audio; the pool advertises neither until both succeed.
	if e = s.Blobs.PutOwned(ctx, receiptRef, take.ID, bytes.NewReader(receiptData), "application/json", false); e != nil {
		return profile, fmt.Errorf("retain receipt %s for resource output %s (original retained): %w", receiptRef, receipt.Operation, e)
	}
	if e = s.Blobs.PutOwned(ctx, audioRef, take.ID, bytes.NewReader(audio), "audio/wav", true); e != nil {
		return profile, fmt.Errorf("retain audio %s; exact receipt retained at %s, original resource output %s retained: %w", audioRef, receiptRef, receipt.Operation, e)
	}

	take.BlobRef = audioRef
	take.Provenance.ModelID = sa3Model
	take.Provenance.LicenseLane = "restricted"
	take.Provenance.AppliedRung = "resource-resident-medium"
	take.Provenance.CaptionAsSent = payload.Prompt
	return sa3Profile{Provider: providerSA3, ReceiptRef: receiptRef, ReceiptSHA256: fmt.Sprintf("%x", sha256.Sum256(receiptData)), AudioSHA256: receipt.Output.SHA, Seed: payload.Seed, Steps: steps, CFG: cfg}, nil
}

func (s *ModuleState) receiptTake(w http.ResponseWriter, r *http.Request) {
	take, ok := s.Pool.Get(mux.Vars(r)["id"])
	if !ok || s.Blobs == nil {
		http.NotFound(w, r)
		return
	}
	body, mime, e := s.Blobs.Get(r.Context(), "generation-receipts/"+take.ID+".json")
	if e != nil {
		http.Error(w, "retained resource receipt unavailable", http.StatusServiceUnavailable)
		return
	}
	defer body.Close()
	raw, e := boundedResourceBody(body, 512<<10)
	if e != nil {
		http.Error(w, "read retained receipt failed", http.StatusInternalServerError)
		return
	}
	if mime != "application/json" {
		http.Error(w, "retained receipt MIME mismatch", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Music-Receipt-SHA256", fmt.Sprintf("%x", sha256.Sum256(raw)))
	w.Write(raw)
}
