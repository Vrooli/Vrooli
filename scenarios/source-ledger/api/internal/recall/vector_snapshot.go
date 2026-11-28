package recall

import (
	"context"
	"sync"
)

// The snapshot serves one scope so repeat queries do not decode the same
// corpus. Other scopes and oversized corpora keep the same streaming read.
const maxVectorSnapshotBytes = 256 << 20

type vectorSnapshotVersion struct {
	scope                                 string
	pointer, generation, sources, vectors int64
}

type vectorSnapshot struct {
	mu      sync.Mutex
	version vectorSnapshotVersion
	vectors map[string][][]float64
	loading chan struct{}
}

func readVectorSnapshotVersion(ctx context.Context, scope string, db sqliteReader) (vectorSnapshotVersion, error) {
	version := vectorSnapshotVersion{scope: scope}
	err := db.QueryRowContext(ctx, `
SELECT active.version, COALESCE(g.version,0), sources.version, vectors.version
FROM vector_generation_active active
LEFT JOIN vector_generations g ON g.generation_id=active.generation_id
JOIN vector_generation_mutex sources ON sources.singleton=1
JOIN vector_read_version vectors ON vectors.singleton=1
WHERE active.singleton=1`).Scan(&version.pointer, &version.generation, &version.sources, &version.vectors)
	return version, err
}

func (s *vectorSnapshot) get(ctx context.Context, scope string, db sqliteReader, load func() (map[string][][]float64, error)) (map[string][][]float64, error) {
	for {
		version, err := readVectorSnapshotVersion(ctx, scope, db)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		if s.vectors != nil && s.version == version {
			vectors := s.vectors
			s.mu.Unlock()
			return vectors, nil
		}
		if waiting := s.loading; waiting != nil {
			s.mu.Unlock()
			select {
			case <-waiting:
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		s.loading = make(chan struct{})
		s.mu.Unlock()

		vectors, err := load()
		var cacheable bool
		if err == nil && vectorSnapshotSize(vectors) <= maxVectorSnapshotBytes {
			after, versionErr := readVectorSnapshotVersion(ctx, scope, db)
			cacheable = versionErr == nil && after == version
		}
		s.mu.Lock()
		if cacheable {
			s.version, s.vectors = version, vectors
		}
		close(s.loading)
		s.loading = nil
		s.mu.Unlock()
		return vectors, err
	}
}

func vectorSnapshotSize(vectors map[string][][]float64) int64 {
	var size int64
	for id, spaces := range vectors {
		// Account for the map entry, strings, slice headers and values.
		size += int64(64 + len(id) + 24*len(spaces))
		for _, space := range spaces {
			size += 8 * int64(len(space))
		}
	}
	return size
}
