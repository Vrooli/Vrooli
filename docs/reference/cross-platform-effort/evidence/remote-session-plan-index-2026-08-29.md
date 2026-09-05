# Remote-session plan evidence index — 2026-08-29

This index ties the plan’s current claims to repository evidence. Historical
documents remain linked for traceability and are explicitly labeled as such.

| Claim | Evidence |
| --- | --- |
| The transport uses an existing grantable effect | `packages/api-core/scopecatalog/catalog.go`; Bridge session-scope tests; Web Console remote-session focused tests |
| Shared discovery, admission, relay, and reconnect are owned by one client | `packages/nodeclient/client.go` and `client_test.go`; `packages/api-core/discovery/transport.go` is removed |
| Pair declarations use the same status model and require repository evidence for supported cells | `internal/deployability/resolver.go`, `manifestvalidation.go`, `manifestvalidation_test.go`, and infrastructure-manager manifest projection |
| Current service manifests declare pair portability | `scenarios/web-console/.vrooli/service.json`; `scenarios/vrooli-bridge/.vrooli/service.json` |
| Linux console to Darwin node is live-supported | [`remote-session-plan-live-attempt-2026-08-29.md`](remote-session-plan-live-attempt-2026-08-29.md); live Web Console probe proves Darwin transcript, resize, and `stty` behavior |
| SSH is not advertised without a constructed credentialed backend | `packages/session-core/session.go`, `packages/session-core/README.md`, and `scenarios/web-console/api/remote_targets.go` |
| Documentation was reconciled | Superseded banners and historical labels in the cross-platform HTML audits; current seam notes in `scenarios/vrooli-bridge/docs/internal/SEAMS.md` and `PROBLEMS.md` |

## Targeted validation

The execution recorded passing focused tests for deployability pair validation,
runtime service-manifest validation, session-core, discovery/targetmodel,
scope grammar, scenario-dependency platform verdicts, and Web Console remote
target/session mapping. Broad repository suites were intentionally not used as
the completion gate because the synchronized baseline contains unrelated known
failures and an environmental dependency gap; those findings are recorded in
the plan execution log.

## Definition-of-Done audit

The external agent is now refreshed and the live remote-session path is
available. The repository-wide conformance command completed, but its global
result is nonzero because the synchronized baseline contains unrelated
findings. The component-scoped result for this plan is clean.

| Items | State | Evidence or follow-up |
| --- | --- | --- |
| 1–2, 4–5, 10–12, 14–15 | Met | Repository searches; scope-vocabulary, targetmodel, deployability, picker, and stacking-scale tests |
| 6–9 | Met | `scenarios/web-console/api/handlers/sessions/adapter_error_paths_test.go`; live missing-grant probe recorded in [`remote-session-plan-live-attempt-2026-08-29.md`](remote-session-plan-live-attempt-2026-08-29.md) |
| 13 | Met | `docs/reference/cross-platform-effort/evidence/machines-surface-2026-08-27/04b-scope-audit.png`; `04-choose-permissions.png`; capture-flow JSON |
| 16–18 | Met | `scenarios/web-console/ui/src/__tests__/error-banner.test.ts`; `workspace.test.tsx`; `remote-session-plan-live-attempt-2026-08-29.md` |
| 19–21 | Met | `scenarios/web-console/api/session_launch_exec_test.go`; `terminal-launcher.test.tsx`; live Darwin session transcript and provenance tests |
| 22 | Met | `scenarios/web-console/api/terminal_ws_test.go` (`TestHandleTerminalWS_ReconnectToSameSession`, five-second detach); terminal transport reconnect tests |
| 23–25, 28–32 | Met | Web Console configuration tests/docs; service schema and pair validation tests; session authorization/origin tests; repository searches |
| 3 | Met by live no-edit remote-session proof | `TestLiveRemoteSessionThroughWebConsole` passed against minimouse Darwin amd64 |
| 26 | Met: Linux console → macOS node is `supported` | Web Console and Bridge manifests plus live evidence record |
| 27 | Met by implementation and live node evidence | Registry re-read stores separate `machine_arch=amd64` and `binary_arch=amd64` fields |
| 33 | Component-scoped result met; repository-wide command has 86 unrelated findings and 60 warnings | Full command completed; zero findings for the four plan-owned components; declarations-only check returned no findings |

The live external evidence gap for the Darwin pair and the missing-grant error
boundary is closed. A global zero-finding result is not attainable from the
current synchronized baseline without repairing unrelated repository defects;
this record preserves that distinction instead of attributing those defects to
the remote-session change.
