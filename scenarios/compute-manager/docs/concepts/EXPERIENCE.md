# Experience Design

## Purpose Of This Document

Record the UI decision for Compute Manager: what people will compare
it to, which surface matters most, how that surface lays out at each width,
and what the design is accountable for. It is the prose companion to the
machine-readable contract in `experience/`; where the two disagree,
`experience/` wins because a validator reads it.

> **Status: designed, not built.** The UI is still the generated template. Every
> page in `experience/` is status `draft` and every claim is tier
> `aspirational`, because machine-tier claims gate CI and there are no stable
> selectors to check against. The bindings declared there are the selectors the
> UI will be built to match, not selectors that exist.

## The Comparison

A cloud provider's own instance list, and specifically the moment in one where
you realise something has been running since March.

That sets the bar in a particular direction. Provider consoles are competent at
showing what exists and poor at showing what it has cost you so far, because
cost lives in a separate billing section that nobody opens until the invoice
arrives. The comparison this design wants to win on is not density or polish. It
is that **the liability is on the same screen as the inventory.**

Vocabulary follows from that. Say the amount and the unit. Say when it expires.
Do not say "usage is high". Tone is plain and unalarming, because a page that
shouts about ordinary spending trains people to ignore it on the day it matters.

## The Primary Surface

**Inventory**, at `/`. It is the page an operator opens and stays on, and the
one a scenario author lands on when they want capacity.

Three facts carry it: what exists right now, what it has cost so far, and how
long each instance has left. Cost and expiry lead because both are irreversible
once missed. Provider identity is metadata rather than an organising principle,
so the operator reads capacity instead of reading vendors.

**At desktop width** the total current liability and the unaccounted count sit
above a single table of instances, one row each, with elapsed cost and remaining
lifetime as adjacent numeric columns in tabular numerals. There is one dominant
action, which is requesting capacity.

**At phone width** the table becomes a list of cards, and the two summary
figures stay pinned above them. The columns that survive the collapse are cost
and remaining lifetime; provider, region and size move into the card body. What
must not happen is the summary scrolling away, because a number you have to
scroll to find is a number nobody checks.

**States**, all declared in `experience/pages/dashboard.json`: `loading`,
`empty` (which explains what requesting capacity will cost rather than showing a
bare zero), `default`, `partial` (instances render but the reconciler has not
reported, so the unaccounted count is unknown rather than zero), `stale`
(figures labelled with their observation time), and `request-error`.

The distinction between `partial` and a clean fleet is the one that matters
most. **Unknown rendered as zero is the failure this surface exists to prevent.**

## Shell Configuration

| Setting | Value | Why |
|---|---|---|
| Kit | `vrooli-default` | Operator infrastructure, not a marketed surface. A shared kit means the operator carries one set of habits across the fleet, and nothing here benefits from a distinct visual identity. |
| `density` | `sidebar` | Five surfaces that are navigated between rather than drilled through, and the operator returns to inventory constantly. A persistent sidebar makes the unaccounted badge reachable from anywhere. |
| `mobileNav` | `tabs` | The phone case is checking, not administering. Four tabs cover it, and tabs keep the return path to inventory one tap away. |
| `mainMode` | `scroll` | Inventory is a list that grows. Nothing here is a fixed-height workspace. |

## What The Design Is Accountable For

Three things, in order, that the UI must make true.

1. **Cost and expiry are readable without navigating.** Total current liability
   and the count of instances expiring within the hour are visible in the first
   viewport of the primary surface, at every width. Both are money already being
   spent, and neither should require a click to discover.

2. **The states that matter are distinguishable without colour.** Expiring soon
   and unaccounted are the two states an operator must not miss, and both are
   conventionally red. Red on red is exactly how they get missed, so each
   carries shape or label as well as hue.

3. **Destruction is deliberate and pausing is impossible.** Destroying an
   instance requires typing its name, because it is irreversible and the machine
   may be carrying work. No pause, stop or suspend control exists anywhere,
   because a stopped instance still bills at the full rate on most providers and
   offering the control would imply a saving that does not exist.

## Information Architecture

| Surface | Route | The question it answers |
|---|---|---|
| Inventory | `/` | What do I have, what is it costing me, and what is about to expire? |
| Instance | `/instances/:id` | What is this one machine, is it enrolled, and how do I extend or destroy it? |
| Request capacity | `/request` | What will a machine cost me before I commit to it? |
| Findings | `/findings` | What exists that I cannot account for, and when did anyone last check? |
| Settings | `/settings` | Which providers can I use, do their credentials resolve, and how do they bill? |

Two navigation rules follow from the accountability list. The unaccounted count
on inventory links directly to findings, because unattributed cost should never
require exploration to reach. And nothing on findings can destroy an instance;
that path runs through the instance surface, so marking always precedes
sweeping.

## Cross-References

- `path:../START-HERE.md` — Gate 5
- `path:../guides/choosing-ui.md` — the reasoning behind each section
- `path:../../experience/README.md` — the typed contract
- `path:../../DESIGN.md` — the token contract
- `path:DOMAINS.md` — the domains these surfaces expose
- `path:FLOWS.md` — the journeys these surfaces sit in
