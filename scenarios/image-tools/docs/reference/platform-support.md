# Image Tools platform support

`platform-support-matrix.json` is generated from the current Image Tools
capability list, an observed fleet snapshot and platform-runner receipts. The
target list names the platforms that need qualification; it makes no support
claims. The generator is owned by Image Tools at
`api/internal/platformmatrix`.

## Generate the matrix

From `scenarios/image-tools/api`, run:

```sh
go run ./cmd/platform-matrix \
  --capabilities ../../../../.vrooli/research/topics/image-tools-foundation/CAPABILITIES.md \
  --targets ../docs/reference/platform-targets.json \
  --fleet ../docs/reference/fleet-platform-snapshot.json \
  --receipts ../docs/reference/platform-receipts \
  --out ../docs/reference/platform-support-matrix.json
```

The fleet file uses the `vrooli.fleet-platform-snapshot/v1` shape. Refresh it
from the fleet owner's `vrooli-bridge nodes list --json` observation; include
only node identity, OS, architecture, revision when reported, online state,
dispatchability, and explicitly observed accelerator facts. Do not infer an
accelerator from OS or architecture. The checked-in snapshot records the
observation time and is only valid for that observation.

Each receipt file in the receipt directory uses
`vrooli.image-tools.platform-receipt/v1` and records one node, target and source
revision. It contains a `package_check` plus per-capability raw command results.
Each check carries an exit code, durable evidence path and lowercase SHA-256 of
that artifact. A receipt is rejected if its node is absent or mismatched in the
fleet snapshot, its source revision differs from the fleet observation, the
capability is not in the source list, or required evidence metadata is missing.
Receipts describe actual packaging and capability checks; do not create them by
hand or convert a compile-only result into a capability check.

## Status meanings

| Status | Meaning |
|---|---|
| `qualified` | Matching node revision has a successful desktop package check and successful per-capability check, each with evidence references. |
| `failed` | A matching receipt records a failed package or capability check; evidence remains linked. |
| `unqualified` with `no node` | No fleet node matches that target, including required accelerator facts. |
| `unqualified` with `no passing per-capability receipt` | A matching node exists, but there is no complete successful package and capability receipt. |

The generated matrix is a claim boundary: a missing receipt never means
"supported." Current committed input observes Linux amd64 and Darwin amd64
nodes, but no Image Tools packaging/capability receipts. Linux amd64 therefore
remains unqualified. Linux arm64, macOS arm64 with Metal, and Windows amd64
currently read `unqualified` / `no node`.
