package artifacts

import (
	"context"
	"errors"

	"vrooli-bridge/internal/clock"
)

// Service is the application-layer surface the artifacts handler depends on. It
// orchestrates one artifact distribution: validate the node, create a durable
// record, hand the bytes off to device-sync-hub (the DirectedDelivery seam), and
// track the resulting status — bridge moves no bytes of its own.
type Service interface {
	// Distribute ships an artifact to a node via device-sync-hub directed
	// delivery, recording a durable distribution. On a dry-run it validates and
	// short-circuits before recording or delivering anything.
	Distribute(ctx context.Context, in DistributeInput) (Decision, error)

	// GetDistribution returns one distribution by id.
	GetDistribution(ctx context.Context, id string) (Distribution, error)

	// ListDistributions returns distributions newest-first, narrowed by filter.
	ListDistributions(ctx context.Context, filter ListFilter) ([]Distribution, error)
}

type service struct {
	repo     Repository
	nodes    NodeReader
	delivery DirectedDelivery
	clock    clock.Clock
}

// NewService constructs the production Service.
func NewService(repo Repository, nodes NodeReader, delivery DirectedDelivery, clk clock.Clock) Service {
	return &service{repo: repo, nodes: nodes, delivery: delivery, clock: clk}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Distribute(ctx context.Context, in DistributeInput) (Decision, error) {
	nodeID := trim(in.NodeID)
	source := trim(in.SourceRef)
	dest := trim(in.DestinationPath)
	if nodeID == "" {
		return Decision{}, ErrInvalidDistribution{Field: "node_id", Reason: "required"}
	}
	if source == "" {
		return Decision{}, ErrInvalidDistribution{Field: "source_ref", Reason: "required"}
	}
	if dest == "" {
		return Decision{}, ErrInvalidDistribution{Field: "destination_path", Reason: "required"}
	}

	// Resolve + validate the target node.
	node, err := s.nodes.GetTarget(ctx, nodeID)
	if err != nil {
		var notFound ErrNodeNotFound
		if errors.As(err, &notFound) {
			return Decision{}, notFound
		}
		return Decision{}, err
	}
	if node.Revoked {
		return Decision{}, ErrNodeRevoked{ID: nodeID}
	}

	// Dry-run: validated, but nothing recorded and nothing delivered.
	if in.DryRun {
		return Decision{DryRun: true, Status: StatusPending}, nil
	}

	// Record the durable distribution before handing off, so a delivery that
	// fails still leaves a trail.
	dist, err := s.repo.Create(ctx, Distribution{
		NodeID:          nodeID,
		Name:            trim(in.Name),
		SourceRef:       source,
		DestinationPath: dest,
		Status:          StatusPending,
	})
	if err != nil {
		return Decision{}, err
	}

	// Hand the bytes off to device-sync-hub. Bridge moves nothing itself.
	result, derr := s.delivery.Deliver(ctx, DeliveryRequest{
		NodeID: nodeID, Name: trim(in.Name), SourceRef: source, DestinationPath: dest,
	})
	if derr != nil {
		updated, _ := s.repo.UpdateStatus(ctx, dist.ID, StatusFailed, "", derr.Error())
		return Decision{DistributionID: dist.ID, Status: updated.Status}, ErrDeliveryFailed{NodeID: nodeID}
	}

	status := StatusPending
	if result.Delivered {
		status = StatusDelivered
	}
	updated, err := s.repo.UpdateStatus(ctx, dist.ID, status, result.Ref, result.Detail)
	if err != nil {
		return Decision{}, err
	}

	return Decision{
		DistributionID: updated.ID,
		Status:         updated.Status,
		DeliveryRef:    updated.DeliveryRef,
	}, nil
}

func (s *service) GetDistribution(ctx context.Context, id string) (Distribution, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) ListDistributions(ctx context.Context, filter ListFilter) ([]Distribution, error) {
	return s.repo.List(ctx, filter)
}
