package modules_test

import (
	"testing"

	"agent-manager/internal/modules"
)

func TestAllProtoFilesRegistersEpisodesService(t *testing.T) {
	for _, entry := range modules.AllProtoFiles() {
		if entry.Module == "episodes" && entry.File.Services().ByName("EpisodesService") != nil {
			return
		}
	}
	t.Fatal("EpisodesService must be registered in AllProtoFiles")
}
