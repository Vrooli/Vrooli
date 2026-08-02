# Unbounded-state fleet audit

> Captured 2026-07-31, following the host disk-exhaustion incident.

## Why this exists

On 2026-07-31 the root filesystem reached 100 percent. The largest single
consumer was `architecture-cartographer`'s `graph_snapshots` table — 77.2 GB
across 2,469 rows — and storage-manager could not see a byte of it, because
that scenario was not registered as an owner cleanup provider.

Three of roughly 125 scenarios exposed a cleanup provider. The one holding the
most data was not among them. That gap, not the growth itself, is what made the
incident invisible until the machine was unusable.

This audit generalises the question: **which other scenarios own state that
grows without a bound, and is that state reachable by the tool built to reclaim
it?**

## Method

For every scenario owning a `data/` directory:

1. Measure the directory (`du -sm`).
2. Look for retention machinery in its API (`retention`, `Retention`,
   `PruneBefore` in non-test Go).
3. Assign a verdict.

Verdicts:

| Verdict | Meaning |
| --- | --- |
| `bounded` | The scenario prunes its own state on a retention policy. |
| `registered` | Registered as a storage-manager owner provider. |
| `exempt` | Growth is bounded by construction, or the footprint is negligible. |
| **`unbounded`** | Grows without a bound and is not reachable by storage-manager. |

## Findings

### Registered owner providers

These four delegate deletion to their owning scenario. All are
`safe_with_owner` and **disabled by default** — registration makes state
*visible*, not automatically deletable.

| Provider | Owner scenario | Data | Verdict |
| --- | --- | --- | --- |
| `architecture-cartographer-snapshots` | architecture-cartographer | 3.6 GB | registered + bounded |
| `workspace-sandbox-retention` | workspace-sandbox | — | registered |
| `test-genie-run-retention` | test-genie | 15 MB | registered + bounded |
| `web-console-sessions` | web-console | — | registered |

### Scenarios with their own retention

Bounded by an owner-side policy. Not registered with storage-manager, which is
acceptable while their own retention holds.

| Scenario | Data | Verdict |
| --- | --- | --- |
| agent-manager | 2.3 GB | bounded — retention across 13 files |
| browser-automation-studio | 1.2 GB | bounded — retention across 17 files |
| tunnel-manager | 226 MB | bounded |
| vrooli-memory | 148 MB | bounded |
| unit-health | 88 MB | bounded |
| git-control-tower | 11 MB | bounded |
| system-monitor | < 1 MB | bounded — metrics retention scheduler |
| data-backup-manager, device-sync-hub, prompt-manager, network-manager, knowledge-observatory, agent-inbox | < 3 MB | bounded |

### Unbounded — the follow-up work

No retention machinery and no cleanup provider. Ranked by current footprint.

| Scenario | Data | Risk |
| --- | --- | --- |
| **experience-manager** | 432 MB | Highest non-registered consumer. Single SQLite file, no pruning found. |
| **signal-inbox** | 401 MB | Inbox-shaped data is append-only by nature; the exact shape that grew unbounded in the incident. |
| **code-facts** | 194 MB | Derived facts re-extracted per run — the same "every distinct code state retained forever" shape as `graph_snapshots`. |
| **workflow-health** | 83 MB | Per-run health records. |
| **security-health** | 38 MB | Per-run findings. |
| react-component-library | 22 MB | Lower priority. |
| scenario-completeness-scoring | 19 MB | Per-run scores; accumulates per scenario per run. |
| ai-gateway | 11 MB | Request/response records. |

**These are recorded, not fixed.** Retention for eight scenarios is outside this
plan's change boundary, which covers only the four scenarios the incident
touched. `code-facts` and `signal-inbox` deserve the most attention: both share
the structural property that caused the incident — a new row per distinct
observation, with nothing that ever deletes one.

### Exempt

Every remaining scenario holds under 2 MB, most of them an empty or
schema-only database. Their growth is bounded in practice by how little they
write; if any starts growing, the check below is what should catch it.

## The check

`TestArchitectureCartographerIsRegistered` and
`TestOwnerProvidersDeleteThroughTheirOwner` in
`api/internal/providers/registry_owner_test.go` pin the registration and the
ownership boundary, so the specific regression that caused this incident
cannot recur silently.

The broader fleet check — failing when a scenario declares a data directory but
registers neither a provider nor an exemption — needs a fleet-wide inventory
this plan's change boundary does not cover. It is recorded here as the open
follow-up rather than half-built.

## Standing decision

Owner providers stay **disabled by default**. Registration makes state visible
to preview; enabling deletion for another scenario's private data is an
operator decision, and the autonomous critical-band path deliberately refuses
`safe_with_owner` providers for the same reason.
