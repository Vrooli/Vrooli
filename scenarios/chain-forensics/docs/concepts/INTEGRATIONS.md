# Integrations — Chain Forensics

## Public data providers

| Provider | Use | Terms/limits | Fallback |
|---|---|---|---|
| [mempool.space REST](https://mempool.space/docs/api/rest) | BTC address, transaction, and block observations | Keyless public read API; shared service, no SLA. Bound/cool down requests, cache immutable transaction bodies, honor HTTP failures. No broadcast endpoints. | Serve verified local capture with stale-source marker; allow operator-configured self-hosted Esplora later. |
| [Blockscout API](https://docs.blockscout.com/devs/apis) | EVM reads only after an eligible no-key endpoint is verified | Do not rely on Pro/keyed routes; never treat raw JSON-RPC as indexed address history. Use known contracts and bounded block ranges. | Expose history as incomplete rather than silently omit it. |
| [Public Ethereum JSON-RPC](https://ethereum.org/developers/docs/apis/json-rpc/) | Chain IDs, block headers, receipts, bounded logs | Keyless public endpoints vary in rate limits and retention; no indexing guarantee. No new credentials. | Cache captured facts; mark scan bounds and missing intervals. |
| [Base public RPC](https://docs.base.org/base-chain/api-reference/ethereum-json-rpc-api) | Base native and ERC-20 logs | Read-only public RPC; no SLA or arbitrary historical address index. Keep `eth_getLogs` bounded to known contract and block ranges. | Same incomplete-range behavior as Ethereum. |
| [Solana public RPC docs](https://github.com/solana-foundation/developer-content/tree/main/docs/rpc) | Finalized transaction/signature reads; SPL and Pump-style instructions | Shared endpoint is rate-limited and not production infrastructure. Use finalized commitment, exponential backoff, and explicit history gaps. The canonical docs site returned HTTP 429 during local link validation. | Reuse cached responses or report unavailable. |

Across and Wormhole joins use published protocol documentation plus on-chain
source/destination events; hosted tracker APIs are not required. Pump-style
parsing uses the [published Pump program docs](https://github.com/pump-fun/pump-public-docs)
and transaction instructions. These specs are schemas/decoding references, not
proof that a wallet has a real-world owner.

All provider access is free, keyless, read-only, and rate-aware. No paid
indexers, account-gated endpoints, credentials, external contact, transaction
broadcasting, or upload of packet contents is in scope. Provider terms and
endpoint reachability must be rechecked before each new adapter is promoted.

## Internal composition

The scenario owns persistence, adapters, cases, and private packets. A
deterministic Go library owns chain-neutral evidence contracts, graph building,
and explainable confidence grades. The target external consumer is Crypto
Infrastructure CI2; its repository path is currently absent, so an integration
stub here would not count as consumer evidence.

