// Package pairing is the Connect handler for the PairingService: one-touch
// bootstrap (issue/redeem), the request/approve fallback, and atomic revocation
// support. It translates the pairing proto wire types to the pairing domain's
// shapes and gates the owner-only RPCs through internal/auth. Node mutual-auth
// verification (the read side of the credentials this domain writes) lives in
// internal/nodeauth.
package pairing

import (
	"time"

	"vrooli-bridge/internal/pairing"
	"vrooli-bridge/pairingwords"

	"google.golang.org/protobuf/types/known/timestamppb"

	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
)

func timeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t.UTC())
}

func statusToProto(s pairing.RequestStatus) pairingv1.PairingRequestStatus {
	switch s {
	case pairing.RequestPending:
		return pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_PENDING
	case pairing.RequestApproved:
		return pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_APPROVED
	case pairing.RequestRejected:
		return pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_REJECTED
	default:
		return pairingv1.PairingRequestStatus_PAIRING_REQUEST_STATUS_UNSPECIFIED
	}
}

func requestToProto(r pairing.PairingRequest, confirmationWords []string) *pairingv1.PairingRequest {
	return &pairingv1.PairingRequest{
		Id:                r.ID,
		Name:              r.Name,
		Os:                r.OS,
		Arch:              r.Arch,
		Endpoint:          r.Endpoint,
		Capabilities:      append([]string(nil), r.Capabilities...),
		Status:            statusToProto(r.Status),
		CreatedAt:         timeToProto(r.CreatedAt),
		DecidedAt:         timeToProto(r.DecidedAt),
		NodeId:            r.NodeID,
		ConfirmationWords: append([]string(nil), confirmationWords...),
		KeyFingerprint:    pairingwords.Fingerprint(r.PublicKey),
	}
}
