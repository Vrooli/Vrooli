package clients

import (
	"context"
	"fmt"

	"audio-tools/internal/httpc"
)

// UnavailableClient holds the shared transport identity for LPBS endpoints
// that are intentionally not available until the audio gateway ships.
type UnavailableClient struct {
	BaseURL string
	Doer    httpc.Doer
}

func newUnavailableClient(baseURL string) UnavailableClient {
	return UnavailableClient{BaseURL: baseURL, Doer: httpc.DefaultDoer()}
}

func (UnavailableClient) IsAvailable(context.Context) bool { return false }
func (UnavailableClient) Model() string                    { return "lpbs-default" }

// Unimplemented reports the shared temporary boundary for LPBS audio methods.
func Unimplemented(capability string) error {
	return fmt.Errorf("lpbs-%s: gateway endpoint not implemented (execute/lpbs-audio-gateway-endpoints)", capability)
}
