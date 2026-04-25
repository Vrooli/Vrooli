# Decision Modifications — Contract

## What this is

`modifications` is a structured, scoped exception an operator attaches to a
decision when they **accept** one of the presented options but disagree with, or
want to add to, part of that option's rationale.

It is distinct from `notes`:

| Field | Purpose | Shape |
|---|---|---|
| `notes` | Free-form decision-wide commentary. | `string` |
| `modifications` | Structured, machine-readable scoping of the selected option's rationale. | object (see below) |

Both may coexist on a single decision.

## Shape

```jsonc
{
  "excluded_clauses": ["string", ...],  // parts of the option's rationale the operator does NOT accept
  "additions":        ["string", ...],  // extra scope / constraints the operator layers on top
  "rationale":        "string"          // explanation of the modification
}
```

All sub-fields are optional **individually**, but at least one must be present
(arrays non-empty, or `rationale` non-empty). An entirely empty object is
rejected — operators should omit the field instead.

- `rationale` is bounded at 4096 UTF-8 bytes.
- Array entries must be non-empty strings (whitespace-only is rejected).

## Semantic rules

1. **Where applicable.** `modifications` ships only on the **accept** path.
   Reject and defer paths do not carry it.
2. **Operator intent is authoritative.** The server does **not** validate that
   `excluded_clauses` strings literally appear in the selected option's
   rationale text. Substring matching would be brittle and can penalize operator
   paraphrase. Consumers read `excluded_clauses` as operator-scoped intent, not
   as a text-diff.
3. **Accept-once immutability.** Once a decision is accepted with
   `modifications`, the field cannot be changed. A subsequent attempt to set
   `modifications` on the same decision is rejected. If amendment ever becomes a
   need, it will be tracked as a distinct backlog item rather than a silent
   mutation.
4. **Absent vs. empty.** A decision record without the field means the operator
   did not scope the acceptance. Consumers must treat absent and any valid
   non-null value as first-class states.
5. **Relationship to `notes`.** Do not parse prose out of `notes` to
   reconstruct `modifications`. If an operator put modifications in `notes`,
   that is a legacy / mis-filed record — treat it as unstructured commentary
   and move on.

## Consumer guidance

- **Workshop agents / plan synthesizers.** When resolving a decision, feed
  `modifications` to plan synthesis as a first-class structured input, distinct
  from `notes`. The plan should reflect `excluded_clauses` as scope removed and
  `additions` as scope added.
- **Auto-create flows** (e.g., initiative auto-create from decision acceptance).
  Propagate `modifications` into generated artifacts' metadata so downstream
  items can read the operator's scope decisions without re-inferring them.
- **Display.** Render `modifications` as a distinct block from `notes`. Do not
  collapse them together.

## Surfaces

- **API:** `PATCH /teams/{id}/decisions/{decisionId}` accepts `modifications`
  in the request body. The stored `DecisionEntry` exposes it on reads.
- **CLI:** `prompt-manager team decision-accept` supports
  `--modifications='{...}'` and `--modifications-file=<path>` (mutually
  exclusive). `decision-show` and `decision-list` render a distinct
  `Modifications:` block.
- **UI:** The decision-accept form exposes `excluded_clauses`, `additions`, and
  `rationale` as structured inputs once an option is selected, collapsed behind
  an "Add modifications" affordance. Persisted modifications render as a
  read-only block.

## Out of scope

- Editing or amending `modifications` after first accept.
- `modifications` on reject / defer paths.
- Server-side textual matching of `excluded_clauses` against option rationale.
- A `notes` → `modifications` migration or auto-parser.

## Why this contract is separate from proto/struct comments

The contract captures the **meaning** — what consumers should do with the data
— which is load-bearing across API, CLI, UI, and downstream synthesis code.
Struct comments link here to ensure schema readers land on this contract before
writing consumer logic.
