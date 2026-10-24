// Package manageddns contains the scenario-to-cloud client for the typed DNS
// capability owned by tunnel-manager. Provider credentials stay with the
// tunnel-manager credential authority.
package manageddns

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"
)

type Client struct {
	config configconnect.ConfigServiceClient
}

func New(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	return &Client{config: configconnect.NewConfigServiceClient(http.DefaultClient, baseURL)}
}

func (c *Client) Ensure(ctx context.Context, profile, hostname, recordType, content, owner string, ttl int, proxied, dryRun bool) (string, bool, error) {
	resp, err := c.config.EnsureDNSRecord(ctx, connect.NewRequest(&configv1.EnsureDNSRecordRequest{
		ProviderProfile: profile, Hostname: hostname, Type: recordType, Content: content,
		Ttl: int32(ttl), Proxied: proxied, Owner: owner, DryRun: dryRun,
	}))
	if err != nil {
		return "", false, err
	}
	return resp.Msg.RecordId, resp.Msg.Created, nil
}

func (c *Client) Update(ctx context.Context, profile, hostname, recordType, content, owner string, ttl, priority int, proxied, dryRun bool) (string, error) {
	resp, err := c.config.UpdateDNSRecord(ctx, connect.NewRequest(&configv1.UpdateDNSRecordRequest{ProviderProfile: profile, Hostname: hostname, Type: recordType, Content: content, Ttl: int32(ttl), Priority: int32(priority), Proxied: proxied, Owner: owner, DryRun: dryRun}))
	if err != nil {
		return "", err
	}
	return resp.Msg.RecordId, nil
}

type SPFResult struct {
	RecordID   string
	Merged     string
	LookupCost int
	Changed    bool
}

func (c *Client) EnsureSPF(ctx context.Context, profile, hostname, mechanism, owner string, ttl int, dryRun bool) (SPFResult, error) {
	resp, err := c.config.EnsureSPFRecord(ctx, connect.NewRequest(&configv1.EnsureSPFRecordRequest{ProviderProfile: profile, Hostname: hostname, Mechanism: mechanism, Ttl: int32(ttl), Owner: owner, DryRun: dryRun}))
	if err != nil {
		return SPFResult{}, err
	}
	return SPFResult{RecordID: resp.Msg.RecordId, Merged: resp.Msg.MergedValue, LookupCost: int(resp.Msg.LookupCost), Changed: resp.Msg.Changed}, nil
}
