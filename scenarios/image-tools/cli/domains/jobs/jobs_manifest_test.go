package jobs

import (
	"os"
	"path/filepath"
	"testing"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"

	"github.com/vrooli/cli-core/cliapp"
)

// TestJobsManifestCoversJobsService asserts that every RPC on JobsService is
// either bound by a manifest command or documented in the manifest's `omitted`
// array (WatchJob is omitted there because it is the hand-appended server-stream
// `jobs watch` command). Adding a new RPC without binding/omitting it fails here.
func TestJobsManifestCoversJobsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, jobsv1.File_image_tools_v1_jobs_jobs_proto, "JobsService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
