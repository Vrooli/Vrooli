package webp

import (
	"bytes"
	"fmt"
	"image"
	"io"
	
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	
	"image-tools/plugins"
)

type WebPPlugin struct {
	name string
}

func New() *WebPPlugin {
	return &WebPPlugin{
		name: "webp-optimizer",
	}
}

func (p *WebPPlugin) Name() string {
	return p.name
}

func (p *WebPPlugin) SupportedFormats() []string {
	return []string{"webp"}
}

func (p *WebPPlugin) CanProcess(format string) bool {
	return format == "webp"
}

func (p *WebPPlugin) Compress(input io.Reader, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	img, err := webp.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WebP: %w", err)
	}
	
	quality := float32(options.Quality)
	if quality == 0 {
		quality = 85
	}
	
	var buf bytes.Buffer
	opts := &webp.Options{
		Lossless: false,
		Quality:  quality,
	}
	
	err = webp.Encode(&buf, img, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}
	
	bounds := img.Bounds()
	processedSize := int64(buf.Len())
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "webp",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			SizeBytes: processedSize,
		},
		ProcessedSize: processedSize,
	}, nil
}

func (p *WebPPlugin) Resize(input io.Reader, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	img, err := webp.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WebP: %w", err)
	}
	
	var resized image.Image
	
	if options.MaintainAspect {
		resized = imaging.Fit(img, options.Width, options.Height, imaging.Lanczos)
	} else {
		resized = imaging.Resize(img, options.Width, options.Height, imaging.Lanczos)
	}
	
	var buf bytes.Buffer
	quality := float32(options.Quality)
	if quality == 0 {
		quality = 90
	}
	
	opts := &webp.Options{
		Lossless: false,
		Quality:  quality,
	}
	
	err = webp.Encode(&buf, resized, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to encode resized WebP: %w", err)
	}
	
	bounds := resized.Bounds()
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "webp",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			SizeBytes: int64(buf.Len()),
		},
		ProcessedSize: int64(buf.Len()),
	}, nil
}

func (p *WebPPlugin) Convert(input io.Reader, targetFormat string, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	if targetFormat == "webp" {
		return p.Compress(input, options)
	}
	return nil, fmt.Errorf("WebP plugin cannot convert to %s", targetFormat)
}

func (p *WebPPlugin) StripMetadata(input io.Reader) (*plugins.ProcessResult, error) {
	img, err := webp.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WebP: %w", err)
	}
	
	var buf bytes.Buffer
	opts := &webp.Options{
		Lossless: false,
		Quality:  90,
	}
	
	err = webp.Encode(&buf, img, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to encode stripped WebP: %w", err)
	}
	
	bounds := img.Bounds()
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "webp",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			SizeBytes: int64(buf.Len()),
		},
		ProcessedSize: int64(buf.Len()),
	}, nil
}

func (p *WebPPlugin) GetInfo(input io.Reader) (*plugins.ImageInfo, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	
	reader := bytes.NewReader(data)
	config, err := webp.DecodeConfig(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WebP config: %w", err)
	}
	
	colorSpace := "RGB"
	switch config.ColorModel {
	case image.GrayModel:
		colorSpace = "Grayscale"
	case image.RGBAModel:
		colorSpace = "RGBA"
	}
	
	return &plugins.ImageInfo{
		Format:     "webp",
		Width:      config.Width,
		Height:     config.Height,
		SizeBytes:  int64(len(data)),
		Metadata:   make(map[string]string),
		ColorSpace: colorSpace,
	}, nil
}