package inventory

import "context"

type ConservativeResolverSource struct{}

func (ConservativeResolverSource) Discover(context.Context) ([]Observation, []string, error) {
	return nil, []string{"Resolver-derived client discovery is not configured yet; no device identities were invented."}, ErrUnsupported
}
