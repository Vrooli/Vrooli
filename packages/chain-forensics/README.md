# Chain Forensics Go package

`github.com/vrooli/chain-forensics` is the transport-independent evidence
contract used by the chain-forensics scenario API, CLI and UI, and intended
for Crypto Infrastructure CI2. It contains
exact decimal atomic quantities, chain-specific occurrence identities,
provenance records, separate legal claims, typed evidence grades, strict case
validation, Bitcoin transaction normalization, exact UTXO outpoint joins,
Ethereum/Base/Arbitrum EVM transfer normalization, protocol-correlated bridge
flows, and an explicit private packet serializer. Its read-only mempool.space client
limits each transaction response to 1 MiB and each address page to 4 MiB. The
Bitcoin, EVM and Solana readers share process-wide typed caches built on
one package-owned `CaptureCache[T]` implementation, plus one provider-origin
rate limiter. Cache keys
include provider/network identity and request cursor where relevant; cache
capacities and expiry match the chain's finality and reorganization window.
The default limiter serializes reads per provider origin, uses the longest
requested minimum interval, bounds its registry to 64 origins, and lets queued
calls cancel through context. Each typed cache pool also has a 64 MiB estimated
weight ceiling (320 MiB across the five pools) plus its entry cap. A `429` is
surfaced with `Retry-After`; when supplied, the delay pauses later reads from
the same provider origin across all clients and chains. No reader automatically
retries. The mempool client
keeps a 128-entry/10-minute cache of confirmed transactions and a separate
16-page/10-minute cache for
confirmed address-history pages. Address queries accept an explicit
last-seen transaction cursor and return at most one 25-transaction page per
request. The address-page bound accommodates large historical consolidations
while retaining the provider's fixed 25-transaction page limit. Both paths
return provider retry delays without automatic retries.
See the [mempool.space REST reference](https://mempool.space/docs/api/rest#addresses)
for its shared-endpoint rate limits and page contract. The package performs no
identity resolution, transaction signing, or network writes. Captured sources
retain exact provider result bytes and, for JSON-RPC clients, the successful
HTTP response envelope with separate SHA-256 digests. The SQLite case document
carries those bytes so packet exports can be checked against normalized facts.

ERC-20 normalization consumes an already captured JSON-RPC log plus its
containing block timestamp. It checks the canonical Transfer event signature,
indexed-address padding, confirmed block position, transaction hash and source
capture. Token contract and exact integer base units are retained; `scale: 0`
means trusted token decimals have not yet been joined. This is a decoding
primitive, not a claim of address ownership. `EVMRPCClient` reads a confirmed
transaction receipt, its matching block, and the transaction object; it verifies
their chain position and adds a native ETH occurrence only when the signed
transaction carries non-zero value. It also extracts canonical ERC-20 Transfer
logs, bounds response size, spaces calls, and caches normalized results for 30
seconds (up to 256 entries) because a
recent receipt can reorganize. It preserves a 429 `Retry-After` without
automatic retries. Each source hashes and retains both the JSON-RPC result
object and the exact successful HTTP envelope. The scenario appends sources and normalized
occurrences with optimistic case-version checking. No signing or submission
method exists.

`MergeBridgeCorrelationEdges` provides protocol adapters for OP Stack withdrawals,
Arbitrum Nitro withdrawals, Wormhole Token Bridge transfers, and deBridge DLN
orders. The OP
Stack adapter matches an L2 `MessagePassed` withdrawal hash to a successful L1
`WithdrawalFinalized` event and requires source/destination native transfer
occurrences backed by retained transaction/receipt and internal-call sources.
The Arbitrum Nitro adapter matches the L2 `ArbSys.L2ToL1Tx` position, sender,
target, and call value to the L1 Outbox `executeTransaction` tuple and retained
internal transfer. Both write `protocol_correlated` edges only when every
field and source body agrees; address reuse, equal amounts, and timing alone
do not create bridge edges. These edges prove protocol message continuity, not
real-world identity, ownership, intent, or exchange-account control.

The deBridge adapter correlates the documented EVM `CreatedOrder` and
`FulfilledOrder` events by indexed `orderId`. It validates the source and
destination proxy addresses, event topics, complete bounded ABI tuple, and
the `giveChainId` / `takeChainId` fields against chain identities in the
normalized source and destination occurrences. Supported EVM chain IDs are
Ethereum `1`, Base `8453`, and Arbitrum `42161`. It decodes EVM token addresses
only (20-byte order address fields); Solana and other chain engines remain
unresolved. Give and take tokens and amounts stay separate. A match is
protocol correlation, not equal-unit, equal-value, or beneficial-owner
evidence. Malformed offsets, oversized data, unknown chains, duplicate
candidate transfers, and mismatched order terms produce no edge. Deployment
addresses and event identities come from the primary
[`DlnSource` contract](https://github.com/debridge-finance/dln-contracts/blob/main/contracts/DLN/DlnSource.sol),
[`DlnDestination` contract](https://github.com/debridge-finance/dln-contracts/blob/main/contracts/DLN/DlnDestination.sol),
and [deployment table](https://github.com/debridge-finance/dln-contracts#dln-smart-contracts).
A private unrelated Ethereum→Arbitrum candidate is retained in the external
CF6 research run at
`~/.vrooli/research/topics/crypto-foundations/runs/debt-crypto-models-20261112/evidence/packages-chain-forensics/cf6-dln-live-order-candidate-20261116/`.
The replay inputs used by this package's regression are kept separately under
`testdata/cf6-dln-live-order-candidate-20261116/`.
Its exact raw receipts reproduce one order ID, full terms, percent-fee-adjusted
source transfer, actual destination fill and receiver. The destination receipt
uses the verified amount-bearing selector from the exact explorer transaction
ABI. Partial-fill, wrong-order, altered-body, duplicate and unrelated-flow
controls are covered in both fixture tests and a live-candidate regression.
This order has no established relation to the private rug lead, so it advances
adapter validation only and does not close the two-bridge/exchange showcase.

## Offline public case

`examples/cases/us-v-1hq3/case.json` is a private reconstruction of the
published 1HQ3 flow set. The raw DOJ complaint PDF is bundled under
`examples/cases/us-v-1hq3/complaint-captures/` and its file hash is checked
against timestamped provenance. Full mempool.space 2013/2015 transaction
responses and four confirmed address-history pages are included under
`examples/cases/us-v-1hq3/source-captures/`, each with retrieval time and
SHA-256 provenance. The four pages reproduce all 54 May 6, 2012 outputs to
the two complaint-named addresses, 1BAD and 1BBqj, totaling 70,411.46 BTC.
The package regression replays exact output indexes, addresses, satoshis,
timestamps and block heights from those raw pages; this establishes capture
coverage only for the retained page set. The 2013 confirmed
transaction has 54 inputs (31 from 1BAD and 23 from 1BBqj), one output of
69,471.08220100 BTC to 1HQ3, and an exact total of 6947108220100 satoshis.
The case records each input and its membership in that single-output
transaction. Complaint attribution remains a `legal_claim`, separate from the
54 observed outputs: the captured public chain pages do not establish that
Silk Road sent them. The case also records the exact input/output relationship
in an April 2015 spend, plus the later exact join of the 101 BTC outpoint into
a 25-input transaction. The 101 BTC
destination remains an unverified service label; this package makes no
ownership, identity, intent, or liability claim.

The case also bundles matching Blockstream Esplora responses under
`examples/cases/us-v-1hq3/counterexample-captures/`. Its reproducibility test
checks each raw capture against its timestamped SHA-256 provenance and compares
the normalized transaction facts across both providers. For the 2015 response,
mempool.space reports `sigops: 8` while Esplora omits that optional field; the
txid, block, time, input outpoint/value, and outputs agree. This narrow provider
difference does not change the flow. A separate counterexample prevents
transaction-level attribution: the same input also returns 69,370.08220100 BTC
to 1HQ3, so the 101 BTC output cannot stand for the entire input flow. The
private packet retains that counterexample and the provider difference. It
still does not independently verify the complaint's BTC-e destination label.

Run the deterministic replay and validation tests from this directory:

```bash
go test ./...
```

Inspect the bundled case as JSON or explicitly write a private local packet
without overwriting an existing path:

```bash
go run ./cmd/chain-forensics case inspect us-v-1hq3-public-forfeiture --json
go run ./cmd/chain-forensics case packet us-v-1hq3-public-forfeiture --out /tmp/1hq3.packet.json
```

The package CLI is a runnable Go-library surface. The scenario connects this
library to SQLite case storage, Connect, its registered Vrooli CLI and a
case-review UI. Through `chain-forensics cases capture-bitcoin`, a confirmed
keyless mempool.space transaction can be attached directly to a case; exact
previous-output references become observed flow edges only when both endpoints
exist in the dossier. Ethereum and Base receipt capture are also exposed
through the served API and CLI, including one confirmed Arbitrum transaction.
Native-value transfers and ERC-20 receipt logs use the same evidence model.
The first package-backed Crypto Infrastructure consumer now lives in
`packages/crypto-infrastructure-evaluator`; it validates typed cases and
private packet digests and reports flow-grade coverage. This is an integration
increment, not completion of the CI2 evaluator showcase, which still requires
observer/control inventories, adversary models, automated linkage attacks and
three deployment proving targets.

An EVM wallet snapshot reads `eth_chainId`, the latest block, native balance,
and selected ERC-20 `balanceOf`/`decimals` values at that pinned block number;
it then rereads the block hash and refuses to return a mixed snapshot after a
reorganization. The scenario exposes this through
`chain-forensics cases capture-evm-wallet --case-id <id> --chain base --address <address> [--token-contracts <a,b>]`.
Snapshots contain exact atomic strings, chain/address/block identity, token
scale when the contract reports it, and source IDs. They do not add a fiat
valuation or prove Treasury control of an address. Configure
`CHAIN_FORENSICS_BASE_RPC_URL` for an isolated local fork and
`CHAIN_FORENSICS_ETHEREUM_RPC_URL` or `CHAIN_FORENSICS_ARBITRUM_RPC_URL` for a
custom endpoint; defaults are keyless public RPCs. No transaction-signing or
broadcast method is exposed.

Solana account-history and known-signature acquisition uses
`CHAIN_FORENSICS_SOLANA_HISTORY_RPC_URL` (defaulting to the keyless official
`api.mainnet.solana.com` endpoint; the distinct `api.mainnet-beta.solana.com`
endpoint previously returned HTTP 429 for history reads). A caller can fetch
up to 25 sequential finalized signature-only pages of up to 1,000 entries each
in one append to discover older transactions without downloading every
transaction body, or use the normal
address page for at most 25 signature-plus-transaction captures. The exact
signature result and HTTP envelope are retained; a signature page alone is not
transaction-level transfer evidence. A caller can also fetch one known finalized
signature directly with `SolanaRPCClient.FetchTransaction`; this avoids
enumerating address history, keeps the external signature as a lookup key only,
and records that its selector page is not part of the evidence. Transactions are fetched with raw
`json` encoding, bounded to 1 MiB each, rate-spaced, and cached for ten minutes with
a 128-transaction
and 16-page cap. SPL observations come only from exact pre/post token-account
atomic balance deltas; native SOL observations include exact pre/post lamport
balance deltas. Both carry `measurement=net_account_balance_delta` and the
measured account, separate from explicit transfer occurrences because a net
delta can combine transfers, fees, rent and program effects. Bridge correlators
reject these deltas as transfer evidence. Failed transaction signatures are
retained as sources without token transfers. Missing block times remain explicitly unknown. A 429
and provider `Retry-After` are returned to the caller without automatic retry.
Transaction source IDs include the exact result hash so the same signature from
another provider does not collide with an earlier source capture.
Raw account indexes are resolved against static and loaded message keys.
System Program `transfer` instructions and SPL Token / Token-2022 `transfer`
and `transferChecked` instructions are decoded from their instruction bytes;
provider-parsed instructions remain supported. Both forms are emitted as exact
`measurement=instruction_transfer` observations, including inner instructions.
Checked SPL transfers take mint, atomic amount, and decimals from the parsed
instruction; unchecked transfers require the transaction's balance metadata to
identify the mint. If a legacy transfer has no available mint context, the
capture retains its program, accounts, base58 instruction data, source ID and
resolution reason in `UnresolvedTransfers` without inventing an asset or amount.
These instruction observations bind the token-account source and destination
exactly, but do not identify account owners or establish a cross-transaction/
bridge attribution by themselves.

Owner inventory uses `CHAIN_FORENSICS_SOLANA_RPC_URL` (defaulting to the keyless
official `api.mainnet-beta.solana.com` endpoint) and makes one finalized
`getTokenAccountsByOwner` request per SPL program (`Tokenkeg…` or Token-2022)
with `jsonParsed`. It retains the exact RPC
result and HTTP envelope and stores token-account addresses separately from the
requested owner public key. Results are bounded to 10,000 accounts, cached for
two minutes, and rate-spaced by the shared Solana origin limiter. An RPC response
does not establish who controls an owner address or provide its transaction
history.
Pump program identification must use the documented on-chain program and
instruction/account evidence, not an unofficial trading API; no key-bearing or
write API is used.

The RPC transaction object also carries a narrowly versioned Pump `create_v2`
decoder. It matches the exact Pump program ID and the `create_v2` discriminator
from [`pump.json` at pinned commit `639b87fb`](https://github.com/pump-fun/pump-public-docs/blob/639b87fb45c3adeffa6f7c4a1bf0cb85e392bf22/idl/pump.json),
then checks the documented Borsh argument/account layout. Other Pump
discriminators and malformed matches stay unresolved. The decoded fields are
instruction facts, not claims about deployed code behavior or account owners.
The full research capture, pinned IDL bytes, hashes, limits, and replay
instructions are retained in the external CF6 research run at
`~/.vrooli/research/topics/crypto-foundations/runs/debt-crypto-models-20261112/evidence/packages-chain-forensics/cf6-pump-createv2-20261114/`.
The single raw transaction needed by the package regression is kept under
`testdata/cf6-pump-createv2-20261114/`.
The scenario Connect capture path persists decoded and unresolved Pump
instruction records with their source IDs in SQLite and private packet JSON;
the registered CLI reports their counts, and the case UI shows decoded facts
and unresolved discriminators. Replaying a capture is idempotent, and a
same-ID/different-content conflict is rejected. This keeps the decoder's
exact pinned-version boundary visible across the case path; it does not prove
deployed program behavior or account ownership.

The public fixtures are illustrative, bounded evidence—not a complete case
history. The CI2 package consumer does not yet establish deployment anonymity,
decentralization or resistance to the full adversary set.

