package modules_test

import (
	"testing"

	"code-facts/internal/database"
	"code-facts/internal/modules"
	"github.com/vrooli/api-core/moduletest"
)

func TestRegistryContracts(t *testing.T) {
	moduletest.AssertRegistryContracts(t,
		modules.AllEndpoints,
		modules.AllSchemas,
		database.SystemSchema,
		allProtoFiles,
		true,
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
