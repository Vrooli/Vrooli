package svg

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	
	"image-tools/plugins"
)

type SVGPlugin struct {
	name string
}

func New() *SVGPlugin {
	return &SVGPlugin{
		name: "svg-optimizer",
	}
}

func (p *SVGPlugin) Name() string {
	return p.name
}

func (p *SVGPlugin) SupportedFormats() []string {
	return []string{"svg"}
}

func (p *SVGPlugin) CanProcess(format string) bool {
	return format == "svg"
}

func (p *SVGPlugin) Compress(input io.Reader, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	
	svg := string(data)
	originalSize := int64(len(data))
	
	// Basic SVG optimization
	optimized := p.optimizeSVG(svg)
	
	processedSize := int64(len(optimized))
	ratio := float64(processedSize) / float64(originalSize)
	savings := (1 - ratio) * 100
	
	// Extract dimensions
	width, height := p.extractDimensions(optimized)
	
	return &plugins.ProcessResult{
		OutputData: []byte(optimized),
		OutputInfo: plugins.ImageInfo{
			Format:    "svg",
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

func (p *SVGPlugin) Resize(input io.Reader, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	
	svg := string(data)
	
	// Update viewBox and dimensions
	resized := p.resizeSVG(svg, options.Width, options.Height, options.MaintainAspect)
	
	return &plugins.ProcessResult{
		OutputData: []byte(resized),
		OutputInfo: plugins.ImageInfo{
			Format:    "svg",
			Width:     options.Width,
			Height:    options.Height,
			SizeBytes: int64(len(resized)),
		},
		ProcessedSize: int64(len(resized)),
	}, nil
}

func (p *SVGPlugin) Convert(input io.Reader, targetFormat string, options plugins.ProcessOptions) (*plugins.ProcessResult, error) {
	if targetFormat == "svg" {
		return p.Compress(input, options)
	}
	return nil, fmt.Errorf("SVG plugin cannot convert to %s", targetFormat)
}

func (p *SVGPlugin) StripMetadata(input io.Reader) (*plugins.ProcessResult, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	
	svg := string(data)
	
	// Remove metadata, comments, and unnecessary attributes
	cleaned := p.stripSVGMetadata(svg)
	
	width, height := p.extractDimensions(cleaned)
	
	return &plugins.ProcessResult{
		OutputData: []byte(cleaned),
		OutputInfo: plugins.ImageInfo{
			Format:    "svg",
			Width:     width,
			Height:    height,
			SizeBytes: int64(len(cleaned)),
		},
		ProcessedSize: int64(len(cleaned)),
	}, nil
}

func (p *SVGPlugin) GetInfo(input io.Reader) (*plugins.ImageInfo, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	
	svg := string(data)
	width, height := p.extractDimensions(svg)
	
	metadata := make(map[string]string)
	
	// Extract title if present
	if match := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(svg); len(match) > 1 {
		metadata["title"] = match[1]
	}
	
	// Extract description if present
	if match := regexp.MustCompile(`<desc>([^<]+)</desc>`).FindStringSubmatch(svg); len(match) > 1 {
		metadata["description"] = match[1]
	}
	
	return &plugins.ImageInfo{
		Format:     "svg",
		Width:      width,
		Height:     height,
		SizeBytes:  int64(len(data)),
		Metadata:   metadata,
		ColorSpace: "Vector",
	}, nil
}

func (p *SVGPlugin) optimizeSVG(svg string) string {
	// Remove comments
	svg = regexp.MustCompile(`<!--[\s\S]*?-->`).ReplaceAllString(svg, "")
	
	// Remove unnecessary whitespace
	svg = regexp.MustCompile(`\s+`).ReplaceAllString(svg, " ")
	svg = regexp.MustCompile(`>\s+<`).ReplaceAllString(svg, "><")
	
	// Remove empty groups
	svg = regexp.MustCompile(`<g[^>]*>\s*</g>`).ReplaceAllString(svg, "")
	
	// Remove default attributes
	svg = regexp.MustCompile(`\s+fill="none"`).ReplaceAllString(svg, "")
	svg = regexp.MustCompile(`\s+stroke="none"`).ReplaceAllString(svg, "")
	
	// Optimize number precision
	svg = regexp.MustCompile(`(\d+\.\d{3})\d+`).ReplaceAllString(svg, "$1")
	
	// Remove unnecessary namespaces
	svg = regexp.MustCompile(`\s+xmlns:xlink="[^"]*"`).ReplaceAllString(svg, "")
	
	return strings.TrimSpace(svg)
}

func (p *SVGPlugin) stripSVGMetadata(svg string) string {
	// Remove metadata elements
	svg = regexp.MustCompile(`<metadata[\s\S]*?</metadata>`).ReplaceAllString(svg, "")
	
	// Remove editor-specific attributes
	svg = regexp.MustCompile(`\s+inkscape:[^=]+=("[^"]*"|'[^']*')`).ReplaceAllString(svg, "")
	svg = regexp.MustCompile(`\s+sodipodi:[^=]+=("[^"]*"|'[^']*')`).ReplaceAllString(svg, "")
	svg = regexp.MustCompile(`\s+illustrator:[^=]+=("[^"]*"|'[^']*')`).ReplaceAllString(svg, "")
	
	// Also apply optimization
	return p.optimizeSVG(svg)
}

func (p *SVGPlugin) resizeSVG(svg string, width, height int, maintainAspect bool) string {
	// Update width and height attributes
	widthRe := regexp.MustCompile(`width="[^"]*"`)
	heightRe := regexp.MustCompile(`height="[^"]*"`)
	
	svg = widthRe.ReplaceAllString(svg, fmt.Sprintf(`width="%d"`, width))
	svg = heightRe.ReplaceAllString(svg, fmt.Sprintf(`height="%d"`, height))
	
	if maintainAspect {
		// Add preserveAspectRatio if not present
		if !strings.Contains(svg, "preserveAspectRatio") {
			svg = strings.Replace(svg, "<svg", `<svg preserveAspectRatio="xMidYMid meet"`, 1)
		}
	}
	
	return svg
}

func (p *SVGPlugin) extractDimensions(svg string) (int, int) {
	width, height := 0, 0
	
	// Try to extract from width/height attributes
	if match := regexp.MustCompile(`width="(\d+)"`).FindStringSubmatch(svg); len(match) > 1 {
		fmt.Sscanf(match[1], "%d", &width)
	}
	
	if match := regexp.MustCompile(`height="(\d+)"`).FindStringSubmatch(svg); len(match) > 1 {
		fmt.Sscanf(match[1], "%d", &height)
	}
	
	// If not found, try viewBox
	if width == 0 || height == 0 {
		if match := regexp.MustCompile(`viewBox="[^"]*\s+(\d+)\s+(\d+)"`).FindStringSubmatch(svg); len(match) > 2 {
			fmt.Sscanf(match[1], "%d", &width)
			fmt.Sscanf(match[2], "%d", &height)
		}
	}
	
	return width, height
}