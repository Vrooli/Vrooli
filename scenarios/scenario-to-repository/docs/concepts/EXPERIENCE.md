# Experience Design

## Purpose Of This Document

Record the UI decision for Source repository: what people will compare
it to, which surface matters most, how that surface lays out at each width,
and what the design is accountable for. It is the prose companion to the
machine-readable contract in `experience/`; where the two disagree,
`experience/` wins because a validator reads it.

The "Design decision" orientation gate (step id `design-language`) reads this file. See
`path:../guides/choosing-ui.md` for how to answer each section.

## The Comparison

The primary comparison is Git Control Tower's diff-first operator workspace.
That sets the bar for dense source vocabulary, explicit evidence identity,
quiet motion, and reviewable rather than decorative status.

## The Primary Surface

The source readiness workspace is the first surface. On phones it becomes a
stepwise source/details/review flow; on desktop it uses a source tree beside a
gate inspector. It exposes loading, empty, partial, stale, request-error,
permission-denied, unavailable, verified, and awaiting-human-publication states.

## Shell Configuration

| Setting | Value | Why |
|---|---|---|
| Kit | `vrooli-default` | The operational kit provides dense evidence surfaces, semantic status, and responsive shell primitives. |
| `density` | `sidebar` | Source tree and evidence inspector benefit from persistent context. |
| `mobileNav` | `tabs` | Mobile preserves the five review steps without hidden hover actions. |
| `mainMode` | `scroll` | Long manifests, notices, and verification output need readable scrolling. |

## What The Design Is Accountable For

Three things, in order, that the UI must make true.

1. Make included, excluded, and unresolved source obligations legible.
2. Bind every green status to the exact artifact and evidence identity.
3. Keep human publication visibly separate from automated preparation.

## Information Architecture

| Surface | Route | The question it answers |
|---|---|---|
| Home | `/` | What source export can I safely prepare or review next? |
| Settings | `/settings` | What changes behaviour for everything? |

## Cross-References

- `path:../START-HERE.md` — Gate 5
- `path:../guides/choosing-ui.md` — the reasoning behind each section
- `path:../../experience/README.md` — the typed contract
- `path:../../DESIGN.md` — the token contract
