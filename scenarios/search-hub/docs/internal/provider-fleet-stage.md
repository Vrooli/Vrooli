# Provider-description fleet stage

Search Hub's provider registry is the source owner for its description-routing
vectors. `internal/experimentarms.ProviderDescriptionFleetSource` snapshots
the routing-profile projection from `internal/registry`, embeds the same
segmentation used by `internal/routing.EmbeddingDescriptionIndex`, and reads
durable update/tombstone rows produced by the provider registry triggers.
Snapshots are bound to the namespace-injected arm registry and the source
profile digest is verified before replay.

The production route is
`POST /v1/embedding-experiments/arms/{armID}/generation`. It uses the immutable
recipe stored on the arm and an arm/recipe deterministic source snapshot. It
does not accept request-provided recipe, source, collection, snapshot, or peak
bytes. It requires the lifecycle to compose:

1. `FleetDescriptionEncoder`, with a complete verified recipe matching the
   arm's immutable recipe;
2. `FleetStagePeakEstimator`, derived from the source point count and recipe;
3. `aisearch.FleetAdmissionGate`, which produces fresh exact-operation
   admission consumed by the shared Qdrant fleet adapter.

`api/main.go` currently constructs the route through `Registry.NewFleetGenerationStage`
with no dependencies. The constructor reports the missing trusted dependencies
and returns nil, so the handler returns `503 unavailable` before reading or
mutating Qdrant. This is intentional and records the current owner API gap:

- Search Hub has no full-recipe local embedding runtime adapter. Its existing
  description-index CLI embedder exposes only the legacy recipe string and
  cannot attest artifact digest, tokenizer, runtime build, or precision.
- Search Hub has no production implementation of `FleetAdmissionGate` and no
  trusted peak-byte estimator wired to the resource capacity owner.
- The Qdrant resource's currently available supported read surface does not
  establish the live collection inventory, active alias, ownership, or bytes.
- The candidate fleet adapter has no steady-state query/upsert/delete API yet;
  staging does not switch `EmbeddingDescriptionIndex`'s live query path.

Do not enable physical operations, infer capacity from environment variables,
or count fake-owner tests as live migration, disk, retrieval-quality, or
reclaimed-byte evidence. Enabling the route requires the exact dependencies
above and an independently reviewed owner integration.
