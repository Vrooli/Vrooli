# Progress — Workflow Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-02 | Codex | done | Completed Phase 10 rollout hardening slice: workflow-health now self-registers `workflow.flow`, `workflow.test`, and `workflow.fragment` providers with Search Hub; Search Hub federation was verified for flow/test/fragment queries; workflow-health provider-contract, workflow phase, API/CLI tests, and scenario test passed. Also hardened the delegated Test Genie workflow smoke case with a registered selector and executable self-dashboard assertion. |
| 2026-07-02 | Codex | done | Migrated Test Genie to the canonical delegated `workflow` phase backed by workflow-health, kept `playbooks`/`e2e` as deprecated aliases, added workflow finding source mapping, documented the workflow phase, and replaced the stale self BAS case with an executable observer overview smoke workflow. |
| 2026-07-02 | Codex | done | Implemented the first workflow-health operator UI slice: replaced template navigation with Overview, Inventory, Search, Runs, Findings, Fixes, and Settings routes; added workflow-specific selectors, route tests, localized copy, accessible landmark labels, dense operational view models, and full UI validation. |
| 2026-07-02 | Codex | done | Implemented workflow search/action discovery slice: `workflow.flow`, `workflow.test`, and `workflow.fragment` search leaves, deterministic intent ranking, mutating guardrails, `WorkflowSearchService`, CLI `workflows search`, regenerated proto/endpoint metadata, and docs updates. |
| 2026-07-02 | Codex | done | Exposed workflow-health through shared `ScenarioValidationService`: validation handler, fix preview/apply RPCs, native catalog/execution detail, endpoint metadata, CLI `validate scenario`/`fix preview`/`fix apply`, and provider response tests. |
| 2026-07-02 | Codex | done | Implemented the first safe execution and artifact ingestion slice: fakeable BAS client seam, static-only/dry-run paths, observer execution, mutating fail-closed preflight, timeline/latest artifact writer, and focused execution tests. |
| 2026-07-02 | Codex | done | Implemented the static workflow validation, maturity, and deterministic fix slice: `.vrooli/maturity.json`, `api/internal/validation`, rule/spec drift tests, maturity assessment construction, registry rebuild, metadata stubs, execution-mode normalization, and reset normalization. |
| 2026-07-02 | Codex | done | Implemented the initial workflow asset catalog model and scanner for BAS cases, flows, actions, seeds, registry-only entries, requirement links, selector/route refs, safety profiles, and subflow/fixture dependency edges. |
| 2026-07-02 | Codex | done | Generated the workflow-health react-vite scaffold with the `vrooli-default` design kit, replaced starter PRD/requirements with workflow-health contract targets, and documented the Phase 1 ownership model. |
| 2026-07-02 | Codex | partial | Generator post-hook reached proto regeneration but module validation failed until `api/go.mod` gained a local `github.com/vrooli/cli-core` replace; rerun focused validation after contract edits. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map


## 2026-10-12 — Finite-delivery Planner selected-operation repair

- Changed: completed the dedicated `StartSelectedValidationRun` handler, service
  mount, CLI `validate start-selected`, endpoint generation and public selection
  readback using the existing durable ledger and execution owner. Ordinary start
  rejects explicit selection; normalized case identity distinguishes default from
  selected replay, and the database unique key arbitrates concurrent handlers.
  Conflicting workflow-path bindings refuse before execution; context/target
  bindings cannot widen saved case paths. Restart retains explicit recovery
  failure. Wait subscribes before reading to avoid missing a terminal notification.
  Durable start is classified as a write in the source CLI manifest. The shared
  CLI transport owns invocation-time API selection; no local workaround remains.
- Ownership: the six pre-existing dirty execution/repository files exactly retain
  the hashes in the approved handoff's migration annex. Current native inventory
  identified no Planner/Workflow Health writer. Their historical contributor and
  return remain unknown; this integrates reviewed current interfaces under the
  owner's direct implementation authorization, without claiming authorship,
  replacing their bytes, clearing historical denials or recreating F8 attribution.
- Test simplification: replaced the inherited mutable BAS fixture's incidental
  nine-node assertion with actual string/bool/integer subflow-argument preservation;
  retained the separate actual BAS catalog decoder check. Replaced `runtime.Caller`
  as a config locator because `-trimpath` rewrites source filenames. These two
  pre-existing tests failed in broad API execution before this repair and now pass.
- Verified: Proto's `make verify-committed-gen`; scoped handler/execution/repository
  `go test -race -v` and `go vet`; actual selected-service socket and module-mount
  tests; canonical replay, catalog drift, distinct-handler concurrency, invalid
  selections, terminal/reopened and recovery-failed readback; selected CLI's old-peer
  `Unimplemented` refusal with zero ordinary-start requests; governed `make endpoints`.
  The first Test Genie run `20261012-002826-0697c915` failed unit execution on the
  two inherited tests while contracts/quality/api passed; after their repair,
  `20261012-003310-37877503` passed all four phases. A later static-path refactor
  briefly failed the mixed-selection regression and was corrected before the final
  focused race pass; those failed observations remain evidence.
- Remaining/unverified: independent acceptance, final source-stable broader return,
  installed/serving API adoption, current isolated Planner runtime and original
  D6 task continuation. The original cases and numeric SQL/file primary-write
  counters were not executed by this component repair. F6/A2 acceptance, F8's
  unresolved historical attribution, manual-review disposition and whole E1
  remain with their original owner. No team wake, replacement, grant change or
  service restart was performed.


Qualification checkpoint: `20261012-003518-0d76b71f` returned
`provider_unavailable`: unit-health, cli-health and api-health ownership waits
hit context deadlines; quality passed. This is unavailable final-source evidence,
not a code regression. The final scoped race check includes
`TestSelectedExecutionOwnsRealHTTPLeaseHeartbeatAndZeroPrimaryWrites`: production
RoutingIsolation/devrouting over a real HTTP socket, real SQLite and RoutedRoots,
request test-mode routing, an observed provider heartbeat, lease cleanup,
zero SQL/file primary-write counters and an independent empty primary readback.
BAS browser transport is substituted, so this qualifies the integration boundary
and does not qualify original Planner D6 or serving adoption.

Supported isolated runtime actions were attempted and refused by the lifecycle
owner: Workflow Health's variant needs a rebuild but shares served build outputs
with live Workflow Health; Planner's variant likewise shares outputs with
`personal-planner@shadow`. The earlier dynamic-target variant-dependency request
was refused because Planner is not a declared Workflow Health dependency. The
native owner requires a coordinated original-instance rebuild/restart first;
root's deploy report additionally names unrelated dirty shared AUTH/identity
inputs without current adoption clearance. No manifest dependency was invented,
no force/restart bypass was used, and no new instance was started. The normal
Planner start workflow also rebuilt its stale Authenticator dependency under the
existing `rebuild_only` freshness policy; it did not restart that consumer.
Serving adoption is blocked on that concrete current-owner disposition. Original
D6 continuation and its three real synthetic assertions remain separate.

Final source qualification: `20261012-005432-61899fdc` passed `unit`,
`contracts`, `quality` and `api` after the shared provider owners released their
earlier runs. The single native waiter returned terminal `passed` with
`success: true`; actual result and findings are retained in
`/tmp/planner-final-stable-suite.json` and
`/tmp/planner-final-stable-findings.json`. Warnings remain: 28 unit, 9 contract,
2 quality (plus one informational) and 7 API findings. They are maturity
findings, not failing phase execution. The native waiter's optional agent park
was refused because the attached run had no wakeable session; it continued
inline and returned the terminal result. Generated-artifact freshness used the
governed Proto owner command and selected artifact `08f3c95129af...`.
All six inherited input hashes still match their intake identities. Independent
root source review found and resolved the overly broad desktop-binding refusal;
complete target/context-only binding now preserves the selected case paths and
`CasesProvided`. The real HTTP/SQLite/leased-root integration additionally
checks the primary observation file is absent independently of write counters.
This completes the component's focused and prescribed broader source checks;
serving adoption and original D6 remain under the recorded owner holds above.
