package jpeg

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	
	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	
	"image-tools/plugins"
)

type JPEGPlugin struct {
	name string
}

func New() *JPEGPlugin {
	return &JPEGPlugin{
		name: "jpeg-optimizer",
	}
}

func (p *JPEGPlugin) Name() string {
	return p.name
}

func (p *JPEGPlugin) SupportedFormats() []string {
	return []string{"jpeg", "jpg"}
}

func (p *JPEGPlugin) CanProcess(format string) bool {
	for _, f := range p.SupportedFormats() {
		if f == format {
			return true
		}
	}
	return false
}

func (p *JPEGPlugin) Compress(input io.Reader, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	// Read input data to get original size
	inputData, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	originalSize := int64(len(inputData))
	
	// Decode image
	img, err := jpeg.Decode(bytes.NewReader(inputData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %w", err)
	}
	
	quality := options.Quality
	if quality == 0 {
		quality = 85
	}
	
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}
	
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	processedSize := int64(buf.Len())
	ratio := float64(processedSize) / float64(originalSize)
	savings := (1 - ratio) * 100
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "jpeg",
			Width:     width,
			Height:    height,
			SizeBytes: processedSize,
		},
		OriginalSize:     originalSize,
		ProcessedSize:    processedSize,
		CompressionRatio: ratio,
		SavingsPercent:   savings,
	}, nil
}

func (p *JPEGPlugin) Resize(input io.Reader, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	img, err := jpeg.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %w", err)
	}
	
	var resized image.Image
	
	if options.MaintainAspect {
		resized = imaging.Fit(img, options.Width, options.Height, imaging.Lanczos)
	} else {
		resized = imaging.Resize(img, options.Width, options.Height, imaging.Lanczos)
	}
	
	var buf bytes.Buffer
	quality := options.Quality
	if quality == 0 {
		quality = 95
	}
	
	err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode resized JPEG: %w", err)
	}
	
	bounds := resized.Bounds()
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "jpeg",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			SizeBytes: int64(buf.Len()),
		},
		ProcessedSize: int64(buf.Len()),
	}, nil
}

func (p *JPEGPlugin) Convert(input io.Reader, targetFormat string, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	if targetFormat == "jpeg" || targetFormat == "jpg" {
		return p.Compress(input, options)
	}
	return nil, fmt.Errorf("JPEG plugin cannot convert to %s", targetFormat)
}

func (p *JPEGPlugin) StripMetadata(input io.Reader) (*plugins.ProcessResult, error) {
	img, err := jpeg.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %w", err)
	}
	
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	if err != nil {
		return nil, fmt.Errorf("failed to encode stripped JPEG: %w", err)
	}
	
	bounds := img.Bounds()
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "jpeg",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			SizeBytes: int64(buf.Len()),
		},
		ProcessedSize: int64(buf.Len()),
	}, nil
}

func (p *JPEGPlugin) GetInfo(input io.Reader) (*plugins.ImageInfo, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	
	reader := bytes.NewReader(data)
	img, err := jpeg.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %w", err)
	}
	
	bounds := img.Bounds()
	metadata := make(map[string]string)
	
	reader.Seek(0, 0)
	x, err := exif.Decode(reader)
	if err == nil {
		metadata["camera"] = getExifTag(x, exif.Model)
		metadata["datetime"] = getExifTag(x, exif.DateTime)
		metadata["orientation"] = getExifTag(x, exif.Orientation)
	}
	
	return &plugins.ImageInfo{
		Format:     "jpeg",
		Width:      bounds.Dx(),
		Height:     bounds.Dy(),
		SizeBytes:  int64(len(data)),
		Metadata:   metadata,
		ColorSpace: "RGB",
	}, nil
}

func getExifTag(x *exif.Exif, tag exif.FieldName) string {
	val, err := x.Get(tag)
	if err != nil {
		return ""
	}
	return val.String()
}