# Scenario reference: scenario-to-plugin

`scenario-to-plugin` is the agent-runtime ramp. It owns Agent Plugin
composition, skill and MCP conformance, supply-chain attestation, clean-room
install rehearsal, and the publication handoff to agent runtimes and skill
registries. deployment-manager owns the profile, target decision, release gate,
and release record.

Status: documented, not implemented. The contract below is the intended shape
and is not a support claim.

## Current route

The ramp produces one output shape:

- **Agent Plugin package** — an Agent Plugins 1.0.0 folder carrying a root
  `plugin.json`, optional `skills/` (Agent Skills `SKILL.md`), and optional
  `mcp.json`, published with a Cosign signature, SLSA provenance, and a
  CycloneDX SBOM attached as referrers.

Distribution is adapter-shaped rather than format-bound. Agent Plugins 1.0.0
and the Claude Code native plugin layout are different schemas at different
paths, and neither is treated as canonical. A second adapter must not require
re-authoring skill content.

The ramp consumes standalone-installable capabilities only. Making a scenario
standalone-installable is an upstream prerequisite, not ramp work; the ramp
refuses composition when it is unmet rather than deferring the failure to
rehearsal.

## Evidence profile

This ramp emits `protocol`-profile journeys, not `visual`. A published plugin
is proven by a clean-room install and command exercise, not by a recording, so
`GateVisual` is not applicable and `GateProtocol`, `GateJourney`,
`GateCapture`, `GatePersistence`, and `GateGovernance` carry the decision. The
provider-neutral semantics are owned by `packages/delivery-ramp-go`; the claim
vocabulary is defined in
[plugin publication evidence contract](../../../../docs/reference/plugin-publication-evidence-contract.md).

A ramp verdict that omits scanner clearance, signature, provenance, SBOM, or
the skill-to-CLI drift result is incomplete and must not be reported as
passing.

## Integration contract

The ramp calls deployment-manager for target planning, approval, and
release-gate decisions, and reports exact source, target, artifact, and
evidence identity back to the governance plane. It publishes only after the
gate permits it, for the same source commit.

The ramp must not:

- publish to any channel before the release gate permits it;
- report a package as publishable when the skill-to-CLI drift check has not run;
- describe composition or scanner success as proof that the package installs;
- embed registry credentials, user credentials, or entitlement secrets in an
  artifact, an SBOM, or a verdict;
- request or require privilege escalation in any emitted install path;
- publish a package that pulls the full Vrooli runtime without disclosing it;
- treat a passing rehearsal on the build host as clean-room evidence.

## Revocation

Revocation is a ramp responsibility with a governance record. When a published
version is withdrawn, the ramp withdraws or flags the artifact in every channel
it reached and records the per-channel outcome; deployment-manager holds the
release record that the revocation supersedes.

Read the [scenario-to-plugin PRD](../../../scenario-to-plugin/PRD.md) for the
operational targets and the
[packaging matrix](../guides/packaging-matrix.md) for the common ramp contract.
