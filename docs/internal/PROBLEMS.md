# Documentation Problems

This index retains unresolved documentation follow-ups. Dates identify the
original observations; they are not fresh readiness assessments. Detailed
probes, test results, and subsystem handoffs are preserved in the
[tracker archive](PROGRESS.md#tracker-condensation-and-portable-references--2026-09-07).
Use [documentation placement rules](SEAMS.md#documentation-and-execution-artifacts)
for retention decisions.

## Retired root problem ledger — 2026-09-08

The root `PROBLEMS.md` and `PROBLEMS_TEMPLATE.md` were retired after their
last reported incidents (2025-01-28) and a reference scan found no live
consumer of their embedded-marker or five-minute-scan model. Their detailed
entries were not promoted: they had no current owner, current reproduction,
or recent evidence, and several claims conflict with the current control-plane
and Knowledge Observatory ownership model.

Exact source bytes are preserved under
`plan-artifacts/docs-cleanup-20260908-opportunities-1-2/root-ledger/` with
SHA-256 verification. Reopen a specific issue only after a current owner
reproduces it and records it in the owning scenario or control-plane ledger.

## Capability discovery adoption — 2026-09-06

- **Prompt Manager:** projected skill copies refresh at startup. An explicit
  refresh command with content-drift reporting remains a tooling opportunity.
  Follow the [canonical editing/refresh contract](../reference/cli-commands.md).
- **Discovery behavior:** the revised AGENTS.md rule passed two bounded
  read-only probes. Broader model coverage, discovery-outage behavior, and
  actual TV actuation were not established. Recheck those gaps through
  Agent Manager and the relevant capability owner before claiming coverage.

## Agent knowledge preservation investigation — 2026-09-05

- **Knowledge Observatory / Search Hub:** text and hybrid retrieval returned
  different path identities for the same scoped source
  (Scenario QA `knw-1788585034013675605`). Recheck identity consistency,
  status/supersession metadata transport, and moved-source recovery.
- **Retrieval coverage:** HTML, JSON, and tracker exclusion was observed;
  archive discovery was indirect. The provider evaluation
  `b6398396-ec2c-4012-8d75-f67fbf377e41` met 18/23 expectations.
  Recheck current rules, rejected alternatives, historical context,
  supersession, archive recovery, and degraded retrieval through existing
  owner evaluations. These observations do not establish answer correctness.
- **Recovery:** all 222 original archive members restored with matching hashes
  on Linux. Remote replication, native macOS/Windows restore, and complete
  service-free bootstrap/reference closure remain unverified.

The [placement contract](SEAMS.md#documentation-and-execution-artifacts) owns
HTML classification and preservation. Routine cleanup does not depend on
building a new metadata system. Plan Manager owns plan artifact closeout;
Knowledge Observatory owns documentation discovery and verification.

## Highest-Value Remaining Follow-Up

Retained subsystem handoffs below need owner remeasurement before resuming work.
Their complete dated evidence remains in the tracker archive.

| Owner | Unresolved handoff | Observed |
|---|---|---|
| [React Component Library](../../scenarios/react-component-library/docs/internal/PROBLEMS.md) | Ledger proto layout migration; documentation/surface/i18n/composition findings; broad UI/unit/lint and provider debt. Targeted locale, version-model, and component-test passes did not establish scenario-wide readiness. Scenario QA `knw-1788190879155112230`. | 2026-08-31 |
| [Storage Manager](../../scenarios/storage-manager/docs/internal/PROBLEMS.md) | Define multi-mount recovery allocation and controller tests; free bytes on one device must not count as relief on another. | 2026-09-02 |
| [Control-plane platform support](../reference/platform-support.md) | Native launchd and Task Scheduler watchdog/log-bound safeguard verification; builds and fixtures did not establish native-host readiness. | 2026-09-02 |
