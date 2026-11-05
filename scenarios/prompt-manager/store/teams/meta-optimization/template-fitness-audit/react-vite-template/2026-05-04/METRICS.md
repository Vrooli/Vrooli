# Metrics — 4 numerical + 1 meta

Each scenario in `SCENARIOS.md` produces five values. The first four are graded numerically (lower is better, except where noted); the fifth is a yes/no judgment that counterweights the numerical metrics.

Sub-lens definitions live in [`docs/agent-system/REFERENCE_PATTERN_FITNESS.md`](../../../../../../../../docs/agent-system/REFERENCE_PATTERN_FITNESS.md). This file is the operational restatement.

---

## 1. Per-replica cost — `lines_added`

**Sub-lens**: per-replica cost.

**What it measures**: How much *non-test* code an author has to write to do the workflow. Proxy for boilerplate tax.

**Computation**: per-cell command lives in `SCENARIOS.md` §"Measurement command suite" — commit-free `diff -rN` against the snapshot taken in preliminary step 4, scoped to `cli/`, `api/`, `ui/src/`, with `_test.go` / `.test.ts` / `.test.tsx` excluded.

Run formatters before measuring (`gofumpt -w` over Go files; `pnpm prettier --write` over TS/TSX). The recipe locks formatter versions per `SCENARIOS.md`.

**Interpretation**: For "add" workflows (1, 2, 3, 4), lower is better — less boilerplate. For "delete" workflow (5), higher is *worse* (more cleanup tax). For "rename" workflow (6), lower is better but the floor is constrained by the actual rename target count.

**Special case — Scenario 5 (delete)**: record absolute lines *removed*. Recipe in `SCENARIOS.md` §"Measurement command suite" provides the `^<`-grep equivalent.

## 2. Drift surface count — `drift_count`

**Sub-lens**: drift surface map.

**What it measures**: How many places where two pieces of information must agree but only convention enforces it.

**Computation**: Manual walk of the diff. For each piece of information that appears in two or more places, classify the enforcement:

- **type-system** — disagreement is a compile error
- **CI check** — disagreement is a build-time failure
- **hope** — disagreement is invisible until runtime

Count entries classified as "hope". Type-system and CI-check entries are recorded for context but don't count toward this metric.

Record each as:
```
- (location-A, location-B, enforcement: type/CI/hope)
```

**Examples** (worked into the iteration-1 audit):
- `r.HandleFunc("/api/v1/notes", ...)` in `handler.go` vs `endpoints.go::Path: "/api/v1/notes"` → **hope** (counts).
- A proto field name vs the JSON tag a frontend consumes → **type-system** (does not count if generated from the proto).
- `cli_commands_seed.json` name vs `register.go::Command.Name` → **hope** (counts; partially CI-checked today).

## 3. Contract location — `contract_grade`

**Sub-lens**: contract location audit.

**What it measures**: For each non-trivial precondition or invariant the workflow introduces, where does the contract live.

**Computation**: Manual classification per precondition. Record as:

```
- (precondition: "<text>", location: type-signature | CI-check | code-comment | nowhere)
```

The metric is a tuple `(type-signature count, CI-check count, code-comment count, nowhere count)`. Lower numbers in the right two columns are better.

**Examples**:
- "callers must pass an empty `ID`" — if encoded as a separate `CreateInput` DTO that lacks the field, this is **type-signature** (best). If it lives in a `// Note: leave ID zero` comment, **code-comment**. If it's documented only in `REPLACING-NOTES.md`, **docs only / nowhere from a copy-paste-modify perspective**.

## 4. Central-registry edits — `central_edits`

**Sub-lens**: coordinated-edit count.

**What it measures**: How many files outside the workflow's primary domain folder must change. The most architecturally meaningful number — captures whether the architecture is paying compounding cost per replica.

**Computation**: per-cell command lives in `SCENARIOS.md` §"Measurement command suite" — commit-free `diff -rqN` against the snapshot, filtering domain folders. Cross-reference against the **central files list** in `SCENARIOS.md`. Files that show in this count but aren't on that list are flagged as a *definition gap*.

**Interpretation**: For all workflows, lower is better. Stopping rule §1 keys on this metric.

## 5. Meta-metric — `junior_doable`

**Sub-lens**: implicit; counterweights numerical gaming.

**What it measures**: Could a junior engineer, given only `docs/internal/REPLACING-NOTES.md` and the template tree (no Slack, no senior tap), execute this workflow?

**Computation**: Yes/No + one-sentence reason. The reason is the load-bearing part — "yes because the docs walk through every step" vs "yes because it's mechanical given the existing pattern" vs "no because the new error-code path requires knowledge of envelope conventions not in REPLACING-NOTES.md".

**Why this exists**: The four numerical metrics can be optimized by collapsing into clever DSLs that are unreadable to a new author. This metric prevents that. If `lines_added` drops 90% but `junior_doable` flips from yes to no, the iteration regressed even though the numbers improved.

---

## Result format

Each scenario × iteration produces a row:

| Scenario | Iter | lines_added | drift_count | contract (TS, CI, comment, nowhere) | central_edits | junior_doable |
|----------|------|-------------|-------------|------------------------------------|---------------|---------------|

The cell for `lines_added` and `central_edits` includes the exact command + branch SHA used to produce the number, in a footnote or inline comment, so future iterations can replay the recipe deterministically.
