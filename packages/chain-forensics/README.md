# Chain Forensics Go package

`github.com/vrooli/chain-forensics` is the transport-independent evidence
contract intended for scenario APIs and Crypto Infrastructure CI2. It contains
exact decimal atomic quantities, chain-specific occurrence identities,
provenance records, separate legal claims, typed evidence grades, strict case
validation, Bitcoin transaction normalization, exact UTXO outpoint joins,
Ethereum/Base ERC-20 Transfer-log normalization, and an explicit private packet serializer. Its read-only mempool.space client
limits response size, spaces uncached requests, caches confirmed transactions
by txid, and returns provider retry delays without automatic retries. It
performs no identity resolution, transaction signing, or network writes.

ERC-20 normalization consumes an already captured JSON-RPC log plus its
containing block timestamp. It checks the canonical Transfer event signature,
indexed-address padding, confirmed block position, transaction hash and source
capture. Token contract and exact integer base units are retained; `scale: 0`
means trusted token decimals have not yet been joined. This is a decoding
primitive, not an RPC source client or a claim of address ownership.

## Offline public case

`examples/cases/us-v-1hq3/case.json` is a curated excerpt from the U.S.
forfeiture complaint for the 1HQ3 address. Two full mempool.space transaction
responses are included under `examples/cases/us-v-1hq3/source-captures/` with
retrieval time and SHA-256 in the typed case record. The complaint narrative is
represented as a `legal_claim`; it is not promoted into a chain observation.
The BTC facts include a 2013 output to 1HQ3 and the exact input/output
relationship in an April 2015 spend. The 101 BTC destination remains an
unverified service label; this package makes no ownership, identity, intent,
or liability claim.

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

The package CLI is a runnable Go-library surface. The scenario's registered
Vrooli CLI, Connect API, and UI are still unconnected and remain required.

The public fixtures are illustrative, bounded evidence—not a complete case
history. CI2 is not yet an importing consumer. Do not count this package as
integrated until its owner-owned evaluator imports it and exercises a real
test.

