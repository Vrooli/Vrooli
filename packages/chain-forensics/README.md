# Chain Forensics Go package

`github.com/vrooli/chain-forensics` is the transport-independent evidence
contract used by the chain-forensics scenario API, CLI and UI, and intended
for Crypto Infrastructure CI2. It contains
exact decimal atomic quantities, chain-specific occurrence identities,
provenance records, separate legal claims, typed evidence grades, strict case
validation, Bitcoin transaction normalization, exact UTXO outpoint joins,
Ethereum/Base/Arbitrum EVM transfer normalization, protocol-correlated bridge
flows, and an explicit private packet serializer. Its read-only mempool.space client
limits each transaction response to 1 MiB and each address page to 4 MiB, spaces uncached requests, keeps a 128-entry/10-minute
process cache of confirmed transactions by txid, and a separate 16-page/10-minute
cache for confirmed address-history pages. Address queries accept an explicit
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

`MergeBridgeCorrelationEdges` provides two exact protocol adapters. The OP
Stack adapter matches an L2 `MessagePassed` withdrawal hash to a successful L1
`WithdrawalFinalized` event and requires source/destination native transfer
occurrences backed by retained transaction/receipt and internal-call sources.
The Arbitrum Nitro adapter matches the L2 `ArbSys.L2ToL1Tx` position, sender,
target, and call value to the L1 Outbox `executeTransaction` tuple and retained
internal transfer. Both write `protocol_correlated` edges only when every
field and source body agrees; address reuse, equal amounts, and timing alone
do not create bridge edges. These edges prove protocol message continuity, not
real-world identity, ownership, intent, or exchange-account control.

## Offline public case

`examples/cases/us-v-1hq3/case.json` is a curated excerpt from the U.S.
forfeiture complaint for the 1HQ3 address. The raw DOJ complaint PDF is bundled
under `examples/cases/us-v-1hq3/complaint-captures/` and its file hash is checked
against timestamped provenance. Two full mempool.space transaction
responses are included under `examples/cases/us-v-1hq3/source-captures/` with
retrieval time and SHA-256 in the typed case record. The 2013 confirmed
transaction has 54 inputs (31 from 1BAD and 23 from 1BBqj), one output of
69,471.08220100 BTC to 1HQ3, and an exact total of 6947108220100 satoshis.
The case records each input and its membership in that single-output
transaction. It does not claim to reconstruct the 54 earlier Silk Road
transfers alleged in the complaint. The complaint narrative is represented as
a `legal_claim`; it is not promoted into a chain observation. The case also
records the exact input/output relationship in an April 2015 spend. The 101 BTC
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

Solana acquisition uses `CHAIN_FORENSICS_SOLANA_RPC_URL` (defaulting to
PublicNode's keyless mainnet RPC after the official shared endpoint returned
429 during the live bounded capture) and requests one finalized page of at most 25
signatures for an address. Transactions are fetched with `jsonParsed`, bounded
to 1 MiB each, rate-spaced, and cached for ten minutes with a 128-transaction
and 16-page cap. SPL observations come only from exact pre/post token-account
atomic balance deltas; failed transaction signatures are retained as sources
without token transfers. Missing block times remain explicitly unknown. A 429
and provider `Retry-After` are returned to the caller without automatic retry.
Pump program identification must use the documented on-chain program and
instruction/account evidence, not an unofficial trading API; no key-bearing or
write API is used.

The public fixtures are illustrative, bounded evidence—not a complete case
history. The CI2 package consumer does not yet establish deployment anonymity,
decentralization or resistance to the full adversary set.

