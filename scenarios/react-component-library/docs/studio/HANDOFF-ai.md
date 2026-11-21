# Handoff — AI design assistance

Status: implemented and verified live against `ai-gateway` (`ollama` / `qwen3.5:9b`). The model call
is real; nothing is applied without an explicit click.

## Why this shape

A design assistant is useful only if its output can be **judged before it lands**. The studio
therefore has no "chat that edits as it talks". It has:

```
prompt + scope + revision
      │
      ▼   POST /api/designs/{id}/assist
designassist (bounded context + patch schema + instruction)
      │
      ▼   designinference (durable idempotency, privacy class, local-only profile)
ai-gateway ──► ollama
      │
      ▼   strict JSON decode + JSON Schema + semantic validation
proposal (inert)
      │
      ▼   Apply as one step
applyPatch (client re-validates the live tree) ──► one command-stack entry
```

The model never writes storage. A proposal can be discarded, and an applied proposal is undone
with Cmd/Ctrl+Z like any other command.

## Server contract

`POST /api/designs/{id}/assist`

```json
{
  "prompt": "Rewrite the headline to announce next week's launch.",
  "scope": { "kind": "layer", "frameId": "aperture-desktop", "nodeId": "aperture.desktop.title" },
  "revision": 6,
  "idempotencyKey": "8ce4…"
}
```

- `scope.kind` is `design`, `frame` or `layer`; `frameId` is required for frame/layer, `nodeId` for
  layer. `design` takes neither.
- `revision` is **required**: a proposal against a stale document is refused with `409` and the
  current `design`, exactly like a save.
- `idempotencyKey`: 8–200 characters of `[A-Za-z0-9._:-]`. A retry with the same key returns the
  durable operation instead of dispatching again. The studio generates a new key per attempt.

Response:

```json
{
  "operation": {
    "id": "inference_…", "state": "completed", "provider": "ollama", "model": "qwen3.5:9b",
    "inputTokens": 1172, "outputTokens": 69
  },
  "revision": 6,
  "proposal": {
    "summary": "The headline announces next week's launch.",
    "operations": [{ "kind": "setText", "nodeId": "aperture.desktop.title", "text": "Launching next week" }]
  }
}
```

Errors are the standard envelope:

| Status | Code | Meaning |
|---|---|---|
| 400 | `invalid_argument` | Malformed prompt/scope/revision/key, or context too large |
| 404 | `not_found` | No design with that id |
| 409 | `conflict` | `revision` did not match; body also carries `design` |
| 422 | `invalid_proposal` | The model returned a patch this document cannot apply |
| 502 | `ai_failed` | The gateway refused or failed the request |
| 503 | `ai_unavailable` | No runner configured, or the gateway could not be reached |

`GET`/`PUT` on the design are unaffected; assistance is the only route that dispatches inference.

## Patch vocabulary

Eight operations maximum, one flat object per operation (`kind` plus optional fields):

| Kind | Fields | Notes |
|---|---|---|
| `setText` | `nodeId`, `text` | Only `text`, `button`, `chip` layers |
| `setProp` | `nodeId`, `key`, `value` | Existing prop; `x/y/w/h` only on a slot |
| `setTheme` | `key`, `value` | Palette tokens; hex or a colour name |
| `addLayer` | `parentId`, `node` | Addable types only; one authored level of children |
| `removeLayer` | `nodeId` | Never the page root or a slot |
| `moveLayer` | `nodeId`, `parentId`, `index?` | Never into itself or a descendant |

`addLayer` never carries ids (the client assigns them), `src`/`href` are blocked everywhere, and a
layer-scoped request may only target ids present in the context it was sent (the selection, its
parent, siblings and ancestors).

The JSON schema is intentionally inside the gateway's **enforceable subset**:

- no `type` arrays (`["string","number","boolean"]` is rejected);
- no `maxProperties`, `anyOf`, `oneOf`, `$ref`/`$defs`, or object-valued `additionalProperties`;
- `additionalProperties` must be a boolean.

That is why `value` is unconstrained in the schema and checked semantically. If the gateway's subset
grows, tighten the schema; do not add constructs it rejects.

## Context and validation

- The server reads the design from storage and builds the context itself; the client never uploads
  a second copy. Context is capped at 400 KiB, with a clear 400 asking for a narrower scope.
- `designinference` uses the existing `extract.structured` role (4096 output tokens, confidential
  privacy class, local-only profile). It is the only role the durable contract accepts today.
- `designassist.Validate` simulates the operations in order on a private copy: a remove followed by
  an edit of the removed layer is rejected before the proposal is returned.
- The client (`studio/patch.ts`) re-validates and applies against the **live** tree, assigns new ids
  (`<parent>.<type>-<stamp>-<n>`), and returns `{doc, changed}`. A stale proposal is refused with an
  "ask again" state. Staleness is the **document signature**, not the revision: an autosave of an
  unchanged document bumps the revision without changing a layer, and that must not invalidate a
  proposal made seconds earlier.
- Applying goes through `useStudio.applyProposal`, which pushes **one** command labelled
  `AI · <summary>`; Undo reverses the whole patch.

## UI

`studio/StudioAssist.tsx` inside the inspector's AI tab:

- scope chips (layer / frame / design), subject, prompt textarea, `⌘/Ctrl+Enter`;
- the request aborts on unmount; a 3-minute deadline covers a local model;
- proposal card with the summary, provider/model, a humanised operation list (layer names, not ids),
  Apply / Discard;
- honest failure copy per code, including "start it with `vrooli scenario start ai-gateway`";
- a stale banner instead of an Apply button when the document moved.

## Running it

```bash
vrooli scenario start ai-gateway         # local-only profile resolves ollama
curl -s localhost:19894/health           # readiness
# then in the studio: select a layer → AI → prompt → Propose
```

Model choice belongs to the gateway's role catalog; the scenario only requests a role, profile and
privacy class.

## Tests

- `api/internal/designassist`: context scoping, prompt/scope limits, schema compilation, every
  unsafe op (page/slot removal, cycle, `src`, placement on a non-slot, deep add, bad colour,
  over-count), remove-then-edit simulation.
- `api/handlers/designs`: happy path (inert, revision unchanged), stale revision `409`, invalid
  proposal `422`, unreachable/`dispatch_unknown` `503`, failed `502`, envelope validation, and the
  no-runner `503`.
- `ui/src/workbench/studio/patch.test.ts`: apply/undo shape, deterministic ids, immutable input,
  live-tree rejection.
- `ui/src/workbench/designs.test.ts`: proposal error mapping and the local-write distinction.
- Live: `/tmp/opencode/ai-live.py` applied a real proposal in 2.6 s and the toast named the
  summary; screenshot in `evidence/studio-ai-proposal.png`.

## Deliberate limits

- No streaming, multi-turn conversation, model picker or image input. One prompt → one bounded
  patch.
- A proposal is not a check: it never claims acceptance, evidence or RCL coverage.
- The gateway is optional. Without it the panel says so and the studio is fully usable.
