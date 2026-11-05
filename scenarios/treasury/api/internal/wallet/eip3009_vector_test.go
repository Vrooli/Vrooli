package wallet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	secp "github.com/go-webauthn/x/crypto/secp256k1"
	"github.com/go-webauthn/x/crypto/secp256k1/ecdsa"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func vectorKeccak(parts ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, part := range parts {
		h.Write(part)
	}
	return h.Sum(nil)
}

func vectorWord(value uint64) []byte {
	return new(big.Int).SetUint64(value).FillBytes(make([]byte, 32))
}

func vectorAddress(t *testing.T, address string) []byte {
	t.Helper()
	b, err := hex.DecodeString(address[2:])
	require.NoError(t, err)
	require.Len(t, b, 20)
	return append(make([]byte, 12), b...)
}

func TestExportPublicZeroValueERC3009Simulation(t *testing.T) {
	input, output := os.Getenv("TREASURY_PUBLIC_ERC3009_DOMAIN"), os.Getenv("TREASURY_PUBLIC_ERC3009_OUTPUT")
	if input == "" || output == "" {
		t.Skip("test-only public zero-value fixture export not requested")
	}
	data, err := os.ReadFile(input)
	require.NoError(t, err)
	var domain struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		Token     string `json:"token"`
		ChainID   uint64 `json:"chain_id"`
		Separator string `json:"domain_separator"`
	}
	require.NoError(t, json.Unmarshal(data, &domain))
	require.EqualValues(t, 84532, domain.ChainID)
	scalar := make([]byte, 32)
	scalar[31] = 1
	key := secp.PrivKeyFromBytes(scalar)
	defer key.Zero()
	clear(scalar)
	address := Address(key.PubKey())
	require.Equal(t, "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf", address)
	domainHash := func(chain uint64) []byte {
		return vectorKeccak(vectorKeccak([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)")), vectorKeccak([]byte(domain.Name)), vectorKeccak([]byte(domain.Version)), vectorWord(chain), vectorAddress(t, domain.Token))
	}
	require.Equal(t, domain.Separator, "0x"+hex.EncodeToString(domainHash(84532)))
	typeHash := vectorKeccak([]byte("TransferWithAuthorization(address from,address to,uint256 value,uint256 validAfter,uint256 validBefore,bytes32 nonce)"))
	require.Equal(t, "7c7c6cdb67a18743f49ec6fa9b35f50d52ed05cbed4cc592e13b44501c1a2267", hex.EncodeToString(typeHash))
	selector := vectorKeccak([]byte("transferWithAuthorization(address,address,uint256,uint256,uint256,bytes32,uint8,bytes32,bytes32)"))[:4]
	now := time.Now().UTC()
	expires := uint64(now.Add(10 * time.Minute).Unix())
	nonce := sha256.Sum256([]byte("Vrooli PUBLIC ZERO VALUE simulation only " + strconv.FormatInt(now.UnixNano(), 10)))
	encode := func(chain, until uint64, signedTo, calledTo string) string {
		values := append(append(append(append(append(append([]byte{}, vectorAddress(t, address)...), vectorAddress(t, signedTo)...), vectorWord(0)...), vectorWord(0)...), vectorWord(until)...), nonce[:]...)
		digest := vectorKeccak([]byte{0x19, 0x01}, domainHash(chain), vectorKeccak(typeHash, values))
		sig := ecdsa.SignCompact(key, digest, false)
		require.True(t, sig[0] == 27 || sig[0] == 28)
		called := append(append(append(append(append(append([]byte{}, vectorAddress(t, address)...), vectorAddress(t, calledTo)...), vectorWord(0)...), vectorWord(0)...), vectorWord(until)...), nonce[:]...)
		called = append(called, vectorWord(uint64(sig[0]))...)
		called = append(called, sig[1:]...)
		return "0x" + hex.EncodeToString(append(append([]byte{}, selector...), called...))
	}
	fixture := map[string]any{"public_known_vector_do_not_fund": true, "test_network_only": true, "amount_atomic": "0", "from": address, "to": address, "chain_id": 84532, "token": domain.Token, "name": domain.Name, "version": domain.Version, "domain_separator": domain.Separator, "created_at": now.Format(time.RFC3339Nano), "valid_before": expires, "broadcasts": 0, "calls": []map[string]any{
		{"name": "correct_zero_value", "expected": "success", "data": encode(84532, expires, address, address)},
		{"name": "wrong_chain", "expected": "revert", "data": encode(8453, expires, address, address)},
		{"name": "changed_recipient", "expected": "revert", "data": encode(84532, expires, address, "0x1111111111111111111111111111111111111111")},
		{"name": "expired", "expected": "revert", "data": encode(84532, 1, address, address)},
	}}
	bytes, err := json.MarshalIndent(fixture, "", "  ")
	require.NoError(t, err)
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = file.Write(append(bytes, '\n'))
	require.NoError(t, err)
	require.NoError(t, file.Close())
	fmt.Println("known-public-key zero-value ERC3009 simulation fixture; no live key or transaction")
}
