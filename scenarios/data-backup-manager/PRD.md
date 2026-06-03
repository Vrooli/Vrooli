# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Provide Vrooli a dependable, engine-backed backup and verified-restore capability. Scenarios self-register the runtime state they own (databases, filesystem trees, vector/cache stores); the manager snapshots it on a schedule to one or more encrypted destinations and proves it can be restored.
- **Primary users/verticals**: The Vrooli platform itself (every scenario with mutable runtime state — prompt-manager, swarm-manager, agent-manager), plus operators running self-hosted Vrooli installs who need disaster recovery.
- **Deployment surfaces**: CLI (operator + scenario self-registration), API (Connect-RPC for programmatic registration and orchestration), UI (destinations, plans, run history, storage usage, restore).
- **Value promise**: Makes runtime state safe to keep out of git. Removes "we commit it so we don't lose it" as a justification, and gives the platform a real recovery story instead of an implicit one.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Self-registration of targets | Scenarios idempotently register/deregister backup targets (owner+name keyed); catalog is reconstructable from re-registration on boot
- [x] OT-P0-002 | Six source kinds | Capture filesystem, SQLite, Postgres, Redis, Qdrant, and object-storage sources into consistent artifacts
- [x] OT-P0-003 | Multiple destinations | Configure multiple backup destinations (local filesystem, S3/MinIO) as kopia repositories
- [x] OT-P0-004 | Backup plans | Many-to-many plans bind targets to destinations with per-plan schedule and retention
- [x] OT-P0-005 | Scheduled + on-demand execution | In-process scheduler runs plans on cadence; operators and scenarios can trigger a run manually
- [x] OT-P0-006 | Verified restore | Restore a target to a chosen location; a verify mode test-restores to scratch and checksums the result
- [x] OT-P0-007 | Encryption on by default | Every destination is encrypted by default; passphrases and access keys come from the vault resource, never config files
- [x] OT-P0-008 | Storage limits | Per-destination caps that are configurable and default to alert+block (no silent eviction of backups)
- [x] OT-P0-009 | Catalog & run history | List targets, destinations, plans, and runs; show last-success per target and browse snapshot contents
- [x] OT-P0-010 | Health & observability | Health endpoint flags overdue/failed backups; backup outcomes are emitted as events for platform monitoring
- [x] OT-P0-011 | Three coordinated surfaces | API, CLI, and UI all expose the registration / destination / plan / run / restore model over the same Connect-RPC contract

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Quiesce hooks | Targets declare pre/post hooks so live databases get application-consistent snapshots
- [x] OT-P1-002 | GFS retention | Grandfather-father-son retention policies per plan
- [x] OT-P1-003 | Destination dry-run | Verify a destination is reachable and writable before a plan depends on it
- [x] OT-P1-004 | Restore granularity | Restore a single path or point-in-time from a snapshot, not just the whole artifact
- [ ] OT-P1-005 | First-customer migration | prompt-manager registers `store/teams/**` and stops committing it to git
- [x] OT-P1-006 | Alerting integration | Stale and failed backups surface through platform monitoring/notifications

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Additional backends | B2, GCS, Azure, and SFTP destinations via kopia
- [ ] OT-P2-002 | Cross-destination replication | Replicate snapshots between destinations for offsite tiers
- [ ] OT-P2-003 | Backup analytics | Usage and growth trends per destination and target
- [ ] OT-P2-004 | Automated restore drills | Scheduled, reported restore verification across plans

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (Connect-RPC), React + Vite + Tailwind UI, Go CLI — react-vite template.
- Data + storage expectations: SQLite via `modernc.org/sqlite` for the manager's own catalog and run history (per-domain schema, greenfield). Backup artifacts live in kopia repositories, never under the scenario source tree.
- Integration strategy: wrap the `kopia` resource for all repository/snapshot/restore/dedup/encryption work (wrap-not-use); source secrets from the `vault` resource; read source data through each source's resource CLI (postgres, redis, qdrant, minio). No bespoke crypto, dedup, or scheduler-as-a-service (no n8n).
- Non-goals / guardrails: Not a git replacement and not a source-tree backup tool. Does not implement its own encryption or dedup. Does not silently delete backups to stay under a cap. Stays agnostic of which scenarios use it — scenarios register themselves.

## 🤝 Dependencies & Launch Plan
- Required resources: `kopia` (backup engine), `vault` (secrets); source-kind resources used on demand: `postgres`, `redis`, `qdrant`, `minio`.
- Scenario dependencies: none required to run; prompt-manager is the first registration customer (OT-P1-005).
- Operational risks: backups that cannot restore (mitigated by the verified-restore gate); destination living under the root it protects (mitigated by separate-root policy); redis namespace snapshots are best-effort, not transactional point-in-time.
- Launch sequencing: kopia resource ready → destinations + encryption → plans + scheduling → verified restore proven → register prompt-manager store and stop committing it.

## 🎨 UX & Branding
- Look & feel: operational-console tone using the platform default design tokens; calm, status-forward, dark/light parity. Health and storage state are the visual centerpiece.
- Accessibility: WCAG AA; status conveyed by label/icon as well as color; keyboard-navigable destination/plan/run tables.
- Voice & messaging: precise and reassuring — "last verified restore", "next run", "within cap". Avoid alarmist language; reserve red for genuine failure/overdue.
- Branding hooks: inherits platform shell, tokens, and iconography; no scenario-specific branding.

## 📎 Appendix
- Design decisions (kopia wrap, alert+block default, encryption-on, Source/Destination/Plan model): `docs/internal/DECISIONS.md`.
- Companion resource plan: `docs/plans/kopia-resource-plan.md` (repo root).
