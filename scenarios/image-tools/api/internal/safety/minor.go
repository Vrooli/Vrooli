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

// MinorVerdict is the reviewer's answer about the supplied images. The legacy
// Minor/Sexual pair remains for operator-supplied reviewers that report only the
// aggregate flags; the bundled reviewer also reports a structured action, a
// reason code, per-image findings, thresholds and the model ids it read.
type MinorVerdict struct {
	// Action is allow, flag or refuse. Normalize derives it from Minor/Sexual for
	// a reviewer that reports only the aggregate pair.
	Action Action `json:"action,omitempty"`
	// ReasonCode is the machine-readable why (first refused image, or the
	// aggregate flag reason).
	ReasonCode ReasonCode `json:"reason_code,omitempty"`
	// ReasonCodes lists every distinct reason across the reviewed images.
	ReasonCodes []ReasonCode `json:"reason_codes,omitempty"`
	// Minor reports that at least one image depicts a person who is, or appears
	// to be, a minor in sexual content (the co-occurrence verdict).
	Minor bool `json:"minor"`
	// Sexual reports that at least one image is sexual content.
	Sexual bool `json:"sexual"`
	// Detail is a short, human-readable note (never the image itself).
	Detail string `json:"detail,omitempty"`
	// Images carries the reviewer's per-image measurements when it reports them
	// (the bundled reviewer does; an operator-supplied one need not).
	Images []MinorImageFinding `json:"images,omitempty"`
	// Thresholds are the values the reviewer applied (0..1), keyed by the
	// classifier id or "age"/"face.min_px". Recorded on the job as evidence.
	Thresholds map[string]float64 `json:"thresholds,omitempty"`
	// Models are the model ids the reviewer read for this request.
	Models []string `json:"models,omitempty"`
	// Strict reports that the strict classifier thresholds were applied (an
	// NSFW-capable model produced the output). Replay/calibration uses it to
	// know which bar produced the verdict.
	Strict bool `json:"strict,omitempty"`
	// ProfileName names the resolved declarative profile that produced the
	// verdict (the bundled reviewer reports it).
	ProfileName string `json:"profile_name,omitempty"`
	// ReviewerVersion identifies the profile/weight revision that produced the
	// verdict, so a job record is reproducible after a calibration change.
	ReviewerVersion string `json:"reviewer_version,omitempty"`
}

// Normalize fills Action and ReasonCode for a reviewer that reports only the
// aggregate Minor/Sexual pair. It never weakens a refusal.
func (v *MinorVerdict) Normalize() {
	if v.Action == "" {
		switch {
		case v.Minor && v.Sexual:
			v.Action = ActionRefuse
		case v.Sexual:
			v.Action = ActionFlag
		default:
			v.Action = ActionAllow
		}
	}
	if v.ReasonCode == "" {
		switch v.Action {
		case ActionRefuse:
			v.ReasonCode = ReasonSexualMinorFace
		case ActionFlag:
			v.ReasonCode = ReasonSexualAdult
		default:
			v.ReasonCode = ReasonClean
		}
	}
	if len(v.ReasonCodes) == 0 {
		v.ReasonCodes = []ReasonCode{v.ReasonCode}
	}
}

// Refuses reports whether the verdict blocks the request.
func (v MinorVerdict) Refuses() bool {
	if v.Action != "" {
		return v.Action == ActionRefuse
	}
	return v.Minor && v.Sexual
}

// MinorImageFinding is one image's measurements. The scenario records the NSFW score
// and sensitivity on the job so the UI can blur, and the numbers make a pass auditable.
type MinorImageFinding struct {
	NSFWScore float64 `json:"nsfw_score"`
	// RecallScore is the high-recall classifier's score (the bundled reviewer's
	// second classifier), when one ran.
	RecallScore float64 `json:"recall_score,omitempty"`
	// Scores is every classifier's score keyed by model id. It lets an exported
	// corpus replay a candidate threshold without re-running the models.
	Scores     map[string]float64 `json:"nsfw_scores,omitempty"`
	Sexual     bool               `json:"sexual"`
	Action     Action             `json:"action,omitempty"`
	ReasonCode ReasonCode         `json:"reason_code,omitempty"`
	FacesFound int                `json:"faces_found"`
	// AssessedFaces is how many found faces were large enough for the age model.
	AssessedFaces int     `json:"assessed_faces,omitempty"`
	MaxPUnder20   float64 `json:"max_p_under_20"`
	MinorFace     bool    `json:"minor_face"`
	// Faces carries the per-face measurements (size and P(age < 20)). A corpus
	// can sweep the age threshold from these without retaining the image.
	Faces []MinorFaceFinding `json:"faces,omitempty"`
	// Unassessable marks a sexual image whose subject's age the face stage could not
	// establish (no face found, or none large enough to read). It is the only refusal
	// a second, face-independent signal may clear (see SecondOpinionReviewer).
	Unassessable bool   `json:"unassessable"`
	Refuse       bool   `json:"refuse"`
	Detail       string `json:"detail,omitempty"`
}

// MinorFaceFinding is one detected face's measurements. It intentionally omits
// the age-bin probabilities: the corpus only needs the number that decides the
// minor-face verdict.
type MinorFaceFinding struct {
	Box           [4]float64 `json:"box,omitempty"`
	DetectorScore float64    `json:"detector_score,omitempty"`
	SizePx        float64    `json:"size_px"`
	PUnder20      float64    `json:"p_under_20,omitempty"`
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
	profile  ReviewProfile
	health   healthTracker
}

const minorReviewMaxBytes = 64 << 20

// NewCommandReviewer parses a command line into a reviewer. It uses the shipped
// local review profile unless the caller supplies one; a custom reviewer may
// ignore the profile fields it does not understand.
func NewCommandReviewer(command string) (*CommandReviewer, error) {
	return NewCommandReviewerWithProfile(command, DefaultReviewProfile(TierLocal))
}

// NewCommandReviewerWithProfile parses a command line and attaches the profile
// sent in the review request.
func NewCommandReviewerWithProfile(command string, profile ReviewProfile) (*CommandReviewer, error) {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return nil, fmt.Errorf("%s is empty", MinorSafetyEnv)
	}
	r := &CommandReviewer{program: fields[0], args: fields[1:], timeout: 60 * time.Second, maxBytes: minorReviewMaxBytes, profile: profile}
	r.health.setConfigured("command reviewer")
	return r, nil
}

// Health reports the reviewer's last observed state and counters.
func (r *CommandReviewer) Health() ReviewHealth {
	if r == nil {
		return ReviewHealth{}
	}
	return r.health.snapshot()
}

// Available reports whether the reviewer was configured successfully. A
// runtime failure updates Health but does not permanently mark it unavailable.
func (r *CommandReviewer) Available() bool { return r != nil }

// FromEnv builds the configured reviewer, or returns (nil, nil) when the
// deployment has not configured one. A malformed value is an error rather than
// a silent disable.
func MinorReviewerFromEnv() (MinorReviewer, error) {
	return MinorReviewerFromEnvWithProfile(DefaultReviewProfile(TierLocal))
}

// MinorReviewerFromEnvWithProfile is MinorReviewerFromEnv with the profile the
// deployment resolved.
func MinorReviewerFromEnvWithProfile(profile ReviewProfile) (MinorReviewer, error) {
	raw := strings.TrimSpace(os.Getenv(MinorSafetyEnv))
	if raw == "" {
		return nil, nil
	}
	reviewer, err := NewCommandReviewerWithProfile(raw, profile)
	if err != nil {
		return nil, err
	}
	return reviewer, nil
}

type minorReviewRequest struct {
	Images []minorReviewImage `json:"images"`
	// Profile is the declarative reviewer configuration. The bundled reviewer
	// reads it; an operator-supplied one may ignore it.
	Profile ReviewProfile `json:"profile,omitempty"`
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
	request := minorReviewRequest{
		Images: make([]minorReviewImage, 0, len(images)), Profile: r.profile,
		Strict: opts.strict, AssumeSexual: opts.assumeSexual,
	}
	var total int64
	for _, image := range images {
		total += int64(len(image))
		if total > r.maxBytes {
			err := fmt.Errorf("minor-safety review input exceeds %d bytes", r.maxBytes)
			r.health.record(err, false, time.Now())
			return MinorVerdict{}, &ReviewError{Kind: ReasonReviewerInvalid, Cause: err}
		}
		request.Images = append(request.Images, minorReviewImage{
			MediaType: detectMediaType(image),
			Base64:    base64.StdEncoding.EncodeToString(image),
		})
	}
	raw, err := json.Marshal(request)
	if err != nil {
		err = fmt.Errorf("encode minor-safety review: %w", err)
		r.health.record(err, false, time.Now())
		return MinorVerdict{}, &ReviewError{Kind: ReasonReviewerInvalid, Cause: err}
	}

	out, stderr, err := runJSONProgram(ctx, r.program, r.args, r.env, r.timeout, raw)
	if err != nil {
		var timeout *programTimeout
		var reviewErr error
		switch {
		case errors.As(err, &timeout):
			reviewErr = fmt.Errorf("minor-safety reviewer timed out after %s", r.timeout)
		case len(missingWeightsFromExit(err, stderr)) > 0:
			ids := missingWeightsFromExit(err, stderr)
			missing := &MissingWeightsError{IDs: ids}
			r.health.record(missing, false, time.Now())
			return MinorVerdict{}, missing
		default:
			reviewErr = fmt.Errorf("minor-safety reviewer failed: %w (%s)", err, strings.TrimSpace(stderr))
		}
		r.health.record(reviewErr, false, time.Now())
		return MinorVerdict{}, &ReviewError{Kind: ReasonReviewerUnavailable, Cause: reviewErr}
	}
	var verdict MinorVerdict
	if err := json.Unmarshal(out, &verdict); err != nil {
		parseErr := fmt.Errorf("minor-safety reviewer returned invalid JSON: %w", err)
		r.health.record(parseErr, false, time.Now())
		return MinorVerdict{}, &ReviewError{Kind: ReasonReviewerInvalid, Cause: parseErr}
	}
	verdict.Normalize()
	r.health.record(nil, verdict.Refuses(), time.Now())
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
	return NewBundledReviewerWithProfile(cfg, DefaultReviewProfile(TierLocal))
}

// NewBundledReviewerWithProfile builds the bundled reviewer with the deployment's
// declarative profile. The profile is sent in every request, so thresholds and
// model ids are data rather than code.
func NewBundledReviewerWithProfile(cfg BundledReviewerConfig, profile ReviewProfile) (*CommandReviewer, error) {
	env, err := bundledSidecarEnv("bundled minor-safety reviewer", cfg)
	if err != nil {
		return nil, err
	}
	r := &CommandReviewer{
		program:  cfg.Python,
		args:     []string{"-m", "image_tools_sidecar.minor_review", "--models-root", cfg.ModelsRoot},
		env:      env,
		timeout:  bundledReviewerTimeout,
		maxBytes: minorReviewMaxBytes,
		profile:  profile,
	}
	r.health.setConfigured("bundled reviewer")
	return r, nil
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
	return MinorVerdict{}, &ReviewError{Kind: ReasonReviewerUnavailable, Cause: fmt.Errorf("minor-safety reviewer unavailable: %w", r.cause)}
}

// Available reports that this reviewer cannot run; discovery must not claim it.
func (r unavailableReviewer) Available() bool { return false }

// Health reports the construction failure.
func (r unavailableReviewer) Health() ReviewHealth {
	return ReviewHealth{Configured: true, Available: false, Detail: firstLine(r.cause.Error())}
}
