// Package providerdemand owns the scoped demand-lease lifecycle shared by
// scenario validation providers.
package providerdemand

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/vrooli/api-core/demand"
)

var sequence uint64

func Acquire(ctx context.Context, client demand.LeaseClient, request demand.AcquireRequest, defaultConsumerID, releaseReason, acquireError string) (func(), error) {
	if client == nil {
		return nil, nil
	}
	consumerID := strings.TrimSpace(request.ConsumerID)
	if consumerID == "" {
		consumerID = defaultConsumerID
	}
	request.RequestID = fmt.Sprintf("validate:%s:%d", strings.TrimSpace(request.RequestID), atomic.AddUint64(&sequence, 1))
	request.ConsumerID = consumerID
	request.LeaseID = demand.StableLeaseID(consumerID, request.Scenario, request.RequestID)
	request.Kind = demand.KindDependency
	_, release, err := demand.AcquireScoped(ctx, client, request, releaseReason)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", acquireError, err)
	}
	return release, nil
}
