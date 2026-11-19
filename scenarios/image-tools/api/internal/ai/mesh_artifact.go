package ai

import (
	"fmt"
	"io"
	"os"

	"image-tools/internal/meshbundle"
)

// validateMeshArtifact checks a finished image_to_object bundle before it is
// stored. The durable artifact is a ZIP of model.glb + manifest.json carrying the
// checkpoint digest; a truncated download or a provider that wrote the wrong
// shape would otherwise be published as a successful mesh and discovered only by
// whoever opens it later.
//
// The check reads no more than the archive ceiling, so a runaway provider cannot
// make the API read an unbounded file.
func validateMeshArtifact(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("image_to_object: open bundle: %w", err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, meshbundle.MaxArchiveBytes+1))
	if err != nil {
		return fmt.Errorf("image_to_object: read bundle: %w", err)
	}
	return meshbundle.Validate(data)
}
