# Flows — Chain Forensics

## Published-case reconstruction

1. The operator opens an existing public case by stable case ID.
2. Case analysis loads the public source references and known address set.
3. Source acquisition reads bounded public endpoints, or serves the verified
   local capture if the source is unavailable. It records retrieval time,
   canonical URL, raw response hash, and provider completeness.
4. The normalizer emits chain-specific transaction/output occurrences with
   exact atomic quantities. Invalid or unsupported records remain a visible
   source gap.
5. The graph builder links only explicit transaction inputs and outputs. Any
   case narrative link (for example the complaint's identified relationship)
   remains a separate assertion edge with a filing citation.
6. The grader assigns evidence kinds and rule versions. It does not infer a
   natural person from an address or a service label.
7. The UI and CLI render the same graph, showing source claims, derived links,
   gaps, and counter-explanations.
8. The operator may create a private packet. It is stored locally and has no
   publish, submit, or contact action.

## Bridge correlation (deferred)

The source-chain decoder emits a protocol message occurrence. A destination
decoder must independently emit the matching protocol identity. A join must
check chain pair, bridge contract/program, message ID, asset, normalized amount,
and recipient constraints. Source-only events remain pending; timestamp or
amount similarity alone never creates an edge. Across and Wormhole are the
initial protocol targets, decoded from on-chain events and public specs rather
than hosted tracking APIs.

## Exchange deposit clustering (deferred)

Cluster rules run against sourced public exchange deposit labels and explicit
transaction evidence. Each versioned result records inclusion rules,
exclusions, source freshness, and counter-evidence. A cluster conveys possible
service routing only; no statement of common ownership or natural-person
identity follows from membership.

