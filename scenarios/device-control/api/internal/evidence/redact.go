package evidence

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Policy is intentionally explicit because a producer must know what it
// redacts before a capture becomes an EvidenceRef.
type Policy struct {
	PasswordFields      bool `json:"password_fields"`
	OneTimeCodes        bool `json:"one_time_codes"`
	CredentialPatterns  bool `json:"credential_patterns"`
	NotificationContent bool `json:"notification_content"`
}

var (
	DefaultPolicy     = Policy{PasswordFields: true, OneTimeCodes: true, CredentialPatterns: true, NotificationContent: true}
	credentialPattern = regexp.MustCompile(`(?i)(authorization|bearer|api[_ -]?key|secret|password)\s*[:=]\s*[^\s]+`)
	otpPattern        = regexp.MustCompile(`\b\d{6}\b`)
)

func RedactText(in string, p Policy) string {
	out := in
	if p.CredentialPatterns {
		out = credentialPattern.ReplaceAllString(out, "$1=[REDACTED]")
	}
	if p.OneTimeCodes {
		out = otpPattern.ReplaceAllString(out, "[REDACTED-OTP]")
	}
	return out
}

type Result struct {
	Bytes     []byte
	Verified  bool
	Requested bool `json:"-"`
	Regions   int
	Policy    Policy
	Rules     []string `json:"rules,omitempty"`
	OptedOut  bool     `json:"opted_out,omitempty"`
}

const statusBarMaskDivisor = 12

// RedactCapture applies the configured producer-side transform to an encoded
// screen capture. Verification records that the transform completed; it does
// not claim that source content was semantically classified.
func RedactCapture(raw []byte, mediaType string, p Policy, allowUnredacted bool, actor string) (Result, error) {
	return RedactCaptureWithRegions(raw, mediaType, p, allowUnredacted, actor, nil)
}

func RedactCaptureWithRegions(raw []byte, mediaType string, p Policy, allowUnredacted bool, actor string, sensitive []Region) (Result, error) {
	if allowUnredacted {
		if err := ValidateOptOut(actor); err != nil {
			return Result{}, err
		}
		return Result{Bytes: raw, Policy: p, OptedOut: true, Rules: []string{"unredacted_opt_out", "redaction_not_requested", "content_classification_not_performed"}}, nil
	}
	if err := ValidatePolicy(p); err != nil {
		return Result{}, err
	}
	if isMP4(raw) {
		result, err := redactVideo(raw, p)
		if err != nil {
			return Result{}, err
		}
		result.Requested = true
		return result, nil
	}
	if !IsEncodedImage(raw) {
		result := RedactFrame(raw, p)
		if !result.Verified {
			return Result{}, fmt.Errorf("capture format cannot be redacted safely")
		}
		result.Requested = true
		result.Rules = []string{"content_classification_not_performed"}
		if p.CredentialPatterns {
			result.Rules = append(result.Rules, "credential_patterns")
		}
		if p.OneTimeCodes {
			result.Rules = append(result.Rules, "one_time_codes")
		}
		return result, nil
	}
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Result{}, fmt.Errorf("decode capture for redaction: %w", err)
	}
	regions := []Region{}
	if p.PasswordFields || p.OneTimeCodes || p.CredentialPatterns || p.NotificationContent {
		height := decoded.Bounds().Dy() / statusBarMaskDivisor
		if height > 0 {
			regions = append(regions, Region{X0: decoded.Bounds().Min.X, Y0: decoded.Bounds().Min.Y, X1: decoded.Bounds().Max.X, Y1: decoded.Bounds().Min.Y + height})
		}
	}
	regions = append(regions, sensitive...)
	if len(regions) == 0 {
		return Result{Bytes: raw, Verified: true, Requested: true, Policy: p, Rules: []string{"status_bar_identifiers_not_present", "content_classification_not_performed"}}, nil
	}
	result := RedactImage(raw, mediaType, regions, p)
	if !result.Verified {
		return Result{}, fmt.Errorf("capture redaction failed")
	}
	result.Requested = true
	result.Rules = []string{"status_bar_identifiers", "content_classification_not_performed"}
	if len(sensitive) > 0 {
		result.Rules = append(result.Rules, "flow_sensitive_regions")
	}
	return result, nil
}

// CaptureWithPolicy leaves local evidence untouched unless redaction was
// explicitly selected. allowUnredacted is the legacy audited opt-out field.
func CaptureWithPolicy(raw []byte, mediaType string, p Policy, redactSensitive, allowUnredacted bool, actor string, sensitive []Region) (Result, error) {
	if allowUnredacted {
		return RedactCaptureWithRegions(raw, mediaType, p, true, actor, sensitive)
	}
	if !redactSensitive {
		return Result{Bytes: raw, Policy: p, Rules: []string{"redaction_not_requested", "content_classification_not_performed"}}, nil
	}
	return RedactCaptureWithRegions(raw, mediaType, p, false, actor, sensitive)
}

// ValidatePolicy rejects empty redaction requests. An empty policy does not
// name a transform and therefore cannot verify a capture.
func ValidatePolicy(p Policy) error {
	if !p.PasswordFields && !p.OneTimeCodes && !p.CredentialPatterns && !p.NotificationContent {
		return fmt.Errorf("redaction policy is empty; requested redaction must name protected regions")
	}
	return nil
}

func isMP4(raw []byte) bool {
	return len(raw) >= 8 && string(raw[4:8]) == "ftyp"
}

// redactVideo masks only the status-bar band for native recordings. The
// input/output files are owner-only temporary files and are removed before
// returning; consumers receive only the resulting bytes.
func redactVideo(raw []byte, p Policy) (Result, error) {
	input, err := os.CreateTemp("", "device-control-redact-*.mp4")
	if err != nil {
		return Result{}, fmt.Errorf("create video redaction input: %w", err)
	}
	inputPath := input.Name()
	defer os.Remove(inputPath)
	if err := input.Chmod(0o600); err != nil {
		_ = input.Close()
		return Result{}, fmt.Errorf("protect video redaction input: %w", err)
	}
	if _, err := input.Write(raw); err != nil {
		_ = input.Close()
		return Result{}, fmt.Errorf("write video redaction input: %w", err)
	}
	if err := input.Close(); err != nil {
		return Result{}, fmt.Errorf("close video redaction input: %w", err)
	}
	output, err := os.CreateTemp("", "device-control-redacted-*.mp4")
	if err != nil {
		return Result{}, fmt.Errorf("create video redaction output: %w", err)
	}
	outputPath := output.Name()
	_ = output.Close()
	defer os.Remove(outputPath)
	cmd := exec.CommandContext(context.Background(), "ffmpeg", "-y", "-loglevel", "error", "-i", inputPath, "-vf", fmt.Sprintf("drawbox=x=0:y=0:w=iw:h=ih/%d:color=black:t=fill", statusBarMaskDivisor), "-c:v", "libx264", "-pix_fmt", "yuv420p", outputPath)
	if combined, err := cmd.CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("redact native video: %w (%s)", err, strings.TrimSpace(string(combined)))
	} else if len(combined) > 0 {
		return Result{}, fmt.Errorf("redact native video: unexpected ffmpeg output")
	}
	redacted, err := os.ReadFile(outputPath)
	if err != nil {
		return Result{}, fmt.Errorf("read redacted native video: %w", err)
	}
	return Result{Bytes: redacted, Verified: true, Policy: p, Regions: 1, Rules: []string{"status_bar_identifiers", "content_classification_not_performed"}}, nil
}

func RedactFrame(raw []byte, p Policy) Result {
	// Pixel-level masking is transport-specific; the producer still stamps a
	// verified policy result. Metadata/text fixtures are redacted here, while
	// image adapters supply detected regions before this boundary.
	if len(raw) == 0 {
		return Result{Bytes: nil, Verified: true, Policy: p}
	}
	if IsEncodedImage(raw) {
		return Result{Bytes: nil, Verified: false, Policy: p}
	}
	text := string(raw)
	redacted := RedactText(text, p)
	regions := strings.Count(redacted, "[REDACTED")
	return Result{Bytes: []byte(redacted), Verified: true, Regions: regions, Policy: p}
}

type Region struct{ X0, Y0, X1, Y1 int }

// RedactImage is the producer boundary for screen captures. A caller must
// provide detector-approved regions; each region is physically painted before
// the image is encoded. Raw PNG/JPEG bytes are never stamped verified by the
// text-only RedactFrame path.
func RedactImage(raw []byte, _ string, regions []Region, p Policy) Result {
	decoded, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Result{Policy: p}
	}
	if len(regions) == 0 {
		return Result{Bytes: raw, Policy: p}
	}
	masked := image.NewRGBA(decoded.Bounds())
	draw.Draw(masked, masked.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	for _, r := range regions {
		bounds := image.Rect(r.X0, r.Y0, r.X1, r.Y1).Intersect(masked.Bounds())
		if bounds.Empty() {
			continue
		}
		draw.Draw(masked, bounds, image.NewUniform(image.Black), image.Point{}, draw.Src)
	}
	var out bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&out, masked, &jpeg.Options{Quality: 90}); err != nil {
			return Result{Policy: p}
		}
	default:
		if err := png.Encode(&out, masked); err != nil {
			return Result{Policy: p}
		}
	}
	return Result{Bytes: out.Bytes(), Verified: true, Regions: len(regions), Policy: p}
}

func IsEncodedImage(raw []byte) bool {
	return bytes.HasPrefix(raw, []byte("\x89PNG")) || bytes.HasPrefix(raw, []byte("\xff\xd8\xff"))
}

func ValidateOptOut(actor string) error {
	if strings.TrimSpace(actor) == "" {
		return fmtError("unredacted capture requires an actor")
	}
	return nil
}

type simpleError string

func (e simpleError) Error() string { return string(e) }
func fmtError(s string) error       { return simpleError(s) }
