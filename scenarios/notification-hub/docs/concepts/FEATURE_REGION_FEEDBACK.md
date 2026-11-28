# Region feedback attachments

Notification Hub accepts a region-feedback directory as a capture packet. This is an EX12
consumer contract. It does not change who produces EX9 bundles: BAS remains their owner.

## Attachment layout

Attach one folder with this shape:

```text
capture-review/
  ux-region-feedback.json
  captures/before.png
  captures/after.png
  regions/crops/<crop-name>.png
  regions/<region-id>.json
```

`ux-region-feedback.json` uses `ux-region-feedback/v1` with `kind: "bundle"` (older manifests
without `kind` remain readable). Its `before` and `after` captures each
name an immutable capture ID, a measured `sha256:<64 lowercase hex>` build identity, route,
interaction-state ID, data ID, fidelity label, viewport dimensions, and screenshot path plus
raw SHA-256. Fidelity is exactly `captured`, `composited`, `rendered`, or `sketch`. Every image
and crop is an attached PNG, JPEG, or WebP whose path, media type, size and SHA-256 agree with
the folder manifest. Screenshots are limited to 20 MiB each, crops to 5 MiB, and the compare
manifest to 1 MiB.

Each selectable region has a stable safe ID, outline node ID, human label, component name,
crop evidence ID, normalized image bounds, and attached crop path plus digest. The matching
`regions/<id>.json` file is the target for Hub's existing attachment-note format. The folder is
needed because Hub validates a note's path against an attached folder entry; this preserves
compatibility with existing answers and adds no new protobuf fields.

```json
{
  "schema": "ux-region-feedback/v1",
  "kind": "bundle",
  "id": "compare-2026-11-15-01",
  "route": "/settings",
  "state_id": "account-panel-open",
  "data_id": "settings-fixture-7",
  "before": {
    "id": "capture-before",
    "build_id": "sha256:<64 lowercase hex digits>",
    "route": "/settings",
    "state_id": "account-panel-open",
    "data_id": "settings-fixture-7",
    "fidelity": "captured",
    "viewport": {"width": 390, "height": 844},
    "image": {"path": "captures/before.png", "sha256": "<64 lowercase hex digits>"}
  },
  "after": {
    "id": "capture-after",
    "build_id": "sha256:<64 lowercase hex digits>",
    "route": "/settings",
    "state_id": "account-panel-open",
    "data_id": "settings-fixture-7",
    "fidelity": "composited",
    "viewport": {"width": 390, "height": 844},
    "image": {"path": "captures/after.png", "sha256": "<64 lowercase hex digits>"}
  },
  "regions": [{
    "id": "settings-nav",
    "node_id": "outline-node-23",
    "label": "Settings navigation",
    "component": "BottomNav",
    "evidence_id": "crop-23",
    "bounds": {"x": 0.04, "y": 0.78, "width": 0.92, "height": 0.16},
    "crop": {"path": "regions/crops/settings-nav.png", "sha256": "<64 lowercase hex digits>"}
  }]
}
```

The same `ux-region-feedback/v1` contract also has `kind: "correction"`. That event carries the
same capture identity fields and the same region object (ID, outline node, component, crop
path/digest and normalized bounds), plus the one-sentence note, exact owner scenario and private
local corpus destination. Its complete JSON Schema is
[`region-feedback-v1.schema.json`](../../../ui-health/docs/region-feedback-v1.schema.json).
The correction producer's crop preview URL is local presentation state and is never added to the
event. Hub packet answers retain the exact `regions/<region-id>.json` path and text; consumers
bind those to the bundle attachment and packet ID when dispatching the shared correction event.

## Operator flow

Open a folder in the packet viewer and select `ux-region-feedback.json`. The Hub displays the
before and after screenshots with their fidelity labels, route, viewport and build. A route,
state, data or viewport mismatch is called out and the screenshots are not described as a direct
comparison. The after image has region hit targets; the list below it is also keyboard- and
touch-answerable. Selecting a region shows its crop and component/node identity. The operator
adds the correction as one note on `capture-review/regions/<region-id>.json`, then sends it with
the decision. The existing Hub answer and resolution event carries the exact folder path and
text; the packet itself carries the capture/build/region identity.

The packet checker validates the manifest and stored bytes. Warnings are advisory under current
Hub policy and have stable `region_feedback_invalid` and `region_feedback_incomplete` codes.

**Limits:** this consumer contract is source work. It does not create or synthesize an EX9
capture. A real phone or flagship claim needs a privacy-safe, authorized EX9 packet, and EX9
producer changes remain with its owner.
