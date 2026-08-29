// Package pairingclient owns the node-side, no-pre-shared-code enrollment
// flow. It sends the node's public identity once, displays the key-derived
// words, and polls the public request status until the owner decides.
package pairingclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"vrooli-bridge/agent/internal/nodecred"

	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
	pairingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing/pairing_v1connect"
)

var ErrRejected = errors.New("pairing request was rejected")

type Facts struct {
	Name         string
	OS           string
	Arch         string
	Endpoint     string
	Capabilities []string
}

type Result struct {
	RequestID         string
	NodeID            string
	ControlPlaneKey   string
	ConfirmationWords []string
}

type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	PollEvery  time.Duration
	Display    func([]string)
}

func (c Client) Join(ctx context.Context, cred *nodecred.Credential, facts Facts) (Result, error) {
	if cred == nil {
		return Result{}, errors.New("pairing: node credential is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		return Result{}, errors.New("pairing: control-plane URL is required")
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{}
	}
	api := pairingconnect.NewPairingServiceClient(hc, baseURL)
	requested, err := api.RequestPairing(ctx, connect.NewRequest(&pairingv1.RequestPairingRequest{
		NodePublicKey: cred.PublicKeyBase64(),
		Name:          facts.Name, Os: facts.OS, Arch: facts.Arch, Endpoint: facts.Endpoint,
		Capabilities: append([]string(nil), facts.Capabilities...),
	}))
	if err != nil {
		return Result{}, fmt.Errorf("request pairing: %w", err)
	}
	if requested == nil || requested.Msg == nil || strings.TrimSpace(requested.Msg.GetRequestId()) == "" {
		return Result{}, errors.New("pairing: server returned no request id")
	}
	result := Result{RequestID: requested.Msg.GetRequestId(), ConfirmationWords: append([]string(nil), requested.Msg.GetConfirmationWords()...)}
	if len(result.ConfirmationWords) > 0 && c.Display != nil {
		c.Display(result.ConfirmationWords)
	}

	interval := c.PollEvery
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		status, err := api.GetPairingRequest(ctx, connect.NewRequest(&pairingv1.GetPairingRequestRequest{RequestId: result.RequestID}))
		if err != nil {
			return Result{}, fmt.Errorf("poll pairing request %s: %w", result.RequestID, err)
		}
		if status == nil || status.Msg == nil || status.Msg.GetRequest() == nil {
			return Result{}, errors.New("pairing: server returned no request state")
		}
		request := status.Msg.GetRequest()
		if words := request.GetConfirmationWords(); len(words) > 0 && !sameWords(result.ConfirmationWords, words) {
			result.ConfirmationWords = append([]string(nil), words...)
			if c.Display != nil {
				c.Display(result.ConfirmationWords)
			}
		}
		switch request.GetStatus() {
		case pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_APPROVED:
			if strings.TrimSpace(request.GetNodeId()) == "" || strings.TrimSpace(status.Msg.GetControlPlanePublicKey()) == "" {
				return Result{}, errors.New("pairing: approval did not return node identity and control-plane key")
			}
			result.NodeID = request.GetNodeId()
			result.ControlPlaneKey = status.Msg.GetControlPlanePublicKey()
			return result, nil
		case pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_REJECTED:
			return Result{}, ErrRejected
		}
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func sameWords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(strings.TrimSpace(a[i]), strings.TrimSpace(b[i])) {
			return false
		}
	}
	return true
}
