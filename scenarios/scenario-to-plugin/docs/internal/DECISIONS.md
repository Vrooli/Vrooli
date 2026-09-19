# Decisions — Scenario to Plugin

Durable choices made for this scenario, with the reasoning that would
otherwise have to be rediscovered. A decision belongs here when reversing
it would be expensive or when a future agent would plausibly re-litigate
it.

## Purpose Of This Document

Use this document to answer:

- Why is the scenario shaped this way?
- Which choices are settled, and on what grounds?
- What would have to change for a decision to be revisited?

Decisions that affect the whole fleet belong in repository-level ADRs, not
here. This page holds scenario-local choices.

## Decision Log

### D-001 — This is a delivery ramp, not a publishing tool

**Decision.** Scenario to Plugin implements the common ramp contract and
sits in the Ramp plane defined by ADR-005, alongside `scenario-to-desktop`
and the mobile ramps. It consumes a target plan, produces artifacts, runs
target-native validation, emits evidence, and asks `deployment-manager`
for the release decision.

**Why.** A standalone publishing script would have re-derived approval,
evidence, and release-record semantics that the governance plane already
owns, and it would have had no answer for "who approved this". Modeling it
as a ramp also means the shared spine's `Builder`/`Driver`/`Distributor`
seams and its evidence profiles apply directly.

**Revisit if.** ADR-005's plane boundary changes.

### D-002 — Reuse `packages/delivery-ramp-go` rather than defining local evidence

**Decision.** Journey, evidence, verdict, disposition, and validation-matrix
semantics come from the shared spine. This scenario implements the seams
and owns only its adapters.

**Why.** The spine already defines a `protocol` evidence profile that does
not require visual artifacts, which is exactly this ramp's shape — there
is no screen to record. Defining a parallel evidence model would have
produced a second dialect for `deployment-manager` to understand and would
have made cross-ramp comparison impossible.

**Revisit if.** The spine cannot express a needed disposition. Then extend
the package, where every ramp inherits the change — not this scenario.

### D-003 — The evidence profile is `protocol`, not `visual`

**Decision.** Rehearsal emits `ProfileProtocol` and does not attempt
visual capture. `GateVisual` is not applicable.

**Why.** The desktop and mobile ramps record video because a human-visible
launch is the claim. Here the claim is that an agent can install and run
commands, which is proven by exit statuses and redacted output. A
screen recording of a terminal would be theater, not evidence.

**Revisit if.** A future channel requires a human-reviewable artifact.

### D-004 — The drift gate is P0, not a lint

**Decision.** A skill documenting a command absent from the wrapped CLI's
pinned `cli-manifest` fails the package.

**Why.** This is the failure mode most likely to occur over time and the
one no registry scanner detects. It is also our only defensible
differentiator in a market with hundreds of thousands of unreviewed
skills — volume is abundant, verifiability is not. Making it a warning
would mean it is ignored exactly when it matters.

**Revisit if.** Never as a downgrade. If the check produces false
positives, fix the resolution logic rather than lowering the severity.

### D-005 — Compare against a *pinned* manifest revision, and record it

**Decision.** Every drift check records the `cli-manifest` revision it
compared against.

**Why.** Without it, a drift failure is ambiguous: the skill may have
regressed, or the CLI may have moved under a correct skill. The remediation
differs completely. Recording the revision also makes a published version
re-verifiable against the surface it was actually built against.

**Revisit if.** Never. Do not backfill or normalize this column.

### D-006 — No shared resources

**Decision.** SQLite plus a local capture store. No Postgres, Redis,
Qdrant, or Ollama.

**Why.** A ramp that itself needed the full resource fleet to produce a
standalone package would contradict the contract it enforces, and it would
weaken the rehearsal's isolation claim. The data is low-volume,
single-writer, and host-scoped.

**Revisit if.** Multi-host publication coordination becomes real.

### D-007 — No model in any gate

**Decision.** Every gate is deterministic. Model output may inform
advisory suggestions but may never decide a gate.

**Why.** A verdict must be reproducible by a third party from the emitted
evidence — a registry, an auditor, or a consuming agent. A model-judged
supply-chain gate is unreproducible and therefore unverifiable, which
defeats the point of attesting anything.

**Revisit if.** Never for gating. `OT-P2-003` (drift-repair proposals) is
the supported advisory shape.

### D-008 — Skill content is owned by the wrapped scenario, not by this ramp

**Decision.** This ramp reads declarations and composes packages. It never
authors or edits skill content, and it never writes into another
scenario's tree.

**Why.** Co-locating skill content with the CLI it documents is what makes
drift a local, visible concern at CLI-change time. An authoring surface
here would move ownership to the wrong place and make drift *more* likely,
not less — the maintainer changing the CLI would not see the skill.

**Revisit if.** Evidence shows scenario owners cannot maintain skill
content in place.

### D-009 — Distribution is an adapter seam; no format is canonical

**Decision.** A channel is an adapter over one already-composed package.
Signed OCI ships first; the Claude Code marketplace descriptor is a second
adapter over the same package.

**Why.** There are at least two incompatible plugin manifest formats in
active use — the cross-vendor Agent Plugins root manifest and Claude
Code's own, which share a filename but nothing else. The packaging
landscape is young and contested. Treating any one as canonical would
force re-authoring when it loses, and would double authoring cost for
every additional target.

**Revisit if.** The formats converge. Even then, keep the seam.

### D-010 — Publish requires two parties; revoke requires one

**Decision.** Publication needs an operator *and* a `deployment-manager`
release decision bound to the same source commit. Revocation needs only an
operator.

**Why.** Making something reachable should be harder than making it
unreachable. Requiring approval to *stop* shipping something would slow
incident response for no security benefit.

**Revisit if.** Never.

### D-011 — Confirm publication by retrieval, never by a successful push

**Decision.** A channel is recorded as published only after the artifact
is retrievable at the published digest.

**Why.** A successful upload call is not a successful publication.
Recording published state from a request is how a channel silently drops
an artifact while the console reports success — and it would corrupt the
revocation fan-out, which is derived from publication history.

**Revisit if.** Never.

### D-012 — A revoked publication is not deleted

**Decision.** Revocation records withdrawal. It never deletes the
`publications` row.

**Why.** Deleting the row erases the fact that the artifact was ever
published, which is the opposite of what an incident response needs, and
it destroys the fan-out needed to withdraw from remaining channels.
`revoked_partial` is a real terminal state because some registries cannot
hard-delete a version.

**Revisit if.** Never.

### D-013 — Standalone install is an upstream prerequisite, not this ramp's job

**Decision.** This ramp fails closed when a scenario is not
standalone-installable. It does not attempt to make it so.

**Why.** Standalone install is an architectural property of the wrapped
scenario. A ramp that papered over it would produce packages that pull the
full runtime by stealth — the specific trust violation the channel
doctrine names, and the one `PLG-REHEARSE-NO-STEALTH` exists to catch.

**Revisit if.** Never. If this blocks every scenario, the correct response
is to make one scenario standalone-installable, not to relax the gate.

### D-014 — Retire the pre-standard MCP implementation without migrating its templates

**Decision.** `scenario-to-mcp` is prior art only. The ramp may reuse the
idea of declaring tools and launching a stdio server when it composes an
optional MCP adapter, but it must not copy the old manifest or server
implementation.

**Evidence reviewed.** `scenarios/scenario-to-mcp/templates/manifest-template.json`
uses a hand-rolled top-level `protocol_version: "1.0"` and embeds Vrooli
ports and a scenario path. That shape is not the Agent Plugins `mcp.json`
contract. `templates/server-template.js` is a Node stdio server that imports
the MCP SDK, executes child processes, calls axios, and attempts registry
registration; it is not a portable, declaration-owned adapter.

**Reusable.** The explicit tool list, server command/argument split, and
stdio transport are useful inputs to `PLG-COMPOSE-MCP` after validation.
**Not reusable.** The legacy protocol shape, hard-coded Vrooli runtime
environment, child-process escape hatch, mutable registry callback, and
template interpolation are rejected because they bypass the new declaration,
conformance, and governance gates.

**Revisit if.** The Agent Plugins MCP component contract changes.

### D-015 — Declarations own source paths; packages use fixed Agent Plugins locations

**Decision.** A wrapped scenario declares the owned `skills/<name>/SKILL.md`
source and standalone artifacts in `.vrooli/service.json`. Composition copies
those files into the fixed package locations and writes only the closed
Agent Plugins 1.0.0 metadata manifest; skill prose is never synthesized.

**Why.** This keeps ownership and drift visible at the source boundary while
matching the portable package contract. The declaration is the machine-checkable
readiness predicate; `plugin.json` is metadata, not a second component registry.

**Revisit if.** The Agent Plugins specification changes its fixed component
locations or the service schema gains a more precise source ownership model.

## Superseded Decisions

| Decision | Superseded by | Note |
|---|---|---|
| `scenario-to-mcp` as the agent-integration surface | This scenario | `scenario-to-mcp` predates the Agent Plugins standard, was never registered in the packaging matrix, was referenced by nothing outside itself, and added MCP endpoints for internal consumption rather than acting as a delivery ramp. Retiring it is part of this scenario's introduction. |

## Cross-References

- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — the pipeline these decisions shape
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency choices and their reasoning
- [`SECURITY.md`](SECURITY.md) — threat model and the no-model-in-a-gate rule
- [`../../PRD.md`](../../PRD.md) — non-goals and guardrails
- `scenarios/deployment-manager/docs/decisions/005-governance-plane-boundary.md` — ADR-005
