package pairingclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"vrooli-bridge/agent/internal/nodecred"

	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
	pairingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing/pairing_v1connect"
)

func TestJoinDisplaysWordsAndWaitsForApproval(t *testing.T) {
	cred, err := nodecred.LoadOrCreate(t.TempDir() + "/node.key")
	if err != nil {
		t.Fatal(err)
	}
	var polls atomic.Int32
	mux := http.NewServeMux()
	mux.Handle(pairingconnect.PairingServiceRequestPairingProcedure, connect.NewUnaryHandler(
		pairingconnect.PairingServiceRequestPairingProcedure,
		func(_ context.Context, req *connect.Request[pairingv1.RequestPairingRequest]) (*connect.Response[pairingv1.RequestPairingResponse], error) {
			if req.Msg.GetNodePublicKey() != cred.PublicKeyBase64() {
				t.Fatalf("node public key = %q, want generated credential", req.Msg.GetNodePublicKey())
			}
			return connect.NewResponse(&pairingv1.RequestPairingResponse{RequestId: "req-1", ConfirmationWords: []string{"amber", "orbit", "cedar"}}), nil
		},
	))
	mux.Handle(pairingconnect.PairingServiceGetPairingRequestProcedure, connect.NewUnaryHandler(
		pairingconnect.PairingServiceGetPairingRequestProcedure,
		func(_ context.Context, _ *connect.Request[pairingv1.GetPairingRequestRequest]) (*connect.Response[pairingv1.GetPairingRequestResponse], error) {
			status := pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_PENDING
			if polls.Add(1) > 1 {
				status = pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_APPROVED
			}
			return connect.NewResponse(&pairingv1.GetPairingRequestResponse{
				Request:               &pairingv1.PairingRequest{Id: "req-1", Status: status, NodeId: "node-1", ConfirmationWords: []string{"amber", "orbit", "cedar"}},
				ControlPlanePublicKey: "control-plane-key",
			}), nil
		},
	))
	server := httptest.NewServer(mux)
	defer server.Close()

	var displayed []string
	result, err := (Client{BaseURL: server.URL, PollEvery: time.Millisecond, Display: func(words []string) { displayed = append([]string(nil), words...) }}).Join(
		context.Background(), cred, Facts{Name: "scratch-mac", OS: "darwin", Arch: "arm64"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequestID != "req-1" || result.NodeID != "node-1" || result.ControlPlaneKey != "control-plane-key" {
		t.Fatalf("join result = %+v", result)
	}
	if len(displayed) != 3 || displayed[0] != "amber" || polls.Load() < 2 {
		t.Fatalf("displayed=%v polls=%d, want three words and a pending poll", displayed, polls.Load())
	}
}

func TestJoinRejectsMissingCredential(t *testing.T) {
	if _, err := (Client{BaseURL: "http://bridge.test"}).Join(context.Background(), nil, Facts{}); err == nil {
		t.Fatal("missing credential was accepted")
	}
}
