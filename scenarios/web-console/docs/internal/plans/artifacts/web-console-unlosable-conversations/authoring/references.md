[DOC: /home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/plans/artifacts/web-console-unlosable-conversations/README.md] — incident evidence, live integrity snapshot, identifiers, limitations, and ownership split
[DOC: /home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/plans/artifacts/web-console-unlosable-conversations/recovered-discussion-summary.md] — locally preserved summary of this discussion and its distinction from the separately resumed architecture chat
[CODE: /home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/plans/artifacts/web-console-unlosable-conversations/integrity-audit.sql] — read-only reproducible integrity queries
[DOC: /home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/plans/persistent-session-recovery-hardening-plan.md] — earlier recovery plan and lessons; superseded where this plan is more complete
[DOC: /home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/INVARIANTS.md] — current invariants to repair and extend
[DOC: /home/matthalloran8/Vrooli/scenarios/web-console/docs/internal/TEMPORAL-FLOWS.md] — current temporal flows to raise to Level 4
[DOC: /home/matthalloran8/Vrooli/scenarios/web-console/docs/guides/SESSION_RECOVERY.md] — operator recovery behavior
[DOC: /home/matthalloran8/Vrooli/scenarios/web-console/docs/guides/CONVERSATION_TRACKING.md] — tracking/search behavior
[DOC: /home/matthalloran8/Vrooli/scenarios/web-console/docs/reference/data-model.md] — current persistence model
[DOC: /home/matthalloran8/Vrooli/docs/TESTING.md] — mandatory server-owned test and single-wait protocol
[REQ: scenarios/web-console/requirements/03-durable-session-continuity] — current continuity requirements, including the in-progress archive requirement with failing validation
[REQ: scenarios/web-console/PRD.md#OT-P0-003] — Durable Session Continuity outcome
[REQ: scenarios/web-console/PRD.md#OT-P1-004] — Operational Observability outcome
[CODE: scenarios/web-console/api/conversation_repository.go:352] — archive search inner join that hides conversation records without a session row
[CODE: scenarios/web-console/api/process_manager.go] — process supervision currently entangled with lifecycle policy
[CODE: scenarios/web-console/api/session_store.go] — current session mutation surface and deletion seams
[CODE: scenarios/web-console/api] — archive, WebSocket, expiry, cleanup, repository, checkpoint, and workspace lifecycle callers requiring complete inventory
[CODE: scenarios/web-console/cli] — CLI parity and automation surface
[CODE: scenarios/web-console/ui] — archive/search/recovery user experience
[CODE: packages/proto/schemas/web-console/v1] — canonical Web Console transport contracts
[DOC: plan:75004329-144f-4937-baa0-3bfa3a756c99] — active cross-run discovery dependency and authority boundary
[DOC: scenario-qa:knw-1788545141995388770] — archive/recovery defect report
[DOC: vrooli-memory:eef203d5-7500-4504-aec8-2c6e52149fd8] — investigation record
[DOC: /home/matthalloran8/.vrooli/state/vrooli/web-console/sessions/codex/a7e71c3c-e422-4c89-916a-03f92906fb89/sessions/2026/09/03/rollout-2026-09-03T23-17-07-01a06a6b-88da-7422-b391-bb59c5f5e5e0.jsonl] — operator-only incident evidence; sensitive, immutable, never commit or paste into logs
