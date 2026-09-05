# Plugin publication evidence contract

This is the canonical contract for what a plugin publication claim asserts. A
successful build, a clean scanner run, or an attached signature is not proof
that a published capability works. Every supported claim requires a machine
assertion and reviewer-visible evidence, and composition results are reported
separately from install evidence.

The provider-neutral contract owner is the shared Go module
`packages/delivery-ramp-go`: it owns the journey/evidence schema, disposition
rules, gate vocabulary, and reference-only verdict semantics.
`scenario-to-plugin` is the agent-runtime ramp and owns the Agent Plugins
composition, conformance, attestation, rehearsal, and distribution adapter
code. `deployment-manager` decides release and records what shipped; it never
learns a package format.

This document describes an intended contract for a ramp that is documented but
not implemented. It is not a support claim for any capability.

## Terminology

| Term | Meaning |
|---|---|
| Declaration | The scenario-owned statement of what it publishes: skills, the CLI surface each wraps, entitlement tier, and any MCP server. |
| Package | A composed Agent Plugins 1.0.0 folder, before attestation. |
| Artifact | An attested, digest-addressed package: signature, provenance, and SBOM bound to that digest. |
| Channel | A destination that can receive an artifact — a registry, a marketplace descriptor, or an OCI repository. |
| Publication | The act of placing an artifact in a channel, permitted only by a passing release gate. |

The word *plugin* here means the Agent Plugins packaging unit. It is unrelated
to the commercial tiers in `docs/monetization/strategy/TIERS.md` and to the
technical deployment tiers in
[`scenario-to-desktop-evidence-and-tier-contract.md`](scenario-to-desktop-evidence-and-tier-contract.md).

## Evidence profile

Plugin publication runs the `protocol` profile defined by
`packages/delivery-ramp-go`. `GateVisual` is not applicable: an agent-facing
capability is proven by installing it and running its documented commands, not
by a recording. A ramp that reports `visual_launch` for a plugin target is
misreporting.

The deciding gates are `protocol_readiness`, `semantic_journey`,
`capture_integrity`, `artifact_persistence`, and `governance_reporting`. A gate
with no check is `unverified`, never `passed`.

## Required evidence classes

A publication verdict is complete only when all six are present and each
resolves to a stored, reviewable reference:

1. **Conformance** — the package validates against the Agent Plugins
   specification, and each `SKILL.md` validates against the Agent Skills
   specification. Hidden Unicode, bidirectional marks, and angle brackets in
   frontmatter are rejections, not warnings.
2. **Skill-to-CLI drift** — every command shown in a skill body exists in the
   wrapped scenario's CLI surface at the pinned version. This is the claim that
   distinguishes a maintained skill from a scraped one; a verdict without it is
   incomplete regardless of what else passed.
3. **Install safety** — every download in an emitted install path is pinned to
   an immutable reference and verified against a recorded checksum, and no
   emitted path requests privilege escalation.
4. **Scanner clearance** — a recorded verdict from each configured scanner,
   with the scanner identity and version.
5. **Attestation** — a signature, a provenance attestation, and an SBOM, each
   bound to the artifact digest rather than to a tag.
6. **Clean-room rehearsal** — the artifact installed in an isolated workspace
   that does not carry the build host's Vrooli runtime, followed by execution
   of the declared commands.

## What is not proof

- Composition success is not conformance. A well-formed folder can document
  commands that do not exist.
- Scanner clearance is not install evidence. A package can be clean and still
  fail to install.
- A rehearsal on the build host is not clean-room evidence. Ambient runtime
  state is the specific thing the rehearsal exists to exclude.
- An attached signature is not a trust claim about behavior. It attests origin,
  nothing more.
- A channel accepting an upload is not a publication claim until the channel
  confirms the artifact is retrievable at its digest.

## Reference-only semantics

`deployment-manager` stores evidence references and verdicts, never artifact
bytes. Package bytes, signatures, SBOMs, scanner reports, and rehearsal logs
stay in the producing ramp's capture store. Every emitted `TargetVerdict`
carries references only.

Credentials never enter evidence. Registry tokens, user credentials, and
entitlement material must not appear in an artifact, an SBOM, a rehearsal log,
or a verdict. An entitlement-gated skill packages a sign-in path that resolves
at run time; it does not carry a secret.

## Revocation

A revocation record names the artifact digest, the channels the artifact
reached, and the per-channel withdrawal outcome. A revocation whose per-channel
outcome is unknown is reported as unknown, never as withdrawn. The release
record it supersedes stays in `deployment-manager`.

## Related

- [`scenario-to-desktop-evidence-and-tier-contract.md`](scenario-to-desktop-evidence-and-tier-contract.md) — the precedent this contract follows
- [`cross-ramp-delivery-spine-phase1.md`](cross-ramp-delivery-spine-phase1.md) — the shared spine's seam record
- `scenarios/deployment-manager/docs/decisions/005-governance-plane-boundary.md` — the four-plane split
- `scenarios/deployment-manager/docs/guides/packaging-matrix.md` — the common ramp contract
- `scenarios/deployment-manager/docs/scenarios/scenario-to-plugin.md` — the ramp's governance-facing reference
- `docs/monetization/catalogs/channels/skill-registries.md` — whether a capability *should* be published, which this contract does not decide
