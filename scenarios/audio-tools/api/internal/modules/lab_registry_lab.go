//go:build audio_lab

package modules

import (
	corpusH "audio-tools/handlers/corpus"
	experimentH "audio-tools/handlers/experiment"
	soakH "audio-tools/handlers/soak"
	validationH "audio-tools/handlers/validation"
	"audio-tools/internal/corpus"
	"audio-tools/internal/experiment"
	"audio-tools/internal/modulekit"
	"github.com/vrooli/api-core/database"
)

func appendLabEndpoints(product []modulekit.EndpointDescriptor) []modulekit.EndpointDescriptor {
	out := append([]modulekit.EndpointDescriptor{}, product...)
	out = append(out, corpusH.Endpoints...)
	out = append(out, experimentH.Endpoints...)
	out = append(out, soakH.Endpoints...)
	out = append(out, validationH.Endpoints...)
	return out
}

func appendLabSchemas(product []database.SchemaProvider) []database.SchemaProvider {
	out := append([]database.SchemaProvider{}, product...)
	return append(out, LabSchemas()...)
}

func labSchemas() []database.SchemaProvider {
	return []database.SchemaProvider{
		database.SchemaProviderFunc(corpus.Schema),
		database.SchemaProviderFunc(experiment.Schema),
	}
}

func IsLabBuild() bool { return true }
