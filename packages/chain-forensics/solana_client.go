package chainforensics

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	DefaultSolanaRPC        = "https://api.mainnet-beta.solana.com"
	DefaultSolanaHistoryRPC = "https://api.mainnet.solana.com"
	maxSolanaResponseBytes  = 1 << 20
	SPLTokenProgramID       = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	SPLToken2022ProgramID   = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
)

const (
	pumpProgramID                       = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"
	pumpCreateV2Discriminator           = "d6904cec5f8b31b4"
	pumpCreateV2DecoderVersion          = "pump-public-docs/idl/pump.json@639b87fb45c3adeffa6f7c4a1bf0cb85e392bf22#create_v2"
	pumpBuyExactQuoteInV2Discriminator  = "c2ab1c46684d5b2f"
	pumpBuyExactQuoteInV2DecoderVersion = "pump-public-docs/idl/pump.json@639b87fb45c3adeffa6f7c4a1bf0cb85e392bf22#buy_exact_quote_in_v2"
	pumpBuyExactQuoteInV2ExpectedLength = 25
)

var ErrSolanaProviderRateLimited = errors.New("Solana RPC provider rate limited the request")

type SolanaProviderError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e *SolanaProviderError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("Solana RPC returned HTTP %d; retry after %s", e.StatusCode, e.RetryAfter)
	}
	return fmt.Sprintf("Solana RPC returned HTTP %d", e.StatusCode)
}

func (e *SolanaProviderError) Is(target error) bool {
	return target == ErrSolanaProviderRateLimited && e.StatusCode == http.StatusTooManyRequests
}

type SolanaSignature struct {
	Signature string          `json:"signature"`
	Slot      uint64          `json:"slot"`
	BlockTime *int64          `json:"blockTime"`
	Err       json.RawMessage `json:"err"`
}

type CapturedSolanaTransaction struct {
	Transaction                SolanaTransaction
	Source                     Source
	Occurrences                []Occurrence
	UnresolvedTransfers        []UnresolvedSolanaTransfer
	PumpInstructions           []PumpInstruction
	UnresolvedPumpInstructions []UnresolvedPumpInstruction
	JupiterOrderEngineFills    []JupiterOrderEngineFill
	Failed                     bool
}

// UnresolvedSolanaTransfer retains an instruction that appears to be a token
// or native transfer but lacks enough public fields to identify its asset or
// amount. It prevents a valid finalized transaction from being discarded or
// silently treating an unknown mint as a known flow.
type UnresolvedSolanaTransfer struct {
	ID          string   `json:"id"`
	Transaction string   `json:"transaction_id"`
	Slot        uint64   `json:"slot"`
	EventIndex  uint32   `json:"event_index"`
	ProgramID   string   `json:"program_id"`
	Accounts    []string `json:"accounts,omitempty"`
	Data        string   `json:"data,omitempty"`
	Reason      string   `json:"reason"`
	SourceID    string   `json:"source_id"`
}

// JupiterOrderEngineFill records the public arguments of one Jupiter RFQ
// Order Engine fill. Taker/maker are instruction roles, not ownership claims.
// Bytes after the pinned IDL arguments are preserved as opaque trailing data.
type JupiterOrderEngineFill struct {
	ID                 string `json:"id"`
	Transaction        string `json:"transaction_id"`
	Slot               uint64 `json:"slot"`
	Index              uint32 `json:"index"`
	ProgramID          string `json:"program_id"`
	SourceID           string `json:"source_id"`
	DecoderVersion     string `json:"decoder_version"`
	Taker              string `json:"taker_role_account"`
	Maker              string `json:"maker_role_account"`
	InputMint          string `json:"input_mint"`
	InputTokenProgram  string `json:"input_token_program"`
	OutputMint         string `json:"output_mint"`
	OutputTokenProgram string `json:"output_token_program"`
	InputAmount        string `json:"input_amount_atomic"`
	OutputAmount       string `json:"output_amount_atomic"`
	ExpireAt           int64  `json:"expire_at_unix"`
	TrailingDataHex    string `json:"trailing_data_hex,omitempty"`
}

// PumpInstruction is a source-grounded decode of an exact Pump IDL instruction
// discriminator. It describes instruction arguments, not the deployed program's
// behavior or the identity/control of any account.
type PumpInstruction struct {
	ID              string `json:"id"`
	Transaction     string `json:"transaction"`
	Slot            uint64 `json:"slot"`
	Index           uint32 `json:"index"`
	ProgramID       string `json:"program_id"`
	SourceID        string `json:"source_id"`
	Kind            string `json:"kind"`
	DecoderVersion  string `json:"decoder_version"`
	Mint            string `json:"mint"`
	User            string `json:"user"`
	Creator         string `json:"creator"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	MetadataURI     string `json:"metadata_uri"`
	MayhemMode      bool   `json:"mayhem_mode"`
	CashbackEnabled bool   `json:"cashback_enabled"`
	CreatorFeeBPS   uint64 `json:"creator_fee_bps"`
	HolderReward    bool   `json:"holder_reward"`
}

type UnresolvedPumpInstruction struct {
	ID                 string `json:"id"`
	Transaction        string `json:"transaction"`
	Slot               uint64 `json:"slot"`
	Index              uint32 `json:"index"`
	ProgramID          string `json:"program_id"`
	SourceID           string `json:"source_id"`
	Reason             string `json:"reason"`
	Discriminator      string `json:"discriminator,omitempty"`
	CandidateKind      string `json:"candidate_kind,omitempty"`
	DecoderVersion     string `json:"decoder_version,omitempty"`
	ExpectedDataLength uint32 `json:"expected_data_length,omitempty"`
	ObservedDataLength uint32 `json:"observed_data_length,omitempty"`
}

type CapturedSolanaPage struct {
	Address      string
	Before       string
	NextBefore   string
	Signatures   []SolanaSignature
	Transactions []CapturedSolanaTransaction
	Source       Source
}

type CapturedSolanaTokenAccountSnapshot struct {
	Snapshot SolanaTokenAccountSnapshot
	Source   Source
}

var (
	sharedSolanaPageCache        = NewWeightedCaptureCache(16, defaultCaptureCacheMaxBytes, cloneSolanaPage, solanaPageCacheWeight)
	sharedSolanaTransactionCache = NewWeightedCaptureCache(128, defaultCaptureCacheMaxBytes, cloneCapturedSolanaTx, solanaTransactionCacheWeight)
	sharedSolanaTokenCache       = NewWeightedCaptureCache(32, defaultCaptureCacheMaxBytes, cloneSolanaTokenSnapshot, solanaTokenSnapshotCacheWeight)
)

func cloneSolanaTokenSnapshot(snapshot CapturedSolanaTokenAccountSnapshot) CapturedSolanaTokenAccountSnapshot {
	clone := snapshot
	clone.Snapshot.Accounts = append([]SolanaTokenAccount(nil), snapshot.Snapshot.Accounts...)
	clone.Source.ResultBody = append([]byte(nil), snapshot.Source.ResultBody...)
	clone.Source.RawResponseBody = append([]byte(nil), snapshot.Source.RawResponseBody...)
	return clone
}

func solanaTokenSnapshotCacheWeight(snapshot CapturedSolanaTokenAccountSnapshot) int64 {
	return int64(len(snapshot.Source.ResultBody)+len(snapshot.Source.RawResponseBody)) + int64(len(snapshot.Snapshot.Accounts))*256
}

func knownSolanaTokenProgram(programID string) bool {
	return programID == SPLTokenProgramID || programID == SPLToken2022ProgramID
}

func solanaPageCacheWeight(page CapturedSolanaPage) int64 {
	weight := int64(len(page.Source.ResultBody)+len(page.Source.RawResponseBody)) + int64(len(page.Signatures))*256
	for _, tx := range page.Transactions {
		weight += solanaTransactionCacheWeight(tx)
	}
	return weight
}

func solanaTransactionCacheWeight(captured CapturedSolanaTransaction) int64 {
	weight := int64(len(captured.Source.ResultBody)+len(captured.Source.RawResponseBody)) + int64(len(captured.Occurrences))*512 + int64(len(captured.UnresolvedTransfers))*384
	weight += int64(len(captured.PumpInstructions)+len(captured.UnresolvedPumpInstructions)+len(captured.JupiterOrderEngineFills)) * 512
	weight += int64(len(captured.Transaction.Transaction.Message.AccountKeys)) * 64
	weight += int64(len(captured.Transaction.Transaction.Message.Instructions)+len(captured.Transaction.Meta.PreTokenBalances)+len(captured.Transaction.Meta.PostTokenBalances)) * 256
	return weight
}

type SolanaTransaction struct {
	Slot        uint64 `json:"slot"`
	BlockTime   *int64 `json:"blockTime"`
	Transaction struct {
		Signatures []string `json:"signatures"`
		Message    struct {
			AccountKeys  []json.RawMessage   `json:"accountKeys"`
			Instructions []SolanaInstruction `json:"instructions"`
		} `json:"message"`
	} `json:"transaction"`
	Meta struct {
		Err               json.RawMessage           `json:"err"`
		Fee               uint64                    `json:"fee"`
		PreBalances       []uint64                  `json:"preBalances"`
		PostBalances      []uint64                  `json:"postBalances"`
		PreTokenBalances  []SolanaTokenBalance      `json:"preTokenBalances"`
		PostTokenBalances []SolanaTokenBalance      `json:"postTokenBalances"`
		InnerInstructions []SolanaInnerInstructions `json:"innerInstructions"`
		LoadedAddresses   struct {
			Writable []string `json:"writable"`
			Readonly []string `json:"readonly"`
		} `json:"loadedAddresses"`
	} `json:"meta"`
}

type SolanaInstruction struct {
	ProgramID      string          `json:"programId"`
	Program        string          `json:"program,omitempty"`
	Accounts       []string        `json:"accounts"`
	AccountIndexes []uint32        `json:"-"`
	ProgramIDIndex *uint32         `json:"programIdIndex,omitempty"`
	Data           string          `json:"data"`
	Parsed         json.RawMessage `json:"parsed,omitempty"`
}

// UnmarshalJSON accepts both the account-index representation returned by
// Solana's raw `json` encoding and the address/parsed representation returned
// by `jsonParsed`. Keeping the indexes lets normalization resolve instructions
// against the exact message key table without discarding the wire evidence.
func (ix *SolanaInstruction) UnmarshalJSON(data []byte) error {
	var raw struct {
		ProgramID      string          `json:"programId"`
		Program        string          `json:"program,omitempty"`
		Accounts       json.RawMessage `json:"accounts"`
		ProgramIDIndex *uint32         `json:"programIdIndex,omitempty"`
		Data           string          `json:"data"`
		Parsed         json.RawMessage `json:"parsed,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	ix.ProgramID, ix.Program, ix.ProgramIDIndex, ix.Data, ix.Parsed = raw.ProgramID, raw.Program, raw.ProgramIDIndex, raw.Data, raw.Parsed
	if len(raw.Accounts) == 0 || string(raw.Accounts) == "null" {
		return nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw.Accounts, &values); err != nil {
		return err
	}
	for _, value := range values {
		var address string
		if err := json.Unmarshal(value, &address); err == nil {
			ix.Accounts = append(ix.Accounts, address)
			continue
		}
		var index uint32
		if err := json.Unmarshal(value, &index); err != nil {
			return fmt.Errorf("instruction account must be an address or index: %w", err)
		}
		ix.AccountIndexes = append(ix.AccountIndexes, index)
	}
	return nil
}

type SolanaInnerInstructions struct {
	Index        uint32              `json:"index"`
	Instructions []SolanaInstruction `json:"instructions"`
}

type parsedSolanaInstruction struct {
	Type string `json:"type"`
	Info struct {
		Source      string          `json:"source"`
		Destination string          `json:"destination"`
		Mint        string          `json:"mint"`
		Amount      json.RawMessage `json:"amount"`
		Lamports    json.RawMessage `json:"lamports"`
		TokenAmount struct {
			Amount   json.RawMessage `json:"amount"`
			Decimals uint32          `json:"decimals"`
		} `json:"tokenAmount"`
	} `json:"info"`
}

type SolanaTokenBalance struct {
	AccountIndex uint32 `json:"accountIndex"`
	Mint         string `json:"mint"`
	Owner        string `json:"owner"`
	ProgramID    string `json:"programId"`
	UIAmount     struct {
		Amount   string `json:"amount"`
		Decimals uint32 `json:"decimals"`
	} `json:"uiTokenAmount"`
}

type SolanaRPCClient struct {
	endpoint string
	http     *http.Client
	now      func() time.Time
	interval time.Duration
	pages    *CaptureCache[CapturedSolanaPage]
	txs      *CaptureCache[CapturedSolanaTransaction]
	request  atomic.Uint64
}

func NewSolanaRPCClient(endpoint string, client *http.Client, now func() time.Time) (*SolanaRPCClient, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHost(u.Hostname()))) {
		return nil, errors.New("Solana RPC endpoint must be HTTPS (HTTP only on loopback), without credentials, query, or fragment")
	}
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("Solana RPC redirects are disabled") }}
	}
	if now == nil {
		now = time.Now
	}
	return &SolanaRPCClient{endpoint: strings.TrimRight(endpoint, "/"), http: client, now: now, interval: 250 * time.Millisecond, pages: sharedSolanaPageCache, txs: sharedSolanaTransactionCache}, nil
}

func (c *SolanaRPCClient) FetchAddressPage(ctx context.Context, address, before string) (CapturedSolanaPage, error) {
	if !validSolanaPubkey(address) {
		return CapturedSolanaPage{}, errors.New("Solana address must be a base58-encoded 32-byte public key")
	}
	if before != "" && !validSolanaSignature(before) {
		return CapturedSolanaPage{}, errors.New("Solana before cursor must be a base58-encoded 64-byte signature")
	}
	key := c.endpoint + "|page|" + address + ":" + before
	if page, ok := c.pages.Get(key, c.now()); ok {
		return page, nil
	}
	params := []any{address, map[string]any{"commitment": "finalized", "limit": 25}}
	if before != "" {
		params[1].(map[string]any)["before"] = before
	}
	body, responseBody, err := c.callCaptured(ctx, "getSignaturesForAddress", params)
	if err != nil {
		return CapturedSolanaPage{}, err
	}
	var signatures []SolanaSignature
	if err := json.Unmarshal(body, &signatures); err != nil {
		return CapturedSolanaPage{}, fmt.Errorf("decode Solana signature page: %w", err)
	}
	if len(signatures) > 25 {
		return CapturedSolanaPage{}, errors.New("Solana RPC returned more than the requested 25 signatures")
	}
	page := CapturedSolanaPage{Address: address, Before: before, Signatures: signatures, Transactions: make([]CapturedSolanaTransaction, 0, len(signatures))}
	seen := make(map[string]struct{}, len(signatures))
	pageSource := sourceForBody("solana-address-page-"+address+"-"+before, "Solana finalized address signature page", c.endpoint, "getSignaturesForAddress(finalized,limit=25)", c.now().UTC(), body, "One bounded finalized signature page; exact result and HTTP response bytes are retained.", responseBody)
	pageSource.ID += "-" + pageSource.SHA256
	page.Source = pageSource
	for _, sig := range signatures {
		if !validSolanaSignature(sig.Signature) || sig.Slot == 0 {
			return CapturedSolanaPage{}, fmt.Errorf("Solana signature page contains invalid or failed entry %q", sig.Signature)
		}
		if _, exists := seen[sig.Signature]; exists {
			return CapturedSolanaPage{}, fmt.Errorf("Solana signature page repeats %q", sig.Signature)
		}
		seen[sig.Signature] = struct{}{}
		captured, err := c.fetchTransaction(ctx, sig.Signature, pageSource.ID, sig.BlockTime)
		if err != nil {
			return CapturedSolanaPage{}, err
		}
		if captured.Transaction.Slot != sig.Slot {
			return CapturedSolanaPage{}, fmt.Errorf("Solana signature slot %d disagrees with transaction slot %d", sig.Slot, captured.Transaction.Slot)
		}
		page.Transactions = append(page.Transactions, captured)
	}
	if len(signatures) > 0 {
		page.NextBefore = signatures[len(signatures)-1].Signature
	}
	now := c.now()
	c.pages.Put(key, page, now.Add(10*time.Minute), now)
	return page, nil
}

// FetchAddressSignatures captures one finalized signature page without fetching
// every referenced transaction. This is for efficient, bounded discovery of
// older address activity; callers must fetch selected signatures separately
// before making transaction-level flow claims.
func (c *SolanaRPCClient) FetchAddressSignatures(ctx context.Context, address, before string, limit uint32) (CapturedSolanaPage, error) {
	if !validSolanaPubkey(address) {
		return CapturedSolanaPage{}, errors.New("Solana address must be a base58-encoded 32-byte public key")
	}
	if before != "" && !validSolanaSignature(before) {
		return CapturedSolanaPage{}, errors.New("Solana before cursor must be a base58-encoded 64-byte signature")
	}
	if limit == 0 || limit > 1000 {
		return CapturedSolanaPage{}, errors.New("Solana signature-only page limit must be from 1 through 1000")
	}
	key := fmt.Sprintf("%s|signatures|%s:%s:%d", c.endpoint, address, before, limit)
	if page, ok := c.pages.Get(key, c.now()); ok {
		return page, nil
	}
	options := map[string]any{"commitment": "finalized", "limit": limit}
	if before != "" {
		options["before"] = before
	}
	body, responseBody, err := c.callCaptured(ctx, "getSignaturesForAddress", []any{address, options})
	if err != nil {
		return CapturedSolanaPage{}, err
	}
	var signatures []SolanaSignature
	if err := json.Unmarshal(body, &signatures); err != nil {
		return CapturedSolanaPage{}, fmt.Errorf("decode Solana signature page: %w", err)
	}
	if uint32(len(signatures)) > limit {
		return CapturedSolanaPage{}, errors.New("Solana RPC returned more than the requested signature count")
	}
	seen := make(map[string]struct{}, len(signatures))
	for _, signature := range signatures {
		if !validSolanaSignature(signature.Signature) || signature.Slot == 0 {
			return CapturedSolanaPage{}, fmt.Errorf("Solana signature page contains invalid entry %q", signature.Signature)
		}
		if _, exists := seen[signature.Signature]; exists {
			return CapturedSolanaPage{}, fmt.Errorf("Solana signature page repeats %q", signature.Signature)
		}
		seen[signature.Signature] = struct{}{}
	}
	citation := fmt.Sprintf("getSignaturesForAddress(commitment=finalized,limit=%d", limit)
	if before != "" {
		citation += ",before=" + before
	}
	citation += ")"
	source := sourceForBody("solana-address-signatures-"+address+"-"+before, "Solana finalized address signature page", c.endpoint, citation, c.now().UTC(), body, "One bounded finalized signature-only page; exact result and HTTP response bytes are retained. This page locates transaction signatures but contains no transaction-level transfer evidence; fetch selected signatures separately.", responseBody)
	source.ID += "-" + source.SHA256
	page := CapturedSolanaPage{Address: address, Before: before, Signatures: signatures, Source: source}
	if len(signatures) > 0 {
		page.NextBefore = signatures[len(signatures)-1].Signature
	}
	now := c.now()
	c.pages.Put(key, page, now.Add(2*time.Minute), now)
	return page, nil
}

// FetchTokenAccountsByOwner retrieves one finalized SPL token-program account
// inventory for an owner. The exact jsonParsed result and HTTP envelope are
// retained; the token-account address is never conflated with its owner.
func (c *SolanaRPCClient) FetchTokenAccountsByOwner(ctx context.Context, owner, programID string) (CapturedSolanaTokenAccountSnapshot, error) {
	if !validSolanaPubkey(owner) {
		return CapturedSolanaTokenAccountSnapshot{}, errors.New("Solana token-account owner must be a base58-encoded 32-byte public key")
	}
	if !knownSolanaTokenProgram(programID) {
		return CapturedSolanaTokenAccountSnapshot{}, fmt.Errorf("unsupported SPL token program %q", programID)
	}
	key := c.endpoint + "|token-accounts|" + owner + ":" + programID
	if snapshot, ok := sharedSolanaTokenCache.Get(key, c.now()); ok {
		return snapshot, nil
	}
	params := []any{owner, map[string]any{"programId": programID}, map[string]any{"commitment": "finalized", "encoding": "jsonParsed"}}
	body, responseBody, err := c.callCaptured(ctx, "getTokenAccountsByOwner", params)
	if err != nil {
		return CapturedSolanaTokenAccountSnapshot{}, err
	}
	var result struct {
		Context struct {
			Slot uint64 `json:"slot"`
		} `json:"context"`
		Value []struct {
			Pubkey  string `json:"pubkey"`
			Account struct {
				Owner string `json:"owner"`
				Data  struct {
					Program string `json:"program"`
					Parsed  struct {
						Info struct {
							Mint        string `json:"mint"`
							Owner       string `json:"owner"`
							TokenAmount struct {
								Amount   string `json:"amount"`
								Decimals uint32 `json:"decimals"`
							} `json:"tokenAmount"`
						} `json:"info"`
					} `json:"parsed"`
				} `json:"data"`
			} `json:"account"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return CapturedSolanaTokenAccountSnapshot{}, fmt.Errorf("decode finalized SPL owner inventory: %w", err)
	}
	if result.Context.Slot == 0 {
		return CapturedSolanaTokenAccountSnapshot{}, errors.New("Solana RPC token-account inventory is missing its finalized context slot")
	}
	if len(result.Value) > 10000 {
		return CapturedSolanaTokenAccountSnapshot{}, errors.New("Solana RPC token-account inventory exceeds 10000 accounts")
	}
	accounts := make([]SolanaTokenAccount, 0, len(result.Value))
	seen := make(map[string]struct{}, len(result.Value))
	for _, item := range result.Value {
		account := SolanaTokenAccount{Address: item.Pubkey, Mint: item.Account.Data.Parsed.Info.Mint, Owner: item.Account.Data.Parsed.Info.Owner, ProgramID: item.Account.Owner, AtomicQuantity: item.Account.Data.Parsed.Info.TokenAmount.Amount, Scale: item.Account.Data.Parsed.Info.TokenAmount.Decimals, ScaleKnown: true}
		if !validSolanaPubkey(account.Address) || !validSolanaPubkey(account.Mint) || account.Owner != owner || account.ProgramID != programID || !decimalPattern.MatchString(account.AtomicQuantity) {
			return CapturedSolanaTokenAccountSnapshot{}, fmt.Errorf("Solana RPC returned incomplete or mismatched token account %q", account.Address)
		}
		if _, exists := seen[account.Address]; exists {
			return CapturedSolanaTokenAccountSnapshot{}, fmt.Errorf("Solana RPC repeated token account %q", account.Address)
		}
		seen[account.Address] = struct{}{}
		accounts = append(accounts, account)
	}
	limitations := "One finalized, RPC-parsed token-account inventory filtered by the SPL program ID at its returned context slot. Token-account addresses differ from owner addresses; the result does not identify a natural person, key controller, or transaction history. The RPC's parsed account fields are retained verbatim as the source result."
	source := sourceForBody("solana-token-accounts-"+owner+"-"+programID, "Solana finalized SPL token accounts by owner", c.endpoint, "getTokenAccountsByOwner(finalized,jsonParsed,programId)", c.now().UTC(), body, limitations, responseBody)
	source.ID += "-" + source.SHA256
	snapshot := CapturedSolanaTokenAccountSnapshot{Snapshot: SolanaTokenAccountSnapshot{ID: "solana-token-snapshot-" + owner + "-" + programID + "-" + strconv.FormatUint(result.Context.Slot, 10) + "-" + source.SHA256[:12], Owner: owner, ProgramID: programID, ContextSlot: result.Context.Slot, SourceID: source.ID, Accounts: accounts}, Source: source}
	now := c.now()
	sharedSolanaTokenCache.Put(key, snapshot, now.Add(2*time.Minute), now)
	return snapshot, nil
}

// FetchTransaction resolves one caller-supplied finalized signature without
// first enumerating an address history. The exact signed transaction, result
// object, successful HTTP envelope, SPL balance deltas, and Pump instruction
// decodes are retained by the returned capture. A signature is only a lookup
// key; it does not supply address ownership or attribution evidence.
func (c *SolanaRPCClient) FetchTransaction(ctx context.Context, signature string) (CapturedSolanaTransaction, error) {
	if !validSolanaSignature(signature) {
		return CapturedSolanaTransaction{}, errors.New("Solana transaction signature must be a base58-encoded 64-byte signature")
	}
	return c.fetchTransaction(ctx, signature, "caller-supplied-signature", nil)
}

func (c *SolanaRPCClient) fetchTransaction(ctx context.Context, signature, pageSourceID string, pageBlockTime *int64) (CapturedSolanaTransaction, error) {
	cacheKey := c.endpoint + "|tx|" + pageSourceID + "|" + signature
	if tx, ok := c.txs.Get(cacheKey, c.now()); ok {
		return tx, nil
	}
	body, responseBody, err := c.callCaptured(ctx, "getTransaction", []any{signature, map[string]any{"commitment": "finalized", "encoding": "json", "maxSupportedTransactionVersion": 1}})
	if err != nil {
		return CapturedSolanaTransaction{}, err
	}
	var tx SolanaTransaction
	if err := json.Unmarshal(body, &tx); err != nil {
		return CapturedSolanaTransaction{}, fmt.Errorf("decode Solana transaction: %w", err)
	}
	if tx.Slot == 0 || len(tx.Transaction.Signatures) == 0 || tx.Transaction.Signatures[0] != signature {
		return CapturedSolanaTransaction{}, errors.New("Solana transaction is missing finalized identity")
	}
	blockTimestamp := tx.BlockTime
	if blockTimestamp == nil {
		blockTimestamp = pageBlockTime
	}
	keyByIndex := make(map[uint32]string, len(tx.Transaction.Message.AccountKeys))
	for i, raw := range tx.Transaction.Message.AccountKeys {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			var key struct {
				Pubkey string `json:"pubkey"`
			}
			if err := json.Unmarshal(raw, &key); err != nil {
				return CapturedSolanaTransaction{}, fmt.Errorf("decode Solana account key %d: %w", i, err)
			}
			text = key.Pubkey
		}
		keyByIndex[uint32(i)] = text
	}
	nextKeyIndex := uint32(len(tx.Transaction.Message.AccountKeys))
	for _, address := range tx.Meta.LoadedAddresses.Writable {
		keyByIndex[nextKeyIndex] = address
		nextKeyIndex++
	}
	for _, address := range tx.Meta.LoadedAddresses.Readonly {
		keyByIndex[nextKeyIndex] = address
		nextKeyIndex++
	}
	if err := normalizeSolanaInstructions(&tx, keyByIndex); err != nil {
		return CapturedSolanaTransaction{}, err
	}
	pre := make(map[uint32]SolanaTokenBalance, len(tx.Meta.PreTokenBalances))
	post := make(map[uint32]SolanaTokenBalance, len(tx.Meta.PostTokenBalances))
	for _, balance := range tx.Meta.PreTokenBalances {
		pre[balance.AccountIndex] = balance
	}
	for _, balance := range tx.Meta.PostTokenBalances {
		post[balance.AccountIndex] = balance
	}
	indexes := make(map[uint32]struct{}, len(pre)+len(post))
	for i := range pre {
		indexes[i] = struct{}{}
	}
	for i := range post {
		indexes[i] = struct{}{}
	}
	ordered := make([]int, 0, len(indexes))
	for i := range indexes {
		ordered = append(ordered, int(i))
	}
	sort.Ints(ordered)
	blockTime := ""
	if blockTimestamp != nil {
		blockTime = time.Unix(*blockTimestamp, 0).UTC().Format(time.RFC3339)
	}
	source := sourceForBody("solana-transaction-"+signature, "Solana finalized transaction", c.endpoint, "getTransaction(json,finalized)", c.now().UTC(), body, "Exact finalized result and HTTP response bytes are retained. Raw account indexes are resolved against static and loaded message keys; transfer instructions are decoded separately from pre/post native lamport and SPL account-level balance observations. "+solanaTransactionLineage(pageSourceID), responseBody)
	source.ID += "-" + source.SHA256
	if blockTimestamp == nil {
		source.Limitations += " Provider omitted block time; slot is retained but occurrence timestamp is unavailable."
	}
	captured := CapturedSolanaTransaction{Transaction: tx, Source: source, Failed: tx.Meta.Err != nil && string(tx.Meta.Err) != "null"}
	if !captured.Failed {
		captured.PumpInstructions, captured.UnresolvedPumpInstructions = decodePumpInstructions(tx.Transaction.Message.Instructions)
		for i := range captured.PumpInstructions {
			captured.PumpInstructions[i].ID = fmt.Sprintf("solana:%s:pump:%d", signature, captured.PumpInstructions[i].Index)
			captured.PumpInstructions[i].Transaction = signature
			captured.PumpInstructions[i].Slot = tx.Slot
			captured.PumpInstructions[i].ProgramID = pumpProgramID
			captured.PumpInstructions[i].SourceID = source.ID
		}
		for i := range captured.UnresolvedPumpInstructions {
			id := fmt.Sprintf("solana:%s:pump:%d", signature, captured.UnresolvedPumpInstructions[i].Index)
			if captured.UnresolvedPumpInstructions[i].CandidateKind != "" {
				id += ":candidate:" + captured.UnresolvedPumpInstructions[i].CandidateKind
			}
			captured.UnresolvedPumpInstructions[i].ID = id
			captured.UnresolvedPumpInstructions[i].Transaction = signature
			captured.UnresolvedPumpInstructions[i].Slot = tx.Slot
			captured.UnresolvedPumpInstructions[i].ProgramID = pumpProgramID
			captured.UnresolvedPumpInstructions[i].SourceID = source.ID
		}
		captured.JupiterOrderEngineFills = decodeJupiterOrderEngineFills(tx.Transaction.Message.Instructions)
		for i := range captured.JupiterOrderEngineFills {
			fill := &captured.JupiterOrderEngineFills[i]
			fill.ID = fmt.Sprintf("solana:%s:jupiter-rfq-fill:%d", signature, fill.Index)
			fill.Transaction = signature
			fill.Slot = tx.Slot
			fill.SourceID = source.ID
		}
	}
	if !captured.Failed {
		if len(tx.Meta.PreBalances) != len(keyByIndex) || len(tx.Meta.PostBalances) != len(keyByIndex) {
			return CapturedSolanaTransaction{}, errors.New("Solana native balance vectors do not match transaction account keys")
		}
		for i, before := range tx.Meta.PreBalances {
			after := tx.Meta.PostBalances[i]
			if before == after {
				continue
			}
			key := keyByIndex[uint32(i)]
			if !validSolanaPubkey(key) {
				return CapturedSolanaTransaction{}, fmt.Errorf("Solana native balance index %d has invalid account key", i)
			}
			direction, value := "balance_debit", new(big.Int).SetUint64(before)
			if after > before {
				direction, value = "balance_credit", new(big.Int).SetUint64(after-before)
			} else {
				value.Sub(value, new(big.Int).SetUint64(after))
			}
			captured.Occurrences = append(captured.Occurrences, Occurrence{ID: fmt.Sprintf("solana:%s:lamports:%d", signature, i), Chain: ChainSolana, Transaction: signature, Block: tx.Slot, Timestamp: blockTime, EventIndex: uint32(i), Direction: direction, Account: key, Asset: "SOL", Atomic: value.String(), Scale: 9, Measurement: MeasurementNetBalanceDelta, Kind: EvidenceObservation, SourceID: source.ID})
		}
		captured.Source = source
	}
	if !captured.Failed {
		mintByAccount := make(map[string]SolanaTokenBalance, len(pre)+len(post))
		for index, balance := range pre {
			mintByAccount[keyByIndex[index]] = balance
		}
		for index, balance := range post {
			mintByAccount[keyByIndex[index]] = balance
		}
		appendTransfer := func(ix SolanaInstruction, eventIndex uint32) error {
			transfer, ok, err := decodeSolanaTransfer(ix, mintByAccount)
			if err != nil {
				captured.UnresolvedTransfers = append(captured.UnresolvedTransfers, UnresolvedSolanaTransfer{
					ID:          fmt.Sprintf("solana:%s:unresolved-transfer:%d", signature, eventIndex),
					Transaction: signature, Slot: tx.Slot, EventIndex: eventIndex, ProgramID: ix.ProgramID,
					Accounts: append([]string(nil), ix.Accounts...), Data: ix.Data, Reason: err.Error(), SourceID: source.ID,
				})
				return nil
			}
			if !ok {
				return err
			}
			captured.Occurrences = append(captured.Occurrences, Occurrence{
				ID: fmt.Sprintf("solana:%s:transfer:%d", signature, eventIndex), Chain: ChainSolana,
				Transaction: signature, Block: tx.Slot, Timestamp: blockTime, EventIndex: eventIndex,
				Direction: "transfer", From: transfer.From, To: transfer.To, Account: transfer.From,
				Asset: transfer.Asset, Atomic: transfer.Atomic, Scale: transfer.Scale,
				Measurement: MeasurementInstructionTransfer, Kind: EvidenceObservation, SourceID: source.ID,
			})
			return nil
		}
		for i, ix := range tx.Transaction.Message.Instructions {
			if err := appendTransfer(ix, uint32(i)); err != nil {
				return CapturedSolanaTransaction{}, fmt.Errorf("decode Solana instruction %d transfer: %w", i, err)
			}
		}
		eventIndex := uint32(len(tx.Transaction.Message.Instructions))
		for _, group := range tx.Meta.InnerInstructions {
			for _, ix := range group.Instructions {
				if err := appendTransfer(ix, eventIndex); err != nil {
					return CapturedSolanaTransaction{}, fmt.Errorf("decode Solana inner instruction %d transfer: %w", eventIndex, err)
				}
				eventIndex++
			}
		}
		if len(captured.UnresolvedTransfers) > 0 {
			source.Limitations += fmt.Sprintf(" %d transfer instructions lacked enough fields to resolve asset or amount; retained in unresolvedTransfers without emitting a typed flow.", len(captured.UnresolvedTransfers))
			captured.Source = source
		}
	}
	if captured.Failed {
		source.Limitations += " Transaction failed on-chain; no native or token balance delta observations are emitted."
		captured.Source = source
	}
	for _, rawIndex := range ordered {
		if captured.Failed {
			break
		}
		index := uint32(rawIndex)
		a, aok := pre[index]
		b, bok := post[index]
		mint, decimals, owner := "", uint32(0), ""
		if aok {
			mint, decimals, owner = a.Mint, a.UIAmount.Decimals, a.Owner
		}
		if bok {
			if mint != "" && mint != b.Mint {
				return CapturedSolanaTransaction{}, errors.New("Solana token mint changed for one account index")
			}
			if mint == "" {
				mint = b.Mint
			}
			if aok && decimals != b.UIAmount.Decimals {
				return CapturedSolanaTransaction{}, errors.New("Solana token decimals changed across one transaction")
			}
			decimals = b.UIAmount.Decimals
			if owner == "" {
				owner = b.Owner
			}
		}
		if mint == "" {
			return CapturedSolanaTransaction{}, errors.New("Solana token balance is missing mint")
		}
		before, after := new(big.Int), new(big.Int)
		if aok {
			if _, ok := before.SetString(a.UIAmount.Amount, 10); !ok || before.Sign() < 0 {
				return CapturedSolanaTransaction{}, errors.New("invalid pre-transaction SPL atomic amount")
			}
		}
		if bok {
			if _, ok := after.SetString(b.UIAmount.Amount, 10); !ok || after.Sign() < 0 {
				return CapturedSolanaTransaction{}, errors.New("invalid post-transaction SPL atomic amount")
			}
		}
		delta := new(big.Int).Sub(after, before)
		if delta.Sign() == 0 {
			continue
		}
		account := owner
		if account == "" {
			account = keyByIndex[index]
		}
		direction, from, to := "output", "", account
		value := new(big.Int).Set(delta)
		if delta.Sign() < 0 {
			direction, from, to = "input", account, ""
			value.Neg(value)
		}
		occurrence := Occurrence{ID: fmt.Sprintf("solana:%s:spl:%d", signature, index), Chain: ChainSolana, Transaction: signature, Block: tx.Slot, Timestamp: blockTime, EventIndex: index, Direction: direction, From: from, To: to, Account: keyByIndex[index], Asset: "SPL:" + mint, Atomic: value.String(), Scale: decimals, Measurement: MeasurementNetBalanceDelta, Kind: EvidenceObservation, SourceID: source.ID}
		captured.Occurrences = append(captured.Occurrences, occurrence)
	}
	now := c.now()
	c.txs.Put(cacheKey, captured, now.Add(10*time.Minute), now)
	return captured, nil
}

type SolanaTransfer struct {
	From   string
	To     string
	Asset  string
	Atomic string
	Scale  uint32
}

const MeasurementInstructionTransfer = "instruction_transfer"

const (
	solanaSystemProgram                  = "11111111111111111111111111111111"
	solanaTokenProgram                   = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	solanaToken2022                      = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
	jupiterOrderEngineProgramID          = "61DFfeTKM7trxYcPQCM78bJ794ddZprZpAwAnLiwTpYH"
	jupiterOrderEngineFillDecoderVersion = "jup-ag/rfq-webhook-toolkit/idls/order_engine.json@3aa4fce515d4b5827ee2d48f08041ea706d911cd#sha256:1e0257f9684ce747f4544c27695abdfc8414a3f5ba710d0943877b884a3ef033"
)

var jupiterOrderEngineFillDiscriminator = []byte{168, 96, 183, 163, 92, 10, 40, 160}

func decodeJupiterOrderEngineFills(instructions []SolanaInstruction) []JupiterOrderEngineFill {
	var fills []JupiterOrderEngineFill
	for i, ix := range instructions {
		if ix.ProgramID != jupiterOrderEngineProgramID {
			continue
		}
		data, ok := decodeBase58(ix.Data)
		if !ok || len(data) < 32 || !bytes.Equal(data[:8], jupiterOrderEngineFillDiscriminator) || len(ix.Accounts) < 10 {
			continue
		}
		fills = append(fills, JupiterOrderEngineFill{
			Index: uint32(i), ProgramID: ix.ProgramID, DecoderVersion: jupiterOrderEngineFillDecoderVersion,
			Taker: ix.Accounts[0], Maker: ix.Accounts[1], InputMint: ix.Accounts[6], InputTokenProgram: ix.Accounts[7],
			OutputMint: ix.Accounts[8], OutputTokenProgram: ix.Accounts[9],
			InputAmount:  new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[8:16])).String(),
			OutputAmount: new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[16:24])).String(),
			ExpireAt:     int64(binary.LittleEndian.Uint64(data[24:32])), TrailingDataHex: fmt.Sprintf("%x", data[32:]),
		})
	}
	return fills
}

func normalizeSolanaInstructions(tx *SolanaTransaction, keys map[uint32]string) error {
	normalize := func(ix *SolanaInstruction) error {
		if ix.ProgramIDIndex != nil {
			program, ok := keys[*ix.ProgramIDIndex]
			if !ok {
				return fmt.Errorf("Solana instruction program index %d is outside the message key table", *ix.ProgramIDIndex)
			}
			ix.ProgramID = program
		}
		if len(ix.AccountIndexes) > 0 {
			for _, index := range ix.AccountIndexes {
				account, ok := keys[index]
				if !ok {
					return fmt.Errorf("Solana instruction account index %d is outside the message key table", index)
				}
				ix.Accounts = append(ix.Accounts, account)
			}
		}
		if len(ix.Parsed) != 0 {
			return nil
		}
		data, ok := decodeBase58(ix.Data)
		if !ok || len(data) == 0 {
			return nil
		}
		var kind string
		info := map[string]any{}
		switch ix.ProgramID {
		case solanaSystemProgram:
			if len(data) != 12 || binary.LittleEndian.Uint32(data[:4]) != 2 || len(ix.Accounts) < 2 {
				return nil
			}
			kind = "transfer"
			info["source"], info["destination"] = ix.Accounts[0], ix.Accounts[1]
			info["lamports"] = new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[4:])).String()
		case solanaTokenProgram, solanaToken2022:
			switch data[0] {
			case 3: // SPL Token Transfer: u8 instruction + u64 amount
				if len(data) != 9 || len(ix.Accounts) < 2 {
					return nil
				}
				kind = "transfer"
				info["source"], info["destination"] = ix.Accounts[0], ix.Accounts[1]
				info["amount"] = new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[1:])).String()
			case 12: // SPL Token TransferChecked: tag + u64 amount + decimals
				if len(data) != 10 || len(ix.Accounts) < 3 {
					return nil
				}
				kind = "transferChecked"
				info["source"], info["mint"], info["destination"] = ix.Accounts[0], ix.Accounts[1], ix.Accounts[2]
				info["tokenAmount"] = map[string]any{"amount": new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[1:9])).String(), "decimals": uint32(data[9])}
			}
		}
		if kind == "" {
			return nil
		}
		parsed, err := json.Marshal(map[string]any{"type": kind, "info": info})
		if err != nil {
			return err
		}
		ix.Parsed = parsed
		return nil
	}
	for i := range tx.Transaction.Message.Instructions {
		if err := normalize(&tx.Transaction.Message.Instructions[i]); err != nil {
			return fmt.Errorf("normalize Solana instruction %d: %w", i, err)
		}
	}
	for groupIndex := range tx.Meta.InnerInstructions {
		for instructionIndex := range tx.Meta.InnerInstructions[groupIndex].Instructions {
			if err := normalize(&tx.Meta.InnerInstructions[groupIndex].Instructions[instructionIndex]); err != nil {
				return fmt.Errorf("normalize Solana inner instruction %d: %w", instructionIndex, err)
			}
		}
	}
	return nil
}

func decodeSolanaTransfer(ix SolanaInstruction, mintByAccount map[string]SolanaTokenBalance) (SolanaTransfer, bool, error) {
	if len(ix.Parsed) == 0 {
		return SolanaTransfer{}, false, nil
	}
	var parsed parsedSolanaInstruction
	if err := json.Unmarshal(ix.Parsed, &parsed); err != nil {
		return SolanaTransfer{}, false, fmt.Errorf("decode parsed instruction: %w", err)
	}
	if parsed.Type != "transfer" && parsed.Type != "transferChecked" {
		return SolanaTransfer{}, false, nil
	}
	from, to := parsed.Info.Source, parsed.Info.Destination
	if !validSolanaPubkey(from) || !validSolanaPubkey(to) {
		return SolanaTransfer{}, false, errors.New("transfer is missing valid source or destination")
	}
	var amountRaw json.RawMessage
	transfer := SolanaTransfer{From: from, To: to}
	switch ix.ProgramID {
	case solanaSystemProgram:
		if parsed.Type != "transfer" {
			return SolanaTransfer{}, false, nil
		}
		amountRaw, transfer.Asset, transfer.Scale = parsed.Info.Lamports, "SOL", 9
	case solanaTokenProgram, solanaToken2022:
		transfer.Asset = parsed.Info.Mint
		amountRaw = parsed.Info.Amount
		if parsed.Type == "transferChecked" {
			if parsed.Info.Mint == "" {
				return SolanaTransfer{}, false, errors.New("checked SPL transfer is missing mint")
			}
			amountRaw, transfer.Scale = parsed.Info.TokenAmount.Amount, parsed.Info.TokenAmount.Decimals
		} else if balance, ok := mintByAccount[from]; ok {
			transfer.Asset, transfer.Scale = balance.Mint, balance.UIAmount.Decimals
		}
		if transfer.Asset == "" {
			return SolanaTransfer{}, false, errors.New("SPL transfer mint is unavailable from instruction and transaction balances")
		}
		transfer.Asset = "SPL:" + transfer.Asset
	default:
		return SolanaTransfer{}, false, nil
	}
	amount, err := parseSolanaAtomicAmount(amountRaw)
	if err != nil || amount.Sign() <= 0 {
		return SolanaTransfer{}, false, errors.New("transfer amount is missing, invalid, or nonpositive")
	}
	transfer.Atomic = amount.String()
	return transfer, true, nil
}

func parseSolanaAtomicAmount(raw json.RawMessage) (*big.Int, error) {
	var text string
	if len(raw) == 0 {
		return nil, errors.New("missing amount")
	}
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
	} else {
		text = string(raw)
	}
	amount, ok := new(big.Int).SetString(text, 10)
	if !ok || amount.Sign() < 0 {
		return nil, errors.New("amount is not a nonnegative integer")
	}
	return amount, nil
}

func solanaTransactionLineage(pageSourceID string) string {
	if pageSourceID == "caller-supplied-signature" {
		return "The transaction was looked up by an externally supplied signature; this capture does not include the address-history page that selected it."
	}
	return "Address-page source: " + pageSourceID
}

func decodePumpInstructions(instructions []SolanaInstruction) ([]PumpInstruction, []UnresolvedPumpInstruction) {
	decoded := make([]PumpInstruction, 0)
	unresolved := make([]UnresolvedPumpInstruction, 0)
	for i, ix := range instructions {
		if ix.ProgramID != pumpProgramID {
			continue
		}
		data, ok := decodeBase58(ix.Data)
		if !ok || len(data) < 8 {
			unresolved = append(unresolved, UnresolvedPumpInstruction{Index: uint32(i), Reason: "instruction data is not valid base58 or is shorter than a discriminator"})
			continue
		}
		disc := fmt.Sprintf("%x", data[:8])
		if disc == pumpBuyExactQuoteInV2Discriminator {
			unresolved = append(unresolved, UnresolvedPumpInstruction{
				Index: uint32(i), Reason: fmt.Sprintf("pinned official Pump IDL identifies buy_exact_quote_in_v2 but expects %d total instruction bytes; observed %d. Historical deployed layout is unverified, so arguments and account roles remain unresolved", pumpBuyExactQuoteInV2ExpectedLength, len(data)),
				Discriminator: disc, CandidateKind: "buy_exact_quote_in_v2", DecoderVersion: pumpBuyExactQuoteInV2DecoderVersion,
				ExpectedDataLength: pumpBuyExactQuoteInV2ExpectedLength, ObservedDataLength: uint32(len(data)),
			})
			continue
		}
		if disc != pumpCreateV2Discriminator {
			unresolved = append(unresolved, UnresolvedPumpInstruction{Index: uint32(i), Reason: "Pump discriminator is not in this decoder version", Discriminator: disc})
			continue
		}
		item, err := decodePumpCreateV2(uint32(i), ix.Accounts, data[8:])
		if err != nil {
			unresolved = append(unresolved, UnresolvedPumpInstruction{Index: uint32(i), Reason: err.Error(), Discriminator: disc})
			continue
		}
		decoded = append(decoded, item)
	}
	return decoded, unresolved
}

func decodePumpCreateV2(index uint32, accounts []string, data []byte) (PumpInstruction, error) {
	// Account order is pinned to the IDL revision in pumpCreateV2DecoderVersion:
	// mint at 0 and user at 5. Fewer accounts remain unresolved.
	if len(accounts) < 6 {
		return PumpInstruction{}, errors.New("create_v2 has fewer than six required accounts")
	}
	reader := pumpBorshReader{data: data}
	name, err := reader.string()
	if err != nil {
		return PumpInstruction{}, fmt.Errorf("decode create_v2 name: %w", err)
	}
	symbol, err := reader.string()
	if err != nil {
		return PumpInstruction{}, fmt.Errorf("decode create_v2 symbol: %w", err)
	}
	uri, err := reader.string()
	if err != nil {
		return PumpInstruction{}, fmt.Errorf("decode create_v2 URI: %w", err)
	}
	creator, err := reader.pubkey()
	if err != nil {
		return PumpInstruction{}, fmt.Errorf("decode create_v2 creator: %w", err)
	}
	mayhem, err := reader.bool()
	if err != nil {
		return PumpInstruction{}, fmt.Errorf("decode create_v2 mayhem mode: %w", err)
	}
	// Pump's OptionBool/OptionU64 names are one-field IDL structs, not
	// Borsh Option<T> tags; decode the exact bool/u64 fields.
	cashback, err := reader.bool()
	if err != nil {
		return PumpInstruction{}, fmt.Errorf("decode create_v2 cashback option: %w", err)
	}
	creatorFeeBPS, err := reader.u64()
	if err != nil {
		return PumpInstruction{}, fmt.Errorf("decode create_v2 creator fee option: %w", err)
	}
	holderReward, err := reader.bool()
	if err != nil {
		return PumpInstruction{}, fmt.Errorf("decode create_v2 holder reward option: %w", err)
	}
	if reader.remaining() != 0 {
		return PumpInstruction{}, errors.New("create_v2 contains trailing bytes outside the documented arguments")
	}
	if !validSolanaPubkey(accounts[0]) || !validSolanaPubkey(accounts[5]) {
		return PumpInstruction{}, errors.New("create_v2 mint or user account is invalid")
	}
	return PumpInstruction{Index: index, Kind: "create_v2", DecoderVersion: pumpCreateV2DecoderVersion, Mint: accounts[0], User: accounts[5], Creator: creator, Name: name, Symbol: symbol, MetadataURI: uri, MayhemMode: mayhem, CashbackEnabled: cashback, CreatorFeeBPS: creatorFeeBPS, HolderReward: holderReward}, nil
}

type pumpBorshReader struct {
	data   []byte
	offset int
}

func (r *pumpBorshReader) remaining() int { return len(r.data) - r.offset }
func (r *pumpBorshReader) take(n int) ([]byte, error) {
	if n < 0 || r.remaining() < n {
		return nil, errors.New("truncated Borsh value")
	}
	b := r.data[r.offset : r.offset+n]
	r.offset += n
	return b, nil
}

func (r *pumpBorshReader) string() (string, error) {
	raw, err := r.take(4)
	if err != nil {
		return "", err
	}
	n := binary.LittleEndian.Uint32(raw)
	if n > 1<<20 {
		return "", errors.New("string exceeds 1 MiB")
	}
	b, err := r.take(int(n))
	return string(b), err
}

func (r *pumpBorshReader) pubkey() (string, error) {
	b, err := r.take(32)
	if err != nil {
		return "", err
	}
	return encodeBase58(b), nil
}

func (r *pumpBorshReader) bool() (bool, error) {
	b, err := r.take(1)
	if err != nil {
		return false, err
	}
	if b[0] > 1 {
		return false, errors.New("non-canonical bool")
	}
	return b[0] == 1, nil
}

func (r *pumpBorshReader) u64() (uint64, error) {
	b, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (c *SolanaRPCClient) call(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	result, _, err := c.callCaptured(ctx, method, params)
	return result, err
}

func (c *SolanaRPCClient) callCaptured(ctx context.Context, method string, params []any) (json.RawMessage, []byte, error) {
	request := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  []any  `json:"params"`
	}{"2.0", c.request.Add(1), method, params}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	release, err := sharedEndpointRateLimiter.Acquire(ctx, c.endpoint, c.interval)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	bounded, err := io.ReadAll(io.LimitReader(resp.Body, maxSolanaResponseBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if len(bounded) > maxSolanaResponseBytes {
		return nil, nil, errors.New("Solana RPC response exceeds 1 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), c.now())
		if resp.StatusCode == http.StatusTooManyRequests && retryAfter > 0 {
			sharedEndpointRateLimiter.DeferUntil(c.endpoint, c.now().Add(retryAfter))
		}
		return nil, nil, &SolanaProviderError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bounded, &envelope); err != nil {
		return nil, nil, err
	}
	if envelope.Error != nil {
		return nil, nil, fmt.Errorf("Solana RPC error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return nil, nil, errors.New("Solana RPC returned no result")
	}
	return envelope.Result, append([]byte(nil), bounded...), nil
}

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func decodeBase58(raw string) ([]byte, bool) {
	if raw == "" {
		return nil, false
	}
	n := new(big.Int)
	base := big.NewInt(58)
	for _, r := range raw {
		i := strings.IndexRune(base58Alphabet, r)
		if i < 0 {
			return nil, false
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(i)))
	}
	decoded := n.Bytes()
	zeros := 0
	for zeros < len(raw) && raw[zeros] == '1' {
		zeros++
	}
	return append(bytes.Repeat([]byte{0}, zeros), decoded...), true
}

func encodeBase58(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	zeroes := 0
	for zeroes < len(raw) && raw[zeroes] == 0 {
		zeroes++
	}
	n := new(big.Int).SetBytes(raw)
	base := big.NewInt(58)
	rem := new(big.Int)
	var out []byte
	for n.Sign() > 0 {
		n.QuoRem(n, base, rem)
		out = append(out, base58Alphabet[rem.Int64()])
	}
	for i := 0; i < zeroes; i++ {
		out = append(out, '1')
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
func validSolanaPubkey(raw string) bool    { b, ok := decodeBase58(raw); return ok && len(b) == 32 }
func validSolanaSignature(raw string) bool { b, ok := decodeBase58(raw); return ok && len(b) == 64 }
func cloneSolanaPage(page CapturedSolanaPage) CapturedSolanaPage {
	clone := page
	clone.Source.ResultBody = append([]byte(nil), page.Source.ResultBody...)
	clone.Source.RawResponseBody = append([]byte(nil), page.Source.RawResponseBody...)
	clone.Signatures = append([]SolanaSignature(nil), page.Signatures...)
	clone.Transactions = make([]CapturedSolanaTransaction, len(page.Transactions))
	for i, tx := range page.Transactions {
		clone.Transactions[i] = cloneCapturedSolanaTx(tx)
	}
	return clone
}

func cloneCapturedSolanaTx(tx CapturedSolanaTransaction) CapturedSolanaTransaction {
	clone := tx
	clone.Source = tx.Source
	clone.Source.ResultBody = append([]byte(nil), tx.Source.ResultBody...)
	clone.Source.RawResponseBody = append([]byte(nil), tx.Source.RawResponseBody...)
	clone.Occurrences = append([]Occurrence(nil), tx.Occurrences...)
	clone.UnresolvedTransfers = append([]UnresolvedSolanaTransfer(nil), tx.UnresolvedTransfers...)
	for i := range clone.UnresolvedTransfers {
		clone.UnresolvedTransfers[i].Accounts = append([]string(nil), tx.UnresolvedTransfers[i].Accounts...)
	}
	clone.PumpInstructions = append([]PumpInstruction(nil), tx.PumpInstructions...)
	clone.UnresolvedPumpInstructions = append([]UnresolvedPumpInstruction(nil), tx.UnresolvedPumpInstructions...)
	clone.JupiterOrderEngineFills = append([]JupiterOrderEngineFill(nil), tx.JupiterOrderEngineFills...)
	clone.Transaction.Transaction.Signatures = append([]string(nil), tx.Transaction.Transaction.Signatures...)
	clone.Transaction.Transaction.Message.AccountKeys = append([]json.RawMessage(nil), tx.Transaction.Transaction.Message.AccountKeys...)
	clone.Transaction.Meta.PreTokenBalances = append([]SolanaTokenBalance(nil), tx.Transaction.Meta.PreTokenBalances...)
	clone.Transaction.Meta.PostTokenBalances = append([]SolanaTokenBalance(nil), tx.Transaction.Meta.PostTokenBalances...)
	return clone
}
