# Capture outline patch v1

The `rcl.capture-outline-patch/v1` contract joins an immutable capture, one exact outline region, a same-revision RCL composition change, and a component-backed preview. The canonical machine-readable shape is [capture-outline-patch-v1.schema.json](../contracts/capture-outline-patch-v1.schema.json); the RCL Studio consumer and validator are `ui/src/workbench/captureOutlinePatch.ts`.

The patch is a transfer artifact. It is not a new capture engine and it does not claim that an implementation has been built or served. The source capture carries its scenario, build, route, state, data, viewport, screenshot and outline digests. The target carries the exact region and outline node, the RCL display/frame that owns the mapped nodes, the mapped RCL node set, normalized crop bounds, and an optional component identity for custom or no-RCL regions. The preview is accepted only as `composited` or `rendered`, and must preserve the source route, state, data and viewport. `sketch` cannot satisfy this contract.

Composition operations reuse `DesignPatchOperation`. Every edit must stay within the mapped region nodes in the named frame, must fit the 1–8 operation limit, and must name the exact design and base revision. `applyCaptureOutlinePatch` rejects a missing frame, a node mapped to another frame, or a stale design revision before it delegates to the existing local patch engine. The engine retains its own node, type, and structure checks.

Validate and apply in the RCL Studio before exporting the object:

```ts
const applied = applyCaptureOutlinePatch(currentDesign, candidatePatch);
```

A packet importer should validate the same JSON Schema and pass the patch through this consumer before offering the changes to an implementer. Preserve `patch_id`, source capture identity, target region identity, preview fidelity, operation list, and base revision in the handoff. A missing outline ID, mismatched preview identity, stale base revision, or out-of-region operation is a refusal, not a best-effort import.

Focused validation:

```bash
cd scenarios/react-component-library/ui
pnpm exec vitest run src/workbench/captureOutlinePatch.test.ts
```

These tests prove the source contract and local patch gate only. EX11 remains open until three real RCL/custom/no-RCL chains complete through a phone packet answer and a fresh implementation, and the outline method is compared with frozen-page editing on those same cases.
