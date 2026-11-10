# Data — Chain Forensics

## Ownership and records

| Record | Owner | Key identity | Retention boundary |
|---|---|---|---|
| Case | Case analysis | Stable local case ID | Local SQLite; user may delete |
| Source capture | Source acquisition | Provider + request URL + capture hash | Cache immutable responses; refresh only mutable head data |
| Chain occurrence | Source acquisition | Chain + tx/signature + block/slot + event index | Rebuildable from captured source; never keyed by amount/time alone |
| Flow edge | Case analysis | From occurrence/output to destination occurrence/output | Derived, versioned, rebuildable |
| Label | Labels and clusters | Chain + address + label source + scope + observed-at | Preserve publisher and expiry/review status |
| Cluster candidate | Labels and clusters | Rule ID/version + member occurrence/address | Hypothesis with evidence and counter-evidence |
| Evidence packet | Case analysis | Case version + packet ID | Private local artifact; explicit export only |

Quantities are exact integer atomic units with an explicit asset and scale; do
not round through floating point. A transaction occurrence is not an address
identity, and a shared transaction input is not proof of common personhood.
Valuation (when later added) is a separate dated observation with its own
source.

## Provenance and confidence

Every external fact carries source name, canonical URL, retrieval timestamp,
response SHA-256, and relevant source record/transaction identifier. Legal
filings are stored as **published assertions** with paragraph/page reference;
they do not silently become chain facts. Every derived edge records the rule
version and parent evidence IDs. Cache provenance remains attached when the
network is offline.

Evidence grades describe what joins are supported, not the likelihood that a
person controls an address:

- **Observed**: a node or transfer is directly present in a chain response.
- **Protocol-correlated**: both sides match a protocol message/transfer identity
  under a versioned decoder.
- **Source-corroborated**: an independent public source links a public label to
  an address/event; this does not establish exclusive control.
- **Hypothesis**: a deterministic heuristic proposes a link; counter-evidence
  and competing explanations are retained.
- **Unresolved**: the source is missing, incomplete, contradictory, or rate
  limited. No inferred edge is emitted.

The first 1HQ3 slice is partial by design: it includes selected directly
verifiable BTC transactions and the DOJ complaint's public narrative. It does
not reconstruct all 277 address transactions, the 54 historical transfers, or
the private ownership dispute.

## Privacy boundary

The local store contains only public chain records, public source references,
and analyst-authored case notes. Packets are private local artifacts, default
to redacting analyst notes, and never send data to external destinations.
Telemetry must not include wallet lists, packet text, labels, source response
bodies, or user notes. There is no identity-enrichment provider, tip submission,
contact workflow, or public publishing route.

