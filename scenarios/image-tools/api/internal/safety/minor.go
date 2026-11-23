package safety

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// MinorSafetyEnv names an operator-supplied minor-safety reviewer that REPLACES the
// bundled one. The boundary no longer depends on it: the scenario ships a local
// reviewer (see NewBundledReviewer: NSFW classifier + face detector + age model) and
// uses it whenever this is unset, on every deployment tier. The override exists for a
// deployment that has its own, stronger detector (see docs/internal/SECURITY.md).
const MinorSafetyEnv = "IMAGE_TOOLS_MINOR_SAFETY_CMD"

// MinorVerdict is the reviewer's answer about the supplied images.
type MinorVerdict struct {
	// Minor reports that at least one image depicts a person who is, or appears
	// to be, a minor.
	Minor bool `json:"minor"`
	// Sexual reports that at least one image is sexual content.
	Sexual bool `json:"sexual"`
	// Detail is a short, human-readable note (never the image itself).
	Detail string `json:"detail,omitempty"`
	// Images carries the reviewer's per-image measurements when it reports them
	// (the bundled reviewer does; an operator-supplied one need not).
	Images []MinorImageFinding `json:"images,omitempty"`
}

// MinorImageFinding is one image's measurements. The scenario records the NSFW score
// and sensitivity on the job so the UI can blur, and the numbers make a pass auditable.
type MinorImageFinding struct {
	NSFWScore   float64 `json:"nsfw_score"`
	Sexual      bool    `json:"sexual"`
	FacesFound  int     `json:"faces_found"`
	MaxPUnder20 float64 `json:"max_p_under_20"`
	MinorFace   bool    `json:"minor_face"`
	Refuse      bool    `json:"refuse"`
}

// Sensitive reports whether any reviewed image was classified sexual, and the highest
// NSFW score seen. A reviewer that reports no per-image findings falls back to its
// aggregate sexual flag.
func (v MinorVerdict) Sensitive() (bool, float64) {
	if len(v.Images) == 0 {
		return v.Sexual, 0
	}
	var top float64
	var sexual bool
	for _, img := range v.Images {
		sexual = sexual || img.Sexual
		top = max(top, img.NSFWScore)
	}
	return sexual, top
}

type reviewOption struct{ strict, assumeSexual bool }

type reviewOptionKey struct{}

// WithStrictReview marks the images under review as produced by a model able to make
// explicit images. The reviewer then asks a lower sexual-content bar, because its
// classifier is weaker on synthetic images than on photographs.
func WithStrictReview(ctx context.Context) context.Context {
	o, _ := ctx.Value(reviewOptionKey{}).(reviewOption)
	o.strict = true
	return context.WithValue(ctx, reviewOptionKey{}, o)
}

// WithAssumeSexual makes the reviewer judge the age finding as if every image were
// sexual: a probe that shows what the boundary would do, without needing sexual
// content to exist. It can only make a review stricter.
func WithAssumeSexual(ctx context.Context) context.Context {
	o, _ := ctx.Value(reviewOptionKey{}).(reviewOption)
	o.assumeSexual = true
	return context.WithValue(ctx, reviewOptionKey{}, o)
}

// MinorReviewer classifies images for the one non-configurable refusal:
// sexual content involving minors. It is deliberately not a general content
// filter and never removes a capability on its own.
type MinorReviewer interface {
	Review(ctx context.Context, images [][]byte) (MinorVerdict, error)
}

// CommandReviewer runs an operator-supplied program that implements the review
// contract. The command is parsed without a shell (whitespace splits argv), so a
// configured value cannot become arbitrary shell evaluation.
//
// Contract: the program receives one JSON object on stdin,
//
//	{"images":[{"media_type":"image/png","base64":"..."}]}
//
// and must write one JSON object to stdout: {"minor":bool,"sexual":bool,"detail":"..."}.
// A non-zero exit, a timeout, or malformed output is an error; the gate treats
// that as "review unavailable" and fails closed on a deployment that requires it.
type CommandReviewer struct {
	program  string
	args     []string
	env      []string
	timeout  time.Duration
	maxBytes int64
}

const minorReviewMaxBytes = 64 << 20

// NewCommandReviewer parses a command line into a reviewer.
func NewCommandReviewer(command string) (*CommandReviewer, error) {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return nil, fmt.Errorf("%s is empty", MinorSafetyEnv)
	}
	return &CommandReviewer{program: fields[0], args: fields[1:], timeout: 60 * time.Second, maxBytes: minorReviewMaxBytes}, nil
}

// FromEnv builds the configured reviewer, or returns (nil, nil) when the
// deployment has not configured one. A malformed value is an error rather than
// a silent disable.
func MinorReviewerFromEnv() (MinorReviewer, error) {
	raw := strings.TrimSpace(os.Getenv(MinorSafetyEnv))
	if raw == "" {
		return nil, nil
	}
	reviewer, err := NewCommandReviewer(raw)
	if err != nil {
		return nil, err
	}
	return reviewer, nil
}

type minorReviewRequest struct {
	Images []minorReviewImage `json:"images"`
	// Strict and AssumeSexual are optional hints a reviewer may ignore; both can
	// only make the review stricter (see WithStrictReview / WithAssumeSexual).
	Strict       bool `json:"strict,omitempty"`
	AssumeSexual bool `json:"assume_sexual,omitempty"`
}

type minorReviewImage struct {
	MediaType string `json:"media_type"`
	Base64    string `json:"base64"`
}

// programTimeout reports that a sidecar program ran past its deadline.
type programTimeout struct{}

func (*programTimeout) Error() string { return "program timed out" }

// runJSONProgram runs one local program with a JSON request on stdin and returns
// its stdout and stderr. The reviewer and the face counter share it, so both have
// the same deadline, environment and missing-weights behaviour.
func runJSONProgram(ctx context.Context, program string, args, env []string, timeout time.Duration, stdin []byte) ([]byte, string, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, program, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil && errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return nil, stderr.String(), &programTimeout{}
	}
	return out, stderr.String(), err
}

// missingWeightsFromExit names the model ids a program asked the host to acquire:
// it exited with the missing-weights code and listed them on stderr.
func missingWeightsFromExit(err error, stderr string) []string {
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == reviewerExitMissingWeights {
		return missingWeights(stderr)
	}
	return nil
}

func (r *CommandReviewer) Review(ctx context.Context, images [][]byte) (MinorVerdict, error) {
	if len(images) == 0 {
		return MinorVerdict{}, nil
	}
	opts, _ := ctx.Value(reviewOptionKey{}).(reviewOption)
	request := minorReviewRequest{Images: make([]minorReviewImage, 0, len(images)), Strict: opts.strict, AssumeSexual: opts.assumeSexual}
	var total int64
	for _, image := range images {
		total += int64(len(image))
		if total > r.maxBytes {
			return MinorVerdict{}, fmt.Errorf("minor-safety review input exceeds %d bytes", r.maxBytes)
		}
		request.Images = append(request.Images, minorReviewImage{
			MediaType: detectMediaType(image),
			Base64:    base64.StdEncoding.EncodeToString(image),
		})
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return MinorVerdict{}, fmt.Errorf("encode minor-safety review: %w", err)
	}

	out, stderr, err := runJSONProgram(ctx, r.program, r.args, r.env, r.timeout, raw)
	if err != nil {
		var timeout *programTimeout
		if errors.As(err, &timeout) {
			return MinorVerdict{}, fmt.Errorf("minor-safety reviewer timed out after %s", r.timeout)
		}
		if ids := missingWeightsFromExit(err, stderr); len(ids) > 0 {
			return MinorVerdict{}, &MissingWeightsError{IDs: ids}
		}
		return MinorVerdict{}, fmt.Errorf("minor-safety reviewer failed: %w (%s)", err, strings.TrimSpace(stderr))
	}
	var verdict MinorVerdict
	if err := json.Unmarshal(out, &verdict); err != nil {
		return MinorVerdict{}, fmt.Errorf("minor-safety reviewer returned invalid JSON: %w", err)
	}
	return verdict, nil
}

// detectMediaType sniffs enough of the first bytes to label the payload without
// importing an image decoder into the policy package.
func detectMediaType(data []byte) string {
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp"
	case len(data) >= 6 && string(data[:6]) == "GIF89a":
		return "image/gif"
	case len(data) >= 2 && string(data[:2]) == "BM":
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}

// BundledReviewerConfig locates everything the bundled reviewer runs from.
type BundledReviewerConfig struct {
	// Python is the scenario's private venv interpreter. The reviewer never falls
	// back to a host Python: its weights and wheels are governed inputs.
	Python string
	// SidecarPath is the directory holding the materialized image_tools_sidecar package.
	SidecarPath string
	// ModelsRoot is the data root that holds models/<id>/.
	ModelsRoot string
	// Libraries are the venv's NVIDIA runtime directories; onnxruntime-gpu and torch
	// load them at import time even though the reviewer only computes on the CPU.
	Libraries []string
}

// BundledReviewerModels are the registry ids the bundled reviewer reads for EVERY
// output (the sexual classifier). The engine acquires and holds them
// (Deps.ReviewModels) so the reviewer can always run.
var BundledReviewerModels = []string{"adamcodd-vit-nsfw"}

// BundledReviewerEscalationModels are read only for an output the classifier calls
// sexual (face finder and age model). The engine acquires them up front when the
// producing model is NSFW-capable and on demand otherwise, so ordinary work never
// depends on them (Deps.ReviewEscalationModels).
var BundledReviewerEscalationModels = []string{"yunet", "fairface-vit-age"}

// reviewerExitMissingWeights is minor_review.EXIT_MISSING_WEIGHTS.
const reviewerExitMissingWeights = 4

// MissingWeightsError means the reviewer needs model weights that are not installed.
// It is not a verdict: the caller acquires IDs through managed acquisition and
// retries, and fails closed if that cannot be done.
type MissingWeightsError struct{ IDs []string }

func (e *MissingWeightsError) Error() string {
	return "minor-safety reviewer needs model weights that are not installed: " + strings.Join(e.IDs, ", ")
}

var missingWeightsPattern = regexp.MustCompile(`missing-weights=([A-Za-z0-9._,-]+)`)

// missingWeights reads the model ids the reviewer named on stderr, if any.
func missingWeights(stderr string) []string {
	m := missingWeightsPattern.FindStringSubmatch(stderr)
	if m == nil {
		return nil
	}
	return strings.Split(m[1], ",")
}

const bundledReviewerTimeout = 90 * time.Second

// NewBundledReviewer builds the reviewer the scenario ships: a local program (the
// sidecar's minor_review module) that classifies sexual content, finds faces and
// estimates their age, all on the CPU so it never takes VRAM from a generation. It
// speaks the same stdin/stdout contract as an operator-supplied reviewer, and the
// same fail-closed rules apply: a non-zero exit or a timeout is "review unavailable".
func NewBundledReviewer(cfg BundledReviewerConfig) (*CommandReviewer, error) {
	env, err := bundledSidecarEnv("bundled minor-safety reviewer", cfg)
	if err != nil {
		return nil, err
	}
	return &CommandReviewer{
		program:  cfg.Python,
		args:     []string{"-m", "image_tools_sidecar.minor_review", "--models-root", cfg.ModelsRoot},
		env:      env,
		timeout:  bundledReviewerTimeout,
		maxBytes: minorReviewMaxBytes,
	}, nil
}

// bundledSidecarEnv validates the sidecar locations and builds the CPU-only
// environment every bundled safety program (the reviewer, the face counter) runs
// in, so neither can take VRAM from a generation.
func bundledSidecarEnv(who string, cfg BundledReviewerConfig) ([]string, error) {
	for name, value := range map[string]string{"python": cfg.Python, "sidecar path": cfg.SidecarPath, "models root": cfg.ModelsRoot} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%s: %s is required", who, name)
		}
	}
	env := []string{
		"PYTHONPATH=" + cfg.SidecarPath,
		"CUDA_VISIBLE_DEVICES=",
		"IMAGE_TOOLS_DEVICE=cpu",
		"IMAGE_TOOLS_CPU_THREADS=4",
		"OMP_NUM_THREADS=4",
		"MKL_NUM_THREADS=4",
	}
	if len(cfg.Libraries) > 0 {
		libs := strings.Join(cfg.Libraries, string(os.PathListSeparator))
		if current := os.Getenv("LD_LIBRARY_PATH"); current != "" {
			libs += string(os.PathListSeparator) + current
		}
		env = append(env, "LD_LIBRARY_PATH="+libs)
	}
	return env, nil
}

// UnavailableReviewer is the reviewer used when the configured one cannot even be
// constructed. It fails every review with the cause, so the boundary stays closed
// instead of silently turning off.
func UnavailableReviewer(cause error) MinorReviewer { return unavailableReviewer{cause: cause} }

type unavailableReviewer struct{ cause error }

func (r unavailableReviewer) Review(context.Context, [][]byte) (MinorVerdict, error) {
	return MinorVerdict{}, fmt.Errorf("minor-safety reviewer unavailable: %w", r.cause)
}
