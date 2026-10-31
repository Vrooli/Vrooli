# Work entry point

Choose the work model from the outcome. Team definitions stay in Prompt Manager's common
registry; the work owner retains authoritative state. This directory holds delivery product
homes. Managed plans and experiments stay with their existing owners.

| Work | Use | Authoritative state |
| --- | --- | --- |
| Bounded task or investigation | A direct assignment with a verifiable finish | Assignment and the owning scenario's evidence |
| Ordered implementation | Plan Manager plan; family when several plans need coordination | Plan Manager managed store and its returned artifact directory |
| Adaptive improvement | Approved mandate with target and sensors | Mandate owner; Swarm/Agent Manager own admitted execution |
| Finite product delivery | One orchestrator and independent surface workers | Product home here; native commission in Agent Manager |
| Experiment | Existing domain experiment owner | Its managed store; no second writable experiment register here |

## Delivery ownership

| Owner | Holds |
| --- | --- |
| Prompt Manager `team.json` | Team identity, members, heartbeat policy and `homePath` |
| `delivery/<team-id>/CHARTER.md` | Product target, selected sources, acceptance, constraints and proposed budget |
| `STATUS.md` | Current surface assessment, active units, forecast and one handoff |
| `LOG.md` | Accepted-unit IDs, worker attribution, evidence and reopen records |
| `FEEDBACK.md` | Operator/supervisor directions and their dispositions |
| Agent Manager enrollment | Destination reference, persisted commission clock, worker limit, terminal snapshot and operator decision |

A team queue limit controls heartbeat member runs. The commission separately limits native
workers under one orchestrator. A proposed charter budget starts no clock. Research observes
product delivery; it does not govern product acceptance. Archive references preserve
evidence and applicable product obligations, not obsolete assertions that a research
benchmark is the full product contract. Required behavior, protection, accessibility,
recovery and independently evidenced acceptance remain intact.

The four current files are the resume path. Keep CHARTER + STATUS + open FEEDBACK below
30,000 bytes. Active worker briefs remain in `epochs/`; selected acceptance sources belong in
`sources/` or explicit authoritative references. Archives preserve history and unresolved
obligations; their old instructions do not silently supersede the current charter.

Charter, feedback and durable source maps are eligible for Git. STATUS, LOG and operational
evidence are ignored. Preserve ignored runtime records through their owner; a source commit
is not a backup of the current commission or evidence.

## Operate a delivery team

1. Read its charter, status, open feedback and current native effort board. Reconcile retained
   owners before a new commission; parked is not terminal. Record supersession of obsolete
   instructions without deleting their history or weakening product requirements.
2. Qualify required sources/routes and isolated fixtures. Keep unmet human-only decisions
   explicit; continue independent authorized work when a held journey does not block it.
3. Commission through `agent-manager effort delivery start --effort-ref <ref> --commission <id>
   --budget 24h --workers 2` under the existing operator owner path. Then enable only the intended
   team through Prompt Manager. No migration or source edit implicitly activates it.
4. Use `effort delivery pause|resume|stop --effort-ref <ref>` for the native clock; pause and stop
   require `--reason`. Disabling heartbeat prevents new heartbeat dispatch; it does not pause
   retained execution or the clock. Pausing requires parked/terminal owners. Blocked time counts
   unless the operator explicitly pauses.
5. Read the native pressure: ON TRACK before 75%, APPROACHING at 75%, OVER at 100%, CONVERGE at
   150%, HARD STOP at 200%. A 24h target has a 48h hard stop, excluding explicit pauses.
   Time changes priorities and stopping, never acceptance. At HARD STOP, new/queued work is
   fenced; the owner gives two minutes from actual observation to checkpoint, stops remaining
   runs, verifies terminal readback, freezes outcomes and raises an operator decision.
6. Review accepted, partial, remaining and unverified outcomes and the actual decision's
   publication/answer. Accept a partial result, recommission or change scope explicitly.
   A stopped commission cannot restart itself through heartbeat.

The [owner reference](../../scenarios/agent-manager/docs/reference/effort-supervision.md)
provides commands and formats. Skills `large-effort-orchestration` and
`large-effort-supervision` provide judgment; they do not own another clock or ledger.

## Migration cohort dispositions — 2026-10-23

The consolidation cohort contained 18 finite delivery records, accounted for below. Discover
current teams through Prompt Manager; this dated cohort is not a live registry. Team/clock state is a live owner fact;
this inventory does not authorize activation, infer product acceptance or replace that owner.

| Teams | Disposition and authoritative home |
| --- | --- |
| browser-automation-studio-delivery; desktop-packaging-delivery; experience-quality-delivery; external-knowledge-delivery; marketing-i16-delivery; nooch-delivery; notification-hub-delivery; personal-planner-delivery; rcl-prompt-manager-ux-design-campaign | Adopted: `delivery/<team-id>/`; registry `homePath` and enrolled destinations agree |
| aquila-commercial-release | Retain managed effort: `aquila-commercial-release-2026-09-17` |
| aquila-marketing-and-capability-maturity | Retain managed effort: `aquila-marketing-capability-maturity-2026-09-17` |
| aquila-product-presentation | Retain managed effort: `aquila-configurable-product-presentation` |
| fleet-storage-disk-health | Retain managed mixed investigation/task effort: `fleet-storage-disk-health-2026-09-14` |
| landing-page-vps-qualification | Retain managed qualification: `landing-page-business-suite-vps-qualification` |
| unattended-self-improvement | Retain managed Plan Manager family: `unattended-self-improvement` |
| aquila-launch | Already archived; retain historical `aquila-launch-2026-09-17`, not another product commission |
| qualification-finite-20260914; qualification-finite-success-20260914 | Archived disposable fixtures, disabled; retain registry and native qualification evidence, not product homes |

Managed effort slugs above resolve beneath the existing runtime effort root, normally
`~/.vrooli/plan-artifacts/efforts/`. Each retained managed team's existing operating contract
links its actual README. Their plan/mixed-work concurrency and permissions remain owner-defined;
being finite does not require conversion into product epochs. The historical qualification
records are not reusable product teams; prepare a fresh isolated fixture for another test.
Archiving preserves native evidence: the success fixture has terminal owner readback; the
other exact effort was not returned, so its historical outcome remains unverified.

The [reorganization record](../../docs/internal/delivery-reliability-research/DELIVERY_TEAM_REORG_PROPOSAL.md)
retains decisions and receipts. Its migration sections explain history; this page and the owner
reference explain current operation. Plans/experiments relocation is deferred, not promised work.
