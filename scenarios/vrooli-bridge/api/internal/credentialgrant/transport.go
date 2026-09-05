package credentialgrant

import (
	"fmt"

	"github.com/google/uuid"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	"github.com/vrooli/vrooli/packages/proto/sealing"

	"vrooli-bridge/internal/channelsign"
)

// SealPush is the only control-plane construction path for credential frames.
// The caller supplies a value from the authority; this function returns only
// the sealed frame and never records or logs the plaintext.
func SealPush(signer channelsign.Signer, grant Grant, nodeID string, recipientPublic []byte, value string) ([]byte, error) {
	if grant.NodeID != nodeID {
		return nil, fmt.Errorf("credential push: grant node mismatch")
	}
	if err := validateGrantMetadata(grant); err != nil {
		return nil, fmt.Errorf("credential push: %w", err)
	}
	aad := sealing.CredentialContext(nodeID, grant.LogicalID, grant.Field, grant.Generation)
	sealed, err := sealing.Seal(recipientPublic, []byte(value), aad)
	if err != nil {
		return nil, fmt.Errorf("credential push: seal: %w", err)
	}
	frame := &channelv1.ServerFrame{
		FrameId: uuid.NewString(),
		Payload: &channelv1.ServerFrame_CredentialPush{CredentialPush: &channelv1.CredentialPush{
			GrantId: grant.ID, NodeId: grant.NodeID, LogicalId: grant.LogicalID, Field: grant.Field,
			Generation: grant.Generation, Retention: string(grant.Retention), SealedValue: sealed, Aad: aad,
		}},
	}
	payload, err := channelsign.Marshal(signer, frame)
	if err != nil {
		return nil, fmt.Errorf("credential push: sign: %w", err)
	}
	return payload, nil
}

// GrantFrame distributes metadata-only node consent. It is sent before a
// CredentialPush and is itself covered by the control-plane signature.
func GrantFrame(signer channelsign.Signer, grant Grant, nodeID string) ([]byte, error) {
	if grant.NodeID != nodeID {
		return nil, fmt.Errorf("credential grant: invalid node binding")
	}
	if err := validateGrantMetadata(grant); err != nil {
		return nil, fmt.Errorf("credential grant: %w", err)
	}
	frame := &channelv1.ServerFrame{
		FrameId: uuid.NewString(),
		Payload: &channelv1.ServerFrame_CredentialGrant{CredentialGrant: &channelv1.CredentialGrant{
			GrantId: grant.ID, NodeId: grant.NodeID, LogicalId: grant.LogicalID, Field: grant.Field,
			Class: string(grant.Class), Retention: string(grant.Retention), Generation: grant.Generation,
		}},
	}
	return channelsign.Marshal(signer, frame)
}

// PurgeFrame asks a reachable node to remove only grant-owned addresses.
// Local-only credentials are never named by this frame.
func PurgeFrame(signer channelsign.Signer, nodeID string, addresses []string) ([]byte, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("credential purge: node id is required")
	}
	frame := &channelv1.ServerFrame{
		FrameId: uuid.NewString(),
		Payload: &channelv1.ServerFrame_CredentialPurge{CredentialPurge: &channelv1.CredentialPurge{
			NodeId: nodeID, Addresses: append([]string(nil), addresses...),
		}},
	}
	return channelsign.Marshal(signer, frame)
}
