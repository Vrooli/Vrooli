# Research Packet

## Uniqueness Check

- Repo search: `rg -l "scenario-to-cloud" scenarios/`
- Result: `deployment-manager` already contains planning docs for `scenario-to-cloud` and treats it as future/stubbed. This scenario formalizes and implements that packager.

## Related Scenarios (in-repo)

- `scenarios/deployment-manager`: orchestrates tier workflows; will own profiles and invoke this packager.
- `scenarios/scenario-dependency-analyzer`: source of truth for scenario+resource dependency graphs.
- `scenarios/vrooli-autoheal`: must be included in “mini Vrooli” deployments to keep Tier-1 health guarantees.
- `scenarios/scenario-to-desktop`: reference implementation for “scenario-to-*” packagers (API/UI/CLI + requirements tracking).

## Key Decisions (P0)

- VPS-first, Ubuntu-only, SSH + scp tarball deploy.
- Native resources (no Docker) via a “mini Vrooli” bundle plus a deployment-local native `vrooli` binary that runs setup on the VPS.
- Edge via Caddy + Let’s Encrypt; DNS is manual prerequisite.
- Fixed ports on VPS: UI 3000, API 3001, WS 3002 (overridden at start time).

## External References (starter list)

- Caddy HTTPS automation (Let’s Encrypt HTTP-01)
- SSH automation patterns (future P1/P2 hardening)
- Idempotent deployment patterns (future rollback/update work)

## Notes / Findings

- **API health dependency shape**: `test-genie` dependencies phase reads `vrooli scenario status --json` and fails if any `diagnostics.health_checks.api.dependencies.*.connected` is false/null. The Vrooli status collector derives this from the API `/health` response at `.dependencies.<name>.connected` (boolean). For scenarios without a real DB, return `dependencies.database.connected=true` to avoid `null → false` conversion.


## Monetization readiness assessment — 2026-09-07

Scenario-to-cloud has substantial deployment machinery, but secure and recoverable production readiness is not demonstrated. The biggest remaining work is authority, safe updates and recovery, cross-scenario integration, and credible acceptance evidence.

Assessment: 2026-09-07, local shared working tree. Source inspection, documentation comparison, requirements/registry inventory and read-only Test Genie history lookup. No scenario suite, live VPS operation, secret read, deployment or browser journey was executed. No formal W1–W3 gate was run: W0 is unverifiable because the named-goal query returned no matches. Source findings below are an inventory, not a certification. An empty history does not prove no historical tests ever ran elsewhere.

Open the interactive visual assessment (preserved HTML).

### Recommended ownership

Deployment Manager owns profiles, promotion and evidence review. Scenario-to-cloud owns desired VPS deployment state and release lifecycle. Bridge owns trusted node identity, reach and typed remote dispatch, composing existing byte transport. Onboarding owns machine configuration and required operator input. The control plane owns host checks, privileged repair and resource lifecycle. The credential authority owns secret storage and access. Test Genie/BAS own validation runs and journey evidence; autoheal owns ongoing supervision. Provider provisioning may be a cloud adapter; it is not a reason to add provider-specific account logic to Bridge.

### 01. Management authority — Launch blocker

**Observed:** API exposes deploy, terminal, secret mutations, process control and host repair. main.go does not configure Authentication; shared server authentication is conditional. The default bind is loopback and terminal has same-origin checks. Those reduce exposure but do not establish caller or deployment authorization.

**Recommendation:** Use shared operator identity and permission scopes on every sensitive route. Bind grants to target, deployment, environment and action; revoke them end to end. Keep customer traffic separate from fleet administration. Make terminal access an explicit, audited capability.

**Acceptance:** Given an anonymous, revoked, wrong-target or insufficient-scope caller, every protected HTTP and WebSocket operation is rejected before SSH or any side effect. Verify through local, proxy and Bridge entry points.

**Sources:** [scenarios/scenario-to-cloud/api/main.go](../../api/main.go), [packages/api-core/server/server.go](../../../../packages/api-core/server/server.go), [scenarios/scenario-to-cloud/api/handlers_terminal.go](../../api/handlers_terminal.go)

### 02. Update safety and data — Launch blocker

**Observed:** Setup uploads, cleans scenario directories, then extracts into the active workdir. Mutable directories are preserved using explicit paths and a name-based fallback. This is not atomic release activation or a verified backup. Rollback and backups are deferred by the PRD.

**Recommendation:** Stage immutable releases beside the running release; verify before activation; switch traffic only after readiness. Declare data ownership instead of relying on directory names. Coordinate migration, backup, rollback compatibility and retention. Support maintenance-window updates where zero downtime is not feasible.

**Acceptance:** Given failures during upload, extraction, setup, migration or health checks, the prior service remains usable or an explicit recovery action restores it within the agreed recovery time. Restore database and object data to a new host and prove application invariants.

**Sources:** [scenarios/scenario-to-cloud/api/vps/setup.go](../../api/vps/setup.go), [scenarios/scenario-to-cloud/PRD.md](../../PRD.md)

### 03. Durable deployment execution — Launch blocker

**Observed:** Run admission uses an atomic database status transition; progress is persisted and SSE reconnects are supported. Execution launches a goroutine with a background timeout. No restart reconciliation worker was found in the reviewed startup and orchestrator paths; completion-step helpers have no production callers in the scanned cloud API.

**Recommendation:** Reuse durable job infrastructure for admission, leases, step receipts, crash reconciliation and one blocking wait. Fence concurrent writes by target plus deployment. Distinguish disconnect, cancellation request, abort and unknown remote outcome.

**Acceptance:** Given API restart, SSH loss, target reboot or duplicate requests at each mutating step, the same operation is recoverable by ID, no stale worker can overwrite a successor, and no deployment remains falsely running or successful.

**Sources:** [scenarios/scenario-to-cloud/api/handlers_deployment.go](../../api/handlers_deployment.go), [scenarios/scenario-to-cloud/api/deployment/orchestrator.go](../../api/deployment/orchestrator.go), [scenarios/scenario-to-cloud/api/persistence/deployment.go](../../api/persistence/deployment.go)

### 04. Bridge and onboarding — Integration gap

**Observed:** The cloud manifest carries host, port, user, key path and workdir. The orchestrator uses SSHRunner/SCPRunner. Source search found no cloud API integration with vrooli-bridge or nodereach. Setup invokes the control-plane setup command; a direct onboarding integration was not found. Bridge itself has dispatch, machine and onboarding modules.

**Recommendation:** Keep cloud responsible for desired deployments and release lifecycle. Use Bridge for trusted target identity, reach and typed dispatch; use onboarding for machine configuration. Reuse existing artifact transport. A fresh VPS bootstrap must establish control-plane identity before normal dispatch can work.

**Acceptance:** Given an enrolled target, deploy using its node identity without re-entering keys. Prove first enrollment, revoked grants, offline target, protocol mismatch, interrupted transfer and a recoverable onboarding handoff. Any retained SSH path enforces the same policy.

**Sources:** [scenarios/scenario-to-cloud/api/domain/manifest.go](../../api/domain/manifest.go), [scenarios/scenario-to-cloud/api/deployment/orchestrator.go](../../api/deployment/orchestrator.go), [scenarios/vrooli-bridge/api/internal/onboarding/client.go](../../../vrooli-bridge/api/internal/onboarding/client.go)

### 05. Host repair ownership — Architecture gap

**Observed:** Cloud implements apt cleanup, journal vacuum, Docker prune, firewall changes and process termination through remote shell commands. This conflicts with the repository rule assigning host detection/remediation to internal/ control-plane owners. SSH defaults to root.

**Recommendation:** Move generic host checks and repair implementations to their control-plane owners. Cloud may schedule declared repairs and show results. Use least-privilege deployment identities with explicit elevation for the steps that need it, and scoped cleanup with previewable effects.

**Acceptance:** Given two unrelated workloads on one host, repair and cleanup affect only authorized state. Validate privilege denial, firewall lockout prevention, bounded deletion and control-plane package behavior.

**Sources:** [scenarios/scenario-to-cloud/api/vps/preflight/handlers.go](../../api/vps/preflight/handlers.go), [scenarios/scenario-to-cloud/api/sshidentity/resolve.go](../../api/sshidentity/resolve.go), [AGENTS.md](../../../../AGENTS.md)

### 06. Credential lifecycle — Partial implementation

**Observed:** Provisioning uses the target credential authority over SSH standard input, preserves existing generated values and avoids plaintext secrets.json. Management reads return metadata. CRUD exists, but secret updates plus a scenario restart do not by themselves prove provider-side rotation, dependent-resource compatibility or host-loss recovery.

**Recommendation:** Retain this authority seam. Preserve descriptor identity across dependency graphs; test normalized-name collisions and multiple deployments. Add versioned rotation, consumer acknowledgement, revocation, audit and encrypted recovery. A host-bound store needs a separate tested disaster-recovery path.

**Acceptance:** Given a credential update, all intended consumers use the new version and the previous version is revoked after a safe transition. Test locked stores, lost host, partial rotation, dependencies, database passwords and canary-value leakage across logs, argv, archives and evidence.

**Sources:** [scenarios/scenario-to-cloud/api/credentials/lifecycle.go](../../api/credentials/lifecycle.go), [scenarios/scenario-to-cloud/api/secrets/handlers_management.go](../../api/secrets/handlers_management.go), [scenarios/scenario-to-cloud/api/vps/deploy.go](../../api/vps/deploy.go)

### 07. Artifact identity and dependency closure — Partial implementation

**Observed:** Mini-bundle builder, dependency manifests and native Linux amd64/arm64 CLI upload exist. Manifest includes all packages and autoheal. The inspected upload/extract path does not verify signed release provenance before activation; the CLI is built separately from the current local tree.

**Recommendation:** Pin a coherent source revision, dependency graph, toolchain and native binaries in one release identity. Verify digest and trusted provenance on target. Declare supported OS/resource combinations and reject unsupported capabilities early; include tools, safeguards and licenses.

**Acceptance:** Given a tampered artifact, mismatched CLI, wrong architecture or changed dependency graph, activation fails before mutation. A clean host starts the complete declared closure with no undeclared local files or services.

**Sources:** [scenarios/scenario-to-cloud/api/bundle/builder.go](../../api/bundle/builder.go), [scenarios/scenario-to-cloud/api/vps/native_cli.go](../../api/vps/native_cli.go), [scenarios/scenario-to-cloud/api/vps/setup.go](../../api/vps/setup.go)

### 08. Deployment Manager health contract — Launch blocker

**Observed:** The reviewed consumer hardcodes landing-page-business-suite in the deployment-ID URL and marks every HTTP 200 healthy without consuming the health body. The cloud handler returns HTTP 200 with a computed health report, including unhealthy reports. This is a concrete false-green path in that adapter.

**Recommendation:** Resolve a typed deployment reference from profile, target and environment. Validate health status, freshness, release digest and report schema. Bind evidence and approval to that identity. Reject unknown or stale health; do not let HTTP transport success authorize a release.

**Acceptance:** Given HTTP 200 with unhealthy, malformed, stale or wrong-target content, promotion is denied. Given a valid deployment ID distinct from the scenario name, the consumer resolves and checks the intended target.

**Sources:** [scenarios/deployment-manager/api/deployments/cloud_client.go](../../../deployment-manager/api/deployments/cloud_client.go), [scenarios/scenario-to-cloud/api/handlers_health.go](../../api/handlers_health.go)

### 09. Edge and ongoing health — Partial implementation

**Observed:** DNS, SSH, disk, RAM, ports, TLS checks, Caddy configuration, HTTPS verification, live-state, logs, drift and process controls exist. Their existence is stronger than a stub; it does not prove continuous production reliability.

**Recommendation:** Define deployment health separately from host presence and application readiness. Integrate autoheal policies, external synthetic checks, certificate renewal alerts, capacity and backup monitoring. Keep database, metrics and management listeners private.

**Acceptance:** Given bad DNS, IPv6 mismatch, blocked ACME, expired certificate, reboot, disk exhaustion or lost dependency, health becomes actionable and alerts arrive. Validate renewal, recovery, public-port exposure and maintenance behavior on a real VPS.

**Sources:** [scenarios/scenario-to-cloud/api/vps/preflight/runner.go](../../api/vps/preflight/runner.go), [scenarios/scenario-to-cloud/api/vps/deploy.go](../../api/vps/deploy.go), [scenarios/scenario-to-cloud/api/handlers_health.go](../../api/handlers_health.go)

### 10. Executable acceptance and receipts — Evidence gap

**Observed:** Test Genie returned no recorded runs. All seven paths in the BAS registry are absent. VPS setup/deploy/TLS YAML files explicitly say manual/E2E placeholder. Requirements contain 10 in_progress, 3 complete and 4 planned entries; validation refs contain 15 failing, 6 implemented and 4 planned statuses. These are stored metadata, not fresh test results.

**Recommendation:** Replace placeholder acceptance with runnable owner-managed journeys and traceable assertions. Use QEMU for fresh-host system behavior and a disposable VPS for real DNS/TLS/network boundaries. Publish immutable receipts to deployment-manager.

**Acceptance:** Given a release candidate, every mandatory journey produces a current receipt tied to artifact, target, environment and asserted outcome. Missing files, stale evidence, skipped critical tests and unavailable targets cannot produce green.

**Sources:** [scenarios/scenario-to-cloud/bas/registry.json](../../bas/registry.json), [scenarios/scenario-to-cloud/test/playbooks/vps/deploy.yaml](../../test/playbooks/vps/deploy.yaml), [scenarios/scenario-to-cloud/requirements/05-deploy-start/module.json](../../requirements/05-deploy-start/module.json)

### 11. Operator experience and API parity — Unvalidated quality

**Observed:** The UI includes a deployment wizard, details, secrets, files, terminal, history, drift and VPS management. CLI groups cover lifecycle and disposable hosts. No browser walkthrough, accessibility audit or usability session was performed in this review; world-class UX is not established.

**Recommendation:** Use one plan → review → execute → inspect/recover flow across UI, CLI and API. Show target identity, change, expected interruption, credential references and recovery action. Provide resumable onboarding, meaningful progress, safe logs and exact next actions.

**Acceptance:** Given the same deployment intent through each surface, the plan and outcome agree. Prove keyboard and screen-reader operation, interrupted sessions, readable error recovery, long-running jobs and target disambiguation with BAS evidence and human usability review.

**Sources:** [scenarios/scenario-to-cloud/ui/src/components/deployments/DeploymentDetails.tsx](../../ui/src/components/deployments/DeploymentDetails.tsx), [scenarios/scenario-to-cloud/cli/app.go](../../cli/app.go)

### 12. Scope and ownership truth — Contract gap

**Observed:** No swarm-manager goal naming scenario-to-cloud was returned. The PRD excludes backups, puts rollback at P2 and describes file-based P0 state, while implementation uses Postgres. Existing problem and skill text also lag the implemented async, inspection and QEMU surfaces.

**Recommendation:** Reconcile the approved launch intent with operational targets and requirements before formal certification. Make safe updates, restore, access enforcement, secret lifecycle, Bridge integration and evidence required. Define the initial Linux support envelope rather than claiming any scenario on any VPS.

**Acceptance:** Given the accepted launch scope, each required behavior has a P0 target, an owner, executable assertions and a required receipt. Documentation and CLI help describe the actual supported behavior.

**Sources:** [scenarios/scenario-to-cloud/PRD.md](../../PRD.md), [scenarios/scenario-to-cloud/docs/internal/PROBLEMS.md](PROBLEMS.md), [scenarios/scenario-to-cloud/docs/local-qemu.md](../local-qemu.md)

### Completion sequence

1. **Reconcile the launch contract** — Cloud + Deployment Manager. Accept the ownership split, supported Linux distributions/architectures, workload isolation model, update downtime policy and recovery objectives. Promote recovery/security/evidence obligations to launch requirements. Exit: A bounded contract and requirement-to-evidence map; no invented percentage-complete score.

2. **Close the management trust boundary** — Shared identity + Bridge + control plane. Enforce caller and target authorization; remove the false-green health adapter; move generic host repair to internal/ owners. Design bootstrap and enrollment around existing onboarding. Exit: Negative authorization tests and typed target/health contract tests pass before exposing remote administration.

3. **Make deployment a recoverable operation** — Cloud + durable job owner. Introduce persistent execution ownership, fencing, immutable release staging, migration policy, data backup/restore and safe activation. Bind the native CLI to the same release. Exit: Crash injection and restore drills prove the previous service or a declared recovery path survives each failure point.

4. **Complete transport and credential integration** — Bridge + onboarding + credential authority. Connect enrolled-machine dispatch and artifact delivery; preserve one configuration surface; implement complete credential rotation and recovery. Exit: Direct and Bridge paths, when supported, satisfy one acceptance contract on clean and existing hosts.

5. **Prove the release on independent environments** — Test Genie + BAS + cloud. Run deterministic contracts, real-service integration, disposable QEMU, staging VPS, adversarial tests, load/soak and operator journeys. Use representative workload shapes rather than one named product. Exit: Current, immutable evidence identifies exact artifact, target, environment, assertions and limitations.

6. **Operate and promote with confidence** — Deployment Manager + cloud + autoheal. Bind approvals to release and target, rehearse publishing and recovery, add external alerting and rotation/backup schedules, and document incident handling. Exit: A canary is promoted only when required receipts pass; monitoring continues after release.

### Validation lanes

- **Contract and unit:** Manifest compatibility, permissions, path/shell injection boundaries, receipt identity, health-body parsing, credential naming and idempotency.

- **Service integration:** Real persistence, credential authority, dependency analyzer, Bridge grants and job lifecycle; no mock may substitute for the boundary being claimed.

- **Disposable QEMU:** Fresh install, actual systemd/privilege behavior, headless encrypted store, reboot, interrupted setup, snapshot/reset and host-bound recovery. QEMU lane exists; no successful lane run was verified here.

- **Staging VPS:** Real DNS, ACME staging issuance then bounded production issuance, TLS renewal, external network policy, enrolled Bridge reach, upgrade, rollback and new-host restore.

- **Adversarial and failure:** Anonymous/revoked/wrong-target calls, archive traversal, credential canaries, tampered releases, interrupted SSH, API crash, duplicate jobs, resource exhaustion and concurrent deployments.

- **Performance and UX:** Workload-shaped load, a proposed 24–72 hour staging soak sized to risk, alert delivery, UI/CLI/API parity, accessibility and human recovery drill. Agree latency, error-rate, recovery-time and data-loss budgets before running.

### Generality and launch decision

Use representative fixtures: stateless web, database plus migrations, object uploads, scenario-to-scenario dependencies, multiple resources, a headless/API-only service and a resource-heavy workload. Cover fresh and existing hosts, amd64 and arm64 where declared, two unrelated deployments, and an existing full Vrooli installation. Reject unsupported combinations explicitly. This proves reusable capability without tailoring to the first commercial scenario.

For the monetized backend boundary, deployment evidence must show environment separation, private credentials, correct callback/webhook routing and persistence through update/restore. Subscription correctness, Stripe account activation and desktop signing are separate owners and are not certified by this cloud review.

Require a current evidence bundle containing release digest/provenance, dependency closure, node identity and OS, configuration digest, secret version references without values, run IDs, assertions, logs/redacted recordings, recovery measurements and unresolved limitations. Deployment Manager must reject an artifact/target mismatch or any missing mandatory result. Repeat affected validation whenever relevant inputs change.

Define measurable service objectives, recovery-time and recovery-point objectives, and an accepted residual-risk record. No finite test suite can establish 100% absence of vulnerabilities. A launch decision should rest on demonstrated controls, independent security review, repeatable recovery and continuing monitoring. Delta uploads, Kubernetes, managed-service swaps and broad provider catalogs can follow after this foundation; safe full-bundle updates cannot.

### External design references

- [OWASP Authorization](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html): deny by default, least privilege and authorization on each request.
- [OWASP Secrets Management](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html): cover creation, distribution, rotation, revocation and recovery.
- [SLSA artifact verification](https://slsa.dev/spec/v1.2/verifying-artifacts): verify provenance against artifact identity and a trusted build policy.

These references inform recommendations; they are not evidence that the local implementation satisfies them.

## Preserved visual sources

Historical HTML and editable canvas sources live beneath the protected
control-plane runtime home (`~/.vrooli` for the invoking user).
These are dated evidence or design supplements; this relocation does not
change plan status or establish release readiness.

- `plan-artifacts/docs-html-progress-20260908/scenarios/scenario-to-cloud/docs/internal/cloud-readiness-2026-09-07.html`

The originating installation must retain these artifacts with its durable backups.
Exact originals and SHA-256 hashes are in `backups/docs-html-progress-20260908/manifest.json`.
