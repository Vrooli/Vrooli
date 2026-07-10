package codecs

import "agent-manager/internal/adapters/runner"

// WithCatalogModels decorates a codec with the validated catalog's static
// model inventory. The wrapped codec remains the owner of runtime-discovered
// entries, so callers see one deduplicated static-plus-dynamic capability view
// without model identifiers being compiled into runner mechanics.
func WithCatalogModels(codec Codec, staticModels []string) Codec {
	models := append([]string(nil), staticModels...)
	return WithCatalogModelSource(codec, func() []string {
		return append([]string(nil), models...)
	})
}

// WithCatalogModelSource decorates a codec with a live view of the active
// catalog revision. The source is evaluated for every capabilities request so
// an atomic catalog reload affects new callers without rebuilding runners.
func WithCatalogModelSource(codec Codec, source func() []string) Codec {
	if codec == nil {
		return nil
	}
	return &catalogModelsCodec{
		Codec:  codec,
		source: source,
	}
}

type catalogModelsCodec struct {
	Codec
	source func() []string
}

func (c *catalogModelsCodec) Capabilities() runner.Capabilities {
	capabilities := c.Codec.Capabilities()
	var staticModels []string
	if c.source != nil {
		staticModels = c.source()
	}
	models := make([]string, 0, len(staticModels)+len(capabilities.SupportedModels))
	seen := make(map[string]struct{}, cap(models))
	for _, model := range append(append([]string(nil), staticModels...), capabilities.SupportedModels...) {
		if model == "" {
			continue
		}
		if _, exists := seen[model]; exists {
			continue
		}
		seen[model] = struct{}{}
		models = append(models, model)
	}
	capabilities.SupportedModels = models
	return capabilities
}
