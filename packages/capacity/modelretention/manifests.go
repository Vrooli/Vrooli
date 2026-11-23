package modelretention

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxManifestBytes bounds one manifest read; real manifests are a few KiB.
const maxManifestBytes = 1 << 20

// ModelFootprint is one tag's blob accounting.
type ModelFootprint struct {
	// Bytes is every blob the tag references.
	Bytes int64
	// UniqueBytes are blobs no other installed tag references: what deleting
	// this tag alone frees.
	UniqueBytes int64
}

// Footprint is a store's blob accounting read from its manifests.
type Footprint struct {
	Models map[string]ModelFootprint
	// StoreBytes counts each referenced blob once.
	StoreBytes int64
}

type manifestDescriptor struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type manifestDocument struct {
	Config manifestDescriptor   `json:"config"`
	Layers []manifestDescriptor `json:"layers"`
}

// ReadFootprint walks <modelRoot>/manifests/<host>/<namespace>/<repo>/<tag>.
// Names follow the store's own listing: the default registry host and the
// "library" namespace are omitted, as /api/tags does.
func ReadFootprint(modelRoot, defaultRegistry string) (Footprint, error) {
	root := filepath.Join(modelRoot, "manifests")
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		// An empty store has no manifests directory yet.
		return Footprint{Models: map[string]ModelFootprint{}}, nil
	}
	blobs := map[string]map[string]int64{} // model -> digest -> size
	refs := map[string]int{}
	sizes := map[string]int64{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name, ok := manifestName(filepath.ToSlash(rel), defaultRegistry)
		if !ok {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxManifestBytes {
			return fmt.Errorf("manifest %s is larger than %d bytes", rel, maxManifestBytes)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc manifestDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("decode manifest %s: %w", rel, err)
		}
		set := map[string]int64{}
		for _, d := range append([]manifestDescriptor{doc.Config}, doc.Layers...) {
			if d.Digest == "" || d.Size < 0 {
				continue
			}
			set[d.Digest] = d.Size
		}
		blobs[name] = set
		for digest, size := range set {
			refs[digest]++
			sizes[digest] = size
		}
		return nil
	})
	if err != nil {
		return Footprint{}, fmt.Errorf("read model manifests under %s: %w", root, err)
	}
	out := Footprint{Models: make(map[string]ModelFootprint, len(blobs))}
	for _, size := range sizes {
		out.StoreBytes += size
	}
	for name, set := range blobs {
		var fp ModelFootprint
		for digest, size := range set {
			fp.Bytes += size
			if refs[digest] == 1 {
				fp.UniqueBytes += size
			}
		}
		out.Models[name] = fp
	}
	return out, nil
}

func manifestName(rel, defaultRegistry string) (string, bool) {
	parts := strings.Split(rel, "/")
	if len(parts) < 4 {
		return "", false
	}
	host, tag := parts[0], parts[len(parts)-1]
	path := parts[1 : len(parts)-1]
	if len(path) == 2 && path[0] == "library" {
		path = path[1:]
	}
	name := strings.Join(path, "/") + ":" + tag
	if !strings.EqualFold(host, defaultRegistry) {
		name = host + "/" + name
	}
	return CanonicalName(name), true
}
