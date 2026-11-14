# Experience Design

## Purpose Of This Document

Record the UI decision for Chain Forensics: what people will compare
it to, which surface matters most, how that surface lays out at each width,
and what the design is accountable for. It is the prose companion to the
machine-readable contract in `experience/`; where the two disagree,
`experience/` wins because a validator reads it.

The "Design decision" orientation gate (step id `design-language`) reads this file. It passes when the
placeholders below are gone and `ui/src/pages/DashboardPage.tsx` no longer
carries the `PLACEHOLDER:home-surface` marker. See
`path:scenarios/chain-forensics/docs/guides/choosing-ui.md` for how to answer each section.

## The Comparison

Operators will compare this to Chainalysis Reactor and Blockchair's transaction
views: an investigator must follow a trace and inspect its evidence without
losing the path. Chain Forensics is local-first and source-forward, so the bar
is a dense case workspace, persistent provenance, explicit uncertainty, and a
clear boundary between public observations and analyst interpretations. It
should feel like a professional case desk, not a dashboard or attribution
oracle.

## The Primary Surface

The operator opens the case workspace at `/`. Desktop uses a persistent case
rail on the left, a flow/timeline canvas in the center, and an evidence
inspector on the right. Selecting an edge keeps its provenance, rule version,
grade, and caveats visible beside the path. On phones the case rail becomes a
full-screen drawer and the workspace becomes three route-level tabs: Flow,
Evidence, and Sources. The active case and selected event remain visible when
moving between tabs. Primary actions sit at the bottom of the mobile case
panel; export is an explicit local-file action.

The page declares loading (cache or public source read in progress), empty (no
cases yet, with an offline published example to open), partial (a provider is
rate-limited or history is outside known bounds), stale (cached evidence has a
last-captured timestamp), and request-error (retry or open the cached case).
An export state confirms a private local packet was written and names the path;
there is no upload or tip submission action.

## Shell Configuration

| Setting | Value | Why |
|---|---|---|
| Kit | `vrooli-default` | Its operational shell and state components suit repeated source review and dense evidence. |
| `density` | `sidebar` | Keep cases and evidence within one desktop workspace. |
| `mobileNav` | `tabs` | Convert the three desktop panels into full-width phone views. |
| `mainMode` | `scroll` | Long transaction timelines and cited source excerpts need predictable document scrolling. |

## What The Design Is Accountable For

Three things, in order, that the UI must make true.

1. A reviewer can tell which statements are observed, derived by a named rule, or quoted from a filing, and open each source.
2. Every cross-chain or cluster claim displays its grade, unresolved links, and counter-explanations beside the path.
3. Case material and analyst notes stay local; the UI never sends a packet or identifies a natural person from an address.

## Information Architecture

| Surface | Route | The question it answers |
|---|---|---|
| Case workspace | `/` | What facts and evidence support this flow, and what is still unknown? |
| Case review | `/cases/:id` | What is the source-linked timeline for this case? (Route added with the first UI slice.) |
| Settings | `/settings` | What changes behaviour for everything? |

## Cross-References

- `path:scenarios/chain-forensics/docs/START-HERE.md` — Gate 5
- `path:scenarios/chain-forensics/docs/guides/choosing-ui.md` — the reasoning behind each section
- `path:scenarios/chain-forensics/experience/README.md` — the typed contract
- `path:scenarios/chain-forensics/DESIGN.md` — the token contract
