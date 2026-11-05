# Go To Market — Scenario to Plugin

This document records how the capability this ramp publishes reaches an
audience. It is deliberately about *what this ramp puts into the world*,
not about the ramp itself — nobody outside Vrooli will ever run
Scenario to Plugin.

## Purpose Of This Document

Use this document to answer:

- Who is the audience for a published package, and what do they need to
  believe before installing it?
- Which channels apply, and which explicitly do not?
- What has to exist before a launch is honest?
- What experiments would falsify the hypothesis?

## Audience And Positioning

The audience is **agents, not humans**. A coding agent running in Claude
Code, Codex CLI, Cursor, Copilot, Kiro, or Windsurf encounters a task,
searches available capabilities, and decides whether to load and run one.
That decision is made from a description, a permissions declaration, and
whatever trust signals the registry surfaces.

This changes what positioning means. Persuasion does not work on this
audience; verifiable properties do. The positioning is therefore a claim
about evidence rather than about benefit:

> A Vrooli plugin is one whose documented commands are proven to exist in
> the CLI it wraps, whose install is proven to work on a machine with no
> Vrooli runtime, and whose signature, provenance, and SBOM are attached
> to the artifact you are about to run.

The competitive context makes this sharp. Skill directories now index
hundreds of thousands of entries scraped from public repositories with no
security review, and a meaningful share of published skills carry at least
one critical issue. Volume is not the scarce thing. **Verifiability is.**

Two structural advantages are ours specifically and neither is marketing
language:

- **Wrap-not-use.** The wrapped CLI enforces auth, scope, and audit at its
  own layer, so a narrow `allowed-tools` declaration is honest rather than
  cosmetic. Skills that must request broad shell access to be useful fail
  scanner heuristics; ours do not need to.
- **Proven non-drift.** No registry scanner checks whether a skill's
  documented commands still exist. Ours does, and records the manifest
  revision it checked against.

## Channels

| Channel | Applies? | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|---|
| `skill-registries` | **Primary.** | Agents discover capabilities through registries rather than search. A signed, scanner-clean, non-drifting plugin gets curated, and curation drives installs of the underlying scenario. | One standalone-installable capability; one composed and published package; per-plugin attribution. | Install count by registry, referrer traffic, curation tier, install→subscription correlation. |
| `oss-discovery` | Adjacent. | A published package is also a public artifact with its own discoverability. Incidental, not a separate effort. | None beyond publication. | Inbound references that did not come through a registry. |
| `community-content` | Adjacent, optional. | "We built a drift gate for published agent skills" is a credible technical story precisely because almost nobody does it. | A factual writeup, no paywall framing, linking to the capability rather than the company. | Referral traffic and inbound questions about the drift gate specifically. |
| `in-product-expansion` | No. | Internal to a Vrooli installation; this channel is external. Different surface, no overlap. | — | — |
| `web-seo` | No. | Human audience, different discovery flow. A published plugin may drive incidental traffic; do not conflate the two when measuring. | — | — |
| `app-stores` | No. | Human users and a completely different review, signing, and distribution apparatus. | — | — |

Two anti-patterns are penalized by the channel mechanics themselves and
must be respected as hard rules rather than preferences:

- **Do not pay for placement.** Agents discount "Sponsored" and "Promoted"
  labels. Paid placement is negative signal on this channel.
- **Do not broaden `allowed-tools` for convenience.** It lowers trust
  score and is caught by `PLG-CONF-TOOLS` before publication anyway.

## Launch Motion

1. Make one capability genuinely standalone-installable. This is upstream
   of this ramp and is the actual blocker; everything else is downstream
   of it.
2. Compose, check, attest, and rehearse that capability's package. Do not
   publish yet.
3. Read the rehearsal evidence as an outsider would. If the journey does
   not convince a skeptical reader that the thing installs and works on a
   clean machine, fix that before publishing rather than publishing and
   explaining.
4. Publish to the signed OCI channel. Confirm by retrieval.
5. Open `skill-registries` and `oss-discovery` together, once. The package
   gives the community something to run; a factual writeup explains why
   the drift gate exists.
6. Instrument attribution before, not after — a launch without it produces
   an unfalsifiable result.
7. Hold for the pilot window. Add a second capability only after the first
   has evidence, not because the pipeline now exists.

The Claude Code adapter (`OT-P1-001`) is a strong second step but not a
launch gate: it is a second format for the same package, and adding it
before one package has been accepted anywhere would be building breadth
before proof.

## Messaging

Precise and non-promotional, in the same voice as the product surface.

- Say what was checked and what it found. Never say "secure" or "verified"
  without naming the evidence behind the word.
- Credit the ecosystem rather than attacking it. The honest wedge is
  complementary: the standard is good, the tooling is young, and drift is
  a real unsolved problem that we happen to have the substrate to solve.
- Never describe a capability as published until a channel has confirmed
  retrieval.
- Never imply a package is endorsed by a registry when it is merely listed.

Anti-messaging, explicitly: do not market the ramp. External audiences do
not care that Vrooli has a publishing pipeline; they care whether the one
thing they are about to install works and can be trusted.

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Publish one capability's package and measure installs | `skill-registries` | To be set with the monetization team. | If installs materially exceed direct Vrooli installs, standalone capability leads and the bundle framing follows. |
| Measure install → subscription correlation over the pilot window | `skill-registries` | To be set with the monetization team. | No measurable correlation by the end of the window means sunset the channel, not publish more packages. |
| Track drift-gate failures against published versions over time | in-product telemetry | Any published package failing re-verification is a process failure. | A rising rate means skill ↔ CLI ownership is wrong and the declaration should move closer to the CLI. |
| Compare curation tier against self-published listing | `skill-registries` | Curated placement in at least one registry. | Failing to earn curation with full attestation means the trust story is not the bottleneck; find the real one. |

Thresholds are deliberately unset. Inventing them here would create
numbers with no provenance; they belong to the monetization team and to
Offer Desk's trigger machinery, where they can be evaluated rather than
argued.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — commercial role and buyer
- [`../../PRD.md`](../../PRD.md) — operational targets and launch sequencing
- [`../../../../docs/monetization/catalogs/channels/skill-registries.md`](../../../../docs/monetization/catalogs/channels/skill-registries.md) — channel doctrine, anti-patterns, telemetry
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — what the ramp actually does
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — the trust claims behind the positioning
