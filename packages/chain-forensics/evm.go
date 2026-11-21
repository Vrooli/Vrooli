package chainforensics

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ERC20TransferTopic is keccak256("Transfer(address,address,uint256)").
const ERC20TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

var evmHashPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

// EVMLog is the JSON-RPC log subset required to normalize an ERC-20 transfer.
// Timestamp must come from the containing block response; a log alone has no
// timestamp. The original response provenance is retained on the occurrence.
type EVMLog struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
	LogIndex         string   `json:"logIndex"`
}

// NormalizeERC20TransferLog converts one confirmed JSON-RPC ERC-20 Transfer
// log into an exact-quantity occurrence. The asset is identified by its
// contract address and the quantity remains in atomic token units (scale 0)
// until trusted token metadata is separately captured. It does not imply that
// either address has exclusive control of an entity.
func NormalizeERC20TransferLog(chain Chain, source Source, log EVMLog, blockTimestamp string) (Occurrence, error) {
	if chain != ChainEthereum && chain != ChainBase && chain != ChainArbitrum {
		return Occurrence{}, fmt.Errorf("ERC-20 transfers are not supported on chain %q", chain)
	}
	if err := validateEVMSource(source); err != nil {
		return Occurrence{}, err
	}
	contract, err := normalizeEVMAddress(log.Address)
	if err != nil {
		return Occurrence{}, fmt.Errorf("invalid ERC-20 contract: %w", err)
	}
	if len(log.Topics) != 3 || !strings.EqualFold(log.Topics[0], ERC20TransferTopic) {
		return Occurrence{}, errors.New("log is not a canonical ERC-20 Transfer event")
	}
	from, err := addressFromTopic(log.Topics[1])
	if err != nil {
		return Occurrence{}, fmt.Errorf("invalid Transfer sender: %w", err)
	}
	to, err := addressFromTopic(log.Topics[2])
	if err != nil {
		return Occurrence{}, fmt.Errorf("invalid Transfer recipient: %w", err)
	}
	amount, err := parseEVMQuantity(log.Data)
	if err != nil {
		return Occurrence{}, fmt.Errorf("invalid Transfer amount: %w", err)
	}
	block, err := parseEVMIndex(log.BlockNumber)
	if err != nil || block == 0 {
		return Occurrence{}, errors.New("Transfer log must have a confirmed block number")
	}
	logIndex, err := parseEVMIndex(log.LogIndex)
	if err != nil || logIndex > uint64(^uint32(0)) {
		return Occurrence{}, errors.New("Transfer log has an invalid log index")
	}
	if !evmHashPattern.MatchString(log.TransactionHash) {
		return Occurrence{}, errors.New("Transfer log has an invalid transaction hash")
	}
	if strings.TrimSpace(blockTimestamp) == "" {
		return Occurrence{}, errors.New("Transfer log is missing its block timestamp")
	}
	return Occurrence{
		ID:          fmt.Sprintf("%s:%s:log:%d", chain, strings.ToLower(log.TransactionHash), logIndex),
		Chain:       chain,
		Transaction: strings.ToLower(log.TransactionHash),
		Block:       block,
		Timestamp:   blockTimestamp,
		EventIndex:  uint32(logIndex),
		Direction:   "transfer",
		From:        from,
		To:          to,
		Asset:       "erc20:" + contract,
		Atomic:      amount,
		Scale:       0,
		Kind:        EvidenceObservation,
		SourceID:    source.ID,
	}, nil
}

func validateEVMSource(source Source) error {
	if source.ID == "" || source.Name == "" || source.URL == "" || source.Retrieved == "" || !sha256Pattern.MatchString(source.SHA256) {
		return errors.New("EVM log has incomplete source provenance")
	}
	u, err := url.ParseRequestURI(source.URL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return errors.New("EVM log has an invalid source URL")
	}
	if _, err := time.Parse(time.RFC3339, source.Retrieved); err != nil {
		return errors.New("EVM log has an invalid source retrieval timestamp")
	}
	return nil
}

func normalizeEVMAddress(value string) (string, error) {
	if len(value) != 42 || !strings.HasPrefix(value, "0x") {
		return "", errors.New("address must contain 20 bytes")
	}
	if _, err := hex.DecodeString(value[2:]); err != nil {
		return "", errors.New("address must contain 20 hexadecimal bytes")
	}
	return strings.ToLower(value), nil
}

func addressFromTopic(topic string) (string, error) {
	if len(topic) != 66 || !strings.HasPrefix(strings.ToLower(topic), "0x") {
		return "", errors.New("indexed address topic must contain 32 bytes")
	}
	bytes, err := hex.DecodeString(topic[2:])
	if err != nil {
		return "", errors.New("indexed address topic must be hexadecimal")
	}
	for _, b := range bytes[:12] {
		if b != 0 {
			return "", errors.New("indexed address topic has nonzero padding")
		}
	}
	return "0x" + hex.EncodeToString(bytes[12:]), nil
}

func parseEVMQuantity(value string) (string, error) {
	if len(value) != 66 || !strings.HasPrefix(strings.ToLower(value), "0x") {
		return "", errors.New("quantity must contain 32 bytes")
	}
	bytes, err := hex.DecodeString(value[2:])
	if err != nil {
		return "", errors.New("quantity must be hexadecimal")
	}
	return new(big.Int).SetBytes(bytes).String(), nil
}

func parseEVMIndex(value string) (uint64, error) {
	if !strings.HasPrefix(strings.ToLower(value), "0x") || len(value) < 3 {
		return 0, errors.New("index must be hexadecimal")
	}
	digits := value[2:]
	if len(digits)%2 != 0 {
		digits = "0" + digits
	}
	decoded, err := hex.DecodeString(digits)
	if err != nil || len(decoded) > 8 {
		return 0, errors.New("index is not a valid uint64 hexadecimal value")
	}
	return new(big.Int).SetBytes(decoded).Uint64(), nil
}
