# Component Library Gaps

## Purpose Of This Document

Record what `react-component-library` (RCL) does and does not yet provide for
the experience defined in `path:../concepts/EXPERIENCE.md`, so the UI is built
by adopting the library rather than by forking around it. Measured on branch
`agi`, 2026-09-01, against 167 catalogued components.

This is a report about the shared library, not a backlog this scenario owns.
Items here belong to RCL and should be raised there.

## Summary

| Finding | Scale | Blocks |
|---|---|---|
| No conversation UI family exists at all | 8 components needed | The primary surface |
| Components exist but carry anatomy-only stories | 86 of 167 (51%) | Adoption and component specs |
| `ExperienceSurface` pinned at a version with no contract | 24 pins, 5 scenarios | `spec validate` for every generated scenario |

## What Is Already Good And Should Be Adopted

Worth stating first, because most of this UX is buildable from the library today.

| Component | Version | Depth | Used for |
|---|---|---|---|
| `ApprovalPrompt` | 1.0.12 | 527 lines, 7 stories including `permission-denied` and `retry` | The capability gate, almost exactly |
| `Avatar` | 1.3.14 | 414 lines, 9 stories | The agent avatar rendered from the descriptor triple |
| `VoiceInputButton` | 4.3.4 | 310 lines, 7 states | Composer voice input and call mode |
| `AsyncPanel` | 1.0.4 | 5 stories matching the region lifecycle exactly | Every declared async region |
| `EmptyState` | 1.7.14 | 3 stories | The shared empty treatment |

`ApprovalPrompt` in particular already models submit, success, error, retry and
permission-denied. The capability gate is a wrapper over it, not a new component.

## Gap 1 — The Conversation Family Does Not Exist

A catalogue search for `conversation`, `chat`, `thread`, `bubble`, `composer`,
`typing`, `transcript`, `reaction`, `receipt` and `inbox` returns **nothing**.

`Message` exists, and is not this. Its catalog id is `ai.message` and its
contract is an **AI-console transcript card**: a bordered, elevated panel with
32px padding and a 72ch measure, taking `actor`, `content`, `certainty`,
`urgency`, `citations`, `activity`, `actions` and `onRetry`. That is the right
component for an agent console. It has no sidedness, no author grouping, no day
divider, no delivery state and no reactions, so it is the wrong component for a
message thread. It also carries one anatomy-only story with zero declared
argument fields, so there is nothing to adopt against even where it fits.

### Components RCL would need

| # | Component | Why it is library-shaped, not scenario-shaped |
|---|---|---|
| 1 | `Transcript` | A virtualised message log with `role="log"`, consecutive-author grouping, day dividers and jump-to-latest. `VirtualList` exists to build on. Any scenario with a conversation needs this. |
| 2 | `MessageBubble` | Author, timestamp, delivery state, attachment slot, and an agent-authored marker. Needs a **non-sided variant**: two-sided layout stops working the moment a room has three authors. |
| 3 | `MessageComposer` | Attach, send, busy, disabled-with-reason, draft preservation across failure, safe-area anchoring, voice-input slot. The disabled-with-reason behaviour is the part everyone re-implements badly. |
| 4 | `ThreadListItem` | Leading accent edge, preview truncation that never truncates identity, unread state that is not colour alone. |
| 5 | `ConversationPage` | A page template beside `CollectionPage`, `DetailPage` and `DashboardPage`: list plus detail, chrome pinned, content-only scroll, mobile stack with a real back affordance. `MasterDetail` is close but is a component, not a page template, and carries one story. |
| 6 | `CallSurface` | Full-viewport call layout with controls reachable in every state including error. Distinct enough from a dialog that composing one from `Dialog` produces the wrong dismissal semantics. |
| 7 | `ListeningIndicator` | Continuous liveness with a reduced-motion static form. Small, but it is the only ambient animation and getting its reduced-motion behaviour wrong is an accessibility defect. |
| 8 | `RankBadge` | An **ordinal** badge. `StatusBadge` is categorical — neutral, success, warning, danger — which cannot express that `known` is below `trusted`. Reusable well beyond this scenario: severity ladders, maturity rungs, tier systems. |

Items 1–4 are the minimum for a credible thread. Items 5–7 are what make call
mode and mobile work. Item 8 is the smallest and the most reusable.

## Gap 2 — Half The Catalogue Is Anatomy Only

**86 of 167 components (51%) carry one story or fewer.** A single anatomy story
asserts that something renders; it declares no states. That matters directly
here, because `experience-component` specs anchor each declared state to a named
story, so a one-story component can only ever have one specified state.

Components this UX depends on that are in that group:

| Component | Stories | Consequence |
|---|---|---|
| `AttachmentPreview` | 1 | Cannot specify the over-limit rejection state |
| `BudgetBar` | 1 | Cannot specify pressure or exhaustion |
| `MasterDetail` | 1 | Cannot specify empty, loading or mobile-stacked |
| `DataTable` | 1 | Cannot specify empty, loading or error |
| `CollectionPage` | 1 | Same |
| `Timeline` | 2 | Cannot specify a failed or partial entry |

Separately, **only 41 of 167 components (25%) have their own experience contract**
inside RCL. A component with no contract cannot be pinned as a region component
without the error described next.

## Gap 3 — A Version Pin That Breaks Every Generated Scenario

`experience-manager spec validate` reports, as a hard **ERROR**:

```
region "notes" pins library component experience-surface@1.0.0
without a canonical experience contract
```

The cause is a version mismatch, not a missing contract. `ExperienceSurface` is
at `1.0.3`, and RCL's contract at `experience/components/experience-surface.json`
references `versions/1.0.3/story.json`. The `react-vite` template pins `1.0.0`.

Scale: **24 pins at `1.0.0` across 5 scenarios** — `browser-automation-studio`,
`hello-mobile`, `infrastructure-manager`, `landing-page-business-suite` and
`switchboard`. Three pins sit at `1.0.2`, one at `1.0.3`.

Consequence: every scenario generated from the template ships an experience spec
that fails validation out of the box, for following the pattern its own
`experience/README.md` tells it to follow. The fix is a one-line template change
plus a re-pin of existing scenarios. This is the cheapest item in this document
and the only one that is currently red.

## What This Scenario Will Do Meanwhile

Nothing is forked. The experience contract in `experience/components/` already
grounds every local component on a real catalog story, including the four that
wrap components the conversation family will eventually replace:

| Local spec | Grounded on | Replace with, when it lands |
|---|---|---|
| `transcript` | `Message@1.0.11` | `Transcript` + `MessageBubble` |
| `thread-list` | `List@1.0.6` | `ThreadListItem` |
| `trust-tier-badge` | `StatusBadge@1.2.2` | `RankBadge` |
| `async-region` | `AsyncPanel@1.0.4` | keep — this one is already right |

Building against local specs that name their intended replacement keeps the
adoption path visible instead of letting a fork harden.

## Cross-References

- `path:../concepts/EXPERIENCE.md` — the experience this measures against
- `path:../../experience/README.md` — the depth ladder and validation command
- `path:../../../react-component-library/PRD.md` — the library's own contract
