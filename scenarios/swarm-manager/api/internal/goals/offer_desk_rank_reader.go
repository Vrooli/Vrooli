package goals

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	offersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
)

// OfferDeskRankReader resolves the current release rank from Offer Desk at
// read time. It deliberately does not copy rank into Swarm Manager storage.
type OfferDeskRankReader struct {
	client offersconnect.CatalogServiceClient
	ladder offersconnect.ReleaseLadderServiceClient
}

func NewOfferDeskRankReader(ctx context.Context) (*OfferDeskRankReader, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "offer-desk")
	if err != nil {
		return nil, err
	}
	return NewOfferDeskRankReaderAt(base, http.DefaultClient), nil
}

func NewOfferDeskRankReaderAt(base string, client *http.Client) *OfferDeskRankReader {
	if client == nil {
		client = http.DefaultClient
	}
	return &OfferDeskRankReader{
		client: offersconnect.NewCatalogServiceClient(client, strings.TrimRight(base, "/")),
		ladder: offersconnect.NewReleaseLadderServiceClient(client, strings.TrimRight(base, "/")),
	}
}

// DerivedUrgency resolves an enabling deliverable's urgency from the typed
// release graph. It is deliberately read-time data, never goal persistence.
func (r *OfferDeskRankReader) DerivedUrgency(name string) (int, error) {
	if r == nil || r.ladder == nil {
		return 0, fmt.Errorf("offer desk release ladder is unavailable")
	}
	response, err := r.ladder.GetEnablingDeliverables(context.Background(), connect.NewRequest(&offersv1.ReleaseLadderRequest{}))
	if err != nil {
		return 0, err
	}
	for _, item := range response.Msg.GetEnabling() {
		if item.GetNode().GetName() == name {
			return int(item.GetDerivedUrgency()), nil
		}
	}
	return 0, fmt.Errorf("enabling deliverable %q was not found", name)
}

func (r *OfferDeskRankReader) ReleaseRank(name string) (int, error) {
	nodes, err := r.deliverables()
	if err != nil {
		return 0, err
	}
	for _, node := range nodes {
		if node.GetName() == name {
			return int(node.GetReleaseRank()), nil
		}
	}
	return 0, fmt.Errorf("deliverable %q was not found", name)
}

func (r *OfferDeskRankReader) MaxReleaseRank() (int, error) {
	nodes, err := r.deliverables()
	if err != nil {
		return 0, err
	}
	max := 0
	for _, node := range nodes {
		if int(node.GetReleaseRank()) > max {
			max = int(node.GetReleaseRank())
		}
	}
	return max, nil
}

func (r *OfferDeskRankReader) ValidateDeliverable(name string) (bool, error) {
	nodes, err := r.deliverables()
	if err != nil {
		return false, err
	}
	for _, node := range nodes {
		if node.GetName() == name {
			return true, nil
		}
	}
	return false, nil
}

func (r *OfferDeskRankReader) deliverables() ([]*offersv1.Node, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("offer desk rank reader is unavailable")
	}
	response, err := r.client.ListNodes(context.Background(), connect.NewRequest(&offersv1.ListNodesRequest{Kind: offersv1.NodeKind_DELIVERABLE}))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetNodes(), nil
}
