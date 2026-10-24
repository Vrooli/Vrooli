# Advisory maturity acceptance dossier

This is a living evidence ledger. A row is not earned merely because its
source exists; the stated oracle must be run against the current tree.

| Deliverable | Current standing | Evidence / limitation |
|---|---|---|
| DEL-GCT-01 safety admission | focused gate green; comprehensive admission degraded | `SAFETY-ADMISSION-REPORT.md`, `TestSafetyAdmissionGate`; delegated Agent Manager bootstrap remains separate |
| DEL-GCT-02 authority boundary | focused green; remaining non-Git writer inventory explicit | `AUTHORITY-MATRIX.md`, policy/mutation tests, direct Git/SSH/registry/precommit-writer principal tests, tracked-binary and file-content writer guards, automated forged-header refusal; typed configuration writers remain |
| DEL-GCT-03 domain foundations | implemented contract/docs; full architecture receipt pending | `DOMAIN-MAP.md`, bounded read/write seams |
| DEL-GCT-04 subjects/evidence | focused green | `internal/advisory`, subject/evidence tests and draft endpoint |
| DEL-GCT-05 durable jobs | focused green | `internal/reviewjobs`, restart/idempotency/exact-ID tests |
| DEL-GCT-06 trailers | focused green | `internal/trailers` grammar/resolver tests |
| DEL-GCT-07 provenance | producer consumer/search source contract partial | `internal/provenance`, Workspace Sandbox consumer, uncertainty-preserving model |
| DEL-GCT-08 shared provenance search | registered experimental source | Search Hub provider `git-control-tower.git-provenance`; live scoped query; owner-backed positive corpus still pending |
| DEL-GCT-09 advisory primitives | focused/live contract green; corpus gate added | draft endpoint, typed program, five mode fixtures, fail-closed evidence/coverage validation, and `internal/advisory/fixtures/grounding-corpus.json` coverage test |
| DEL-GCT-10 constrained review | review job and profile hardening partial; planted-defect corpus gate added | exact receipt and safety tests; review validation now requires visible omissions and safe revision-bound findings; independent precision/recall execution receipt remains pending |
| DEL-GCT-11 governed programs | declared and executable focused fixtures | `change-evidence` `prog_4c4ed9e5-8c1d-40d6-a79f-5dd3952b94fb`, `advisory` `prog_7aabe61f-8bf0-48cf-a884-ecdc2e509b13`, `setpoint-read` `prog_8ec3a147-edc9-4456-a105-a6a5a22f8d1b`; selector-free validation pending |
| DEL-GCT-12 skills/learning | feature files registered and read successfully; full skill validation pending | eight GCT skills are present in Prompt Manager, owner reads succeeded, and service manifest links the three programs; divergence and maturity validation remain |
| DEL-GCT-13 improvement | cold-start/unavailable contract | setpoint-read preserves missing observations; comparable measured repair pending |
| DEL-GCT-14 collaboration | host-neutral contract implemented | collaboration model/tests, live capability-status endpoint, and host contract; live provider adapters excluded by scope |
| DEL-GCT-15 mentions | normalized durable dedupe and reply-outbox contract green | mention model/store/handler; fresh and legacy SQLite schemas cover idempotent reply claims; Switchboard ingress activation remains external |
| DEL-GCT-16 workspace | subject bar and draft composer implemented | `DESIGN.md`, subject bar/draft composer/capability-card UI tests and type-check |
| DEL-GCT-17 PR/release UX | local draft and capability surfaces implemented; connected workspace pending | draft kinds, local non-publishing composer, capability-specific unavailable states; connected host UX not live |
| DEL-GCT-18 human controls | pure merge/undo preconditions focused green | controls package and human-only contract; no automated actuation |
| DEL-GCT-19 operational hardening | documentation/compatibility ledger added; isolation repair in progress | `ADVISORY-OPERATIONS.md`, `RETIRED-COMPATIBILITY-LEDGER.md`; routed DB/file-root seam is now wired, but a fresh receipt is pending |
| DEL-GCT-20 final proof | not earned | API/CLI/UI focused suites pass; comprehensive run `20260906-182152-dfca5497` terminated failed with routed-isolation and unrelated inherited health/fixture findings; repairs are being validated before a replacement run |

The dossier intentionally preserves unearned and externally owned work. It is
not a certification claim.

## Verified operator authorization evidence

The verified-operator authorization plan adds an explicit acceptance packet:

| Area | Evidence |
|---|---|
| Authenticator contract | `AUTHENTICATOR-INTEGRATION.md`; policygate RS256/JWKS, claim, failure, and revocation tests |
| Mutation ownership | `AUTHORITY-MATRIX.md`; domain writer guards and Connect interceptor tests |
| Durable approval | `intents_test.go`; atomic single-consume, exact binding, expiry/replay, restart persistence, hashed-at-rest ID |
| Adversarial refusal | `ADVERSARIAL-AUTHORIZATION-MATRIX.md`; forged headers, invalid credentials, agent intent, stale subject, and no-writer cases |
| Operator parity | UI exact preview/confirmation dialog and CLI preview/`confirm` or `--yes` flow; both submit the same intent-bound commit contract |
| Greenfield transport decision | typed proto/Connect is the supported UI, CLI, and inter-scenario surface; independent REST writers are migration residue | 38 REST route registrations remain in the API (27 GET and 11 non-GET) and 31 direct UI fetch callsites remain across non-typed domains; the CLI has no remaining repo-domain REST callsites. Repository registry, status, diff, groups, sync, history, approved changes, provenance, provenance search, advisory drafts, review-run admission, auditor scan/status/rules/violations/fix methods, file tree, directory, related-file, content-search, credentials, remote URL, SSH keys, grouping rules, gitignore health/remediation, tracked-binary health/remediation, file save/delete, discard, ignore, push, pull, upstream, precommit, staging, commit, branch, worktree, baseline, evidence, and human-control paths are typed. The mention ingress is an explicit `webhook_receiver` exception; remaining UI REST domains need their own proto contracts before route removal. |

Live evidence status (2026-09-06): the supported lifecycle is healthy after the
review-job store initialization deadlock was fixed. `vrooli scenario status
git-control-tower --json` reports API `18710` and UI `21400` running and healthy;
`GET /health` reports readiness and a connected database. A live typed
`HumanControlService/GetAuthorityStatus` call without credentials returns the
unauthenticated advisory state. A typed `RepoService/CreateCommit` call without
credentials returns HTTP 401 before the writer. A typed worktree mutation is
refused by policy, and legacy REST push/file-write calls return HTTP 401;
the same REST push with `X-Vrooli-Caller: human` and
`X-Vrooli-Authorized: true` also returns HTTP 401. The former REST status,
diff, groups, and sync-status routes return 404 while their typed Connect
procedures succeed. These are automated/live refusal and read receipts, not
human actuation.

The latest typed auditor-fix slice also returns `400` for an empty
`AuditorService/PreviewFix` request, `401` for unauthenticated
`AuditorService/ApplyFix`, and `404` for the retired `/api/v1/repo/rules-fix`
writer. The scoped Test Genie `programs` phase passed; its `unit` phase remains
limited by existing unit-health contract debt and a shared `api-core/eventbus`
compile mismatch filed to scenario-qa as `knw-1788742994833506131`.

The typed auditor-read slice also returns live `200` responses for
`StartCheck`, `ListRules`, and `ListViolations`, a typed error for an unknown
job, and `404` for the retired auditor REST read routes. These GCT methods still
proxy the separately owned `scenario-auditor` HTTP contract until that scenario
publishes its own proto/Connect surface.

The current live refusal receipt also records `200` for
`HumanControlService/GetAuthorityStatus` without credentials with the
sign-in advisory, `200` for `PrepareMutation` with the typed operation name,
`401` for `RepoService/CreateCommit` without an intent, and `401` for a forged
auditor apply request. These probes reached the running API at port `18710`;
no repository writer was invoked.

The packet intentionally has no human commit. Automated tests and refusal
receipts do not claim that a human clicked Confirm or created a commit; that
evidence must be captured separately through the operator runbook with explicit
operator authorization and a confirmed target repository.

This packet distinguishes implementation evidence from live human actuation.
Automated tests do not claim that a human clicked Confirm or created a commit;
that evidence must be captured separately through the operator runbook.
