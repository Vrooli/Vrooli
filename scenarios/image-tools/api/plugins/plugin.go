package plugins

import (
	"io"
)

type ImageInfo struct {
	Format     string            `json:"format"`
	Width      int               `json:"width"`
	Height     int               `json:"height"`
	SizeBytes  int64             `json:"size_bytes"`
	Metadata   map[string]string `json:"metadata"`
	ColorSpace string            `json:"color_space"`
}

type ProcessOptions struct {
	Quality        int                    `json:"quality,omitempty"`
	Width          int                    `json:"width,omitempty"`
	Height         int                    `json:"height,omitempty"`
	MaintainAspect bool                   `json:"maintain_aspect,omitempty"`
	StripMetadata  bool                   `json:"strip_metadata,omitempty"`
	Algorithm      string                 `json:"algorithm,omitempty"`
	CustomOptions  map[string]interface{} `json:"custom_options,omitempty"`
}

type ProcessResult struct {
	OutputData       []byte  `json:"-"`
	OutputInfo       ImageInfo `json:"output_info"`
	OriginalSize     int64   `json:"original_size"`
	ProcessedSize    int64   `json:"processed_size"`
	CompressionRatio float64 `json:"compression_ratio"`
	SavingsPercent   float64 `json:"savings_percent"`
}

type Plugin interface {
	Name() string
	SupportedFormats() []string
	CanProcess(format string) bool
	
	Compress(input io.Reader, options ProcessOptions) (*ProcessResult, error)
	Resize(input io.Reader, options ProcessOptions) (*ProcessResult, error)
	Convert(input io.Reader, targetFormat string, options ProcessOptions) (*ProcessResult, error)
	StripMetadata(input io.Reader) (*ProcessResult, error)
	GetInfo(input io.Reader) (*ImageInfo, error)
}

type PluginRegistry struct {
	plugins map[string]Plugin
}

func NewRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

func (r *PluginRegistry) Register(plugin Plugin) {
	for _, format := range plugin.SupportedFormats() {
		r.plugins[format] = plugin
	}
}

func (r *PluginRegistry) GetPlugin(format string) (Plugin, bool) {
	plugin, ok := r.plugins[format]
	return plugin, ok
}

func (r *PluginRegistry) ListPlugins() map[string][]string {
	result := make(map[string][]string)
	for _, plugin := range r.plugins {
		result[plugin.Name()] = plugin.SupportedFormats()
	}
	return result
}