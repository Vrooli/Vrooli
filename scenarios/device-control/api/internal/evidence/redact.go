package evidence

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
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
		return Result{}, fmt.Errorf("video redaction is unavailable; content detection is not implemented")
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
	if len(sensitive) == 0 {
		return Result{}, fmt.Errorf("image redaction requires explicit sensitive regions; content detection is unavailable")
	}
	result := RedactImage(raw, mediaType, sensitive, p)
	if !result.Verified {
		return Result{}, fmt.Errorf("capture redaction failed or a sensitive region is invalid")
	}
	result.Requested = true
	result.Rules = []string{"flow_sensitive_regions", "content_classification_not_performed"}
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

// RedactImage paints explicit caller-supplied regions. It does not detect
// private content or imply that other areas are safe to share.
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
		if r.X1 <= r.X0 || r.Y1 <= r.Y0 {
			return Result{Policy: p}
		}
		bounds := image.Rect(r.X0, r.Y0, r.X1, r.Y1)
		if !bounds.In(masked.Bounds()) {
			return Result{Policy: p}
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
