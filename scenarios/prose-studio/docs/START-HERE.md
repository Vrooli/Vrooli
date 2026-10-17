# Start Here — Prose Studio

Prose Studio turns a query and a named writing profile into a governed,
measured candidate set. It owns generation mechanism and convergence; the
consumer owns editorial intent, claims, and the declaration files that define
its voice.

## Read first

1. [`../PRD.md`](../PRD.md) — product intent, non-goals, and P0/P1/P2 targets.
2. [`../DESIGN.md`](../DESIGN.md) — operator-surface accessibility and visual
   contracts.
3. [`concepts/DOMAINS.md`](concepts/DOMAINS.md) — ownership boundaries.
4. [`concepts/FLOWS.md`](concepts/FLOWS.md) — generation, convergence, and
   document assembly sequences.

## Initialization Protocol

Run the scenario setup lifecycle, confirm the declaration scan, and then
validate the scenario requirements before making product changes:

```bash
make setup
vrooli scenario requirements validate prose-studio --json
```

## First vertical slice

Create a style, create a profile that references it, inspect the resolved
instruction, generate through `write.default` or `write.diverse`, and verify
that every candidate contains measurements, machine-generation disclosure, and
full model/token provenance. Then use the same session to pin, reject, reroll,
and commit. There is no alternate agent path: `generate` is a wrapper over the
session path.

## Architecture Rules

Keep model inference behind the ai-gateway seam, keep sampler and selection
kinds governed in the API, and treat consumer declaration files as immutable
authority. UI and CLI surfaces call the same Connect contract; they do not
reimplement policy or persistence.

## Consumer declarations

Place version-controlled files under the consuming scenario's
`.vrooli/prose-studio/` directory. Each file must contain `schema_version`,
`kind`, a namespaced `key`, `created_by`, and a `record`. Run
`prose-studio prose declarations-validate --json '{"root":"/path/to/repo"}'` or call the
RPC directly. Files are authoritative; API writes to a declared key are
refused with `profile_is_declared`. Deleting a file produces an `unregistered`
projection so historical provenance remains inspectable.

## Validation

Use the scenario-owned lifecycle and test server:

```bash
vrooli scenario requirements validate prose-studio --json
vrooli scenario test prose-studio
```

The shared package can be tested independently from
`packages/textmetrics`. Do not report semantic similarity or perplexity: the
current gateway exposes neither an embedding nor log-probability surface.
