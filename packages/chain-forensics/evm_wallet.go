package chainforensics

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

var evmAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

type WalletTokenBalance struct {
	Contract   string `json:"contract"`
	Atomic     string `json:"atomic_quantity"`
	Scale      uint32 `json:"scale,omitempty"`
	ScaleKnown bool   `json:"scale_known"`
}

type EVMWalletSnapshot struct {
	ID           string               `json:"id"`
	Chain        Chain                `json:"chain"`
	Address      string               `json:"address"`
	Block        uint64               `json:"block_height"`
	BlockHash    string               `json:"block_hash"`
	Timestamp    string               `json:"timestamp"`
	NativeAsset  string               `json:"native_asset"`
	NativeAtomic string               `json:"native_atomic_quantity"`
	NativeScale  uint32               `json:"native_scale"`
	Tokens       []WalletTokenBalance `json:"tokens,omitempty"`
	SourceIDs    []string             `json:"source_ids"`
}

type CapturedEVMWalletSnapshot struct {
	Snapshot EVMWalletSnapshot
	Sources  []Source
}

// FetchWalletSnapshot pins native and selected ERC-20 balances to one block.
// It verifies the expected chain id and re-reads the pinned block afterward
// so a reorganization cannot silently mix balances from different states.
func (c *EVMRPCClient) FetchWalletSnapshot(ctx context.Context, chain Chain, address string, tokenContracts []string) (CapturedEVMWalletSnapshot, error) {
	if chain != ChainEthereum && chain != ChainBase {
		return CapturedEVMWalletSnapshot{}, fmt.Errorf("unsupported EVM chain %q", chain)
	}
	if !evmAddressPattern.MatchString(address) {
		return CapturedEVMWalletSnapshot{}, errors.New("wallet address must be a 20-byte 0x-prefixed value")
	}
	address = strings.ToLower(address)
	contracts := make([]string, 0, len(tokenContracts))
	seen := make(map[string]struct{}, len(tokenContracts))
	for _, contract := range tokenContracts {
		if !evmAddressPattern.MatchString(contract) {
			return CapturedEVMWalletSnapshot{}, fmt.Errorf("token contract %q must be a 20-byte 0x-prefixed value", contract)
		}
		contract = strings.ToLower(contract)
		if _, exists := seen[contract]; exists {
			return CapturedEVMWalletSnapshot{}, fmt.Errorf("duplicate token contract %s", contract)
		}
		seen[contract] = struct{}{}
		contracts = append(contracts, contract)
	}

	chainIDRaw, err := c.call(ctx, "eth_chainId", []any{})
	if err != nil {
		return CapturedEVMWalletSnapshot{}, err
	}
	chainIDText, err := decodeRPCString(chainIDRaw)
	if err != nil {
		return CapturedEVMWalletSnapshot{}, err
	}
	chainID, err := parseBigHexQuantity(chainIDText)
	if err != nil || chainID.Cmp(big.NewInt(expectedEVMChainID(chain))) != 0 {
		return CapturedEVMWalletSnapshot{}, fmt.Errorf("RPC chain id %s does not match %s", chainIDText, chain)
	}

	blockBody, blockResponse, err := c.callCaptured(ctx, "eth_getBlockByNumber", []any{"latest", false})
	if err != nil {
		return CapturedEVMWalletSnapshot{}, err
	}
	var block struct {
		Hash      string `json:"hash"`
		Number    string `json:"number"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(blockBody, &block); err != nil {
		return CapturedEVMWalletSnapshot{}, fmt.Errorf("decode EVM snapshot block: %w", err)
	}
	if !evmHashPattern.MatchString(block.Hash) {
		return CapturedEVMWalletSnapshot{}, errors.New("latest block is missing a valid hash")
	}
	blockNumber, err := parseEVMIndex(block.Number)
	if err != nil || blockNumber == 0 {
		return CapturedEVMWalletSnapshot{}, errors.New("latest block is missing a valid number")
	}
	blockTimestamp, err := parseEVMIndex(block.Timestamp)
	if err != nil || blockTimestamp == 0 || blockTimestamp > uint64(^uint64(0)>>1) {
		return CapturedEVMWalletSnapshot{}, errors.New("latest block is missing a valid timestamp")
	}
	blockTag := fmt.Sprintf("0x%x", blockNumber)
	retrieved := c.now().UTC()
	sources := []Source{sourceForBody("evm-wallet-block-"+string(chain)+"-"+strings.ToLower(block.Hash), "EVM wallet snapshot block", c.endpoint, "eth_getBlockByNumber(latest)", retrieved, blockBody, "Snapshot is pinned to this block hash and rechecked after all balance reads; exact result and HTTP response bytes are retained.", blockResponse)}
	snapshot := EVMWalletSnapshot{ID: "evm-wallet-" + string(chain) + "-" + address + "-" + strings.ToLower(block.Hash), Chain: chain, Address: address, Block: blockNumber, BlockHash: strings.ToLower(block.Hash), Timestamp: time.Unix(int64(blockTimestamp), 0).UTC().Format(time.RFC3339), NativeAsset: "ETH", NativeScale: 18}

	nativeBody, nativeResponse, err := c.callCaptured(ctx, "eth_getBalance", []any{address, blockTag})
	if err != nil {
		return CapturedEVMWalletSnapshot{}, err
	}
	nativeRaw, err := decodeRPCString(nativeBody)
	if err != nil {
		return CapturedEVMWalletSnapshot{}, fmt.Errorf("decode native wallet balance response: %w", err)
	}
	native, err := parseBigHexQuantity(nativeRaw)
	if err != nil {
		return CapturedEVMWalletSnapshot{}, fmt.Errorf("decode native wallet balance: %w", err)
	}
	snapshot.NativeAtomic = native.String()
	nativeSource := sourceForBody("evm-wallet-native-"+string(chain)+"-"+address+"-"+strings.ToLower(block.Hash), "EVM native wallet balance", c.endpoint, "eth_getBalance", retrieved, nativeBody, "Read-only balance pinned to block hash "+strings.ToLower(block.Hash)+"; exact result and HTTP response bytes are retained.", nativeResponse)
	sources = append(sources, nativeSource)
	snapshot.SourceIDs = append(snapshot.SourceIDs, sources[0].ID, nativeSource.ID)

	for _, contract := range contracts {
		balanceData := "0x70a08231" + strings.Repeat("0", 24) + strings.TrimPrefix(address, "0x")
		balanceBody, balanceResponse, err := c.callCaptured(ctx, "eth_call", []any{map[string]string{"to": contract, "data": balanceData}, blockTag})
		if err != nil {
			return CapturedEVMWalletSnapshot{}, fmt.Errorf("read ERC-20 balance for %s: %w", contract, err)
		}
		balanceRaw, err := decodeRPCString(balanceBody)
		if err != nil {
			return CapturedEVMWalletSnapshot{}, fmt.Errorf("decode ERC-20 balance response for %s: %w", contract, err)
		}
		balance, err := parseABIWord(balanceRaw)
		if err != nil {
			return CapturedEVMWalletSnapshot{}, fmt.Errorf("decode ERC-20 balance for %s: %w", contract, err)
		}
		token := WalletTokenBalance{Contract: contract, Atomic: balance.String()}
		balanceSource := sourceForBody("evm-wallet-token-"+string(chain)+"-"+address+"-"+contract+"-"+strings.ToLower(block.Hash), "EVM ERC-20 wallet balance", c.endpoint, "eth_call(balanceOf)", retrieved, balanceBody, "Read-only balanceOf result pinned to block hash "+strings.ToLower(block.Hash)+"; token decimals are queried separately; exact HTTP response bytes are retained.", balanceResponse)
		sources = append(sources, balanceSource)
		snapshot.SourceIDs = append(snapshot.SourceIDs, balanceSource.ID)

		decimalsData, decimalsResponse, err := c.callCaptured(ctx, "eth_call", []any{map[string]string{"to": contract, "data": "0x313ce567"}, blockTag})
		if err == nil {
			decimalsRaw, decodeErr := decodeRPCString(decimalsData)
			decimals, parseErr := parseABIWord(decimalsRaw)
			if decodeErr != nil {
				parseErr = decodeErr
			}
			if parseErr == nil && decimals.IsUint64() && decimals.Uint64() <= 255 {
				token.Scale = uint32(decimals.Uint64())
				token.ScaleKnown = true
				scaleSource := sourceForBody("evm-wallet-decimals-"+string(chain)+"-"+contract+"-"+strings.ToLower(block.Hash), "EVM ERC-20 decimals", c.endpoint, "eth_call(decimals)", retrieved, decimalsData, "Token decimals are contract-reported metadata pinned to block hash "+strings.ToLower(block.Hash)+"; exact HTTP response bytes are retained.", decimalsResponse)
				sources = append(sources, scaleSource)
				snapshot.SourceIDs = append(snapshot.SourceIDs, scaleSource.ID)
			}
		}
		snapshot.Tokens = append(snapshot.Tokens, token)
	}

	checkBody, checkResponse, err := c.callCaptured(ctx, "eth_getBlockByNumber", []any{blockTag, false})
	if err != nil {
		return CapturedEVMWalletSnapshot{}, err
	}
	var rechecked struct {
		Hash string `json:"hash"`
	}
	if err := json.Unmarshal(checkBody, &rechecked); err != nil || !strings.EqualFold(rechecked.Hash, block.Hash) {
		return CapturedEVMWalletSnapshot{}, errors.New("pinned block changed during wallet snapshot; no mixed snapshot was returned")
	}
	checkSource := sourceForBody("evm-wallet-block-check-"+string(chain)+"-"+strings.ToLower(block.Hash), "EVM wallet snapshot block recheck", c.endpoint, "eth_getBlockByNumber(pinned)", retrieved, checkBody, "Confirms the pinned block remained canonical across all balance reads; exact HTTP response bytes are retained.", checkResponse)
	sources = append(sources, checkSource)
	snapshot.SourceIDs = append(snapshot.SourceIDs, checkSource.ID)
	return CapturedEVMWalletSnapshot{Snapshot: snapshot, Sources: sources}, nil
}

func decodeRPCString(raw []byte) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("expected JSON-RPC string result: %w", err)
	}
	return value, nil
}

func expectedEVMChainID(chain Chain) int64 {
	if chain == ChainEthereum {
		return 1
	}
	return 8453
}

func parseBigHexQuantity(raw string) (*big.Int, error) {
	if !strings.HasPrefix(raw, "0x") || len(raw) < 3 {
		return nil, errors.New("expected a nonempty 0x-prefixed hexadecimal quantity")
	}
	value, ok := new(big.Int).SetString(raw[2:], 16)
	if !ok || value.Sign() < 0 || value.BitLen() > 256 {
		return nil, errors.New("invalid uint256 hexadecimal quantity")
	}
	return value, nil
}

func parseABIWord(raw string) (*big.Int, error) {
	if !strings.HasPrefix(raw, "0x") || len(raw) != 66 {
		return nil, errors.New("expected exactly one 32-byte ABI word")
	}
	if _, err := hex.DecodeString(raw[2:]); err != nil {
		return nil, errors.New("ABI word is not hexadecimal")
	}
	value, ok := new(big.Int).SetString(raw[2:], 16)
	if !ok {
		return nil, errors.New("invalid ABI word")
	}
	return value, nil
}
