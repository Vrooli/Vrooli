package evidence

import (
	"bytes"
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
	PasswordFields     bool `json:"password_fields"`
	OneTimeCodes       bool `json:"one_time_codes"`
	CredentialPatterns bool `json:"credential_patterns"`
}

var DefaultPolicy = Policy{PasswordFields: true, OneTimeCodes: true, CredentialPatterns: true}
var credentialPattern = regexp.MustCompile(`(?i)(authorization|bearer|api[_ -]?key|secret|password)\s*[:=]\s*[^\s]+`)
var otpPattern = regexp.MustCompile(`\b\d{6}\b`)

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
	Bytes    []byte
	Verified bool
	Regions  int
	Policy   Policy
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
func RedactImage(raw []byte, mediaType string, regions []Region, p Policy) Result {
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
		mediaType = "image/jpeg"
	default:
		if err := png.Encode(&out, masked); err != nil {
			return Result{Policy: p}
		}
		mediaType = "image/png"
	}
	_ = mediaType
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
