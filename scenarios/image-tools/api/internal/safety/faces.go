package safety

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// FaceCountFunc counts the people one request's input images show. The submit edge
// builds it once per request (memoised), so the engine's routing and the gate's
// decision share a single detector run.
type FaceCountFunc = func(ctx context.Context) (int, error)

// FaceCounter counts faces in images. It answers the consent gate's one question
// for low-weight operations: does this image show anyone.
type FaceCounter interface {
	CountFaces(ctx context.Context, images [][]byte) (int, error)
}

// ErrNoFaceCounter means the deployment has no detector, so "no people" cannot be
// shown and the consent requirement stands.
var ErrNoFaceCounter = errors.New("no face detector is configured")

// Consent details: why a request did or did not need an affirmation, in the form
// the job's meta records as consent_basis.
const (
	// ConsentDetailLocal: the route stays on this machine.
	ConsentDetailLocal = "not_required_local"
	// ConsentDetailControlOff: the deployment turned the route-locality control off.
	ConsentDetailControlOff = "not_required_control_off"
	// ConsentDetailDetectorUnavailable: the people check could not run, so the
	// requirement stands (fail closed).
	ConsentDetailDetectorUnavailable = "detector_unavailable"
)

// ConsentDetailFaces is the detail for a people check that ran: faces_detected=N.
func ConsentDetailFaces(n int) string { return fmt.Sprintf("faces_detected=%d", n) }

// WithFaces settles a face-gated requirement with the outcome of the people check.
// Faces found keep it required; no faces clear it; a check that could not run keeps
// it required, because "nobody is shown" is something the gate must establish, not
// assume. A requirement that is not face-gated is returned unchanged.
func (r ConsentRequirement) WithFaces(count int, err error) ConsentRequirement {
	if !r.FaceGated {
		return r
	}
	r.FaceGated = false
	switch {
	case err != nil:
		r.Detail = ConsentDetailDetectorUnavailable
		r.Reason += " (the check for people in the image could not run: " + err.Error() + ")"
	case count > 0:
		r.Detail = ConsentDetailFaces(count)
	default:
		r.Required, r.Basis, r.Reason = false, "", ""
		r.Detail = ConsentDetailFaces(0)
	}
	return r
}

// CountFaces runs the deployment's face counter. A deployment without one fails
// closed with ErrNoFaceCounter.
func (g *Gate) CountFaces(ctx context.Context, images [][]byte) (int, error) {
	if g.faces == nil {
		return 0, ErrNoFaceCounter
	}
	return g.faces.CountFaces(ctx, images)
}

// SetFaceCounter attaches the detector low-weight consent decisions use.
func (g *Gate) SetFaceCounter(c FaceCounter) { g.faces = c }

// ConsentForInput is ConsentFor plus the people check: a low-weight request whose
// route leaves this machine is asked for consent only when its input shows people.
// faces is nil when the caller has no image to look at, which keeps the requirement.
func (g *Gate) ConsentForInput(ctx context.Context, w Weight, op, tier string, faces FaceCountFunc) ConsentRequirement {
	need := g.policy.ConsentFor(w, op, tier)
	if !need.FaceGated {
		return need
	}
	if faces == nil {
		return need.WithFaces(0, ErrNoFaceCounter)
	}
	count, err := faces(ctx)
	return need.WithFaces(count, err)
}

// EvaluateRouteInput decides a submit once its route is known, looking at the
// input for people when the consent rule depends on it. The decision carries the
// consent_basis to record on the job.
func (g *Gate) EvaluateRouteInput(ctx context.Context, w Weight, op, tier string, consentAffirmed bool, faces FaceCountFunc) Decision {
	return g.policy.decideRoute(g.ConsentForInput(ctx, w, op, tier, faces), w, op, consentAffirmed)
}

// CommandFaceCounter runs a local program that implements the face-count contract.
//
// Contract: the program receives {"images":[{"media_type":"image/png","base64":"..."}]}
// on stdin and writes {"counts":[<faces in each image>]} to stdout. A non-zero exit,
// a timeout or malformed output is an error, which the gate treats as "could not
// check" and answers by keeping the consent requirement.
type CommandFaceCounter struct {
	program  string
	args     []string
	env      []string
	timeout  time.Duration
	maxBytes int64
}

const faceCountTimeout = 30 * time.Second

// NewBundledFaceCounter builds the counter the scenario ships: the sidecar's
// face_count module (YuNet on the CPU, a fraction of a second per image).
func NewBundledFaceCounter(cfg BundledReviewerConfig) (*CommandFaceCounter, error) {
	env, err := bundledSidecarEnv("bundled face counter", cfg)
	if err != nil {
		return nil, err
	}
	return &CommandFaceCounter{
		program:  cfg.Python,
		args:     []string{"-m", "image_tools_sidecar.face_count", "--models-root", cfg.ModelsRoot},
		env:      env,
		timeout:  faceCountTimeout,
		maxBytes: minorReviewMaxBytes,
	}, nil
}

type faceCountResponse struct {
	Counts []int `json:"counts"`
}

// CountFaces returns the total number of faces across images.
func (c *CommandFaceCounter) CountFaces(ctx context.Context, images [][]byte) (int, error) {
	if len(images) == 0 {
		return 0, nil
	}
	request := minorReviewRequest{Images: make([]minorReviewImage, 0, len(images))}
	var total int64
	for _, image := range images {
		total += int64(len(image))
		if total > c.maxBytes {
			return 0, fmt.Errorf("face count input exceeds %d bytes", c.maxBytes)
		}
		request.Images = append(request.Images, minorReviewImage{MediaType: detectMediaType(image), Base64: base64.StdEncoding.EncodeToString(image)})
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return 0, fmt.Errorf("encode face count request: %w", err)
	}
	out, stderr, err := runJSONProgram(ctx, c.program, c.args, c.env, c.timeout, raw)
	if err != nil {
		var timeout *programTimeout
		if errors.As(err, &timeout) {
			return 0, fmt.Errorf("face counter timed out after %s", c.timeout)
		}
		if ids := missingWeightsFromExit(err, stderr); len(ids) > 0 {
			return 0, &MissingWeightsError{IDs: ids}
		}
		return 0, fmt.Errorf("face counter failed: %w (%s)", err, strings.TrimSpace(stderr))
	}
	var resp faceCountResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, fmt.Errorf("face counter returned invalid JSON: %w", err)
	}
	if len(resp.Counts) != len(images) {
		return 0, fmt.Errorf("face counter answered for %d of %d images", len(resp.Counts), len(images))
	}
	sum := 0
	for _, n := range resp.Counts {
		if n < 0 {
			return 0, fmt.Errorf("face counter returned a negative count")
		}
		sum += n
	}
	return sum, nil
}
