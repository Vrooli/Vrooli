package dns

import (
	"context"
	"errors"
)

// FakeResolver provides a controllable Resolver for testing.
// Configure expected hosts and their IPs, or set Err for failure scenarios.
type FakeResolver struct {
	// Hosts maps hostnames to IP addresses
	Hosts map[string][]string
	// Err is returned for all lookups if set
	Err error
	// Errs is returned for matching hostnames (takes precedence over Err)
	Errs map[string]error
}

// Ensure FakeResolver implements Resolver at compile time.
var _ Resolver = (*FakeResolver)(nil)

// LookupHost returns configured IPs for the host, or an error if not found.
func (f *FakeResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	if err, ok := f.Errs[host]; ok {
		return nil, err
	}
	if f.Err != nil {
		return nil, f.Err
	}
	ips, ok := f.Hosts[host]
	if !ok {
		return nil, errors.New("no such host")
	}
	return ips, nil
}
