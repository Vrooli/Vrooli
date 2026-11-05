package ffmpegvideo

import (
	"encoding/json"
	"io/fs"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	alltools "github.com/vrooli/vrooli/internal/tools"
)

func TestDedicatedManifestPinsExactPairedArchiveWithoutChangingGenericFFmpeg(t *testing.T) {
	data, err := fs.ReadFile(alltools.Manifests, "ffmpeg-video/tool.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest hostreqkit.ToolManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	artifact := PinnedArtifact()
	if manifest.Name != "ffmpeg-video" || len(manifest.Commands) != 1 || manifest.Commands[0] != "ffmpeg-video" {
		t.Fatalf("dedicated managed identity = %q / %#v", manifest.Name, manifest.Commands)
	}
	if manifest.Acquisition == nil || len(manifest.Acquisition.Targets) != 1 {
		t.Fatal("expected one digest-pinned platform acquisition target")
	}
	target := manifest.Acquisition.Targets[0]
	if target.When["os"] != "linux" || target.When["arch"] != "amd64" || target.URL != artifact.ArchiveURL || target.SHA256 != artifact.ArchiveSHA256 || target.Archive != "tar.xz" || target.Layout != "dir" || target.BinPath != artifact.FFmpegPath {
		t.Fatalf("acquisition target does not match inspected candidate: %+v", target)
	}
	genericData, err := fs.ReadFile(alltools.Manifests, "ffmpeg/tool.json")
	if err != nil {
		t.Fatal(err)
	}
	var generic struct {
		Name     string   `json:"name"`
		Commands []string `json:"commands"`
	}
	if err := json.Unmarshal(genericData, &generic); err != nil {
		t.Fatal(err)
	}
	if generic.Name != "ffmpeg" || len(generic.Commands) != 1 || generic.Commands[0] != "ffmpeg" {
		t.Fatalf("general ffmpeg declaration was changed: %+v", generic)
	}
}
