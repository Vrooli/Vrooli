package ops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/disintegration/imaging"
	exif "github.com/dsoprea/go-exif/v3"
)

// MetaTag is one extracted metadata entry (read mode result).
type MetaTag struct {
	IFD   string `json:"ifd"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MetaReport is the JSON result of a metadata read.
type MetaReport struct {
	Format string    `json:"format"`
	Width  int       `json:"width"`
	Height int       `json:"height"`
	HasGPS bool      `json:"has_gps"`
	Tags   []MetaTag `json:"tags"`
}

// metadataRun is the metadata operation. It is a full RunFunc (not a pixel
// transform) because read mode produces a JSON report and strip/auto-orient
// must work from the original encoded bytes (EXIF orientation is lost once
// Go's decoders drop it).
//
// Modes (mutually-ordered): AutoOrient → apply EXIF orientation to pixels and
// drop metadata; StripAll/StripGPS → re-encode to drop metadata; otherwise →
// read and return a JSON tag report.
//
// NOTE: in v1 a GPS-only strip is implemented as a full metadata strip (the
// privacy-safe superset: removing all EXIF necessarily removes location).
// Selective GPS removal that preserves other tags is a documented future
// refinement (see docs/internal/PROBLEMS.md).
func metadataRun(in RunInput) (RunResult, error) {
	p := in.Params
	switch {
	case p.AutoOrient:
		return autoOrient(in)
	case p.StripAll || p.StripGPS:
		return stripMetadata(in)
	default:
		return readMetadata(in)
	}
}

func autoOrient(in RunInput) (RunResult, error) {
	// Re-decode the original bytes with EXIF auto-orientation applied, then
	// re-encode (which drops the now-applied orientation tag).
	img, err := imaging.Decode(bytes.NewReader(in.Bytes), imaging.AutoOrientation(true))
	if err != nil {
		// Not a format imaging can auto-orient (e.g. webp/avif): fall back to
		// the already-decoded image unchanged.
		img = in.Img
	}
	format := resolveOutputFormat(in.Params.Format, in.Meta.Format)
	data, err := Encode(img, format, EncodeOptions{Quality: in.Params.Quality, Lossless: in.Params.Lossless})
	if err != nil {
		return RunResult{}, err
	}
	b := img.Bounds()
	return RunResult{Bytes: data, Format: format, Mime: MIMEFor(format), Width: b.Dx(), Height: b.Dy()}, nil
}

func stripMetadata(in RunInput) (RunResult, error) {
	// Re-encoding through Go's encoders writes no EXIF/IPTC/XMP, so the result
	// carries no metadata.
	format := resolveOutputFormat(in.Params.Format, in.Meta.Format)
	data, err := Encode(in.Img, format, EncodeOptions{Quality: in.Params.Quality, Lossless: in.Params.Lossless})
	if err != nil {
		return RunResult{}, err
	}
	b := in.Img.Bounds()
	return RunResult{Bytes: data, Format: format, Mime: MIMEFor(format), Width: b.Dx(), Height: b.Dy()}, nil
}

func readMetadata(in RunInput) (RunResult, error) {
	report := MetaReport{Format: in.Meta.Format, Width: in.Meta.Width, Height: in.Meta.Height}
	rawExif, err := exif.SearchAndExtractExif(in.Bytes)
	if err == nil {
		tags, _, terr := exif.GetFlatExifData(rawExif, nil)
		if terr == nil {
			for _, t := range tags {
				if t.ChildIfdPath != "" {
					continue // structural entry, not a value
				}
				report.Tags = append(report.Tags, MetaTag{IFD: t.IfdPath, Name: t.TagName, Value: t.Formatted})
				if strings.Contains(strings.ToLower(t.IfdPath), "gps") {
					report.HasGPS = true
				}
			}
		}
	}
	data, err := json.Marshal(report)
	if err != nil {
		return RunResult{}, fmt.Errorf("ops: marshal metadata report: %w", err)
	}
	return RunResult{Bytes: data, Format: "json", Mime: "application/json"}, nil
}
