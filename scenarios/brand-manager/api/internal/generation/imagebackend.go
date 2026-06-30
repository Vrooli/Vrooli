package generation

import "context"

// ImageBackend is the narrow seam the generation Service uses for ALL image
// work. It is brand-manager's compound-value boundary onto image-tools: the
// production implementation (internal/imagetools) talks to the image-tools
// scenario over HTTP — it submits the model-backed AI operation, waits once for
// the durable job, downloads the result blob, and (for deterministic icon
// derivation) calls image-tools' synchronous ops edge. Service unit tests
// substitute a fake so they never reach a real image-tools.
//
// brand-manager owns brand semantics + prompt recipes + asset/version policy;
// image-tools owns model/backend/provider selection and durable jobs. This seam
// is deliberately small: it exposes only the operations brand-manager needs, in
// brand-manager's vocabulary (no image-tools job ids, no operation enum bag).
type ImageBackend interface {
	// Status reports image-tools reachability plus per-operation readiness so a
	// UI/CLI can warn before spending a generation. It never errors — an
	// unreachable backend is reported as Available=false with a Detail.
	Status(ctx context.Context) ImageBackendStatus

	// Generate runs text_to_image from a brand-aware prompt.
	Generate(ctx context.Context, req ImageGenerateRequest) (ImageOutput, error)

	// Edit runs edit_instruct on the source image with a natural-language
	// instruction.
	Edit(ctx context.Context, req ImageEditRequest) (ImageOutput, error)

	// RemoveBackground runs background_removal on the source image, producing a
	// transparent cutout.
	RemoveBackground(ctx context.Context, req ImageRemoveBackgroundRequest) (ImageOutput, error)

	// Resize runs the deterministic resize op (aspect-preserving fit) to fit
	// within width×height. Synchronous — no model, no job lifecycle.
	Resize(ctx context.Context, src []byte, width, height int) (ImageOutput, error)

	// Flatten composites the source onto a solid-background canvas of width×height
	// (background is a "#rrggbb" hex color), producing an opaque image. Used for
	// Apple-touch / maskable launcher icons where transparency renders poorly.
	Flatten(ctx context.Context, src []byte, width, height int, background string) (ImageOutput, error)
}

// ImageGenerateRequest is a text_to_image request in brand-manager's vocabulary.
type ImageGenerateRequest struct {
	Prompt         string
	NegativePrompt string
	Width          int
	Height         int
	ModelOverride  string
	AllowBYOK      bool
	QualityPolicy  string
	FallbackPolicy string
	Priority       string
	AllowReclaim   bool
	Seed           int64
}

// ImageEditRequest is an edit_instruct request: edit Source by Instruction.
type ImageEditRequest struct {
	Source         []byte
	Instruction    string
	ModelOverride  string
	AllowBYOK      bool
	QualityPolicy  string
	FallbackPolicy string
	Priority       string
	AllowReclaim   bool
	Seed           int64
}

// ImageRemoveBackgroundRequest is a background_removal request over Source.
type ImageRemoveBackgroundRequest struct {
	Source        []byte
	ModelOverride string
	AllowBYOK     bool
}

// ImageOutput is the bytes + provenance image-tools returns for any operation.
type ImageOutput struct {
	Data     []byte
	MimeType string
	ModelID  string   // empty for deterministic ops
	Tier     string   // local-gpu | local-cpu | byok-cloud | deterministic
	Warnings []string // non-fatal selection cautions
}

// ImageBackendStatus is the readiness projection Status reports.
type ImageBackendStatus struct {
	Available  bool
	Detail     string
	Operations []ImageOperationStatus
}

// ImageOperationStatus is one operation's readiness on image-tools.
type ImageOperationStatus struct {
	Operation string // generate | edit | remove_background
	Ready     bool
	ModelID   string
	Tier      string
	Hint      string
	Warnings  []string
}

// ErrImageBackendUnavailable is returned when image-tools cannot be reached.
// Handlers translate it into a Connect Unavailable response.
type ErrImageBackendUnavailable struct {
	Detail string
}

func (e ErrImageBackendUnavailable) Error() string {
	if e.Detail == "" {
		return "image-tools is unavailable"
	}
	return "image-tools is unavailable: " + e.Detail
}

// ErrImageBackendNotReady is returned when image-tools is reachable but cannot
// run the operation: the model/backend is not installed, or an opt-in BYOK cloud
// key is missing. Hint carries the actionable remediation. Handlers translate it
// into a Connect FailedPrecondition response.
type ErrImageBackendNotReady struct {
	Operation string
	Hint      string
}

func (e ErrImageBackendNotReady) Error() string {
	msg := "image-tools cannot run " + e.Operation
	if e.Hint != "" {
		msg += ": " + e.Hint
	}
	return msg
}

// ErrImageJobFailed is returned when the image-tools durable job failed or the
// result blob could not be downloaded. Handlers translate it into a Connect
// Internal response (the failure is on image-tools' side, not the caller's).
type ErrImageJobFailed struct {
	Operation string
	Detail    string
}

func (e ErrImageJobFailed) Error() string {
	return "image-tools " + e.Operation + " failed: " + e.Detail
}
