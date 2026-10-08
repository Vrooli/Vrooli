# Experience Design

## Purpose Of This Document

Record the UI decision for Hello Python: what people will compare
it to, which surface matters most, how that surface lays out at each width,
and what the design is accountable for. It is the prose companion to the
machine-readable contract in `experience/`; where the two disagree,
`experience/` wins because a validator reads it.

The "Design decision" orientation gate (step id `design-language`) reads this file. It passes when the
placeholders below are gone and `ui/src/pages/DashboardPage.tsx` no longer
carries the `PLACEHOLDER:home-surface` marker. See
`path:../guides/choosing-ui.md` for how to answer each section.

## The Comparison

[EXPERIENCE-TODO: Name the product people will compare this one to, and say
what that sets the bar for: density, motion, vocabulary, tone.]

## The Primary Surface

[EXPERIENCE-TODO: Name the one surface people open first and stay on. Describe
it at phone width and at desktop width. List its states: loading, empty,
partial, error, and any product-specific state.]

## Shell Configuration

| Setting | Value | Why |
|---|---|---|
| Kit | `vrooli-default` | [EXPERIENCE-TODO: keep or name the kit, and say why it fits] |
| `density` | `sidebar` | [EXPERIENCE-TODO] |
| `mobileNav` | `tabs` | [EXPERIENCE-TODO] |
| `mainMode` | `scroll` | [EXPERIENCE-TODO] |

## What The Design Is Accountable For

Three things, in order, that the UI must make true.

1. [EXPERIENCE-TODO]
2. [EXPERIENCE-TODO]
3. [EXPERIENCE-TODO]

## Information Architecture

| Surface | Route | The question it answers |
|---|---|---|
| Home | `/` | [EXPERIENCE-TODO] |
| Settings | `/settings` | What changes behaviour for everything? |

## Cross-References

- `path:../START-HERE.md` — Gate 5
- `path:../guides/choosing-ui.md` — the reasoning behind each section
- `path:../../experience/README.md` — the typed contract
- `path:../../DESIGN.md` — the token contract
