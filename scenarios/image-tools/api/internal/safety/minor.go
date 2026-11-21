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
	"strings"
	"time"
)

// MinorSafetyEnv names the operator-supplied minor-safety reviewer. There is no
// bundled minor-age classifier: the shipped NSFW model answers "sexual or not"
// and nothing about age. Rather than ship a detector that cannot answer the
// question, the boundary is a real, configurable seam with an explicit contract,
// and a deployment that serves other people must supply one (see
// docs/internal/SECURITY.md).
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
}

type minorReviewImage struct {
	MediaType string `json:"media_type"`
	Base64    string `json:"base64"`
}

func (r *CommandReviewer) Review(ctx context.Context, images [][]byte) (MinorVerdict, error) {
	if len(images) == 0 {
		return MinorVerdict{}, nil
	}
	request := minorReviewRequest{Images: make([]minorReviewImage, 0, len(images))}
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

	reviewCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	cmd := exec.CommandContext(reviewCtx, r.program, r.args...)
	cmd.Stdin = bytes.NewReader(raw)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(reviewCtx.Err(), context.DeadlineExceeded) {
			return MinorVerdict{}, fmt.Errorf("minor-safety reviewer timed out after %s", r.timeout)
		}
		return MinorVerdict{}, fmt.Errorf("minor-safety reviewer failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
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
