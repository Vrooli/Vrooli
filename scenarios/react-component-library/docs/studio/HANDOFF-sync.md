# Handoff — cross-tab freshness (studio sync)

Status: module + hook + tests landed in the working tree, **not committed**; not yet wired into
`useStudio`. The coordinator owns integration. This closes remaining work **6** in `PLAN.md`
("cross-tab refresh").

## Why this layer exists

The `409` envelope from `HANDOFF-api.md` is a *save* safety net. It cannot announce:

- a design created in another tab (nothing saves, so nothing conflicts),
- an uploaded/removed asset,
- a rename, duplicate or delete.

Until now those changes required a reload. This layer broadcasts them and — more importantly —
decides what a receiving tab may do, so a live signal cannot become silent data loss.

## Files

| Path | Role |
|---|---|
| `ui/src/workbench/studio/sync.ts` | Event contract, validator, transport (BroadcastChannel + window fallback), `currentTabId`, `reconcileDesign` |
| `ui/src/workbench/studio/sync.test.ts` | Validator, every reconcile branch, both transports, close/detach, malformed payloads |
| `ui/src/workbench/studio/useStudioSync.ts` | Hook: coalescing, reconciliation dispatch, focus/visibility refetch, teardown |
| `ui/src/workbench/studio/useStudioSync.test.tsx` | Reload vs conflict, coalescing, focus/visibility, unmount, malformed events, latest-handler ref |

No production file outside this set was touched. No strings, CSS, locale, API or `useStudio` edits.

## Event contract

```ts
type StudioSyncEvent =
  | { type: "designs-changed"; source: string; designId?: string; revision?: number }
  | { type: "assets-changed"; designId: string; source: string }
  | { type: "template-changed"; source: string };
```

- `source` is **stamped by the transport**, never trusted from the caller. `post()` overwrites it
  with `currentTabId()`, so a tab cannot impersonate another. Receivers drop any event whose
  `source` equals their own id.
- `designs-changed` without `designId`/`revision` is the coarse signal: "the list moved"
  (create, duplicate, delete, rename). A tab refetches the summary list; it does **not** reload an
  open document.
- `designs-changed` **with** both `designId` and `revision` is targeted: "I saved this design and
  the server assigned revision N". This is what makes an idle tab reload an open document without
  waiting for a `409`. Both fields are required together; a half-written target is malformed.
  The optional pair is an **additive** extension of the brief's event type — without it a
  `designs-changed` event cannot "name the active design", and reconciliation would be impossible.
  Every shape the brief specified still type-checks unchanged.
- `assets-changed` names the design whose asset list changed.
- `template-changed` is posted for completeness. Templates are still tab-local in this slice and
  `useStudioSync` has no template callback, so receipt is a deliberate no-op (it does not even arm
  the coalescing timer). The coordinator may consume it later.

Transport selection, in order:

1. `BroadcastChannel(name)` when constructible — name `rcl.studio.sync` by default;
2. `window` `CustomEvent` with the same name when `BroadcastChannel` is missing **or** its
   constructor throws (private mode / policy), which is what jsdom uses;
3. an inert channel when there is no `window` (no crash outside a browser).

`subscribe` returns an unsubscribe; `close()` clears listeners, removes the DOM/BroadcastChannel
listener and closes the underlying channel. A closed channel ignores `post` and `subscribe`.

Every received payload passes `isStudioSyncEvent`. Malformed input — wrong type, missing `source`,
`assets-changed` without a `designId`, a string revision, `Infinity`, a primitive — is dropped
without throwing. One bad message cannot break the listener.

## Reconciliation rules

`reconcileDesign` is the safety contract. Order is deliberate:

| # | Condition | Decision | Why |
|---|---|---|---|
| 1 | `remoteId !== activeId` | `ignore` | Nothing on screen to update |
| 2 | `saving` | `ignore` | Our save will settle and broadcast its own revision |
| 3 | `dirty` | `conflict` | Never silently discard unsaved local work |
| 4 | `remoteRevision > activeRevision` | `reload` | The only safe automatic adoption |
| 5 | otherwise (equal or lower) | `ignore` | Replay/reorder, or nothing new |

`dirty` wins over a newer revision, and a save in flight wins over both: the save is already based
on a revision the server will answer, so turning it into a conflict would be wrong. A dirty design
with an *equal* revision is still a conflict, because that means the local document diverged
without being saved.

## Coalescing and refetch behaviour

`useStudioSync(handlers)` accumulates signals in a pending set and drains it on a single
250 ms timer (`STUDIO_SYNC_COALESCE_MS`). It is a throttle window, not a debounce: a continuous
stream cannot starve the callbacks.

Per flush:

- Each distinct `assets-changed` design is delivered **once**, and only when it equals
  `activeDesignId`. Only the active design's asset list is cached by the studio; a signal for
  another design is not actionable until it is opened (`useStudio` refetches on open). This was the
  documented choice from the brief: `onAssetsChanged` is not called for inactive designs.
- Targeted `designs-changed` for the active design runs `reconcileDesign` once using the **highest**
  revision in the window. `reload` → `onReloadDesign()`, `conflict` → `onConflict()`,
  `ignore` → no design callback.
- If any design signal arrived and `saving` is false, `onDesignsChanged()` runs exactly once. When
  `saving` is true the list refetch is skipped, because the in-flight save owns the next
  authoritative revision; its own broadcast settles the list afterwards.
- `focus` (window) and `visibilitychange` → `document.visibilityState === "visible"` each request a
  list refetch through the same 250 ms window. The refetch is skipped while `saving`, as specified.

Teardown: unmount sets a mounted guard, clears the timer, drops the pending set, removes both DOM
listeners, unsubscribes and closes the channel. A callback can never run after unmount, even if a
message was already queued.

Handlers live in a ref updated every render, so a render that flips `dirty`/`saving` does not
tear down and recreate the channel (which would drop messages in the gap).

## Wiring `useStudioSync` into `useStudio`

Illustrative, not applied — the coordinator owns `useStudio.ts`. `dirty` uses the existing
`docSignature` comparison; `onConflict` must refetch the remote document because the existing
banner needs `{ designId, current }`, and the sync event carries only the revision.

```ts
import { getDesign, listDesignAssets, listDesigns } from "../designs";
import { docSignature } from "./model";
import { useStudioSync } from "./useStudioSync";

// inside useStudio(), after docs/saved/saving/conflict state exist:

const adoptRemote = useCallback((remote: DesignDocument) => {
  setDocs((prev) => prev.map((doc) => (doc.id === remote.id ? remote : doc)));
  setSaved((prev) => ({ ...prev, [remote.id]: savedFromDoc(remote) }));
  // The old command history no longer applies to the adopted document.
  setHistoryMap((prev) => ({ ...prev, [remote.id]: emptyStack<DesignDocument>() }));
  setSelectedId(null);
  setSaveFailed(false);
}, []);

const reloadActive = useCallback(async () => {
  const id = activeIdRef.current;
  if (!id) return;
  try {
    adoptRemote(await getDesign(id));
    setConflict(null);
  } catch {
    // Leave the local document; the next focus/save will try again.
  }
}, [adoptRemote]);

const openExternalConflict = useCallback(async () => {
  const id = activeIdRef.current;
  if (!id) return;
  try {
    const current = await getDesign(id);
    if (current.revision > (savedRef.current[id]?.revision ?? 0)) {
      setConflict({ designId: id, current });
    }
  } catch {
    /* offline: the next broadcast or save will surface it */
  }
}, []);

const activeForSync = docs.find((doc) => doc.id === activeId) ?? docs[0];
const sync = useStudioSync({
  activeDesignId: activeForSync?.id ?? null,
  activeRevision: activeForSync?.revision ?? 0,
  dirty: activeForSync
    ? saved[activeForSync.id]?.signature !== docSignature(activeForSync)
    : false,
  saving,
  onReloadDesign: () => { void reloadActive(); },
  onConflict: () => { void openExternalConflict(); },
  onDesignsChanged: () => {
    // Summary list: add ids we do not hold, drop ids the server deleted,
    // then refetch the active document if its summary moved.
    void refreshSummaries();
  },
  onAssetsChanged: (designId) => {
    void listDesignAssets(designId).then((list) =>
      setAssets((prev) => ({ ...prev, [designId]: list })),
    );
  },
});
```

`onDesignsChanged` is a *summary* refetch. `useStudio` holds full documents, so the coordinator
should merge `listDesigns()` into `docs` (fetch new ids with `getDesign`) rather than replacing the
open document with a summary. A coarse event is not permission to overwrite open edits — that is
what `reconcileDesign` exists for.

### Own-save / own-change broadcasts (required call sites)

The same-tab channel filters its own events, so these calls only affect other tabs.

| Where | Call |
|---|---|
| `persist()` after `saveDesign` returns, right after adopting `persisted.revision` | `sync.notifyDesignsChanged({ designId, revision: persisted.revision })` |
| `createFromTemplate()` after `createDesign` | `sync.notifyDesignsChanged()` |
| `duplicateActive()` after `createDesign` | `sync.notifyDesignsChanged()` |
| `deleteActive()` after `deleteDesign` | `sync.notifyDesignsChanged()` |
| `importAsset()` after `uploadDesignAsset` | `sync.notifyAssetsChanged(designId)` |
| `removeAsset()` after `deleteDesignAsset` | `sync.notifyAssetsChanged(designId)` |
| Template shelf writes (if the coordinator makes the shelf durable) | `sync.notifyTemplateChanged()` |

The `persist()` call must use the revision returned by the save, not the local one: that is the
value other tabs compare against. If it is omitted, the call degrades to a coarse list refresh and
an idle tab will not reload the open document until its own save hits `409`.

`resolveConflict("take-theirs")` does not change server state, so it needs no broadcast.
`resolveConflict("keep-mine")` re-saves, and `persist()` broadcasts from there.

## Validation

```bash
cd scenarios/react-component-library/ui
pnpm exec vitest run src/workbench/studio/sync.test.ts src/workbench/studio/useStudioSync.test.tsx
pnpm exec tsc --noEmit
pnpm exec eslint src/workbench/studio/sync.ts src/workbench/studio/sync.test.ts \
  src/workbench/studio/useStudioSync.ts src/workbench/studio/useStudioSync.test.tsx
```

Results: **44 tests pass** (23 `sync.test.ts`, 21 `useStudioSync.test.tsx`); `tsc --noEmit` clean;
eslint **0 errors / 0 warnings**. Not run, per the brief: `pnpm build`, scenario restart, commit.

Coverage of the required cases:

- **(a)** idle + newer remote → `onReloadDesign`; dirty → `onConflict`; saving → neither.
- **(b)** own-tab events filtered in `sync.ts` for both transports (asserted directly).
- **(c)** burst of design + asset events → one callback each; timer boundary at 249/250 ms;
  highest revision in the window wins.
- **(d)** `focus` and visible `visibilitychange` refetch; hidden does not; saving skips.
- **(e)** unmount cancels a pending burst, a pending focus refetch, and detaches the channel.
- **(f)** malformed payloads (missing source, wrong type, missing designId, primitive, `null`) are
  dropped without throwing in both the channel and the hook.

## Deliberate limits / follow-ups

- **Not integrated.** Until `useStudio` calls the hook and the broadcast sites above, the module is
  inert; the `409` path remains the only cross-tab protection.
- **Targeted broadcast is opt-in.** Callers that omit `{ designId, revision }` get coarse refresh
  only. Documented so the coordinator does not mistake it for full live sync.
- **No presence / no ephemeral cursors.** The channel carries change signals, not collaboration
  state, and never carries document bodies — the server stays the only source of truth.
- **Assets for inactive designs are not pushed.** They are fetched when the design is opened,
  matching `useStudio`'s lazy asset cache.
