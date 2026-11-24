# Root region-correction provider

`RegionCorrectionProvider` is the authored RCL component at
`library/components/region-correction-provider/`. It wraps an application's root, so it does not
depend on `AppShell`. The active draft is `0.1.0-draft.1`; it is not yet a published package
version.

## Host integration

The host supplies two callbacks:

- `resolveEvidence(element)` maps the selected DOM element to one exact outline node, RCL/custom
  component, capture/build identity, route/state/data identity, viewport, crop, and private local
  evidence path. It returns `freshness: "current"` only after comparing the capture build with the
  currently served build. When that comparison fails, return `freshness: "stale"`; the provider
  explains the stale capture and sends nothing.
- `submitCorrection(event)` posts the provider's `ux-region-feedback/v1` correction object to the
  local UI Health intake and returns its durable receipt. The receipt must name the same correction
  ID, owner scenario, corpus, and exact correction body. The provider reports success only for that
  matching `queued` receipt; an unconfirmed response stays visible and is safe to retry with the
  same correction ID.

Wrap the actual app root once:

```tsx
<RegionCorrectionProvider
  resolveEvidence={resolveEvidence}
  submitCorrection={submitCorrection}
>
  <App />
</RegionCorrectionProvider>
```

The host owns route and data identity. Configure the exact owner scenario and private EX1 corpus
in that host; do not accept either value from page text or operator input. Mark custom hit targets
with `data-ux-feedback-target`. Elements that already expose both `data-ux-node-id` and
`data-ux-component` can be selected directly. Interactive controls require the explicit marker.
Ordinary touch movement cancels the long-press; the report button offers a tap-and-select path,
and the keyboard shortcut opens the focused region.

The provider preview URL is same-origin presentation state and is never part of the correction
event. The event carries evidence paths and digests, not image bytes; the host must keep those
artifacts available through the local evidence owner. Do not send evidence to a remote service or
include client, third-party, or real-user media.

## Receipt boundary

UI Health validates the shared correction schema and atomically stores the exact event in its local
owner and EX1 queues. Its receipt confirms `queued` persistence and includes the correction body
and event digest. It does not claim that an owner agent has opened or completed the fix, or that EX1
has created a tested rule. Keep those states distinct in any host UI.

Hub packet answers use a different delivery path: the answer retains the attachment name,
`regions/<id>.json` path, and text, then Research Desk carries those exact fields to the implementing
agent. A Hub packet answer is not a served-page correction receipt.
