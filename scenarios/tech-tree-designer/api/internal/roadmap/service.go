package roadmap

import (
	"context"
	"fmt"
	"sort"

	graphdomain "tech-tree-designer/internal/graph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	roadmapv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap"
)

type GraphProvider interface {
	Describe(ctx context.Context, req graphdomain.SourceRequest) (*graphv1.TechTreeGraph, error)
}

type Service struct {
	repo  Repository
	graph GraphProvider
}

func NewService(repo Repository, graph GraphProvider) *Service {
	return &Service{repo: repo, graph: graph}
}

func (s *Service) ListSectors(ctx context.Context) ([]Sector, error) {
	return s.repo.ListSectors(ctx)
}

func (s *Service) UpsertSector(ctx context.Context, sector Sector) (Sector, error) {
	return s.repo.UpsertSector(ctx, sector)
}

func (s *Service) ListMilestones(ctx context.Context) ([]Milestone, error) {
	return s.repo.ListMilestones(ctx)
}

func (s *Service) UpsertMilestone(ctx context.Context, milestone Milestone) (Milestone, error) {
	return s.repo.UpsertMilestone(ctx, milestone)
}

func (s *Service) GetProgress(ctx context.Context, filter ProgressFilter) (*roadmapv1.ProgressRollup, error) {
	if _, err := NormalizeTier(filter.Tier); err != nil {
		return nil, err
	}
	if s.graph == nil {
		return nil, fmt.Errorf("graph provider is not configured")
	}
	graph, err := s.graph.Describe(ctx, graphdomain.SourceRequest{})
	if err != nil {
		return nil, err
	}
	buckets := map[string]*roadmapv1.ProgressBucket{}
	for _, node := range graph.GetNodes() {
		sector := node.GetSector()
		tier := node.GetTier()
		if filter.Sector != "" && sector != filter.Sector {
			continue
		}
		if filter.Tier != "" && tier != filter.Tier {
			continue
		}
		key := ProgressBucketKey(sector, tier)
		bucket := buckets[key]
		if bucket == nil {
			bucket = ProgressBucketProto(sector, tier)
			buckets[key] = bucket
		}
		if node.GetKind() == graphv1.NodeKind_NODE_KIND_PLANNED {
			bucket.Planned++
		} else {
			bucket.Live++
		}
		switch normalizeStability(node.GetStability()) {
		case "stable":
			bucket.Stable++
		case "beta":
			bucket.Beta++
		}
	}
	out := &roadmapv1.ProgressRollup{Buckets: make([]*roadmapv1.ProgressBucket, 0, len(buckets))}
	for _, bucket := range buckets {
		out.Buckets = append(out.Buckets, bucket)
	}
	sort.Slice(out.Buckets, func(i, j int) bool {
		if out.Buckets[i].GetSector() != out.Buckets[j].GetSector() {
			return out.Buckets[i].GetSector() < out.Buckets[j].GetSector()
		}
		return out.Buckets[i].GetTier() < out.Buckets[j].GetTier()
	})
	return out, nil
}
