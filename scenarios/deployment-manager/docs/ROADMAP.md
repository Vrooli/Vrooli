# Deployment Manager Roadmap

> **Implementation status and planned work for the deployment system.**
>
> This document tracks what's complete, in progress, and planned. For known bugs and blockers, see [PROBLEMS.md](PROBLEMS.md).

## Current Milestone: End-to-End Bundled Desktop

The primary goal is to ship a complete bundled desktop app (UI + API + resources) with secrets, swaps, migrations, and runtime control surface validated.

---

## Status Overview

| Component | Status | Notes |
|-----------|--------|-------|
| **CLI** | Working | Full command set implemented |
| **API** | Working | Core routes functional |
| **UI** | In Progress | Basic dashboard; swap UI pending |
| **Thin Client Desktop** | Working | UI bundled; connects to Tier 1 |
| **Bundled Desktop** | Partial | Components exist; integration pending |
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
- [x] `bundle export` CLI command - Export production-ready manifest with checksum
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
- [x] Migration from `/docs/deployment/` to scenario-local docs

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

- [ ] deployment-manager → scenario-to-desktop handoff
  - **Current**: Manual bundle.json transfer required
  - **Needed**: Direct API call or shared filesystem handoff

---

## Not Started

### Automation

- [ ] Automated binary cross-compilation
  - **Current**: Manual `GOOS/GOARCH` builds per platform
  - **Needed**: scenario-to-desktop reads manifest and compiles from source

- [ ] Asset procurement automation
  - **Current**: Assets (Chromium, models, seeds) manually placed
  - **Needed**: Download/build/verify step before packaging

- [ ] Code signing integration
  - **Current**: Installers unsigned by default
  - **Needed**: Certificate configuration in generation flow

### Validation

- [ ] End-to-end bundled build validation
  - **Current**: No scenario has completed full pipeline
  - **Needed**: picker-wheel or similar as reference implementation
  - **Validation**: Install on clean machine, verify offline operation

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

1. **Pick an item** from "Not Started" or "In Progress"
2. **Check PROBLEMS.md** for related blockers
3. **Update this file** when starting work (move to In Progress)
4. **Update PROGRESS.md** with implementation notes
5. **Mark complete** when merged and validated

---

## Related Documentation

- [DEPLOYMENT-GUIDE.md](DEPLOYMENT-GUIDE.md) - User-facing deployment guide
- [PROBLEMS.md](PROBLEMS.md) - Known issues and blockers
- [PROGRESS.md](PROGRESS.md) - Implementation progress notes
- [SEAMS.md](SEAMS.md) - Integration points
- [Bundled Runtime Plan](/docs/plans/bundled-desktop-runtime-plan.md) - Technical architecture
