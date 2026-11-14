package baseusdc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
	walletpkg "treasury/internal/wallet"
)

const (
	Network       = "eip155:8453"
	ChainID       = uint64(8453)
	USDCAddress   = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	USDCDecimals  = uint32(6)
	TransferAsset = Network + "/erc20:" + USDCAddress
)

var (
	ErrPending         = errors.New("Base transaction is not included")
	ErrFailed          = errors.New("Base transaction receipt reports failure")
	ErrWrongNetwork    = errors.New("Base RPC chain identity mismatch")
	ErrNoTransfer      = errors.New("no positive native-USDC transfer to the Treasury wallet")
	ErrNotCanonical    = errors.New("Base receipt does not match the provider's canonical block")
	ErrRPCFinality     = errors.New("Base RPC finalized head is unavailable")
	ErrRPCUnavailable  = errors.New("Base RPC is unavailable")
	evAddress          = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	evHash             = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	evData32           = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	providerIDPattern  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,63}$`)
	transferEventTopic = eventTopic("Transfer(address,address,uint256)")
	baseUSDCAssetID    = TransferAsset
)

type Verifier struct {
	rpc      RPC
	provider string
	now      func() time.Time
}

type Observation struct {
	EconomicKey          string    `json:"economic_key"`
	InclusionRevision    string    `json:"inclusion_revision"`
	WalletID             string    `json:"wallet_id"`
	Network              string    `json:"network"`
	AssetID              string    `json:"asset_id"`
	Decimals             uint32    `json:"decimals"`
	From                 string    `json:"from"`
	To                   string    `json:"to"`
	AtomicAmount         string    `json:"atomic_amount"`
	TransactionHash      string    `json:"transaction_hash"`
	LogIndex             uint64    `json:"log_index"`
	BlockNumber          uint64    `json:"block_number"`
	BlockHash            string    `json:"block_hash"`
	BlockTime            time.Time `json:"block_time"`
	Finality             string    `json:"finality"`
	FinalizedHeadNumber  uint64    `json:"finalized_head_number"`
	FinalizedHeadHash    string    `json:"finalized_head_hash"`
	ProviderID           string    `json:"provider_id"`
	ObservedAt           time.Time `json:"observed_at"`
	WalletAddressBinding string    `json:"wallet_address_binding"`
}

type receipt struct {
	Status          string `json:"status"`
	TransactionHash string `json:"transactionHash"`
	BlockNumber     string `json:"blockNumber"`
	BlockHash       string `json:"blockHash"`
	Logs            []log  `json:"logs"`
}

type log struct {
	Address         string   `json:"address"`
	Topics          []string `json:"topics"`
	Data            string   `json:"data"`
	TransactionHash string   `json:"transactionHash"`
	BlockNumber     string   `json:"blockNumber"`
	BlockHash       string   `json:"blockHash"`
	LogIndex        string   `json:"logIndex"`
	Removed         bool     `json:"removed"`
}

type block struct {
	Number    string `json:"number"`
	Hash      string `json:"hash"`
	Timestamp string `json:"timestamp"`
}

func NewVerifier(rpc RPC, providerID string, now func() time.Time) (*Verifier, error) {
	if rpc == nil || !providerIDPattern.MatchString(strings.TrimSpace(providerID)) {
		return nil, errors.New("Base RPC and provider ID are required")
	}
	if now == nil {
		now = time.Now
	}
	return &Verifier{rpc: rpc, provider: strings.TrimSpace(providerID), now: now}, nil
}

// VerifyWalletTransaction reads an EVM receipt and Base's finalized head. It
// proves only that this RPC currently reports exact native-USDC Transfer logs
// to a persisted Treasury wallet address. It never signs, writes a ledger
// posting, treats a balance as income, values USDC as dollars, or allocates.
func (v *Verifier) VerifyWalletTransaction(ctx context.Context, wallet walletpkg.Wallet, txHash string) ([]Observation, error) {
	if v == nil || v.rpc == nil {
		return nil, errors.New("Base receipt verifier is unavailable")
	}
	if strings.TrimSpace(wallet.ID) == "" || wallet.Network != Network || !evAddress.MatchString(wallet.Address) || wallet.CustodyReference == "" || !evHash.MatchString(txHash) {
		return nil, errors.New("verified Treasury Base wallet binding and exact transaction hash are required")
	}

	var chainID string
	if err := v.rpc.Call(ctx, "eth_chainId", []any{}, &chainID); err != nil {
		return nil, fmt.Errorf("%w: chain identity read failed", ErrRPCUnavailable)
	}
	chain, err := parseQuantity(chainID, "chain ID")
	if err != nil || !chain.IsUint64() || chain.Uint64() != ChainID {
		return nil, ErrWrongNetwork
	}

	canonicalTx := strings.ToLower(txHash)
	var txReceipt *receipt
	if err := v.rpc.Call(ctx, "eth_getTransactionReceipt", []any{canonicalTx}, &txReceipt); err != nil {
		return nil, fmt.Errorf("%w: transaction receipt read failed", ErrRPCUnavailable)
	}
	if txReceipt == nil {
		return nil, ErrPending
	}
	if !evHash.MatchString(txReceipt.TransactionHash) || strings.ToLower(txReceipt.TransactionHash) != canonicalTx {
		return nil, errors.New("Base receipt transaction hash does not match the request")
	}
	status, err := parseQuantity(txReceipt.Status, "receipt status")
	if err != nil {
		return nil, err
	}
	if status.Cmp(big.NewInt(1)) != 0 {
		return nil, ErrFailed
	}
	receiptBlock, err := parseQuantity(txReceipt.BlockNumber, "receipt block number")
	if err != nil || !receiptBlock.IsUint64() || !evHash.MatchString(txReceipt.BlockHash) {
		return nil, errors.New("Base receipt block identity is invalid")
	}
	receiptBlockNumber := receiptBlock.Uint64()
	receiptBlockHash := strings.ToLower(txReceipt.BlockHash)
	walletAddress := strings.ToLower(wallet.Address)

	seenIndexes := make(map[uint64]struct{}, len(txReceipt.Logs))
	observations := make([]Observation, 0, len(txReceipt.Logs))
	for _, item := range txReceipt.Logs {
		if !evHash.MatchString(item.TransactionHash) || strings.ToLower(item.TransactionHash) != canonicalTx ||
			item.BlockHash == "" || strings.ToLower(item.BlockHash) != receiptBlockHash || item.BlockNumber == "" {
			return nil, errors.New("Base receipt log disagrees with its transaction or block identity")
		}
		logBlock, parseErr := parseQuantity(item.BlockNumber, "log block number")
		if parseErr != nil || !logBlock.IsUint64() || logBlock.Uint64() != receiptBlockNumber {
			return nil, errors.New("Base receipt log block number mismatch")
		}
		logIndex, parseErr := parseQuantity(item.LogIndex, "log index")
		if parseErr != nil || !logIndex.IsUint64() {
			return nil, errors.New("Base receipt log index is invalid")
		}
		index := logIndex.Uint64()
		if _, exists := seenIndexes[index]; exists {
			return nil, errors.New("Base receipt contains duplicate log indexes")
		}
		seenIndexes[index] = struct{}{}
		if item.Removed {
			return nil, ErrNotCanonical
		}
		if !strings.EqualFold(item.Address, USDCAddress) {
			continue
		}
		if len(item.Topics) == 0 || !strings.EqualFold(item.Topics[0], transferEventTopic) {
			continue
		}
		if len(item.Topics) != 3 || !validIndexedAddress(item.Topics[1]) || !validIndexedAddress(item.Topics[2]) || !evData32.MatchString(item.Data) {
			return nil, errors.New("native-USDC Transfer log encoding is malformed")
		}
		if !strings.EqualFold(topicAddress(item.Topics[2]), walletAddress) {
			continue
		}
		amount, ok := new(big.Int).SetString(strings.TrimPrefix(item.Data, "0x"), 16)
		if !ok || amount.Sign() <= 0 {
			continue
		}
		observations = append(observations, Observation{
			EconomicKey:          fmt.Sprintf("%s/%s/%s/log/%d/%s", Network, USDCAddress, canonicalTx, index, walletAddress),
			InclusionRevision:    canonicalTx + "/log/" + strconv.FormatUint(index, 10) + "@" + receiptBlockHash,
			WalletID:             wallet.ID,
			Network:              Network,
			AssetID:              baseUSDCAssetID,
			Decimals:             USDCDecimals,
			From:                 topicAddress(item.Topics[1]),
			To:                   walletAddress,
			AtomicAmount:         amount.String(),
			TransactionHash:      canonicalTx,
			LogIndex:             index,
			BlockNumber:          receiptBlockNumber,
			BlockHash:            receiptBlockHash,
			ProviderID:           v.provider,
			ObservedAt:           v.now().UTC(),
			WalletAddressBinding: wallet.Address,
		})
	}
	if len(observations) == 0 {
		return nil, ErrNoTransfer
	}

	var finalizedHead *block
	if err := v.rpc.Call(ctx, "eth_getBlockByNumber", []any{"finalized", false}, &finalizedHead); err != nil {
		return nil, fmt.Errorf("%w: provider did not return a finalized head", ErrRPCFinality)
	}
	if finalizedHead == nil {
		return nil, fmt.Errorf("%w: empty result", ErrRPCFinality)
	}
	headNumber, err := parseQuantity(finalizedHead.Number, "finalized head number")
	if err != nil || !headNumber.IsUint64() || !evHash.MatchString(finalizedHead.Hash) {
		return nil, fmt.Errorf("%w: malformed finalized head identity", ErrRPCFinality)
	}
	var canonical *block
	blockNumberHex := "0x" + strconv.FormatUint(receiptBlockNumber, 16)
	if err := v.rpc.Call(ctx, "eth_getBlockByNumber", []any{blockNumberHex, false}, &canonical); err != nil {
		return nil, fmt.Errorf("%w: canonical block read failed", ErrRPCUnavailable)
	}
	if canonical == nil {
		return nil, errors.New("canonical Base block is unavailable")
	}
	canonicalNumber, err := parseQuantity(canonical.Number, "canonical block number")
	if err != nil || !canonicalNumber.IsUint64() || canonicalNumber.Uint64() != receiptBlockNumber || !evHash.MatchString(canonical.Hash) || strings.ToLower(canonical.Hash) != receiptBlockHash {
		return nil, ErrNotCanonical
	}
	blockTimestamp, err := parseQuantity(canonical.Timestamp, "block timestamp")
	if err != nil || !blockTimestamp.IsInt64() {
		return nil, errors.New("canonical Base block timestamp is invalid")
	}
	finalized := headNumber.Uint64() >= receiptBlockNumber
	for i := range observations {
		observations[i].BlockTime = time.Unix(blockTimestamp.Int64(), 0).UTC()
		observations[i].FinalizedHeadNumber = headNumber.Uint64()
		observations[i].FinalizedHeadHash = strings.ToLower(finalizedHead.Hash)
		if finalized {
			observations[i].Finality = "finalized"
		} else {
			observations[i].Finality = "included"
		}
	}
	return observations, nil
}

// PriorCanonical distinguishes a missing receipt from a provider-reported
// reorganization. A pending/missing receipt alone never authorizes a reversal;
// the previously accepted block must have a different canonical hash.
func (v *Verifier) PriorCanonical(ctx context.Context, prior []Observation) (bool, error) {
	if v == nil || v.rpc == nil || len(prior) == 0 {
		return false, errors.New("prior inclusion evidence is required")
	}
	// A failed receipt lookup can still be resolved from independent canonical
	// block evidence. Recheck the network before trusting that evidence; a
	// provider failure or a different chain never becomes an accounting debit.
	var chainID string
	if err := v.rpc.Call(ctx, "eth_chainId", []any{}, &chainID); err != nil {
		return false, fmt.Errorf("%w: prior inclusion chain read failed", ErrRPCUnavailable)
	}
	chain, err := parseQuantity(chainID, "chain ID")
	if err != nil || !chain.IsUint64() || chain.Uint64() != ChainID {
		return false, ErrWrongNetwork
	}
	for _, observation := range prior {
		var canonical *block
		if err := v.rpc.Call(ctx, "eth_getBlockByNumber", []any{"0x" + strconv.FormatUint(observation.BlockNumber, 16), false}, &canonical); err != nil {
			return false, fmt.Errorf("%w: prior canonical block read failed", ErrRPCUnavailable)
		}
		if canonical == nil {
			return false, fmt.Errorf("%w: prior canonical height is unavailable", ErrRPCUnavailable)
		}
		number, err := parseQuantity(canonical.Number, "prior canonical block number")
		if err != nil || !number.IsUint64() || number.Uint64() != observation.BlockNumber || !evHash.MatchString(canonical.Hash) {
			return false, errors.New("prior canonical block identity is invalid")
		}
		if !strings.EqualFold(canonical.Hash, observation.BlockHash) {
			return false, nil
		}
	}
	return true, nil
}

func parseQuantity(value, field string) (*big.Int, error) {
	if len(value) < 3 || !strings.HasPrefix(value, "0x") || (len(value) > 3 && value[2] == '0') {
		return nil, fmt.Errorf("%s is not a canonical hex quantity", field)
	}
	parsed, ok := new(big.Int).SetString(value[2:], 16)
	if !ok || parsed.Sign() < 0 {
		return nil, fmt.Errorf("%s is invalid", field)
	}
	return parsed, nil
}

func validIndexedAddress(topic string) bool {
	if !evHash.MatchString(topic) || !strings.EqualFold(topic[:26], "0x000000000000000000000000") {
		return false
	}
	return true
}

func topicAddress(topic string) string { return "0x" + strings.ToLower(topic[len(topic)-40:]) }

func eventTopic(signature string) string {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(signature))
	return "0x" + fmt.Sprintf("%x", h.Sum(nil))
}
