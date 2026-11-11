package modules_test

import (
	"testing"

	"architecture-cartographer/internal/database"
	"architecture-cartographer/internal/modules"
	"github.com/vrooli/api-core/moduletest"
)

func TestRegistryContracts(t *testing.T) {
	moduletest.AssertRegistryContracts(t,
		modules.AllEndpoints,
		modules.AllSchemas,
		database.SystemSchema,
		allProtoFiles,
		false,
	)
}

func allProtoFiles() []moduletest.ProtoFile {
	entries := modules.AllProtoFiles()
	files := make([]moduletest.ProtoFile, 0, len(entries))
	for _, entry := range entries {
		files = append(files, moduletest.ProtoFile{
			Module:   entry.Module,
			File:     entry.File,
			Services: entry.Services,
		})
	}
	return files
}
