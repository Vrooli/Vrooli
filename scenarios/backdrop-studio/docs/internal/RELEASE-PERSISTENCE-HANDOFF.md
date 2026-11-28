# Backdrop Studio release persistence handoff

Status: implemented in the bounded Backdrop Studio release owner; no real
candidate was released. Verified 2026-09-15 after a governed scenario restart;
Backdrop Studio is healthy on its assigned API/UI ports.

## Durable contract

- `api/internal/release/release.go` is the owner of released backdrop bytes and
  metadata.
- A qualified release is stored under the routed `data` root at
  `released-backdrops/<release-id>/`.
- `asset.png`, `metadata.json`, and `metadata.sha256` are prepared with
  `storage.WriteFileAtomicInRoot` and published by an atomic rooted directory
  rename. The final directory is immutable from the release API's point of
  view; a second content hash for the same release ID is refused.
- `GetContext` reloads the files after every request. It refuses missing or
  altered metadata, altered bytes, length/hash mismatch, non-PNG bytes, MIME
  mismatch, or dimension mismatch. Therefore the returned same-origin asset
  URI is backed by bytes that survive an API restart and are verified before
  serving.
- `filerouting.RoutedRoots.PickRequired` selects live data or an active
  test-mode lease. Test-mode release/read requests without a lease fail closed;
  leased artifacts cannot appear in the live tree.

The release wire response now includes `job_id` (field 16), `mime_type`
(field 17), and `content_hash` (field 18). `job_id` is copied only from the
render candidate evidence chain; the release request has no caller-controlled
job-proof field. The generated Go/TypeScript/Python/manifest artifacts were
published as `5d785a6c0816727ac4d459372d2dcac20b6fbc798b12b28c933628558efbdea7`.

## Validation evidence

Run from `scenarios/backdrop-studio/api`:

```text
go test ./internal/release ./handlers/release
go test ./internal/render -run 'Test(Candidate|ProceduralLane|ModelBacked)'
```

The focused persistence tests cover restart retrieval, byte tamper refusal,
metadata tamper refusal, isolation, and concurrent release calls. The existing
qualification tests remain in the same suite and continue to prove that the
render owner supplies candidate bytes and measurements rather than caller
booleans.

The focused owner suites passed. The selected render run completed in 10.063s.
The previously observed whole `internal/render` race run timed out in the
expensive `TestEverySeededProceduralStyleRenders/molten-terrain` PNG
compression case; this handoff does not claim that broad race suite passed.

The isolated owner qualification regression is
`handlers/release.TestReleaseReferenceAndAssetEndpointPreserveQualifiedFactsInLeasedRoot`.
It calls the real release handler and `GetReference`, fetches the returned
same-origin URI through the real asset HTTP handler, decodes the returned PNG,
and compares candidate bytes, SHA-256, dimensions, MIME type, job, style,
surface, placement, measured ratio/threshold, and reserved regions. The test
uses a disposable leased routed root and asserts zero primary writes. The
publisher-context regression is
`internal/release.TestModelBackedReleasePassesRequestContextToPublisher`;
model-backed handoff now receives the request/test-mode context rather than a
new background context.

For an optional machine-readable test receipt, set
`BACKDROP_QUALIFICATION_RECEIPT_PATH` when running the handler test. The
default is no receipt write; the emitted receipt is explicitly marked
`isolated-owner-qualification` and is not a release or publication receipt.

Validation run on 2026-09-15:

```text
go test ./internal/release ./handlers/release
go test -race ./internal/release ./handlers/release
```

Both commands passed. The opt-in receipt probe also passed with a temporary
output path and recorded `test_root_writes: 1` and
`primary_writes_during_test_mode: 0`. No live Backdrop release, candidate
promotion, mobile-art qualification, or public activation was performed.

The authorized durable test-owned receipt from the same fixture run is
`.vrooli/plan-artifacts/efforts/aquila-configurable-product-presentation/evidence/backdrop-isolated-owner-qualification-20260915.json`.
It is explicitly an isolated qualification receipt, not evidence that a real
Backdrop candidate was released or promoted.

The parent independently reran the handler qualification under `-race` and retained
the canonical effort receipt at
`/home/matthalloran8/.vrooli/plan-artifacts/efforts/aquila-configurable-product-presentation/evidence/backdrop-isolated-owner-qualification-parent-20260915T1104.json`.
Receipt creation is exclusive and occurs after all assertions. The earlier worker
receipt above is repository-relative and remains preserved. The parent also passed
`go test . -run '^TestReleaseCLIUsesOwnerQualificationAndPropagatesRejection$' -count=1`
from the Backdrop CLI module: the CLI does not manufacture measurements or a
passing boolean, and propagates the owner's rejection.

## Parent integration constraints

Landing Page Business Suite owns the presentation asset join. It may consume
the released backdrop's same-origin `uri` and metadata only after a live GET
returns the expected bytes and verification headers (`Content-Type`,
`Content-Length`, `ETag`, `X-Content-SHA256`, and immutable cache policy).
Backdrop Studio does not implement the LPBS join, web-console branding, or
publication workflow here.

The current candidate source is still process-memory-backed. A candidate render
that was never released before an API restart is not recoverable by this
bounded persistence repair; candidate durability is a separate owner decision.
No real-candidate publication, paid generation, deployment, team activation,
or descendants are part of this handoff.
