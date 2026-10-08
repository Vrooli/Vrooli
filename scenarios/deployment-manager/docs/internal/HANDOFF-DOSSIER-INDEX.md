# Deployment Manager handoff dossier index

This index is the durable retrieval map for the current execution. `Local
evidence` means the repository and server-owned checks are complete for that
surface. `External proof open` means the implementation is prepared but the
plan still requires a real device, hosted authority, signing authority,
independent reviewer, or measured operator drill.

| Deliverable | Current proof and retrieval path | Standing |
| --- | --- | --- |
| DEL01 launch contract and frozen inventory | `~/.vrooli/plans/deployment-manager-trustworthy-commercial-releases-exact.md`; `scenarios/deployment-manager/docs/internal/QUALIFICATION-GAP-REGISTER.md` | External proof open for target approval |
| DEL02 safety defects and fixes | `scenarios/deployment-manager/api`; `scenarios/landing-page-business-suite/api` | Local evidence |
| DEL03 canonical contracts and identities | `packages/proto/schemas/deployment-manager/v1/releases`; `scenarios/deployment-manager/api/releases/identity_storage.go` | Local evidence |
| DEL04 least-privilege boundaries | `scenarios/deployment-manager/api/internal/authz/mutation.go`; `scenarios/deployment-manager/api/internal/transport`; `scenarios/deployment-manager/api/server/server.go`; service capability profile in `scenarios/deployment-manager/.vrooli/service.json` | Local human/write/destructive policy evidence; external Bridge target and reauthorization matrix open |
| DEL05 durable execution and reconciliation | `scenarios/deployment-manager/api/releases/repo.go`; `scenarios/deployment-manager/api/releases/handlers.go` | Local evidence; external owner receipts open |
| DEL06 immutable signed candidates | `scenarios/deployment-manager/api/releases/identity.go`; `internal/releaseauthority` | Native signing proof open |
| DEL07 typed retrievable evidence | `scenarios/deployment-manager/api/releases/types.go`; `scenarios/deployment-manager/api/releases/identity_storage.go`; `scenarios/deployment-manager/api/internal/releases/schema.sql`; `scenarios/deployment-manager/api/internal/readiness/repository.go`; `scenarios/deployment-manager/api/internal/readiness/schema.sql` | Local receipt durability and replacement-history evidence; final retention and cross-store review open |
| DEL08 automatic readiness observations | `scenarios/deployment-manager/api/readiness`; `scenarios/deployment-manager/api/releases/handlers.go`; `scenarios/deployment-manager/api/server/server.go`; `scenarios/scenario-to-desktop/api/evidence/producer.go`; `packages/proto/schemas/deployment-manager/v1/releases/contracts.proto` | Desktop owner delivery, cloud observability owner query, and candidate operational-ownership declaration are locally wired; remaining producer invocation, compatibility comparison, and external qualification remain open |
| DEL09 exact approval decision | `scenarios/deployment-manager/api/releases/identity_storage.go`; `scenarios/deployment-manager/api/releases/identity_handler_test.go` | Local evidence |
| DEL10 ramp and operation conformance | `scenarios/deployment-manager/api/deployments`; `scenarios/scenario-to-cloud/api`; `scenarios/scenario-to-desktop/api` | Hosted/native proof open |
| DEL11 exact desktop publication | `scenarios/deployment-manager/api/deployments/orchestrator_release.go`; `scenarios/scenario-to-desktop/api/pipeline/stage_deploy.go`; `scenarios/scenario-to-desktop/api/deploy/lpbs_client.go` | Typed release identity now reaches the atomic promotion request; hosted and native proof open |
| DEL12 immutable hosted artifacts and feed revisions | `scenarios/landing-page-business-suite/api/internal/delivery/channel_revision.go`; `scenarios/landing-page-business-suite/api/channel_revision_test.go`; `scenarios/scenario-to-desktop/api/deploy/lpbs_client.go` | LPBS validates artifact app/release columns and identity metadata; S2D requires the echoed predecessor, complete artifact set, and release identity before accepting the channel receipt; designated hosted destination proof open |
| DEL13 trust lifecycle and native signing | `internal/releaseauthority`; `scenarios/deployment-manager/docs/DEPLOYMENT-GUIDE.md` | Real rotation and native acceptance open |
| DEL14 updater receipts and interruption | `scenarios/scenario-to-desktop/templates/vanilla`; `scenarios/scenario-to-desktop/api/evidence/producer.go`; `scenarios/deployment-manager/api/releases/identity_storage.go` | Local receipt ledger and authenticated typed owner route verified; live authenticated delivery and installed-client proof remain open |
| DEL15 shared runtime and commercial continuity | `scenarios/deployment-manager/docs/internal/QUALIFICATION-GAP-REGISTER.md` | Deployed fixture proof open |
| DEL16 reusable qualification corpus | `scenarios/deployment-manager`; `scenarios/test-genie` | Native cells open |
| DEL17 operator workspace | `scenarios/deployment-manager/ui`; `scenarios/deployment-manager/api/releases/handlers.go` | Full effectful journey review open |
| DEL18 CLI/API/program parity | `scenarios/deployment-manager/cli/releases`; `scenarios/deployment-manager/cli/deployments`; `scenarios/deployment-manager/cli/manifest.json`; `scenarios/deployment-manager/.vrooli/endpoints.json`; `packages/proto/gen/go/deployment-manager/v1/releases`; `packages/proto/gen/go/deployment-manager/v1/deployments` | Local evidence |
| DEL19 rollout, recovery, alerts, and restore | `scenarios/deployment-manager/docs/DEPLOYMENT-GUIDE.md`; snapshot `70d525a55d517bc07642eeeea8b519ed`; restore `eac55d00-9860-439e-bb63-d265ed74f893` | External reconciliation and repair drills open |
| DEL20 clean final architecture | `scenarios/deployment-manager/docs/internal/ARCHITECTURE-MIGRATION-LEDGER.md` | Migration review remains open |
| DEL21 automated qualification | Full deployment-manager Go suite; final Test Genie run `20260909-015652-b85d3a4e`, phases `structure,unit,api`; full owner Go suites | Local evidence; unit maturity warning remains recorded |
| DEL22 hosted and physical-device evidence | `scenarios/deployment-manager/docs/internal/QUALIFICATION-GAP-REGISTER.md` | External proof open |
| DEL23 independent certification and handoff | This index, `scenarios/deployment-manager/docs/internal/QUALIFICATION-GAP-REGISTER.md`, and `scenarios/deployment-manager/docs/internal/PROBLEMS.md` | Cannot close while mandatory external cells remain open |

The canonical reviewer read path is `GET /api/v1/releases/{release_id}/dossier`,
the typed `ReleasesService.Dossier` RPC, or
`deployment-manager releases dossier <release-id> --format json`. Every
missing identity, receipt, or external qualification cell must remain visible
in the dossier or this index until its attributable receipt is retained.
