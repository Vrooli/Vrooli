package resolver

import (
	"context"
	"errors"
)

var ErrClientUnsupported = errors.New("adguard home client is not configured")

type ConservativeAdGuardClient struct{}

func (ConservativeAdGuardClient) Check(context.Context, BackendConfig) (ClientStatus, error) {
	return ClientStatus{
		Status:           "configured_unverified",
		FilteringEnabled: false,
		Warnings:         []string{"AdGuard Home configuration is stored by secret reference, but no verified resource-backed client is connected yet."},
		Checks:           []string{"Backend configuration present.", "Filtering activity is not claimed until AdGuard Home confirms it."},
	}, nil
}

func (ConservativeAdGuardClient) PreviewUpstreams(_ context.Context, _ BackendConfig, upstreams []string) ([]string, error) {
	return []string{"Previewed upstream update; no resolver changes were applied.", "Upstreams requested: " + joinUpstreams(upstreams)}, nil
}

func (ConservativeAdGuardClient) UpdateUpstreams(context.Context, BackendConfig, []string) (ClientStatus, []string, error) {
	return ClientStatus{}, nil, ErrClientUnsupported
}
