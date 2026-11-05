package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

type BuildResult struct {
	GenerationID string
	SourceDigest string
	Files        int
	Bytes        int64
	Roles        map[Role]int
	RoleBytes    map[Role]int64
	MaxBatch     int
}

type Builder struct {
	Repository Repository
	Discoverer Discoverer
	Clock      Clock
	BatchSize  int
	// SkipBegin supports durable runtimes that create the shadow generation
	// before its job row, whose schema has a generation foreign key.
	SkipBegin bool
}

func (b Builder) Build(ctx context.Context, generation Generation) (result BuildResult, err error) {
	if b.Repository == nil || b.Discoverer == nil || b.Clock == nil {
		return BuildResult{}, fmt.Errorf("catalog builder requires repository, discoverer, and clock")
	}
	if b.BatchSize <= 0 {
		b.BatchSize = 256
	}
	if b.BatchSize > 4096 {
		b.BatchSize = 4096
	}
	if generation.CreatedAt.IsZero() {
		generation.CreatedAt = b.Clock.Now()
	}
	if !b.SkipBegin {
		if err := b.Repository.BeginGeneration(ctx, generation); err != nil {
			return BuildResult{}, err
		}
	}
	completed := false
	defer func() {
		if !completed && err != nil {
			_ = b.Repository.FailGeneration(context.WithoutCancel(ctx), generation.ID, err.Error())
		}
	}()
	iterator, err := b.Discoverer.Open(ctx)
	if err != nil {
		return BuildResult{}, err
	}
	defer iterator.Close()
	result = BuildResult{GenerationID: generation.ID, Roles: map[Role]int{}, RoleBytes: map[Role]int64{}}
	digest := sha256.New()
	batch := make([]SourceFile, 0, b.BatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if len(batch) > result.MaxBatch {
			result.MaxBatch = len(batch)
		}
		if err := b.Repository.UpsertFiles(ctx, generation.ID, batch); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	for {
		file, ok, nextErr := iterator.Next(ctx)
		if nextErr != nil {
			err = nextErr
			return result, err
		}
		if !ok {
			break
		}
		file.Generation = generation.ID
		batch = append(batch, file)
		result.Files++
		result.Bytes += file.Size
		result.Roles[file.Role]++
		result.RoleBytes[file.Role] += file.Size
		_, _ = digest.Write([]byte(file.Path))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(file.Hash))
		_, _ = digest.Write([]byte{0})
		if len(batch) == b.BatchSize {
			if err = flush(); err != nil {
				return result, err
			}
		}
	}
	if err = flush(); err != nil {
		return result, err
	}
	result.SourceDigest = "sha256:" + hex.EncodeToString(digest.Sum(nil))
	if err = b.Repository.CompleteGeneration(ctx, generation.ID, result.SourceDigest, generation.DescriptorDigest); err != nil {
		return result, err
	}
	completed = true
	return result, nil
}

func SortedRoleCounts(counts map[Role]int) []string {
	roles := make([]string, 0, len(counts))
	for role, count := range counts {
		roles = append(roles, fmt.Sprintf("%s=%d", role, count))
	}
	sort.Strings(roles)
	return roles
}
