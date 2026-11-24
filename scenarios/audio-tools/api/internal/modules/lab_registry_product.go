//go:build !audio_lab

package modules

import (
	"audio-tools/internal/modulekit"
	"github.com/vrooli/api-core/database"
)

func appendLabEndpoints(product []modulekit.EndpointDescriptor) []modulekit.EndpointDescriptor {
	return product
}

func appendLabSchemas(product []database.SchemaProvider) []database.SchemaProvider {
	return product
}

func labSchemas() []database.SchemaProvider { return nil }

func IsLabBuild() bool { return false }
