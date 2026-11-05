# Known gaps and implementation frontier

## What belongs here

Unmet product obligations, bounded integration gaps and observed scaffold/provider failures with evidence. A specified future capability is not a shipped feature.

## What does NOT belong here

A duplicate work queue, guessed live outages, fabricated certifications or local copies of other owners' state. Use canonical plans/QA for work and preserve references here.

## Entry template

Record concern, owner, desired behavior, observed evidence, impact, repair route and completion proof. Keep unknowns explicit.

## Entries

| Concern | Owner / observed state | Required next proof |
|---|---|---|
| Product implementation | E1 project workspace and E2 photographic saved-shot journey implemented; physical shot integration active | Complete physical workflow, broad library, creation, accounts/desktop and final release evidence |
| Professional 3D craft | Fixed photographic phone/laptop exports and native2560px phone hero approved;48-frame phone rotation verified | Physical saved-shot integration, operator motion review and laptop scene/articulation |
| Laptop articulation | Exact face-preserving partition overview passed13views; hinge/clearance unqualified | Supported base/lid transforms, contact/closed envelope and clearance across the claimed motion scope |
| Blender GPU path | Managed5.2.2LTS CPU/CUDA/OptiX probes and representative OptiX phone scene passed | Integrated profile/tool staging, long-clip performance and packaged-runtime proof |
| Broad library | Device Studio: source leads and dimensions; only core source assets inspected | Acquired/authored reviewed assets and recipes per group |
| Generic media publication | Asset Studio: resolvable byte/model/movie/provenance gaps | Owner adapter tests; pending release remains honest |
| Provider metadata/cost | Image Tools/Gateway: controls/model identity/unknown-cost gaps | End-to-end receipt and bounded operation qualification |
| Desktop credit execution | LPBS/account/Gateway: scoped lease not connected to paid consumer-token route | Selected-account trusted spend adapter, revocation/switch/cancel tests |
| Browser sign-in | Web Console/LPBS source callback mismatch; QA knw-1792451955790814674 | Real owner callback contract regression; no live outage asserted |
| Usage replay | LPBS exact-source SQL A,B,A overcount; QA knw-1792451879369663074 | Owner regression/repair; no live-wallet loss asserted |
| Shared account UI | RCL AccountSettingsSurface placeholder | Draft/qualify reusable surface from working components |
| Worker isolation | Resource limits are not a sandbox | Filesystem/network/process boundary and malicious import probes |
| Desktop native qualification | S2D/runtime/platform owners | Current-host bundle/tool/offline proof required for this effort; Windows/macOS native proof deferred by operator, architecture still required |
| Template experience pin | Inherited Notes spec pointed at unavailable 1.0.0 contract; scaffold corrected to actual 1.0.3 export, QA knw-1792459024617298162 | Template owner fixes source so future generation agrees with resolved package |
| Template reference docs | Inherited health/auth/evidence claims contradicted source; scaffold docs corrected, QA knw-1792459031717825830 | Owner repairs template documentation |
| CLI primitive evidence | Generated static artifact is absent; no L4 certification claimed | Owner-generated evidence and CLI Health verification during implementation |
| Template deep status | React-vite quarantined; agent observed VCS-stamping failure in temp workspace | Owner deep validation; this scaffold setup/start/structure passed |

Research details and source evidence are indexed in [research](../research/README.md). The foundation does not repair unrelated owners or imply credentials/signing access exists.

## Architecture Drift

The approved Light-table workspace and first project/shot domains are implemented, and the Notes example was removed through its owner. Library/creation/jobs/settings surfaces and complete experience-route qualification remain release work. Reconcile each orientation/experience gate through its owning check; a prior scaffold claim is not the current product state.

## Cross-references

[PROGRESS.md](PROGRESS.md), [TESTING.md](TESTING.md), [INTEGRATIONS.md](../concepts/INTEGRATIONS.md), [DEPLOYMENT.md](../operations/DEPLOYMENT.md).


### 2026-10-20: existing cross-domain shot foreign key

Before adding the reviewed asset library, `storage-manager validate scenario
device-studio` passed with one advisory: `api/internal/shots/schema.sql:3` has a
hard foreign key to projects. The new library will use soft cross-domain IDs.
Reconcile the existing shot/project lifecycle coupling during the release storage
review; do not silently edit the live table definition without the existing-data
conversion procedure. Receipt: `goal/evidence/E3/pre-library-storage.json`.
