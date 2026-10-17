// Package operation is the CLI surface of durable cloud operations: get,
// wait, resume, cancel and list over the generated OperationsService client.
// The JSON output is the server's standing unchanged (proto JSON); the human
// output is the shared cli-core/operationstanding view.
package operation

import (
	"context"
	"time"

	"connectrpc.com/connect"
	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations/operationsv1connect"

	"scenario-to-cloud/cli/internal/transport"
)

// DefaultWaitTimeout is the observer bound sent when the caller gives none.
const DefaultWaitTimeout = 300 * time.Second

// waitSlack widens the client deadline beyond the server bound so the
// observer never gives up before the owner answers still_pending.
const waitSlack = 30 * time.Second

// Client provides typed access to operations.
type Client struct {
	tr      transport.Transport
	service operationsv1connect.OperationsServiceClient
}

// NewClient builds the client over the shared transport.
func NewClient(tr transport.Transport) *Client {
	return &Client{tr: tr, service: operationsv1connect.NewOperationsServiceClient(tr.HTTP, tr.BaseURL)}
}

// Get returns the standing of one operation.
func (c *Client) Get(ctx context.Context, id string) (*operationsv1.OperationStanding, error) {
	resp, err := c.service.GetOperation(ctx, connect.NewRequest(&operationsv1.GetOperationRequest{OperationId: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// Wait blocks server-side for up to timeout (0 = the owner's default). The
// call is made on a bounded HTTP client covering the server bound plus
// slack. A still_pending answer is the observer bound elapsing; the
// operation is unchanged.
func (c *Client) Wait(ctx context.Context, id string, timeout time.Duration) (*operationsv1.OperationStanding, error) {
	service := c.service
	seconds := int32(0)
	if timeout > 0 {
		seconds = int32(timeout / time.Second)
		if c.tr.Bounded != nil {
			service = operationsv1connect.NewOperationsServiceClient(c.tr.Bounded(timeout+waitSlack), c.tr.BaseURL)
		}
	}
	resp, err := service.WaitOperation(ctx, connect.NewRequest(&operationsv1.WaitOperationRequest{OperationId: id, TimeoutSeconds: seconds}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// Cancel records the cancellation intent.
func (c *Client) Cancel(ctx context.Context, id string) (*operationsv1.OperationStanding, error) {
	resp, err := c.service.CancelOperation(ctx, connect.NewRequest(&operationsv1.CancelOperationRequest{OperationId: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// List returns the operations of a deployment.
func (c *Client) List(ctx context.Context, deploymentID string) (*operationsv1.ListDeploymentOperationsResponse, error) {
	resp, err := c.service.ListDeploymentOperations(ctx, connect.NewRequest(&operationsv1.ListDeploymentOperationsRequest{DeploymentId: deploymentID}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
