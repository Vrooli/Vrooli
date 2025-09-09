package png

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	
	"github.com/disintegration/imaging"
	
	"image-tools/plugins"
)

type PNGPlugin struct {
	name string
}

func New() *PNGPlugin {
	return &PNGPlugin{
		name: "png-optimizer",
	}
}

func (p *PNGPlugin) Name() string {
	return p.name
}

func (p *PNGPlugin) SupportedFormats() []string {
	return []string{"png"}
}

func (p *PNGPlugin) CanProcess(format string) bool {
	return format == "png"
}

func (p *PNGPlugin) Compress(input io.Reader, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	img, err := png.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}
	
	var buf bytes.Buffer
	encoder := &png.Encoder{
		CompressionLevel: png.BestCompression,
	}
	
	err = encoder.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}
	
	bounds := img.Bounds()
	processedSize := int64(buf.Len())
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "png",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			SizeBytes: processedSize,
		},
		ProcessedSize: processedSize,
	}, nil
}

func (p *PNGPlugin) Resize(input io.Reader, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	img, err := png.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}
	
	var resized image.Image
	
	if options.MaintainAspect {
		resized = imaging.Fit(img, options.Width, options.Height, imaging.Lanczos)
	} else {
		resized = imaging.Resize(img, options.Width, options.Height, imaging.Lanczos)
	}
	
	var buf bytes.Buffer
	encoder := &png.Encoder{
		CompressionLevel: png.BestCompression,
	}
	
	err = encoder.Encode(&buf, resized)
	if err != nil {
		return nil, fmt.Errorf("failed to encode resized PNG: %w", err)
	}
	
	bounds := resized.Bounds()
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "png",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			SizeBytes: int64(buf.Len()),
		},
		ProcessedSize: int64(buf.Len()),
	}, nil
}

func (p *PNGPlugin) Convert(input io.Reader, targetFormat string, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	if targetFormat == "png" {
		return p.Compress(input, options)
	}
	return nil, fmt.Errorf("PNG plugin cannot convert to %s", targetFormat)
}

func (p *PNGPlugin) StripMetadata(input io.Reader) (*plugins.ProcessResult, error) {
	img, err := png.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}
	
	var buf bytes.Buffer
	encoder := &png.Encoder{
		CompressionLevel: png.BestCompression,
	}
	
	err = encoder.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode stripped PNG: %w", err)
	}
	
	bounds := img.Bounds()
	
	return &plugins.ProcessResult{
		OutputData: buf.Bytes(),
		OutputInfo: plugins.ImageInfo{
			Format:    "png",
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			SizeBytes: int64(buf.Len()),
		},
		ProcessedSize: int64(buf.Len()),
	}, nil
}

func (p *PNGPlugin) GetInfo(input io.Reader) (*plugins.ImageInfo, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	
	reader := bytes.NewReader(data)
	config, err := png.DecodeConfig(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG config: %w", err)
	}
	
	colorSpace := "RGB"
	switch config.ColorModel {
	case image.GrayModel:
		colorSpace = "Grayscale"
	case image.Gray16Model:
		colorSpace = "Grayscale16"
	case image.RGBAModel:
		colorSpace = "RGBA"
	case image.RGBA64Model:
		colorSpace = "RGBA64"
	}
	
	return &plugins.ImageInfo{
		Format:     "png",
		Width:      config.Width,
		Height:     config.Height,
		SizeBytes:  int64(len(data)),
		Metadata:   make(map[string]string),
		ColorSpace: colorSpace,
	}, nil
}