# Source Ledger proto contracts

These schemas are the wire contract for the extracted, scope-aware ledger
service. They are authored here and generated into Go, Connect-Go,
TypeScript, JavaScript, and Python bindings by the repository proto pipeline.

The five domain services deliberately keep the engine boundary explicit:
journal, recall, forest, facets, and scopes. Every request message carries a
scope field, including scope-registry operations; the service validates the
field before touching storage. Health and error messages are shared contracts.

Run `cd packages/proto && make generate && make lint` after changing these
schemas. Generated files under `packages/proto/gen/` are outputs and must not
be hand-edited.
