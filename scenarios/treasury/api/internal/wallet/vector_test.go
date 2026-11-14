package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	secp "github.com/go-webauthn/x/crypto/secp256k1"
	"github.com/go-webauthn/x/crypto/secp256k1/ecdsa"
	"github.com/stretchr/testify/require"
)

// This opt-in fixture exposes only PUBLIC vector material. The scalar1 key is
// universally known and must never be provisioned or advertised for deposits.
func TestExportPublicEcrecoverVector(t *testing.T) {
	path := os.Getenv("TREASURY_PUBLIC_VECTOR_OUTPUT")
	if path == "" {
		t.Skip("public-vector artifact export not requested")
	}
	scalar := make([]byte, 32)
	scalar[31] = 1
	key := secp.PrivKeyFromBytes(scalar)
	defer key.Zero()
	clear(scalar)
	w := Wallet{ID: "public-known-vector-do-not-fund", BookID: "isolated-public-fixture", OperatorIdentity: "public-test-vector", Network: "eip155:84532", Address: Address(key.PubKey()), PublicKey: hex.EncodeToString(key.PubKey().SerializeUncompressed()), CreatedAt: time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)}
	nonce := "public_vector_nonce_20261027"
	message := proofMessage(w, nonce)
	hash := personalHash(message)
	compact := ecdsa.SignCompact(key, hash, false)
	eth := append(append([]byte{}, compact[1:]...), compact[0])
	proof := Proof{Wallet: w, Message: message, Signature: "0x" + hex.EncodeToString(eth), Scheme: "eip191-personal-sign-fixed-research-v1"}
	require.True(t, Verify(proof, nonce))
	vword := make([]byte, 32)
	vword[31] = eth[64]
	calldata := append(append(append([]byte{}, hash...), vword...), eth[:64]...)
	out := map[string]any{"public_known_vector_do_not_fund": true, "proof": proof, "nonce": nonce, "hash": "0x" + hex.EncodeToString(hash), "ecrecover_calldata": "0x" + hex.EncodeToString(calldata), "expected_address": w.Address}
	data, err := json.MarshalIndent(out, "", "  ")
	require.NoError(t, err)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	require.NoError(t, err)
	_, err = file.Write(append(data, '\n'))
	require.NoError(t, err)
	require.NoError(t, file.Close())
	fmt.Println("public ecrecover vector retained; no wallet was provisioned")
}
