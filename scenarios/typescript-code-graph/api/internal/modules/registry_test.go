package modules_test

import (
	"testing"

	"github.com/vrooli/api-core/moduletest"
	"typescript-code-graph/internal/database"
	"typescript-code-graph/internal/modules"
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
			Module: entry.Module,
			File:   entry.File,
		})
	}
	return files
}
