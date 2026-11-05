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
	"os"
	"reflect"
	"time"

	core "music-tools/internal/composition"
	"music-tools/internal/models"
	"music-tools/internal/styles"
)

type songRequest struct {
	Profile  string  `json:"profile"`
	Prompt   string  `json:"prompt"`
	Lyrics   string  `json:"lyrics"`
	Duration float64 `json:"duration"`
	Seed     uint32  `json:"seed"`
	Steps    int     `json:"steps"`
	Guidance float64 `json:"guidance"`
	Revision string  `json:"expected_revision"`
	Timeout  int     `json:"timeout_seconds"`
}

func songRequestFor(model models.Model, compiled styles.Compiled, request composeRequest, seed uint32) songRequest {
	steps, guidance := compiled.Params.Steps, compiled.Params.Guidance
	if steps == 0 {
		steps = model.Runtime.DefaultSteps
	}
	if guidance == 0 {
		guidance = model.Runtime.DefaultGuidance
	}
	return songRequest{Profile: model.Runtime.Profile, Prompt: compiled.Caption, Lyrics: styles.ResolveLyrics(compiled.Lyrics),
		Duration: float64(request.Duration), Seed: seed, Steps: steps, Guidance: guidance, Revision: model.Revision, Timeout: model.Runtime.TimeoutSeconds}
}

type songReceipt struct {
	Version     int         `json:"schema_version"`
	Status      string      `json:"status"`
	Operation   string      `json:"operation_id"`
	RequestSHA  string      `json:"request_sha256"`
	Effective   songRequest `json:"effective_request"`
	CatalogSHA  string      `json:"catalog_sha256"`
	ArtifactSHA string      `json:"artifact_sha256"`
	Model       struct {
		ID       string `json:"id"`
		Revision string `json:"revision"`
		Source   string `json:"source_revision"`
		License  string `json:"license"`
	} `json:"model"`
	Worker struct {
		Seed           *uint32           `json:"seed"`
		ModelID        string            `json:"model_id"`
		Revision       string            `json:"revision"`
		FilesVerified  int               `json:"weight_files_verified"`
		VoiceReference bool              `json:"voice_reference_used"`
		Versions       map[string]string `json:"versions"`
		Sampling       struct {
			Steps       *int     `json:"steps"`
			Guidance    *float64 `json:"guidance"`
			DecoderSeed *uint32  `json:"decoder_seed"`
		} `json:"sampling"`
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

func validSongSnapshot(model models.Model) error {
	current, err := declaredCompositionModel(model.Engine, model.Variant)
	if err != nil {
		return err
	}
	if model.Runtime == nil || model.Runtime.Protocol != "song-v1" || current.ID != model.ID || current.Revision != model.Revision || !reflect.DeepEqual(current.Runtime, model.Runtime) {
		return fmt.Errorf("recorded song model identity/runtime no longer matches the declared profile; no fallback")
	}
	return nil
}

func (s *ModuleState) songHTTP(ctx context.Context, model models.Model, method, path string, body []byte) (*http.Response, error) {
	if model.Runtime == nil || s.HTTP == nil {
		return nil, fmt.Errorf("song runtime/client is missing")
	}
	endpoint := os.Getenv(model.Runtime.URLVariable)
	if endpoint == "" {
		endpoint = model.Runtime.DefaultURL
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || !(u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return nil, fmt.Errorf("song resource requires an HTTP loopback origin")
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := *s.HTTP
	client.Timeout = 0 // the bounded per-operation context owns the longer song deadline
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return fmt.Errorf("local resource redirects refused") }
	return client.Do(req)
}

func (s *ModuleState) songReady(ctx context.Context, model models.Model) error {
	if err := validSongSnapshot(model); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := s.songHTTP(ctx, model, http.MethodGet, "/health", nil)
	if err != nil {
		return fmt.Errorf("song resource unavailable; use normal lifecycle to activate %s: %w", model.Runtime.Resource, err)
	}
	defer response.Body.Close()
	raw, err := boundedResourceBody(response.Body, 65536)
	if err != nil {
		return err
	}
	var health struct {
		Protocol    string `json:"protocol"`
		CatalogSHA  string `json:"catalog_sha256"`
		ArtifactSHA string `json:"artifact_sha256"`
		Profiles    map[string]struct {
			Available bool   `json:"available"`
			ID        string `json:"model_id"`
			Revision  string `json:"revision"`
		} `json:"profiles"`
	}
	if err = json.Unmarshal(raw, &health); err != nil {
		return err
	}
	profile, ok := health.Profiles[model.Runtime.Profile]
	if ok && !profile.Available {
		return fmt.Errorf("song profile %s is not acquired; use vrooli resource acquisition data %s --group %s, then normal resource start", model.Runtime.Profile, model.Runtime.Resource, model.Runtime.Profile)
	}
	if response.StatusCode != http.StatusOK || health.Protocol != model.Runtime.Protocol || health.CatalogSHA != model.Runtime.CatalogSHA256 || health.ArtifactSHA != model.Runtime.ArtifactSHA256 || !ok || !profile.Available || profile.ID != model.ID || profile.Revision != model.Revision {
		return fmt.Errorf("song resource is unavailable or its selected model/catalog identity mismatches")
	}
	return nil
}

func (s *ModuleState) songAdmission(ctx context.Context, payload composeJobPayload) (func(), map[string]string, error) {
	if payload.ModelSnapshot == nil || payload.ResolvedStyle == nil {
		return nil, nil, fmt.Errorf("song admission lacks an immutable model/style snapshot")
	}
	model := *payload.ModelSnapshot
	if model.Engine != payload.Provider || model.Variant != payload.ResolvedStyle.Params.Variant {
		return nil, nil, fmt.Errorf("song model snapshot does not match queued provider/style")
	}
	if err := validateProviderInput(model.Engine, *payload.ResolvedStyle, payload.composeRequest, s.PermissiveOnly); err != nil {
		return nil, nil, err
	}
	if err := s.songReady(ctx, model); err != nil {
		return nil, nil, err
	}
	return nil, map[string]string{"composition_provider": model.Engine, "capacity_ownership": "resource:" + model.Runtime.Resource}, nil
}

func validateSongReceipt(receipt songReceipt, request songRequest, raw []byte, model models.Model) error {
	if receipt.Version != 1 || receipt.Status != "completed" || !operationHex.MatchString(receipt.Operation) || receipt.RequestSHA != fmt.Sprintf("%x", sha256.Sum256(raw)) || receipt.Effective != request || receipt.CatalogSHA != model.Runtime.CatalogSHA256 || receipt.ArtifactSHA != model.Runtime.ArtifactSHA256 {
		return fmt.Errorf("song receipt input/catalog identity mismatch")
	}
	if receipt.Model.ID != model.ID || receipt.Model.Revision != model.Revision || receipt.Model.License != model.License || receipt.Model.Source != model.Runtime.SourceRevision || receipt.Worker.Seed == nil || *receipt.Worker.Seed != request.Seed || receipt.Worker.ModelID != model.ID || receipt.Worker.Revision != model.Revision || receipt.Worker.FilesVerified != model.Runtime.WeightFiles || receipt.Worker.VoiceReference || !reflect.DeepEqual(receipt.Worker.Versions, model.Runtime.Versions) {
		return fmt.Errorf("song receipt worker/model/seed identity mismatch")
	}
	sampling := receipt.Worker.Sampling
	if sampling.Guidance == nil || *sampling.Guidance != request.Guidance || (model.Runtime.MaxSteps > 0 && (sampling.Steps == nil || *sampling.Steps != request.Steps)) || (sampling.DecoderSeed != nil && *sampling.DecoderSeed != request.Seed) {
		return fmt.Errorf("song receipt actual sampling controls mismatch")
	}

	o := receipt.Output
	if !receiptHex.MatchString(o.SHA) || o.Bytes < 44 || o.Bytes > 128<<20 || o.Frames <= 0 || o.Rate < 16000 || o.Rate > 96000 || o.Channels < 1 || o.Channels > 2 || math.IsNaN(o.Duration) || math.IsInf(o.Duration, 0) || o.Duration != float64(o.Frames)/float64(o.Rate) || o.Duration > request.Duration+1 {
		return fmt.Errorf("song receipt audio geometry is invalid")
	}
	return nil
}

func validateSongAudio(audio []byte, receipt songReceipt) error {
	if int64(len(audio)) != receipt.Output.Bytes || fmt.Sprintf("%x", sha256.Sum256(audio)) != receipt.Output.SHA {
		return fmt.Errorf("song audio bytes/hash mismatch")
	}
	validated, err := core.ValidatedWAV(bytes.NewReader(audio))
	if err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, validated); err != nil {
		return err
	}
	channels, rate, align := 0, 0, 0
	for offset := 12; offset+8 <= len(audio); {
		size := int(binary.LittleEndian.Uint32(audio[offset+4 : offset+8]))
		end := offset + 8 + size
		if end > len(audio) {
			return fmt.Errorf("song WAV chunk exceeds its file")
		}
		switch string(audio[offset : offset+4]) {
		case "fmt ":
			if size < 16 {
				return fmt.Errorf("song WAV format is incomplete")
			}
			data := audio[offset+8 : end]
			channels, rate, align = int(binary.LittleEndian.Uint16(data[2:])), int(binary.LittleEndian.Uint32(data[4:])), int(binary.LittleEndian.Uint16(data[12:]))
		case "data":
			if align <= 0 || int64(size/align) != receipt.Output.Frames || channels != receipt.Output.Channels || rate != receipt.Output.Rate {
				return fmt.Errorf("song WAV geometry differs from its receipt")
			}
			return nil
		}
		offset = end + size%2
	}
	return fmt.Errorf("song WAV has no audio frames")
}

func (s *ModuleState) generateSongTake(ctx context.Context, take *core.Take, request composeRequest, compiled styles.Compiled, model models.Model) (map[string]any, error) {
	if err := s.songReady(ctx, model); err != nil {
		return nil, err
	}
	if take.Provenance.Seed < 0 || take.Provenance.Seed > math.MaxUint32 {
		return nil, fmt.Errorf("song seed outside uint32")
	}
	payload := songRequestFor(model, compiled, request, uint32(take.Provenance.Seed))
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if len(raw) > 65536 {
		return nil, fmt.Errorf("song request exceeds 64KiB")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(payload.Timeout+15)*time.Second)
	defer cancel()
	response, err := s.songHTTP(ctx, model, http.MethodPost, "/v1/compose", raw)
	if err != nil {
		return nil, fmt.Errorf("song compose failed; resource worker may be draining: %w", err)
	}
	receiptData, err := boundedResourceBody(response.Body, 512<<10)
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("song resource %s: %s", response.Status, receiptData)
	}
	var receipt songReceipt
	if err := json.Unmarshal(receiptData, &receipt); err != nil {
		return nil, err
	}
	if err := validateSongReceipt(receipt, payload, raw, model); err != nil {
		return nil, err
	}
	response, err = s.songHTTP(ctx, model, http.MethodGet, "/v1/operations/"+receipt.Operation+"/audio", nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("song audio returned %s", response.Status)
	}
	audio, err := boundedResourceBody(response.Body, receipt.Output.Bytes)
	if err != nil {
		return nil, err
	}
	if err := validateSongAudio(audio, receipt); err != nil {
		return nil, err
	}
	if s.Blobs == nil {
		return nil, fmt.Errorf("managed audio storage is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	receiptRef, audioRef := "generation-receipts/"+take.ID+".json", "out/"+take.ID+".wav"
	if err := s.Blobs.PutOwned(ctx, receiptRef, take.ID, bytes.NewReader(receiptData), "application/json", false); err != nil {
		return nil, err
	}
	if err := s.Blobs.PutOwned(ctx, audioRef, take.ID, bytes.NewReader(audio), "audio/wav", true); err != nil {
		return nil, err
	}
	take.BlobRef = audioRef
	take.Provenance.ModelID, take.Provenance.LicenseLane, take.Provenance.AppliedRung = model.ID, string(model.Licence), "isolated:"+model.Runtime.Profile
	take.Provenance.CaptionAsSent = payload.Prompt
	return map[string]any{"provider": model.Engine, "profile": model.Runtime.Profile, "receipt_ref": receiptRef,
		"receipt_sha256": fmt.Sprintf("%x", sha256.Sum256(receiptData)), "audio_sha256": receipt.Output.SHA,
		"seed": payload.Seed, "actual_duration_seconds": receipt.Output.Duration, "requested_duration_seconds": payload.Duration,
		"lyrics_sha256": lyricsHash(payload.Lyrics)}, nil
}
