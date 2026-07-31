# Descriptors — Channel Manager

Platform behaviour and warming practice live here as validated JSON rather than in
Go. Adding a platform or a warming variant is a descriptor plus a validation run,
not a build. This is the same idiom the fleet uses for search providers
(`.vrooli/search.json`) and for `vrooli-memory`'s harness adapters.

```
data/
  platforms/<platform>.json                 # what a platform allows
  warming-programs/<id>.json                # how an identity is warmed
```

**The files are the source of truth.** Channel Manager loads only the immediate
JSON files in `platforms/` and `warming-programs/`; nested files are unsupported.
The SQLite state record deliberately stores accumulated operator state only, never
a descriptor cache. A malformed descriptor fails startup loudly — a silently
skipped platform descriptor would remove a cadence ceiling.

## Every descriptor carries provenance

```json
"provenance": {
  "source_kind": "measured | vendor-doc | operator-folklore | hand-written",
  "confidence": "measured | corroborated | speculative",
  "captured_at": "YYYY-MM-DD",
  "sources": ["..."],
  "note": "...",
  "revisit_trigger": "..."
}
```

This is required, not optional metadata (`CHANMGR-P0-018`). A future agent editing a
threshold must be able to see whether it is overriding measured data or a guess.

**Everything shipped today is `speculative`.** The warming programs are distilled
from operator practice shared publicly, not from platform documentation, and no
number in them has been measured on any account this fleet operates. The platform
cadence ceilings are conservative guesses chosen to sit safely under any reported
limit. Treat all of it as a starting hypothesis.

## Two note surfaces, doing different jobs

- **`provenance`** — why the descriptor was written this way. Authored with it,
  revised when the source changes.
- **`observations[]`** — what actually happened. Append-only, written by real runs,
  never hand-edited (`CHANMGR-P1-006`). This is the path from folklore to
  measurement: after enough runs, the observations say more than the source note
  ever did, and that is the moment to raise `confidence`.

Step-, phase-, and gate-level `note` fields carry the reasoning for a specific
number. Use them freely — they are the difference between a range someone can
revise and a constant nobody dares touch.

## Adding or editing a descriptor

```bash
channel-manager warming programs validate            # schema + cross-reference checks
channel-manager platforms validate
```

Validation checks more than shape. A warming program naming an action kind its
platform does not declare fails, and so does a program whose counts exceed the
platform's ceilings — the ceiling always wins, so a badly written program fails at
plan generation rather than at the platform.

## What does not belong here

- **Credentials.** They live in `vault`; an identity holds a path.
- **Handles and account state.** Those are rows in `identities`, not descriptors.
- **Comment text.** Warming requires substantive comments, and generating them at
  scale produces exactly the low-effort pattern the warming exists to avoid. The
  descriptor declares the constraint; a human or an agent authors each comment
  (D-009).
- **Channel strategy.** Which channels are active and what each account is *for* is
  operator-curated canon in `docs/marketing/strategy/CHANNELS.md`. Read-only input.
