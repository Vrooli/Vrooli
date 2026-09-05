package vroolicli

import (
	"context"
	"fmt"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// CredentialsList returns credential addresses and non-secret presence state.
func (c *Client) CredentialsList(ctx context.Context) (*cliv1.CliCredentialList, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	out, err := c.run(ctx, "credentials", "list", "--format", "json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.CliCredentialList{})
	if err != nil {
		return nil, fmt.Errorf("credentials list: %w", err)
	}
	return resp, nil
}

// CredentialsStatus returns non-secret status for one credential address.
func (c *Client) CredentialsStatus(ctx context.Context, identity, field string) (*cliv1.CliCredentialStatus, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	if identity == "" || field == "" {
		return nil, fmt.Errorf("credential identity and field are required")
	}
	out, err := c.run(ctx, "credentials", "status", "--identity", identity, "--field", field, "--format", "json")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.CliCredentialStatus{})
	if err != nil {
		return nil, fmt.Errorf("credentials status: %w", err)
	}
	return resp, nil
}

// BreakGlassStatus returns break-glass readiness metadata without key material.
func (c *Client) BreakGlassStatus(ctx context.Context) (*cliv1.CliBreakGlassStatus, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	out, err := c.run(ctx, "--json", "break-glass", "status")
	if err != nil {
		return nil, err
	}
	resp, err := decode(out, &cliv1.CliBreakGlassStatus{})
	if err != nil {
		return nil, fmt.Errorf("break-glass status: %w", err)
	}
	return resp, nil
}
