# Deployment Manager Roadmap

> **Implementation status and planned work for the deployment system.**
>
> This document tracks what's complete, in progress, and planned. For known bugs and blockers, see [internal/PROBLEMS.md](internal/PROBLEMS.md).

## Current Milestone: End-to-End Bundled Desktop

The primary goal is to ship a complete bundled desktop app (UI + API + resources) with secrets, swaps, migrations, and runtime control surface validated.

---

## Status Overview

| Component | Status | Notes |
|-----------|--------|-------|
| **CLI** | ✅ Working | Full command set including `deploy-desktop` |
| **API** | ✅ Working | Core routes functional |
| **UI** | In Progress | Basic dashboard; swap UI pending |
| **Thin Client Desktop** | ✅ Working | UI bundled; connects to Tier 1 |
| **Bundled Desktop** | ✅ Implemented | Pipeline is available; native release evidence remains target- and environment-gated |
| **Mobile (Tier 3)** | Not Started | Documentation placeholder |
| **SaaS (Tier 4)** | Not Started | Documentation placeholder |
| **Enterprise (Tier 5)** | Vision | Future hardware appliance |

---

## Completed

### CLI & Core API

- [x] CLI framework with global `--json` and `--format` flags
- [x] `status` command - API health check
- [x] `analyze` command - Dependency DAG via scenario-dependency-analyzer
- [x] `fitness` command - Tier fitness scoring with blockers/warnings
- [x] Profile CRUD (`create`, `show`, `update`, `delete`, `list`)
- [x] Profile versioning with history and rollback
- [x] Profile export/import (JSON format)
- [x] Swap commands (`list`, `analyze`, `cascade`, `apply`)
- [x] Secrets commands (`identify`, `template`, `validate`)
- [x] `validate` command - Pre-deployment checks
- [x] `logs` command - Telemetry viewing with filters
- [x] `packagers` command - List available packagers
- [x] `package` command - Invoke packagers
- [x] `configure` command - API base and token configuration

### Bundle System

- [x] Bundle manifest schema v0.1
- [x] Manifest validation (`POST /api/v1/bundles/validate`)
- [x] Manifest assembly (`POST /api/v1/bundles/assemble`)
- [x] Manifest export with checksum (`POST /api/v1/bundles/export`)
- [x] Secrets merging into manifests (`POST /api/v1/bundles/merge-secrets`)
- [x] Example manifests (desktop-happy.json, desktop-playwright.json)
- [x] `bundle assemble` CLI command - Assemble manifest from scenario
- [x] `bundle export` CLI command - Export release-candidate manifest with checksum
- [x] `bundle validate` CLI command - Validate manifest against schema

### Runtime Supervisor (scenario-to-desktop)

- [x] Core supervisor with service lifecycle management
- [x] Dynamic port allocation
- [x] Health and readiness monitoring
- [x] Secret injection (env and file targets)
- [x] Migration tracking and idempotent execution
- [x] Control API (`/healthz`, `/readyz`, `/ports`, `/logs/tail`, `/shutdown`)
- [x] Telemetry recording (JSONL format)
- [x] GPU detection (optional/required handling)
- [x] Asset verification with checksums
- [x] CLI shim scaffolding (`runtimectl`)

### Electron Integration

- [x] Template with bundled/external-server/cloud-api modes
- [x] Auto-updater integration (electron-updater)
- [x] Three update channels (dev, beta, stable)
- [x] GitHub Releases and self-hosted providers
- [x] Installer formats: MSI (Windows), PKG (macOS), AppImage + DEB (Linux)
- [x] Deployment telemetry collection

### Documentation

- [x] Hub-and-spokes documentation structure
- [x] CLI command reference (6 docs)
- [x] API reference (7 docs)
- [x] Tier reference docs (5 tiers)
- [x] Workflow guides (desktop, mobile, saas, troubleshooting)
- [x] Technical guides (8 topics)
- [x] Example case studies and manifests
- [x] Hub-and-spokes ownership between the project Deployment Hub and scenario-local docs

---

## In Progress

### deployment-manager UI

- [ ] Swap toggle UI
  - **Current**: Swap suggestions generated; UI doesn't allow interactive toggle
  - **Needed**: Add swap selection with fitness recalculation

- [ ] Bundle export workflow UI
  - **Current**: API functional; no UI workflow
  - **Needed**: Wizard to configure profile → generate manifest → download

### Integration

- [x] deployment-manager → scenario-to-desktop handoff
  - **Completed**: The `deploy-desktop` workflow invokes the scenario-to-desktop pipeline.
  - **Contract**: deployment-manager owns target planning and admission; scenario-to-desktop owns generation, runtime, packaging, and native evidence.

---

## Open Work

The items below include both follow-up work and capabilities that remain
environment-gated. A checked item means the implementation exists; it does not
mean every platform or release profile has been approved.

### Automation

- [x] Automated binary cross-compilation
  - **Completed**: scenario-to-desktop reads manifest `build` config and compiles automatically
  - **Supported**: Go, Rust, npm, and custom build types
  - **Reference**: [Build and packaging](../../scenario-to-desktop/docs/guides/build-and-packaging.md)

- [ ] Asset procurement automation
  - **Current**: Assets (Chromium, models, seeds) manually placed
  - **Needed**: Download/build/verify step before packaging

- [ ] Code signing integration
  - **Current**: Release trust and OS signing are separate gates; local development artifacts are not promotable by default.
  - **Needed**: Complete per-platform signing and notarization automation.

### Validation

- [x] End-to-end bundled build validation
  - **Completed**: `hello-desktop` scenario validates full pipeline
  - **Location**: `scenarios/hello-desktop/`
  - **Purpose**: Zero-dependency scenario for pipeline validation
  - **Status**: ✅ Linux native journey is validated; Windows and macOS claims remain separately evidence-gated.
  - **Tutorial**: See [Hello Desktop Walkthrough](tutorials/hello-desktop-walkthrough.md)

- [ ] Clean machine installation test
  - **Current**: Not automated
  - **Needed**: Install on clean VM, verify offline operation

### Tier 3 (Mobile)

- [ ] scenario-to-ios packager
- [ ] scenario-to-android packager
- [ ] Mobile-specific swap suggestions
- [ ] App store deployment guides

### Tier 4 (SaaS/Cloud)

- [ ] scenario-to-cloud packager
- [ ] DigitalOcean deployment automation
- [ ] AWS deployment automation
- [ ] Terraform/Pulumi templates
- [ ] Container image generation

### Tier 5 (Enterprise)

- [ ] Hardware appliance specification
- [ ] Air-gapped deployment support
- [ ] Enterprise licensing integration
- [ ] Compliance documentation

---

## Future Considerations

### Performance & Scale

- [ ] Differential updates (currently full reinstall per version)
- [ ] Delta manifests for faster updates
- [ ] Parallel binary compilation
- [ ] Build caching

### Developer Experience

- [ ] `vrooli deploy` CLI shortcut
- [ ] VS Code extension integration
- [ ] GitHub Actions for automated releases
- [ ] One-click deploy from web UI

### Platform Edge Cases

- [ ] Windows long path handling
- [ ] macOS Gatekeeper/notarization automation
- [ ] Linux AppImage permissions
- [ ] ARM64 Linux support

---

## How to Contribute

1. **Pick an item** from "Open Work" or "In Progress"
2. **Check internal/PROBLEMS.md** for related blockers
3. **Update this file** when starting work (move to In Progress)
4. **Update PROGRESS.md** with implementation notes
5. **Mark complete** when merged and validated

---

## Related Documentation

- [DEPLOYMENT-GUIDE.md](DEPLOYMENT-GUIDE.md) - User-facing deployment guide
- [internal/PROBLEMS.md](internal/PROBLEMS.md) - Known issues and blockers
- [PROGRESS.md](PROGRESS.md) - Implementation progress notes
- [Integration seams](internal/SEAMS.md) - Integration points
- [Scenario-to-desktop architecture](../../scenario-to-desktop/docs/concepts/ARCHITECTURE.md)
- [Desktop evidence and tier contract](../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md)
