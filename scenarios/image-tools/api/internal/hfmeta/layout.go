package hfmeta

import (
	"path"
	"strings"
)

// checkpointExts are the single-file checkpoint extensions a base model ships as
// (a self-contained weight blob, no model_index.json).
var checkpointExts = map[string]struct{}{
	".safetensors": {},
	".ckpt":        {},
	".gguf":        {},
	".pt":          {},
	".bin":         {},
}

// DetectLayout classifies a file listing as a diffusers repo (model_index.json
// present) or a single-file checkpoint (a top-level weight blob and no
// model_index.json). It is pure so layout detection is unit-tested without a
// network. An empty/ambiguous listing returns LayoutUnknown so the import wizard
// asks rather than guesses.
func DetectLayout(files []FileInfo) Layout {
	hasCheckpoint := false
	for _, f := range files {
		name := strings.ToLower(path.Base(f.Path))
		if name == "model_index.json" {
			return LayoutDiffusersRepo
		}
		ext := strings.ToLower(path.Ext(name))
		if _, ok := checkpointExts[ext]; ok {
			// Only a TOP-LEVEL checkpoint marks single-file; a .safetensors shard
			// inside transformer/ belongs to a repo (already caught by model_index).
			if !strings.Contains(strings.Trim(f.Path, "/"), "/") {
				hasCheckpoint = true
			}
		}
	}
	if hasCheckpoint {
		return LayoutSingleFile
	}
	return LayoutUnknown
}
