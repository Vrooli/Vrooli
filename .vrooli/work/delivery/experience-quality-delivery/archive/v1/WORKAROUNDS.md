# Infrastructure and workaround record

## Open

### S01 — documented finite-leader provisioning is absent in current CLI

- Observation, 2026-10-16: `prompt-manager team heartbeat-bind-effort` returns `unknown subcommand`. The skill and docs/reference/heartbeat-cli.md prescribe this disabled binding path. Current CLI teams/teams.go and heartbeat request models omit finite-leader handling; existing retained team's heartbeat read omits its historical finiteLeader fields.
- Interpretation: an observed documentation/runtime capability mismatch. Removal intent and rollout history have not been established. No authentication/admission denial occurred on this read. This is not proof that the machine or Agent Manager is unavailable.
- Current approach: create the native finite team, source-ledger scope, leader/member documents, full operating contract and a disabled on-demand heartbeat via supported owner operations; retain the proposed binding as unapplied. Do not write runtime storage, resurrect removed code or invent a scheduler. Report through scenario-qa and retain its receipt in SETUP.md.
- Return condition: the owner supplies a currently supported finite-effort/liveness provisioning route and its evidence, or the discrepancy is repaired by its owner. Requalify before activation; preparation and source review continue.
- Intake receipt: scenario-qa `knw-1792165359676022214` under `bug-inbox/prompt-confusion/finite-delivery-setup-skill-prescribes-unavailable-heartbeat-binding`; publication verified, repair/admission not inferred.
- Outcome: runtime dispatch remains prohibited by the owner's instruction regardless of this issue.

### S02 — discovery providers partly degraded

Initial Search Hub recall returned useful records but source-ledger recall/agent-manager/CLI providers reported timeout or open reranker circuits. Direct maintained source and native owner reads supplied the preparation evidence. Preserve provider degradation as retrieval limits; no outage inference beyond those exact reports. Do not rerun broad search as a polling loop.

## Resolved

None. Log repeated workarounds and route durable fixes to the owning domain; do not harden preparation scripts into permanent runtime infrastructure.
