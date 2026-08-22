# Portability Model

## Purpose Of This Document

This is the **single canonical source** for the **Portability axis**: the model that answers *"does this capability resolve on that host OS, and if it does not, is the absence a gap or a decision?"*

It owns two domains — [`portability`](DOMAINS.md#portability) and [`ladder`](DOMAINS.md#ladder) — and it is the document a reader should reach for when the question is about **platforms**, not about **reliability**. [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) owns the reliability denominator and has nothing to say about Windows or macOS. The two words sit close enough together that the wrong one is a plausible first guess; this section exists so that guess is corrected in one line rather than one afternoon.

This document describes the **ideal** model, not what exists today, matching how [`TRUST-MODEL.md`](TRUST-MODEL.md) and the sibling instrument's models are authored: the model is written at full maturity so the distance between it and reality is *measurable* rather than invisible. Current adoption is recorded in [§ Current State](#current-state) as data, never by trimming the model down to what already works.

## Why Portability Is A Separate Denominator

Coverage asks *how much of the platform's reliability is instrumented at all*. Its denominator is the set of cells owners have authored, and every cell is a question about **this** running system.

Portability asks a question that has no reading on this host at all: *would this work somewhere else?* Its denominator is the capability vocabulary crossed with the host OS axis, and most of its cells describe machines nobody is standing on. Folding the two would corrupt both:

> **A coverage ratio answers "how much are we looking at". A portability grid answers "where would this run". A host that is fully instrumented and runs on exactly one OS is 100% covered and barely portable, and a board that averages those two numbers has told the operator nothing true.**

The separation is also what keeps the axis honest about its own blindness. A portability cell for `windows` on a Linux host is not *uninstrumented* — it is **unobservable from here**, permanently, no matter how much sensor work is done. Coverage has no vocabulary for that, and giving it one would weaken the meaning of `MISSING` for every reliability cell.

### Portability is not conformance

The second boundary is the one that decides where work lands, and it is easy to get wrong:

> **Portability asks whether a declaration says a capability resolves on an OS. Conformance asks whether that declaration is true.**

Portability is *measurement*: it has a denominator, it produces a ratio, and it belongs in this instrument. Conformance is *pass/fail*: a manifest that declares `windows` while its handler only compiles on Linux is not 60% correct, it is wrong. Conformance therefore belongs beside the resolver in the control plane, as a gate, and never becomes a cell here. This is the same distinction that keeps `platform-code-audit` deferred as prose in [`DOMAINS.md`](DOMAINS.md#deferred-domains) — the judgment half has no defensible denominator, and inventing one would repeat the production-ledger mistake.

## The Capability Vocabulary

A **capability** is one named platform ability, declared beside the thing that implements it. The vocabulary is **operator-owned and closed**: a declaration naming a capability outside it is an error, not a new capability.

The vocabulary lives in exactly one file:

```
.vrooli/capability-vocabulary.json
```

It carries the capability names and the per-OS `platform_policies` that say when an absent implementation is a deliberate decision rather than a gap. **Every other list of capability names is derived from this file, never maintained beside it.** A hand-maintained second copy — a schema enum, a documentation table, a UI constant — is a drift source, and drift in the vocabulary is not cosmetic: it silently changes which declarations validate.

## Provider Roles And Control Roles

This is the load-bearing distinction in the whole model, and the one the axis is most often got wrong on.

A capability is a *category*. Several things can declare the same one. What differs is **whether they substitute for each other**:

| Role | Semantics | Resolves when | Example |
|---|---|---|---|
| `primary` | Provider. The canonical implementation for a platform. | **Any one** provider resolves on that OS. | `winget` for `package-management` |
| `peer` | Provider. A platform-specific alternative to the primary. | Same as `primary`; used only to break ties. | `apt-get` for `package-management` |
| `control` | **Not a provider.** An independently required control. | **Every** `control` declarer resolves on that OS. | `remote_session_protection` for `remote-desktop` |

> **Providers are `OR`. Controls are `AND`.** A host with `winget` does not need `apt-get` — those are alternatives, and reporting the capability green is correct. A host with remote-desktop *access* is not thereby protected by remote-desktop *session hardening* — those are different controls that happen to share a category, and reporting one green because the other resolved is a false claim about the security posture of the machine.

Every **safeguard** is a `control`. This follows from what a safeguard is: `internal/safeguards/` holds host controls that `vrooli setup` installs, each of which does one thing no sibling does. Five safeguards declare `crash-forensics` — `pstore-native`, `pstore-ramoops`, `crashkernel-reserve`, `kdump-observability`, `pstore-observability` — and they are not five ways of doing the same job. They are five jobs.

Every **tool** is a provider. This also follows from what a tool is: an installable executable that satisfies a need some other tool could satisfy instead.

### Why the resolution must report the absent declarers

Resolving a capability produces a winner. It must also produce **everything that did not resolve**:

> **A capability resolution that reports only its winner cannot distinguish "this OS is covered" from "this OS is covered by one thing out of four".**

This holds for providers too, not only controls. `package-management` resolving through `winget` on Windows while `apt-get`, `dnf`, `pacman` and `rpm` are Linux-only is the correct verdict — and *"resolves via winget; apt-get, dnf, pacman, rpm are Linux-only"* is a strictly better readout than a bare green lamp. The absent set is the difference between an operator trusting the grid and re-deriving it by hand.

## Resolution, Situation And Qualification

Three vocabularies describe one cell, and they answer three different questions. Collapsing any two loses information the operator needs.

**Resolution status — *does it run here?*** Closed vocabulary:

| Status | Meaning |
|---|---|
| `implemented` | An implementation is available on this host OS. |
| `degraded` | An implementation is available with known functional limits. |
| `unwired` | A mechanism is named for this host OS but no implementation is declared for it. |
| `ineligible` | Every declaration deliberately marks this host OS out of scope, and none names a mechanism to wire. |
| `peerless` | Nothing at all is declared for this host OS. |
| `status_invalid` | A declaration authored a platform status outside the vocabulary. Terminal — a verdict about the manifest, not the platform. |

**Qualification — *how much do we believe it?*** This is deliberately independent of resolution, because **a cross-compiled implementation and one proven on real hardware both resolve as implemented and are not the same claim**: `qualified`, `build_verified`, `unqualified`, `degraded`, `ineligible`, `undeclared`.

**Situation — *is the absence a gap or a decision?*** This is the row-level classification an operator actually reads, and the only one that carries intent:

| Situation | Meaning |
|---|---|
| `built_everywhere` | A declared implementation resolves on all three host OSes. |
| `no_work_required` | The absence is policy: this OS was deliberately scoped out and the vocabulary's `platform_policies` says so. |
| `no_equivalent_ever` | No equivalent mechanism exists on that platform. Closing this is not a matter of effort. |
| `real_peer_nobody_wired` | A real mechanism exists on that platform and nobody has wired it. **This is the only situation that is unambiguously work.** |

## The Device Ladder

The `ladder` domain grades **device class × rung × host OS**. Rungs are ordered, not scored:

```
identity  →  telemetry  →  anticipation
"what is it"  "what is it doing"  "when will it fail"
```

A device class cannot hold a higher rung than the one below it, because you cannot anticipate a failure on a device you cannot name. The ladder joins three typed sources through `api-core/discovery` — `system-monitor/device-graph`, this scenario's own `portability` domain, and `vrooli-autoheal/check-platforms` — and reports source availability explicitly, so an unreadable source becomes a visible entry rather than a silently shorter grid.

### The single-host limit is structural, and must read as such

Every ladder cell for a host OS other than the one this instrument runs on reports `unread` with zero devices seen. **You cannot read a Windows thermal sensor from a Linux host.** No amount of sensor work changes that; it needs a second machine.

This is honest, and it must stay distinguishable from an instrumentation failure. `unread` because nothing was sampled and `untrusted` because a sample failed are different facts with different owners, and a reader who cannot tell them apart will either chase a sensor bug that does not exist or ignore one that does.

## Where Declarations Live

Two placement rules, each with a reason that survives restatement.

**Declarations stay with the thing they describe.** A tool's platform block belongs in that tool's manifest; a safeguard's belongs in `safeguard.json`. The `portability` domain only ever *reads*. Pulling the declarations into the instrument would make this domain a second roster of the fleet, and a roster drifts from the thing it lists the moment either side changes — the failure operating-model rule 6 exists to prevent.

**The resolver stays in the control plane.** `internal/deployability` is a pure function over declarations, and it lives in `internal/` because **`vrooli setup` must resolve a capability with no scenario running.** Scenarios reach it through the `packages/deployability` shim. The split is therefore: control plane owns *resolution*, this instrument owns *aggregation, trust, history and ranking*, and `vrooli capability ledger` is the flat readout for anyone who needs the answer without an instrument.

| Question | Surface |
|---|---|
| What does the repo declare, right now, with nothing running? | `vrooli capability ledger` |
| Which scenarios are blocked on which OS? | `vrooli capability fleet` |
| The same, with situation classification and trust | `infrastructure-manager portability grid` |
| How far up the ladder is each device class? | `infrastructure-manager ladder status` |
| Is a declaration actually *true*? | The conformance gate — see [§ Current State](#current-state) |

## Load-Bearing Constants

| Constant | Value | Why it is load-bearing |
|---|---|---|
| Host OS axis | `linux`, `macos`, `windows` | Every capability is resolved against all three, so an OS with no declaration is reported as `peerless` rather than being absent from the row. A missing row and a peerless row are different claims. |
| Capability vocabulary | `.vrooli/capability-vocabulary.json` | The single source. Every other list is derived. |
| Delivery tiers | `tier-1-local` … `tier-5-enterprise` | The fleet readout resolves each scenario's dependency closure per tier; a tier upgrade is a computed single-change delta, never an authored claim. |
| Manifest root | Reported on every readout | A grid computed against the wrong tree is a complete-looking answer about a repository nobody asked about. A missing root is an error, never an empty grid — an empty grid reads as "this repository declares no capabilities", which is a claim rather than a failure. |

## Current State

Recorded as data. The distance between this model and the code is the point of measuring it.

| Model element | State | Evidence |
|---|---|---|
| Capability vocabulary as single source | **Held by a live drift gate.** Both consumer schema enums are checked against the 41-name vocabulary in the control-plane test surface. | `internal/deployability/capabilityvocabulary_test.go` |
| `control` role | **Implemented.** Safeguards declare `control`; providers remain `primary` or `peer`. | `.vrooli/schemas/safeguard.schema.json`, `internal/safeguards/*/safeguard.json` |
| `AND` resolution for controls | **Implemented.** A provider only resolves fully when every control resolves; incomplete cells retain the provider and name absent controls. | `internal/deployability/capability.go` |
| Absent declarers reported | **Implemented.** Every resolution branch reports unresolved declarers, including the winner branch. | `internal/deployability/capability.go` |
| Scenario participation | **2 of 120.** Only `system-monitor` and `vrooli-autoheal` declare `service.platform_capabilities`, contributing 14 of the 41 capabilities between them. | `scenarios/*/.vrooli/service.json` |
| Safeguard enumeration | **Available.** `vrooli host safeguard list` reports every manifest with capability, role, declared platforms, and an explicit observed-state value; focused lookup accepts hyphenated and underscored names. | `internal/cli/vroolicli/hostinstall.go` |
| Conformance gate | **Available.** `vrooli capability conformance` discovers claims, cross-compiles their Go modules, and fails with manifest-level compiler evidence; the repo contract names the gate. | `internal/deployability/conformance.go`, `.vrooli/repo-contract.json` |
| `unread` vs `untrusted` on non-local OSes | **Not distinguished.** Non-local ladder rows carry the generic untrusted reason rather than a structural "no such host was sampled" code. | `api/internal/ladder/` |

## Governing Principles

1. **Portability is a separate denominator from coverage.** Never average them, never file one under the other's name.
2. **Providers are `OR`; controls are `AND`.** A capability is a category, not a unit of coverage. Every safeguard is a control.
3. **Report the absent declarers, always.** A winner-only resolution cannot distinguish full coverage from partial.
4. **Declarations stay with the thing they describe.** This domain aggregates and never authors. A roster here would drift by construction.
5. **The resolver stays in the control plane.** `vrooli setup` must resolve with nothing running.
6. **One vocabulary, derived everywhere else.** A second hand-maintained list is a drift source that silently changes what validates.
7. **Unobservable is not uninstrumented.** A cell for an OS nobody is standing on must say so in its own words, and must never be counted as a sensor gap somebody could close.
8. **Measurement here, conformance in the control plane, judgment in the audit lane.** A declaration that is false is a gate failure, not a lower ratio.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — the `portability` and `ladder` domain contracts
- [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) — the reliability denominator this axis is *not*
- [`CONDITION-MODEL.md`](CONDITION-MODEL.md) — legs, banding, reading history
- [`TRUST-MODEL.md`](TRUST-MODEL.md) — the closed trust vocabulary applied to readings
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — the typed source contracts the ladder joins
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules

Upstream canon this axis implements (cited, never restated):

- `internal/deployability/` — the pure resolver and its vocabularies
- `.vrooli/capability-vocabulary.json` — the operator-owned capability list
- `docs/infra-health/operating/OPERATING_MODEL.md` rules 3 and 6 — supervise-don't-operate, contract-not-roster
